// ingest.go implements ladderbench's ingest subcommand: the CLI-level caller that correlates a finished
// dispatch attempt with its subagent transcript, takes custody of the session's launch inputs and
// transcript, extracts usage and the final answer, runs the gates, and records the run session's own
// terminal marker (ingest.json) on success.

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

// ingestSettingsRelativePath, ingestServerDeclarationFilename, and ingestAgentsRelativeDir mirror
// internal/ladder/session.go's own unexported filename constants. ingest.go lives in a different
// package and cannot reference those directly, so it names the same fixed literals a session's scratch
// directory writes under.
const (
	ingestSettingsRelativePath      = ".claude/settings.json"
	ingestServerDeclarationFilename = ".mcp.json"
	ingestAgentsRelativeDir         = ".claude/agents"
)

// ingestTranscriptWait bounds LocateTranscript's wait for a matched metadata's sibling transcript to be
// flushed to disk before ingest hard-errors, matching DaemonExitTimeout's own generous margin rather
// than a tight bound that would flake under a slow filesystem.
const ingestTranscriptWait = 30 * time.Second

func ingestCommand() *cobra.Command {
	var configID string
	var rep int

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "correlate and record one finished dispatch attempt's outcome",
		Long: `ingest is called once a run session's dispatch for --config-id's --rep repetition has
finished. In order: it enforces the single-flight predicate before touching any file; derives this
attempt's correlation description with ladder.DispatchDescription, taking the attempt index from
ladder.NextAttempt -- the same derivation next-run uses, so the two commands provably build the same
string; locates the subagent transcript that description names and copies it, its metadata, and the
session's own launch inputs (its settings document, its run agent definition, and its server declaration
when the config has one) into the run directory; extracts usage.json and answer.json; takes the worktree
dirtiness observation before anything could restore the worktree and erase that evidence; runs the
gates; and, only on success, writes ingest.json.

ingest never destroys evidence on failure -- invalidation is a separate command (see invalidate --help)
-- and a truncated outcome is never retried. It enforces the full pin set before running the gates, which
is what makes the turn ceiling readable: the ceiling ships blank and the gate would otherwise compare
against nothing.`,
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
			if configID == "" {
				return fmt.Errorf("ingest: --config-id is required")
			}
			if !cmd.Flags().Changed("rep") {
				return fmt.Errorf("ingest: --rep is required")
			}
			return runIngest(cmd.OutOrStdout(), l, repoRoot, resultsRoot, configID, rep, "", ingestTranscriptWait)
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "the LadderConfig id this dispatch attempt belongs to (required)")
	cmd.Flags().IntVar(&rep, "rep", 0, "1-based repetition index (required)")

	return cmd
}

// runIngest performs ingest's full sequence for one dispatch attempt of configID's rep-th repetition.
// projectsRoot and transcriptWait are threaded through to ladder.LocateTranscript explicitly -- an empty
// projectsRoot selects LocateTranscript's own ~/.claude/projects default -- so a test can point them at a
// fixture tree without touching the operator's real projects directory.
func runIngest(out io.Writer, l *ladder.Ladder, repoRoot, resultsRoot, configID string, rep int, projectsRoot string, transcriptWait time.Duration) error {
	if err := ladder.RequirePins(l); err != nil {
		return err
	}
	config, err := ladder.ConfigByID(l, configID)
	if err != nil {
		return err
	}

	if err := ladder.CheckSingleFlight(resultsRoot, configID, rep); err != nil {
		return err
	}

	attempt, err := ladder.NextAttempt(resultsRoot, configID, rep)
	if err != nil {
		return err
	}
	description := ladder.DispatchDescription(configID, rep, attempt)

	scratchDir, err := ladder.SessionDir(l, repoRoot, configID, rep)
	if err != nil {
		return err
	}
	transcriptPath, metaPath, err := ladder.LocateTranscript(projectsRoot, scratchDir, description, transcriptWait)
	if err != nil {
		return err
	}

	runDir := ladder.RunDirPath(resultsRoot, configID, rep)
	if err := ladder.CopyTranscriptCustody(transcriptPath, metaPath, runDir); err != nil {
		return err
	}
	if err := copySessionLaunchInputs(scratchDir, runDir, config); err != nil {
		return err
	}

	copiedTranscriptPath := filepath.Join(runDir, "transcript.jsonl")
	records, err := ladder.ReadTranscript(copiedTranscriptPath)
	if err != nil {
		return err
	}

	definitionPath := filepath.Join(runDir, ingestAgentsRelativeDir, config.ID+".md")
	grantedTools, err := ladder.GrantedToolsFromDefinition(definitionPath)
	if err != nil {
		return err
	}

	usage, err := ladder.ExtractUsage(records, copiedTranscriptPath, transcriptPath, grantedTools)
	if err != nil {
		return err
	}
	if err := writeIngestJSONDocument(filepath.Join(runDir, "usage.json"), usage); err != nil {
		return err
	}

	answer, err := parseFinalAnswer(records)
	if err != nil {
		return err
	}
	if err := writeIngestJSONDocument(filepath.Join(runDir, "answer.json"), answer); err != nil {
		return err
	}

	// The dirtiness observation is taken here, before anything in this command (or any command run
	// after it) could restore the worktree -- restore-worktree is a separate subcommand ingest never
	// calls, but the observation is still taken as early as the transcript allows so it is never
	// accidentally moved past a future caller's own restore step.
	worktree := targetDirFor(l, config, rep)
	dirtied := ladder.ObserveWorktreeDirtied(worktree)

	cacheDir, err := ladder.UserCacheDir()
	if err != nil {
		return err
	}
	env := ladder.ScrubbedEnv()

	taskText, err := ladder.TaskTextFor(l, repoRoot, config.Task)
	if err != nil {
		return err
	}
	report := ladder.RunGates(records, l, config, *l.RunModel, repoRoot, worktree, *l.MaxTurns, dirtied, cacheDir, env, taskText)

	switch ingestOutcome(report) {
	case "ingested":
		rec := ladder.NewIngestRecord(configID, rep, attempt, report)
		if err := ladder.WriteIngestJSON(runDir, rec); err != nil {
			return err
		}
		fmt.Fprintf(out, "ingest: ingested config %q rep %d attempt %d\n", configID, rep, attempt)
	case "truncated":
		fmt.Fprintf(out, "ingest: truncated config %q rep %d attempt %d: %s\n", configID, rep, attempt, fatalFindingsSummary(report))
	default:
		fmt.Fprintf(out, "ingest: failed config %q rep %d attempt %d: %s\n", configID, rep, attempt, fatalFindingsSummary(report))
	}
	return nil
}

