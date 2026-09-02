package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAnnexSpec(t *testing.T) {
	cases := map[string]struct {
		spec AnnexSpec
		ok   bool
	}{
		"toc-dir ok":               {AnnexSpec{Kind: AnnexTocDir, Dirs: []string{"internal/x"}}, true},
		"toc-dir needs dirs":       {AnnexSpec{Kind: AnnexTocDir}, false},
		"toc-dir rejects files":    {AnnexSpec{Kind: AnnexTocDir, Dirs: []string{"a"}, Files: []string{"b"}}, false},
		"toc-file ok":              {AnnexSpec{Kind: AnnexTocFile, Files: []string{"a.go"}}, true},
		"impact ok":                {AnnexSpec{Kind: AnnexImpact, Symbol: "Run", InFile: "a.go", Within: "internal", DropCallers: 1}, true},
		"impact needs in_file":     {AnnexSpec{Kind: AnnexImpact, Symbol: "Run"}, false},
		"impact negative drop":     {AnnexSpec{Kind: AnnexImpact, Symbol: "Run", InFile: "a.go", DropCallers: -1}, false},
		"plan-pack ok":             {AnnexSpec{Kind: AnnexPlanPack, Use: []SymbolRef{{Symbol: "X", InFile: "a.go"}}, Change: []SymbolRef{{Symbol: "Z", InFile: "b.go", Within: "internal"}}}, true},
		"plan-pack needs entries":  {AnnexSpec{Kind: AnnexPlanPack}, false},
		"plan-pack use no within":  {AnnexSpec{Kind: AnnexPlanPack, Use: []SymbolRef{{Symbol: "X", InFile: "a.go", Within: "internal"}}}, false},
		"plan-pack rejects symbol": {AnnexSpec{Kind: AnnexPlanPack, Symbol: "X", Change: []SymbolRef{{Symbol: "Z", InFile: "b.go"}}}, false},
		"unknown kind":             {AnnexSpec{Kind: "grep"}, false},
		"absolute path":            {AnnexSpec{Kind: AnnexTocDir, Dirs: []string{"/tmp/x"}}, false},
	}
	for name, c := range cases {
		rule := validateAnnexSpec(c.spec)
		if (rule == "") != c.ok {
			t.Errorf("%s: validateAnnexSpec() = %q, want ok=%v", name, rule, c.ok)
		}
	}
}

func TestIsControl(t *testing.T) {
	if !IsControl(LadderConfig{}) {
		t.Error("empty config should be a control")
	}
	if IsControl(LadderConfig{Allowed: []string{"toc_dir"}}) {
		t.Error("tool-granted config is not a control")
	}
	if IsControl(LadderConfig{Annex: "x"}) {
		t.Error("annex config is not a control")
	}
}

// fakeCLI returns canned stdout per leading verb and records every call.
func fakeCLI(t *testing.T, calls *[][]string, byVerb map[string]string) CLIRunner {
	return func(bin, dir string, args ...string) (string, error) {
		*calls = append(*calls, args)
		out, ok := byVerb[args[0]+" "+args[1]]
		if !ok {
			out, ok = byVerb[args[0]]
		}
		if !ok {
			t.Fatalf("fake CLI: unexpected call %v", args)
		}
		return out, nil
	}
}

