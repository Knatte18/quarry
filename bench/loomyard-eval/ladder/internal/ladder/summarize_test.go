package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

/* CARD 31: per-config medians, ranges, and completeness */

func TestSummariseCell_OddRunCountTakesMiddleValue(t *testing.T) {
	runs := []map[string]any{{"num_turns": 9}, {"num_turns": 1}, {"num_turns": 5}}
	cell := SummariseCell("a1-toc-file", runs, 3)
	if got := cell.Stats["num_turns"].Median; got != 5 {
		t.Errorf("SummariseCell().Stats[num_turns].Median = %v; want 5", got)
	}
	if !cell.Complete {
		t.Error("SummariseCell().Complete = false; want true")
	}
}

func TestSummariseCell_EvenRunCountTakesMeanOfMiddleTwo(t *testing.T) {
	runs := []map[string]any{{"num_turns": 1}, {"num_turns": 3}}
	cell := SummariseCell("a1-toc-file", runs, 2)
	if got := cell.Stats["num_turns"].Median; got != 2.0 {
		t.Errorf("SummariseCell().Stats[num_turns].Median = %v; want 2.0", got)
	}
}

func TestSummariseCell_WithTwoOfThreeRunsIsIncomplete(t *testing.T) {
	runs := []map[string]any{{"num_turns": 1}, {"num_turns": 3}}
	cell := SummariseCell("a1-toc-file", runs, 3)
	if cell.Complete {
		t.Error("SummariseCell().Complete = true; want false")
	}
	if got := cell.Stats["num_turns"].N; got != 2 {
		t.Errorf("SummariseCell().Stats[num_turns].N = %d; want 2", got)
	}
}

func TestSummariseCell_MetricMissingFromOneRunReducesOnlyThatMetricsN(t *testing.T) {
	runs := []map[string]any{
		{"num_turns": 5, "recall": 1.0},
		{"num_turns": 7},
	}
	cell := SummariseCell("a1-toc-file", runs, 2)
	if got := cell.Stats["num_turns"].N; got != 2 {
		t.Errorf("SummariseCell().Stats[num_turns].N = %d; want 2", got)
	}
	if got := cell.Stats["recall"].N; got != 1 {
		t.Errorf("SummariseCell().Stats[recall].N = %d; want 1", got)
	}
}

func TestSummariseCell_DeniedToolAttemptsProvisionalMarkerPropagatesAndClears(t *testing.T) {
	provisionalRuns := []map[string]any{
		{"denied_tool_attempts": 1, "denied_tool_attempts_provisional": true},
		{"denied_tool_attempts": 0, "denied_tool_attempts_provisional": false},
	}
	provisionalCell := SummariseCell("a5-bundle", provisionalRuns, 2)
	if !provisionalCell.Stats["denied_tool_attempts"].Provisional {
		t.Error("SummariseCell().Stats[denied_tool_attempts].Provisional = false; want true when any contributing run carries it")
	}

	clearRuns := []map[string]any{
		{"denied_tool_attempts": 1, "denied_tool_attempts_provisional": false},
		{"denied_tool_attempts": 0, "denied_tool_attempts_provisional": false},
	}
	clearCell := SummariseCell("a5-bundle", clearRuns, 2)
	if clearCell.Stats["denied_tool_attempts"].Provisional {
		t.Error("SummariseCell().Stats[denied_tool_attempts].Provisional = true; want false when no contributing run carries it")
	}
}

func TestLoadRuns_IgnoresARunWhoseRunJSONIsNotComplete(t *testing.T) {
	root := t.TempDir()
	writeSummarizeRun(t, root, "a1-toc-file", 1, nil, nil, nil, "complete")
	writeSummarizeRun(t, root, "a1-toc-file", 2, nil, nil, nil, "running")
	writeSummarizeRun(t, root, "a1-toc-file", 3, nil, nil, nil, "complete")

	runs, err := LoadRuns(root, "a1-toc-file", 3)
	if err != nil {
		t.Fatalf("LoadRuns() = _, %v; want nil error", err)
	}
	if got := len(runs); got != 2 {
		t.Errorf("LoadRuns() returned %d runs; want 2", got)
	}
}

