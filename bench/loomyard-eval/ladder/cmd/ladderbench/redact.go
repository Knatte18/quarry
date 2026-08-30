// redact.go implements ladderbench's redact subcommand: the CLI-level caller that writes a run's
// redacted answer beside the original and prints the assembled scorer prompt for the scoring session to
// dispatch.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func redactCommand() *cobra.Command {
	var configID string
	var rep int

	cmd := &cobra.Command{
		Use:   "redact",
		Short: "write a run's redacted answer and print the assembled scorer prompt",
		Long: `redact reads --config-id's --rep-th run's answer.json, writes the redacted copy beside it
as answer.redacted.json (ladder.WriteRedacted), and prints the assembled scorer prompt
(ladder.BuildScorerPrompt) on standard output for the scoring session to dispatch.

It resolves the ladder-declared answer-key (the task's fasit) and task-text paths against the repository
root the root command resolves, never against the process working directory -- matching how
ladder.TaskTextFor and ladder.SchemaFor take that same root.

The printed prompt embeds the task's unstripped answer key -- this is why redact is only ever run inside
the dedicated scoring session and never in a session that also hosts a run agent: a run agent that saw
this output would see the very answer it is being scored against.

redact enforces the full pin set.`,
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
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			if configID == "" {
				return fmt.Errorf("redact: --config-id is required")
			}
			if !cmd.Flags().Changed("rep") {
				return fmt.Errorf("redact: --rep is required")
			}
			return runRedact(cmd.OutOrStdout(), l, repoRoot, resultsRoot, configID, rep)
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "the LadderConfig id whose run to redact and score (required)")
	cmd.Flags().IntVar(&rep, "rep", 0, "1-based repetition index (required)")

	return cmd
}

// runRedact writes configID's rep-th run's redacted answer and prints the assembled scorer prompt to
// out.
func runRedact(out io.Writer, l *ladder.Ladder, repoRoot, resultsRoot, configID string, rep int) error {
	config, err := ladder.ConfigByID(l, configID)
	if err != nil {
		return err
	}
	task, ok := l.Tasks[config.Task]
	if !ok {
		return fmt.Errorf("redact: config %q references unknown task %q", config.ID, config.Task)
	}

	runDir := ladder.RunDirPath(resultsRoot, configID, rep)
	redactedAnswer, err := ladder.WriteRedacted(runDir)
	if err != nil {
		return err
	}

	fasitPath := filepath.Join(repoRoot, task.Fasit)
	fasitData, err := os.ReadFile(fasitPath)
	if err != nil {
		return fmt.Errorf("redact: read %s: %w", fasitPath, err)
	}
	var fasit map[string]any
	if err := json.Unmarshal(fasitData, &fasit); err != nil {
		return fmt.Errorf("redact: parse %s: %w", fasitPath, err)
	}

	taskText, err := ladder.TaskTextFor(l, repoRoot, config.Task)
	if err != nil {
		return err
	}

	prompt, err := ladder.BuildScorerPrompt(l, config, redactedAnswer, fasit, taskText)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, prompt)
	return nil
}
