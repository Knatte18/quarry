// worktree.go ports the git seam and the two worktree-filesystem gates from scripts/gates.py and
// scripts/run_ladder.py: the single git(1) runner every git call in the suite goes through, the
// ambient-context presence gate, the worktree-dirtied observation, and the pinned-worktree lifecycle --
// building, restoring, removing, and idempotently ensuring the two disposable task worktrees the warm
// cells run against.

package ladder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HarnessError is raised when the harness cannot proceed safely: a stale worktree at a declared path, a
// worktree adopted at the wrong pin, a failed server build, a malformed or timed-out MCP call, a warm-up
// that left no daemon.json behind, a preflight probe whose denial did not block, a truncated run, or an
// exhausted attempt cap -- mirroring the Python port's own HarnessError.
type HarnessError struct {
	// Message describes the specific condition the harness refused to proceed past.
	Message string
}

// Error implements the error interface.
func (e *HarnessError) Error() string {
	return e.Message
}

// GitRunner is the seam every function in this file's pinned-worktree lifecycle calls git through,
// mirroring the Python port's own injected `git=run_git` default parameter -- passing a fake here is
// what lets a test assert the exact argument vectors without a real git call.
type GitRunner func(args ...string) (string, error)

// NeutraliseWorktree deletes CLAUDE.md, CONSTRAINTS.md, and .claude/ from the disposable worktree at
// path. This is a mutation of the disposable checkout only; the live source checkout is never touched.
func NeutraliseWorktree(path string) error {
	for _, name := range []string{"CLAUDE.md", "CONSTRAINTS.md"} {
		target := filepath.Join(path, name)
		if _, err := os.Stat(target); err == nil {
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("ladder: neutralise worktree: remove %s: %w", target, err)
			}
		}
	}
	claudeDir := filepath.Join(path, ".claude")
	if _, err := os.Stat(claudeDir); err == nil {
		if err := os.RemoveAll(claudeDir); err != nil {
			return fmt.Errorf("ladder: neutralise worktree: remove %s: %w", claudeDir, err)
		}
	}
	return nil
}

// BuildWorktree builds one disposable task worktree at path, pinned to sha, off sourceRepo: `git -C
// <sourceRepo> worktree add <path> <sha>`, then NeutraliseWorktree, then an assertion that
// GateWorktreeNeutralised passes.
//
// Returns a *HarnessError when a directory already exists at path, so a stale worktree is never silently
// reused. EnsureTaskWorktrees is the idempotent caller; nothing else calls this directly.
func BuildWorktree(sourceRepo, path, sha string, git GitRunner) error {
	if _, err := os.Stat(path); err == nil {
		return &HarnessError{Message: fmt.Sprintf("build_worktree: a directory already exists at %s -- refusing to reuse a stale worktree", path)}
	}

	if _, err := git("-C", sourceRepo, "worktree", "add", path, sha); err != nil {
		return fmt.Errorf("ladder: build worktree: %w", err)
	}
	if err := NeutraliseWorktree(path); err != nil {
		return err
	}

	if findings := GateWorktreeNeutralised(path); len(findings) != 0 {
		messages := make([]string, len(findings))
		for i, f := range findings {
			messages[i] = f.Message
		}
		return &HarnessError{Message: fmt.Sprintf("build_worktree: %s failed gate_worktree_neutralised: %v", path, messages)}
	}
	return nil
}

// RestoreWorktree restores a task worktree to its pinned commit after a run: `git -C <path> reset
// --hard` followed by `git -C <path> clean -fdx`, then NeutraliseWorktree again, since `clean -fdx`
// restores the ambient-context files the neutralisation removed.
//
// RestoreWorktree performs all three steps as one unconditional unit, run after every attempt regardless
// of that attempt's outcome: an attempt that edited files must never survive into the retry's own
// dirtiness observation, so nothing here is skippable based on what the preceding attempt did.
func RestoreWorktree(path string, git GitRunner) error {
	if _, err := git("-C", path, "reset", "--hard"); err != nil {
		return fmt.Errorf("ladder: restore worktree: reset --hard: %w", err)
	}
	if _, err := git("-C", path, "clean", "-fdx"); err != nil {
		return fmt.Errorf("ladder: restore worktree: clean -fdx: %w", err)
	}
	return NeutraliseWorktree(path)
}

// RemoveWorktree removes a disposable task worktree: `git -C <sourceRepo> worktree remove --force
// <path>`.
func RemoveWorktree(sourceRepo, path string, git GitRunner) error {
	if _, err := git("-C", sourceRepo, "worktree", "remove", "--force", path); err != nil {
		return fmt.Errorf("ladder: remove worktree: %w", err)
	}
	return nil
}

