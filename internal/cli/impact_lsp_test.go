//go:build lsp

// impact_lsp_test.go exercises the "impact" verb against a real, held-open gopls subprocess to
// prove the brief's central requirement: every caller entry's "enclosing_range" reaches back over
// the enclosing declaration's docstring, not merely its "func" line. This depends on gopls'
// real textDocument/references resolution over a real, on-disk fixture tree, so it is not
// reproducible against a fake server. The test is guarded on exec.LookPath("gopls") (via t.Skip),
// exactly like assertnocallers_lsp_test.go's own live-tier test.

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// methodDeclarationPattern matches a Go method declaration carrying a receiver — "func (recv
// *Type) Name(" — and captures the method name, distinct from interfaceMethodPattern's bare,
// receiver-less interface-method line shape.
var methodDeclarationPattern = regexp.MustCompile(`^func\s*\([^)]*\)\s*([A-Za-z_][A-Za-z0-9_]*)\(`)

// findMethodDeclarationPosition returns the quarry.Position of a Go method declaration (e.g. "func
// (inv *Invoice) ApplyDiscount(...)") in file, located by scanning the source for
// methodDeclarationPattern rather than a hard-coded line number, mirroring
// findInterfaceMethodPosition's own scan-don't-hard-code discipline for the receiver-method shape
// that helper's pattern does not match.
func findMethodDeclarationPosition(t *testing.T, file, methodName string) (line, character int) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("findMethodDeclarationPosition: read %s: %v", file, err)
	}
	for i, l := range strings.Split(string(data), "\n") {
		m := methodDeclarationPattern.FindStringSubmatchIndex(l)
		if m == nil {
			continue
		}
		name := l[m[2]:m[3]]
		if name != methodName {
			continue
		}
		return i + 1, m[2] + 1
	}
	t.Fatalf("findMethodDeclarationPosition: no method declaration of %q found in %s", methodName, file)
	return 0, 0
}

// docCommentStartLine returns the 1-based line number of the first line of the "//" doc-comment
// block immediately preceding funcName's "func" declaration line in file (with or without a
// receiver), or the declaration line itself when it carries no such block. It is the scan this
// test uses to prove a caller's "enclosing_range.start_line" lands on the docstring rather than the
// "func" line, without hard-coding either line number.
func docCommentStartLine(t *testing.T, file, funcName string) int {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("docCommentStartLine: read %s: %v", file, err)
	}
	declPattern := regexp.MustCompile(`^func\s*(\([^)]*\)\s*)?` + regexp.QuoteMeta(funcName) + `\(`)
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		if !declPattern.MatchString(l) {
			continue
		}
		start := i
		for start > 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "//") {
			start--
		}
		return start + 1
	}
	t.Fatalf("docCommentStartLine: no declaration of %q found in %s", funcName, file)
	return 0
}

