// offsets_test.go covers the byte-offset seam batch 3 is built on: Symbol.DeclStart, Symbol.BodyStart
// and Symbol.DeclEnd, filled by every builder in the Go strategy. Every fixture here is an inline
// string parsed through treesitter.WithTree, exactly like golang_test.go's own fixtures, so this
// file adds no new dependency on testdata.

package engine

import (
	"strings"
	"testing"
)

// offsetCase is one table row shared by TestSymbolOffsets_PerShape and
// TestSignatureSpanInvariant_PerShape: a builder shape, the source it is extracted from, which
// symbol in the extraction result the shape targets, the declaration text that symbol's DeclStart
// and DeclEnd should bracket, whether its BodyStart equals its DeclEnd, and the synthesized keyword
// prefix — "" for an ungrouped shape — that must be stripped from Signature before comparing it
// against the trimmed [DeclStart, BodyStart) span.
type offsetCase struct {
	name            string
	src             string
	kind            Kind
	owner           string // "" for a package-level symbol
	symbolName      string
	wantDeclText    string
	bodyEqualsEnd   bool
	syntheticPrefix string
}

// offsetCases is the eight shapes TestSymbolOffsets_PerShape and TestSignatureSpanInvariant_PerShape
// both drive, covering all six offset-filling builders: goDeclSymbol (function, method),
// goUngroupedTypeSymbol (struct-bodied type, type alias), goGroupedTypeSymbol (grouped interface
// type), goInterfaceMethodSymbols (interface method element), goUngroupedConstOrVarSymbols
// (ungrouped const), and goGroupedConstOrVarSymbols (grouped var).
func offsetCases() []offsetCase {
	groupedInterfaceSrc := "package p\n" +
		"\n" +
		"type (\n" +
		"\tReader interface {\n" +
		"\t\tRead() (int, error)\n" +
		"\t}\n" +
		")\n"
	return []offsetCase{
		{
			name:          "UngroupedFunction",
			src:           "package p\n\nfunc Foo() {\n\treturn\n}\n",
			kind:          KindFunction,
			symbolName:    "Foo",
			wantDeclText:  "func Foo() {\n\treturn\n}",
			bodyEqualsEnd: false,
		},
		{
			name:          "Method",
			src:           "package p\n\ntype T struct{}\n\nfunc (t T) M() {\n\treturn\n}\n",
			kind:          KindMethod,
			owner:         "T",
			symbolName:    "M",
			wantDeclText:  "func (t T) M() {\n\treturn\n}",
			bodyEqualsEnd: false,
		},
		{
			name:          "UngroupedStructType",
			src:           "package p\n\ntype S struct {\n\tX int\n}\n",
			kind:          KindType,
			symbolName:    "S",
			wantDeclText:  "type S struct {\n\tX int\n}",
			bodyEqualsEnd: false,
		},
		{
			name:            "GroupedInterfaceType",
			src:             groupedInterfaceSrc,
			kind:            KindType,
			symbolName:      "Reader",
			wantDeclText:    "Reader interface {\n\t\tRead() (int, error)\n\t}",
			bodyEqualsEnd:   false,
			syntheticPrefix: "type ",
		},
		{
			name:          "InterfaceMethodElement",
			src:           groupedInterfaceSrc,
			kind:          KindMethod,
			owner:         "Reader",
			symbolName:    "Read",
			wantDeclText:  "Read() (int, error)",
			bodyEqualsEnd: true,
		},
		{
			name:          "UngroupedConst",
			src:           "package p\n\nconst X = 1\n",
			kind:          KindConst,
			symbolName:    "X",
			wantDeclText:  "const X = 1",
			bodyEqualsEnd: true,
		},
		{
			name:            "GroupedVar",
			src:             "package p\n\nvar (\n\tX = 1\n)\n",
			kind:            KindVar,
			symbolName:      "X",
			wantDeclText:    "X = 1",
			bodyEqualsEnd:   true,
			syntheticPrefix: "var ",
		},
		{
			name:          "TypeAlias",
			src:           "package p\n\ntype Alias = int\n",
			kind:          KindType,
			symbolName:    "Alias",
			wantDeclText:  "type Alias = int",
			bodyEqualsEnd: true,
		},
	}
}

