// plan4_test.go pins docs/rewrite-plan.md §4's file example — "toc
// internal/reedengine/render/layout.go" — at the CLI level. T3 already pins the same answer at the
// engine level (internal/engine/testdata/loomyard/render-layout-file.json); repeating it here, over
// Run's full request pipeline and RenderJSON's actual encoder bytes, is what proves the rendering
// layer added nothing and dropped nothing between the engine's answer and the CLI's stdout.
//
// The assertion decodes stdout into a quarry.DirAnswer and checks fields, rather than
// string-matching §4's own hand-formatted example: that example is line-wrapped for reading in the
// plan document and is not itself a byte-exact transcript. Doc and bandHeader's signature are left
// unasserted because §4 abridges both with "...".

package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// TestPlan4Example pins Run([]string{"toc", "--root", repo, "internal/reedengine/render/layout.go"})
// against docs/rewrite-plan.md §4's file example.
func TestPlan4Example(t *testing.T) {
	repo := loomyardRepo(t)

	args := []string{"toc", "--root", repo, "internal/reedengine/render/layout.go"}
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("Run(%v) code = %d, stderr = %q; want %d", args, code, stderr.String(), exitOK)
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run(%v) stderr = %q; want empty", args, stderr.String())
	}

	raw := stdout.Bytes()
	if bytes.Contains(raw, []byte(`"ok"`)) {
		t.Errorf("stdout contains an \"ok\" key on the success path: %s", raw)
	}
	if !bytes.Contains(raw, []byte("\n  \"")) {
		t.Errorf("stdout does not appear to use two-space indentation: %s", raw)
	}
	if !bytes.HasSuffix(raw, []byte("\n")) || bytes.HasSuffix(raw, []byte("\n\n")) {
		t.Errorf("stdout = %q; want exactly one trailing newline", raw)
	}

	var answer quarry.DirAnswer
	if err := json.Unmarshal(raw, &answer); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", raw, err)
	}

	if answer.Dir != "internal/reedengine/render" {
		t.Errorf("Dir = %q; want %q", answer.Dir, "internal/reedengine/render")
	}
	if answer.Package != "render" {
		t.Errorf("Package = %q; want %q", answer.Package, "render")
	}
	if answer.Language != "go" {
		t.Errorf("Language = %q; want %q", answer.Language, "go")
	}
	if len(answer.Files) != 1 {
		t.Fatalf("len(Files) = %d; want 1", len(answer.Files))
	}
	fe := answer.Files[0]
	if fe.Name != "layout.go" {
		t.Fatalf("Files[0].Name = %q; want %q", fe.Name, "layout.go")
	}
	if fe.Symbols == nil {
		t.Fatalf("Files[0].Symbols is nil; want non-nil")
	}
	syms := *fe.Symbols
	if len(syms) < 4 {
		t.Fatalf("len(Symbols) = %d; want at least 4", len(syms))
	}

	want := []struct {
		id        string
		kind      quarry.Kind
		start     int
		sigEnd    int
		end       int
		signature string
	}{
		{
			id: "internal/reedengine/render#placement", kind: quarry.KindType,
			start: 16, sigEnd: 20, end: 29,
			signature: "type placement struct",
		},
		{
			id: "internal/reedengine/render#buildStackBody", kind: quarry.KindFunction,
			start: 31, sigEnd: 34, end: 50,
			signature: "func buildStackBody(box Box, panes []placement) string",
		},
		{
			id: "internal/reedengine/render#wrapLayout", kind: quarry.KindFunction,
			start: 52, sigEnd: 54, end: 56,
			signature: "func wrapLayout(body string) string",
		},
		{
			id: "internal/reedengine/render#bandHeader", kind: quarry.KindFunction,
			start: 58, sigEnd: 63, end: 76,
			// §4 abridges bandHeader's signature with "..."; it is not asserted here.
		},
	}

	for i, w := range want {
		got := syms[i]
		if got.ID != w.id {
			t.Errorf("Symbols[%d].ID = %q; want %q", i, got.ID, w.id)
		}
		if got.Kind != w.kind {
			t.Errorf("Symbols[%d].Kind = %q; want %q", i, got.Kind, w.kind)
		}
		if got.Start != w.start {
			t.Errorf("Symbols[%d].Start = %d; want %d", i, got.Start, w.start)
		}
		if got.SigEnd != w.sigEnd {
			t.Errorf("Symbols[%d].SigEnd = %d; want %d", i, got.SigEnd, w.sigEnd)
		}
		if got.End != w.end {
			t.Errorf("Symbols[%d].End = %d; want %d", i, got.End, w.end)
		}
		if w.signature != "" && got.Signature != w.signature {
			t.Errorf("Symbols[%d].Signature = %q; want %q", i, got.Signature, w.signature)
		}
	}

	if !strings.HasPrefix(syms[3].Signature, "func bandHeader(") {
		t.Errorf("Symbols[3].Signature = %q; want it to start with %q", syms[3].Signature, "func bandHeader(")
	}
}
