// glyph_test.go covers glyph assignment, the walk's widened Go extraction, and the two
// unspellable-unit entry points, over the committed testdata/glyphs/ and testdata/units/ fixtures.

package engine

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// glyphsUnit is the glyph unit of every fixture under testdata/glyphs/, queried from the quarry
// module root.
const glyphsUnit = "internal/engine/testdata/glyphs"

// gid builds the expected package-level glyph id for name under glyphsUnit.
func gid(name string) string {
	return glyphsUnit + "#" + name
}

// gmid builds the expected method glyph id for name owned by owner under glyphsUnit.
func gmid(owner, name string) string {
	g := glyph.Glyph{Lang: glyph.Go, Unit: glyphsUnit, Owner: []string{owner}, Name: name}
	return g.String()
}

// openGlyphFixtures opens the quarry module root and answers a symbol-requesting TOC query over
// testdata/glyphs/, shared by every test in this file that reads its symbols.
func openGlyphFixtures(t *testing.T) DirAnswer {
	t.Helper()
	r := openModuleRepo(t)
	got, err := r.TOC(glyphsUnit, TOCOptions{Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf("TOC(%q, ...) returned error: %v", glyphsUnit, err)
	}
	return got
}

// symbolsIn returns the Symbols of the file named base inside dir, failing the test if the file is
// absent or was not extracted.
func symbolsIn(t *testing.T, dir DirAnswer, base string) []Symbol {
	t.Helper()
	entry := entryByName(t, dir.Files, base)
	if entry.Symbols == nil {
		t.Fatalf("file %q has no Symbols; want a non-nil pointer", base)
	}
	return *entry.Symbols
}

// symbolByID returns the Symbol in symbols whose ID is id, failing the test if none matches.
func symbolByID(t *testing.T, symbols []Symbol, id string) Symbol {
	t.Helper()
	for _, sym := range symbols {
		if sym.ID == id {
			return sym
		}
	}
	t.Fatalf("no symbol with id %q in %+v", id, symbols)
	return Symbol{}
}

// TestGlyph_PackageLevelFunctionID asserts a package-level function's id is its unit, "#", and its
// bare name — no owner segment.
func TestGlyph_PackageLevelFunctionID(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "iface.go")
	sym := symbolByID(t, symbols, gid("AnonParam"))
	if sym.Kind != KindFunction {
		t.Errorf("Kind = %q; want %q", sym.Kind, KindFunction)
	}
}

// TestGlyph_MethodOwnerNameID asserts an ordinary method's id is "owner.name" under its unit.
func TestGlyph_MethodOwnerNameID(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "blank.go")
	sym := symbolByID(t, symbols, gmid("BlankImpl", "M"))
	if sym.Kind != KindMethod {
		t.Errorf("Kind = %q; want %q", sym.Kind, KindMethod)
	}
}

// TestGlyph_InterfaceMethodOwnerIsInterfaceName asserts a named interface's own methods are owned
// by that interface's name, and that an embedded interface's own method is owned by the embedded
// interface itself, never by the embedder.
func TestGlyph_InterfaceMethodOwnerIsInterfaceName(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "iface.go")

	for _, name := range []string{"M1", "M2"} {
		sym := symbolByID(t, symbols, gmid("Iface", name))
		if sym.Kind != KindMethod {
			t.Errorf("%s Kind = %q; want %q", name, sym.Kind, KindMethod)
		}
	}
	sym := symbolByID(t, symbols, gmid("Embedded", "E"))
	if sym.Kind != KindMethod {
		t.Errorf("E Kind = %q; want %q", sym.Kind, KindMethod)
	}
}

// TestGlyph_GenericReceiverOwnerIsBareTypeNameRoundTrips asserts a method on a generic type is
// owned by the type's bare name, never the parameterised spelling, by round-tripping the id
// through glyph.Parse and String — the case a receiver-text-only check would otherwise miss.
func TestGlyph_GenericReceiverOwnerIsBareTypeNameRoundTrips(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "generic.go")

	for _, name := range []string{"Get", "Set"} {
		wantID := gmid("Box", name)
		sym := symbolByID(t, symbols, wantID)
		g, err := glyph.Parse(glyph.Go, sym.ID)
		if err != nil {
			t.Fatalf("glyph.Parse(%q) failed: %v", sym.ID, err)
		}
		if got := g.String(); got != wantID {
			t.Errorf("round trip of %q = %q; want %q", sym.ID, got, wantID)
		}
		if len(g.Owner) != 1 || g.Owner[0] != "Box" {
			t.Errorf("Owner = %v; want [\"Box\"], never a parameterised spelling", g.Owner)
		}
	}
}

