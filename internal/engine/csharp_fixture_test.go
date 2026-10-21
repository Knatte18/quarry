// csharp_fixture_test.go covers the committed C# fixtures under testdata/csharp.
// It checks the round-trip tree for the ids and kinds it must list, the collision fixture for its stated outcomes,
// and a real C# checkout named by the environment for the round trip.

package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// csharpCheckout returns the root of the C# checkout named by QUARRY_CSHARP_REPO, never a tracked machine-specific path.
// It skips, with the reason, when the variable is unset or names no directory.
func csharpCheckout(t testing.TB) string {
	t.Helper()

	repo := os.Getenv("QUARRY_CSHARP_REPO")
	if repo == "" {
		t.Skip("QUARRY_CSHARP_REPO is unset; this machine has no C# checkout")
	}
	info, err := os.Stat(repo)
	if err != nil || !info.IsDir() {
		t.Skipf("QUARRY_CSHARP_REPO %q is not an existing directory; this machine has no C# checkout", repo)
	}
	return repo
}

// openCSharpFixture opens the committed fixture directory name, relative to testdata/csharp, as a repository of its own.
func openCSharpFixture(t *testing.T, name string) *Repo {
	t.Helper()
	return openRepo(t, filepath.Join(repoModuleRoot(t), "internal", "engine", "testdata", "csharp", name))
}

func TestRoundTrip_CSharpFixture(t *testing.T) {
	t.Parallel()
	r := openCSharpFixture(t, "tree")
	walked := harvestWalkSymbols(t, r)

	kindsByID := make(map[string][]Kind)
	for _, sym := range walked {
		kindsByID[sym.id] = append(kindsByID[sym.id], sym.kind)
	}

	wantKinds := []struct {
		id   string
		kind Kind
	}{
		{"global#Standalone", KindType},
		{"global#Standalone.Run()", KindMethod},
		{"Loomyard.Root#Entry", KindType},
		{"Loomyard.Engine#IProvider<>", KindType},
		{"Loomyard.Engine#IProvider<>.Value", KindProperty},
		{"Loomyard.Engine#Renderer", KindType},
		{"Loomyard.Engine#Renderer.Renderer()", KindConstructor},
		{"Loomyard.Engine#Renderer.static Renderer()", KindConstructor},
		{"Loomyard.Engine#Renderer.~Renderer()", KindMethod},
		{"Loomyard.Engine#Renderer.Draw(int)", KindMethod},
		{"Loomyard.Engine#Renderer.Draw(double)", KindMethod},
		{"Loomyard.Engine#Renderer.Draw()", KindMethod},
		{"Loomyard.Engine#Renderer.this[int]", KindIndexer},
		{"Loomyard.Engine#Renderer.operator +(Renderer,Renderer)", KindOperator},
		{"Loomyard.Engine#Renderer.operator <(Renderer,Renderer)", KindOperator},
		{"Loomyard.Engine#Renderer.operator >(Renderer,Renderer)", KindOperator},
		{"Loomyard.Engine#Renderer.operator >>(Renderer,int)", KindOperator},
		{"Loomyard.Engine#Renderer.implicit operator System.Text.Rune(Renderer)", KindOperator},
		{"Loomyard.Engine#Renderer.Name", KindProperty},
		{"Loomyard.Engine#Renderer.X", KindField},
		{"Loomyard.Engine#Renderer.Y", KindField},
		{"Loomyard.Engine#Renderer.Limit", KindConst},
		{"Loomyard.Engine#Renderer.Changed", KindEvent},
		{"Loomyard.Engine#Renderer.Resized", KindEvent},
		{"Loomyard.Engine#Renderer.IProvider<A>.Value", KindProperty},
		{"Loomyard.Engine#Renderer.IProvider<B>.Value", KindProperty},
		{"Loomyard.Engine#Renderer.System.IDisposable.Dispose()", KindMethod},
		{"Loomyard.Engine#Renderer.Cache", KindType},
		{"Loomyard.Engine#Renderer.Cache.Get<>(string)", KindMethod},
		{"Loomyard.Engine#Renderer.OnDraw()", KindMethod},
		{"Loomyard.Engine#Renderer.Flush()", KindMethod},
		{"Loomyard.Shapes#Box<>", KindType},
		{"Loomyard.Shapes#Box<>.Box(T)", KindConstructor},
		{"Loomyard.Shapes#Vector", KindType},
		{"Loomyard.Shapes#Vector.X", KindField},
		{"Loomyard.Shapes#IShape", KindType},
		{"Loomyard.Shapes#IShape.Area()", KindMethod},
		{"Loomyard.Shapes#Color", KindType},
		{"Loomyard.Shapes#Color.Red", KindConst},
		{"Loomyard.Shapes#Notify", KindType},
		{"Loomyard.Shapes#Point", KindType},
		{"Loomyard.Shapes#Point.X", KindProperty},
		{"Loomyard.Shapes#Pair", KindType},
		{"Loomyard.Shapes#Pair.Left", KindProperty},
		{"Loomyard.Shapes#Cell", KindType},
		{"Loomyard.Shapes#Cell.Row", KindProperty},
		{"Loomyard.Shapes#Holder", KindType},
		{"Loomyard.Shapes#Holder.Holder(string)", KindConstructor},
		{"Loomyard.Spread#Left", KindType},
		{"Loomyard.Spread#Right", KindType},
		{"Loomyard.Spaced#Spaced", KindType},
		{"Loomyard.Mixed#Widget", KindType},
		{"Loomyard.Mixed#Widget.Spin()", KindMethod},
	}
	for _, want := range wantKinds {
		kinds, ok := kindsByID[want.id]
		if !ok {
			t.Errorf("walk lists no %q", want.id)
			continue
		}
		if kinds[0] != want.kind {
			t.Errorf("walk lists %q as kind %q; want %q", want.id, kinds[0], want.kind)
		}
	}

	dispose := "Loomyard.Engine#Renderer.System.IDisposable.Dispose()"
	res := resolveOne(t, r, dispose)
	if res.Status != StatusFound || len(res.Symbols) != 1 {
		t.Fatalf("Resolve(%q) = %+v; want found once", dispose, res)
	}
	var walkedSpan spanTuple
	for _, sym := range walked {
		if sym.id == dispose {
			walkedSpan = sym.tuple
		}
	}
	got := res.Symbols[0]
	if gotSpan := (spanTuple{File: got.File, Start: got.Start, SigEnd: got.SigEnd, End: got.End}); gotSpan != walkedSpan {
		t.Errorf("Resolve(%q) span = %+v; want the walk's %+v", dispose, gotSpan, walkedSpan)
	}

	point := "Loomyard.Shapes#Point.X"
	if res := resolveOne(t, r, point); res.Status != StatusFound || len(res.Symbols) != 1 {
		t.Errorf("Resolve(%q) = %+v; want found once", point, res)
	}
}

