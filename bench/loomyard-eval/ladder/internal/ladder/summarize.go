// summarize.go ports scripts/summarize.py: per-config medians and full ranges over every metric
// usage.go and score.go produce, and the disjoint-range separation rule applied independently to the
// three comparison types the discussion defines -- rung-vs-control, rung-vs-rung, and warm-vs-cold.
//
// Completeness is never re-derived here: LoadRuns calls IsComplete (runstate.go), so there is exactly
// one definition of a complete run across the suite.

package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Metrics is the ordered set of names SummariseCell reduces to a median/min/max/n record, drawn from
// every run's flattened usage.json and score.json fields. Ported from the Python's METRICS, with
// wall_clock_ms and cost_usd dropped: neither has a source under agent dispatch (see usage.go's Usage
// struct, whose field set already excludes both), and a metric column that is always null reads as a
// measurement that failed rather than one that was never taken. decoy_admitted and summary_matches are
// deliberately excluded, same as the Python: they are qualitative or boolean and get their own per-cell
// treatment in BuildSummary instead of a meaningless median.
var Metrics = []string{
	"duration_ms",
	"num_turns",
	"tool_uses",
	"quarry_tool_uses",
	"input_tokens",
	"output_tokens",
	"cache_read_input_tokens",
	"cache_creation_input_tokens",
	"bash_grep_count",
	"grep_tool_count",
	"grep_fallback_total",
	"denied_tool_attempts",
	"recall",
	"precision",
	"lookalikes_matched",
}

// GrepMetrics is the subset of Metrics excluded from a rung-vs-control comparison: the control's
// preamble differs in steering as well as tools, so a grep-usage gap between it and a rung cannot be
// attributed to the tool exposure alone.
var GrepMetrics = []string{"bash_grep_count", "grep_tool_count", "grep_fallback_total"}

// runJSONObservations is the fixed set of run.json observations LoadRuns lifts onto each per-run
// mapping, recorded by the harness as non-fatal gate findings rather than summarised metrics.
var runJSONObservations = []string{"worktree_dirtied", "target_origin_quarry_mention", "cold_no_daemon_backed_call"}

// SummarizeError is raised when a comparison is requested across configs that cannot be compared --
// e.g. CompareRungs given two configs from different ladders.
type SummarizeError struct {
	// Message describes the offending comparison request.
	Message string
}

// Error implements the error interface.
func (e *SummarizeError) Error() string {
	return e.Message
}

/* PER-CONFIG MEDIANS, RANGES, AND COMPLETENESS */

// MetricStats is one metric's summarised median/min/max/n record over the runs that carried it.
type MetricStats struct {
	// Median is the metric's true median across the contributing runs.
	Median float64 `json:"median"`
	// Min is the metric's minimum value across the contributing runs.
	Min float64 `json:"min"`
	// Max is the metric's maximum value across the contributing runs.
	Max float64 `json:"max"`
	// N is the count of runs that carried this metric, which may be fewer than the cell's total run
	// count when a metric is absent from some runs.
	N int `json:"n"`
	// Provisional is set only on the denied_tool_attempts metric's own stats record: true when any run
	// that contributed a value to it also carried usage.go's Usage.DeniedToolAttemptsProvisional. Every
	// other metric's record leaves this false and, via omitempty, out of the written JSON entirely.
	Provisional bool `json:"provisional,omitempty"`
}

// Cell is one config's summarised results.
type Cell struct {
	// ConfigID is the LadderConfig id this cell belongs to.
	ConfigID string
	// Runs is the list of flattened per-run mappings LoadRuns read.
	Runs []map[string]any
	// Complete is true only when len(Runs) equals the config's reps.
	Complete bool
	// Stats maps a Metrics name to its summarised record, present only for metrics carried by at least
	// one run.
	Stats map[string]MetricStats
}