func TestLoadRuns_FlattensTokensToTopLevel(t *testing.T) {
	root := t.TempDir()
	writeSummarizeRun(t, root, "a1-toc-file", 1, nil, nil, nil, "complete")

	runs, err := LoadRuns(root, "a1-toc-file", 1)
	if err != nil {
		t.Fatalf("LoadRuns() = _, %v; want nil error", err)
	}
	if got := runs[0]["input_tokens"]; got != float64(100) {
		t.Errorf("LoadRuns()[0][input_tokens] = %v; want 100", got)
	}
	if got := runs[0]["output_tokens"]; got != float64(50) {
		t.Errorf("LoadRuns()[0][output_tokens] = %v; want 50", got)
	}
	if _, present := runs[0]["tokens"]; present {
		t.Error("LoadRuns()[0] carries a tokens key; want it flattened away")
	}
}

func TestLoadRuns_CarriesRunJSONObservations(t *testing.T) {
	root := t.TempDir()
	writeSummarizeRun(t, root, "a1-toc-file", 1, nil, nil, map[string]any{
		"worktree_dirtied":             true,
		"target_origin_quarry_mention": false,
	}, "complete")

	runs, err := LoadRuns(root, "a1-toc-file", 1)
	if err != nil {
		t.Fatalf("LoadRuns() = _, %v; want nil error", err)
	}
	if got := runs[0]["worktree_dirtied"]; got != true {
		t.Errorf("LoadRuns()[0][worktree_dirtied] = %v; want true", got)
	}
	if got := runs[0]["target_origin_quarry_mention"]; got != false {
		t.Errorf("LoadRuns()[0][target_origin_quarry_mention] = %v; want false", got)
	}
}

/* CARD 32: disjoint ranges and the comparison value type */

func TestRangesDisjoint_TrueForNonOverlapping(t *testing.T) {
	if !RangesDisjoint([2]float64{1, 5}, [2]float64{6, 9}) {
		t.Error("RangesDisjoint((1,5), (6,9)) = false; want true")
	}
}

func TestRangesDisjoint_FalseForOverlapping(t *testing.T) {
	if RangesDisjoint([2]float64{1, 5}, [2]float64{3, 9}) {
		t.Error("RangesDisjoint((1,5), (3,9)) = true; want false")
	}
}

func TestRangesDisjoint_TouchingAtOneEndpointIsNotDisjoint(t *testing.T) {
	if RangesDisjoint([2]float64{1, 5}, [2]float64{5, 9}) {
		t.Error("RangesDisjoint((1,5), (5,9)) = true; want false")
	}
}

func TestBuildComparison_OverTwoSyntheticCells(t *testing.T) {
	left := SummariseCell("a1-toc-file", []map[string]any{{"duration_ms": 10}, {"duration_ms": 12}, {"duration_ms": 14}}, 3)
	right := SummariseCell("a0-none", []map[string]any{{"duration_ms": 20}, {"duration_ms": 22}, {"duration_ms": 24}}, 3)

	comparison, ok := buildComparison("rung-vs-control", left, right, "duration_ms")
	if !ok {
		t.Fatal("buildComparison() ok = false; want true")
	}
	if comparison.Kind != "rung-vs-control" || comparison.Left != "a1-toc-file" || comparison.Right != "a0-none" {
		t.Errorf("buildComparison() = %+v; want kind=rung-vs-control left=a1-toc-file right=a0-none", comparison)
	}
	if !comparison.Separated {
		t.Error("buildComparison().Separated = false; want true for disjoint ranges")
	}

	_, ok = buildComparison("rung-vs-control", left, right, "recall")
	if ok {
		t.Error("buildComparison() ok = true for a metric neither cell carries; want false")
	}
}

/* CARD 33: the three comparison families */

func cellFromMetric(configID, metric string, values []int) Cell {
	var runs []map[string]any
	for _, v := range values {
		runs = append(runs, map[string]any{metric: v})
	}
	return SummariseCell(configID, runs, len(values))
}

