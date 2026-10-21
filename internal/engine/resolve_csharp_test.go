// resolve_csharp_test.go covers Resolve and SpansOf for C# glyphs over scratch trees:
// the namespace scan, overload and partial matching, the self-glyph listings, the per-call build bound,
// and the merge of answers from the Go and C# alphabets.

package engine

import (
	"reflect"
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// resolveOne resolves a single target against r and fails the test on an engine error.
func resolveOne(t *testing.T, r *Repo, target string) ResolveResult {
	t.Helper()
	results, err := r.Resolve([]string{target})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", target, err)
	}
	return results[0]
}

// matchCount returns how many declarations res carries, whether found, multipart or ambiguous.
func matchCount(res ResolveResult) int {
	return len(res.Symbols) + len(res.Candidates)
}

func TestResolve_CSharp(t *testing.T) {
	t.Parallel()
	const shapeSource = "namespace Ns;\n\nclass Shape {\n    void Draw(int a) {}\n    void Draw(double a) {}\n    void Draw() {}\n}\n"
	const ctorSource = "namespace Ns;\n\nclass C {\n    static C() {}\n    C() {}\n}\n"
	const explicitSource = "namespace Ns;\n\ninterface IProvider<T> { T Value { get; } }\nclass A {}\nclass B {}\nclass R : IProvider<A>, IProvider<B> {\n    A IProvider<A>.Value => null;\n    B IProvider<B>.Value => null;\n}\n"
	const twoDirsA = "namespace Ns {\n    class A {}\n}\n"
	const twoDirsB = "namespace Ns {\n    class B {}\n}\n"

	tests := []struct {
		name       string
		files      map[string]string
		target     string
		wantStatus Status
		wantUnit   Status
		wantCount  int
	}{
		{"OverloadInt", map[string]string{"s/shape.cs": shapeSource}, "Ns#Shape.Draw(int)", StatusFound, "", 1},
		{"OverloadDouble", map[string]string{"s/shape.cs": shapeSource}, "Ns#Shape.Draw(double)", StatusFound, "", 1},
		{"OverloadParameterless", map[string]string{"s/shape.cs": shapeSource}, "Ns#Shape.Draw()", StatusFound, "", 1},
		{"OverloadWithoutParametersIsNotFound", map[string]string{"s/shape.cs": shapeSource}, "Ns#Shape.Draw", StatusNotFound, StatusFound, 0},
		{"InstanceConstructor", map[string]string{"c/c.cs": ctorSource}, "Ns#C.C()", StatusFound, "", 1},
		{"StaticConstructor", map[string]string{"c/c.cs": ctorSource}, "Ns#C.static C()", StatusFound, "", 1},
		{"ExplicitImplementationA", map[string]string{"r/r.cs": explicitSource}, "Ns#R.IProvider<A>.Value", StatusFound, "", 1},
		{"ExplicitImplementationB", map[string]string{"r/r.cs": explicitSource}, "Ns#R.IProvider<B>.Value", StatusFound, "", 1},
		{"NamespaceInTwoDirectoriesFirst", map[string]string{"one/a.cs": twoDirsA, "two/b.cs": twoDirsB}, "Ns#A", StatusFound, "", 1},
		{"NamespaceInTwoDirectoriesSecond", map[string]string{"one/a.cs": twoDirsA, "two/b.cs": twoDirsB}, "Ns#B", StatusFound, "", 1},
		{"MemberOfPrefixOnlyNamespace", map[string]string{"one/a.cs": "namespace A.B { class X {} }\n"}, "A#X", StatusNotFound, StatusFound, 0},
		{"GlobalNamespace", map[string]string{"g/g.cs": "class G {}\n"}, "global#G", StatusFound, "", 1},
		{
			"PartialTypeAcrossFiles",
			map[string]string{"p/one.cs": "namespace Ns;\npartial class P {}\n", "p/two.cs": "namespace Ns;\npartial class P {}\n"},
			"Ns#P", StatusMultipart, "", 2,
		},
		{
			"PartialMethodParts",
			map[string]string{"q/q.cs": "namespace Ns;\npartial class Q {\n    partial void M();\n    partial void M() {}\n}\n"},
			"Ns#Q.M()", StatusMultipart, "", 2,
		},
		{
			"NonPartialConditionalDuplicate",
			map[string]string{"d/d.cs": "namespace Ns;\n#if X\nclass D {}\n#else\nclass D {}\n#endif\n"},
			"Ns#D", StatusAmbiguous, "", 2,
		},
		{"UndeclaredNamespaceSelf", map[string]string{"s/shape.cs": shapeSource}, "Nope.Ns#", StatusNotFound, StatusNotFound, 0},
		{"UndeclaredNamespaceMember", map[string]string{"s/shape.cs": shapeSource}, "Nope.Ns#X()", StatusNotFound, StatusNotFound, 0},
		{
			"GitignoredFileIsNotScanned",
			map[string]string{".gitignore": "skip.cs\n", "s/skip.cs": "namespace Hidden;\nclass X {}\n", "s/keep.cs": "namespace Ns;\nclass Y {}\n"},
			"Hidden#X", StatusNotFound, StatusNotFound, 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := openScratchRepo(t, "resolve-csharp-"+tt.name, tt.files)
			res := resolveOne(t, r, tt.target)
			if res.Status != tt.wantStatus || res.Unit != tt.wantUnit || matchCount(res) != tt.wantCount {
				t.Errorf("Resolve(%q) = status %q, unit %q, %d matches; want status %q, unit %q, %d matches",
					tt.target, res.Status, res.Unit, matchCount(res), tt.wantStatus, tt.wantUnit, tt.wantCount)
			}
		})
	}

	t.Run("SelfGlyphListings", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "resolve-csharp-self", map[string]string{
			"one/a.cs":     "namespace A.B;\nclass X {}\n",
			"one/other.cs": "namespace Z;\nclass W {}\n",
			"two/b.cs":     "namespace A.B { class Y {} }\n",
		})

		prefix := resolveOne(t, r, "A#")
		if prefix.Status != StatusFound || prefix.Listings == nil || len(*prefix.Listings) != 0 || prefix.Listing != nil {
			t.Errorf("Resolve(%q) = %+v; want found with an empty, non-nil listings and no listing", "A#", prefix)
		}

		declared := resolveOne(t, r, "A.B#")
		if declared.Status != StatusFound || declared.Listings == nil || declared.Listing != nil {
			t.Fatalf("Resolve(%q) = %+v; want found with listings and no listing", "A.B#", declared)
		}
		var gotDirs []string
		var gotFiles [][]string
		for _, listing := range *declared.Listings {
			gotDirs = append(gotDirs, listing.Dir)
			var names []string
			for _, file := range listing.Files {
				names = append(names, file.Name)
			}
			gotFiles = append(gotFiles, names)
			if len(listing.Dirs) != 0 {
				t.Errorf("listing %q carries dirs %v; want none", listing.Dir, listing.Dirs)
			}
		}
		if want := []string{"one", "two"}; !reflect.DeepEqual(gotDirs, want) {
			t.Errorf("listing directories = %v; want %v", gotDirs, want)
		}
		if want := [][]string{{"a.cs"}, {"b.cs"}}; !reflect.DeepEqual(gotFiles, want) {
			t.Errorf("listing files = %v; want %v", gotFiles, want)
		}

		undeclared := resolveOne(t, r, "Nope#")
		if undeclared.Status != StatusNotFound || undeclared.Unit != StatusNotFound || undeclared.Listings != nil {
			t.Errorf("Resolve(%q) = %+v; want not_found, unit not_found, no listings", "Nope#", undeclared)
		}
	})

	t.Run("SpansOfAgreesWithResolve", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "resolve-csharp-spans", map[string]string{"s/shape.cs": shapeSource})
		res := resolveOne(t, r, "Ns#Shape.Draw(double)")
		if res.Status != StatusFound {
			t.Fatalf("Resolve(Ns#Shape.Draw(double)) status = %q; want found", res.Status)
		}
		g := glyph.Glyph{Lang: glyph.CSharp, Unit: "Ns", Owner: []string{"Shape"}, Name: "Draw", Params: []string{"double"}}
		spans, err := r.SpansOf(g)
		if err != nil {
			t.Fatalf("SpansOf(%v) returned error: %v", g, err)
		}
		if len(spans) != 1 || spans[0].File != res.Symbols[0].File || spans[0].Start != res.Symbols[0].Start || spans[0].End != res.Symbols[0].End {
			t.Errorf("SpansOf(%v) = %+v; want the span Resolve found, %+v", g, spans, res.Symbols[0])
		}
	})

	t.Run("BuildsEachCSharpFileOnce", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "resolve-csharp-builds", map[string]string{
			"one/a.cs":    twoDirsA,
			"two/b.cs":    twoDirsB,
			"two/bare.cs": "class Bare {}\n",
		})
		m, err := newUnitMemo(r)
		if err != nil {
			t.Fatalf("newUnitMemo returned error: %v", err)
		}
		if _, err := r.resolve([]string{"Ns#A", "Ns#B"}, m); err != nil {
			t.Fatalf("resolve returned error: %v", err)
		}
		assertBuilds(t, m.files.builds, []string{"one/a.cs", "two/b.cs", "two/bare.cs"})
		if m.scans != 1 {
			t.Errorf("scans = %d; want 1", m.scans)
		}
	})

	t.Run("NoCSharpTargetBuildsNoScan", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "resolve-csharp-noscan", map[string]string{"one/a.cs": twoDirsA})
		m, err := newUnitMemo(r)
		if err != nil {
			t.Fatalf("newUnitMemo returned error: %v", err)
		}
		if _, err := r.resolve([]string{"missing/dir#F", "other/dir#"}, m); err != nil {
			t.Fatalf("resolve returned error: %v", err)
		}
		assertBuilds(t, m.files.builds, nil)
		if m.scans != 0 {
			t.Errorf("scans = %d; want 0", m.scans)
		}
	})
}

