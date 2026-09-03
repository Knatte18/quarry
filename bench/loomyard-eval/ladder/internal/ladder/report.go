// report.go renders a Summary and its Provenance into the plain-text per-cell table an operator reads
// after a run or a report subcommand invocation: one header block, one row per cell, and the findings
// that do not fit a row -- gate-1 findings, comparisons, the incomplete and invalid lists, and the two
// drift observations. It depends on no table library; every column is fixed-width via text/tabwriter,
// the standard library's own column aligner.

package ladder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// TableFile is the file name a results root's table is written under: the single spelling of that
// name, matching how provenance.go pins ProvenanceFile and summarize.go pins SummaryFile.
const TableFile = "table.txt"

// tableColumnNames is the per-cell row's fixed column order: the cell identity, the sample count, the
// twelve cost-metric medians in costMetricNames order, then recall and precision with their own
// sample count, then a flags column.
var tableColumnNames = []string{
	"cell", "ladder", "n",
	"turns", "duration_ms", "cost_usd", "cache_read", "cache_creation",
	"output_tokens", "input_tokens_total", "tool_uses", "prefixed_tool_uses",
	"grep_fallback", "tool_result_bytes", "read_bytes",
	"recall", "recall_n", "precision", "precision_n",
	"flags",
}

// RenderTable renders s and p into the per-cell table: a header block naming the results root's base
// name, the effective repetition count and the CLI version, stating the cache caveat in the harness's
// own words -- the first repetition of a root pays cache creation while later repetitions read it, so
// cache-read and cache-creation figures are reported separately, never summed, and the median over
// repetitions is the honest statistic -- followed by one row per cell and, below the rows, every
// gate-1 finding verbatim, every comparison and its disjointness verdict, the incomplete list, the
// invalid list, any server-hash-drift warning and any session-fingerprint-drift observations.
func RenderTable(s *Summary, p *Provenance) string {
	var b strings.Builder

	fmt.Fprintf(&b, "results root: %s\n", s.Meta.ResultsRoot)
	fmt.Fprintf(&b, "reps: %d\n", p.RepsEffective)
	fmt.Fprintf(&b, "claude_code_version: %s\n", p.ClaudeVersion)
	fmt.Fprintln(&b,
		"cache caveat: the first repetition of a root pays cache creation while later repetitions "+
			"read it -- cache-read and cache-creation figures are reported separately, never summed, "+
			"and per-repetition numbers are not interchangeable; the median over repetitions is the "+
			"honest statistic.")
	fmt.Fprintln(&b)

	tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(tableColumnNames, "\t"))
	for _, cell := range s.Cells {
		fmt.Fprintln(tw, strings.Join(tableRow(cell), "\t"))
	}
	tw.Flush()

	fmt.Fprintln(&b)
	for _, cell := range s.Cells {
		if cell.Gate1 != nil {
			fmt.Fprintln(&b, cell.Gate1.Message)
		}
	}

	fmt.Fprintln(&b)
	for _, c := range s.Comparisons {
		fmt.Fprintf(&b, "comparison: %s vs %s (control) on %s: median %s=%s cell-range=%s "+
			"control-range=%s separated=%v\n",
			c.Cell, c.Control, c.Metric,
			formatFloat(c.CellMedian), formatFloat(c.ControlMedian),
			formatRange(c.CellRange), formatRange(c.ControlRange), c.Separated)
	}

	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "incomplete: %s\n", formatIDList(s.Incomplete))
	fmt.Fprintf(&b, "invalid: %s\n", formatIDList(s.Invalid))

	if f := WarnOnServerHashDrift(p); f != nil {
		fmt.Fprintln(&b, f.Message)
	}
	for _, f := range CompareFingerprints(p) {
		fmt.Fprintln(&b, f.Message)
	}

	return b.String()
}

// tableRow renders one cell's row, in tableColumnNames order.
func tableRow(cell CellRecord) []string {
	row := []string{cell.ID, cell.Ladder, formatSampleCount(cell)}
	for _, metric := range costMetricNames {
		row = append(row, formatMetric(cell.Metrics[metric]))
	}
	row = append(row,
		formatMetric(cell.Metrics["recall"]), formatMetricN(cell.Metrics["recall"]),
		formatMetric(cell.Metrics["precision"]), formatMetricN(cell.Metrics["precision"]),
	)
	row = append(row, strings.Join(rowFlags(cell), ","))
	return row
}

// formatSampleCount returns the cell's cost sample count: the N of any cost metric, since every cost
// metric is recomputed from the same set of present-and-not-void repetitions and so shares one N. A
// cell with no cost metrics at all -- every repetition blinding-failed or absent -- reports 0.
func formatSampleCount(cell CellRecord) string {
	for _, metric := range costMetricNames {
		if stats, ok := cell.Metrics[metric]; ok {
			return fmt.Sprintf("%d", stats.N)
		}
	}
	return "0"
}

// rowFlags returns the short markers RenderTable's header comment declares: BLINDING_FAILED when the
// cell's blinding-failed count is non-zero, MAX_TURNS or UNSCORED when the corresponding counter is
// non-zero, and GATE1 when gate 1 fired.
func rowFlags(cell CellRecord) []string {
	var flags []string
	if cell.BlindingFailedCount > 0 {
		flags = append(flags, "BLINDING_FAILED")
	}
	if cell.MaxTurnsCount > 0 {
		flags = append(flags, "MAX_TURNS")
	}
	if cell.UnscoredCount > 0 {
		flags = append(flags, "UNSCORED")
	}
	if cell.Gate1 != nil {
		flags = append(flags, "GATE1")
	}
	return flags
}

// formatMetric formats stats as its median, or "-" when the metric carries no stats at all.
func formatMetric(stats MetricStats) string {
	if stats.N == 0 {
		return "-"
	}
	return formatFloat(stats.Median)
}

// formatMetricN formats stats.N, or "-" when the metric carries no stats at all.
func formatMetricN(stats MetricStats) string {
	if stats.N == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", stats.N)
}

// formatFloat formats f with up to four decimal places, trimming trailing zeros and a trailing
// decimal point.
func formatFloat(f float64) string {
	s := fmt.Sprintf("%.4f", f)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

// formatRange formats r as "[min, max]".
func formatRange(r [2]float64) string {
	return fmt.Sprintf("[%s, %s]", formatFloat(r[0]), formatFloat(r[1]))
}

// formatIDList formats ids as a comma-joined list, or "none" when empty.
func formatIDList(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

// WriteTable writes table to resultsRoot/table.txt.
func WriteTable(resultsRoot, table string) error {
	path := filepath.Join(resultsRoot, TableFile)
	if err := os.WriteFile(path, []byte(table), 0o644); err != nil {
		return fmt.Errorf("write table %s: %w", path, err)
	}
	return nil
}
