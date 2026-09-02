package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func init() {
	// Tests run with the package directory as cwd; main() resolves these from the repo root instead.
	repoRoot, _ = filepath.Abs("../../../../..")
	ladderDir = filepath.Join(repoRoot, "bench/loomyard-eval/ladder")
}

func TestDefaultDatedResultsRoot(t *testing.T) {
	cases := map[string]string{
		"bench/loomyard-eval/ladder/ladder.yaml":          "",
		"bench/loomyard-eval/ladder/ladder-followup.yaml": "-followup",
		"bench/loomyard-eval/ladder/ladder-task05.yaml":   "-task05",
		"bench/loomyard-eval/ladder/ladder-toc.yaml":      "-toc",
		"bench/loomyard-eval/ladder/ladder-annex.yaml":    "-annex",
		"bench/loomyard-eval/ladder/ladder-compact.yaml":  "-compact",
		"somewhere/custom.yaml":                           "-custom",
	}
	for in, suffix := range cases {
		got := defaultDatedResultsRoot(in)
		if !strings.HasPrefix(got, filepath.Join(ladderDir, "results")+string(filepath.Separator)) {
			t.Errorf("%s: %q not under results/", in, got)
		}
		if !strings.HasSuffix(got, suffix) {
			t.Errorf("%s: %q does not end in %q", in, got, suffix)
		}
		base := filepath.Base(got)
		if len(base) < len("2006-01-02") || base[4] != '-' || base[7] != '-' {
			t.Errorf("%s: %q does not start with a date", in, got)
		}
	}
}

