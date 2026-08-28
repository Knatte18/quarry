// python_test.go covers the Python Strategy's Symbols, Header, Package, Generated, and TestFile
// methods over inline fixtures, following the helper shape golang_test.go established.

package toc

import (
	"testing"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/quarryengine/treesitter"
)

// pythonExtraction is every Python Strategy method's output for one parsed fixture, gathered in a
// single treesitter.WithTree call so each table row stays one fixture plus its expectation.
type pythonExtraction struct {
	Symbols        []Symbol
	Header         string
	Package        string
	Partial        bool
	Generated      bool
	GeneratedKnown bool
}

// extractPython parses src as Python and runs every registered pythonStrategy method against the
// resulting tree, returning the combined result. It fails the test immediately if the "python"
// strategy is not registered or the parse itself errors — both would be a setup bug, not a case
// under test.
func extractPython(t *testing.T, src string) pythonExtraction {
	t.Helper()
	strategy, ok := StrategyFor("python")
	if !ok {
		t.Fatal(`StrategyFor("python") reported ok == false; want the registered Python strategy`)
	}
	var got pythonExtraction
	b := []byte(src)
	err := treesitter.WithTree("python", b, func(root *ts.Node, partial bool) error {
		got.Symbols = strategy.Symbols(root, b)
		got.Header = strategy.Header(root, b)
		got.Package = strategy.Package(root, b)
		got.Partial = partial
		got.Generated, got.GeneratedKnown = strategy.Generated(root, b)
		return nil
	})
	if err != nil {
		t.Fatalf("treesitter.WithTree(\"python\", ...) returned error: %v", err)
	}
	return got
}

// TestPythonStrategy_Symbols covers Symbols across every declaration shape the Python strategy
// handles: docstring association, container descent into a class body, decorator unwrapping, and
// every SigEnd shape.
func TestPythonStrategy_Symbols(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []Symbol
	}{
		{
			name: "ModuleFunctionWithDocstringRangeIsTheDefsOwnSpanNotAdjustedUpward",
			src: "def foo():\n" +
				"    \"\"\"Foo does a thing.\"\"\"\n" +
				"    return 1\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "foo", Signature: "def foo():", Docstring: "Foo does a thing.", Start: 1, SigEnd: 1, End: 3},
			},
		},
		{
			name: "ModuleFunctionWithNoDocstring",
			src:  "def bar():\n    return 1\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "bar", Signature: "def bar():", Start: 1, SigEnd: 1, End: 2},
			},
		},
		{
			name: "ClassWithOwnDocstring",
			src: "class C:\n" +
				"    \"\"\"C does a thing.\"\"\"\n",
			want: []Symbol{
				{Kind: KindType, Name: "C", Signature: "class C:", Docstring: "C does a thing.", Start: 1, SigEnd: 1, End: 2},
			},
		},
		{
			name: "MethodInsideClassIsFoundWithOwnerAndBareName",
			src: "class C:\n" +
				"    def m(self):\n" +
				"        return 1\n",
			want: []Symbol{
				{Kind: KindType, Name: "C", Signature: "class C:", Start: 1, SigEnd: 1, End: 3},
				{Kind: KindMethod, Name: "m", Owner: "C", Signature: "def m(self):", Start: 2, SigEnd: 2, End: 3},
			},
		},
		{
			name: "NestedFunctionInsideFunctionBodyIsNotListed",
			src: "def outer():\n" +
				"    def inner():\n" +
				"        return 1\n" +
				"    return inner\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "outer", Signature: "def outer():", Start: 1, SigEnd: 1, End: 4},
			},
		},
		{
			name: "DecoratedFunctionListedWithRangeStartingAtTheDecoratorLine",
			src: "@staticmethod\n" +
				"def deco():\n" +
				"    return 1\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "deco", Signature: "def deco():", Start: 1, SigEnd: 2, End: 3},
			},
		},
		{
			name: "MultiLineDefSignatureReturnedWholeThroughClosingColon",
			src: "def multi(\n" +
				"    a,\n" +
				"    b,\n" +
				"):\n" +
				"    return a + b\n",
			want: []Symbol{
				{
					Kind: KindFunction, Name: "multi",
					Signature: "def multi(\n    a,\n    b,\n):",
					Start:     1, SigEnd: 4, End: 5,
				},
			},
		},
		{
			name: "SingleLineDefWithDocstringOnNextLineSigEndIsTheDefLineBodyExcluded",
			src: "def single():\n" +
				"    \"\"\"Docstring.\"\"\"\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "single", Signature: "def single():", Docstring: "Docstring.", Start: 1, SigEnd: 1, End: 2},
			},
		},
		{
			name: "SingleLineDefWithInlineReturnSigEndIsClampedToTheDefLine",
			src:  "def f(): return 1\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "f", Signature: "def f():", Start: 1, SigEnd: 1, End: 1},
			},
		},
		{
			name: "ClassSigEndIsTheClassColonLine",
			src:  "class Empty:\n    pass\n",
			want: []Symbol{
				{Kind: KindType, Name: "Empty", Signature: "class Empty:", Start: 1, SigEnd: 1, End: 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPython(t, tt.src).Symbols
			assertSymbolsEqual(t, got, tt.want)
		})
	}
}

