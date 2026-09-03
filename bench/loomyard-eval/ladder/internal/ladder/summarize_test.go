package ladder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestProvenance writes a minimal provenance record naming selectedCells and repsEffective,
// exactly the two fields Summarize reads off it.
func writeTestProvenance(t *testing.T, root string, selectedCells []string, repsEffective int) {
	t.Helper()
	p := &Provenance{
		SelectedCells: selectedCells,
		RepsEffective: repsEffective,
		ClaudeVersion: "2.5.0 (Claude Code)",
	}
	if err := WriteProvenance(root, p); err != nil {
		t.Fatalf("WriteProvenance() error = %v", err)
	}
}

// writeTestTranscript writes a repetition's transcript.jsonl carrying exactly numTurns,
// durationMS, costUSD and quarryToolUses calls of the granted mcp__quarry__toc tool -- enough for
// ComputeMetrics to recompute every cost metric summarizeCell reads.
func writeTestTranscript(t *testing.T, dir string, numTurns int, durationMS int64, costUSD float64, quarryToolUses int) {
	t.Helper()
	var blocks []string
	for i := 0; i < quarryToolUses; i++ {
		blocks = append(blocks, fmt.Sprintf(
			`{"type":"tool_use","id":"tu%d","name":"mcp__quarry__toc","input":{}}`, i,
		))
	}
	assistant := fmt.Sprintf(
		`{"type":"assistant","message":{"id":"msg1","content":[%s]}}`, strings.Join(blocks, ","),
	)
	result := fmt.Sprintf(
		`{"type":"result","num_turns":%d,"duration_ms":%d,"total_cost_usd":%v,"terminal_reason":"end_turn"}`,
		numTurns, durationMS, costUSD,
	)
	data := assistant + "\n" + result + "\n"
	path := filepath.Join(dir, TranscriptFile)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

// writeTestScore writes a repetition's score.json carrying exactly recall and precision.
func writeTestScore(t *testing.T, dir string, recall, precision float64) {
	t.Helper()
	data := fmt.Sprintf(`{"scored":true,"recall":%v,"precision":%v}`, recall, precision)
	path := filepath.Join(dir, ScoreFile)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

// repFixture is one repetition's fixture input: the RunState fields summarizeCell reads plus,
// for a present-and-not-void repetition, the transcript and score to write alongside it.
type repFixture struct {
	blindingFailed bool
	maxTurnsHit    bool
	scored         bool
	numTurns       int
	durationMS     int64
	costUSD        float64
	quarryToolUses int
	recall         float64
	precision      float64
}

// writeTestRep writes one repetition directory under root for cellID's rep, from a repFixture and
// the cell-level fields every one of its repetitions shares.
func writeTestRep(t *testing.T, root, cellID, ladder, task string, allowed []string, isControl bool, rep int, f repFixture) {
	t.Helper()
	dir := RepDir(root, cellID, rep)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dir, err)
	}

	if !f.blindingFailed {
		writeTestTranscript(t, dir, f.numTurns, f.durationMS, f.costUSD, f.quarryToolUses)
		if !f.maxTurnsHit && f.scored {
			writeTestScore(t, dir, f.recall, f.precision)
		}
	}

	state := RunState{
		State:          "complete",
		ConfigID:       cellID,
		Ladder:         ladder,
		Task:           task,
		Allowed:        allowed,
		IsControl:      isControl,
		Rep:            rep,
		MCPPrefix:      "mcp__quarry__",
		Scored:         f.scored,
		BlindingFailed: f.blindingFailed,
		MaxTurnsHit:    f.maxTurnsHit,
	}
	if err := WriteRunState(dir, state); err != nil {
		t.Fatalf("WriteRunState(%s) error = %v", dir, err)
	}
}

func TestComputeStats_OddSampleCount(t *testing.T) {
	stats := computeStats([]float64{5, 1, 3})
	if stats.Median != 3 {
		t.Errorf("Median = %v, want 3", stats.Median)
	}
	if stats.Min != 1 {
		t.Errorf("Min = %v, want 1", stats.Min)
	}
	if stats.Max != 5 {
		t.Errorf("Max = %v, want 5", stats.Max)
	}
	if stats.N != 3 {
		t.Errorf("N = %d, want 3", stats.N)
	}
}

func TestComputeStats_EvenSampleCount(t *testing.T) {
	stats := computeStats([]float64{4, 1, 3, 2})
	if stats.Median != 2.5 {
		t.Errorf("Median = %v, want 2.5", stats.Median)
	}
	if stats.Min != 1 {
		t.Errorf("Min = %v, want 1", stats.Min)
	}
	if stats.Max != 4 {
		t.Errorf("Max = %v, want 4", stats.Max)
	}
	if stats.N != 4 {
		t.Errorf("N = %d, want 4", stats.N)
	}
}