// TestGlyph_BlankIdentifierNeverListed asserts every "_"-named declaration in blank.go — a var
// initializer, an interface-assertion var, and a type declaration — is absent from Symbols,
// leaving only the two declarations blank.go declares under real names.
func TestGlyph_BlankIdentifierNeverListed(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "blank.go")

	wantIDs := map[string]bool{
		gid("BlankIface"):       true,
		gmid("BlankIface", "M"): true,
		gid("BlankImpl"):        true,
		gmid("BlankImpl", "M"):  true,
	}
	if len(symbols) != len(wantIDs) {
		t.Fatalf("len(symbols) = %d %+v; want %d — the three blank declarations contribute nothing", len(symbols), symbols, len(wantIDs))
	}
	for _, sym := range symbols {
		if !wantIDs[sym.ID] {
			t.Errorf("unexpected symbol id %q; a blank-named declaration must never be listed", sym.ID)
		}
	}
}

// TestGlyph_InitSharesOneIDThreeSpansInOrder asserts three func init() declarations in one file
// are listed as three separate symbols sharing the one id "<unit>#init", in file order.
func TestGlyph_InitSharesOneIDThreeSpansInOrder(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "inits.go")

	if len(symbols) != 3 {
		t.Fatalf("len(symbols) = %d; want 3", len(symbols))
	}
	wantID := gid("init")
	wantStarts := []int{6, 8, 10}
	for i, sym := range symbols {
		if sym.ID != wantID {
			t.Errorf("symbols[%d].ID = %q; want %q", i, sym.ID, wantID)
		}
		if sym.Start != wantStarts[i] {
			t.Errorf("symbols[%d].Start = %d; want %d", i, sym.Start, wantStarts[i])
		}
	}
}

// TestGlyph_ConstAndVarShapes covers the ungrouped, single-spec-grouped, several-names-per-spec,
// and bare-iota const/var shapes: each asserted on id, span, signature, and the absence of sigend.
func TestGlyph_ConstAndVarShapes(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "decls.go")

	tests := []struct {
		name       string
		id         string
		kind       Kind
		start, end int
		signature  string
	}{
		{"UngroupedConst", gid("UngroupedConst"), KindConst, 6, 7, "const UngroupedConst = 1"},
		{"UngroupedVar", gid("UngroupedVar"), KindVar, 9, 10, "var UngroupedVar int"},
		{"GroupedConst", gid("GroupedConst"), KindConst, 13, 14, "const GroupedConst = 2"},
		{"GroupedVar", gid("GroupedVar"), KindVar, 18, 19, "var GroupedVar string"},
		{"GroupedMultiA", gid("GroupedMultiA"), KindConst, 23, 24, "const GroupedMultiA, GroupedMultiB = 3, 4"},
		{"GroupedMultiB", gid("GroupedMultiB"), KindConst, 23, 24, "const GroupedMultiA, GroupedMultiB = 3, 4"},
		{"MondayBareValueSpec", gid("Monday"), KindConst, 31, 31, "const Monday Weekday = iota"},
		{"TuesdayBareIotaSpec", gid("Tuesday"), KindConst, 32, 32, "const Tuesday"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sym := symbolByID(t, symbols, tt.id)
			if sym.Kind != tt.kind {
				t.Errorf("Kind = %q; want %q", sym.Kind, tt.kind)
			}
			if sym.Start != tt.start || sym.End != tt.end {
				t.Errorf("Start/End = %d/%d; want %d/%d", sym.Start, sym.End, tt.start, tt.end)
			}
			if sym.Signature != tt.signature {
				t.Errorf("Signature = %q; want %q", sym.Signature, tt.signature)
			}
			if sym.SigEnd != 0 {
				t.Errorf("SigEnd = %d; want 0 (absent — a const or var declaration has no body)", sym.SigEnd)
			}
		})
	}
}

