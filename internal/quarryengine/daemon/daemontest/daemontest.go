// daemontest exports test-only helpers that drive daemon's exported injection points and inspect
// its recorded state from OUTSIDE package daemon.
// It carries no production code and exists exclusively because daemon's own in-package tests
// cannot import it: Go rejects an in-package test file that imports a package which itself
// imports the package under test (daemon [test] -> daemontest -> daemon fails to build with
// "import cycle not allowed in test"), so daemon's own toolchain_test.go and
// supervised_integration_test.go keep their own copies of withFakeInstaller,
// withTempUserCacheDir, and killRecordedDaemon unchanged, deliberately duplicated rather than
// routed through this package.
// Today the only callers outside package daemon are query's refs_test.go, refs_integration_test.go,
// callers_test.go, and buildtags_lsp_test.go.
// This package is imported only from _test.go files.

package daemontest

import (
	"os"
	"testing"

	"github.com/Knatte18/quarry/internal/quarryengine/daemon"
)

// WithFakeInstaller replaces daemon.InstallGoToolchain with fake for the duration of the calling
// test and restores the previous value via t.Cleanup, mirroring toolchain_test.go's in-package
// withFakeInstaller helper for callers outside package daemon.
func WithFakeInstaller(t *testing.T, fake daemon.ToolchainInstaller) {
	t.Helper()
	original := daemon.InstallGoToolchain
	daemon.InstallGoToolchain = fake
	t.Cleanup(func() { daemon.InstallGoToolchain = original })
}

// WithTempUserCacheDir replaces daemon.UserCacheDir with a closure returning a fresh t.TempDir()
// for the duration of the calling test, restores the previous value via t.Cleanup, and returns
// that directory, mirroring toolchain_test.go's in-package withTempUserCacheDir helper for
// callers outside package daemon.
func WithTempUserCacheDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	original := daemon.UserCacheDir
	daemon.UserCacheDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { daemon.UserCacheDir = original })
	return dir
}

// StateFile delegates to daemon.DaemonStateFile, letting a caller outside package daemon resolve
// a language's state-file path without importing daemon directly.
func StateFile(stateDir, lang string) string {
	return daemon.DaemonStateFile(stateDir, lang)
}

// ConnKindNative, ConnKindSupervised, and ConnKindLegacy re-export daemon.ConnKind's three values,
// letting a package which cannot import daemon under the layering DAG (query's _test.go files,
// per layering_test.go's table) still name a daemon.ConnKind — mirroring why StateFile re-exports
// daemon.DaemonStateFile above.
const (
	ConnKindNative     = daemon.ConnKindNative
	ConnKindSupervised = daemon.ConnKindSupervised
	ConnKindLegacy     = daemon.ConnKindLegacy
)

// KillRecordedDaemon kills the daemon PID recorded in the state file at statePath, ported from
// supervised_integration_test.go's in-package killRecordedDaemon against daemon's exported
// ReadState and State.PID. It silently returns on any error or a not-found state: it runs from
// t.Cleanup, where a failure to find a dead daemon is not itself a test failure.
func KillRecordedDaemon(t *testing.T, statePath string) {
	t.Helper()
	state, found, err := daemon.ReadState(statePath)
	if err != nil || !found {
		return
	}
	process, err := os.FindProcess(state.PID)
	if err != nil {
		return
	}
	_ = process.Kill()
	_, _ = process.Wait()
}