func TestCompareRungToControl_ExcludesGrepMetricsButCompareRungsDoesNot(t *testing.T) {
	l := mustLoadLadder(t)

	cellFor := func(configID string) Cell {
		var runs []map[string]any
		for i, v := range []int{10, 12, 14} {
			runs = append(runs, map[string]any{"duration_ms": v, "bash_grep_count": []int{1, 2, 3}[i]})
		}
		return SummariseCell(configID, runs, 3)
	}
	a1, err := ConfigByID(l, "a1-toc-file")
	if err != nil {
		t.Fatalf("ConfigByID(a1-toc-file) = _, %v; want nil error", err)
	}
	cells := map[string]Cell{
		"a1-toc-file": cellFor("a1-toc-file"),
		"a2-toc-dir":  cellFor("a2-toc-dir"),
		"a0-none":     cellFor("a0-none"),
	}

	controlComparisons, err := CompareRungToControl(l, cells, a1)
	if err != nil {
		t.Fatalf("CompareRungToControl() = _, %v; want nil error", err)
	}
	for _, comparison := range controlComparisons {
		if isGrepMetric(comparison.Metric) {
			t.Errorf("CompareRungToControl() included grep metric %q; want none", comparison.Metric)
		}
	}

	rungComparisons, err := CompareRungs(l, cells, "a1-toc-file", "a2-toc-dir")
	if err != nil {
		t.Fatalf("CompareRungs() = _, %v; want nil error", err)
	}
	foundGrepMetric := false
	for _, comparison := range rungComparisons {
		if comparison.Metric == "bash_grep_count" {
			foundGrepMetric = true
		}
	}
	if !foundGrepMetric {
		t.Error("CompareRungs() did not include bash_grep_count; want it eligible for rung-vs-rung")
	}
}

func TestCompareRungs_RejectsACrossLadderPair(t *testing.T) {
	l := mustLoadLadder(t)
	cells := map[string]Cell{
		"a1-toc-file": cellFromMetric("a1-toc-file", "duration_ms", []int{1, 2, 3}),
		"b1-symbol":   cellFromMetric("b1-symbol", "duration_ms", []int{1, 2, 3}),
	}

	_, err := CompareRungs(l, cells, "a1-toc-file", "b1-symbol")
	if err == nil {
		t.Fatal("CompareRungs() = _, nil; want a *SummarizeError")
	}
	if _, ok := err.(*SummarizeError); !ok {
		t.Errorf("CompareRungs() error type = %T; want *SummarizeError", err)
	}
}

func TestCompareRungs_IncompleteCellYieldsNoComparisons(t *testing.T) {
	l := mustLoadLadder(t)
	cells := map[string]Cell{
		"a1-toc-file": SummariseCell("a1-toc-file", []map[string]any{{"duration_ms": 1}}, 3),
		"a2-toc-dir":  cellFromMetric("a2-toc-dir", "duration_ms", []int{1, 2, 3}),
	}

	comparisons, err := CompareRungs(l, cells, "a1-toc-file", "a2-toc-dir")
	if err != nil {
		t.Fatalf("CompareRungs() = _, %v; want nil error", err)
	}
	if len(comparisons) != 0 {
		t.Errorf("CompareRungs() returned %d comparisons; want 0 when a cell is incomplete", len(comparisons))
	}
}

func TestCompareWarmCold_ResolvesWarmSideThroughWarmCounterpartField(t *testing.T) {
	l := mustLoadLadder(t)
	cells := map[string]Cell{
		"a5-bundle-cold": cellFromMetric("a5-bundle-cold", "duration_ms", []int{1, 2, 3}),
		"a5-bundle":      cellFromMetric("a5-bundle", "duration_ms", []int{100, 110, 120}),
	}

	comparisons, err := CompareWarmCold(l, cells, "confirmed-cold")
	if err != nil {
		t.Fatalf("CompareWarmCold() = _, %v; want nil error", err)
	}
	if len(comparisons) == 0 {
		t.Fatal("CompareWarmCold() returned 0 comparisons; want at least one")
	}
	if comparisons[0].Kind != "warm-vs-cold" {
		t.Errorf("CompareWarmCold()[0].Kind = %q; want warm-vs-cold", comparisons[0].Kind)
	}
	sides := map[string]bool{comparisons[0].Left: true, comparisons[0].Right: true}
	if !sides["a5-bundle-cold"] || !sides["a5-bundle"] {
		t.Errorf("CompareWarmCold()[0] sides = %+v; want a5-bundle-cold and a5-bundle", comparisons[0])
	}
}