func TestCSharpFixture_Collisions(t *testing.T) {
	t.Parallel()
	r := openCSharpFixture(t, "collisions")
	walked := harvestWalkSymbols(t, r)

	listed := make(map[string]int)
	kinds := make(map[string]Kind)
	for _, sym := range walked {
		listed[sym.id]++
		kinds[sym.id] = sym.kind
	}

	width := "Loomyard.Collisions#Shape.Width"
	if listed[width] != 2 {
		t.Errorf("walk lists %q %d times; want 2 — the positional parameter and its redeclaration", width, listed[width])
	}
	if res := resolveOne(t, r, width); res.Status != StatusAmbiguous || len(res.Candidates) != 2 {
		t.Errorf("Resolve(%q) = %+v; want ambiguous with two candidates", width, res)
	}

	inherited := "Loomyard.Collisions#Child.Id"
	if listed[inherited] != 1 || kinds[inherited] != KindProperty {
		t.Errorf("walk lists %q %d times as kind %q; want once as a property", inherited, listed[inherited], kinds[inherited])
	}
	if res := resolveOne(t, r, inherited); res.Status != StatusFound {
		t.Errorf("Resolve(%q) status = %q; want found", inherited, res.Status)
	}

	handler := "Loomyard.Collisions#Renderer.IHandler"
	answer, err := r.Expand(handler)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", handler, err)
	}
	handle := "Loomyard.Collisions#Renderer.IHandler.Handle()"
	if answer.Status != StatusFound || len(answer.Members) != 2 || answer.Members[0].ID != handle || answer.Members[1].ID != handle {
		t.Errorf("Expand(%q) = %+v; want found with the interface's method and the implementation as members", handler, answer)
	}
	if res := resolveOne(t, r, handle); res.Status != StatusAmbiguous {
		t.Errorf("Resolve(%q) status = %q; want ambiguous", handle, res.Status)
	}
}

func TestRoundTrip_CSharpCheckout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the C# checkout round trip in -short mode")
	}
	r := openRepo(t, csharpCheckout(t))

	walked := assertSymbolRoundTrip(t, r)

	for _, sym := range walked {
		if sym.lang != glyph.CSharp {
			continue
		}
		parsed, err := glyph.Parse(glyph.CSharp, sym.id)
		if err != nil {
			t.Errorf("glyph.Parse(CSharp, %q) failed: %v", sym.id, err)
			continue
		}
		if got := parsed.String(); got != sym.id {
			t.Errorf("glyph.Parse(CSharp, %q).String() = %q; want %q", sym.id, got, sym.id)
		}
	}
}
