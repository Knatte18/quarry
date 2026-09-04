// summarize.go turns a results root into the numbers a reader acts on, by re-deriving every metric
// from the raw tree rather than trusting the diagnostic-only usage.json each repetition wrote. Every
// cost metric is recomputed from that repetition's own transcript.jsonl; recall and precision -- the
// scorer's own judgment, existing nowhere else -- are read from the repetition's score.json. The
// asymmetry is the point: recomputing cost is exactly what keeping the transcripts buys, and it is
// what makes an accounting fix cost a re-report rather than a re-run.
//
// Summarize never loads the ladder file: every fact it needs about a cell -- its ladder letter, task
// id, allowed-tool list and control flag -- comes from that cell's own run.json, so a results root
// stays self-describing after the ladder file that produced it has moved or changed.

package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// SummaryFile is the file name a results root's summary is written under: the single spelling of
// that name, matching how provenance.go pins ProvenanceFile and report.go pins TableFile.
const SummaryFile = "summary.json"

// costMetricNames is the fixed, ordered set of cost-metric names Summarize recomputes from every
// repetition's transcript and RenderTable renders as table columns, in the same order both use.
var costMetricNames = []string{
	"turns",
	"duration_ms",
	"cost_usd",
	"cache_read_input_tokens",
	"cache_creation_input_tokens",
	"output_tokens",
	"input_tokens_total",
	"tool_uses",
	"quarry_tool_uses",
	"grep_fallback_total",
	"tool_result_bytes",
	"read_bytes",
}

// correctnessMetricNames is the fixed set of correctness-metric names Summarize reads from each
// scored repetition's score.json, under the exact keys both scoring rules declare.
var correctnessMetricNames = []string{"recall", "precision"}

// comparisonMetricNames is every metric name a rung-vs-control comparison is eligible over: every
// cost metric plus the two correctness metrics. Unlike V1, no metric is excluded here -- this harness
// renders one identical preamble for every cell, so a grep-usage gap between a rung and its control
// carries no confound to guard against.
func comparisonMetricNames() []string {
	names := make([]string, 0, len(costMetricNames)+len(correctnessMetricNames))
	names = append(names, costMetricNames...)
	names = append(names, correctnessMetricNames...)
	return names
}

// costMetricValues reduces one repetition's recomputed Metrics to the named values costMetricNames
// declares, so summarizeCell and RenderTable read the same field-to-name mapping.
func costMetricValues(m Metrics) map[string]float64 {
	return map[string]float64{
		"turns":                       float64(m.NumTurns),
		"duration_ms":                 float64(m.DurationMS),
		"cost_usd":                    m.TotalCostUSD,
		"cache_read_input_tokens":     float64(m.CacheReadInputTokens),
		"cache_creation_input_tokens": float64(m.CacheCreationInputTokens),
		"output_tokens":               float64(m.OutputTokens),
		"input_tokens_total":          float64(m.InputTokensTotal),
		"tool_uses":                   float64(m.ToolUses),
		"quarry_tool_uses":            float64(m.QuarryToolUses),
		"grep_fallback_total":         float64(m.GrepFallbackTotal),
		"tool_result_bytes":           float64(m.ToolResultBytes),
		"read_bytes":                  float64(m.ReadBytes),
	}
}

// MetricStats is one metric's median, minimum, maximum and sample count over the repetitions that
// contributed a value to it.
type MetricStats struct {
	// Median is the metric's true median across the contributing repetitions: the middle value at an
	// odd sample count, the mean of the two middle values at an even one.
	Median float64 `json:"median"`
	// Min is the metric's minimum value across the contributing repetitions.
	Min float64 `json:"min"`
	// Max is the metric's maximum value across the contributing repetitions.
	Max float64 `json:"max"`
	// N is the count of repetitions that contributed a value to this metric. A cell's cost sample
	// count and its correctness sample count may legitimately differ, since the three exclusion rules
	// apply to correctness but not cost -- each metric's own N reflects only the repetitions that
	// actually contributed to it.
	N int `json:"n"`
}

