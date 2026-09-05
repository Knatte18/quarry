// delta_tokens_test.go covers tokenStreamsForSymbols' byte-range extraction and leaf assignment,
// identicalModuloName's substitution rule, and bodyTokenSimilarity's Jaccard coefficient. Every
// fixture is an inline Go source string parsed through treesitter.WithTree, so this file needs no
// fixture tree and no repository.

package engine

import (
	"testing"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// symbolsAndStreams parses src as Go and returns every extracted symbol alongside its token
// streams, in the same order tokenStreamsForSymbols returns them in. It fails the test immediately
// on a setup error — a strategy lookup failure or a parse error — neither of which is under test
// here.
func symbolsAndStreams(t *testing.T, src string) ([]Symbol, []symbolStreams) {
	t.Helper()
	strategy, ok := StrategyFor("go")
	if !ok {
		t.Fatal(`StrategyFor("go") reported ok == false; want the registered Go strategy`)
	}
	var symbols []Symbol
	var streams []symbolStreams
	b := []byte(src)
	err := treesitter.WithTree("go", b, func(root *ts.Node, partial bool) error {
		symbols = strategy.Symbols(testUnit, root, b)
		streams = tokenStreamsForSymbols(root, b, symbols)
		return nil
	})
	if err != nil {
		t.Fatalf("treesitter.WithTree(\"go\", ...) returned error: %v", err)
	}
	return symbols, streams
}

// findSymbolStreams returns the streams for the first symbol in symbols whose Glyph.Name equals
// name and whose Kind equals kind, failing the test if none matches.
func findSymbolStreams(t *testing.T, symbols []Symbol, streams []symbolStreams, name string, kind Kind) symbolStreams {
	t.Helper()
	for i, sym := range symbols {
		if sym.Glyph.Name == name && sym.Kind == kind {
			return streams[i]
		}
	}
	t.Fatalf("no symbol named %q of kind %q found", name, kind)
	return symbolStreams{}
}

// streamsEqual reports whether two token streams are identical, kind and text alike, position by
// position.
func streamsEqual(a, b tokenStream) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTokenStreams_AnonymousLeavesIncluded(t *testing.T) {
	tests := []struct {
		name string
		srcA string
		srcB string
	}{
		{
			name: "IncrementVsDecrement",
			srcA: "package p\n\nfunc F() {\n\tx++\n}\n",
			srcB: "package p\n\nfunc F() {\n\tx--\n}\n",
		},
		{
			name: "PlusVsMinus",
			srcA: "package p\n\nfunc F() {\n\ty := x + 1\n}\n",
			srcB: "package p\n\nfunc F() {\n\ty := x - 1\n}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbolsA, streamsA := symbolsAndStreams(t, tt.srcA)
			symbolsB, streamsB := symbolsAndStreams(t, tt.srcB)
			bodyA := findSymbolStreams(t, symbolsA, streamsA, "F", KindFunction).body
			bodyB := findSymbolStreams(t, symbolsB, streamsB, "F", KindFunction).body
			if streamsEqual(bodyA, bodyB) {
				t.Errorf("body streams equal for %q vs %q; want different — a named-leaves-only rule "+
					"would make this pair falsely identical", tt.srcA, tt.srcB)
			}
		})
	}
}

func TestTokenStreams_ByteRangeSplit(t *testing.T) {
	t.Run("InterfaceBodyCoversMethodElements", func(t *testing.T) {
		oneMethod := "package p\n\ntype I interface {\n\tA()\n}\n"
		twoMethods := "package p\n\ntype I interface {\n\tA()\n\tB()\n}\n"
		symbolsOne, streamsOne := symbolsAndStreams(t, oneMethod)
		symbolsTwo, streamsTwo := symbolsAndStreams(t, twoMethods)
		bodyOne := findSymbolStreams(t, symbolsOne, streamsOne, "I", KindType).body
		bodyTwo := findSymbolStreams(t, symbolsTwo, streamsTwo, "I", KindType).body
		if streamsEqual(bodyOne, bodyTwo) {
			t.Error("interface I's body stream unchanged after adding method B(); want it to change — " +
				"the body stream must cover the method elements, not merely the opening brace")
		}
	})

	t.Run("StructFieldsLandInBodyNotSignature", func(t *testing.T) {
		symbols, streams := symbolsAndStreams(t, "package p\n\ntype S struct {\n\tX int\n}\n")
		got := findSymbolStreams(t, symbols, streams, "S", KindType)
		for _, tok := range got.signature {
			if tok.text == "X" {
				t.Error("struct field name \"X\" found in the signature stream; want it only in the body stream")
			}
		}
		foundFieldInBody := false
		for _, tok := range got.body {
			if tok.text == "X" {
				foundFieldInBody = true
			}
		}
		if !foundFieldInBody {
			t.Error("struct field name \"X\" not found in the body stream; want it there")
		}
	})
}

