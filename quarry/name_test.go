// name_test.go covers Name's delegation to the engine, RenderNameJSON's and RenderNameText's byte
// contracts, and the deliberate multi-line divergence between the two renderers, over hand-built
// Declaration and NameResult values only — no filesystem, no Open.

package quarry

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestName_Delegation covers that Name returns what the engine returns, positionally, over a batch
// mixing one success and one failure: order is preserved, the result length equals the input length,
// and each entry's unit and target are echoed byte-identically.
func TestName_Delegation(t *testing.T) {
	decls := []Declaration{
		{Unit: "pkg", Decl: "func Sym()"},
		{Unit: "pkg", Decl: "const ( X = 1\nY = 2 )"},
	}
	got := Name(decls)
	if len(got) != len(decls) {
		t.Fatalf("Name() len = %d; want %d", len(got), len(decls))
	}
	for i, d := range decls {
		if got[i].Unit != d.Unit {
			t.Errorf("Name()[%d].Unit = %q; want %q", i, got[i].Unit, d.Unit)
		}
		if got[i].Target != d.Decl {
			t.Errorf("Name()[%d].Target = %q; want %q", i, got[i].Target, d.Decl)
		}
	}
	if got[0].ID == "" {
		t.Errorf("Name()[0] = %+v; want a successful prediction", got[0])
	}
	if got[1].ID != "" {
		t.Errorf("Name()[1] = %+v; want a failed prediction", got[1])
	}
}

// TestName_EmptyInput covers that an empty input returns an empty, non-nil slice.
func TestName_EmptyInput(t *testing.T) {
	got := Name(nil)
	if got == nil {
		t.Fatal("Name(nil) = nil; want an empty, non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("Name(nil) = %+v; want an empty slice", got)
	}
}

// TestRenderNameJSON covers the byte contract: two-space indent, exactly one trailing newline, no
// HTML escaping, and the exact emitted key set for a success payload and for a failure payload.
func TestRenderNameJSON(t *testing.T) {
	tests := []struct {
		name    string
		r       NameResult
		wantKey []string
	}{
		{
			"Success",
			NameResult{Unit: "pkg", Target: "func Sym()", ID: "pkg#Sym", Kind: KindFunction},
			[]string{"unit", "target", "id", "kind"},
		},
		{
			"Failure",
			NameResult{Unit: "pkg", Target: "func Sym(", Error: "declaration does not parse", Reason: NameReasonParse},
			[]string{"unit", "target", "error", "reason"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderNameJSON(tt.r)
			if err != nil {
				t.Fatalf("RenderNameJSON() error = %v", err)
			}
			s := string(got)
			if !strings.Contains(s, "\n  \"unit\"") {
				t.Errorf("RenderNameJSON() = %q; want two-space indentation before \"unit\"", s)
			}
			if !strings.HasSuffix(s, "\n") || strings.HasSuffix(s, "\n\n") {
				t.Errorf("RenderNameJSON() = %q; want exactly one trailing newline", s)
			}
			if strings.Contains(s, `<`) || strings.Contains(s, `>`) || strings.Contains(s, `&`) {
				t.Errorf("RenderNameJSON() = %q; want no HTML-escaped characters", s)
			}
			if strings.Contains(s, `"ok"`) {
				t.Errorf("RenderNameJSON() = %s; want no \"ok\" key", s)
			}

			var m map[string]json.RawMessage
			if err := json.Unmarshal(got, &m); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if len(m) != len(tt.wantKey) {
				t.Fatalf("RenderNameJSON() key set = %v; want exactly %v", keysOf(m), tt.wantKey)
			}
			for _, k := range tt.wantKey {
				if _, ok := m[k]; !ok {
					t.Errorf("RenderNameJSON() = %s; want key %q present", s, k)
				}
			}
		})
	}
}

// keysOf returns m's keys, for a failure message readable without decoding the map again.
func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestRenderNameText covers the success line and the failure line, byte for byte, including that
// each ends with exactly one newline and carries no trailing whitespace.
func TestRenderNameText(t *testing.T) {
	tests := []struct {
		name string
		r    NameResult
		want string
	}{
		{
			"Success",
			NameResult{Unit: "pkg", Target: "func Sym()", ID: "pkg#Sym", Kind: KindFunction},
			"pkg#Sym function\n",
		},
		{
			"FailureWithReasonAndError",
			NameResult{Unit: "pkg", Target: "func Sym(", Error: "declaration does not parse", Reason: NameReasonParse},
			"func Sym( error parse: declaration does not parse\n",
		},
		{
			"FailureEmptyReason",
			NameResult{Unit: "pkg", Target: "func Sym(", Error: "boom"},
			"func Sym( error: boom\n",
		},
		{
			"FailureEmptyError_TotalityGuard",
			NameResult{Unit: "pkg", Target: "func Sym("},
			"func Sym( error\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderNameText(tt.r)
			if got != tt.want {
				t.Errorf("RenderNameText() = %q; want %q", got, tt.want)
			}
			if !strings.HasSuffix(got, "\n") || strings.HasSuffix(got, "\n\n") {
				t.Errorf("RenderNameText() = %q; want exactly one trailing newline", got)
			}
			for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
				if line != strings.TrimRight(line, " \t") {
					t.Errorf("RenderNameText() = %q; line %q has trailing whitespace", got, line)
				}
			}
		})
	}
}

// TestRenderName_MultiLineDivergence pins the deliberate divergence between the two renderers for a
// declaration head spanning lines: RenderNameText contains exactly one newline, at the end, while
// RenderNameJSON still carries the head's own line breaks inside its "target" value. The JSON half is
// asserted after unmarshalling, never against the raw bytes, since renderJSON disables HTML escaping
// only and the encoder still emits "\n" as the two-character escape sequence.
func TestRenderName_MultiLineDivergence(t *testing.T) {
	multiLine := "var (\n\tX = 1\n\tY = 2\n)"
	r := NameResult{Unit: "pkg", Target: multiLine, Error: "declaration declares 2 symbols; exactly one is required", Reason: NameReasonSeveralDeclarations}

	text := RenderNameText(r)
	if strings.Count(text, "\n") != 1 {
		t.Errorf("RenderNameText() = %q; want exactly one newline, at the end", text)
	}
	if !strings.HasSuffix(text, "\n") {
		t.Errorf("RenderNameText() = %q; want it to end with a newline", text)
	}

	jsonBytes, err := RenderNameJSON(r)
	if err != nil {
		t.Fatalf("RenderNameJSON() error = %v", err)
	}
	var decoded struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Target != multiLine {
		t.Errorf("RenderNameJSON() target = %q; want %q (line breaks intact)", decoded.Target, multiLine)
	}
}