// CellRecord is one cell's summarised results: its identity as carried by its own repetitions'
// run.json, its per-metric statistics, the three exclusion counters, and its gate-1 finding when one
// fired.
type CellRecord struct {
	// ID is the cell id.
	ID string `json:"id"`
	// Ladder is the ladder letter this cell belongs to.
	Ladder string `json:"ladder"`
	// Task is the task id this cell runs.
	Task string `json:"task"`
	// Allowed is the tool subset this cell grants.
	Allowed []string `json:"allowed"`
	// Control reports whether this cell is its ladder letter's control.
	Control bool `json:"control"`
	// Metrics maps a metric name -- every entry of costMetricNames plus correctnessMetricNames -- to
	// its summarised statistics, present only for a metric at least one repetition contributed to.
	Metrics map[string]MetricStats `json:"metrics"`
	// MaxTurnsCount is the count of this cell's repetitions that hit the turn ceiling.
	MaxTurnsCount int `json:"max_turns_count"`
	// UnscoredCount is the count of this cell's repetitions whose score record carried a false scored
	// flag for a reason other than the turn ceiling.
	UnscoredCount int `json:"unscored_count"`
	// BlindingFailedCount is the count of this cell's repetitions whose blinding-failed flag was set.
	BlindingFailedCount int `json:"blinding_failed_count"`
	// Gate1 is this cell's gate-1 (granted-tool-used) finding, nil when gate 1 did not fire.
	Gate1 *Finding `json:"gate1,omitempty"`
}

// SummaryMeta is summary.json's meta block. It names only the results root's base name -- never the
// operator-supplied path, and never a wall-clock write time -- per the overview's
// no-machine-paths-in-tracked-output decision, which is also what makes this file byte-for-byte
// golden-comparable.
type SummaryMeta struct {
	// ResultsRoot is the results root directory's own base name.
	ResultsRoot string `json:"results_root"`
}

// Comparison is one structured, mechanically checkable comparison between a rung cell and its own
// ladder letter's control, over one metric. Disjointness is non-overlapping minimum-maximum ranges;
// there is no significance testing.
type Comparison struct {
	// Cell is the rung cell's id.
	Cell string `json:"cell"`
	// Control is the control cell's id.
	Control string `json:"control"`
	// Metric is the compared metric's name.
	Metric string `json:"metric"`
	// CellMedian is the rung cell's median for this metric.
	CellMedian float64 `json:"cell_median"`
	// ControlMedian is the control cell's median for this metric.
	ControlMedian float64 `json:"control_median"`
	// CellRange is the rung cell's (min, max) pair for this metric.
	CellRange [2]float64 `json:"cell_range"`
	// ControlRange is the control cell's (min, max) pair for this metric.
	ControlRange [2]float64 `json:"control_range"`
	// Separated is true only when CellRange and ControlRange do not overlap at all.
	Separated bool `json:"separated"`
}

// RangesDisjoint reports whether the two closed ranges a and b -- each a (min, max) pair -- do not
// overlap at all. Ranges that touch at a single value -- one's max equals the other's min -- are
// treated as NOT disjoint.
func RangesDisjoint(a, b [2]float64) bool {
	return a[1] < b[0] || b[1] < a[0]
}

// Summary is the full record WriteSummary serialises to summary.json: every selected cell's
// CellRecord, every rung-vs-control comparison, the incomplete and invalid cell-id lists, and the
// meta block.
type Summary struct {
	// Cells is every selected cell's CellRecord, sorted by cell id.
	Cells []CellRecord `json:"cells"`
	// Comparisons is every rung-vs-control comparison the two cells' shared metrics support.
	Comparisons []Comparison `json:"comparisons"`
	// Incomplete lists the ids of every cell whose present-and-not-void repetition count is short of
	// the provenance record's selected cells crossed with its effective repetition count.
	Incomplete []string `json:"incomplete"`
	// Invalid lists the ids of every cell with a non-zero blinding-failed count.
	Invalid []string `json:"invalid"`
	// Meta carries the results root's base name and nothing else.
	Meta SummaryMeta `json:"meta"`
}

