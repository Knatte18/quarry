// proberecord.go implements ladderbench's probe-record subcommand: the CLI-level caller that consumes
// one permission probe's transcript and writes or extends probe.json at the results root with that
// probe's own half of the two enforcement-layer facts the matrix depends on.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func probeRecordCommand() *cobra.Command {
	var probeKind string

	cmd := &cobra.Command{
		Use:   "probe-record",
		Short: "record one permission probe's outcome into probe.json",
		Long: `probe-record locates the transcript one permission probe (--probe allowlist|denylist)
produced, determines whether the probed quarry tool call was blocked, and writes or extends
<results-root>/probe.json with that probe's own boolean key: allowlist_blocks for the allowlist probe,
denylist_blocks for the denylist probe. Each invocation writes only its own key, so a second invocation
(the other probe kind) extends the existing document rather than replacing it.

The deny-list probe additionally captures the verbatim text of the errored tool result it provoked into
the same document under denial_shape_observed -- this is what the follow-up task checks the provisional
denial pattern (see internal/ladder/usage.go's DenialShapePattern) against before clearing the
provisional marker.

probe-record halts -- exits non-zero, after still writing probe.json -- when the probe it just recorded
observed no block: a false allowlist_blocks or denylist_blocks means that enforcement layer did not do
its job, which the matrix must not proceed past silently.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := resolveLadder(cmd)
			if err != nil {
				return err
			}
			resultsRoot, err := resolveResultsRoot(cmd)
			if err != nil {
				return err
			}
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			return runProbeRecord(cmd.OutOrStdout(), l, repoRoot, resultsRoot, probeKind, "", ingestTranscriptWait)
		},
	}

	cmd.Flags().StringVar(&probeKind, "probe", "", fmt.Sprintf("which probe's transcript to record (%q or %q, required)", ladder.ProbeKindAllowlist, ladder.ProbeKindDenylist))

	return cmd
}

// probeRecordSessionConfigID returns the configID ladder.SessionDir derives kind's probe scratch
// directory from, mirroring internal/ladder/session.go's own unexported probeSessionConfigID -- this
// file lives in a different package and cannot reference that directly.
func probeRecordSessionConfigID(kind string) (string, error) {
	switch kind {
	case ladder.ProbeKindAllowlist:
		return "probe-allowlist", nil
	case ladder.ProbeKindDenylist:
		return "probe-denylist", nil
	default:
		return "", fmt.Errorf("probe-record: --probe must be %q or %q, got %q", ladder.ProbeKindAllowlist, ladder.ProbeKindDenylist, kind)
	}
}

// runProbeRecord locates kind's probe transcript, determines whether the probed call was blocked, and
// writes or extends probe.json at resultsRoot with that outcome. projectsRoot and wait are threaded
// through to ladder.LocateTranscript explicitly, matching runIngest's own testing seam.
func runProbeRecord(out io.Writer, l *ladder.Ladder, repoRoot, resultsRoot, kind, projectsRoot string, wait time.Duration) error {
	configID, err := probeRecordSessionConfigID(kind)
	if err != nil {
		return err
	}

	scratchDir, err := ladder.SessionDir(l, repoRoot, configID, 1)
	if err != nil {
		return err
	}
	description := ladder.DispatchDescription(configID, 1, 1)
	transcriptPath, _, err := ladder.LocateTranscript(projectsRoot, scratchDir, description, wait)
	if err != nil {
		return err
	}

	records, err := ladder.ReadTranscript(transcriptPath)
	if err != nil {
		return err
	}

	blocked, deniedText, err := probeOutcome(records)
	if err != nil {
		return err
	}

	record, err := readProbeRecordOrDefault(resultsRoot)
	if err != nil {
		return err
	}

	var key string
	switch kind {
	case ladder.ProbeKindAllowlist:
		key = "allowlist_blocks"
	case ladder.ProbeKindDenylist:
		key = "denylist_blocks"
		if blocked {
			record["denial_shape_observed"] = deniedText
		}
	}
	record[key] = blocked

	if err := os.MkdirAll(resultsRoot, 0o755); err != nil {
		return fmt.Errorf("probe-record: create %s: %w", resultsRoot, err)
	}
	if err := writeIngestJSONDocument(filepath.Join(resultsRoot, "probe.json"), record); err != nil {
		return err
	}

	if !blocked {
		return fmt.Errorf("probe-record: %s probe observed no block -- %s is false, halting", kind, key)
	}

	fmt.Fprintf(out, "probe-record: recorded %s = true\n", key)
	return nil
}

// readProbeRecordOrDefault returns the parsed probe.json object at resultsRoot, or an empty mapping when
// it does not yet exist -- probe-record is the first writer of probe.json, so its own first invocation
// must not treat an absent file as an error.
func readProbeRecordOrDefault(resultsRoot string) (map[string]any, error) {
	path := filepath.Join(resultsRoot, "probe.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("probe-record: read %s: %w", path, err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("probe-record: parse %s: %w", path, err)
	}
	return record, nil
}

// toolResultBlocksByID maps a tool_use_id to its matching tool_result content block, across every user
// record's message content -- probe-record's own copy of gates.go's unexported toolResultsByID, which
// this file cannot reference directly from a different package.
func toolResultBlocksByID(records []ladder.Record) map[string]ladder.ContentBlock {
	results := make(map[string]ladder.ContentBlock)
	for _, record := range records {
		if record.Type != "user" {
			continue
		}
		for _, block := range record.Message.Content {
			if block.Type == "tool_result" {
				results[block.ToolUseID] = block
			}
		}
	}
	return results
}

// toolResultText concatenates the text of every "text" block in a tool_result content block's own
// nested Content, in order -- probe-record's own copy of usage.go's unexported toolResultText.
func toolResultText(block ladder.ContentBlock) string {
	var text strings.Builder
	for _, inner := range block.Content {
		if inner.Type == "text" {
			text.WriteString(inner.Text)
		}
	}
	return text.String()
}

// assistantTextContainsSentinel reports whether any assistant text content block across records contains
// sentinel -- probe-record's ground-truth-free fallback for detecting a schema-absent tool, since a
// subagent transcript carries no advertised-tools list an absence could otherwise be checked against (see
// transcript.go's Record doc comment).
func assistantTextContainsSentinel(records []ladder.Record, sentinel string) bool {
	for _, record := range records {
		if record.Type != "assistant" {
			continue
		}
		for _, block := range record.Message.Content {
			if block.Type == "text" && strings.Contains(block.Text, sentinel) {
				return true
			}
		}
	}
	return false
}

// probeOutcome finds the transcript's one call to a quarry tool (an mcp__quarry__* tool_use block --
// every probe's own prompt asks for exactly one) and reports whether its matching tool_result errored,
// alongside that result's verbatim text when it did.
//
// When the transcript carries no quarry tool call at all, that is not itself an error: it is the expected
// shape of a working block, since both enforcement layers this suite probes remove the tool from the
// model's schema entirely rather than producing a call-time refusal (see
// ladder.ProbeNotInSchemaSentinel's doc comment) -- the model is never offered the tool, so it can never
// emit a tool_use block for it. That case is only accepted as blocked when the transcript's own assistant
// text contains the fixed sentinel probePromptBody instructs the agent to reply with, so a probe agent
// that simply failed to attempt the call for some unrelated reason still surfaces as an error rather than
// a silently-accepted false blocked=true.
//
// Returns an error when the transcript carries a quarry tool call with no matching tool_result to read,
// or when it carries neither a quarry tool call nor the not-in-schema sentinel.
func probeOutcome(records []ladder.Record) (blocked bool, deniedText string, err error) {
	resultsByID := toolResultBlocksByID(records)
	for _, block := range ladder.IterToolUseBlocks(records) {
		if !strings.HasPrefix(block.Name, ladder.MCPPrefix) {
			continue
		}
		result, ok := resultsByID[block.ToolUseID]
		if !ok {
			return false, "", fmt.Errorf("probe-record: transcript's call to %q carries no matching tool_result", block.Name)
		}
		if result.IsError {
			return true, toolResultText(result), nil
		}
		return false, "", nil
	}
	if assistantTextContainsSentinel(records, ladder.ProbeNotInSchemaSentinel) {
		return true, ladder.ProbeNotInSchemaSentinel, nil
	}
	return false, "", fmt.Errorf("probe-record: transcript carries no call to a quarry tool and no %s sentinel", ladder.ProbeNotInSchemaSentinel)
}
