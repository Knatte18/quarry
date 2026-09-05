// main.go is the harness's thin entry point: it parses flags for exactly three subcommands, run,
// report and pack, wires them onto the ladder package's exported entry points, and exits. All logic
// lives in package ladder; this file does nothing else. V1's twelve subcommands existed because an
// external orchestrator drove the loop one step at a time -- this harness drives its own loop, so a
// step of that loop would still have nothing left for a subcommand to do. pack is not a step of the
// run loop, though: it produces an artefact -- the kick-start pack -- that the prompt consumes before
// the loop ever starts, which is exactly the distinction that keeps run and report at two and pack
// off the list V1's argument was about.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ladder <run|report|pack> [flags]")
		os.Exit(1)
	}

	var (
		exitNonZero bool
		err         error
	)
	switch os.Args[1] {
	case "run":
		exitNonZero, err = runCommand(os.Args[2:])
	case "report":
		exitNonZero, err = reportCommand(os.Args[2:])
	case "pack":
		err = packCommand(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "usage: ladder <run|report|pack> [flags], got subcommand %q\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if exitNonZero {
		os.Exit(1)
	}
}

// runCommand parses the run subcommand's flags, resolves the quarry repository root from the
// process's own working directory, drives ladder.Run, then summarises the results root, writes the
// summary, and prints and writes the table -- in that order, so a run that ends with an incomplete
// cell or a blinding failure still leaves a summary and table describing exactly what happened.
func runCommand(args []string) (exitNonZero bool, err error) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "", "the ladder file (required)")
	resultsRoot := fs.String("results", "", "the results root (required)")
	cells := fs.String("cells", "", "a comma-separated cell-id list (optional)")
	reps := fs.Int("reps", 0, "a repetition override (optional)")
	claudeBin := fs.String("claude-bin", "claude", "the claude binary path (optional)")
	if err := fs.Parse(args); err != nil {
		return false, fmt.Errorf("run: %w", err)
	}
	if *configPath == "" {
		return false, fmt.Errorf("run: --config is required")
	}
	if *resultsRoot == "" {
		return false, fmt.Errorf("run: --results is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("run: %w", err)
	}
	quarryRepoRoot, err := ladder.ResolveQuarryRepoRoot(cwd)
	if err != nil {
		return false, fmt.Errorf("run: %w", err)
	}

	opts := ladder.RunOptions{
		LadderFilePath:  *configPath,
		ResultsRoot:     *resultsRoot,
		SelectedCells:   splitNonEmpty(*cells),
		RepsOverride:    *reps,
		ClaudeBinPath:   *claudeBin,
		QuarryRepoStart: quarryRepoRoot,
		Runner:          ladder.ExecRunner{},
	}

	runExitNonZero, err := ladder.Run(context.Background(), opts)
	if err != nil {
		return false, fmt.Errorf("run: %w", err)
	}

	reportExitNonZero, err := summarizeAndReport(*resultsRoot)
	if err != nil {
		return false, fmt.Errorf("run: %w", err)
	}

	return runExitNonZero || reportExitNonZero, nil
}

// reportCommand parses the report subcommand's flags and re-derives the summary and the table from
// the raw tree without running or scoring anything. It takes no ladder-file flag: a results root is
// self-describing, since every repetition's state file carries its own cell metadata.
func reportCommand(args []string) (exitNonZero bool, err error) {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	resultsRoot := fs.String("results", "", "the results root (required)")
	if err := fs.Parse(args); err != nil {
		return false, fmt.Errorf("report: %w", err)
	}
	if *resultsRoot == "" {
		return false, fmt.Errorf("report: --results is required")
	}

	exitNonZero, err = summarizeAndReport(*resultsRoot)
	if err != nil {
		return false, fmt.Errorf("report: %w", err)
	}
	return exitNonZero, nil
}

// packCommand parses the pack subcommand's flags, resolves the quarry repository root from the
// process's own working directory exactly as runCommand does, and drives ladder.Pack. It returns no
// non-zero exit signal of its own: a pack either succeeds or errors, with no partial or degraded
// outcome for the caller to report separately.
func packCommand(args []string) error {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	configPath := fs.String("config", "", "the ladder file (required)")
	resultsRoot := fs.String("results", "", "the results root (required)")
	claudeBin := fs.String("claude-bin", "claude", "the claude binary path (optional)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("pack: %w", err)
	}
	if *configPath == "" {
		return fmt.Errorf("pack: --config is required")
	}
	if *resultsRoot == "" {
		return fmt.Errorf("pack: --results is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}
	quarryRepoRoot, err := ladder.ResolveQuarryRepoRoot(cwd)
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}

	opts := ladder.PackOptions{
		LadderFilePath:  *configPath,
		ResultsRoot:     *resultsRoot,
		ClaudeBinPath:   *claudeBin,
		QuarryRepoStart: quarryRepoRoot,
		Runner:          ladder.ExecRunner{},
	}

	if err := ladder.Pack(context.Background(), opts); err != nil {
		return fmt.Errorf("pack: %w", err)
	}
	return nil
}

// summarizeAndReport re-derives the summary and the table for resultsRoot, writes both, prints the
// table to standard output, and reports whether the summary carries an incomplete or an invalid cell
// -- re-deriving without rewriting would leave a stale summary beside a fresh table, which is the
// opposite of what both subcommands exist for.
func summarizeAndReport(resultsRoot string) (exitNonZero bool, err error) {
	summary, err := ladder.Summarize(resultsRoot)
	if err != nil {
		return false, err
	}
	if err := ladder.WriteSummary(resultsRoot, summary); err != nil {
		return false, err
	}

	prov, err := ladder.ReadProvenance(resultsRoot)
	if err != nil {
		return false, err
	}
	if prov == nil {
		return false, fmt.Errorf("missing %s in %s", ladder.ProvenanceFile, resultsRoot)
	}

	table := ladder.RenderTable(summary, prov)
	if err := ladder.WriteTable(resultsRoot, table); err != nil {
		return false, err
	}
	fmt.Print(table)

	return len(summary.Incomplete) > 0 || len(summary.Invalid) > 0, nil
}

// splitNonEmpty splits s on commas and drops empty entries, so an unset --cells flag yields a nil
// slice rather than a slice holding one empty string.
func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, entry := range strings.Split(s, ",") {
		if entry != "" {
			out = append(out, entry)
		}
	}
	return out
}
