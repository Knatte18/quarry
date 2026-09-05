// name_golden_test.go pins the name verb's own goldens, both views, across the six cases the
// maker's reason vocabulary and answer shape require: a method, a free function, a type, a method
// on a receiver type that does not exist, a malformed declaration, and a fragment declaring
// several symbols. Every case needs no environment gate of any kind: the maker reads no
// repository, so this table runs everywhere, unlike after_test.go's own goldens, which need a
// pinned Loomyard checkout.
//
// Each golden file holds the payload bytes and nothing else, per the overview's
// goldens-are-payload-bytes-only decision: no invocation header, unlike the frozen research
// goldens under docs/research/output-formats/after/, which are read as evidence documents rather
// than regression fixtures.

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// nameGoldenCase is one row of the name/ golden table: the golden file's base name (shared by its
// .json and .txt siblings), the unit and declaration to pass through Run, and the exit code Run is
// expected to return for both views.
type nameGoldenCase struct {
	name     string
	unit     string
	decl     string
	exitCode int
}

// nameGoldenCases is testdata/name/'s own table. Every case runs both under JSON and under --text,
// producing the two files its name identifies.
var nameGoldenCases = []nameGoldenCase{
	{
		name:     "method",
		unit:     "pkg",
		decl:     "func (w Widget) Value() int { return 42 }",
		exitCode: exitOK,
	},
	{
		name:     "function",
		unit:     "pkg",
		decl:     "func Make() Widget { return Widget{} }",
		exitCode: exitOK,
	},
	{
		name:     "type",
		unit:     "pkg",
		decl:     "type Widget struct{}",
		exitCode: exitOK,
	},
	{
		name:     "unknown-receiver",
		unit:     "pkg",
		decl:     "func (g Ghost) Boo() {}",
		exitCode: exitOK,
	},
	{
		name:     "malformed",
		unit:     "pkg",
		decl:     "func Bad(",
		exitCode: exitNegative,
	},
	{
		name:     "multi-symbol",
		unit:     "pkg",
		decl:     "func A() {}\nfunc B() {}",
		exitCode: exitNegative,
	},
}

// TestNameGoldens runs every case in nameGoldenCases under both views and compares the resulting
// stdout bytes against the committed golden, or rewrites it under -update.
func TestNameGoldens(t *testing.T) {
	for _, tc := range nameGoldenCases {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, _ := runCLI([]string{"name", "--unit", tc.unit, tc.decl})
			if code != tc.exitCode {
				t.Fatalf("Run(name --unit %s %q) code = %d; want %d", tc.unit, tc.decl, code, tc.exitCode)
			}
			compareNameGolden(t, tc.name+".json", stdout)

			code, stdout, _ = runCLI([]string{"name", "--unit", tc.unit, tc.decl, "--text"})
			if code != tc.exitCode {
				t.Fatalf("Run(name --unit %s %q --text) code = %d; want %d", tc.unit, tc.decl, code, tc.exitCode)
			}
			compareNameGolden(t, tc.name+".txt", stdout)
		})
	}
}

// compareNameGolden compares got byte for byte against testdata/name/name, or — under -update —
// writes got to that path, creating the directory if it does not yet exist. It is shaped exactly
// like compareAfterGolden in after_test.go and deliberately does not call it: that helper
// hard-codes the frozen research path, and its own caller is gated on a Loomyard checkout, both
// wrong for a table that must run on a machine with no checkout.
func compareNameGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", "name", name)

	if *updateGoldens {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("compareNameGolden(%q): mkdir: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("compareNameGolden(%q): write: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("compareNameGolden(%q): read golden: %v", name, err)
	}
	if string(want) != got {
		t.Errorf("golden %q mismatch (-want +got):\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}
