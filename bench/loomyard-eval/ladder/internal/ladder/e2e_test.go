// e2e_test.go is the offline proof that the whole program works end to end: it drives Run against a
// synthetic ladder file, a synthetic task file and a synthetic fasit, all written to a temporary
// directory, with a small local git repository standing in for both the quarry repository and the
// loomyard target repository, and testdata/fakeclaude standing in for the claude binary. It covers
// the happy path, resume, a hard failure, a control-cell blinding failure and the single-run
// advisory lock -- everything except the guarded live smoke test in live_test.go, which needs the
// real CLI.

package ladder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// fakeClaudeBinOnce and fakeClaudeBinPath cache the one build of testdata/fakeclaude every subtest
// of TestE2E shares, so the binary is built exactly once per test run rather than once per subtest.
var (
	fakeClaudeBinOnce sync.Once
	fakeClaudeBinPath string
)

// buildFakeClaudeOnce builds testdata/fakeclaude into dir (the outer TestE2E's own t.TempDir()) the
// first time it is called and returns that path on every call thereafter.
func buildFakeClaudeOnce(t *testing.T, dir string) string {
	t.Helper()
	fakeClaudeBinOnce.Do(func() {
		fakeClaudeBinPath = filepath.Join(dir, "fakeclaude")
		cmd := exec.Command("go", "build", "-o", fakeClaudeBinPath, "./testdata/fakeclaude")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build ./testdata/fakeclaude: %v\n%s", err, out)
		}
	})
	if fakeClaudeBinPath == "" {
		t.Fatal("fake claude binary was never built")
	}
	return fakeClaudeBinPath
}

// runGit runs one git command in dir and fails the test on a non-zero exit.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s (in %s): %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// initGitRepo initialises a fresh git repository under a new t.TempDir() and configures a throwaway
// commit identity, so a commit works regardless of the machine's own git configuration.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "e2e@example.com")
	runGit(t, dir, "config", "user.name", "e2e")
	return dir
}

// commitReadme writes a README into dir and commits it, returning the new commit's full SHA -- the
// single commit both the synthetic quarry repository and the synthetic target repository need.
func commitReadme(t *testing.T, dir string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("synthetic fixture repository\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "synthetic commit")
	return runGit(t, dir, "rev-parse", "HEAD")
}

// syntheticTaskFile writes a minimal task file at path satisfying LoadTaskFile's two required
// headings: a task-text blockquote and a fenced-json output schema.
func syntheticTaskFile(t *testing.T, path string) {
	t.Helper()
	content := "# Synthetic task\n\n" +
		"## `<TASK TEXT>`\n\n" +
		"> Describe the synthetic fixture in one short sentence.\n\n" +
		"## Output schema (exploration tasks)\n\n" +
		"```json\n" +
		"{\n" +
		"  \"relevant_files\": [\"a.go\"],\n" +
		"  \"key_symbols\": [],\n" +
		"  \"summary\": \"...\",\n" +
		"  \"confidence\": \"high|medium|low\",\n" +
		"  \"open_questions\": []\n" +
		"}\n" +
		"```\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write synthetic task file: %v", err)
	}
}

