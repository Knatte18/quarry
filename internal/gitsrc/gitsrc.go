// gitsrc.go declares Repo, the opened-repository handle, and every read-only git operation this
// package exposes. Every operation goes through runGit, the one helper that invokes git against an
// explicit repository root, so the root-passing rule and the failure-wrapping rule each have one
// implementation.

package gitsrc

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

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