// TestImpact_DocstringInclusiveEnclosingRange_Integration proves the impact verb's central claim
// against a real gopls: every caller of billing.Invoice.ApplyDiscount is reported with its
// enclosing declaration's range reaching back over that declaration's own docstring, the two call
// sites inside ProcessRefund share one identical enclosing_range while carrying distinct
// call_site_line values, and the declaration site itself is never reported as a caller.
func TestImpact_DocstringInclusiveEnclosingRange_Integration(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found on $PATH; install with: go install golang.org/x/tools/gopls@latest")
	}

	root := repoRoot(t)
	fixtureRoot := filepath.Join(root, "testdata", "impactfixture")
	declFile := filepath.Join(fixtureRoot, "billing", "invoice.go")
	callerFile := filepath.Join(fixtureRoot, "refund", "refund.go")

	line, character := findMethodDeclarationPosition(t, declFile, "ApplyDiscount")
	posArg := fmt.Sprintf("%s:%d:%d", declFile, line, character)

	// Go's registry entry has HasNativeDaemon true, so the run below spawns a detached,
	// supervised daemon that lives for a ten-minute idle timeout and that the supervised
	// teardown branch deliberately never kills. This test cannot use daemontest (an
	// engine-internal package internal/cli may not import), so it reaps through the
	// sanctioned quarry facade instead, exactly as assertnocallers_lsp_test.go's own
	// live-tier test does.
	stateDir := t.TempDir()
	t.Cleanup(func() { killRecordedDaemonViaFacade(t, stateDir) })

	args := []string{"impact", posArg, "--target-dir", fixtureRoot, "--state-dir", stateDir, "--timeout", "60s"}
	var out bytes.Buffer
	exitCode := RunCLI(&out, args)

	var env map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &env); err != nil {
		t.Fatalf("impact output is not valid JSON: %v; got: %q", err, out.String())
	}

	if exitCode != 0 {
		t.Fatalf("impact exit = %d; want 0. envelope: %v", exitCode, env)
	}
	if ok, _ := env["ok"].(bool); !ok {
		t.Fatalf("impact envelope %v carries ok != true", env)
	}
	if resolution, _ := env["resolution"].(string); resolution != "complete" {
		t.Errorf("impact envelope resolution = %q; want %q", resolution, "complete")
	}

	target, ok := env["target"].(map[string]any)
	if !ok {
		t.Fatalf("impact envelope %v carries no target object", env)
	}
	if name, _ := target["name"].(string); name != "ApplyDiscount" {
		t.Errorf("target.name = %q; want %q", name, "ApplyDiscount")
	}
	if owner, _ := target["owner"].(string); owner != "Invoice" {
		t.Errorf("target.owner = %q; want %q", owner, "Invoice")
	}
	if pkg, _ := target["package"].(string); pkg != "billing" {
		t.Errorf("target.package = %q; want %q", pkg, "billing")
	}

	definition, ok := env["definition"].(map[string]any)
	if !ok {
		t.Fatalf("impact envelope %v carries no definition object", env)
	}
	defLine, _ := definition["line"].(float64)
	defStartLine, _ := definition["start_line"].(float64)
	if !(defStartLine < defLine) {
		t.Errorf("definition.start_line = %v; want strictly less than definition.line = %v (the declaration's own docstring range)", defStartLine, defLine)
	}

	rawCallers, ok := env["callers"].([]any)
	if !ok {
		t.Fatalf("impact envelope %v carries no callers array", env)
	}
	callers := make([]map[string]any, len(rawCallers))
	for i, c := range rawCallers {
		entry, ok := c.(map[string]any)
		if !ok {
			t.Fatalf("callers[%d] = %v; want a JSON object", i, c)
		}
		callers[i] = entry
	}

	// Declaration exclusion survives the real round trip: no reported caller's file is the
	// declaration site itself.
	for _, c := range callers {
		file, _ := c["file"].(string)
		if filepath.Clean(file) == filepath.Clean(declFile) {
			t.Errorf("callers = %+v; want no entry naming the declaration site %q", callers, declFile)
		}
	}

	// One entry per call site: ProcessRefund calls ApplyDiscount twice (two entries sharing
	// one enclosing_range but distinct call_site_line values) and Reconcile calls it once.
	const wantCallers = 3
	if len(callers) != wantCallers {
		t.Fatalf("len(callers) = %d; want %d (three call sites across ProcessRefund and Reconcile)", len(callers), wantCallers)
	}

	processRefundDocStart := docCommentStartLine(t, callerFile, "ProcessRefund")
	reconcileDocStart := docCommentStartLine(t, callerFile, "Reconcile")

	var processRefundEntries []map[string]any
	var reconcileEntries []map[string]any
	for _, c := range callers {
		name, _ := c["name"].(string)
		switch name {
		case "ProcessRefund":
			processRefundEntries = append(processRefundEntries, c)
		case "Reconcile":
			reconcileEntries = append(reconcileEntries, c)
		default:
			t.Errorf("caller entry %+v carries unexpected enclosing name %q; want %q or %q", c, name, "ProcessRefund", "Reconcile")
		}

		enclosingRange, ok := c["enclosing_range"].(map[string]any)
		if !ok {
			t.Fatalf("caller entry %+v carries no enclosing_range object", c)
		}
		startLine, _ := enclosingRange["start_line"].(float64)

		// The brief's central requirement: the enclosing declaration's range reaches back
		// over its docstring, located by scanning the source, never by a hard-coded line
		// number.
		var wantStart int
		switch name {
		case "ProcessRefund":
			wantStart = processRefundDocStart
		case "Reconcile":
			wantStart = reconcileDocStart
		}
		if int(startLine) != wantStart {
			t.Errorf("caller %+v enclosing_range.start_line = %v; want %d (the docstring's own first line, not the func line)", c, startLine, wantStart)
		}
	}

	if len(processRefundEntries) != 2 {
		t.Fatalf("len(ProcessRefund caller entries) = %d; want 2 (it calls ApplyDiscount on two distinct lines)", len(processRefundEntries))
	}
	if len(reconcileEntries) != 1 {
		t.Fatalf("len(Reconcile caller entries) = %d; want 1", len(reconcileEntries))
	}

	firstRange := processRefundEntries[0]["enclosing_range"]
	secondRange := processRefundEntries[1]["enclosing_range"]
	firstMap, _ := firstRange.(map[string]any)
	secondMap, _ := secondRange.(map[string]any)
	if firstMap["start_line"] != secondMap["start_line"] || firstMap["end_line"] != secondMap["end_line"] {
		t.Errorf("ProcessRefund's two caller entries carry different enclosing_range values: %v vs %v; want identical ranges", firstRange, secondRange)
	}

	firstCallLine, _ := processRefundEntries[0]["call_site_line"].(float64)
	secondCallLine, _ := processRefundEntries[1]["call_site_line"].(float64)
	if firstCallLine == secondCallLine {
		t.Errorf("ProcessRefund's two caller entries share call_site_line = %v; want distinct values for the two distinct call sites", firstCallLine)
	}
}