func TestTokenStreams_ManyToManyLeafAssignment(t *testing.T) {
	t.Run("InterfaceMethodElementInBothStreams", func(t *testing.T) {
		symbols, streams := symbolsAndStreams(t, "package p\n\ntype I interface {\n\tA()\n}\n")
		ifaceBody := findSymbolStreams(t, symbols, streams, "I", KindType).body
		methodSig := findSymbolStreams(t, symbols, streams, "A", KindMethod).signature
		if len(methodSig) == 0 {
			t.Fatal("method A's signature stream is empty; want the method_elem's own leaves")
		}
		for _, tok := range methodSig {
			found := false
			for _, ifaceTok := range ifaceBody {
				if tok == ifaceTok {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("token %+v from method A's signature stream not found in interface I's body "+
					"stream; want every method_elem leaf assigned to both", tok)
			}
		}
	})

	t.Run("SharedConstSpecSharesOneStream", func(t *testing.T) {
		symbols, streams := symbolsAndStreams(t, "package p\n\nconst (\n\tA, B = 1, 2\n)\n")
		sigA := findSymbolStreams(t, symbols, streams, "A", KindConst).signature
		sigB := findSymbolStreams(t, symbols, streams, "B", KindConst).signature
		if len(sigA) == 0 {
			t.Fatal("const A's signature stream is empty; want the shared spec's leaves")
		}
		if !streamsEqual(sigA, sigB) {
			t.Error("const A and const B's signature streams differ; want them identical — both names " +
				"share one spec's span verbatim")
		}
	})
}

func TestIdenticalModuloName(t *testing.T) {
	t.Run("IdenticalExceptRenamedIdentifier_RecursiveSelfCall", func(t *testing.T) {
		before := "package p\n\nfunc Old() {\n\tx := 1\n\tOld()\n\t_ = x\n}\n"
		after := "package p\n\nfunc New() {\n\tx := 1\n\tNew()\n\t_ = x\n}\n"
		symbolsBefore, streamsBefore := symbolsAndStreams(t, before)
		symbolsAfter, streamsAfter := symbolsAndStreams(t, after)
		bodyBefore := findSymbolStreams(t, symbolsBefore, streamsBefore, "Old", KindFunction).body
		bodyAfter := findSymbolStreams(t, symbolsAfter, streamsAfter, "New", KindFunction).body
		if !identicalModuloName(bodyBefore, bodyAfter, "Old", "New") {
			t.Error("identicalModuloName(...) = false; want true — the two bodies differ only by the " +
				"renamed identifier, including its recursive self-call")
		}
	})

	t.Run("ExtraStatement_NotIdentical", func(t *testing.T) {
		before := "package p\n\nfunc Old() {\n\tx := 1\n\t_ = x\n}\n"
		after := "package p\n\nfunc New() {\n\tx := 1\n\ty := 2\n\t_ = x\n\t_ = y\n}\n"
		symbolsBefore, streamsBefore := symbolsAndStreams(t, before)
		symbolsAfter, streamsAfter := symbolsAndStreams(t, after)
		bodyBefore := findSymbolStreams(t, symbolsBefore, streamsBefore, "Old", KindFunction).body
		bodyAfter := findSymbolStreams(t, symbolsAfter, streamsAfter, "New", KindFunction).body
		if identicalModuloName(bodyBefore, bodyAfter, "Old", "New") {
			t.Error("identicalModuloName(...) = true; want false — the after body carries one extra statement")
		}
	})

	t.Run("AnonymousOperatorDiffers_NotIdentical", func(t *testing.T) {
		before := "package p\n\nfunc Old() {\n\tx := 1 + 1\n\t_ = x\n}\n"
		after := "package p\n\nfunc New() {\n\tx := 1 - 1\n\t_ = x\n}\n"
		symbolsBefore, streamsBefore := symbolsAndStreams(t, before)
		symbolsAfter, streamsAfter := symbolsAndStreams(t, after)
		bodyBefore := findSymbolStreams(t, symbolsBefore, streamsBefore, "Old", KindFunction).body
		bodyAfter := findSymbolStreams(t, symbolsAfter, streamsAfter, "New", KindFunction).body
		if identicalModuloName(bodyBefore, bodyAfter, "Old", "New") {
			t.Error("identicalModuloName(...) = true; want false — the bodies differ only in an " +
				"anonymous operator token (+ vs -), which the rename substitution never admits")
		}
	})

	t.Run("ReceiverSharesPrefixWithOldName_SignatureIdenticalModuloRename", func(t *testing.T) {
		// A textual substitution of "Run" -> "Execute" over the verbatim signature string would also
		// rewrite the "Run" prefix inside the receiver type "Runner", producing a false mismatch (or a
		// false match, under a naive replace). The token-based rule keys on whole identifier texts, so
		// "Runner" — never equal to "Run" — is left alone.
		before := "package p\n\ntype Runner struct{}\n\nfunc (r *Runner) Run() error {\n\treturn nil\n}\n"
		after := "package p\n\ntype Runner struct{}\n\nfunc (r *Runner) Execute() error {\n\treturn nil\n}\n"
		symbolsBefore, streamsBefore := symbolsAndStreams(t, before)
		symbolsAfter, streamsAfter := symbolsAndStreams(t, after)
		sigBefore := findSymbolStreams(t, symbolsBefore, streamsBefore, "Run", KindMethod).signature
		sigAfter := findSymbolStreams(t, symbolsAfter, streamsAfter, "Execute", KindMethod).signature
		if !identicalModuloName(sigBefore, sigAfter, "Run", "Execute") {
			t.Error("identicalModuloName(...) = false; want true — the receiver's \"Runner\" token must " +
				"not be affected by the \"Run\"->\"Execute\" identifier substitution")
		}
	})
}

func TestBodyTokenSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		before     tokenStream
		after      tokenStream
		beforeName string
		afterName  string
		want       float64
	}{
		{
			name:       "TwoEmptyStreams",
			before:     nil,
			after:      nil,
			beforeName: "Old",
			afterName:  "New",
			want:       1.0,
		},
		{
			name: "IdenticalStreams",
			before: tokenStream{
				{kind: "identifier", text: "x"},
				{kind: "+", text: "+"},
				{kind: "identifier", text: "y"},
			},
			after: tokenStream{
				{kind: "identifier", text: "x"},
				{kind: "+", text: "+"},
				{kind: "identifier", text: "y"},
			},
			beforeName: "Old",
			afterName:  "New",
			want:       1.0,
		},
		{
			name: "DisjointStreams",
			before: tokenStream{
				{kind: "op", text: "+"},
				{kind: "op", text: "-"},
			},
			after: tokenStream{
				{kind: "op", text: "*"},
				{kind: "op", text: "/"},
			},
			beforeName: "Foo",
			afterName:  "Bar",
			want:       0.0,
		},
		{
			// Hand-computed: intersection counts {op:+ -> 1, op:- -> 1}, union counts
			// {op:+ -> 1, op:- -> 1, op:* -> 1, op:/ -> 1}; 2/4 = 0.5.
			name: "PartialOverlap",
			before: tokenStream{
				{kind: "op", text: "+"},
				{kind: "op", text: "-"},
				{kind: "op", text: "*"},
			},
			after: tokenStream{
				{kind: "op", text: "+"},
				{kind: "op", text: "-"},
				{kind: "op", text: "/"},
			},
			beforeName: "Foo",
			afterName:  "Bar",
			want:       0.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bodyTokenSimilarity(tt.before, tt.after, tt.beforeName, tt.afterName)
			if got != tt.want {
				t.Errorf("bodyTokenSimilarity(...) = %v; want %v", got, tt.want)
			}
			if got < 0.0 || got > 1.0 {
				t.Errorf("bodyTokenSimilarity(...) = %v; want a value in the closed interval [0, 1]", got)
			}
		})
	}
}
