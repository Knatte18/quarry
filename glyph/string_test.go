// string_test.go covers Glyph.String: the printed form for a package-level Go name, a method with a
// one-element Owner, the nil-versus-non-nil-empty Params rule, a populated Params list, a
// two-element Owner, the zero Glyph, that String does not mutate its receiver's Owner slice, and
// the Parse/String round trip in both directions over the Go accept table.

package glyph

import (
	"reflect"
	"testing"
)

// TestGlyph_String asserts the exact printed string for a table of hand-built Glyph values.
func TestGlyph_String(t *testing.T) {
	tests := []struct {
		name string
		g    Glyph
		want string
	}{
		{
			name: "package-level Go name, nil Params prints no parentheses",
			g:    Glyph{Lang: Go, Unit: "internal/logger", Name: "stderrHandlerSnapshot"},
			want: "internal/logger#stderrHandlerSnapshot",
		},
		{
			name: "method with one-element Owner, nil Params prints no parentheses",
			g: Glyph{
				Lang: Go, Unit: "internal/reedengine/render",
				Owner: []string{"Renderer"}, Name: "Draw",
			},
			want: "internal/reedengine/render#Renderer.Draw",
		},
		{
			name: "non-nil empty Params prints ()",
			g: Glyph{
				Lang: Go, Unit: "Loomyard.Engine.Layout",
				Owner: []string{"Renderer"}, Name: "Draw", Params: []string{},
			},
			want: "Loomyard.Engine.Layout#Renderer.Draw()",
		},
		{
			name: "populated Params prints comma-separated, no spaces",
			g: Glyph{
				Lang: Go, Unit: "Loomyard.Engine.Layout",
				Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"int", "string"},
			},
			want: "Loomyard.Engine.Layout#Renderer.Draw(int,string)",
		},
		{
			name: "two-element Owner under Lang Go",
			g: Glyph{
				Lang: Go, Unit: "internal/logger",
				Owner: []string{"Outer", "Inner"}, Name: "handle",
			},
			want: "internal/logger#Outer.Inner.handle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.String(); got != tt.want {
				t.Errorf("Glyph.String() = %q; want %q", got, tt.want)
			}
		})
	}
}

// TestGlyph_String_Zero asserts only that String does not panic on the zero Glyph.
func TestGlyph_String_Zero(t *testing.T) {
	var g Glyph
	_ = g.String()
}

// TestGlyph_String_DoesNotMutateOwner asserts that String does not write into its receiver's
// Owner slice, even when Owner has spare capacity that append could otherwise reuse.
func TestGlyph_String_DoesNotMutateOwner(t *testing.T) {
	owner := make([]string, 1, 4)
	owner[0] = "Renderer"
	g := Glyph{Lang: Go, Unit: "internal/reedengine/render", Owner: owner, Name: "Draw"}

	first := g.String()

	if len(owner) != 1 {
		t.Fatalf("len(owner) after String() = %d; want 1", len(owner))
	}
	if owner[0] != "Renderer" {
		t.Fatalf("owner[0] after String() = %q; want %q", owner[0], "Renderer")
	}

	second := g.String()
	if second != first {
		t.Errorf("second String() call = %q; want %q (same as first call)", second, first)
	}
}

// TestRoundTrip_ParseThenString asserts that, for every input in the Go accept table, Parse then
// String returns exactly the original input. This is total over goAccept: the Go alphabet accepts
// only the canonical spelling, so no case is carved out.
func TestRoundTrip_ParseThenString(t *testing.T) {
	for _, tt := range goAccept {
		t.Run(tt.name+"/"+tt.section, func(t *testing.T) {
			g, err := Parse(Go, tt.input)
			if err != nil {
				t.Fatalf("Parse(Go, %q) error = %v; want nil", tt.input, err)
			}
			if got := g.String(); got != tt.input {
				t.Errorf("Parse(Go, %q).String() = %q; want %q", tt.input, got, tt.input)
			}
		})
	}
}

// TestRoundTrip_StringThenParse asserts that, for every input in the Go accept table, parsing the
// printed form of the parsed Glyph yields a Glyph equal to the first. This is total over goAccept,
// like TestRoundTrip_ParseThenString above.
func TestRoundTrip_StringThenParse(t *testing.T) {
	for _, tt := range goAccept {
		t.Run(tt.name+"/"+tt.section, func(t *testing.T) {
			first, err := Parse(Go, tt.input)
			if err != nil {
				t.Fatalf("Parse(Go, %q) error = %v; want nil", tt.input, err)
			}
			second, err := Parse(Go, first.String())
			if err != nil {
				t.Fatalf("Parse(Go, %q) (printed form) error = %v; want nil", first.String(), err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Errorf("round trip through String() = %+v; want %+v", second, first)
			}
		})
	}
}
