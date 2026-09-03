// worktree.go declares the injectable seam every external process this harness runs goes through,
// and the environment layer built on top of it: resolving the quarry repository root, the target
// (loomyard) repository, and the pinned worktree root, and creating, restoring and inspecting a
// task's own detached worktree. Nothing in this file talks to a process directly; every external
// command flows through the Runner interface, which is what makes the offline test layer -- a
// recording fake runner in place of git, go build and claude -- possible.
//
// The single-run advisory lock also lives here rather than in a file of its own, because it is
// created and removed at the worktree-root path ResolveWorktreeRoot already resolves.

package ladder

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// Cmd describes one external process invocation: the working directory, the program name, its
// arguments, environment entries to merge over the inherited environment, and the streams to wire.
// Every field is read by a Runner implementation; nothing outside this package constructs a Cmd and
// runs it directly.
type Cmd struct {
	// Dir is the process's working directory. An empty Dir inherits the caller's own.
	Dir string
	// Name is the program to run, e.g. "git", "go" or the configured claude binary.
	Name string
	// Args is the argument vector, not including Name.
	Args []string
	// Env is a set of environment entries merged over the inherited environment, entry by entry --
	// never a full replacement. Both the server build's cgo variable and a test's fake-binary
	// configuration flow through this field rather than mutating the harness's own process
	// environment.
	Env map[string]string
	// Stdin, when non-nil, is wired to the process's standard input.
	Stdin io.Reader
	// Stdout, when non-nil, is wired to the process's standard output.
	Stdout io.Writer
	// Stderr, when non-nil, is wired to the process's standard error. It is load-bearing for the
	// offline test layer: the fake claude binary the tests drive reports its own flag-assertion
	// failures on this stream.
	Stderr io.Writer
}

// Runner runs one external process described by c and reports its outcome. Every external process
// this harness starts -- claude, git, go build -- goes through a Runner rather than exec.Command at
// the call site, so a test can substitute a recording fake for the whole external world.
type Runner interface {
	Run(ctx context.Context, c Cmd) error
}

// ExecRunner is the production Runner, backed by os/exec.
type ExecRunner struct{}

// Run implements Runner by starting c.Name with c.Args under os/exec, merging c.Env over the
// inherited environment and wiring c.Stdin, c.Stdout and c.Stderr when set.
func (ExecRunner) Run(ctx context.Context, c Cmd) error {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr

	if len(c.Env) > 0 {
		env := os.Environ()
		for k, v := range c.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s %s: %w", c.Name, strings.Join(c.Args, " "), err)
	}
	return nil
}

// ResolveQuarryRepoRoot walks upward from start to the enclosing git repository's top level and
// returns it. It is the single producer of the quarry repository root path; every other resolver in
// this file, and the run subcommand, take it as an input rather than re-deriving it.
func ResolveQuarryRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve quarry repo root: %w", err)
	}
	for {
		if info, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil && info != nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("resolve quarry repo root: no enclosing git repository found above %s", start)
		}
		dir = parent
	}
}

