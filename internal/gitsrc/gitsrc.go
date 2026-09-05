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
