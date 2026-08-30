// recordscore.go implements ladderbench's record-score subcommand: the CLI-level caller that validates a
// scoring session's reply, stamps it, writes score.json, and -- once the run's artifact set is complete
// -- writes run.json last.

package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func recordScoreCommand() *cobra.Command {
	var configID string
	var rep int

	cmd := &cobra.Command{
		Use:   "record-score",
		Short: "validate a scorer reply, write score.json, and write run.json once artifacts are complete",
		Long: `record-score reads a scorer reply from standard input, validates it against
--config-id's task schema (ladder.ParseScorerReply), and stamps the pinned scorer model, effort, and
prompt template (the task schema the template was chosen from) into the score record before writing
score.json. Nothing is written when the reply fails validation.

It then runs the complete-artifacts gate (ladder.GateRunCompleteArtifacts) and, only once it passes,
writes run.json last -- assembling the run marker's payload by reading the run directory's own
ingest.json (ladder.ReadIngestRecord) and passing it to ladder.RunJSONPayload. This is the only path by
which an observation taken in the run session reaches the marker the summariser and the cold-cell
disposition both read; without it, every observation would be stranded in the run session that took it.

run.json is written last and remains the sole definition of a complete run (ladder.IsComplete): a
directory carrying score.json but not run.json is, by that same definition, not yet complete -- exactly
the state a failing artifacts gate leaves it in.

record-score enforces the full pin set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := resolveLadder(cmd)
			if err != nil {
				return err
			}
			if err := ladder.RequirePins(l); err != nil {
				return err
			}
			resultsRoot, err := resolveResultsRoot(cmd)
			if err != nil {
				return err
			}
			if configID == "" {
				return fmt.Errorf("record-score: --config-id is required")
			}
			if !cmd.Flags().Changed("rep") {
				return fmt.Errorf("record-score: --rep is required")
			}
			reply, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("record-score: read scorer reply from standard input: %w", err)
			}
			return runRecordScore(cmd.OutOrStdout(), l, resultsRoot, configID, rep, string(reply))
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "the LadderConfig id this scorer reply belongs to (required)")
	cmd.Flags().IntVar(&rep, "rep", 0, "1-based repetition index (required)")

	return cmd
}

// runRecordScore validates reply against configID's task schema, writes score.json, and -- only once
// the run's artifact set is complete -- writes run.json.
func runRecordScore(out io.Writer, l *ladder.Ladder, resultsRoot, configID string, rep int, reply string) error {
	config, err := ladder.ConfigByID(l, configID)
	if err != nil {
		return err
	}
	task, ok := l.Tasks[config.Task]
	if !ok {
		return fmt.Errorf("record-score: config %q references unknown task %q", config.ID, config.Task)
	}

	record, err := ladder.ParseScorerReply(reply, task.Schema)
	if err != nil {
		return err
	}
	record["model"] = l.Scorer.Model
	record["effort"] = l.Scorer.Effort
	record["prompt_template"] = task.Schema

	runDir := ladder.RunDirPath(resultsRoot, configID, rep)
	if err := writeIngestJSONDocument(filepath.Join(runDir, "score.json"), record); err != nil {
		return err
	}

	if findings := ladder.GateRunCompleteArtifacts(runDir); len(findings) > 0 {
		fmt.Fprintf(out, "record-score: wrote score.json for config %q rep %d, but run.json is withheld -- artifacts incomplete: %s\n", configID, rep, findingsSummary(findings))
		return nil
	}

	ingestRecord, err := ladder.ReadIngestRecord(runDir)
	if err != nil {
		return err
	}
	if _, err := ladder.WriteRunJSON(runDir, ladder.RunJSONPayload(ingestRecord, *l.RunModel)); err != nil {
		return err
	}

	fmt.Fprintf(out, "record-score: scored config %q rep %d; run complete\n", configID, rep)
	return nil
}

// findingsSummary joins findings as "<gate>: <message>" pairs.
func findingsSummary(findings []ladder.GateFinding) string {
	messages := make([]string, len(findings))
	for i, finding := range findings {
		messages[i] = fmt.Sprintf("%s: %s", finding.Gate, finding.Message)
	}
	return strings.Join(messages, "; ")
}
