// worktree.go ports the git seam and the two worktree-filesystem gates from scripts/gates.py and
// scripts/run_ladder.py: the single git(1) runner every git call in the suite goes through, the
// ambient-context presence gate, and the worktree-dirtied observation. Building and restoring a task
// worktree, which also calls through RunGit, lands in a later batch alongside the session machinery that
// drives them.

package ladder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