// readJSONObjectFile reads and parses path as a JSON object.
func readJSONObjectFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// LoadRuns reads every complete run of configID under <resultsRoot>/raw/<configID>/<n>/ for n in
// 1..reps.
//
// A run directory is skipped when IsComplete returns false for it -- this is the suite's single
// definition of completeness, never re-derived here. For each complete run, usage.json, score.json, and
// run.json are merged into one flat mapping: usage.json's tokens.<class> fields are lifted to top-level
// <class> keys, every other usage.json and score.json field is copied unchanged, and run.json's
// worktree_dirtied, target_origin_quarry_mention, and cold_no_daemon_backed_call observations are
// carried across when present.
//
// The observation lift reads the top-level keys RunJSONPayload (runstate.go) writes onto run.json. In
// the Python this ports, run.json never carried those keys, so its own lift -- and every metric
// downstream of it -- never fired; RunJSONPayload's payload is a deliberate repair of that broken chain
// (see its own doc comment), and this port's run.json actually carries the keys, so the lift here is
// live. A later reader must not "restore" the Python's silence by dropping the keys RunJSONPayload
// writes.
func LoadRuns(resultsRoot, configID string, reps int) ([]map[string]any, error) {
	var runs []map[string]any
	for n := 1; n <= reps; n++ {
		dir := RunDirPath(resultsRoot, configID, n)
		if !IsComplete(dir) {
			continue
		}

		usage, err := readJSONObjectFile(filepath.Join(dir, "usage.json"))
		if err != nil {
			return nil, fmt.Errorf("ladder: LoadRuns: %w", err)
		}
		score, err := readJSONObjectFile(filepath.Join(dir, "score.json"))
		if err != nil {
			return nil, fmt.Errorf("ladder: LoadRuns: %w", err)
		}
		runRecord, err := readJSONObjectFile(filepath.Join(dir, "run.json"))
		if err != nil {
			return nil, fmt.Errorf("ladder: LoadRuns: %w", err)
		}

		merged := make(map[string]any, len(usage)+len(score))
		for key, value := range usage {
			if key == "tokens" {
				if tokens, ok := value.(map[string]any); ok {
					for tokenKey, tokenValue := range tokens {
						merged[tokenKey] = tokenValue
					}
				}
				continue
			}
			merged[key] = value
		}
		for key, value := range score {
			merged[key] = value
		}
		for _, observation := range runJSONObservations {
			if value, ok := runRecord[observation]; ok {
				merged[observation] = value
			}
		}

		runs = append(runs, merged)
	}
	return runs, nil
}

// median returns the true median of values: the middle value at odd length, the mean of the two middle
// values at even length.
func median(values []float64) float64 {
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	count := len(ordered)
	midpoint := count / 2
	if count%2 == 1 {
		return ordered[midpoint]
	}
	return (ordered[midpoint-1] + ordered[midpoint]) / 2
}

