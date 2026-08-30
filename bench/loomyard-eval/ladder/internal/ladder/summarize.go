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
// cannot be made from fewer than reps cold runs -- when either cell is incomplete, or when every cold
// run recorded cold_no_daemon_backed_call: none of them started a daemon, so there is no warmth contrast
// to draw.
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
		return nil, fmt.Errorf("ladder: CompareWarmCold: no cold config found in ladder")
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
