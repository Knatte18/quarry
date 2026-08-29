//go:build lsp

// refs_targetdir_lsp_test.go exercises "refs" against a real, held-open gopls subprocess to prove
// a relative "file:line:col" positional argument resolves against --target-dir, not the process's
// own cwd. This is the regression parseQuery/inFileQuery's RunE call sites (refs, definition,
// assert-no-callers, impact) used to have: "base" was hardcoded to the seam cwd even when
// --target-dir pointed the query at a different directory entirely, breaking a relative position
// argument the moment cwd and --target-dir diverged -- the exact call shape every other
// --target-dir-relative flag (--within, --in-file) already worked with in the same RunE. This
// depends on gopls' real textDocument/references resolution over a real, on-disk fixture tree, so
// it is not reproducible against a fake server. The test is guarded on exec.LookPath("gopls") (via
// t.Skip), exactly like assertnocallers_lsp_test.go's own live-tier test.

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRunCLI_Refs_RelativePositionResolvesAgainstTargetDirNotCwd proves "refs <relative
// file:line:col> --target-dir <dir>" resolves the position against <dir>, not the process's cwd,
// by running from a cwd deliberately unrelated to the fixture module.
func TestRunCLI_Refs_RelativePositionResolvesAgainstTargetDirNotCwd(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found on $PATH; install with: go install golang.org/x/tools/gopls@latest")
	}

	root := repoRoot(t)
	fixtureRoot := filepath.Join(root, "testdata", "clockfixture")
	pollFile := filepath.Join(fixtureRoot, "builder", "poll.go")

	pos := findInterfaceMethodPosition(t, pollFile, "Now")
	relFile, err := filepath.Rel(fixtureRoot, pos.File)
	if err != nil {
		t.Fatalf("filepath.Rel(%q, %q) failed: %v", fixtureRoot, pos.File, err)
	}
	posArg := fmt.Sprintf("%s:%d:%d", relFile, pos.Line, pos.Character)

	// cwd is deliberately unrelated to fixtureRoot -- the old cwd-based resolution would try
	// (and fail) to open relFile under this directory instead of under --target-dir.
	cwd := t.TempDir()
	stateDir := t.TempDir()
	t.Cleanup(func() { killRecordedDaemonViaFacade(t, stateDir) })

	var out bytes.Buffer
	exitCode := RunCLIIn(cwd, &out, []string{"refs", posArg, "--target-dir", fixtureRoot, "--state-dir", stateDir, "--timeout", "60s"})

	var env map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &env); err != nil {
		t.Fatalf("refs output is not valid JSON: %v; got: %q", err, out.String())
	}

	if ok, _ := env["ok"].(bool); !ok {
		t.Fatalf("RunCLIIn(refs %s --target-dir %s) ok = false (exit %d); want true -- the relative position must resolve against --target-dir. envelope: %v", posArg, fixtureRoot, exitCode, env)
	}
	if exitCode != 0 {
		t.Errorf("RunCLIIn(refs %s --target-dir %s) exit = %d; want 0", posArg, fixtureRoot, exitCode)
	}

	refs, _ := env["references"].([]any)
	if len(refs) == 0 {
		t.Errorf("references = %v; want at least one reference to the Now interface method", refs)
	}
}
