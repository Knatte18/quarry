//go:build lsp

// assertnocallers_lsp_test.go exercises "assert-no-callers" against a real, held-open gopls
// subprocess to prove the interface-method conflation fix issue #1 describes: by default,
// assert-no-callers' declaration-based verification excludes every structurally-identical but
// unrelated interface's own call sites, and --no-verify reinstates the older, noisier unfiltered
// behaviour. This is gopls's own textDocument/implementation behaviour, not this wrapper's wiring,
// so it is not reproducible against a fake server. The test is guarded on exec.LookPath("gopls")
// (via t.Skip), exactly like internal/quarryengine/query's own live-tier tests.

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/internal/proc"
	"github.com/Knatte18/quarry/quarry"
)

// interfaceMethodPattern matches a bare interface-method declaration line (no receiver, no "func"
// keyword) and captures the method name — the shape an interface body's method signature takes,
// distinct from a top-level "func Name(" declaration.
var interfaceMethodPattern = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\(`)

// findInterfaceMethodPosition returns the quarry.Position of an interface method declaration
// (e.g. "Now() time.Time" inside a "type clock interface { ... }" body) in file, located by
// scanning the source rather than a hard-coded line number.
func findInterfaceMethodPosition(t *testing.T, file, methodName string) quarry.Position {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("findInterfaceMethodPosition: read %s: %v", file, err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		m := interfaceMethodPattern.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		name := line[m[4]:m[5]]
		if name != methodName {
			continue
		}
		return quarry.Position{File: file, Line: i + 1, Character: m[4] + 1}
	}
	t.Fatalf("findInterfaceMethodPosition: no interface method declaration of %q found in %s", methodName, file)
	return quarry.Position{}
}

// repoRoot returns this worktree's module root, scanning up from this file's own location —
// internal/cli/assertnocallers_lsp_test.go is two directories below the repo root — mirroring
// internal/quarryengine/query/refs_integration_test.go's own repoRoot helper, which cli cannot
// import (per the layering-is-non-negotiable Shared Decision, internal/cli reaches the engine only
// through quarry/facade.go).
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("repoRoot: could not determine quarry source directory location")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// TestAssertNoCallers_InterfaceConflation_Integration proves assert-no-callers' default
// declaration-verified caller set excludes runner's and sched's own structurally-identical but
// unrelated clock.Now call sites, matches the callers-verified figure
// docs/implementation-widening-spike.md records, and that --no-verify reinstates the wider,
// unfiltered set including those call sites — pinning both the fix and its escape hatch against a
// live gopls.
func TestAssertNoCallers_InterfaceConflation_Integration(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found on $PATH; install with: go install golang.org/x/tools/gopls@latest")
	}

	root := repoRoot(t)
	fixtureRoot := filepath.Join(root, "testdata", "clockfixture")
	tickFile := filepath.Join(fixtureRoot, "runner", "tick.go")
	waitFile := filepath.Join(fixtureRoot, "sched", "wait.go")

	pos := findInterfaceMethodPosition(t, filepath.Join(fixtureRoot, "builder", "poll.go"), "Now")
	posArg := fmt.Sprintf("%s:%d:%d", pos.File, pos.Line, pos.Character)

	// Go's registry entry has HasNativeDaemon true, so the first run below spawns a
	// detached, supervised daemon that lives for a ten-minute idle timeout and that the
	// supervised teardown branch deliberately never kills. This test cannot use
	// daemontest (an engine-internal package internal/cli may not import), so it reaps
	// through the sanctioned quarry facade instead: resolve the state file with
	// quarry.DaemonStateFile, decode its recorded pid, and kill it via proc.KillPID.
	// Both runs below share this one --state-dir, so there is exactly one daemon to reap.
	stateDir := t.TempDir()
	t.Cleanup(func() { killRecordedDaemonViaFacade(t, stateDir) })

	verifiedCallers, verifiedExit := runAssertNoCallers(t, fixtureRoot, stateDir, posArg, false)

	for _, c := range verifiedCallers {
		file, _ := c["file"].(string)
		if filepath.Clean(file) == filepath.Clean(tickFile) || filepath.Clean(file) == filepath.Clean(waitFile) {
			t.Errorf("assert-no-callers (verified) callers = %+v; want no call site from runner/tick.go or sched/wait.go", verifiedCallers)
		}
		if filepath.Base(filepath.Dir(file)) != "builder" {
			t.Errorf("assert-no-callers (verified) callers = %+v; want every entry inside the builder package, got file %q", verifiedCallers, file)
		}
	}

	// callers-verified: 2, per docs/implementation-widening-spike.md's recorded figure for
	// this exact position — references-verified (3) minus the one declaration site
	// filterUnexpectedCallers removes. Not the spike's references-unfiltered (7, which
	// still includes the declaration) or references-verified (3) figures, and not issue
	// #1's pre-fix "31 -> 2" measurement, taken against a different repository.
	const wantCallersVerified = 2
	if len(verifiedCallers) != wantCallersVerified {
		t.Errorf("assert-no-callers (verified) len(callers) = %d; want %d (docs/implementation-widening-spike.md's callers-verified figure)", len(verifiedCallers), wantCallersVerified)
	}
	if verifiedExit != 1 {
		t.Errorf("assert-no-callers (verified) exit = %d; want 1 (violation:true whenever callers is non-empty)", verifiedExit)
	}

	unverifiedCallers, unverifiedExit := runAssertNoCallers(t, fixtureRoot, stateDir, posArg, true)

	if len(unverifiedCallers) <= len(verifiedCallers) {
		t.Errorf("assert-no-callers --no-verify len(callers) = %d; want strictly more than the verified count %d", len(unverifiedCallers), len(verifiedCallers))
	}

	foundOutOfScope := false
	for _, c := range unverifiedCallers {
		file, _ := c["file"].(string)
		if filepath.Clean(file) == filepath.Clean(tickFile) || filepath.Clean(file) == filepath.Clean(waitFile) {
			foundOutOfScope = true
			break
		}
	}
	if !foundOutOfScope {
		t.Errorf("assert-no-callers --no-verify callers = %+v; want at least one call site from runner/tick.go or sched/wait.go", unverifiedCallers)
	}
	if unverifiedExit != 1 {
		t.Errorf("assert-no-callers --no-verify exit = %d; want 1 (violation:true whenever callers is non-empty)", unverifiedExit)
	}
}

// runAssertNoCallers runs "assert-no-callers" against posArg (a "file:line:col" positional
// argument) within fixtureRoot, using stateDir as the shared --state-dir, and returns the decoded
// "callers" list from the JSON envelope alongside the process exit code. noVerify threads
// --no-verify when true.
func runAssertNoCallers(t *testing.T, fixtureRoot, stateDir, posArg string, noVerify bool) ([]map[string]any, int) {
	t.Helper()

	args := []string{"assert-no-callers", posArg, "--target-dir", fixtureRoot, "--state-dir", stateDir, "--timeout", "60s"}
	if noVerify {
		args = append(args, "--no-verify")
	}

	var out bytes.Buffer
	exitCode := RunCLI(&out, args)

	var env map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &env); err != nil {
		t.Fatalf("assert-no-callers output is not valid JSON: %v; got: %q", err, out.String())
	}

	rawCallers, _ := env["callers"].([]any)
	callers := make([]map[string]any, len(rawCallers))
	for i, c := range rawCallers {
		entry, ok := c.(map[string]any)
		if !ok {
			t.Fatalf("assert-no-callers callers[%d] = %v; want a JSON object", i, c)
		}
		callers[i] = entry
	}

	if violation, _ := env["violation"].(bool); !violation && len(callers) > 0 {
		t.Errorf("assert-no-callers envelope %v carries a non-empty callers list without violation:true", env)
	}

	return callers, exitCode
}

// killRecordedDaemonViaFacade reaps the supervised Go daemon recorded under stateDir, reading its
// state file and pid through the sanctioned quarry facade rather than the engine-internal
// daemontest helper internal/cli may not import. An already-dead process's KillPID error is the
// expected outcome here, not a test failure, since this runs from t.Cleanup.
func killRecordedDaemonViaFacade(t *testing.T, stateDir string) {
	t.Helper()

	statePath := quarry.DaemonStateFile(stateDir, "go")
	data, err := os.ReadFile(statePath)
	if err != nil {
		// No state file recorded (e.g. the daemon never started): nothing to reap.
		return
	}

	var state struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Logf("killRecordedDaemonViaFacade: unmarshal %s: %v", statePath, err)
		return
	}
	if state.PID == 0 {
		return
	}

	// An error here (e.g. os.ErrProcessDone, or "no such process") is the expected
	// outcome when the daemon has already exited on its own; it is deliberately not
	// treated as a test failure.
	_ = proc.KillPID(state.PID)
}
