//go:build lsp

// ensureserver_integration_test.go exercises ensureNative against a real,
// network-installed gopls, mirroring refs_integration_test.go's and
// toolchain_integration_test.go's //go:build lsp-tagged,
// t.Skip(registry.BuiltinRegistry()["go"].InstallHint)-gated style: the tag names its real
// precondition, a real language-server binary on $PATH, so this file is
// excluded from the plain `go test` verify and run separately with
// `-tags lsp`. Even though ensureNative itself ignores $PATH and resolves
// its own toolchain-managed binary, the skip gate here is about whether
// this machine can plausibly run a real gopls at all (network + `go
// install` capability), which exec.LookPath("gopls") is a reasonable,
// cheap proxy for reusing rather than inventing a second capability probe.
// This test spawns no git and needs no git-environment isolation — it
// spawns only gopls.
//
// It also now covers EnsureServer's supervised dispatch: since the
// engine-supervised-flip batch, EnsureServer routes Go through
// ensureSupervised first, falling back to ensureNative (still directly
// exercised above) only on failure. TestEnsureServer_Integration_
// SupervisedDispatch below drives EnsureServer itself end to end and
// asserts it lands on the supervised strategy and reuses one daemon across
// calls, complementing supervised_integration_test.go's direct
// ensureSupervised coverage.

package daemon

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
)

// errNoWorkspaceSymbolCandidates reports when workspace/symbol returns no candidates.
var errNoWorkspaceSymbolCandidates = errors.New("workspace/symbol returned zero candidates; want at least one")

// repoRoot returns the repository root, resolved relative to this file's own location via
// runtime.Caller(0) rather than the process's cwd. It walks four filepath.Dir levels, not the
// two a sibling copy in refs_integration_test.go used before the engine-repackage move — this
// file now sits at internal/quarryengine/daemon/, three directories below the repo root rather
// than one, so two levels would silently resolve to internal/quarryengine/ instead of the module
// root. query keeps its own copy in refs_integration_test.go, walking the same four levels from
// its own new location; daemon cannot import query, so this copy is duplicated rather than
// shared, per the test-support-helpers Shared Decision.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("repoRoot: could not determine quarry source directory location")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}

// TestEnsureNative_Integration verifies ensureNative's chain works end-to-end against a real gopls.
func TestEnsureNative_Integration(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip(registry.BuiltinRegistry()["go"].InstallHint)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ensureNative(ctx, "go", registry.BuiltinRegistry()["go"], repoRoot(t), 30*time.Second)
	if err != nil {
		t.Fatalf("ensureNative() returned unexpected error: %v", err)
	}
	defer client.Kill()

	if client == nil {
		t.Fatal("ensureNative() returned a nil client with a nil error")
	}
	if client.Closed() {
		t.Error("ensureNative() returned a client with closed = true; want an open, ready-to-use connection")
	}
}

// TestEnsureNative_Integration_SharedDaemonWireCompatibility verifies two ensureNative calls share
// the same daemon.
func TestEnsureNative_Integration_SharedDaemonWireCompatibility(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip(registry.BuiltinRegistry()["go"].InstallHint)
	}

	root := repoRoot(t)
	entry := registry.BuiltinRegistry()["go"]

	type result struct {
		location string
		err      error
	}
	results := make([]result, 2)

	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			client, err := ensureNative(ctx, "go", entry, root, 30*time.Second)
			if err != nil {
				results[i] = result{err: err}
				return
			}
			defer client.Kill()

			symbolCtx, symbolCancel := context.WithTimeout(ctx, 30*time.Second)
			defer symbolCancel()
			candidates, err := client.WorkspaceSymbol(symbolCtx, "Resolve")
			if err != nil {
				results[i] = result{err: err}
				return
			}
			if len(candidates) == 0 {
				results[i] = result{err: errNoWorkspaceSymbolCandidates}
				return
			}
			results[i] = result{location: lsp.FormatLocation(candidates[0].Location)}
		}(i)
	}
	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			t.Fatalf("connection %d: workspace/symbol(\"Resolve\") failed: %v", i, r.err)
		}
	}
	if results[0].location != results[1].location {
		t.Errorf("connection 0 resolved %q, connection 1 resolved %q; want identical first-candidate locations, proving both connections share one gopls daemon's index", results[0].location, results[1].location)
	}
}

// TestEnsureServer_Integration_SupervisedDispatch proves EnsureServer's live dispatch decision end
// to end against a real gopls, not just the mocked-transport unit coverage in ensureserver_test.go:
// since Go's registry entry now dispatches to the supervised strategy first, a call through
// EnsureServer (not ensureSupervised directly, which supervised_integration_test.go already covers)
// must come back with ConnKindSupervised, must have recorded a live daemon state file, and a second
// call against the same anchorRoot must reuse that same daemon (identical PID, stable Address)
// rather than spawning a second one — mirroring TestEnsureSupervised_Integration's own reuse
// assertions, but through the exact entry point production code calls.
func TestEnsureServer_Integration_SupervisedDispatch(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip(registry.BuiltinRegistry()["go"].InstallHint)
	}

	root := repoRoot(t)
	// A fresh temp worktree per test run keeps this test's daemon
	// isolated from any other scout test's own supervised daemon,
	// matching TestEnsureSupervised_Integration's own isolation.
	worktreeRoot := t.TempDir()
	statePath := DaemonStateFile(worktreeRoot, "go")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, kind, err := EnsureServer(ctx, "go", registry.BuiltinRegistry()["go"], root, worktreeRoot, 30*time.Second)
	if err != nil {
		t.Fatalf("EnsureServer() returned unexpected error: %v", err)
	}
	// Per the ConnKindSupervised teardown rule, a supervised connection must
	// never be close()'d or kill()'d — the daemon it dials into is meant to
	// outlive this call. The state-file PID read below is what this test
	// reaps instead, exactly like supervised_integration_test.go's own
	// killRecordedDaemon.
	t.Cleanup(func() { killRecordedDaemon(t, statePath) })

	if kind != ConnKindSupervised {
		t.Errorf("EnsureServer() ConnKind = %v; want ConnKindSupervised (native is now the fallback path only)", kind)
	}
	if client == nil {
		t.Fatal("EnsureServer() returned a nil client with a nil error")
	}

	state, found, err := ReadState(statePath)
	if err != nil {
		t.Fatalf("ReadState() failed: %v", err)
	}
	if !found {
		t.Fatal("ReadState() found = false after EnsureServer() succeeded; want true")
	}
	if _, err := os.FindProcess(state.PID); err != nil {
		t.Fatalf("os.FindProcess(%d) failed: %v", state.PID, err)
	}

	// A second call against the same anchorRoot/lang must reuse the
	// existing daemon rather than spawning a second one.
	if _, _, err := EnsureServer(ctx, "go", registry.BuiltinRegistry()["go"], root, worktreeRoot, 30*time.Second); err != nil {
		t.Fatalf("EnsureServer() second call returned unexpected error: %v", err)
	}

	state2, found, err := ReadState(statePath)
	if err != nil {
		t.Fatalf("ReadState() failed: %v", err)
	}
	if !found {
		t.Fatal("ReadState() found = false after the second EnsureServer() call; want true")
	}
	if state2.PID != state.PID {
		t.Errorf("state.PID after the second EnsureServer() call = %d; want unchanged %d (reuse, not a second spawn)", state2.PID, state.PID)
	}
	if state2.Address != state.Address {
		t.Errorf("state.Address after the second EnsureServer() call = %q; want unchanged %q", state2.Address, state.Address)
	}
}