func TestResolve_AcrossAlphabets(t *testing.T) {
	t.Parallel()

	t.Run("NamespaceMatchingDirectory", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "resolve-across-dir", map[string]string{
			"Norce.Models/foo.cs": "namespace Norce.Models { class Foo {} }\n",
		})
		member := resolveOne(t, r, "Norce.Models#Foo")
		if member.Status != StatusFound || len(member.Symbols) != 1 || member.Symbols[0].Language != "" {
			t.Errorf("Resolve(Norce.Models#Foo) = %+v; want found once, with no language", member)
		}
		self := resolveOne(t, r, "Norce.Models#")
		if self.Status != StatusAmbiguous || !reflect.DeepEqual(self.Languages, []string{"csharp", "go"}) || len(self.Candidates) != 0 {
			t.Errorf("Resolve(Norce.Models#) = %+v; want ambiguous, languages [csharp go], no candidates", self)
		}
	})

	t.Run("SharedGlyphString", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "resolve-across-shared", map[string]string{
			"shared/t.go": "package shared\n\nfunc Thing() {}\n",
			"cs/t.cs":     "namespace shared;\nclass Thing {}\n",
		})
		res := resolveOne(t, r, "shared#Thing")
		if res.Status != StatusAmbiguous {
			t.Fatalf("Resolve(shared#Thing) status = %q; want ambiguous", res.Status)
		}
		var languages []string
		for _, candidate := range res.Candidates {
			languages = append(languages, candidate.Language)
		}
		if want := []string{"csharp", "go"}; !reflect.DeepEqual(languages, want) {
			t.Errorf("candidate languages = %v; want %v", languages, want)
		}
	})

	t.Run("MixedDirectory", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "resolve-across-mixed", map[string]string{
			"mix/a.go":      "package mix\n\nfunc F() {}\n",
			"mix/a_test.go": "package mix_test\n\nfunc Ext() {}\n",
			"mix/b.cs":      "namespace Mix;\nclass C {}\n",
		})
		targets := []string{"mix#F", "mix_test#Ext", "Mix#C"}
		results, err := r.Resolve(targets)
		if err != nil {
			t.Fatalf("Resolve(%v) returned error: %v", targets, err)
		}
		for i, res := range results {
			if res.Status != StatusFound || len(res.Symbols) != 1 {
				t.Errorf("Resolve(%q) = %+v; want found once", targets[i], res)
			}
		}
	})

	t.Run("GoOnlyAnswerCarriesNoLanguageKeys", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "resolve-across-goonly", map[string]string{
			"mix/a.go": "package mix\n\nfunc F() {}\n",
			"mix/b.cs": "namespace Mix;\nclass C {}\n",
		})
		for _, target := range []string{"mix#F", "mix#"} {
			res := resolveOne(t, r, target)
			m := marshalToMap(t, res)
			for _, key := range []string{"language", "languages", "listings"} {
				if _, ok := m[key]; ok {
					t.Errorf("Resolve(%q) marshalled with key %q: %v", target, key, m)
				}
			}
			for _, sym := range res.Symbols {
				if sym.Language != "" {
					t.Errorf("Resolve(%q) symbol carries language %q; want none", target, sym.Language)
				}
			}
		}
	})
}
