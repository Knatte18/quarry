package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// ladderYAMLPath is the committed ladder.yaml this package's tests load, matching
// internal/ladder/ladder_test.go's own fixture path (cmd/ladderbench sits at the same depth under
// bench/loomyard-eval/ladder/ as internal/ladder does).
const ladderYAMLPath = "../../ladder.yaml"

// mustLoadLadderFixture loads the committed ladder.yaml or fails the test.
func mustLoadLadderFixture(t *testing.T) *ladder.Ladder {
	t.Helper()
	l, err := ladder.LoadLadder(ladderYAMLPath)
	if err != nil {
		t.Fatalf("LoadLadder(%q) = _, %v; want nil error", ladderYAMLPath, err)
	}
	return l
}

// mustLoadLadderForSessionsFixture loads the committed ladder.yaml and redirects its session-directory
// template under t.TempDir(), pinning a run model so PrepareRunSession succeeds -- mirroring
// internal/ladder/session_test.go's own mustLoadLadderForSessions helper.
func mustLoadLadderForSessionsFixture(t *testing.T) *ladder.Ladder {
	t.Helper()
	l := mustLoadLadderFixture(t)
	l.SessionDirTemplate = filepath.Join(t.TempDir(), "session-{config_id}-{n}")
	runModel := "claude-opus-5"
	l.RunModel = &runModel
	return l
}