// ResolveLoomyardRepo resolves the target repository's path. It reads the process environment
// variable LADDER_LOOMYARD_REPO first and, when that is unset or empty, parses simple KEY=VALUE
// lines out of .scratch/ladder.env directly beneath quarryRepoRoot -- not the harness's own
// .scratch/ladder/ subdirectory -- ignoring blank lines and lines beginning with '#'. Nothing else
// may read this variable: the documented entry point is a bare `go run`, which cannot source a
// shell-wrapper file, so this function is the environment layer's only reader of it.
func ResolveLoomyardRepo(quarryRepoRoot string) (string, error) {
	const envVar = "LADDER_LOOMYARD_REPO"
	if v := os.Getenv(envVar); v != "" {
		return v, nil
	}

	envFile := filepath.Join(quarryRepoRoot, ".scratch", "ladder.env")
	f, err := os.Open(envFile)
	if err != nil {
		return "", fmt.Errorf("resolve loomyard repo: neither %s nor %s is set", envVar, envFile)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		if strings.TrimSpace(key) == envVar {
			return strings.TrimSpace(value), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("resolve loomyard repo: read %s: %w", envFile, err)
	}

	return "", fmt.Errorf("resolve loomyard repo: neither %s nor a matching line in %s is set", envVar, envFile)
}

// quarryPathToken is the case-insensitive substring ResolveWorktreeRoot refuses in a resolved
// worktree root path. It is a deliberately inline, lowercase strings.Contains check rather than a
// call through match.go's shared matcher: that matcher's two classes govern content the harness
// searches, and this is a filesystem path the harness is about to use, tested case-insensitively
// where the matcher's composed-string form is case-sensitive. See match.go's header comment for the
// same boundary from the other side.
const quarryPathToken = "quarry"

// ResolveWorktreeRoot resolves the base directory task worktrees are created under: the environment
// variable LADDER_WORKTREE_ROOT when set, else XDG_CACHE_HOME joined with "ladder-eval", else the
// user's home directory joined with ".cache" and "ladder-eval". Nothing is ever placed under a
// system temporary directory. It then asserts two invariants on the resolved absolute path, naming
// LADDER_WORKTREE_ROOT as the override in either error: the path must not be quarryRepoRoot and
// must not be under it, and the path must not contain the case-insensitive substring "quarry". Both
// assertions exist because the cell's own working directory reaches the transcript and gate 2's
// checks (b) and (c) scan it -- a worktree under the repository, or one merely named after it, would
// fail every control rep by construction.
func ResolveWorktreeRoot(quarryRepoRoot string) (string, error) {
	base, err := resolveWorktreeRootBase()
	if err != nil {
		return "", fmt.Errorf("resolve worktree root: %w", err)
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve worktree root: %w", err)
	}
	absRepo, err := filepath.Abs(quarryRepoRoot)
	if err != nil {
		return "", fmt.Errorf("resolve worktree root: %w", err)
	}

	if absBase == absRepo || strings.HasPrefix(absBase, absRepo+string(filepath.Separator)) {
		return "", fmt.Errorf(
			"resolve worktree root: %s resolves to %s, which is the quarry repository root or under it -- set LADDER_WORKTREE_ROOT to a directory outside the repository",
			absBase, absRepo,
		)
	}
	if strings.Contains(strings.ToLower(absBase), quarryPathToken) {
		return "", fmt.Errorf(
			"resolve worktree root: %s contains the substring %q -- set LADDER_WORKTREE_ROOT to a path that does not",
			absBase, quarryPathToken,
		)
	}

	return absBase, nil
}

// resolveWorktreeRootBase resolves the unvalidated worktree-root base directory, before
// ResolveWorktreeRoot asserts its two invariants: LADDER_WORKTREE_ROOT when set, else
// XDG_CACHE_HOME joined with "ladder-eval", else the user's home directory joined with ".cache" and
// "ladder-eval".
func resolveWorktreeRootBase() (string, error) {
	if v := os.Getenv("LADDER_WORKTREE_ROOT"); v != "" {
		return v, nil
	}
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "ladder-eval"), nil
	}
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if u.HomeDir == "" {
		return "", fmt.Errorf("resolve home directory: current user has no home directory")
	}
	return filepath.Join(u.HomeDir, ".cache", "ladder-eval"), nil
}

// TaskWorktreePath returns the pinned task worktree path for taskID under worktreeRoot:
// <worktree-root>/worktrees/<task-id>.
func TaskWorktreePath(worktreeRoot, taskID string) string {
	return filepath.Join(worktreeRoot, "worktrees", taskID)
}

// PrepareWorktree creates a detached worktree of repo at pinnedSHA under dest, through r, when dest
// does not yet exist. When dest already exists, PrepareWorktree leaves it in place -- a resumed run
// reuses the worktree a prior invocation already created.
func PrepareWorktree(ctx context.Context, r Runner, repo, taskID, pinnedSHA, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("prepare worktree %s: %w", dest, err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("prepare worktree %s: %w", dest, err)
	}

	if err := r.Run(ctx, Cmd{
		Dir:  repo,
		Name: "git",
		Args: []string{"worktree", "add", "--detach", dest, pinnedSHA},
	}); err != nil {
		return fmt.Errorf("prepare worktree %s for task %s at %s: %w", dest, taskID, pinnedSHA, err)
	}
	return nil
}