// findOffsetSymbol returns the symbol in symbols matching tt's kind, owner and name, failing the
// test immediately if none does — a setup bug in the table, not a case under test.
func findOffsetSymbol(t *testing.T, symbols []Symbol, tt offsetCase) Symbol {
	t.Helper()
	for _, sym := range symbols {
		if sym.Kind != tt.kind || sym.Glyph.Name != tt.symbolName {
			continue
		}
		if tt.owner == "" {
			if len(sym.Glyph.Owner) != 0 {
				continue
			}
		} else {
			if len(sym.Glyph.Owner) != 1 || sym.Glyph.Owner[0] != tt.owner {
				continue
			}
		}
		return sym
	}
	t.Fatalf("no symbol found for kind %q owner %q name %q in %+v", tt.kind, tt.owner, tt.symbolName, symbols)
	return Symbol{}
}

// TestSymbolOffsets_PerShape asserts, for every builder shape, that DeclStart and DeclEnd bracket
// the declaration text the shape's builder cut from, and that BodyStart equals DeclEnd exactly for
// a bodyless shape and is strictly between DeclStart and DeclEnd otherwise. It asserts on a
// non-zero DeclEnd explicitly, rather than only on relations between the three values, so a builder
// left unfilled fails here rather than passing on an all-zero triple.
func TestSymbolOffsets_PerShape(t *testing.T) {
	for _, tt := range offsetCases() {
		t.Run(tt.name, func(t *testing.T) {
			src := []byte(tt.src)
			got := extractGo(t, tt.src).Symbols
			sym := findOffsetSymbol(t, got, tt)

			if sym.DeclEnd == 0 {
				t.Fatalf("DeclEnd = 0; want non-zero for a filled builder")
			}
			declText := string(src[sym.DeclStart:sym.DeclEnd])
			if declText != tt.wantDeclText {
				t.Errorf("src[DeclStart:DeclEnd] = %q; want %q", declText, tt.wantDeclText)
			}

			if tt.bodyEqualsEnd {
				if sym.BodyStart != sym.DeclEnd {
					t.Errorf("BodyStart = %d; want %d (== DeclEnd, no body-bearing child)", sym.BodyStart, sym.DeclEnd)
				}
			} else {
				if !(sym.DeclStart < sym.BodyStart && sym.BodyStart < sym.DeclEnd) {
					t.Errorf("BodyStart = %d; want strictly between DeclStart = %d and DeclEnd = %d", sym.BodyStart, sym.DeclStart, sym.DeclEnd)
				}
			}
		})
	}
}

// TestSignatureSpanInvariant_PerShape asserts the invariant per shape, deliberately not as a flat
// byte identity, which would be false for a grouped shape: the source bytes in
// [DeclStart, BodyStart), trimmed, equal Symbol.Signature for an ungrouped shape, and equal
// Symbol.Signature with its synthesized "type ", "const " or "var " keyword prefix removed for a
// grouped shape.
func TestSignatureSpanInvariant_PerShape(t *testing.T) {
	for _, tt := range offsetCases() {
		t.Run(tt.name, func(t *testing.T) {
			src := []byte(tt.src)
			got := extractGo(t, tt.src).Symbols
			sym := findOffsetSymbol(t, got, tt)

			span := strings.TrimSpace(string(src[sym.DeclStart:sym.BodyStart]))
			want := strings.TrimPrefix(sym.Signature, tt.syntheticPrefix)
			if span != want {
				t.Errorf("trimmed src[DeclStart:BodyStart] = %q; want %q (Signature %q, prefix %q)", span, want, sym.Signature, tt.syntheticPrefix)
			}
		})
	}
}
