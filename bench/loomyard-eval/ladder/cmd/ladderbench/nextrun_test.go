package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// repoRootFixture is this package's directory's own repo-relative distance back to the repository
// root -- cmd/ladderbench sits five directories below it (bench/loomyard-eval/ladder/cmd/ladderbench) --
// so ladder.TaskTextFor/SchemaFor resolve the committed task files and README the same way they do when
// ladderbench is actually invoked from the repository root.
const repoRootFixture = "../../../../.."

func TestPrintNextRun_PendingPathPrintsRepAttemptPromptAndDefinitionName(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	var out bytes.Buffer
	if err := printNextRun(&out, l, repoRootFixture, resultsRoot, "a0-none"); err != nil {
		t.Fatalf("printNextRun() error = %v; want nil", err)
	}

	got := out.String()
	if !strings.Contains(got, "rep: 1") {
		t.Errorf("printNextRun() output = %q; want it to report rep 1", got)
	}
	if !strings.Contains(got, "attempt: 1") {
		t.Errorf("printNextRun() output = %q; want it to report attempt 1", got)
	}
	if !strings.Contains(got, "agent_definition: a0-none") {
		t.Errorf("printNextRun() output = %q; want it to report agent_definition a0-none", got)
	}
	if !strings.Contains(got, "prompt:") {
		t.Errorf("printNextRun() output = %q; want it to print the assembled prompt", got)
	}
}

func TestPrintNextRun_NothingPendingOnceEveryRepetitionIsIngested(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	for n := 1; n <= l.Reps; n++ {
		runDir := ladder.RunDirPath(resultsRoot, "a0-none", n)
		if err := ladder.WriteIngestJSON(runDir, ladder.IngestRecord{ConfigID: "a0-none", Rep: n, Attempt: 1}); err != nil {
			t.Fatalf("WriteIngestJSON(rep %d): %v", n, err)
		}
	}

	var out bytes.Buffer
	if err := printNextRun(&out, l, repoRootFixture, resultsRoot, "a0-none"); err != nil {
		t.Fatalf("printNextRun() error = %v; want nil", err)
	}
	if !strings.Contains(out.String(), "nothing pending") {
		t.Errorf("printNextRun() output = %q; want it to report nothing pending", out.String())
	}
}

func TestPrintNextRun_AttemptIndexAdvancesWithInvalidatedSiblingsPresent(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	runDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	invalidSibling := runDir + ".invalid-1"
	if err := os.MkdirAll(invalidSibling, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", invalidSibling, err)
	}

	var out bytes.Buffer
	if err := printNextRun(&out, l, repoRootFixture, resultsRoot, "a0-none"); err != nil {
		t.Fatalf("printNextRun() error = %v; want nil", err)
	}
	if !strings.Contains(out.String(), "attempt: 2") {
		t.Errorf("printNextRun() output = %q; want attempt 2 with one invalidated sibling present", out.String())
	}
}

func TestPrintNextScoring_FiltersToIngestedButNotYetScored(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	scoredDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	if err := ladder.WriteIngestJSON(scoredDir, ladder.IngestRecord{ConfigID: "a0-none", Rep: 1, Attempt: 1}); err != nil {
		t.Fatalf("WriteIngestJSON(scored): %v", err)
	}
	if _, err := ladder.WriteRunJSON(scoredDir, map[string]any{"config_id": "a0-none", "n": 1}); err != nil {
		t.Fatalf("WriteRunJSON(scored): %v", err)
	}

	pendingDir := ladder.RunDirPath(resultsRoot, "a0-none", 2)
	if err := ladder.WriteIngestJSON(pendingDir, ladder.IngestRecord{ConfigID: "a0-none", Rep: 2, Attempt: 1}); err != nil {
		t.Fatalf("WriteIngestJSON(pending): %v", err)
	}

	var out bytes.Buffer
	if err := printNextScoring(&out, l, resultsRoot); err != nil {
		t.Fatalf("printNextScoring() error = %v; want nil", err)
	}
	got := strings.TrimSpace(out.String())
	if got != pendingDir {
		t.Errorf("printNextScoring() = %q; want %q", got, pendingDir)
	}
}

func TestPrintNextScoring_NothingPendingWhenNoneIngested(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	var out bytes.Buffer
	if err := printNextScoring(&out, l, resultsRoot); err != nil {
		t.Fatalf("printNextScoring() error = %v; want nil", err)
	}
	if !strings.Contains(out.String(), "nothing pending") {
		t.Errorf("printNextScoring() output = %q; want it to report nothing pending", out.String())
	}
}

func TestTargetDirFor_ColdConfigUsesColdWorktreeTemplate(t *testing.T) {
	l := mustLoadLadderFixture(t)
	config, err := ladder.ConfigByID(l, "a5-bundle-cold")
	if err != nil {
		t.Fatalf("ConfigByID: %v", err)
	}
	got := targetDirFor(l, config, 2)
	want := coldWorktreeDir(l, 2)
	if got != want {
		t.Errorf("targetDirFor(cold, 2) = %q; want %q", got, want)
	}
}

func TestTargetDirFor_WarmConfigUsesTaskWorktree(t *testing.T) {
	l := mustLoadLadderFixture(t)
	config, err := ladder.ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID: %v", err)
	}
	got := targetDirFor(l, config, 1)
	want := l.Tasks[config.Task].Worktree
	if got != want {
		t.Errorf("targetDirFor(warm, 1) = %q; want %q", got, want)
	}
}

func TestPairsForConfig_UnknownConfigErrors(t *testing.T) {
	l := mustLoadLadderFixture(t)
	if _, err := pairsForConfig(l, "does-not-exist"); err == nil {
		t.Error("pairsForConfig() = _, nil; want an error for an unknown config id")
	}
}
