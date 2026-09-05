// gitsrc.go declares Repo, the opened-repository handle, and every read-only git operation this
// package exposes. Every operation goes through runGit, the one helper that invokes git against an
// explicit repository root, so the root-passing rule and the failure-wrapping rule each have one
// implementation.

package gitsrc

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo is an opened repository handle whose root has already been proved, by Open, to be the
// repository's own top level. It holds nothing but that root: no cache, no parsed diff, no open
// process — every operation below runs a fresh git invocation.
type Repo struct {
	root string
}

// Open verifies that root names an existing git repository whose top level is root itself, and
// returns a Repo rooted there. It asks git for the top level of root first of all, before any other
// operation.
//
// A root that is not inside a repository at all returns ErrNotARepository. A root inside a
// repository whose top level is elsewhere returns a *RootNotTopLevelError carrying both paths.
//
// The requirement that root be the repository top level exists because git emits paths relative to
// the top level while a caller consumes them as root-relative; without it every path in the answer
// would be silently wrong and a pathspec would select the wrong subtree.
func Open(root string) (*Repo, error) {
	out, err := runGit(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNotARepository, root)
	}
	topLevel := strings.TrimSpace(string(out))

	// Both sides go through symlink evaluation and then cleaning before comparing: the
	// repository-root resolver this project already uses (repopath.ResolveRoot) performs a join
	// and a clean and never resolves symlinks, while git prints the physical path, so a raw
	// string comparison would reject a perfectly valid repository reached through a symlink. A
	// temporary directory is exactly such a path on many platforms (macOS's /tmp is a symlink to
	// /private/tmp, for one), so without this the check would reject this project's own test
	// fixtures, which are built under t.TempDir().
	evalRoot, err := evalAndClean(root)
	if err != nil {
		return nil, fmt.Errorf("gitsrc: open %q: %w", root, err)
	}
	evalTopLevel, err := evalAndClean(topLevel)
	if err != nil {
		return nil, fmt.Errorf("gitsrc: open %q: %w", root, err)
	}
	if evalRoot != evalTopLevel {
		return nil, &RootNotTopLevelError{Root: root, TopLevel: topLevel}
	}
	return &Repo{root: root}, nil
}

// evalAndClean resolves every symlink in p and cleans the result, so two spellings of the same
// physical path compare equal regardless of which one traversed a symlink.
func evalAndClean(p string) (string, error) {
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

// VerifyRevision confirms that rev resolves to a commit in the repository r is rooted at,
// returning *UnknownRevisionError carrying rev exactly as given when it does not. Callers run it
// for every supplied revision before any diff, so an unresolvable revision is reported as such
// rather than surfacing later as a failed diff.
func (r *Repo) VerifyRevision(rev string) error {
	if _, err := runGit(r.root, "rev-parse", "--verify", "--quiet", rev+"^{commit}"); err != nil {
		return &UnknownRevisionError{Rev: rev}
	}
	return nil
}

// Change is one changed path between two revisions, or between a revision and the working tree.
// Status is the raw single-letter git status code -- "A", "M", "D", "U" and so on. gitsrc maps
// nothing from it: the mapping from letter to disposition belongs to the layer that builds
// entries, since this package returns paths, bytes and errors only.
type Change struct {
	// Path is repository-relative, exactly as git reports it.
	Path string
	// Status is git's own single-letter status code, unmapped.
	Status string
}

// ChangedPaths returns the paths that differ between before and after, restricted to pathspec
// (pass "." for the whole repository), together with each path's raw git status letter.
//
// after == "" means the working tree, selecting the one-revision form of the diff rather than the
// two-revision form. Callers invoke this form only when the after side is the working tree: a card
// that creates a file and has not yet staged it is the normal state at the moment the primary
// consumer asks the question, and without this form that file's symbols would be silently absent
// with the files echo unable to record the omission.
//
// Both forms pass "--no-renames", disabling git's own rename and copy detection. This is a
// correctness requirement rather than an optimisation: git's detection is a similarity threshold,
// and letting it run would mean the caller's answer silently inherited the heuristic the whole
// two-tier design exists to replace. With it, a rename arrives as a delete plus an add and is
// classified by the table comparison, which is the only classifier this query is allowed to have.
//
// ChangedPaths uses git's null-delimited output form and splits on the null byte, as every
// path-emitting operation in this package does. Without it git applies its path-quoting
// configuration and quotes any non-ASCII or control-character path while delimiting with newlines,
// so such a path would be read at the wrong location -- silently, since a mangled path simply fails
// to open -- and a newline inside a filename would split one path into two.
func (r *Repo) ChangedPaths(before, after, pathspec string) ([]Change, error) {
	args := []string{"diff", "--no-renames", "--name-status", "-z"}
	if after == "" {
		args = append(args, before)
	} else {
		args = append(args, before, after)
	}
	args = append(args, "--", pathspec)

	out, err := runGit(r.root, args...)
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out)
}

// parseNameStatus parses the null-delimited output of "git diff --name-status -z": alternating
// status and path fields.
func parseNameStatus(out []byte) ([]Change, error) {
	fields := splitNullDelimited(out)
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("git diff --name-status: odd number of null-delimited fields")
	}
	changes := make([]Change, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		changes = append(changes, Change{Status: fields[i], Path: fields[i+1]})
	}
	return changes, nil
}

// splitNullDelimited splits git's "-z" output on the null byte, dropping the trailing empty field
// its final delimiter leaves behind. An empty input yields a nil slice rather than a slice holding
// one empty string.
func splitNullDelimited(out []byte) []string {
	s := strings.TrimSuffix(string(out), "\x00")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\x00")
}

// UntrackedPaths returns the paths under pathspec that are present on disk but not tracked by git.
// It passes git's standard-exclusion flag ("--exclude-standard"), so an untracked file matched by a
// gitignore pattern is never picked up while build output and every other ignored artefact stay out
// of the answer.
func (r *Repo) UntrackedPaths(pathspec string) ([]string, error) {
	out, err := runGit(r.root, "ls-files", "-z", "--others", "--exclude-standard", "--", pathspec)
	if err != nil {
		return nil, err
	}
	return splitNullDelimited(out), nil
}

// runGit runs one git invocation against root, passed explicitly via git's own "-C" flag rather
// than through the process's own working directory, matching how the existing tests in this
// repository already invoke git: a caller running the same query many times over never depends on
// which directory the process happens to be started in.
//
// It captures standard output and returns it whole. On failure it returns an error naming the
// failing subcommand and carrying git's own stderr text, with no prefix reading like an internal
// package path: the command-line layer carries a failed git command's message whole behind its own
// internal-error prefix, so a second, internal prefix here would double it.
func runGit(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		subcommand := ""
		if len(args) > 0 {
			subcommand = args[0]
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", subcommand, msg)
	}
	return stdout.Bytes(), nil
}
