// string_test.go covers Glyph.String: the printed form for a package-level Go name, a method with a
// one-element Owner, the nil-versus-non-nil-empty Params rule, a populated Params list, a
// two-element Owner, the zero Glyph, and that String does not mutate its receiver's Owner slice.

package glyph

import "testing"

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
