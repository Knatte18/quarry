package ladder

import (
	"strings"
	"testing"
)

// TestRenderTable_HeaderCarriesCacheCaveatAndCLIVersion asserts the header block names the results
// root's base name, the effective repetition count, the CLI version, and states the cache caveat.
func TestRenderTable_HeaderCarriesCacheCaveatAndCLIVersion(t *testing.T) {
	s := &Summary{Meta: SummaryMeta{ResultsRoot: "2026-09-03-toc"}}
	p := &Provenance{RepsEffective: 5, ClaudeVersion: "2.5.0 (Claude Code)"}

	table := RenderTable(s, p)

	if !strings.Contains(table, "2026-09-03-toc") {
		t.Error("table does not carry the results root's base name")
	}
	if !strings.Contains(table, "5") {
		t.Error("table does not carry the effective repetition count")
	}
	if !strings.Contains(table, "2.5.0 (Claude Code)") {
		t.Error("table does not carry the CLI version")
	}
	if !strings.Contains(table, "cache creation") || !strings.Contains(table, "cache-read") {
		t.Error("table does not state the cache caveat")
	}
}

// TestRenderTable_FlagsBlindingFailedCell asserts a cell with a non-zero blinding-failed count is
// flagged in its own row, not elsewhere.
func TestRenderTable_FlagsBlindingFailedCell(t *testing.T) {
	s := &Summary{
		Cells: []CellRecord{
			{ID: "a0-none", Ladder: "a", Metrics: map[string]MetricStats{}, BlindingFailedCount: 1},
			{ID: "a1-tool", Ladder: "a", Metrics: map[string]MetricStats{}},
		},
	}
	p := &Provenance{}

	table := RenderTable(s, p)
	lines := strings.Split(table, "\n")

	var a0Line, a1Line string
	for _, line := range lines {
		if strings.HasPrefix(line, "a0-none") {
			a0Line = line
		}
		if strings.HasPrefix(line, "a1-tool") {
			a1Line = line
		}
	}
	if !strings.Contains(a0Line, "BLINDING_FAILED") {
		t.Errorf("a0-none row = %q; want it to carry BLINDING_FAILED", a0Line)
	}
	if strings.Contains(a1Line, "BLINDING_FAILED") {
		t.Errorf("a1-tool row = %q; want no BLINDING_FAILED flag", a1Line)
	}
}

// TestRenderTable_PrintsGate1FindingVerbatim asserts a cell's gate-1 finding is printed verbatim
// below the rows.
func TestRenderTable_PrintsGate1FindingVerbatim(t *testing.T) {
	finding := &Finding{
		Gate:    "granted_tool_used",
		Message: "!! a1-unused: tool-granted config whose agent never called a granted tool in any repetition -- this cell measures the tool's prompt cost, not the tool",
	}
	s := &Summary{
		Cells: []CellRecord{
			{ID: "a1-unused", Ladder: "a", Metrics: map[string]MetricStats{}, Gate1: finding},
		},
	}
	p := &Provenance{}

	table := RenderTable(s, p)
	if !strings.Contains(table, finding.Message) {
		t.Errorf("table does not carry the gate-1 finding %q verbatim", finding.Message)
	}
}

// TestRenderTable_PrintsIncompleteAndInvalidLists asserts the incomplete and invalid lists appear.
func TestRenderTable_PrintsIncompleteAndInvalidLists(t *testing.T) {
	s := &Summary{
		Cells:      []CellRecord{{ID: "a0-none", Ladder: "a", Metrics: map[string]MetricStats{}}},
		Incomplete: []string{"a0-none"},
		Invalid:    []string{"a0-none"},
	}
	p := &Provenance{}

	table := RenderTable(s, p)
	if !strings.Contains(table, "incomplete:") || !strings.Contains(table, "a0-none") {
		t.Error("table does not print the incomplete list")
	}
	if !strings.Contains(table, "invalid:") {
		t.Error("table does not print the invalid list")
	}
}