// RestoreWorktree returns the worktree at dest to its pinned state through r: it discards any
// tracked-file changes and removes any untracked or ignored file.
func RestoreWorktree(ctx context.Context, r Runner, dest string) error {
	if err := r.Run(ctx, Cmd{
		Dir:  dest,
		Name: "git",
		Args: []string{"reset", "--hard", "HEAD"},
	}); err != nil {
		return fmt.Errorf("restore worktree %s: %w", dest, err)
	}
	if err := r.Run(ctx, Cmd{
		Dir:  dest,
		Name: "git",
		Args: []string{"clean", "-fdx"},
	}); err != nil {
		return fmt.Errorf("restore worktree %s: %w", dest, err)
	}
	return nil
}

// WorktreeStatus returns the porcelain status output of the worktree at dest, through r -- the
// input CheckWorktreeDirtied turns into the always-non-fatal worktree_dirtied observation.
func WorktreeStatus(ctx context.Context, r Runner, dest string) (string, error) {
	var out bytes.Buffer
	if err := r.Run(ctx, Cmd{
		Dir:    dest,
		Name:   "git",
		Args:   []string{"status", "--porcelain"},
		Stdout: &out,
	}); err != nil {
		return "", fmt.Errorf("worktree status %s: %w", dest, err)
	}
	return out.String(), nil
}

// lockFileName is the advisory lock file's name, created directly under the resolved worktree
// root -- one level above the worktrees directory, never inside a task worktree, where it would
// trip the dirtied observation every rep and be removed by the pinned restore.
const lockFileName = ".ladder.lock"

// AcquireRunLock creates the advisory single-run lock file directly under worktreeRoot, recording
// the current process id and resultsRoot in it, and returns a release function that removes it.
// The file is opened with the create-and-exclusive flags, so a second holder fails rather than
// truncating the first holder's record. When the lock file already exists, AcquireRunLock returns
// an error reading the existing holder's pid and results root back out of it, falling back to a
// plain message when the file is unreadable or malformed. A stale lock is cleared by the operator;
// AcquireRunLock never reaps one automatically and never tests process liveness -- guessing whether
// a pid belongs to a live run across a reboot is how a lock becomes a no-op.
func AcquireRunLock(worktreeRoot, resultsRoot string) (release func() error, err error) {
	path := filepath.Join(worktreeRoot, lockFileName)

	if err := os.MkdirAll(worktreeRoot, 0o755); err != nil {
		return nil, fmt.Errorf("acquire run lock: %w", err)
	}

	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if openErr != nil {
		if !os.IsExist(openErr) {
			return nil, fmt.Errorf("acquire run lock %s: %w", path, openErr)
		}
		return nil, fmt.Errorf("acquire run lock: %s", describeExistingLock(path))
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "pid=%d\nresults=%s\n", os.Getpid(), resultsRoot); err != nil {
		return nil, fmt.Errorf("acquire run lock %s: %w", path, err)
	}

	return func() error {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("release run lock %s: %w", path, err)
		}
		return nil
	}, nil
}

// describeExistingLock reads the pid and results root back out of the lock file at path and formats
// the exact holder message AcquireRunLock returns when the lock is already held. When path cannot be
// read or parsed, it falls back to a plain message naming only the path.
func describeExistingLock(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("another ladder run holds %s", path)
	}

	var pid string
	var results string
	for _, line := range strings.Split(string(data), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "pid":
			pid = strings.TrimSpace(value)
		case "results":
			results = strings.TrimSpace(value)
		}
	}
	if pid == "" || results == "" {
		return fmt.Sprintf("another ladder run holds %s", path)
	}
	if _, err := strconv.Atoi(pid); err != nil {
		return fmt.Sprintf("another ladder run holds %s", path)
	}
	return fmt.Sprintf("another ladder run holds %s (pid %s, results %s)", path, pid, results)
}
