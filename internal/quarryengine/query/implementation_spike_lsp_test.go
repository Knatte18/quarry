//go:build lsp

// implementation_spike_lsp_test.go is the measurement harness the plan's Batch Scope requires
// before any verification-filter code is written: it drives a real, held-open gopls against the
// three-package clock fixture (testdata/clockfixture) and logs the raw
// textDocument/definition, textDocument/implementation, and textDocument/references results at
// two query positions, so the widening-mode decision recorded in
// docs/implementation-widening-spike.md is derived from an observed measurement rather than
// asserted in code. It asserts nothing about the counts themselves — see
// TestImplementationWidening_Spike's own doc comment for why.

package query

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
)

// interfaceMethodPattern matches an interface method signature line (e.g. "Now() time.Time"),
// which — unlike a concrete method or function declaration — carries no leading "func" keyword.
// It captures the method name.
var interfaceMethodPattern = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\([^)]*\)\s+\S`)

// findInterfaceMethodPosition returns the quarryengine.Position of a method signature declared
// inside an interface body (not a concrete method, which findFuncPosition already covers, since
// an interface method's line has no "func" keyword to key on).
func findInterfaceMethodPosition(t *testing.T, file, methodName string) quarryengine.Position {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("findInterfaceMethodPosition: read %s: %v", file, err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "func") {
			continue
		}
		m := interfaceMethodPattern.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		name := line[m[2]:m[3]]
		if name != methodName {
			continue
		}
		return quarryengine.Position{File: file, Line: i + 1, Character: m[2] + 1}
	}
	t.Fatalf("findInterfaceMethodPosition: no interface method %q found in %s", methodName, file)
	return quarryengine.Position{}
}

// logLocations writes one line per location in the "file:line:character" form the plan's finding
// document expects, grouped under a heading naming the query position and the LSP method that
// produced the list.
func logLocations(t *testing.T, heading string, locations []lsp.Location) {
	t.Helper()
	t.Logf("=== %s (%d locations) ===", heading, len(locations))
	if len(locations) == 0 {
		t.Logf("  (none)")
		return
	}
	for _, loc := range locations {
		t.Logf("  %s", lsp.FormatLocation(loc))
	}
}

// spikeQueryPosition issues textDocument/definition, textDocument/implementation, and
// textDocument/references at pos and logs every returned location. It fails the test (via
// t.Errorf, not t.Fatalf, so the remaining calls still run and log) only when a call itself
// errors — this is a measurement harness, not a correctness assertion, per
// TestImplementationWidening_Spike's own doc comment.
func spikeQueryPosition(ctx context.Context, t *testing.T, client *lsp.Client, label, fileURI string, wirePos lsp.Position) []lsp.Location {
	t.Helper()

	defs, err := client.Definition(ctx, fileURI, wirePos)
	if err != nil {
		t.Errorf("%s: textDocument/definition returned unexpected error: %v", label, err)
	} else {
		logLocations(t, label+" textDocument/definition", defs)
	}

	impls, err := client.Implementation(ctx, fileURI, wirePos)
	if err != nil {
		t.Errorf("%s: textDocument/implementation returned unexpected error: %v", label, err)
	} else {
		logLocations(t, label+" textDocument/implementation", impls)
	}

	refs, err := client.References(ctx, fileURI, wirePos)
	if err != nil {
		t.Errorf("%s: textDocument/references returned unexpected error: %v", label, err)
	} else {
		logLocations(t, label+" textDocument/references", refs)
	}

	return refs
}

// TestImplementationWidening_Spike is a measurement harness, not a correctness test: it logs the
// raw location lists textDocument/definition, textDocument/implementation, and
// textDocument/references return at two positions in the clock fixture, so
// docs/implementation-widening-spike.md can record an observed widening-mode decision (card 4)
// rather than an asserted one. It asserts only that each call returns without error, so a broken
// harness fails loudly instead of silently logging nothing — asserting a specific count here
// would defeat the point of running the spike at all.
func TestImplementationWidening_Spike(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip(registry.BuiltinRegistry()["go"].InstallHint)
	}

	root := repoRoot(t)
	fixtureRoot := filepath.Join(root, "testdata", "clockfixture")
	pollFile := filepath.Join(fixtureRoot, "builder", "poll.go")

	interfacePos := findInterfaceMethodPosition(t, pollFile, "Now")
	concretePos := findFuncPosition(t, pollFile, "Now")

	client, err := lsp.NewClient([]string{"gopls"})
	if err != nil {
		t.Fatalf("lsp.NewClient(gopls) returned unexpected error: %v", err)
	}
	defer client.Kill()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := client.Initialize(ctx, "file://"+fixtureRoot, nil); err != nil {
		t.Fatalf("client.Initialize(%s) returned unexpected error: %v", fixtureRoot, err)
	}

	interfaceWirePos, err := lsp.ToPosition(interfacePos)
	if err != nil {
		t.Fatalf("lsp.ToPosition(interface Now) returned unexpected error: %v", err)
	}
	concreteWirePos, err := lsp.ToPosition(concretePos)
	if err != nil {
		t.Fatalf("lsp.ToPosition(concrete Now) returned unexpected error: %v", err)
	}

	fileURI := "file://" + pollFile

	t.Logf("gopls version: run `gopls version` alongside this test to record it in docs/implementation-widening-spike.md")
	t.Logf("fixture root: %s", fixtureRoot)

	interfaceRefs := spikeQueryPosition(ctx, t, client, "interface-method clock.Now", fileURI, interfaceWirePos)

	// Per-reference definition set: for each location the interface-method
	// position's textDocument/references call returned, issue one
	// textDocument/definition at that location and log it alongside the
	// reference it resolves — this is exactly the per-reference definition
	// set the verification filter card 4 records counts from consumes; the
	// three position-level calls above cannot derive it on their own.
	t.Logf("=== interface-method clock.Now per-reference textDocument/definition (%d references) ===", len(interfaceRefs))
	for _, ref := range interfaceRefs {
		refDefs, err := client.Definition(ctx, ref.URI, ref.Range.Start)
		if err != nil {
			t.Errorf("per-reference definition for %s returned unexpected error: %v", lsp.FormatLocation(ref), err)
			continue
		}
		defsJSON, marshalErr := json.Marshal(refDefs)
		if marshalErr != nil {
			t.Errorf("marshal per-reference definition result for %s: %v", lsp.FormatLocation(ref), marshalErr)
			continue
		}
		t.Logf("  reference %s -> definition %s", lsp.FormatLocation(ref), string(defsJSON))
	}

	spikeQueryPosition(ctx, t, client, "concrete realClock.Now", fileURI, concreteWirePos)
}