// TestPrintSummaryTableFlagsUnusedTool pins the one tell the 2026-09-01 follow-up missed: a tool-granted
// cell whose agent never called its tool must be called out, and the table must render every committed
// cell of that results root.
func TestPrintSummaryTableFlagsUnusedTool(t *testing.T) {
	l, err := ladder.LoadLadder(filepath.Join(ladderDir, "ladder-followup.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := printSummaryTable(&buf, l, filepath.Join(ladderDir, "results/2026-09-01-followup")); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, id := range []string{"b0-none", "b1-symbol", "b2-definition", "b4-lsp-trio"} {
		if !strings.Contains(out, "\n"+id) {
			t.Errorf("table lacks a row for %s:\n%s", id, out)
		}
	}
	if !strings.Contains(out, "!! b1-symbol: tool-granted config whose agent never called a granted tool") {
		t.Errorf("b1-symbol (quarry_tool_uses 0 in every rep) was not flagged:\n%s", out)
	}
	for _, id := range []string{"!! b0-none", "!! b2-definition", "!! b4-lsp-trio"} {
		if strings.Contains(out, id) {
			t.Errorf("%s wrongly flagged:\n%s", id, out)
		}
	}
}

// TestTocLadderLoads pins the toc-only ladder's shape: it loads under LoadLadder's rules, grants
// toc_dir and toc_file only ever alone (never together -- August's a3-toc-pair was dropped on
// purpose), keeps the main matrix's ids for the cells that exist there, and has no cold cell.
func TestTocLadderLoads(t *testing.T) {
	l, err := ladder.LoadLadder(filepath.Join(ladderDir, "ladder-toc.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if l.Reps != 5 {
		t.Errorf("reps = %d, want 5", l.Reps)
	}
	wantIDs := []string{"a0-none", "a1-toc-file", "a2-toc-dir", "b0-none", "b8-toc-dir", "b9-toc-file"}
	if len(l.Configs) != len(wantIDs) {
		t.Fatalf("got %d configs, want %d", len(l.Configs), len(wantIDs))
	}
	for i, c := range l.Configs {
		if c.ID != wantIDs[i] {
			t.Errorf("config %d id = %q, want %q", i, c.ID, wantIDs[i])
		}
		if c.Cold {
			t.Errorf("config %q is cold; the toc ladder has no cold cell", c.ID)
		}
		if len(c.Allowed) > 1 {
			t.Errorf("config %q grants %v; toc cells grant one tool only", c.ID, c.Allowed)
		}
		for _, tool := range c.Allowed {
			if tool != "toc_dir" && tool != "toc_file" {
				t.Errorf("config %q grants %q; only toc tools belong in this ladder", c.ID, tool)
			}
		}
	}
}

// TestAnnexLadderLoads pins the annex ladder's shape: every annex cell grants no tools, every annex
// name resolves, each ladder letter has its bare control and its tool-form counterpart, and the
// degraded cell is the only one that drops callers.
func TestAnnexLadderLoads(t *testing.T) {
	l, err := ladder.LoadLadder(filepath.Join(ladderDir, "ladder-annex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	annexCells := 0
	for _, c := range l.Configs {
		if c.Annex == "" {
			continue
		}
		annexCells++
		if len(c.Allowed) != 0 {
			t.Errorf("%s: annex cell grants tools %v", c.ID, c.Allowed)
		}
		spec, err := ladder.AnnexFor(l, c)
		if err != nil {
			t.Errorf("%s: %v", c.ID, err)
		}
		if (spec.DropCallers > 0) != (c.ID == "b12-annex-plan-degraded") {
			t.Errorf("%s: drop_callers=%d", c.ID, spec.DropCallers)
		}
		control, err := ladder.ControlFor(l, c)
		if err != nil || control.Annex != "" || len(control.Allowed) != 0 {
			t.Errorf("%s: ControlFor = %+v, %v", c.ID, control, err)
		}
	}
	if annexCells != 6 {
		t.Errorf("annex cells = %d, want 6", annexCells)
	}
	for _, id := range []string{"a2-toc-dir", "b5-impact"} {
		c, err := ladder.ConfigByID(l, id)
		if err != nil || len(c.Allowed) != 1 {
			t.Errorf("tool-form counterpart %s: %+v, %v", id, c, err)
		}
	}
}

// TestCompactLadderLoads pins the compact ladder's shape: exactly the control and the two tool cells,
// each tool cell pinning toc_format explicitly (json or compact) so the agent cannot opt in or out,
// and no annex cells (deferred, see HANDOFF.md).
func TestCompactLadderLoads(t *testing.T) {
	l, err := ladder.LoadLadder(filepath.Join(ladderDir, "ladder-compact.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"a0-none": "", "a2-toc-dir": "json", "a9-toc-dir-compact": "compact"}
	if len(l.Configs) != len(want) {
		t.Fatalf("got %d configs, want %d", len(l.Configs), len(want))
	}
	for _, c := range l.Configs {
		format, ok := want[c.ID]
		if !ok {
			t.Errorf("unexpected config %s", c.ID)
			continue
		}
		if c.TOCFormat != format || c.Annex != "" {
			t.Errorf("%s: toc_format=%q annex=%q, want toc_format=%q and no annex", c.ID, c.TOCFormat, c.Annex, format)
		}
	}
	if l.Reps != 5 {
		t.Errorf("reps = %d, want 5", l.Reps)
	}
}

func TestServerChangedWarning(t *testing.T) {
	root := t.TempDir()
	write := func(hashes map[string]string) {
		data, _ := json.Marshal(provenance{ServerHashes: hashes})
		if err := os.WriteFile(filepath.Join(root, "provenance.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if w := serverChangedWarning(root); w != "" {
		t.Errorf("no provenance: %q", w)
	}
	write(map[string]string{"a0-none/1": "aaaa", "a0-none/2": "aaaa"})
	if w := serverChangedWarning(root); w != "" {
		t.Errorf("one build: %q", w)
	}
	write(map[string]string{"a0-none/1": "aaaaaaaaaaaaaa", "a2-toc-dir/3": "bbbbbbbbbbbbbb", "a2-toc-dir/4": "bbbbbbbbbbbbbb"})
	w := serverChangedWarning(root)
	if !strings.Contains(w, "2 distinct builds") || !strings.Contains(w, "a2-toc-dir/3,a2-toc-dir/4") {
		t.Errorf("two builds: %q", w)
	}
}
