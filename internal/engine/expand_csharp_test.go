// expand_csharp_test.go covers Repo.Expand for C# glyphs over scratch trees:
// head and member selection for nested and explicitly implemented members, partial types, the not-a-type and not-found rows,
// and the merge of answers from the Go and C# alphabets.

package engine

import (
	"errors"
	"reflect"
	"testing"
)

// idsOf returns the id of every symbol in syms, in order.
func idsOf(syms []Symbol) []string {
	var ids []string
	for _, sym := range syms {
		ids = append(ids, sym.ID)
	}
	return ids
}

// symbolLanguages returns the language of every symbol in syms, in order.
func symbolLanguages(syms []Symbol) []string {
	var languages []string
	for _, sym := range syms {
		languages = append(languages, sym.Language)
	}
	return languages
}

func TestExpand_CSharp(t *testing.T) {
	t.Parallel()

	t.Run("TypeListsMembersAndNestedMembers", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "expand-csharp-type", map[string]string{
			"o/outer.cs": "namespace Ns;\nclass Outer : System.IDisposable\n{\n    public int F;\n    public void M() {}\n    public class Inner { public void N() {} }\n}\n",
		})
		got, err := r.Expand("Ns#Outer")
		if err != nil {
			t.Fatalf("Expand(Ns#Outer) returned error: %v", err)
		}
		if got.Status != StatusFound || got.Head == nil || got.Head.Start != 2 || got.Head.End != 2 {
			t.Fatalf("Expand(Ns#Outer) = %+v; want found with a head ending at the base list on line 2", got)
		}
		want := []string{"Ns#Outer.F", "Ns#Outer.M()", "Ns#Outer.Inner", "Ns#Outer.Inner.N()"}
		if ids := idsOf(got.Members); !reflect.DeepEqual(ids, want) {
			t.Errorf("member ids = %v; want %v", ids, want)
		}
	})

	t.Run("NestedTypeIgnoresTopLevelNamesake", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "expand-csharp-nested", map[string]string{
			"o/outer.cs": "namespace Ns;\nclass Outer { class Inner { void A() {} } }\nclass Inner { void B() {} }\n",
		})
		tests := []struct {
			target string
			want   []string
		}{
			{"Ns#Outer.Inner", []string{"Ns#Outer.Inner.A()"}},
			{"Ns#Inner", []string{"Ns#Inner.B()"}},
		}
		for _, tt := range tests {
			got, err := r.Expand(tt.target)
			if err != nil {
				t.Fatalf("Expand(%q) returned error: %v", tt.target, err)
			}
			if got.Status != StatusFound || got.ID != tt.target {
				t.Errorf("Expand(%q) = status %q, id %q; want found under the target's own id", tt.target, got.Status, got.ID)
			}
			if ids := idsOf(got.Members); !reflect.DeepEqual(ids, tt.want) {
				t.Errorf("Expand(%q) member ids = %v; want %v", tt.target, ids, tt.want)
			}
		}
	})

	t.Run("ExplicitImplementationBelongsToNestedType", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "expand-csharp-explicit", map[string]string{
			"r/renderer.cs": "namespace Ns;\nclass Renderer : System.IDisposable\n{\n    class System { }\n    void System.IDisposable.Dispose() { }\n}\n",
		})
		got, err := r.Expand("Ns#Renderer.System")
		if err != nil {
			t.Fatalf("Expand(Ns#Renderer.System) returned error: %v", err)
		}
		want := []string{"Ns#Renderer.System.IDisposable.Dispose()"}
		if got.Status != StatusFound || !reflect.DeepEqual(idsOf(got.Members), want) {
			t.Errorf("Expand(Ns#Renderer.System) = %+v; want found with members %v", got, want)
		}
	})

	t.Run("PartialTypeIsMultipart", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "expand-csharp-partial", map[string]string{
			"p/one.cs": "namespace Ns;\npartial class P : System.IDisposable\n{\n    void A() {}\n}\n",
			"p/two.cs": "namespace Ns;\npartial class P\n{\n    void B() {}\n}\n",
		})
		got, err := r.Expand("Ns#P")
		if err != nil {
			t.Fatalf("Expand(Ns#P) returned error: %v", err)
		}
		if got.Status != StatusMultipart || got.Head != nil || len(got.Parts) != 2 {
			t.Fatalf("Expand(Ns#P) = %+v; want multipart with two parts and no head", got)
		}
		if got.Parts[0].File != "p/one.cs" || got.Parts[0].Start != 2 || got.Parts[0].End != 2 {
			t.Errorf("first part = %+v; want the head of p/one.cs, line 2", got.Parts[0])
		}
		if want := []string{"Ns#P.A()", "Ns#P.B()"}; !reflect.DeepEqual(idsOf(got.Members), want) {
			t.Errorf("member ids = %v; want %v", idsOf(got.Members), want)
		}
	})

	t.Run("MethodIsNotAType", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "expand-csharp-method", map[string]string{
			"s/shape.cs": "namespace Ns;\nclass Shape { void Draw() {} }\n",
		})
		got, err := r.Expand("Ns#Shape.Draw()")
		var notAType *NotATypeError
		if !errors.As(err, &notAType) || notAType.Kind != KindMethod {
			t.Fatalf("Expand(Ns#Shape.Draw()) error = %v; want a *NotATypeError of kind %q", err, KindMethod)
		}
		if !isZeroExpandAnswer(got) {
			t.Errorf("Expand(Ns#Shape.Draw()) = %+v; want the zero ExpandAnswer", got)
		}
	})

	t.Run("MissingMemberInDeclaredNamespace", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "expand-csharp-missing", map[string]string{
			"s/shape.cs": "namespace Ns;\nclass Shape {}\n",
		})
		got, err := r.Expand("Ns#Nope")
		if err != nil {
			t.Fatalf("Expand(Ns#Nope) returned error: %v", err)
		}
		if got.Status != StatusNotFound || got.Unit != StatusFound {
			t.Errorf("Expand(Ns#Nope) = %+v; want not_found with unit found", got)
		}
	})

	t.Run("GoTypeAndCSharpTypeShareAGlyph", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "expand-csharp-two-types", map[string]string{
			"shared/t.go": "package shared\n\ntype Thing struct{}\n",
			"cs/t.cs":     "namespace shared;\nclass Thing {}\n",
		})
		got, err := r.Expand("shared#Thing")
		if err != nil {
			t.Fatalf("Expand(shared#Thing) returned error: %v", err)
		}
		if got.Status != StatusAmbiguous || got.Head != nil || got.Members != nil {
			t.Fatalf("Expand(shared#Thing) = %+v; want ambiguous with no head or members", got)
		}
		if want := []string{"csharp", "go"}; !reflect.DeepEqual(symbolLanguages(got.Candidates), want) {
			t.Errorf("candidate languages = %v; want %v", symbolLanguages(got.Candidates), want)
		}
	})

	t.Run("GoFunctionAndCSharpTypeShareAGlyph", func(t *testing.T) {
		t.Parallel()
		r := openScratchRepo(t, "expand-csharp-func-and-type", map[string]string{
			"shared/t.go": "package shared\n\nfunc Thing() {}\n",
			"cs/t.cs":     "namespace shared;\nclass Thing {}\n",
		})
		got, err := r.Expand("shared#Thing")
		if err != nil {
			t.Fatalf("Expand(shared#Thing) returned error: %v", err)
		}
		if got.Status != StatusAmbiguous {
			t.Fatalf("Expand(shared#Thing) = %+v; want ambiguous", got)
		}
		if want := []string{"csharp", "go"}; !reflect.DeepEqual(symbolLanguages(got.Candidates), want) {
			t.Errorf("candidate languages = %v; want %v", symbolLanguages(got.Candidates), want)
		}
	})
}
