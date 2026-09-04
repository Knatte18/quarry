// root.go declares ResolveRoot, the startup-time repository root resolution cmd/quarry-mcp calls
// exactly once before the transport starts, and rootErrorMessage, the sentinel-to-sentence mapping
// it delegates to.

package mcpserver

import (
	"errors"
	"fmt"

	"github.com/Knatte18/quarry/internal/repopath"
)

// rootErrorMessage maps a repopath.ResolveRoot failure to this surface's own startup-failure
// wording, formatted from the sentinel rather than echoed from repopath's own error text, since
// that text is namespaced to the repopath package and would leak an internal package name into
// this binary's diagnostics. It returns the message and true when err wraps
// repopath.ErrNoRepositoryRoot or repopath.ErrRootNotDirectory, and ("", false) for any other
// error, including nil.
//
// It is a named function, mirroring internal/cli's rootUsageMessage, for the same reason that one
// is: the no-repository-root sentence is unreachable from inside a repository without changing the
// process working directory, which these tests never do, so a table test over the formatter is the
// only way to pin it.
func rootErrorMessage(err error, flagRoot, cwd string) (string, bool) {
	if errors.Is(err, repopath.ErrNoRepositoryRoot) {
		return "quarry-mcp: no repository root found above " + cwd + "; pass --root", true
	}
	if errors.Is(err, repopath.ErrRootNotDirectory) {
		return "quarry-mcp: --root is not a directory: " + flagRoot, true
	}
	return "", false
}

// ResolveRoot resolves the repository root for this process's whole lifetime: discovery from cwd
// when flagRoot is empty, or flagRoot itself otherwise, exactly as repopath.ResolveRoot resolves
// it. On success it returns repopath.ResolveRoot's value unchanged, which is always absolute. On
// failure it returns an error carrying rootErrorMessage's sentence when that function reports
// true, and otherwise the underlying error wrapped behind a "quarry-mcp: " prefix.
//
// ResolveRoot is a named, exported symbol rather than logic inlined into cmd/quarry-mcp's main for
// one stated reason: it is the --root path, which is what rescues a falsified cwd-inheritance
// assumption (discussion D3) — the live probe does not exercise it — and cmd/quarry-mcp must stay
// untestable-by-design-because-trivial, with every piece of logic it would otherwise hold pulled
// into this package instead, where it is testable in-process.
//
// ResolveRoot takes cwd as a parameter rather than calling os.Getwd itself, so its tests need no
// process-global state; cmd/quarry-mcp's main is the one place that reads the working directory.
func ResolveRoot(flagRoot, cwd string) (string, error) {
	root, err := repopath.ResolveRoot(flagRoot, cwd)
	if err != nil {
		if msg, ok := rootErrorMessage(err, flagRoot, cwd); ok {
			return "", errors.New(msg)
		}
		return "", fmt.Errorf("quarry-mcp: %w", err)
	}
	return root, nil
}