func TestCompareWarmCold_EmitsNothingWhenEveryColdRunLacksDaemonSignal(t *testing.T) {
	l := mustLoadLadder(t)
	var coldRuns []map[string]any
	for _, v := range []int{1, 2, 3} {
		coldRuns = append(coldRuns, map[string]any{"duration_ms": v, "cold_no_daemon_backed_call": true})
	}
	cells := map[string]Cell{
		"a5-bundle-cold": SummariseCell("a5-bundle-cold", coldRuns, 3),
		"a5-bundle":      cellFromMetric("a5-bundle", "duration_ms", []int{10, 11, 12}),
	}

	comparisons, err := CompareWarmCold(l, cells, "confirmed-cold")
	if err != nil {
		t.Fatalf("CompareWarmCold() = _, %v; want nil error", err)
	}
	if len(comparisons) != 0 {
		t.Errorf("CompareWarmCold() returned %d comparisons; want 0 when every cold run lacks a daemon signal", len(comparisons))
	}
}

func TestCompareWarmCold_EmitsNothingForNotRunOrPartialDisposition(t *testing.T) {
	l := mustLoadLadder(t)
	cells := map[string]Cell{
		"a5-bundle-cold": cellFromMetric("a5-bundle-cold", "duration_ms", []int{1, 2, 3}),
		"a5-bundle":      cellFromMetric("a5-bundle", "duration_ms", []int{10, 11, 12}),
	}

	for _, disposition := range []string{"not-run", "partial"} {
		comparisons, err := CompareWarmCold(l, cells, disposition)
		if err != nil {
			t.Fatalf("CompareWarmCold(%q) = _, %v; want nil error", disposition, err)
		}
		if len(comparisons) != 0 {
			t.Errorf("CompareWarmCold(%q) returned %d comparisons; want 0", disposition, len(comparisons))
		}
	}
}

func TestCompareWarmCold_EmitsNothingWhenLadderDeclaresNoColdConfig(t *testing.T) {
	l := &Ladder{Configs: []LadderConfig{
		{ID: "c0-none", Ladder: "b", Task: "t"},
		{ID: "c1-impact", Ladder: "b", Task: "t", Allowed: []string{"impact"}},
	}}
	cells := map[string]Cell{
		"c0-none":   cellFromMetric("c0-none", "duration_ms", []int{1, 2, 3}),
		"c1-impact": cellFromMetric("c1-impact", "duration_ms", []int{10, 11, 12}),
	}

	comparisons, err := CompareWarmCold(l, cells, "unknown")
	if err != nil {
		t.Fatalf("CompareWarmCold() = _, %v; want nil error", err)
	}
	if len(comparisons) != 0 {
		t.Errorf("CompareWarmCold() returned %d comparisons; want 0 when the ladder declares no cold config", len(comparisons))
	}
}

/* CARD 34: summary building, writing, and the incomplete exit code */

// writeFullMatrix writes ladder.Reps complete runs for every config in l, plus cold_cell.json and
// probe.json, so BuildSummary sees a fully complete matrix. Individual tests perturb specific runs
// afterwards.
func writeFullMatrix(t *testing.T, resultsRoot string, l *Ladder, opts fullMatrixOptions) {
	t.Helper()
	for _, config := range l.Configs {
		task := l.Tasks[config.Task]
		var scoreExtra map[string]any
		if task.Schema == "impact" {
			scoreExtra = map[string]any{"decoy_admitted": false, "lookalikes_matched": 0}
		} else {
			scoreExtra = map[string]any{"summary_matches": true}
		}

		for n := 1; n <= l.Reps; n++ {
			score := map[string]any{}
			for k, v := range scoreExtra {
				score[k] = v
			}
			if config.ID == opts.decoyAdmittedConfig && n == 1 {
				score["decoy_admitted"] = true
			}

			runExtra := map[string]any{}
			if config.ID == opts.worktreeDirtiedConfig && n == 1 {
				runExtra["worktree_dirtied"] = true
			}
			if config.ID == opts.targetOriginConfig && n == 1 {
				runExtra["target_origin_quarry_mention"] = true
			}
			if config.Cold {
				if _, present := runExtra["cold_no_daemon_backed_call"]; !present {
					runExtra["cold_no_daemon_backed_call"] = false
				}
			}

			writeSummarizeRun(t, resultsRoot, config.ID, n, nil, score, runExtra, "complete")
		}
	}

	if opts.coldCellPayload != nil {
		writeJSONFile(t, filepath.Join(resultsRoot, "cold_cell.json"), opts.coldCellPayload)
	}
	if opts.probePayload != nil {
		writeJSONFile(t, filepath.Join(resultsRoot, "probe.json"), opts.probePayload)
	}
}

