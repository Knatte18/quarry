package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// ingestTestModel is the pinned run model used by every ingest fixture in this file -- an id
// ladder.ModelAlias maps successfully, since GrantedToolsFromDefinition and RunAgentDefinition both need
// a resolvable model.
const ingestTestModel = "claude-opus-5"

// runGitCommand runs git with args in dir, failing the test on a non-zero exit.
func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v: %s", args, dir, err, output)
	}
}

// initGitRepoFixture initialises a minimal git repository at dir with one committed file, so
// ladder.ObserveWorktreeDirtied has a real `git status --porcelain` to run against.
func initGitRepoFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitCommand(t, dir, "init", "--quiet")
	runGitCommand(t, dir, "config", "user.email", "test@example.com")
	runGitCommand(t, dir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "committed.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write committed.txt: %v", err)
	}
	runGitCommand(t, dir, "add", "committed.txt")
	runGitCommand(t, dir, "commit", "--quiet", "-m", "initial")
	return dir
}

// ingestFixture bundles everything one ingest-command test needs: a ladder pinned for ingest, its
// results root, its config, the rep this attempt belongs to, the scratch and projects-root directories
// LocateTranscript searches, and the worktree ObserveWorktreeDirtied inspects.
type ingestFixture struct {
	l            *ladder.Ladder
	config       ladder.LadderConfig
	rep          int
	resultsRoot  string
	scratchDir   string
	projectsRoot string
	worktree     string
}

// newIngestFixture loads the committed ladder.yaml, pins run_model/max_turns/run_effort/scorer, points
// config's task worktree at a fresh real git repository, and prepares config's scratch directory with a
// settings document and a run agent definition -- everything ingest reads before it ever touches the
// transcript.
func newIngestFixture(t *testing.T, configID string, maxTurns int) *ingestFixture {
	t.Helper()
	l := mustLoadLadderFixture(t)
	l.SessionDirTemplate = filepath.Join(t.TempDir(), "session-{config_id}-{n}")
	runModel := ingestTestModel
	l.RunModel = &runModel
	max := maxTurns
	l.MaxTurns = &max
	l.RunEffort = "medium"
	l.Scorer.Model = ingestTestModel
	l.Scorer.Effort = "high"

	config, err := ladder.ConfigByID(l, configID)
	if err != nil {
		t.Fatalf("ConfigByID(%q): %v", configID, err)
	}
	rep := 1

	worktree := initGitRepoFixture(t)
	task := l.Tasks[config.Task]
	task.Worktree = worktree
	l.Tasks[config.Task] = task

	resultsRoot := t.TempDir()
	scratchDir, err := ladder.SessionDir(l, config.ID, rep)
	if err != nil {
		t.Fatalf("SessionDir: %v", err)
	}
	if err := ladder.WriteSettings(l, config, mustMkdirAndJoin(t, scratchDir, ".claude", "settings.json")); err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}
	name, body, err := ladder.RunAgentDefinition(l, config, runModel)
	if err != nil {
		t.Fatalf("RunAgentDefinition: %v", err)
	}
	definitionPath := mustMkdirAndJoin(t, scratchDir, ".claude", "agents", name+".md")
	if err := os.WriteFile(definitionPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write agent definition: %v", err)
	}
	if len(config.Allowed) > 0 {
		mcpPath := mustMkdirAndJoin(t, scratchDir, ".mcp.json")
		if err := os.WriteFile(mcpPath, []byte(`{}`), 0o644); err != nil {
			t.Fatalf("write mcp.json: %v", err)
		}
	}

	return &ingestFixture{
		l:            l,
		config:       config,
		rep:          rep,
		resultsRoot:  resultsRoot,
		scratchDir:   scratchDir,
		projectsRoot: t.TempDir(),
		worktree:     worktree,
	}
}

