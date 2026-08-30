package ladder

import (
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