// fullMatrixOptions perturbs writeFullMatrix's synthetic tree. A nil coldCellPayload/probePayload
// simulates the file genuinely being absent.
type fullMatrixOptions struct {
	decoyAdmittedConfig   string
	worktreeDirtiedConfig string
	targetOriginConfig    string
	coldCellPayload       map[string]any
	probePayload          map[string]any
}

func defaultFullMatrixOptions() fullMatrixOptions {
	return fullMatrixOptions{
		coldCellPayload: map[string]any{"disposition": "confirmed-cold", "confirmed_cold_reps": 3},
		probePayload:    map[string]any{"denied_tools_advertised": true},
	}
}

func TestBuildSummary_MetaRecordsPinnedScorerAndReps(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	writeFullMatrix(t, root, l, defaultFullMatrixOptions())

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	if summary.Meta.Scorer.Model != l.Scorer.Model || summary.Meta.Scorer.Effort != l.Scorer.Effort {
		t.Errorf("BuildSummary().Meta.Scorer = %+v; want %+v", summary.Meta.Scorer, l.Scorer)
	}
	if summary.Meta.Reps != l.Reps {
		t.Errorf("BuildSummary().Meta.Reps = %d; want %d", summary.Meta.Reps, l.Reps)
	}
}

func TestBuildSummary_EveryConfigIDAppearsInCells(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	writeFullMatrix(t, root, l, defaultFullMatrixOptions())

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	if got := len(summary.Cells); got != len(l.Configs) {
		t.Errorf("BuildSummary() produced %d cells; want %d", got, len(l.Configs))
	}
	for _, config := range l.Configs {
		if _, ok := summary.Cells[config.ID]; !ok {
			t.Errorf("BuildSummary().Cells missing config id %q", config.ID)
		}
	}
}

func TestBuildSummary_IncompleteListsExactlyTheShortCells(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	writeFullMatrix(t, root, l, defaultFullMatrixOptions())
	if err := os.Remove(filepath.Join(RunDirPath(root, "a3-toc-pair", 2), "run.json")); err != nil {
		t.Fatalf("os.Remove(run.json) = %v; want nil error", err)
	}

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	if len(summary.Incomplete) != 1 || summary.Incomplete[0] != "a3-toc-pair" {
		t.Errorf("BuildSummary().Incomplete = %v; want [a3-toc-pair]", summary.Incomplete)
	}
}

func TestBuildSummary_ComparisonsContainsAllThreeKinds(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	writeFullMatrix(t, root, l, defaultFullMatrixOptions())

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	kinds := map[string]bool{}
	for _, comparison := range summary.Comparisons {
		kinds[comparison.Kind] = true
	}
	for _, want := range []string{"rung-vs-control", "rung-vs-rung", "warm-vs-cold"} {
		if !kinds[want] {
			t.Errorf("BuildSummary().Comparisons missing kind %q; got kinds %v", want, kinds)
		}
	}
}