// mustMkdirAndJoin creates the parent directory of filepath.Join(base...) and returns that joined path.
func mustMkdirAndJoin(t *testing.T, base ...string) string {
	t.Helper()
	path := filepath.Join(base...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	return path
}

// writeIngestFixtureTranscript writes turnCount assistant records (each reporting ingestTestModel and one
// tool_use/tool_result pair when includeToolUse is true) plus a matching subagent metadata file under
// fixture's projectsRoot, at the description this attempt's ladder.DispatchDescription/ladder.NextAttempt
// derive. The final assistant record's text carries answerText's fenced json block.
func writeIngestFixtureTranscript(t *testing.T, fixture *ingestFixture, turnCount int, answerText string) {
	t.Helper()

	attempt, err := ladder.NextAttempt(fixture.resultsRoot, fixture.config.ID, fixture.rep)
	if err != nil {
		t.Fatalf("NextAttempt: %v", err)
	}
	description := ladder.DispatchDescription(fixture.config.ID, fixture.rep, attempt)

	subagentsDir := filepath.Join(fixture.projectsRoot, mangleProjectDirForTest(fixture.scratchDir), "sess-1", "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", subagentsDir, err)
	}

	metaPath := filepath.Join(subagentsDir, "agent-fixture.meta.json")
	metaData, err := json.Marshal(map[string]any{"description": description})
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	var lines []string
	base := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	for i := 0; i < turnCount; i++ {
		record := ladder.Record{
			IsSidechain: true,
			AgentID:     "fixture",
			UUID:        "uuid-" + string(rune('a'+i)),
			Timestamp:   base.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano),
			Type:        "assistant",
			Message: ladder.Message{
				Model:   ingestTestModel,
				Content: []ladder.ContentBlock{{Type: "text", Text: "thinking"}},
			},
		}
		if i == turnCount-1 {
			record.Message.Content = []ladder.ContentBlock{{Type: "text", Text: answerText}}
		}
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal record: %v", err)
		}
		lines = append(lines, string(data))
	}

	transcriptPath := filepath.Join(subagentsDir, "agent-fixture.jsonl")
	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
}

// mangleProjectDirForTest duplicates ladder's own unexported mangleProjectDir, since this test lives in
// a different package: replaces every path separator in cwd with a hyphen.
func mangleProjectDirForTest(cwd string) string {
	return strings.ReplaceAll(cwd, string(filepath.Separator), "-")
}

func TestRunIngest_SuccessWritesUsageAnswerAndIngestMarker(t *testing.T) {
	fixture := newIngestFixture(t, "a0-none", 10)
	writeIngestFixtureTranscript(t, fixture, 3, "```json\n{\"summary\": \"ok\"}\n```\n")

	var out bytes.Buffer
	err := runIngest(&out, fixture.l, repoRootFixture, fixture.resultsRoot, fixture.config.ID, fixture.rep, fixture.projectsRoot, 2*time.Second)
	if err != nil {
		t.Fatalf("runIngest() error = %v; want nil", err)
	}
	if !strings.Contains(out.String(), "ingested") {
		t.Errorf("runIngest() output = %q; want it to report ingested", out.String())
	}

	runDir := ladder.RunDirPath(fixture.resultsRoot, fixture.config.ID, fixture.rep)
	if !ladder.HasIngest(runDir) {
		t.Error("runIngest() success path did not write ingest.json")
	}

	usageData, err := os.ReadFile(filepath.Join(runDir, "usage.json"))
	if err != nil {
		t.Fatalf("read usage.json: %v", err)
	}
	var usage map[string]any
	if err := json.Unmarshal(usageData, &usage); err != nil {
		t.Fatalf("unmarshal usage.json: %v", err)
	}
	if usage["num_turns"].(float64) != 3 {
		t.Errorf("usage.json num_turns = %v; want 3", usage["num_turns"])
	}

	answerData, err := os.ReadFile(filepath.Join(runDir, "answer.json"))
	if err != nil {
		t.Fatalf("read answer.json: %v", err)
	}
	var answer map[string]any
	if err := json.Unmarshal(answerData, &answer); err != nil {
		t.Fatalf("unmarshal answer.json: %v", err)
	}
	if answer["summary"] != "ok" {
		t.Errorf("answer.json summary = %v; want ok", answer["summary"])
	}
}

func TestRunIngest_TruncatedOutcomeWhenTurnsExceedMaxTurns(t *testing.T) {
	fixture := newIngestFixture(t, "a0-none", 2)
	writeIngestFixtureTranscript(t, fixture, 3, "```json\n{\"summary\": \"ok\"}\n```\n")

	var out bytes.Buffer
	err := runIngest(&out, fixture.l, repoRootFixture, fixture.resultsRoot, fixture.config.ID, fixture.rep, fixture.projectsRoot, 2*time.Second)
	if err != nil {
		t.Fatalf("runIngest() error = %v; want nil", err)
	}
	if !strings.Contains(out.String(), "truncated") {
		t.Errorf("runIngest() output = %q; want it to report truncated", out.String())
	}

	runDir := ladder.RunDirPath(fixture.resultsRoot, fixture.config.ID, fixture.rep)
	if ladder.HasIngest(runDir) {
		t.Error("runIngest() truncated outcome must not write ingest.json")
	}
}