func TestRangesDisjoint(t *testing.T) {
	tests := []struct {
		name string
		a, b [2]float64
		want bool
	}{
		{"overlapping", [2]float64{1, 5}, [2]float64{3, 8}, false},
		{"touching_at_a_point", [2]float64{1, 5}, [2]float64{5, 8}, false},
		{"disjoint_a_below_b", [2]float64{1, 3}, [2]float64{4, 8}, true},
		{"disjoint_b_below_a", [2]float64{4, 8}, [2]float64{1, 3}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RangesDisjoint(tt.a, tt.b); got != tt.want {
				t.Errorf("RangesDisjoint(%v, %v) = %v; want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSummarize_MissingProvenanceErrorsNamingFile(t *testing.T) {
	root := t.TempDir()
	_, err := Summarize(root)
	if err == nil {
		t.Fatal("Summarize() = nil error; want an error naming the missing provenance file")
	}
	if !strings.Contains(err.Error(), ProvenanceFile) {
		t.Errorf("Summarize() error = %q; want it to name %q", err, ProvenanceFile)
	}
}

// TestSummarize_IncompleteWhenShortOfSelectedCellsTimesReps asserts the incomplete slice: a cell
// short of its provenance-declared repetition count is added to it, and a cell that meets that
// count exactly is not.
func TestSummarize_IncompleteWhenShortOfSelectedCellsTimesReps(t *testing.T) {
	root := t.TempDir()
	writeTestProvenance(t, root, []string{"a0-none", "a1-tool"}, 2)

	// a0-none: both of its two reps present and complete.
	for rep := 1; rep <= 2; rep++ {
		writeTestRep(t, root, "a0-none", "a", "01-task", nil, true, rep, repFixture{
			scored: true, numTurns: 3, durationMS: 1000, costUSD: 0.1, recall: 0.5, precision: 0.5,
		})
	}
	// a1-tool: only one of its two reps present.
	writeTestRep(t, root, "a1-tool", "a", "01-task", []string{"toc"}, false, 1, repFixture{
		scored: true, numTurns: 4, durationMS: 1200, costUSD: 0.2, quarryToolUses: 1, recall: 0.6, precision: 0.6,
	})

	summary, err := Summarize(root)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if len(summary.Incomplete) != 1 || summary.Incomplete[0] != "a1-tool" {
		t.Errorf("Incomplete = %v; want [a1-tool]", summary.Incomplete)
	}
}

// TestSummarize_ExclusionRulesAppliedIndependently builds one cell whose four repetitions each hit
// exactly one of the three exclusion dispositions plus a normal, fully scored repetition, and
// asserts each disposition's own effect: presence, counters, invalid membership, and the resulting
// cost-vs-correctness sample count split.
func TestSummarize_ExclusionRulesAppliedIndependently(t *testing.T) {
	root := t.TempDir()
	writeTestProvenance(t, root, []string{"b0-none"}, 4)

	// rep 1: blinding-failed -- contributes to neither cost nor correctness, not counted present,
	// and puts the cell in the invalid slice.
	writeTestRep(t, root, "b0-none", "b", "04-task", nil, true, 1, repFixture{
		blindingFailed: true,
	})
	// rep 2: max-turns -- contributes cost but not recall/precision, increments MaxTurnsCount only.
	writeTestRep(t, root, "b0-none", "b", "04-task", nil, true, 2, repFixture{
		maxTurnsHit: true, scored: false, numTurns: 60, durationMS: 5000, costUSD: 0.5,
	})
	// rep 3: unscored for a non-max-turns reason -- contributes cost but not recall/precision,
	// increments UnscoredCount only.
	writeTestRep(t, root, "b0-none", "b", "04-task", nil, true, 3, repFixture{
		scored: false, numTurns: 5, durationMS: 800, costUSD: 0.05,
	})
	// rep 4: a normal, fully scored repetition -- contributes both cost and correctness.
	writeTestRep(t, root, "b0-none", "b", "04-task", nil, true, 4, repFixture{
		scored: true, numTurns: 6, durationMS: 900, costUSD: 0.06, recall: 0.8, precision: 0.9,
	})

	summary, err := Summarize(root)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if len(summary.Cells) != 1 {
		t.Fatalf("Cells = %d entries; want 1", len(summary.Cells))
	}
	cell := summary.Cells[0]

	if cell.BlindingFailedCount != 1 {
		t.Errorf("BlindingFailedCount = %d; want 1", cell.BlindingFailedCount)
	}
	if cell.MaxTurnsCount != 1 {
		t.Errorf("MaxTurnsCount = %d; want 1", cell.MaxTurnsCount)
	}
	if cell.UnscoredCount != 1 {
		t.Errorf("UnscoredCount = %d; want 1", cell.UnscoredCount)
	}

	if len(summary.Invalid) != 1 || summary.Invalid[0] != "b0-none" {
		t.Errorf("Invalid = %v; want [b0-none]", summary.Invalid)
	}
	// presentNonVoid = 3 (reps 2, 3, 4 -- rep 1 excluded), short of RepsEffective = 4.
	if len(summary.Incomplete) != 1 || summary.Incomplete[0] != "b0-none" {
		t.Errorf("Incomplete = %v; want [b0-none]", summary.Incomplete)
	}

	turnsStats, ok := cell.Metrics["turns"]
	if !ok {
		t.Fatal(`Metrics["turns"] missing`)
	}
	if turnsStats.N != 3 {
		t.Errorf(`Metrics["turns"].N = %d; want 3 (reps 2, 3, 4)`, turnsStats.N)
	}

	recallStats, ok := cell.Metrics["recall"]
	if !ok {
		t.Fatal(`Metrics["recall"] missing`)
	}
	if recallStats.N != 1 {
		t.Errorf(`Metrics["recall"].N = %d; want 1 (rep 4 only)`, recallStats.N)
	}
	if turnsStats.N == recallStats.N {
		t.Errorf("cost sample count %d equals correctness sample count %d; want them to differ", turnsStats.N, recallStats.N)
	}
}

// TestSummarize_Gate1FiresWhenGrantedToolNeverUsed asserts CheckGrantedToolUsed's outcome carries
// through to the CellRecord: nil for a control cell, non-nil for a tool-granted cell whose reps
// never called the granted tool, and nil for a tool-granted cell whose reps did.
func TestSummarize_Gate1FiresWhenGrantedToolNeverUsed(t *testing.T) {
	root := t.TempDir()
	writeTestProvenance(t, root, []string{"a0-none", "a1-unused", "a2-used"}, 1)

	writeTestRep(t, root, "a0-none", "a", "01-task", nil, true, 1, repFixture{
		scored: true, numTurns: 3, durationMS: 500, costUSD: 0.05, recall: 0.4, precision: 0.4,
	})
	writeTestRep(t, root, "a1-unused", "a", "01-task", []string{"toc"}, false, 1, repFixture{
		scored: true, numTurns: 3, durationMS: 500, costUSD: 0.05, quarryToolUses: 0, recall: 0.4, precision: 0.4,
	})
	writeTestRep(t, root, "a2-used", "a", "01-task", []string{"toc"}, false, 1, repFixture{
		scored: true, numTurns: 3, durationMS: 500, costUSD: 0.05, quarryToolUses: 2, recall: 0.4, precision: 0.4,
	})

	summary, err := Summarize(root)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}

	byID := map[string]CellRecord{}
	for _, cell := range summary.Cells {
		byID[cell.ID] = cell
	}

	if byID["a0-none"].Gate1 != nil {
		t.Errorf("a0-none Gate1 = %+v; want nil (control cell)", byID["a0-none"].Gate1)
	}
	if byID["a1-unused"].Gate1 == nil {
		t.Error("a1-unused Gate1 = nil; want a finding (granted tool never used)")
	}
	if byID["a2-used"].Gate1 != nil {
		t.Errorf("a2-used Gate1 = %+v; want nil (granted tool was used)", byID["a2-used"].Gate1)
	}
}

// TestSummarize_ComparisonBuiltBetweenRungAndOwnControl asserts a rung-vs-control comparison is
// built for the metrics both sides carry, with the correct disjointness verdict.
func TestSummarize_ComparisonBuiltBetweenRungAndOwnControl(t *testing.T) {
	root := t.TempDir()
	writeTestProvenance(t, root, []string{"a0-none", "a1-tool"}, 1)

	// Control: turns = 10 (fully separated from the rung's turns below).
	writeTestRep(t, root, "a0-none", "a", "01-task", nil, true, 1, repFixture{
		scored: true, numTurns: 10, durationMS: 1000, costUSD: 0.1, recall: 0.5, precision: 0.5,
	})
	// Rung: turns = 4, disjoint from the control's 10.
	writeTestRep(t, root, "a1-tool", "a", "01-task", []string{"toc"}, false, 1, repFixture{
		scored: true, numTurns: 4, durationMS: 1000, costUSD: 0.1, quarryToolUses: 1, recall: 0.5, precision: 0.5,
	})

	summary, err := Summarize(root)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}

	var turnsComparison *Comparison
	for i := range summary.Comparisons {
		if summary.Comparisons[i].Cell == "a1-tool" && summary.Comparisons[i].Metric == "turns" {
			turnsComparison = &summary.Comparisons[i]
		}
	}
	if turnsComparison == nil {
		t.Fatal("no turns comparison found for a1-tool vs its control")
	}
	if turnsComparison.Control != "a0-none" {
		t.Errorf("Control = %q; want a0-none", turnsComparison.Control)
	}
	if !turnsComparison.Separated {
		t.Errorf("Separated = false; want true (4 vs 10 do not overlap)")
	}
}

// TestSummarize_ResultsRootMetaIsBaseNameOnly asserts the meta block never carries the
// operator-supplied path.
func TestSummarize_ResultsRootMetaIsBaseNameOnly(t *testing.T) {
	root := t.TempDir()
	writeTestProvenance(t, root, nil, 1)

	summary, err := Summarize(root)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if summary.Meta.ResultsRoot != filepath.Base(root) {
		t.Errorf("Meta.ResultsRoot = %q; want %q", summary.Meta.ResultsRoot, filepath.Base(root))
	}
	if strings.Contains(summary.Meta.ResultsRoot, string(filepath.Separator)) {
		t.Errorf("Meta.ResultsRoot = %q; want no path separator", summary.Meta.ResultsRoot)
	}
}