func TestGenerateAnnex_TocFullListsThenTocsEveryFile(t *testing.T) {
	worktree := t.TempDir()
	for _, d := range []string{"internal/a", "internal/b", "internal/b/sub"} {
		if err := os.MkdirAll(filepath.Join(worktree, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	listing := `{"ok":true,"results":[{"path":"internal/a","status":"found","files":[{"path":"internal/a/x.go"},{"path":"internal/a/bad.go","error":"unreadable"}]},{"path":"internal/b","status":"found","files":[{"path":"internal/b/y.go"}]}]}`
	var calls [][]string
	run := fakeCLI(t, &calls, map[string]string{"toc dir": listing, "toc file": `{"ok":true,"results":[]}`})

	annex, err := GenerateAnnex(AnnexSpec{Kind: AnnexTocFull, Dirs: []string{"internal/*"}}, worktree, "/bin/fake", run)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"toc", "dir", "internal/a", "internal/b"},
		{"toc", "file", "internal/a/x.go", "internal/b/y.go"},
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	for i := range want {
		if strings.Join(calls[i], " ") != strings.Join(want[i], " ") {
			t.Errorf("call %d = %v, want %v", i, calls[i], want[i])
		}
	}
	if annex.Kind != AnnexTocFull || annex.Bytes != len(annex.Text) || annex.SHA256 == "" || len(annex.Commands) != 2 {
		t.Errorf("annex metadata = %+v", annex)
	}
}

func TestGenerateAnnex_GlobMatchingNothingFails(t *testing.T) {
	worktree := t.TempDir()
	var calls [][]string
	_, err := GenerateAnnex(AnnexSpec{Kind: AnnexTocDir, Dirs: []string{"nothing/*"}}, worktree, "/bin/fake", fakeCLI(t, &calls, nil))
	var herr *HarnessError
	if !asHarnessError(err, &herr) {
		t.Fatalf("err = %v, want *HarnessError", err)
	}
	if len(calls) != 0 {
		t.Errorf("CLI was called for an empty dirs match: %v", calls)
	}
}

func TestGenerateAnnex_PlanPackLocatesUsesThenImpactsChangesAndDropsCallers(t *testing.T) {
	worktree := t.TempDir()
	impact := `{"ok":true,"callers":[{"file":"` + worktree + `/a.go","line":1},{"file":"` + worktree + `/b.go","line":2},{"file":"` + worktree + `/c.go","line":3}],"target":{"name":"Z"}}`
	var calls [][]string
	run := fakeCLI(t, &calls, map[string]string{
		"definition": `{"ok":true,"results":[{"symbol":"X","definitions":[{"file":"` + worktree + `/x.go","line":5}]}]}`,
		"impact":     impact,
	})
	spec := AnnexSpec{
		Kind:        AnnexPlanPack,
		Use:         []SymbolRef{{Symbol: "X", InFile: "x.go"}, {Symbol: "Y", InFile: "x.go"}, {Symbol: "W", InFile: "w.go"}},
		Change:      []SymbolRef{{Symbol: "Z", InFile: "z.go", Within: "internal"}},
		DropCallers: 1,
	}
	annex, err := GenerateAnnex(spec, worktree, "/bin/fake", run)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"definition", "--in-file", "x.go", "X", "Y"},
		{"definition", "--in-file", "w.go", "W"},
		{"impact", "--in-file", "z.go", "--within", "internal", "Z"},
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	for i := range want {
		if strings.Join(calls[i], " ") != strings.Join(want[i], " ") {
			t.Errorf("call %d = %v, want %v", i, calls[i], want[i])
		}
	}
	if annex.DroppedCallers != 1 {
		t.Errorf("DroppedCallers = %d, want 1", annex.DroppedCallers)
	}
	if strings.Contains(annex.Text, "c.go") || !strings.Contains(annex.Text, "b.go") {
		t.Errorf("last caller not dropped: %s", annex.Text)
	}
	if strings.Contains(annex.Text, worktree) {
		t.Errorf("worktree path not relativised: %s", annex.Text)
	}
	if !strings.Contains(annex.Text, "# declaration location of: X, Y (declared in x.go)") ||
		!strings.Contains(annex.Text, "# declaration and every call site of: Z (declared in z.go)") {
		t.Errorf("section headers missing: %s", annex.Text)
	}
}

func TestGenerateAnnex_DropMoreCallersThanPresentFails(t *testing.T) {
	var calls [][]string
	run := fakeCLI(t, &calls, map[string]string{"impact": `{"ok":true,"callers":[{"file":"a.go"}]}`})
	_, err := GenerateAnnex(AnnexSpec{Kind: AnnexImpact, Symbol: "Z", InFile: "z.go", DropCallers: 1}, t.TempDir(), "/bin/fake", run)
	var herr *HarnessError
	if !asHarnessError(err, &herr) {
		t.Fatalf("err = %v, want *HarnessError", err)
	}
}

func TestGenerateAnnex_RefusesTextThatBreaksBlinding(t *testing.T) {
	var calls [][]string
	run := fakeCLI(t, &calls, map[string]string{"toc file": `{"header":"built by Quarry"}`})
	_, err := GenerateAnnex(AnnexSpec{Kind: AnnexTocFile, Files: []string{"a.go"}}, t.TempDir(), "/bin/fake", run)
	var herr *HarnessError
	if !asHarnessError(err, &herr) || !strings.Contains(herr.Message, "blinding") {
		t.Fatalf("err = %v, want a blinding HarnessError", err)
	}
}

func TestWriteAndReadAnnex(t *testing.T) {
	dir := t.TempDir()
	annex := Annex{Text: "hello", Kind: AnnexTocDir, Label: "x", Commands: [][]string{{"toc", "dir", "a"}}, Bytes: 5, SHA256: "abc"}
	if err := WriteAnnex(dir, annex); err != nil {
		t.Fatal(err)
	}
	back, err := ReadAnnex(dir)
	if err != nil || back.Text != "hello" || back.Label != "x" || back.Kind != AnnexTocDir {
		t.Fatalf("ReadAnnex() = %+v, %v", back, err)
	}
	meta, err := os.ReadFile(filepath.Join(dir, AnnexMetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(meta, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, has := decoded["text"]; has {
		t.Error("annex.meta.json must not embed the text")
	}
	if decoded["kind"] != AnnexTocDir || decoded["bytes"].(float64) != 5 {
		t.Errorf("meta = %v", decoded)
	}
	if _, err := ReadAnnex(t.TempDir()); err == nil {
		t.Error("ReadAnnex on an empty dir should fail")
	}
}

func TestAnnexBlockIsNeutral(t *testing.T) {
	block := AnnexBlock("/tmp/wt", Annex{Text: "{}", Label: annexLabels[AnnexPlanPack]})
	for _, forbidden := range []string{"quarry", "mcp", "toc", "impact", "--"} {
		head := strings.SplitN(block, "--- BEGIN ATTACHMENT ---", 2)[0]
		if strings.Contains(strings.ToLower(head), forbidden) && forbidden != "--" {
			t.Errorf("annex block preamble mentions %q: %s", forbidden, head)
		}
	}
	if !strings.Contains(block, "--- BEGIN ATTACHMENT ---\n{}\n--- END ATTACHMENT ---") {
		t.Errorf("block = %s", block)
	}
}

func TestPreambleFor_InsertsAnnexBetweenBodyAndTask(t *testing.T) {
	l := &Ladder{QuarryTools: QuarryTools}
	config := LadderConfig{ID: "x1", Ladder: "a", Annex: "pack"}
	prompt := PreambleFor(l, config, "/tmp/wt", "TASK", "{}", "ANNEX-BLOCK")
	body := strings.Index(prompt, "standard tools")
	annex := strings.Index(prompt, "ANNEX-BLOCK")
	task := strings.Index(prompt, "TASK")
	if !(body >= 0 && body < annex && annex < task) {
		t.Errorf("order body(%d) < annex(%d) < task(%d) violated:\n%s", body, annex, task, prompt)
	}
	control := PreambleFor(l, LadderConfig{ID: "a0", Ladder: "a"}, "/tmp/wt", "TASK", "{}", "")
	if strings.Replace(prompt, "ANNEX-BLOCK\n\n", "", 1) != control {
		t.Error("annex prompt is not the control prompt plus exactly the annex paragraph")
	}
}

func asHarnessError(err error, target **HarnessError) bool {
	h, ok := err.(*HarnessError)
	if ok {
		*target = h
	}
	return ok
}

func TestGenerateAnnex_CompactTocFullUsesJSONListingForDiscoveryOnly(t *testing.T) {
	worktree := t.TempDir()
	if err := os.MkdirAll(filepath.Join(worktree, "internal/a"), 0o755); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	run := func(bin, dir string, args ...string) (string, error) {
		calls = append(calls, args)
		switch strings.Join(args[:2], " ") {
		case "toc dir":
			if len(args) > 2 && args[2] == "--compact" {
				return "# internal/a (package a), 1 files\ninternal/a/x.go: X.", nil
			}
			return `{"ok":true,"files":[{"path":"internal/a/x.go"}],"dirs":[]}`, nil
		case "toc file":
			return "# internal/a/x.go (package a)\n1-2: func X()", nil
		}
		t.Fatalf("unexpected call %v", args)
		return "", nil
	}
	annex, err := GenerateAnnex(AnnexSpec{Kind: AnnexTocFull, Compact: true, Dirs: []string{"internal/a"}}, worktree, "/bin/fake", run)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"toc", "dir", "--compact", "internal/a"},
		{"toc", "dir", "internal/a"},
		{"toc", "file", "--compact", "internal/a/x.go"},
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	for i := range want {
		if strings.Join(calls[i], " ") != strings.Join(want[i], " ") {
			t.Errorf("call %d = %v, want %v", i, calls[i], want[i])
		}
	}
	if strings.Contains(annex.Text, `"ok"`) || !annex.Compact {
		t.Errorf("JSON discovery listing leaked into the annex text, or Compact unset: %+v", annex)
	}
}

func TestMCPConfigDocument_TOCFormatArg(t *testing.T) {
	args := MCPConfigDocument("/bin/srv", "/tmp/t", "compact")["mcpServers"].(map[string]any)["quarry"].(map[string]any)["args"].([]string)
	if strings.Join(args, " ") != "--target-dir /tmp/t --toc-format compact" {
		t.Errorf("args = %v", args)
	}
}