func TestRunIngest_FailedOutcomeOnDeniedToolUse(t *testing.T) {
	fixture := newIngestFixture(t, "a0-none", 10)

	attempt, err := ladder.NextAttempt(fixture.resultsRoot, fixture.config.ID, fixture.rep)
	if err != nil {
		t.Fatalf("NextAttempt: %v", err)
	}
	description := ladder.DispatchDescription(fixture.config.ID, fixture.rep, attempt)

	subagentsDir := filepath.Join(fixture.projectsRoot, mangleProjectDirForTest(fixture.scratchDir), "sess-1", "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", subagentsDir, err)
	}
	metaData, _ := json.Marshal(map[string]any{"description": description})
	if err := os.WriteFile(filepath.Join(subagentsDir, "agent-fixture.meta.json"), metaData, 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	// a0-none exposes no quarry tools, so mcp__quarry__toc_file is denied outright; a tool_use block
	// naming it with no matching error tool_result trips GateDeniedToolsNotUsed.
	record := ladder.Record{
		IsSidechain: true,
		AgentID:     "fixture",
		UUID:        "uuid-a",
		Timestamp:   time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		Type:        "assistant",
		Message: ladder.Message{
			Model: ingestTestModel,
			Content: []ladder.ContentBlock{
				{Type: "text", Text: "```json\n{\"summary\": \"ok\"}\n```\n"},
				{Type: "tool_use", ToolUseID: "call-1", Name: "mcp__quarry__toc_file", Input: map[string]any{}},
			},
		},
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subagentsDir, "agent-fixture.jsonl"), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	var out bytes.Buffer
	err = runIngest(&out, fixture.l, repoRootFixture, fixture.resultsRoot, fixture.config.ID, fixture.rep, fixture.projectsRoot, 2*time.Second)
	if err != nil {
		t.Fatalf("runIngest() error = %v; want nil", err)
	}
	if !strings.Contains(out.String(), "failed") {
		t.Errorf("runIngest() output = %q; want it to report failed", out.String())
	}

	runDir := ladder.RunDirPath(fixture.resultsRoot, fixture.config.ID, fixture.rep)
	if ladder.HasIngest(runDir) {
		t.Error("runIngest() failed outcome must not write ingest.json")
	}
}

func TestRunIngest_SingleFlightViolationErrorsBeforeAnyFileIsCopied(t *testing.T) {
	fixture := newIngestFixture(t, "a0-none", 10)
	fixture.rep = 2 // rep 1 has neither an ingest marker, a run.json, nor an exhausted attempt record.

	var out bytes.Buffer
	err := runIngest(&out, fixture.l, repoRootFixture, fixture.resultsRoot, fixture.config.ID, fixture.rep, fixture.projectsRoot, 10*time.Millisecond)
	if err == nil {
		t.Fatal("runIngest() error = nil; want a single-flight error")
	}

	runDir := ladder.RunDirPath(fixture.resultsRoot, fixture.config.ID, fixture.rep)
	if _, statErr := os.Stat(runDir); statErr == nil {
		t.Errorf("runIngest() created %s before failing the single-flight check", runDir)
	}
}

func TestRunIngest_DirtinessObservationTakenBeforeAnySubsequentRestore(t *testing.T) {
	fixture := newIngestFixture(t, "a0-none", 10)
	if err := os.WriteFile(filepath.Join(fixture.worktree, "scratch.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("dirty worktree: %v", err)
	}
	writeIngestFixtureTranscript(t, fixture, 2, "```json\n{\"summary\": \"ok\"}\n```\n")

	var out bytes.Buffer
	if err := runIngest(&out, fixture.l, repoRootFixture, fixture.resultsRoot, fixture.config.ID, fixture.rep, fixture.projectsRoot, 2*time.Second); err != nil {
		t.Fatalf("runIngest() error = %v; want nil", err)
	}

	runDir := ladder.RunDirPath(fixture.resultsRoot, fixture.config.ID, fixture.rep)
	rec, err := ladder.ReadIngestRecord(runDir)
	if err != nil {
		t.Fatalf("ReadIngestRecord: %v", err)
	}
	found := false
	for _, obs := range rec.Observations {
		if obs.Gate == "worktree_dirtied" && strings.Contains(obs.Message, "true") {
			found = true
		}
	}
	if !found {
		t.Errorf("ingest.json observations = %+v; want a worktree_dirtied: true observation taken before this simulated restore", rec.Observations)
	}

	// Simulate a caller restoring the worktree only after ingest returned -- the marker must already
	// carry the dirtied-true observation ingest took before this point, proving the observation was not
	// deferred to (or re-derived at) write time.
	if err := ladder.RestoreWorktree(fixture.worktree, ladder.RunGit); err != nil {
		t.Fatalf("RestoreWorktree: %v", err)
	}
	recAfter, err := ladder.ReadIngestRecord(runDir)
	if err != nil {
		t.Fatalf("ReadIngestRecord after restore: %v", err)
	}
	if len(recAfter.Observations) != len(rec.Observations) {
		t.Errorf("ingest.json observations changed after a later restore; want the on-disk marker unaffected")
	}
}
