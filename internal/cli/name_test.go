// name_test.go covers the name verb: codeForNameResult as a direct table test, the no-root proof
// runName's dispatch-before-root-resolution earns, the multi-line divergence between the two
// views, and the usage text's own name-verb rows.

package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// TestCodeForNameResult covers codeForNameResult's four branches, mirroring the table tests the
// other three exit-code mappers already have in cli_test.go.
func TestCodeForNameResult(t *testing.T) {
	tests := []struct {
		name string
		r    quarry.NameResult
		want int
	}{
		{"id-set", quarry.NameResult{ID: "pkg#Sym"}, exitOK},
		{"internal-reason", quarry.NameResult{Reason: quarry.NameReasonInternal, Error: "internal error: boom"}, exitInternal},
		{"other-error", quarry.NameResult{Error: "declaration does not parse", Reason: quarry.NameReasonParse}, exitNegative},
		{"neither-id-nor-error", quarry.NameResult{}, exitInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeForNameResult(tt.r); got != tt.want {
				t.Errorf("codeForNameResult(%+v) = %d; want %d", tt.r, got, tt.want)
			}
		})
	}
}

// The internal reason's own bytes on runName's compact-error-envelope route — the envelope on
// stdout, the same sentence on stderr, no payload — are deliberately left untested here.
// codeForNameResult's table above already covers that the internal reason maps to exit 3.
// runName calls the facade directly with no injectable seam, and the facade cannot produce the
// internal reason while the Go grammar is always wired, so the only way to reach those bytes
// would be to add a maker seam to production code for a branch that cannot fire. That trade is
// worse than the untested branch: it would put a test-only indirection on the verb's one hot path
// to cover a condition the code already spells rather than falls through. The branch's contract
// lives in runName's own doc comment, which states the step order and why the internal check
// comes first.

// TestRunName_NoRepositoryRoot proves both halves of the no-root claim in one place: from the
// filesystem root, name answers correctly with no root resolved at all, while toc — a repository
// verb — fails with the no-repository-root usage sentence. One without the other proves nothing:
// the second half is what shows the first is not passing for some unrelated reason.
func TestRunName_NoRepositoryRoot(t *testing.T) {
	t.Chdir("/")

	code, stdout, stderr := runCLI([]string{"name", "--unit", "u/v", "func F() error"})
	if code != exitOK {
		t.Fatalf("name code = %d, stdout = %q, stderr = %q; want %d", code, stdout, stderr, exitOK)
	}
	var result quarry.NameResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
	}
	if result.ID == "" {
		t.Errorf("name result = %+v; want a non-empty id", result)
	}

	code, stdout, stderr = runCLI([]string{"toc", "."})
	if code != exitUsage {
		t.Fatalf("toc code = %d; want %d", code, exitUsage)
	}
	want := "no repository root found above /; pass --root"
	got := failureEnvelope(t, stdout)
	if got != want {
		t.Errorf("toc error = %q; want %q", got, want)
	}
	if !strings.Contains(stderr, want) {
		t.Errorf("toc stderr = %q; want it to contain %q", stderr, want)
	}
}

// TestRunName_MultiLineHead pins the same multi-line-head divergence quarry/name_test.go's own
// TestRenderName_MultiLineDivergence pins at the renderer level, exercised here end to end through
// Run: the declaration is an ungrouped var with a composite literal spanning several lines.
func TestRunName_MultiLineHead(t *testing.T) {
	decl := "var Config = Widget{\n\tValue: 1,\n}"

	code, stdout, stderr := runCLI([]string{"name", "--unit", "pkg", decl, "--text"})
	if code != exitOK {
		t.Fatalf("code = %d, stdout = %q, stderr = %q; want %d", code, stdout, stderr, exitOK)
	}
	if n := strings.Count(stdout, "\n"); n != 1 {
		t.Errorf("stdout = %q; want exactly one newline, at the end", stdout)
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Errorf("stdout = %q; want it to end with a newline", stdout)
	}

	code, stdout, stderr = runCLI([]string{"name", "--unit", "pkg", decl})
	if code != exitOK {
		t.Fatalf("code = %d, stdout = %q, stderr = %q; want %d", code, stdout, stderr, exitOK)
	}
	var decoded struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
	}
	if decoded.Target != decl {
		t.Errorf("target = %q; want %q (line breaks intact)", decoded.Target, decl)
	}
}

// TestUsageText_NameVerb pins that usageText carries the name verb's usage line, the --unit flags
// row, and the --root not-valid-for-name marker, and remains ASCII only.
func TestUsageText_NameVerb(t *testing.T) {
	if !strings.Contains(usageText, "quarry name <declaration> --unit <unit> [--text]") {
		t.Errorf("usageText = %q; want the name usage line", usageText)
	}
	if !strings.Contains(usageText, "--unit <unit>") {
		t.Errorf("usageText = %q; want a --unit flags row", usageText)
	}
	if !strings.Contains(usageText, "not valid for name") {
		t.Errorf("usageText = %q; want the --root not-valid-for-name marker", usageText)
	}
	for i := 0; i < len(usageText); i++ {
		if usageText[i] >= 0x80 {
			t.Fatalf("usageText contains a non-ASCII byte at index %d: %q", i, usageText[i])
		}
	}
}