// syntheticFasit writes a minimal fasit file at path: a "_meta" block BuildScorerPrompt strips, plus
// a couple of ordinary fields.
func syntheticFasit(t *testing.T, path string) {
	t.Helper()
	content := `{"_meta":{"note":"synthetic fixture, not a real fasit"},"relevant_files":["a.go"],"summary":"synthetic fasit summary"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write synthetic fasit: %v", err)
	}
}

// writeSyntheticLadderFile marshals l with gopkg.in/yaml.v3 -- the same library LoadLadder decodes with -- and
// writes it to path.
func writeSyntheticLadderFile(t *testing.T, path string, l *Ladder) {
	t.Helper()
	data, err := yaml.Marshal(l)
	if err != nil {
		t.Fatalf("marshal synthetic ladder file: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write synthetic ladder file: %v", err)
	}
}

// e2eEnv is one subtest's fully wired-up harness environment: a synthetic quarry repository, a
// synthetic target (loomyard) repository, a worktree root and the ladder file's own on-disk paths.
type e2eEnv struct {
	quarryRepoRoot string
	targetRepoPath string
	worktreeRoot   string
	resultsRoot    string
	claudeBinPath  string
	taskFilePath   string
	fasitPath      string
}

// newE2EEnv wires up one subtest's environment: a fresh synthetic quarry repository and target
// repository, each with one commit, the worktree root and results root under fresh t.TempDir()s, and
// the environment variables Run's own resolvers read. It returns the pinned SHA of the target
// repository's one commit alongside the environment, since every ladder file this test writes needs
// it as a task's pinned_sha.
func newE2EEnv(t *testing.T, claudeBinPath string) (*e2eEnv, string) {
	t.Helper()

	quarryRepoRoot := initGitRepo(t)
	commitReadme(t, quarryRepoRoot)

	targetRepoPath := initGitRepo(t)
	pinnedSHA := commitReadme(t, targetRepoPath)

	worktreeRoot := t.TempDir()
	t.Setenv("LADDER_WORKTREE_ROOT", worktreeRoot)
	t.Setenv("LADDER_LOOMYARD_REPO", targetRepoPath)

	taskDir := t.TempDir()
	taskFilePath := filepath.Join(taskDir, "task.md")
	syntheticTaskFile(t, taskFilePath)
	fasitPath := filepath.Join(taskDir, "fasit.json")
	syntheticFasit(t, fasitPath)

	resultsRoot := filepath.Join(t.TempDir(), "root")
	if err := os.MkdirAll(resultsRoot, 0o755); err != nil {
		t.Fatalf("mkdir results root: %v", err)
	}

	return &e2eEnv{
		quarryRepoRoot: quarryRepoRoot,
		targetRepoPath: targetRepoPath,
		worktreeRoot:   worktreeRoot,
		resultsRoot:    resultsRoot,
		claudeBinPath:  claudeBinPath,
		taskFilePath:   taskFilePath,
		fasitPath:      fasitPath,
	}, pinnedSHA
}

// baseLadder returns a *Ladder carrying every field common to this file's synthetic ladder files,
// leaving Configs for the caller to set.
func baseLadder(env *e2eEnv, pinnedSHA string, reps int, taskID string) *Ladder {
	return &Ladder{
		RunModel:    "fake-model",
		RunEffort:   "medium",
		MaxTurns:    4,
		Reps:        reps,
		Scorer:      ScorerSpec{Model: "fake-scorer-model", Effort: "high"},
		QuarryTools: nil,
		SourceRepo:  "env:LADDER_LOOMYARD_REPO",
		Tasks: map[string]Task{
			taskID: {
				TaskFile:  env.taskFilePath,
				PinnedSHA: pinnedSHA,
				Schema:    "exploration",
				Fasit:     env.fasitPath,
			},
		},
	}
}

// setFakeClaudeEnv sets every FAKE_CLAUDE_* environment variable testdata/fakeclaude asserts against,
// for a matrix of exclusively control cells (allowedEnv is always unset, controlEnv is always "1").
func setFakeClaudeEnv(t *testing.T, l *Ladder, stream string) {
	t.Helper()
	t.Setenv("FAKE_CLAUDE_MODEL", l.RunModel)
	t.Setenv("FAKE_CLAUDE_EFFORT", l.RunEffort)
	t.Setenv("FAKE_CLAUDE_MAX_TURNS", strconv.Itoa(l.MaxTurns))
	t.Setenv("FAKE_CLAUDE_TOOLS", strings.Join(BuiltinTools, ","))
	t.Setenv("FAKE_CLAUDE_CONTROL", "1")
	t.Setenv("FAKE_CLAUDE_SCORER_MODEL", l.Scorer.Model)
	t.Setenv("FAKE_CLAUDE_SCORER_EFFORT", l.Scorer.Effort)
	t.Setenv("FAKE_CLAUDE_STREAM", stream)
	t.Setenv("FAKE_CLAUDE_LEAK_PREFIX", l.MCPPrefix())
}

// setFakeClaudeEnvGranted is setFakeClaudeEnv for a granted (non-control) cell driving cfg: it sets
// FAKE_CLAUDE_ALLOWED to cfg's MCP-prefixed granted tool list and leaves FAKE_CLAUDE_CONTROL unset,
// so testdata/fakeclaude asserts the "--allowedTools" flag's value instead of asserting its absence.
func setFakeClaudeEnvGranted(t *testing.T, l *Ladder, cfg Config, stream string) {
	t.Helper()
	t.Setenv("FAKE_CLAUDE_MODEL", l.RunModel)
	t.Setenv("FAKE_CLAUDE_EFFORT", l.RunEffort)
	t.Setenv("FAKE_CLAUDE_MAX_TURNS", strconv.Itoa(l.MaxTurns))
	t.Setenv("FAKE_CLAUDE_TOOLS", strings.Join(BuiltinTools, ","))
	t.Setenv("FAKE_CLAUDE_SCORER_MODEL", l.Scorer.Model)
	t.Setenv("FAKE_CLAUDE_SCORER_EFFORT", l.Scorer.Effort)
	t.Setenv("FAKE_CLAUDE_STREAM", stream)
	t.Setenv("FAKE_CLAUDE_LEAK_PREFIX", l.MCPPrefix())

	prefixed := make([]string, len(cfg.Allowed))
	for i, a := range cfg.Allowed {
		prefixed[i] = l.MCPPrefix() + a
	}
	t.Setenv("FAKE_CLAUDE_ALLOWED", strings.Join(prefixed, ","))
}

// writeStandaloneServerModule writes a minimal, dependency-free Go module under quarryRepoRoot at
// buildTarget, so BuildServer's real `go build` invocation -- run through the same ExecRunner as
// production, never mocked -- has something real to build for a granted-cell e2e case.
func writeStandaloneServerModule(t *testing.T, quarryRepoRoot, buildTarget string) {
	t.Helper()
	dir := filepath.Join(quarryRepoRoot, buildTarget)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(quarryRepoRoot, "go.mod"), []byte("module fakeserver\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
}

// runOpts builds the RunOptions common to this file's subtests.
func runOpts(env *e2eEnv, ladderFilePath string, cells []string) RunOptions {
	return RunOptions{
		LadderFilePath:  ladderFilePath,
		ResultsRoot:     env.resultsRoot,
		SelectedCells:   cells,
		ClaudeBinPath:   env.claudeBinPath,
		QuarryRepoStart: env.quarryRepoRoot,
		Runner:          ExecRunner{},
	}
}

// summarizeAndWriteReport mirrors what cmd/ladder's own summarizeAndReport does, entirely through
// this package's exported entry points: it re-derives the summary, writes it, reads back the
// provenance record, renders the table and writes it, then reads the table back off disk so the
// caller can compare the rendered string against the written one byte for byte.
func summarizeAndWriteReport(t *testing.T, resultsRoot string) (*Summary, string, string) {
	t.Helper()
	summary, err := Summarize(resultsRoot)
	if err != nil {
		t.Fatalf("Summarize(%s) = %v; want no error", resultsRoot, err)
	}
	if err := WriteSummary(resultsRoot, summary); err != nil {
		t.Fatalf("WriteSummary() = %v; want no error", err)
	}
	prov, err := ReadProvenance(resultsRoot)
	if err != nil {
		t.Fatalf("ReadProvenance(%s) = %v; want no error", resultsRoot, err)
	}
	if prov == nil {
		t.Fatalf("ReadProvenance(%s) = nil; want a provenance record", resultsRoot)
	}
	rendered := RenderTable(summary, prov)
	if err := WriteTable(resultsRoot, rendered); err != nil {
		t.Fatalf("WriteTable() = %v; want no error", err)
	}
	writtenBytes, err := os.ReadFile(filepath.Join(resultsRoot, TableFile))
	if err != nil {
		t.Fatalf("read written table: %v", err)
	}
	return summary, rendered, string(writtenBytes)
}

// sixRepFiles is the six per-repetition file names the-six-per-repetition-filenames decision names,
// in the write order runstate.go's header comment states, state file last.
var sixRepFiles = []string{TranscriptFile, AnswerFile, RedactedAnswerFile, UsageFile, ScoreFile, RunStateFile}

// assertStateFileWrittenLast asserts every one of sixRepFiles exists under dir and that
// RunStateFile's own modification time is not earlier than any of the other five.
func assertStateFileWrittenLast(t *testing.T, dir string) {
	t.Helper()
	var stateModTime time.Time
	var others []time.Time
	for _, name := range sixRepFiles {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("stat %s/%s: %v", dir, name, err)
		}
		if name == RunStateFile {
			stateModTime = info.ModTime()
		} else {
			others = append(others, info.ModTime())
		}
	}
	for i, mt := range others {
		if stateModTime.Before(mt) {
			t.Errorf("run.json mtime %v is earlier than %s's mtime %v (index %d)", stateModTime, sixRepFiles[i], mt, i)
		}
	}
}

func TestE2E(t *testing.T) {
	fakeBinPath := buildFakeClaudeOnce(t, t.TempDir())

	t.Run("HappyPath", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.Configs = []Config{{ID: "cell-a", Ladder: "p", Task: "task-1", Allowed: nil}}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "normal")

		exitNonZero, err := Run(context.Background(), runOpts(env, ladderPath, nil))
		if err != nil {
			t.Fatalf("Run() = %v; want no error", err)
		}
		if exitNonZero {
			t.Error("Run() reported a non-zero exit for the happy path")
		}

		worktreePath := TaskWorktreePath(env.worktreeRoot, "task-1")
		if _, err := os.Stat(worktreePath); err != nil {
			t.Errorf("worktree %s was not prepared: %v", worktreePath, err)
		}

		mcpConfigPath := filepath.Join(env.quarryRepoRoot, ".scratch", "ladder", "cell-a-1.json")
		if _, err := os.Stat(mcpConfigPath); err != nil {
			t.Errorf("mcp config %s was not written: %v", mcpConfigPath, err)
		}

		repDir := RepDir(env.resultsRoot, "cell-a", 1)
		assertStateFileWrittenLast(t, repDir)

		if _, err := os.Stat(filepath.Join(repDir, InvalidReasonFile)); !os.IsNotExist(err) {
			t.Errorf("%s exists in a completed repetition's own directory; want it absent", InvalidReasonFile)
		}

		transcriptData, err := os.ReadFile(filepath.Join(repDir, TranscriptFile))
		if err != nil || len(transcriptData) == 0 {
			t.Errorf("transcript was not tee'd to %s: %v", repDir, err)
		}

		var metrics Metrics
		usageData, err := os.ReadFile(filepath.Join(repDir, UsageFile))
		if err != nil {
			t.Fatalf("read usage.json: %v", err)
		}
		if err := json.Unmarshal(usageData, &metrics); err != nil {
			t.Fatalf("decode usage.json: %v", err)
		}
		if metrics.NumTurns != 1 {
			t.Errorf("metrics.NumTurns = %d; want 1", metrics.NumTurns)
		}

		var score ScoreRecord
		scoreData, err := os.ReadFile(filepath.Join(repDir, ScoreFile))
		if err != nil {
			t.Fatalf("read score.json: %v", err)
		}
		if err := json.Unmarshal(scoreData, &score); err != nil {
			t.Fatalf("decode score.json: %v", err)
		}
		if scored, _ := score["scored"].(bool); !scored {
			// The absent "scored" key defaults to the zero value false via the type assertion,
			// exactly like a genuinely unscored record; a genuinely scored record has no "scored"
			// key at all, since ScoreRecord for a real reply is the scorer's own field set.
			if _, hasReason := score["reason"]; hasReason {
				t.Errorf("score.json = %v; want a real scorer reply, not the unscored stand-in", score)
			}
		}
		if recall, ok := score["recall"].(float64); !ok || recall != 1.0 {
			t.Errorf("score.json[recall] = %v; want 1.0", score["recall"])
		}

		summary, rendered, written := summarizeAndWriteReport(t, env.resultsRoot)
		if rendered != written {
			t.Error("the printed (rendered) table does not equal the table written to disk")
		}
		if len(summary.Incomplete) != 0 {
			t.Errorf("summary.Incomplete = %v; want none", summary.Incomplete)
		}
		if len(summary.Invalid) != 0 {
			t.Errorf("summary.Invalid = %v; want none", summary.Invalid)
		}
	})

	t.Run("Resume", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 2, "task-1")
		l.Configs = []Config{
			{ID: "cell-a", Ladder: "ra", Task: "task-1", Allowed: nil},
			{ID: "cell-b", Ladder: "rb", Task: "task-1", Allowed: nil},
		}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "normal")

		exitNonZero, err := Run(context.Background(), runOpts(env, ladderPath, []string{"cell-a"}))
		if err != nil {
			t.Fatalf("first Run() = %v; want no error", err)
		}
		if exitNonZero {
			t.Fatal("first Run() reported a non-zero exit")
		}

		type snapshot struct {
			modTime time.Time
			content []byte
		}
		before := map[string]snapshot{}
		for _, rep := range []int{1, 2} {
			dir := RepDir(env.resultsRoot, "cell-a", rep)
			path := filepath.Join(dir, TranscriptFile)
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat %s after first run: %v", path, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s after first run: %v", path, err)
			}
			before[path] = snapshot{modTime: info.ModTime(), content: data}
		}

		if _, err := os.Stat(RepDir(env.resultsRoot, "cell-b", 1)); !os.IsNotExist(err) {
			t.Fatalf("cell-b rep 1 exists before the second run: %v", err)
		}

		exitNonZero, err = Run(context.Background(), runOpts(env, ladderPath, nil))
		if err != nil {
			t.Fatalf("second Run() = %v; want no error", err)
		}
		if exitNonZero {
			t.Fatal("second Run() reported a non-zero exit")
		}

		for path, want := range before {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat %s after second run: %v", path, err)
			}
			if !info.ModTime().Equal(want.modTime) {
				t.Errorf("%s mtime changed across the second (resumed) run: %v -> %v", path, want.modTime, info.ModTime())
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s after second run: %v", path, err)
			}
			if string(data) != string(want.content) {
				t.Errorf("%s content changed across the second (resumed) run", path)
			}
		}

		for _, rep := range []int{1, 2} {
			dir := RepDir(env.resultsRoot, "cell-b", rep)
			if !RepIsComplete(dir) {
				t.Errorf("cell-b rep %d is not complete after the second run", rep)
			}
		}

		prov, err := ReadProvenance(env.resultsRoot)
		if err != nil {
			t.Fatalf("ReadProvenance() = %v", err)
		}
		wantCells := map[string]bool{"cell-a": true, "cell-b": true}
		if len(prov.SelectedCells) != len(wantCells) {
			t.Errorf("provenance.SelectedCells = %v; want the union %v", prov.SelectedCells, wantCells)
		}
		for _, id := range prov.SelectedCells {
			if !wantCells[id] {
				t.Errorf("provenance.SelectedCells contains unexpected id %q", id)
			}
		}
	})

	t.Run("Failure", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.Configs = []Config{{ID: "cell-fail", Ladder: "f", Task: "task-1", Allowed: nil}}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "partial_fail")

		exitNonZero, err := Run(context.Background(), runOpts(env, ladderPath, nil))
		if err != nil {
			t.Fatalf("Run() = %v; want no error even though the cell fails", err)
		}
		if !exitNonZero {
			t.Error("Run() reported a zero exit for a cell that never produced a valid answer")
		}

		dir := RepDir(env.resultsRoot, "cell-fail", 1)
		for n := 1; n <= MaxAttempts; n++ {
			target := dir + ".invalid-" + strconv.Itoa(n)
			if _, err := os.Stat(target); err != nil {
				t.Errorf("invalid directory %s was not produced: %v", target, err)
			}
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("repetition directory %s still exists; want it renamed away", dir)
		}

		summary, _, _ := summarizeAndWriteReport(t, env.resultsRoot)
		if len(summary.Incomplete) != 1 || summary.Incomplete[0] != "cell-fail" {
			t.Errorf("summary.Incomplete = %v; want [\"cell-fail\"]", summary.Incomplete)
		}
	})

	t.Run("Blinding", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.Configs = []Config{{ID: "cell-blind", Ladder: "y", Task: "task-1", Allowed: nil}}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "leak_prefix")

		exitNonZero, err := Run(context.Background(), runOpts(env, ladderPath, nil))
		if err != nil {
			t.Fatalf("first Run() = %v; want no error", err)
		}
		if !exitNonZero {
			t.Error("first Run() reported a zero exit for a blinding failure")
		}

		dir := RepDir(env.resultsRoot, "cell-blind", 1)
		state, err := ReadRunState(dir)
		if err != nil {
			t.Fatalf("ReadRunState(%s) = %v; want the discarded repetition still on disk", dir, err)
		}
		if state.State != "complete" || !state.BlindingFailed {
			t.Errorf("run state = %+v; want complete with blinding_failed set", state)
		}
		if RepIsComplete(dir) {
			t.Error("RepIsComplete() = true for a blinding-failed repetition; want false")
		}
		if _, err := os.Stat(dir + ".invalid-1"); !os.IsNotExist(err) {
			t.Error("a blinding failure was invalidated/retried; it must not be")
		}

		summary, _, _ := summarizeAndWriteReport(t, env.resultsRoot)
		if len(summary.Invalid) != 1 || summary.Invalid[0] != "cell-blind" {
			t.Errorf("summary.Invalid = %v; want [\"cell-blind\"]", summary.Invalid)
		}

		// Fix the cause and re-run: the second invocation must re-attempt the same repetition
		// rather than skip it.
		setFakeClaudeEnv(t, l, "normal")
		exitNonZero, err = Run(context.Background(), runOpts(env, ladderPath, nil))
		if err != nil {
			t.Fatalf("second Run() = %v; want no error", err)
		}
		if exitNonZero {
			t.Error("second Run() reported a non-zero exit after the cause was fixed")
		}
		if !RepIsComplete(dir) {
			t.Error("second Run() did not re-attempt the previously blinding-failed repetition")
		}
	})

	t.Run("GrantedCellServerNeverConnects", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.QuarryTools = []string{"toc"}
		l.Server = &ServerSpec{Name: "quarry", Build: "./cmd/fakeserver"}
		writeStandaloneServerModule(t, env.quarryRepoRoot, "cmd/fakeserver")

		granted := Config{ID: "cell-grant", Ladder: "g", Task: "task-1", Allowed: []string{"toc"}}
		control := Config{ID: "cell-grant-control", Ladder: "g", Task: "task-1", Allowed: nil}
		l.Configs = []Config{control, granted}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)

		setFakeClaudeEnvGranted(t, l, granted, "normal")
		t.Setenv("FAKE_CLAUDE_SERVER_STATUS_OVERRIDE", "quarry=failed")

		// Only the granted cell is selected: the control cell exists solely to satisfy
		// LoadLadder's one-control-per-letter rule and is never invoked.
		exitNonZero, err := Run(context.Background(), runOpts(env, ladderPath, []string{granted.ID}))
		if err != nil {
			t.Fatalf("Run() = %v; want no error even though the server never connects", err)
		}
		if !exitNonZero {
			t.Error("Run() reported a zero exit for a repetition whose granted server never connected")
		}

		dir := RepDir(env.resultsRoot, granted.ID, 1)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("repetition directory %s still exists; want it renamed away after exhausting attempts", dir)
		}
		for n := 1; n <= MaxAttempts; n++ {
			target := dir + ".invalid-" + strconv.Itoa(n)
			if _, err := os.Stat(target); err != nil {
				t.Errorf("invalid directory %s was not produced: %v", target, err)
				continue
			}
			reasonPath := filepath.Join(target, InvalidReasonFile)
			data, err := os.ReadFile(reasonPath)
			if err != nil {
				t.Errorf("reason file %s was not written: %v", reasonPath, err)
				continue
			}
			if !strings.Contains(string(data), granted.ID) || !strings.Contains(string(data), "quarry") {
				t.Errorf("reason file %s = %q; want it to name the cell and the server", reasonPath, data)
			}
			if !strings.Contains(string(data), "cause: "+CauseServerNotConnected) {
				t.Errorf("reason file %s = %q; want it to carry \"cause: %s\"", reasonPath, data, CauseServerNotConnected)
			}
		}

		summary, _, _ := summarizeAndWriteReport(t, env.resultsRoot)
		if len(summary.Incomplete) != 1 || summary.Incomplete[0] != granted.ID {
			t.Errorf("summary.Incomplete = %v; want [%q]", summary.Incomplete, granted.ID)
		}
	})

	t.Run("InvalidReasonRunnerError", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.Configs = []Config{{ID: "cell-runner-error", Ladder: "re", Task: "task-1", Allowed: nil}}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "partial_fail")

		exitNonZero, err := Run(context.Background(), runOpts(env, ladderPath, nil))
		if err != nil {
			t.Fatalf("Run() = %v; want no error", err)
		}
		if !exitNonZero {
			t.Error("Run() reported a zero exit for a cell that never produced a valid answer")
		}

		dir := RepDir(env.resultsRoot, "cell-runner-error", 1)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("repetition directory %s still exists; want it renamed away", dir)
		}
		for k := 1; k <= MaxAttempts; k++ {
			target := dir + ".invalid-" + strconv.Itoa(k)
			reason := readInvalidReason(t, target)
			if !strings.Contains(reason, "cell: cell-runner-error") {
				t.Errorf("reason file %s = %q; want it to name the cell", target, reason)
			}
			if !strings.Contains(reason, "repetition: 1") {
				t.Errorf("reason file %s = %q; want \"repetition: 1\"", target, reason)
			}
			if !strings.Contains(reason, "cause: "+CauseRunnerError) {
				t.Errorf("reason file %s = %q; want \"cause: %s\"", target, reason, CauseRunnerError)
			}
			if !strings.Contains(reason, "exit_code: 1") {
				t.Errorf("reason file %s = %q; want \"exit_code: 1\"", target, reason)
			}
			if !strings.Contains(reason, fmt.Sprintf("attempt: %d", k)) {
				t.Errorf("reason file %s = %q; want \"attempt: %d\"", target, reason, k)
			}
		}
	})

	t.Run("InvalidReasonUnparseableAnswer", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.Configs = []Config{{ID: "cell-unparseable", Ladder: "up", Task: "task-1", Allowed: nil}}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "no_fence")

		exitNonZero, err := Run(context.Background(), runOpts(env, ladderPath, nil))
		if err != nil {
			t.Fatalf("Run() = %v; want no error", err)
		}
		if !exitNonZero {
			t.Error("Run() reported a zero exit for a cell that never produced a valid answer")
		}

		dir := RepDir(env.resultsRoot, "cell-unparseable", 1)
		for k := 1; k <= MaxAttempts; k++ {
			target := dir + ".invalid-" + strconv.Itoa(k)
			reason := readInvalidReason(t, target)
			if !strings.Contains(reason, "cause: "+CauseUnparseableAnswer) {
				t.Errorf("reason file %s = %q; want \"cause: %s\"", target, reason, CauseUnparseableAnswer)
			}
			if !strings.Contains(reason, fmt.Sprintf("attempt: %d", k)) {
				t.Errorf("reason file %s = %q; want \"attempt: %d\"", target, reason, k)
			}
			for _, line := range strings.Split(reason, "\n") {
				if strings.HasPrefix(line, "exit_code:") {
					t.Errorf("reason file %s = %q; want no \"exit_code:\" line -- the fixture exits zero", target, reason)
				}
			}
		}
	})

	t.Run("InvalidReasonResultError", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.Configs = []Config{{ID: "cell-result-error", Ladder: "rl", Task: "task-1", Allowed: nil}}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "result_error")

		exitNonZero, err := Run(context.Background(), runOpts(env, ladderPath, nil))
		if err != nil {
			t.Fatalf("Run() = %v; want no error", err)
		}
		if !exitNonZero {
			t.Error("Run() reported a zero exit for a cell that never produced a valid answer")
		}

		dir := RepDir(env.resultsRoot, "cell-result-error", 1)
		for k := 1; k <= MaxAttempts; k++ {
			target := dir + ".invalid-" + strconv.Itoa(k)
			reason := readInvalidReason(t, target)
			if !strings.Contains(reason, "cause: "+CauseResultError) {
				t.Errorf("reason file %s = %q; want \"cause: %s\"", target, reason, CauseResultError)
			}
			if !strings.Contains(reason, fmt.Sprintf("attempt: %d", k)) {
				t.Errorf("reason file %s = %q; want \"attempt: %d\"", target, reason, k)
			}
			for _, line := range strings.Split(reason, "\n") {
				if strings.HasPrefix(line, "exit_code:") {
					t.Errorf("reason file %s = %q; want no \"exit_code:\" line -- the fixture exits zero", target, reason)
				}
			}
		}
	})

	t.Run("InvalidReasonReEntry", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.Configs = []Config{{ID: "cell-reentry", Ladder: "rn", Task: "task-1", Allowed: nil}}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "partial_fail")

		opts := runOpts(env, ladderPath, nil)

		if _, err := Run(context.Background(), opts); err != nil {
			t.Fatalf("first Run() = %v; want no error", err)
		}

		dir := RepDir(env.resultsRoot, "cell-reentry", 1)
		for k := 1; k <= MaxAttempts; k++ {
			assertDirExists(t, dir+".invalid-"+strconv.Itoa(k))
		}

		if _, err := Run(context.Background(), opts); err != nil {
			t.Fatalf("second Run() = %v; want no error", err)
		}

		target := dir + ".invalid-4"
		reason := readInvalidReason(t, target)
		if !strings.Contains(reason, "attempt: 1") {
			t.Errorf("reason file %s = %q; want \"attempt: 1\" -- the attempt counter restarts on a re-entered root", target, reason)
		}
		if strings.Contains(reason, "attempt: 4") {
			t.Errorf("reason file %s = %q; want the attempt counter to diverge from the directory suffix", target, reason)
		}
	})

	t.Run("Lock", func(t *testing.T) {
		env, sha := newE2EEnv(t, fakeBinPath)
		l := baseLadder(env, sha, 1, "task-1")
		l.Configs = []Config{{ID: "cell-lock", Ladder: "l", Task: "task-1", Allowed: nil}}
		ladderPath := filepath.Join(t.TempDir(), "ladder.yaml")
		writeSyntheticLadderFile(t, ladderPath, l)
		setFakeClaudeEnv(t, l, "normal")

		worktreeRoot, err := ResolveWorktreeRoot(env.quarryRepoRoot)
		if err != nil {
			t.Fatalf("ResolveWorktreeRoot() = %v", err)
		}
		release, err := AcquireRunLock(worktreeRoot, "/first/holder/results")
		if err != nil {
			t.Fatalf("AcquireRunLock() = %v; want no error", err)
		}
		t.Cleanup(func() { _ = release() })

		_, err = Run(context.Background(), runOpts(env, ladderPath, nil))
		if err == nil {
			t.Fatal("Run() against an already-locked worktree root = nil error; want one naming the first holder")
		}
		pid := strconv.Itoa(os.Getpid())
		if !strings.Contains(err.Error(), "pid "+pid) {
			t.Errorf("Run() error = %q; want it to carry pid %s", err, pid)
		}
		if !strings.Contains(err.Error(), "/first/holder/results") {
			t.Errorf("Run() error = %q; want it to carry the first holder's results root", err)
		}
	})

	t.Run("Report", func(t *testing.T) {
		// Copy the committed fixture root into a fresh temporary directory under the same base-name
		// directory "root": the golden summary and table each carry that base name and no wall-clock
		// time, so preserving it across the copy is load-bearing, not incidental.
		dst := filepath.Join(t.TempDir(), "root")
		copyDir(t, "testdata/results/root", dst)

		// Place a fake "claude" on PATH for the duration of the report path: report.go and
		// summarize.go re-derive everything from the raw tree and never take a Runner, so nothing in
		// this call graph should ever exec it. The marker file is the assertion -- it must never be
		// created.
		binDir := t.TempDir()
		markerPath := filepath.Join(t.TempDir(), "invoked-marker")
		writeNeverInvokedClaudeStub(t, binDir, markerPath)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		summary, rendered, written := summarizeAndWriteReport(t, dst)

		if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
			t.Errorf("the fake claude binary on PATH was invoked during the report path (marker exists: err=%v)", err)
		}

		if rendered != written {
			t.Error("the printed (rendered) table does not equal the table written to disk")
		}

		wantSummary, err := os.ReadFile("testdata/results/golden-summary.json")
		if err != nil {
			t.Fatalf("read golden-summary.json: %v", err)
		}
		gotSummary, err := os.ReadFile(filepath.Join(dst, SummaryFile))
		if err != nil {
			t.Fatalf("read produced summary.json: %v", err)
		}
		if string(gotSummary) != string(wantSummary) {
			t.Errorf("produced summary.json does not match testdata/results/golden-summary.json byte for byte:\ngot:\n%s\nwant:\n%s", gotSummary, wantSummary)
		}

		wantTable, err := os.ReadFile("testdata/results/golden-table.txt")
		if err != nil {
			t.Fatalf("read golden-table.txt: %v", err)
		}
		if written != string(wantTable) {
			t.Errorf("produced table.txt does not match testdata/results/golden-table.txt byte for byte:\ngot:\n%s\nwant:\n%s", written, wantTable)
		}

		if len(summary.Cells) != 1 {
			t.Fatalf("summary.Cells has %d entries; want 1", len(summary.Cells))
		}
		cell := summary.Cells[0]
		recallStats, ok := cell.Metrics["recall"]
		if !ok || recallStats.N != 1 {
			t.Errorf("recall stats = %+v, ok=%v; want N=1, excluding the ceiling repetition", recallStats, ok)
		}
		precisionStats, ok := cell.Metrics["precision"]
		if !ok || precisionStats.N != 1 {
			t.Errorf("precision stats = %+v, ok=%v; want N=1, excluding the ceiling repetition", precisionStats, ok)
		}
		turnsStats, ok := cell.Metrics["turns"]
		if !ok || turnsStats.N != 2 {
			t.Errorf("turns (cost) stats = %+v, ok=%v; want N=2, including the ceiling repetition", turnsStats, ok)
		}
		if cell.MaxTurnsCount != 1 {
			t.Errorf("cell.MaxTurnsCount = %d; want 1", cell.MaxTurnsCount)
		}
		if cell.UnscoredCount != 0 {
			t.Errorf("cell.UnscoredCount = %d; want 0", cell.UnscoredCount)
		}
	})
}

// readInvalidReason reads dir/InvalidReasonFile, failing the test with the path on a read error.
func readInvalidReason(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, InvalidReasonFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// copyDir recursively copies every file and directory under src into dst, creating dst.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture tree %s -> %s: %v", src, dst, err)
	}
}

// writeNeverInvokedClaudeStub writes an executable shell script named "claude" into binDir that
// records its own invocation by creating markerPath and exits non-zero, so a test that places binDir
// on PATH can assert markerPath was never created.
func writeNeverInvokedClaudeStub(t *testing.T, binDir, markerPath string) {
	t.Helper()
	script := "#!/bin/sh\ntouch " + markerPath + "\nexit 1\n"
	path := filepath.Join(binDir, "claude")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write never-invoked claude stub: %v", err)
	}
}