// TestGlyph_InterfaceWalkScope asserts the named interface's methods are listed with the right
// owner, and that none of the anonymous interface positions — a struct field, a function
// parameter, and a generic constraint — nor the embedded interface's own name contribute a
// method symbol beyond its own declaration's methods.
func TestGlyph_InterfaceWalkScope(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "iface.go")

	wantMethodIDs := map[string]bool{
		gmid("Iface", "M1"):   true,
		gmid("Iface", "M2"):   true,
		gmid("Embedded", "E"): true,
	}
	gotMethodIDs := make(map[string]bool)
	for _, sym := range symbols {
		if sym.Kind == KindMethod {
			gotMethodIDs[sym.ID] = true
		}
	}
	if !reflect.DeepEqual(gotMethodIDs, wantMethodIDs) {
		t.Errorf("method ids = %v; want exactly %v", gotMethodIDs, wantMethodIDs)
	}
}

// TestGlyph_InterfaceHeadSpanCoversMembers asserts the interface type symbol's HeadStart/HeadEnd
// cover the whole declaration, and every member symbol's span lies inside that range.
func TestGlyph_InterfaceHeadSpanCoversMembers(t *testing.T) {
	dir := openGlyphFixtures(t)
	symbols := symbolsIn(t, dir, "iface.go")

	iface := symbolByID(t, symbols, gid("Iface"))
	if iface.HeadStart != iface.Start || iface.HeadEnd != iface.End {
		t.Errorf("HeadStart/HeadEnd = %d/%d; want equal to Start/End %d/%d", iface.HeadStart, iface.HeadEnd, iface.Start, iface.End)
	}
	for _, name := range []string{"M1", "M2"} {
		m := symbolByID(t, symbols, gmid("Iface", name))
		if m.Start < iface.HeadStart || m.End > iface.HeadEnd {
			t.Errorf("method %s span [%d,%d] is not inside the head range [%d,%d]", name, m.Start, m.End, iface.HeadStart, iface.HeadEnd)
		}
	}
}

// TestGlyph_UnspellableUnit_BadRuneDirectory covers the first unspellable-unit entry point: a
// directory whose name contains a space, queried from the quarry root as usual, where that file's
// own unit is otherwise a perfectly ordinary repository-relative path except for the bad rune. The
// entry is listed with its header and carries no symbols; nothing else about it is affected.
func TestGlyph_UnspellableUnit_BadRuneDirectory(t *testing.T) {
	r := openModuleRepo(t)
	target := filepath.ToSlash(filepath.Join("internal", "engine", "testdata", "units", "test data", "pkg"))

	got, err := r.TOC(target, TOCOptions{Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf("TOC(%q, ...) returned error: %v", target, err)
	}
	entry := entryByName(t, got.Files, "spaced.go")
	if entry.Header == "" {
		t.Error("Header is empty; want the file's header, unaffected by the unspellable unit")
	}
	if entry.Symbols != nil {
		t.Errorf("Symbols = %v; want nil — this directory's unit contains a disallowed space", entry.Symbols)
	}
	if entry.Error != "" || entry.Lossy {
		t.Errorf("Error=%q Lossy=%v; want both unset", entry.Error, entry.Lossy)
	}
}

// TestGlyph_UnspellableUnit_RepositoryRoot covers the second unspellable-unit entry point: the
// repository root's own unit, "". From the quarry root this fixture's unit is the ordinary,
// spellable "internal/engine/testdata/units" — this case is only reachable by opening
// testdata/units/ itself as its own repository root and querying ".", where the unit genuinely is
// empty.
func TestGlyph_UnspellableUnit_RepositoryRoot(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine this test file's own source location")
	}
	unitsRoot := filepath.Join(filepath.Dir(thisFile), "testdata", "units")

	r, err := Open(unitsRoot)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", unitsRoot, err)
	}
	got, err := r.TOC(".", TOCOptions{Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf("TOC(\".\", ...) returned error: %v", err)
	}
	entry := entryByName(t, got.Files, "root.go")
	if entry.Header == "" {
		t.Error("Header is empty; want the file's header, unaffected by the unspellable unit")
	}
	if entry.Symbols != nil {
		t.Errorf("Symbols = %v; want nil — the repository root's own unit is \"\", never spellable", entry.Symbols)
	}
	if entry.Error != "" || entry.Lossy {
		t.Errorf("Error=%q Lossy=%v; want both unset", entry.Error, entry.Lossy)
	}
}