// Summarize walks resultsRoot's raw tree and recomputes a Summary for it. It reads each repetition's
// run.json for that repetition's cell metadata -- no ladder file is consulted -- recomputes that
// repetition's cost metrics from its own transcript.jsonl, and reads recall and precision from its
// score.json. A results root whose provenance record is absent is an error naming the missing file:
// the run subcommand writes that record before its first repetition, so a raw tree without one is a
// hand-assembled or truncated root, and reporting an empty incomplete list for it would make an
// unknowably short root read as finished.
func Summarize(resultsRoot string) (*Summary, error) {
	prov, err := ReadProvenance(resultsRoot)
	if err != nil {
		return nil, fmt.Errorf("summarize %s: %w", resultsRoot, err)
	}
	if prov == nil {
		return nil, fmt.Errorf(
			"summarize %s: missing %s -- the run subcommand writes this file before its first repetition, so a raw tree without one is a hand-assembled or truncated root",
			resultsRoot, ProvenanceFile,
		)
	}

	cellIDs := append([]string(nil), prov.SelectedCells...)
	sort.Strings(cellIDs)

	var cells []CellRecord
	var incomplete []string
	var invalid []string
	controlByLadder := map[string]CellRecord{}

	for _, cellID := range cellIDs {
		record, presentNonVoid, err := summarizeCell(resultsRoot, cellID, prov.RepsEffective)
		if err != nil {
			return nil, err
		}
		if presentNonVoid < prov.RepsEffective {
			incomplete = append(incomplete, cellID)
		}
		if record.BlindingFailedCount > 0 {
			invalid = append(invalid, cellID)
		}
		cells = append(cells, record)
		if record.Control {
			controlByLadder[record.Ladder] = record
		}
	}

	comparisons := buildComparisons(cells, controlByLadder)

	return &Summary{
		Cells:       cells,
		Comparisons: comparisons,
		Incomplete:  incomplete,
		Invalid:     invalid,
		Meta:        SummaryMeta{ResultsRoot: filepath.Base(resultsRoot)},
	}, nil
}

// summarizeCell builds cellID's CellRecord from its own repetitions, 1..repsEffective, applying the
// three exclusion rules -- all distinct: a blinding-failed repetition contributes nothing at all,
// neither cost nor correctness, is not counted present for completeness, and increments
// BlindingFailedCount; a max-turns repetition contributes cost but not recall or precision and
// increments MaxTurnsCount; a repetition whose score record carries a false scored flag for any other
// reason is likewise excluded from recall and precision and increments UnscoredCount. It also returns
// the present-and-not-void repetition count Summarize compares against repsEffective to populate the
// incomplete list.
func summarizeCell(resultsRoot, cellID string, repsEffective int) (CellRecord, int, error) {
	record := CellRecord{ID: cellID}
	values := map[string][]float64{}
	var perRepQuarryToolUses []int
	metadataCaptured := false
	presentNonVoid := 0

	for rep := 1; rep <= repsEffective; rep++ {
		dir := RepDir(resultsRoot, cellID, rep)
		state, err := ReadRunState(dir)
		if err != nil {
			// Not present: never written, or invalidated and renamed away by InvalidateRep.
			continue
		}
		if !metadataCaptured {
			record.Ladder = state.Ladder
			record.Task = state.Task
			record.Allowed = append([]string(nil), state.Allowed...)
			record.Control = state.IsControl
			metadataCaptured = true
		}
		if state.BlindingFailed {
			record.BlindingFailedCount++
			continue
		}
		presentNonVoid++

		transcript, err := readRepTranscript(dir)
		if err != nil {
			return CellRecord{}, 0, fmt.Errorf("summarize cell %s rep %d: %w", cellID, rep, err)
		}
		metrics := ComputeMetrics(transcript, state.MCPPrefix)
		for name, value := range costMetricValues(metrics) {
			values[name] = append(values[name], value)
		}
		perRepQuarryToolUses = append(perRepQuarryToolUses, metrics.QuarryToolUses)

		if state.MaxTurnsHit {
			record.MaxTurnsCount++
			continue
		}
		if !state.Scored {
			record.UnscoredCount++
			continue
		}

		score, err := readRepScore(dir)
		if err != nil {
			return CellRecord{}, 0, fmt.Errorf("summarize cell %s rep %d: %w", cellID, rep, err)
		}
		recall, recallOK := numericScoreField(score, "recall")
		precision, precisionOK := numericScoreField(score, "precision")
		if recallOK && precisionOK {
			values["recall"] = append(values["recall"], recall)
			values["precision"] = append(values["precision"], precision)
		}
	}

	record.Metrics = map[string]MetricStats{}
	for name, vs := range values {
		if len(vs) == 0 {
			continue
		}
		record.Metrics[name] = computeStats(vs)
	}

	cfg := Config{ID: cellID, Ladder: record.Ladder, Task: record.Task, Allowed: record.Allowed}
	record.Gate1 = CheckGrantedToolUsed(cfg, perRepQuarryToolUses)

	return record, presentNonVoid, nil
}