// mustWriteSkillFixture writes a dummy tracked orchestration skill at the path
// installedSkillSourcePath(repoRoot) resolves, so a test's call into installOrchestrationSkill succeeds
// without the real batch-14 skill file existing yet.
func mustWriteSkillFixture(t *testing.T, repoRoot string) {
	t.Helper()
	path := installedSkillSourcePath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("skill body"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestResolvePrepareSessionMode_RunWithoutRepErrors(t *testing.T) {
	f := prepareSessionFlags{configID: "a0-none", repGiven: false}
	if _, err := resolvePrepareSessionMode(f); err == nil {
		t.Error("resolvePrepareSessionMode() = _, nil; want an error for a run session missing --rep")
	}
}

func TestResolvePrepareSessionMode_RunWithConfigAndRepSucceeds(t *testing.T) {
	f := prepareSessionFlags{configID: "a0-none", rep: 1, repGiven: true}
	mode, err := resolvePrepareSessionMode(f)
	if err != nil {
		t.Fatalf("resolvePrepareSessionMode() = _, %v; want nil error", err)
	}
	if mode != modeRun {
		t.Errorf("resolvePrepareSessionMode() mode = %v; want modeRun", mode)
	}
}

func TestResolvePrepareSessionMode_MutuallyExclusiveModesError(t *testing.T) {
	tests := []struct {
		name string
		f    prepareSessionFlags
	}{
		{"scoring and probe", prepareSessionFlags{scoring: true, probe: ladder.ProbeKindAllowlist}},
		{"scoring and release", prepareSessionFlags{scoring: true, release: true}},
		{"probe and release", prepareSessionFlags{probe: ladder.ProbeKindDenylist, release: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := resolvePrepareSessionMode(tt.f); err == nil {
				t.Errorf("resolvePrepareSessionMode(%+v) = _, nil; want a mutually-exclusive-modes error", tt.f)
			}
		})
	}
}

func TestResolvePrepareSessionMode_UnknownProbeKindErrors(t *testing.T) {
	f := prepareSessionFlags{probe: "bogus"}
	if _, err := resolvePrepareSessionMode(f); err == nil {
		t.Error("resolvePrepareSessionMode() = _, nil; want an error for an unknown --probe kind")
	}
}

func TestResolvePrepareSessionMode_ReleaseAlone(t *testing.T) {
	f := prepareSessionFlags{release: true}
	mode, err := resolvePrepareSessionMode(f)
	if err != nil {
		t.Fatalf("resolvePrepareSessionMode() = _, %v; want nil error", err)
	}
	if mode != modeRelease {
		t.Errorf("resolvePrepareSessionMode() mode = %v; want modeRelease", mode)
	}
}

func TestResolveRunModel_OverrideSatisfiesThePinCheckThatWouldOtherwiseFail(t *testing.T) {
	l := mustLoadLadderFixture(t)
	// Exercise the unset-RunModel branch below regardless of ladder.yaml's own committed value, matching
	// TestRequireSessionPins.
	l.RunModel = nil
	if _, err := resolveRunModel(l, ""); err == nil {
		t.Fatal("resolveRunModel(l, \"\") = _, nil; want an error, since run_model is unset and no override was given")
	}

	effective, err := resolveRunModel(l, "claude-opus-5")
	if err != nil {
		t.Fatalf("resolveRunModel(l, %q) = _, %v; want nil error", "claude-opus-5", err)
	}
	if effective.RunModel == nil || *effective.RunModel != "claude-opus-5" {
		t.Errorf("resolveRunModel(l, %q).RunModel = %v; want a pointer to %q", "claude-opus-5", effective.RunModel, "claude-opus-5")
	}
	if l.RunModel != nil {
		t.Error("resolveRunModel() mutated the original *ladder.Ladder's RunModel; want it left nil, since the override is never written back")
	}
}

func TestSelectsColdPath_KeyedOnColdFieldNotID(t *testing.T) {
	tests := []struct {
		name   string
		config ladder.LadderConfig
		want   bool
	}{
		{"cold field true, id has no cold substring", ladder.LadderConfig{ID: "a5-bundle-cold", Cold: true}, true},
		{"id merely contains a cold substring, cold field false", ladder.LadderConfig{ID: "b0-cold-none", Cold: false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectsColdPath(tt.config); got != tt.want {
				t.Errorf("selectsColdPath(%+v) = %t; want %t", tt.config, got, tt.want)
			}
		})
	}
}

func TestAbortColdAttempt_WritesRecordInvalidatesAndNeverTakesTheLock(t *testing.T) {
	resultsRoot := t.TempDir()

	err := abortColdAttempt(resultsRoot, "a5-bundle-cold", 1)
	if err == nil {
		t.Fatal("abortColdAttempt() = nil; want an error describing the abort")
	}
	if !strings.Contains(err.Error(), "live daemon") {
		t.Errorf("abortColdAttempt() error = %v; want it to describe a live-daemon abort", err)
	}

	invalidDir := filepath.Join(resultsRoot, "raw", "a5-bundle-cold", "1.invalid-1")
	data, readErr := os.ReadFile(filepath.Join(invalidDir, coldAbortFilename))
	if readErr != nil {
		t.Fatalf("read %s in %s: %v", coldAbortFilename, invalidDir, readErr)
	}
	var record coldAbortRecord
	if jsonErr := json.Unmarshal(data, &record); jsonErr != nil {
		t.Fatalf("unmarshal %s: %v", coldAbortFilename, jsonErr)
	}
	want := coldAbortRecord{ConfigID: "a5-bundle-cold", Rep: 1, Attempt: 1, Cause: coldAbortCauseLiveDaemon}
	if record != want {
		t.Errorf("cold_abort.json record = %+v; want %+v", record, want)
	}

	if _, statErr := os.Stat(filepath.Join(resultsRoot, ".session-active")); !os.IsNotExist(statErr) {
		t.Error("abortColdAttempt() left a session lock file behind; want none, since the cold path never takes the lock")
	}
}

func TestPrepareColdSessionAfterGate_FailingGateAbortsWithoutBuildingOrLocking(t *testing.T) {
	l := mustLoadLadderForSessionsFixture(t)
	config, err := ladder.ConfigByID(l, "a5-bundle-cold")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a5-bundle-cold", err)
	}

	resultsRoot := t.TempDir()
	buildCalls := 0
	build := func(dir string, env []string, args ...string) (string, error) {
		buildCalls++
		return "", nil
	}
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	failing := []ladder.GateFinding{{Gate: "cold_before", Fatal: true, Message: "a daemon is already alive"}}
	err = prepareColdSessionAfterGate(cmd, l, config, 1, resultsRoot, t.TempDir(), t.TempDir(), failing, build)
	if err == nil {
		t.Fatal("prepareColdSessionAfterGate() = nil; want an abort error for a failing gate")
	}
	if buildCalls != 0 {
		t.Errorf("build calls = %d; want 0, since a failing gate must abort before building the server", buildCalls)
	}
	if _, statErr := os.Stat(filepath.Join(resultsRoot, ".session-active")); !os.IsNotExist(statErr) {
		t.Error("prepareColdSessionAfterGate() left a session lock file behind; want none")
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q; want no launch command printed on abort", out.String())
	}
}

func TestPrepareColdSessionAfterGate_PassingGateBuildsAndPrintsLaunchCommand(t *testing.T) {
	l := mustLoadLadderForSessionsFixture(t)
	config, err := ladder.ConfigByID(l, "a5-bundle-cold")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a5-bundle-cold", err)
	}

	repoRoot := t.TempDir()
	mustWriteSkillFixture(t, repoRoot)
	resultsRoot := t.TempDir()
	build := func(dir string, env []string, args ...string) (string, error) {
		return "/built/quarry-mcp", nil
	}
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := prepareColdSessionAfterGate(cmd, l, config, 1, resultsRoot, repoRoot, "/target", nil, build); err != nil {
		t.Fatalf("prepareColdSessionAfterGate() error = %v; want nil for a passing gate", err)
	}
	if out.Len() == 0 {
		t.Error("stdout is empty; want the launch command printed")
	}
	if _, statErr := os.Stat(filepath.Join(resultsRoot, ".session-active")); !os.IsNotExist(statErr) {
		t.Error("prepareColdSessionAfterGate() took the session lock; want it never taken on the cold path")
	}
}

func TestColdWorktreeDir_SubstitutesRepetition(t *testing.T) {
	l := &ladder.Ladder{ColdWorktreeTemplate: "/tmp/loomyard-eval-01-cold-{n}"}
	if got, want := coldWorktreeDir(l, 3), "/tmp/loomyard-eval-01-cold-3"; got != want {
		t.Errorf("coldWorktreeDir(l, 3) = %q; want %q", got, want)
	}
}

func TestFirstTaskWorktree_DeterministicAlphabeticalFirst(t *testing.T) {
	l := mustLoadLadderFixture(t)
	worktrees := map[string]string{
		"01-reed-geometry-exploration":   "/wt/01",
		"04-shedadapters-shuttle-impact": "/wt/04",
	}
	got, err := firstTaskWorktree(l, worktrees)
	if err != nil {
		t.Fatalf("firstTaskWorktree() = _, %v; want nil error", err)
	}
	if got != "/wt/01" {
		t.Errorf("firstTaskWorktree() = %q; want %q (the alphabetically-first task key)", got, "/wt/01")
	}
}