// TestPythonStrategy_SymbolsAscendingByStart asserts Symbols returns several declarations of mixed
// kinds in ascending Start order, as the plan's ordering decision requires.
func TestPythonStrategy_SymbolsAscendingByStart(t *testing.T) {
	src := "def first():\n" +
		"    return 1\n" +
		"\n" +
		"class Second:\n" +
		"    pass\n" +
		"\n" +
		"def third():\n" +
		"    return 1\n"
	got := extractPython(t, src).Symbols
	if len(got) != 3 {
		t.Fatalf("Symbols() returned %d symbols; want 3", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Start >= got[i].Start {
			t.Errorf("Symbols()[%d].Start = %d >= Symbols()[%d].Start = %d; want ascending order", i-1, got[i-1].Start, i, got[i].Start)
		}
	}
	wantNames := []string{"first", "Second", "third"}
	for i, name := range wantNames {
		if got[i].Name != name {
			t.Errorf("Symbols()[%d].Name = %q; want %q", i, got[i].Name, name)
		}
	}
}

// TestPythonStrategy_Package asserts Package always returns the empty string, so a later
// "improvement" that derives a module name from the filename fails this test.
func TestPythonStrategy_Package(t *testing.T) {
	got := extractPython(t, "def f():\n    return 1\n").Package
	if got != "" {
		t.Errorf("Package() = %q; want empty string", got)
	}
}

// TestPythonStrategy_Header covers Header's module-docstring preference and its leading-comment
// fallback, including the shebang and PEP 263 coding-line directive cases.
func TestPythonStrategy_Header(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		wantHeader string
	}{
		{
			name: "ShebangCodingLineThenModuleDocstringDocstringWins",
			src: "#!/usr/bin/env python\n" +
				"# -*- coding: utf-8 -*-\n" +
				"\"\"\"Module docstring.\"\"\"\n" +
				"\n" +
				"def f():\n    return 1\n",
			wantHeader: "Module docstring.",
		},
		{
			name:       "ModuleDocstringOnlyIsTheHeader",
			src:        "\"\"\"Module docstring.\"\"\"\n\ndef f():\n    return 1\n",
			wantHeader: "Module docstring.",
		},
		{
			name: "NoDocstringProseCommentBlockIsTheHeader",
			src: "# Prose header comment.\n" +
				"\n" +
				"def f():\n    return 1\n",
			wantHeader: "Prose header comment.",
		},
		{
			name:       "NoDocstringOnlyShebangNoHeader",
			src:        "#!/usr/bin/env python\n\ndef f():\n    return 1\n",
			wantHeader: "",
		},
		{
			name:       "NeitherDocstringNorCommentNoHeaderSymbolsStillReturned",
			src:        "def f():\n    return 1\n",
			wantHeader: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPython(t, tt.src)
			if got.Header != tt.wantHeader {
				t.Errorf("Header() = %q; want %q", got.Header, tt.wantHeader)
			}
			if len(got.Symbols) == 0 {
				t.Error("Symbols() returned no symbols; want the module-level def still returned")
			}
		})
	}
}

// TestPythonStrategy_Classification asserts Generated always reports known == false and TestFile
// reports known == true for both pytest-default name shapes.
func TestPythonStrategy_Classification(t *testing.T) {
	got := extractPython(t, "def f():\n    return 1\n")
	if got.GeneratedKnown {
		t.Error("Generated() known = true; want false for python")
	}

	strategy, ok := StrategyFor("python")
	if !ok {
		t.Fatal(`StrategyFor("python") reported ok == false; want the registered Python strategy`)
	}
	tests := []struct {
		name       string
		base       string
		wantIsTest bool
	}{
		{name: "TestPrefix", base: "test_foo.py", wantIsTest: true},
		{name: "TestSuffix", base: "foo_test.py", wantIsTest: true},
		{name: "NonTest", base: "foo.py", wantIsTest: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsTest, gotKnown := strategy.TestFile(tt.base)
			if gotIsTest != tt.wantIsTest {
				t.Errorf("TestFile(%q) isTest = %v; want %v", tt.base, gotIsTest, tt.wantIsTest)
			}
			if !gotKnown {
				t.Errorf("TestFile(%q) known = false; want true for python", tt.base)
			}
		})
	}
}