func TestBuildSummary_AggregatesWorktreeDirtiedDecoyAdmittedAndDaemonBacked(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	opts := defaultFullMatrixOptions()
	opts.decoyAdmittedConfig = "b6-assert-no-callers"
	opts.worktreeDirtiedConfig = "a1-toc-file"
	opts.targetOriginConfig = "a0-none"
	writeFullMatrix(t, root, l, opts)

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	if got := summary.Cells["b6-assert-no-callers"].DecoyAdmittedCount; got != 1 {
		t.Errorf("Cells[b6-assert-no-callers].DecoyAdmittedCount = %d; want 1", got)
	}
	if got := summary.Cells["a1-toc-file"].WorktreeDirtiedCount; got != 1 {
		t.Errorf("Cells[a1-toc-file].WorktreeDirtiedCount = %d; want 1", got)
	}
	if got := summary.Cells["a0-none"].TargetOriginQuarryMentionCount; got != 1 {
		t.Errorf("Cells[a0-none].TargetOriginQuarryMentionCount = %d; want 1", got)
	}
	if got := summary.Cells["a5-bundle-cold"].DaemonBackedRuns; got != l.Reps {
		t.Errorf("Cells[a5-bundle-cold].DaemonBackedRuns = %d; want %d", got, l.Reps)
	}
}

func TestBuildSummary_SummaryMatchesCarriedThroughVerbatim(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	writeFullMatrix(t, root, l, defaultFullMatrixOptions())

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	matches := summary.Cells["a1-toc-file"].SummaryMatches
	if len(matches) != 3 {
		t.Fatalf("Cells[a1-toc-file].SummaryMatches = %v; want 3 entries", matches)
	}
	for _, m := range matches {
		if m != true {
			t.Errorf("Cells[a1-toc-file].SummaryMatches entry = %v; want true", m)
		}
	}
}

func TestBuildSummary_NotRunColdCellAbsentFromIncompleteAndExitsZero(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	opts := defaultFullMatrixOptions()
	opts.coldCellPayload = map[string]any{"disposition": "not-run", "confirmed_cold_reps": 0}
	writeFullMatrix(t, root, l, opts)
	if err := os.RemoveAll(filepath.Dir(RunDirPath(root, "a5-bundle-cold", 1))); err != nil {
		t.Fatalf("os.RemoveAll(a5-bundle-cold) = %v; want nil error", err)
	}

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	for _, id := range summary.Incomplete {
		if id == "a5-bundle-cold" {
			t.Error("BuildSummary().Incomplete contains a5-bundle-cold; want it absent for a not-run cold cell")
		}
	}
	if SummaryExitCode(summary) != 0 {
		t.Errorf("SummaryExitCode() = %d; want 0", SummaryExitCode(summary))
	}
}

func TestBuildSummary_PartialColdCellAbsentFromIncompleteAndNoWarmColdComparison(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	opts := defaultFullMatrixOptions()
	opts.coldCellPayload = map[string]any{"disposition": "partial", "confirmed_cold_reps": 1}
	writeFullMatrix(t, root, l, opts)
	if err := os.RemoveAll(filepath.Dir(RunDirPath(root, "a5-bundle-cold", 1))); err != nil {
		t.Fatalf("os.RemoveAll(a5-bundle-cold) = %v; want nil error", err)
	}

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	for _, id := range summary.Incomplete {
		if id == "a5-bundle-cold" {
			t.Error("BuildSummary().Incomplete contains a5-bundle-cold; want it absent for a partial cold cell")
		}
	}
	for _, comparison := range summary.Comparisons {
		if comparison.Kind == "warm-vs-cold" {
			t.Error("BuildSummary().Comparisons contains a warm-vs-cold comparison; want none for a partial cold cell")
		}
	}
	if SummaryExitCode(summary) != 0 {
		t.Errorf("SummaryExitCode() = %d; want 0", SummaryExitCode(summary))
	}
}

func TestBuildSummary_ShortColdCellForOtherReasonIsIncomplete(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	writeFullMatrix(t, root, l, defaultFullMatrixOptions())
	if err := os.Remove(filepath.Join(RunDirPath(root, "a5-bundle-cold", 2), "run.json")); err != nil {
		t.Fatalf("os.Remove(run.json) = %v; want nil error", err)
	}

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	found := false
	for _, id := range summary.Incomplete {
		if id == "a5-bundle-cold" {
			found = true
		}
	}
	if !found {
		t.Error("BuildSummary().Incomplete does not contain a5-bundle-cold; want it present when short for another reason")
	}
	if SummaryExitCode(summary) != 1 {
		t.Errorf("SummaryExitCode() = %d; want 1", SummaryExitCode(summary))
	}
}