// copySessionLaunchInputs copies scratchDir's settings document and run agent definition into runDir as
// settings.json and <ingestAgentsRelativeDir>/<config.ID>.md, and its server declaration as mcp.json only
// when config.Allowed is non-empty -- at parity with what the retired Python client wrote per run, since
// a config whose allowed set is empty never carries a server declaration at all (see
// internal/ladder/session.go's PrepareRunSession doc comment).
func copySessionLaunchInputs(scratchDir, runDir string, config ladder.LadderConfig) error {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("ingest: create %s: %w", runDir, err)
	}

	if err := copyFileInto(filepath.Join(scratchDir, ingestSettingsRelativePath), filepath.Join(runDir, "settings.json")); err != nil {
		return err
	}

	definitionSource := filepath.Join(scratchDir, ingestAgentsRelativeDir, config.ID+".md")
	definitionDest := filepath.Join(runDir, ingestAgentsRelativeDir, config.ID+".md")
	if err := os.MkdirAll(filepath.Dir(definitionDest), 0o755); err != nil {
		return fmt.Errorf("ingest: create %s: %w", filepath.Dir(definitionDest), err)
	}
	if err := copyFileInto(definitionSource, definitionDest); err != nil {
		return err
	}

	if len(config.Allowed) > 0 {
		if err := copyFileInto(filepath.Join(scratchDir, ingestServerDeclarationFilename), filepath.Join(runDir, "mcp.json")); err != nil {
			return err
		}
	}
	if config.Annex != "" {
		for _, name := range []string{ladder.AnnexTextFilename, ladder.AnnexMetaFilename} {
			if err := copyFileInto(filepath.Join(scratchDir, name), filepath.Join(runDir, name)); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFileInto copies src's bytes to dst.
func copyFileInto(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("ingest: read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("ingest: write %s: %w", dst, err)
	}
	return nil
}

// writeIngestJSONDocument marshals doc as indented JSON with a trailing newline to path.
func writeIngestJSONDocument(path string, doc any) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("ingest: marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("ingest: write %s: %w", path, err)
	}
	return nil
}

// assistantRecordText concatenates the text of every "text" content block in one assistant record's
// message, in order.
func assistantRecordText(record ladder.Record) string {
	var text strings.Builder
	for _, block := range record.Message.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	return text.String()
}

// parseFinalAnswer parses the answer from the last fenced json block of the final assistant record in
// records, taking ladder.ExtractFencedJSON's inner half.
func parseFinalAnswer(records []ladder.Record) (map[string]any, error) {
	assistants := ladder.AssistantRecords(records)
	if len(assistants) == 0 {
		return nil, fmt.Errorf("ingest: transcript carries no assistant record; cannot parse an answer")
	}
	final := assistants[len(assistants)-1]

	_, inner, err := ladder.ExtractFencedJSON(assistantRecordText(final), "last")
	if err != nil {
		return nil, fmt.Errorf("ingest: parse final answer: %w", err)
	}
	var answer map[string]any
	if err := json.Unmarshal([]byte(inner), &answer); err != nil {
		return nil, fmt.Errorf("ingest: parse final answer: decode fenced json: %w", err)
	}
	return answer, nil
}

// ingestOutcome classifies report as "ingested" (no fatal finding), "truncated" (a fatal max_turns
// finding), or "failed" (any other fatal finding) -- the same truncated-outcome semantics the retired
// claude -p client's own --max-turns flag used to produce, now evaluated post hoc against the finished
// transcript (see internal/ladder/gates.go's GateMaxTurns doc comment).
func ingestOutcome(report ladder.GateReport) string {
	fatal := report.FatalFindings()
	if len(fatal) == 0 {
		return "ingested"
	}
	for _, finding := range fatal {
		if finding.Gate == "max_turns" {
			return "truncated"
		}
	}
	return "failed"
}

// fatalFindingsSummary joins report's fatal findings as "<gate>: <message>" pairs, for printing on a
// truncated or failed outcome.
func fatalFindingsSummary(report ladder.GateReport) string {
	fatal := report.FatalFindings()
	messages := make([]string, len(fatal))
	for i, finding := range fatal {
		messages[i] = fmt.Sprintf("%s: %s", finding.Gate, finding.Message)
	}
	return strings.Join(messages, "; ")
}