// EnsureTaskWorktrees returns a mapping from task key to worktree path, idempotently, because the
// harness is re-invoked to resume and this runs on every invocation.
//
// For each task: when no directory exists at the declared path, BuildWorktree it. When one does exist,
// read `git -C <path> rev-parse HEAD` -- if it equals the task's declared pin, adopt the existing
// worktree by calling RestoreWorktree on it and continue; if it does not, return a *HarnessError naming
// both SHAs, since a worktree at the wrong pin would silently benchmark a different codebase.
func EnsureTaskWorktrees(l *Ladder, git GitRunner) (map[string]string, error) {
	worktrees := make(map[string]string, len(l.Tasks))
	for taskKey, task := range l.Tasks {
		path := task.Worktree
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := BuildWorktree(l.SourceRepo, path, task.PinnedSHA, git); err != nil {
				return nil, err
			}
		} else {
			output, err := git("-C", path, "rev-parse", "HEAD")
			if err != nil {
				return nil, fmt.Errorf("ladder: ensure task worktrees: rev-parse HEAD: %w", err)
			}
			head := strings.TrimSpace(output)
			if head != task.PinnedSHA {
				return nil, &HarnessError{Message: fmt.Sprintf("ensure_task_worktrees: %s is at %q, expected the declared pin %q", path, head, task.PinnedSHA)}
			}
			if err := RestoreWorktree(path, git); err != nil {
				return nil, err
			}
		}
		worktrees[taskKey] = path
	}
	return worktrees, nil
}

// RunGit is the single seam every git call in this package goes through: runs `git <args>` and returns
// its combined stdout+stderr on success, or a *RunGitError capturing that same combined output on a
// non-zero exit -- mirroring the Python port's run_git(args), whose subprocess.run(..., check=True)
// raises with the child's captured output attached. Combined output is captured (rather than stdout
// alone) so a caller inspecting an error's message sees whatever git itself printed to explain the
// failure, which git conventionally writes to stderr.
func RunGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &RunGitError{Args: args, Output: string(output), Err: err}
	}
	return string(output), nil
}

// RunGitError is returned by RunGit when `git <args>` exits non-zero. Its Error() names the failing
// argument vector and carries the child's combined stdout+stderr, mirroring what a caller of the Python
// port's run_git would see in the raised CalledProcessError's own output attribute.
type RunGitError struct {
	// Args is the argument vector RunGit invoked git with, not including "git" itself.
	Args []string
	// Output is the child process's combined stdout and stderr.
	Output string
	// Err is the underlying *exec.ExitError (or a process-start error) RunGit received from the
	// standard library.
	Err error
}

// Error implements the error interface.
func (e *RunGitError) Error() string {
	return fmt.Sprintf("ladder: git %s: %v: %s", strings.Join(e.Args, " "), e.Err, e.Output)
}

// Unwrap exposes the underlying process error so errors.Is/As can match against it.
func (e *RunGitError) Unwrap() error {
	return e.Err
}

// worktreeNeutralisedEntries is the fixed set of ambient-context paths GateWorktreeNeutralised checks
// for, in the order the Python port checked them.
var worktreeNeutralisedEntries = []string{"CLAUDE.md", "CONSTRAINTS.md", ".claude"}

// GateWorktreeNeutralised is fatal, once per offending entry, when CLAUDE.md, CONSTRAINTS.md, or
// .claude/ exists in the task worktree.
func GateWorktreeNeutralised(worktree string) []GateFinding {
	var findings []GateFinding
	for _, name := range worktreeNeutralisedEntries {
		if _, err := os.Stat(filepath.Join(worktree, name)); err == nil {
			findings = append(findings, GateFinding{
				Gate:    "worktree_neutralised",
				Fatal:   true,
				Message: fmt.Sprintf("%s is present in the task worktree", name),
			})
		}
	}
	return findings
}

// ObserveWorktreeDirtied returns a non-fatal finding carrying "true" or "false" from `git -C <worktree>
// status --porcelain` being non-empty. Recorded, never gated.
//
// This must run before the worktree is restored: the restore's `git reset --hard` followed by `git clean
// -fdx` is precisely what erases the evidence this observation reads, so an observation taken after
// restore_worktree would report false for every run regardless of what actually happened during it.
func ObserveWorktreeDirtied(worktree string) GateFinding {
	output, err := RunGit("-C", worktree, "status", "--porcelain")
	if err != nil {
		panic(err)
	}
	dirtied := strings.TrimSpace(output) != ""
	return GateFinding{
		Gate:    "worktree_dirtied",
		Fatal:   false,
		Message: fmt.Sprintf("worktree dirtied: %t", dirtied),
	}
}