func TestBuildSummary_AbsentColdCellJSONAndProbeJSONDoNotError(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	opts := defaultFullMatrixOptions()
	opts.coldCellPayload = nil
	opts.probePayload = nil
	writeFullMatrix(t, root, l, opts)
	if err := os.RemoveAll(filepath.Dir(RunDirPath(root, "a5-bundle-cold", 1))); err != nil {
		t.Fatalf("os.RemoveAll(a5-bundle-cold) = %v; want nil error", err)
	}

	summary, err := BuildSummary(l, root)
	if err != nil {
		t.Fatalf("BuildSummary() = _, %v; want nil error", err)
	}
	if summary.Meta.ColdDisposition != "unknown" {
		t.Errorf("BuildSummary().Meta.ColdDisposition = %q; want unknown", summary.Meta.ColdDisposition)
	}
	found := false
	for _, id := range summary.Incomplete {
		if id == "a5-bundle-cold" {
			found = true
		}
	}
	if !found {
		t.Error("BuildSummary().Incomplete does not contain a5-bundle-cold; want it present when its runs are absent")
	}
}

func TestWriteSummary_WritesSortedKeysJSONWithTrailingNewlineAndRoundTrips(t *testing.T) {
	l := mustLoadLadder(t)
	root := t.TempDir()
	writeFullMatrix(t, root, l, defaultFullMatrixOptions())

	built, err := WriteSummary(l, root)
	if err != nil {
		t.Fatalf("WriteSummary() = _, %v; want nil error", err)
	}

	content, err := os.ReadFile(filepath.Join(root, "summary.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(summary.json) = _, %v; want nil error", err)
	}
	if len(content) == 0 || content[len(content)-1] != '\n' {
		t.Error("summary.json does not end with a trailing newline")
	}

	var roundTripped Summary
	if err := json.Unmarshal(content, &roundTripped); err != nil {
		t.Fatalf("json.Unmarshal(summary.json) = %v; want nil error", err)
	}
	if len(roundTripped.Cells) != len(built.Cells) {
		t.Errorf("round-tripped summary has %d cells; want %d", len(roundTripped.Cells), len(built.Cells))
	}
	if len(roundTripped.Incomplete) != len(built.Incomplete) {
		t.Errorf("round-tripped summary has %d incomplete entries; want %d", len(roundTripped.Incomplete), len(built.Incomplete))
	}
}

// writeSummarizeRun writes a synthetic run directory at
// <resultsRoot>/raw/<configID>/<n>/ carrying usage.json, score.json, and run.json, with reasonable
// defaults for every field summarize.go reads. usageExtra and scoreExtra override the defaults; runExtra
// adds run.json fields beyond config_id/n/state.
func writeSummarizeRun(t *testing.T, resultsRoot, configID string, n int, usageExtra, scoreExtra, runExtra map[string]any, state string) string {
	t.Helper()
	dir := RunDirPath(resultsRoot, configID, n)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%s) = %v; want nil error", dir, err)
	}

	usage := map[string]any{
		"duration_ms":                      1000,
		"num_turns":                        5,
		"tool_uses":                        3,
		"quarry_tool_uses":                 2,
		"tokens":                           map[string]any{"input_tokens": 100, "output_tokens": 50, "cache_read_input_tokens": 10, "cache_creation_input_tokens": 5},
		"bash_grep_count":                  0,
		"grep_tool_count":                  0,
		"grep_fallback_total":              0,
		"denied_tool_attempts":             0,
		"denied_tool_attempts_provisional": true,
	}
	for k, v := range usageExtra {
		usage[k] = v
	}

	score := map[string]any{"recall": 0.5, "precision": 0.5}
	for k, v := range scoreExtra {
		score[k] = v
	}

	run := map[string]any{"config_id": configID, "n": n, "state": state}
	for k, v := range runExtra {
		run[k] = v
	}

	writeJSONFile(t, filepath.Join(dir, "usage.json"), usage)
	writeJSONFile(t, filepath.Join(dir, "score.json"), score)
	writeJSONFile(t, filepath.Join(dir, "run.json"), run)

	return dir
}