// numericValue coerces a JSON-decoded or test-literal value to float64, reporting whether v was
// numeric. Values loaded through LoadRuns arrive as float64 (encoding/json's own number
// representation); values built directly by a Go test may arrive as an int literal instead, so both are
// accepted.
func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// minOf returns the smallest value in values, which must be non-empty.
func minOf(values []float64) float64 {
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// maxOf returns the largest value in values, which must be non-empty.
func maxOf(values []float64) float64 {
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// SummariseCell builds the Cell for configID from its already-loaded runs.
//
// A metric absent from a run -- a schema-only field like recall on a run whose task carries no such
// score field -- is skipped for that run, so its stats record's N reflects only the runs that carried
// it rather than the full run count.
//
// The denied_tool_attempts stats record alone also carries a Provisional marker: true when any run that
// contributed a value to it itself carried denied_tool_attempts_provisional (see usage.go's
// Usage.DeniedToolAttemptsProvisional), clearing to false the moment none of the contributing runs still
// carry it.
func SummariseCell(configID string, runs []map[string]any, reps int) Cell {
	stats := make(map[string]MetricStats)
	for _, metric := range Metrics {
		var values []float64
		provisional := false
		for _, run := range runs {
			raw, ok := run[metric]
			if !ok || raw == nil {
				continue
			}
			value, ok := numericValue(raw)
			if !ok {
				continue
			}
			values = append(values, value)
			if metric == "denied_tool_attempts" {
				if p, ok := run["denied_tool_attempts_provisional"].(bool); ok && p {
					provisional = true
				}
			}
		}
		if len(values) == 0 {
			continue
		}
		record := MetricStats{
			Median: median(values),
			Min:    minOf(values),
			Max:    maxOf(values),
			N:      len(values),
		}
		if metric == "denied_tool_attempts" {
			record.Provisional = provisional
		}
		stats[metric] = record
	}
	return Cell{ConfigID: configID, Runs: runs, Complete: len(runs) == reps, Stats: stats}
}

/* THE THREE DISJOINT-RANGE COMPARISON TYPES */

// Comparison is one structured, mechanically checkable comparison between two cells over one metric.
type Comparison struct {
	// Kind is "rung-vs-control", "rung-vs-rung", or "warm-vs-cold".
	Kind string `json:"kind"`
	// Left is the left-hand config id.
	Left string `json:"left"`
	// Right is the right-hand config id.
	Right string `json:"right"`
	// Metric is the Metrics name compared.
	Metric string `json:"metric"`
	// LeftMedian is the left-hand side's median.
	LeftMedian float64 `json:"left_median"`
	// RightMedian is the right-hand side's median.
	RightMedian float64 `json:"right_median"`
	// LeftRange is the left-hand side's (min, max) pair.
	LeftRange [2]float64 `json:"left_range"`
	// RightRange is the right-hand side's (min, max) pair.
	RightRange [2]float64 `json:"right_range"`
	// Separated is true only when LeftRange and RightRange do not overlap at all.
	Separated bool `json:"separated"`
}

// RangesDisjoint reports whether the two closed ranges a and b -- each a (min, max) pair -- do not
// overlap at all. Ranges that touch at a single value -- one's max equals the other's min -- are
// treated as NOT disjoint.
func RangesDisjoint(a, b [2]float64) bool {
	return a[1] < b[0] || b[1] < a[0]
}

// buildComparison returns one Comparison for metric between leftCell and rightCell, and whether one
// could be built at all -- false when either side carries no stats for that metric.
func buildComparison(kind string, leftCell, rightCell Cell, metric string) (Comparison, bool) {
	leftStats, ok := leftCell.Stats[metric]
	if !ok {
		return Comparison{}, false
	}
	rightStats, ok := rightCell.Stats[metric]
	if !ok {
		return Comparison{}, false
	}

	leftRange := [2]float64{leftStats.Min, leftStats.Max}
	rightRange := [2]float64{rightStats.Min, rightStats.Max}
	return Comparison{
		Kind:        kind,
		Left:        leftCell.ConfigID,
		Right:       rightCell.ConfigID,
		Metric:      metric,
		LeftMedian:  leftStats.Median,
		RightMedian: rightStats.Median,
		LeftRange:   leftRange,
		RightRange:  rightRange,
		Separated:   RangesDisjoint(leftRange, rightRange),
	}, true
}

// isGrepMetric reports whether metric is one of GrepMetrics.
func isGrepMetric(metric string) bool {
	for _, grepMetric := range GrepMetrics {
		if grepMetric == metric {
			return true
		}
	}
	return false
}

// CompareRungToControl compares config against its own ladder's control, resolved through ControlFor
// rather than by parsing config.ID. Emits a comparison for every Metrics entry except GrepMetrics, since
// the control's preamble differs in steering as well as in tools, so a grep-usage gap between it and a
// rung cannot be attributed to the tool exposure alone. This exclusion is CompareRungToControl's own:
// CompareRungs keeps every grep metric eligible for a same-ladder rung pair, whose preambles differ only
// in the tool list -- the asymmetry is encoded here explicitly, one comparison family at a time, rather
// than as a filter the two builders would otherwise have to share.
//
// Emits nothing when either cell is incomplete.
func CompareRungToControl(l *Ladder, cells map[string]Cell, config LadderConfig) ([]Comparison, error) {
	control, err := ControlFor(l, config)
	if err != nil {
		return nil, err
	}
	rungCell := cells[config.ID]
	controlCell := cells[control.ID]
	if !rungCell.Complete || !controlCell.Complete {
		return nil, nil
	}

	var comparisons []Comparison
	for _, metric := range Metrics {
		if isGrepMetric(metric) {
			continue
		}
		if comparison, ok := buildComparison("rung-vs-control", rungCell, controlCell, metric); ok {
			comparisons = append(comparisons, comparison)
		}
	}
	return comparisons, nil
}

// CompareRungs compares two configs in the same ladder, both expected to carry a non-empty allowed set.
// All Metrics are eligible, including the grep metrics, since these preambles are identical except for
// the tool list -- contrast CompareRungToControl's own grep exclusion, made for the opposite reason.
//
// Returns a *SummarizeError when leftID and rightID are not in the same ladder. Emits nothing when
// either cell is incomplete.
func CompareRungs(l *Ladder, cells map[string]Cell, leftID, rightID string) ([]Comparison, error) {
	leftConfig, err := ConfigByID(l, leftID)
	if err != nil {
		return nil, err
	}
	rightConfig, err := ConfigByID(l, rightID)
	if err != nil {
		return nil, err
	}
	if leftConfig.Ladder != rightConfig.Ladder {
		return nil, &SummarizeError{Message: fmt.Sprintf("CompareRungs: %q and %q are not in the same ladder", leftID, rightID)}
	}

	leftCell := cells[leftID]
	rightCell := cells[rightID]
	if !leftCell.Complete || !rightCell.Complete {
		return nil, nil
	}

	var comparisons []Comparison
	for _, metric := range Metrics {
		if comparison, ok := buildComparison("rung-vs-rung", leftCell, rightCell, metric); ok {
			comparisons = append(comparisons, comparison)
		}
	}
	return comparisons, nil
}

// allColdRunsLackDaemonBackedCall reports whether every run in runs carries cold_no_daemon_backed_call
// as true -- runs must be non-empty; callers check that themselves, since an empty cold cell is a
// different (incomplete-cell) case CompareWarmCold already excludes before reaching this check.
func allColdRunsLackDaemonBackedCall(runs []map[string]any) bool {
	for _, run := range runs {
		lacksDaemon, _ := run["cold_no_daemon_backed_call"].(bool)
		if !lacksDaemon {
			return false
		}
	}
	return true
}

// CompareWarmCold compares the ladder's cold cell against the warm cell named by its own
// WarmCounterpart field, resolved through WarmCounterpartFor.
//
// Emits nothing when coldDisposition is "not-run" or "partial" -- a disjoint-range claim at n = reps
// cannot be made from fewer than reps cold runs -- when either cell is incomplete, when every cold run
// recorded cold_no_daemon_backed_call (none of them started a daemon, so there is no warmth contrast to
// draw), or when the ladder file declares no cold config at all -- a distilled companion matrix (e.g.
// one scoped to a single new task) legitimately has no cold cell to compare against, the same as one
// whose cold cell simply hasn't run yet.
func CompareWarmCold(l *Ladder, cells map[string]Cell, coldDisposition string) ([]Comparison, error) {
	if coldDisposition == "not-run" || coldDisposition == "partial" {
		return nil, nil
	}

	var coldConfig LadderConfig
	found := false
	for _, config := range l.Configs {
		if config.Cold {
			coldConfig = config
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}
	warmConfig, err := WarmCounterpartFor(l, coldConfig)
	if err != nil {
		return nil, err
	}

	coldCell := cells[coldConfig.ID]
	warmCell := cells[warmConfig.ID]
	if !coldCell.Complete || !warmCell.Complete {
		return nil, nil
	}
	if len(coldCell.Runs) > 0 && allColdRunsLackDaemonBackedCall(coldCell.Runs) {
		return nil, nil
	}

	var comparisons []Comparison
	for _, metric := range Metrics {
		if comparison, ok := buildComparison("warm-vs-cold", coldCell, warmCell, metric); ok {
			comparisons = append(comparisons, comparison)
		}
	}
	return comparisons, nil
}

/* THE TRACKED summary.json */

// readJSONOrDefault returns the parsed JSON object at path, or def when path does not exist. Neither
// missing file this package reads with it (cold_cell.json, probe.json) is treated as an error --
// summarising a partial matrix is a legitimate intermediate state.
func readJSONOrDefault(path string, def map[string]any) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return def, nil
		}
		return nil, fmt.Errorf("ladder: readJSONOrDefault: read %s: %w", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("ladder: readJSONOrDefault: parse %s: %w", path, err)
	}
	return m, nil
}

// SummaryScorerMeta carries the pinned scoring model and effort into SummaryMeta.
type SummaryScorerMeta struct {
	// Model is the pinned scoring client model id.
	Model string `json:"model"`
	// Effort is the pinned scoring reasoning-effort level.
	Effort string `json:"effort"`
}

// SummaryMeta is summary.json's _meta block: the pinned run model, the pinned scorer mapping, reps, the
// results-root date segment, the number of configs, and the cold cell's disposition (from
// cold_cell.json, or "unknown" when absent).
//
// It carries no denied_tool_attempts_reported flag, unlike the ported Python's build_summary. That
// flag's source was probe.json's advertised-tools key, itself derived from the client's own advertised
// tool list under the retired claude -p port -- a list with no counterpart under agent dispatch, the
// same reason the transcript-sourced advertised-tools usage field was replaced by the
// definition-sourced Usage.GrantedTools (see usage.go). The deny-list probe's generated agent
// definition now grants the denied tool by construction, so the question the old flag asked is answered
// by the definition rather than observed, and the probe record's own observed-denial-shape key is the
// real successor signal -- there is nothing left for a summary meta flag to add.
type SummaryMeta struct {
	// RunModel is the pinned model id all runs shared.
	RunModel string `json:"run_model"`
	// Scorer carries the pinned scoring model and effort.
	Scorer SummaryScorerMeta `json:"scorer"`
	// Reps is the repetitions per config.
	Reps int `json:"reps"`
	// ResultsRootDate is the results root directory's own base name, e.g. "2026-08-29".
	ResultsRootDate string `json:"results_root_date"`
	// NumConfigs is the count of configs in the ladder.
	NumConfigs int `json:"num_configs"`
	// ColdDisposition is cold_cell.json's disposition field, or "unknown" when the file is absent.
	ColdDisposition string `json:"cold_disposition"`
	// ColdConfirmedColdReps is cold_cell.json's confirmed_cold_reps field, or nil when the file is
	// absent or carries no such field.
	ColdConfirmedColdReps any `json:"cold_confirmed_cold_reps"`
}

// CellRecord is one config's summarised results as written into summary.json's cells mapping: its
// Cell.Stats plus completeness and the per-run observations LoadRuns's flattening carried in.
type CellRecord struct {
	// Stats holds the config's per-metric summarised records.
	Stats map[string]MetricStats `json:"stats"`
	// Complete is true only when every repetition completed.
	Complete bool `json:"complete"`
	// DecoyAdmittedCount counts the runs whose decoy_admitted field was true.
	DecoyAdmittedCount int `json:"decoy_admitted_count"`
	// SummaryMatches carries every run's summary_matches value verbatim, in run order, for the runs
	// that carried the field.
	SummaryMatches []any `json:"summary_matches"`
	// WorktreeDirtiedCount counts the runs whose worktree_dirtied observation was true.
	WorktreeDirtiedCount int `json:"worktree_dirtied_count"`
	// TargetOriginQuarryMentionCount counts the runs whose target_origin_quarry_mention observation was
	// true.
	TargetOriginQuarryMentionCount int `json:"target_origin_quarry_mention_count"`
	// DaemonBackedRuns counts the runs that did not record cold_no_daemon_backed_call.
	DaemonBackedRuns int `json:"daemon_backed_runs"`
}

// Summary is the full mapping WriteSummary serialises to summary.json.
type Summary struct {
	// Meta carries the pinned run parameters and the matrix's completeness signals.
	Meta SummaryMeta `json:"_meta"`
	// Cells maps every config id to its CellRecord.
	Cells map[string]CellRecord `json:"cells"`
	// Comparisons is every Comparison the three comparison builders produced.
	Comparisons []Comparison `json:"comparisons"`
	// Incomplete lists the ids of every cell that is not complete, except a cold cell whose
	// cold_cell.json records "not-run" or "partial" -- both are legitimate terminal states of the
	// cold-cell driver, not interrupted runs.
	Incomplete []string `json:"incomplete"`
}

// buildCellRecord aggregates cell's per-run observations into a CellRecord.
func buildCellRecord(cell Cell) CellRecord {
	var summaryMatches []any
	decoyAdmittedCount := 0
	worktreeDirtiedCount := 0
	targetOriginCount := 0
	daemonBackedRuns := 0
	for _, run := range cell.Runs {
		if decoyAdmitted, ok := run["decoy_admitted"].(bool); ok && decoyAdmitted {
			decoyAdmittedCount++
		}
		if match, ok := run["summary_matches"]; ok {
			summaryMatches = append(summaryMatches, match)
		}
		if dirtied, ok := run["worktree_dirtied"].(bool); ok && dirtied {
			worktreeDirtiedCount++
		}
		if mentioned, ok := run["target_origin_quarry_mention"].(bool); ok && mentioned {
			targetOriginCount++
		}
		coldNoDaemon, _ := run["cold_no_daemon_backed_call"].(bool)
		if !coldNoDaemon {
			daemonBackedRuns++
		}
	}
	return CellRecord{
		Stats:                          cell.Stats,
		Complete:                       cell.Complete,
		DecoyAdmittedCount:             decoyAdmittedCount,
		SummaryMatches:                 summaryMatches,
		WorktreeDirtiedCount:           worktreeDirtiedCount,
		TargetOriginQuarryMentionCount: targetOriginCount,
		DaemonBackedRuns:               daemonBackedRuns,
	}
}

// BuildSummary builds the full Summary: _meta, cells, comparisons, and incomplete.
//
// comparisons is every Comparison the three builders produce. rung-vs-control comparisons are built for
// every config with a non-empty allowed set that is not the cold config; rung-vs-rung comparisons are
// built for every pair of such configs within the same ladder; warm-vs-cold is built once, for the
// ladder's single cold cell.
func BuildSummary(l *Ladder, resultsRoot string) (Summary, error) {
	cells := make(map[string]Cell, len(l.Configs))
	for _, config := range l.Configs {
		runs, err := LoadRuns(resultsRoot, config.ID, l.Reps)
		if err != nil {
			return Summary{}, err
		}
		cells[config.ID] = SummariseCell(config.ID, runs, l.Reps)
	}

	coldCellRecord, err := readJSONOrDefault(filepath.Join(resultsRoot, "cold_cell.json"), map[string]any{})
	if err != nil {
		return Summary{}, err
	}
	coldDisposition, _ := coldCellRecord["disposition"].(string)
	if coldDisposition == "" {
		coldDisposition = "unknown"
	}
	coldConfirmedColdReps := coldCellRecord["confirmed_cold_reps"]

	runModel := ""
	if l.RunModel != nil {
		runModel = *l.RunModel
	}

	meta := SummaryMeta{
		RunModel:              runModel,
		Scorer:                SummaryScorerMeta{Model: l.Scorer.Model, Effort: l.Scorer.Effort},
		Reps:                  l.Reps,
		ResultsRootDate:       filepath.Base(resultsRoot),
		NumConfigs:            len(l.Configs),
		ColdDisposition:       coldDisposition,
		ColdConfirmedColdReps: coldConfirmedColdReps,
	}

	cellRecords := make(map[string]CellRecord, len(l.Configs))
	for _, config := range l.Configs {
		cellRecords[config.ID] = buildCellRecord(cells[config.ID])
	}

	var nonControlConfigs []LadderConfig
	for _, config := range l.Configs {
		if !IsControl(config) && !config.Cold {
			nonControlConfigs = append(nonControlConfigs, config)
		}
	}

	var comparisons []Comparison
	for _, config := range nonControlConfigs {
		configComparisons, err := CompareRungToControl(l, cells, config)
		if err != nil {
			return Summary{}, err
		}
		comparisons = append(comparisons, configComparisons...)
	}
	for _, ladderName := range []string{"a", "b"} {
		var rungIDs []string
		for _, config := range nonControlConfigs {
			if config.Ladder == ladderName {
				rungIDs = append(rungIDs, config.ID)
			}
		}
		for i, leftID := range rungIDs {
			for _, rightID := range rungIDs[i+1:] {
				pairComparisons, err := CompareRungs(l, cells, leftID, rightID)
				if err != nil {
					return Summary{}, err
				}
				comparisons = append(comparisons, pairComparisons...)
			}
		}
	}
	warmColdComparisons, err := CompareWarmCold(l, cells, coldDisposition)
	if err != nil {
		return Summary{}, err
	}
	comparisons = append(comparisons, warmColdComparisons...)

	var incomplete []string
	for _, config := range l.Configs {
		cell := cells[config.ID]
		if cell.Complete {
			continue
		}
		if config.Cold && (coldDisposition == "not-run" || coldDisposition == "partial") {
			continue
		}
		incomplete = append(incomplete, config.ID)
	}

	return Summary{
		Meta:        meta,
		Cells:       cellRecords,
		Comparisons: comparisons,
		Incomplete:  incomplete,
	}, nil
}

// WriteSummary serialises BuildSummary(l, resultsRoot) as summary.json into resultsRoot, with a
// trailing newline.
//
// Returns the built Summary, so callers (including the ladderbench summarize subcommand a later batch
// adds) do not have to re-read the file they just wrote.
func WriteSummary(l *Ladder, resultsRoot string) (Summary, error) {
	summary, err := BuildSummary(l, resultsRoot)
	if err != nil {
		return Summary{}, err
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return Summary{}, fmt.Errorf("ladder: WriteSummary: marshal summary: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(filepath.Join(resultsRoot, "summary.json"), data, 0o644); err != nil {
		return Summary{}, fmt.Errorf("ladder: WriteSummary: write summary.json: %w", err)
	}
	return summary, nil
}

// SummaryExitCode returns 1 when summary's Incomplete list is non-empty, 0 otherwise -- a summary of a
// partial matrix is written but must not be mistaken for a finished one.
//
// This ports _exit_code_for_summary. The Python's main -- the command-line entry point that called it --
// has no counterpart here: dispatching that entry point is a cobra subcommand a later batch adds.
func SummaryExitCode(summary Summary) int {
	if len(summary.Incomplete) > 0 {
		return 1
	}
	return 0
}