// buildComparisons builds one Comparison per (rung cell, metric) pair for every metric both the rung
// cell and its own ladder letter's control carry statistics for, in cells order and
// comparisonMetricNames order.
func buildComparisons(cells []CellRecord, controlByLadder map[string]CellRecord) []Comparison {
	var comparisons []Comparison
	for _, cell := range cells {
		if cell.Control {
			continue
		}
		control, ok := controlByLadder[cell.Ladder]
		if !ok {
			continue
		}
		for _, metric := range comparisonMetricNames() {
			cellStats, ok := cell.Metrics[metric]
			if !ok {
				continue
			}
			controlStats, ok := control.Metrics[metric]
			if !ok {
				continue
			}
			cellRange := [2]float64{cellStats.Min, cellStats.Max}
			controlRange := [2]float64{controlStats.Min, controlStats.Max}
			comparisons = append(comparisons, Comparison{
				Cell:          cell.ID,
				Control:       control.ID,
				Metric:        metric,
				CellMedian:    cellStats.Median,
				ControlMedian: controlStats.Median,
				CellRange:     cellRange,
				ControlRange:  controlRange,
				Separated:     RangesDisjoint(cellRange, controlRange),
			})
		}
	}
	return comparisons
}

// readRepTranscript parses dir/transcript.jsonl.
func readRepTranscript(dir string) (*Transcript, error) {
	path := filepath.Join(dir, TranscriptFile)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read transcript %s: %w", path, err)
	}
	defer f.Close()
	t, err := ParseTranscript(f)
	if err != nil {
		return nil, fmt.Errorf("read transcript %s: %w", path, err)
	}
	return t, nil
}

// readRepScore reads and decodes dir/score.json.
func readRepScore(dir string) (ScoreRecord, error) {
	path := filepath.Join(dir, ScoreFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read score %s: %w", path, err)
	}
	var record ScoreRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse score %s: %w", path, err)
	}
	return record, nil
}

// numericScoreField reads key out of record as a float64, reporting whether it was present and
// numeric. encoding/json decodes a JSON number into a Go float64 by default, which is the only shape
// a scorer's own reply produces for recall and precision.
func numericScoreField(record ScoreRecord, key string) (float64, bool) {
	v, ok := record[key]
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

// computeStats reduces values, which must be non-empty, to its MetricStats: the true median, the
// minimum, the maximum and the sample count.
func computeStats(values []float64) MetricStats {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return MetricStats{
		Median: trueMedian(sorted),
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		N:      len(sorted),
	}
}

// trueMedian returns the true median of sorted, which must already be sorted and non-empty: the
// middle value at an odd length, the mean of the two middle values at an even length.
func trueMedian(sorted []float64) float64 {
	n := len(sorted)
	mid := n / 2
	if n%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// WriteSummary writes s to resultsRoot/summary.json.
func WriteSummary(resultsRoot string, s *Summary) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	path := filepath.Join(resultsRoot, SummaryFile)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write summary %s: %w", path, err)
	}
	return nil
}
