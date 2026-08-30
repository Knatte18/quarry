package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// mustLoadLadderForSessions loads the committed ladder.yaml and redirects its session-directory
// templates under t.TempDir(), so every PrepareXSession test writes into the test's own temp directory
// rather than the committed default's /tmp path.
func mustLoadLadderForSessions(t *testing.T) *Ladder {
	t.Helper()
	l := mustLoadLadder(t)
	l.SessionDirTemplate = filepath.Join(t.TempDir(), "session-{config_id}-{n}")
	runModel := "claude-opus-5"
	l.RunModel = &runModel
	return l
}

// listFiles walks dir and returns every regular file's path relative to dir, sorted.
func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				return relErr
			}
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	sort.Strings(files)
	return files
}

func TestPrepareRunSession_BlindedConfigWritesExactlyDefinitionAndSettings(t *testing.T) {
	l := mustLoadLadderForSessions(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}

	inputs, err := PrepareRunSession(l, a0, 1, "/path/to/quarry-mcp", "/path/to/target")
	if err != nil {
		t.Fatalf("PrepareRunSession(l, a0, 1, ...) = _, %v; want nil error", err)
	}
	if inputs.HasServerDeclaration {
		t.Error("PrepareRunSession(l, a0, ...).HasServerDeclaration = true; want false")
	}

	want := []string{
		filepath.Join(".claude", "agents", "a0-none.md"),
		filepath.Join(".claude", "settings.json"),
	}
	got := listFiles(t, inputs.ScratchDir)
	if len(got) != len(want) {
		t.Fatalf("files under %s = %v; want %v", inputs.ScratchDir, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("files under %s = %v; want %v", inputs.ScratchDir, got, want)
			break
		}
	}

	settingsData, err := os.ReadFile(filepath.Join(inputs.ScratchDir, settingsRelativePath))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	if !hasExactDenySet(t, settingsData, []string{"Task"}) {
		t.Errorf("a0-none settings.json does not deny exactly [\"Task\"]: %s", settingsData)
	}
	defData, err := os.ReadFile(filepath.Join(inputs.ScratchDir, agentsRelativeDir, "a0-none.md"))
	if err != nil {
		t.Fatalf("read agent definition: %v", err)
	}
	if containsQuarry(string(defData)) {
		t.Errorf("a0-none agent definition names quarry: %s", defData)
	}
}

func TestPrepareRunSession_RungAlsoWritesServerDeclaration(t *testing.T) {
	l := mustLoadLadderForSessions(t)
	b5, err := ConfigByID(l, "b5-impact")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "b5-impact", err)
	}

	inputs, err := PrepareRunSession(l, b5, 1, "/path/to/quarry-mcp", "/path/to/target")
	if err != nil {
		t.Fatalf("PrepareRunSession(l, b5, 1, ...) = _, %v; want nil error", err)
	}
	if !inputs.HasServerDeclaration {
		t.Error("PrepareRunSession(l, b5, ...).HasServerDeclaration = false; want true")
	}
	if _, err := os.Stat(filepath.Join(inputs.ScratchDir, serverDeclarationFilename)); err != nil {
		t.Errorf("server declaration missing under %s: %v", inputs.ScratchDir, err)
	}
}

func TestPrepareRunSession_NeitherSessionWritesAScorerDefinition(t *testing.T) {
	l := mustLoadLadderForSessions(t)
	for _, id := range []string{"a0-none", "b5-impact"} {
		config, err := ConfigByID(l, id)
		if err != nil {
			t.Fatalf("ConfigByID(l, %q) = _, %v", id, err)
		}
		inputs, err := PrepareRunSession(l, config, 1, "/path/to/quarry-mcp", "/path/to/target")
		if err != nil {
			t.Fatalf("PrepareRunSession(l, %q, ...) = _, %v; want nil error", id, err)
		}
		if _, statErr := os.Stat(filepath.Join(inputs.ScratchDir, agentsRelativeDir, scorerDefinitionFilename)); statErr == nil {
			t.Errorf("PrepareRunSession(l, %q, ...) wrote a scorer definition; want none", id)
		}
	}
}

func TestPrepareScoringSession_WritesScorerDefinitionAndNoRunInputs(t *testing.T) {
	l := mustLoadLadderForSessions(t)
	inputs, err := PrepareScoringSession(l)
	if err != nil {
		t.Fatalf("PrepareScoringSession(l) = _, %v; want nil error", err)
	}
	if inputs.HasServerDeclaration {
		t.Error("PrepareScoringSession(l).HasServerDeclaration = true; want false")
	}
	if inputs.DefinitionName != scorerAgentName {
		t.Errorf("PrepareScoringSession(l).DefinitionName = %q; want %q", inputs.DefinitionName, scorerAgentName)
	}

	want := []string{
		filepath.Join(".claude", "agents", "scorer.md"),
		filepath.Join(".claude", "settings.json"),
	}
	got := listFiles(t, inputs.ScratchDir)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("files under %s = %v; want %v", inputs.ScratchDir, got, want)
	}
	if _, statErr := os.Stat(filepath.Join(inputs.ScratchDir, serverDeclarationFilename)); statErr == nil {
		t.Error("PrepareScoringSession(l) wrote a server declaration; want none")
	}
}

func TestPrepareProbeSession_BothKindsExactWriteLists(t *testing.T) {
	l := mustLoadLadderForSessions(t)

	allowlistInputs, err := PrepareProbeSession(l, ProbeKindAllowlist)
	if err != nil {
		t.Fatalf("PrepareProbeSession(l, ProbeKindAllowlist) = _, %v; want nil error", err)
	}
	wantAllowlistFiles := []string{
		filepath.Join(".claude", "agents", "probe-allowlist.md"),
		filepath.Join(".claude", "settings.json"),
	}
	gotAllowlistFiles := listFiles(t, allowlistInputs.ScratchDir)
	if len(gotAllowlistFiles) != len(wantAllowlistFiles) || gotAllowlistFiles[0] != wantAllowlistFiles[0] || gotAllowlistFiles[1] != wantAllowlistFiles[1] {
		t.Errorf("allowlist probe files = %v; want %v", gotAllowlistFiles, wantAllowlistFiles)
	}
	allowlistSettings, err := os.ReadFile(filepath.Join(allowlistInputs.ScratchDir, settingsRelativePath))
	if err != nil {
		t.Fatalf("read allowlist probe settings.json: %v", err)
	}
	if !hasExactDenySet(t, allowlistSettings, []string{"Task"}) {
		t.Errorf("allowlist probe settings.json does not deny exactly [\"Task\"]: %s", allowlistSettings)
	}

	denylistInputs, err := PrepareProbeSession(l, ProbeKindDenylist)
	if err != nil {
		t.Fatalf("PrepareProbeSession(l, ProbeKindDenylist) = _, %v; want nil error", err)
	}
	wantDenylistFiles := []string{
		filepath.Join(".claude", "agents", "probe-denylist.md"),
		filepath.Join(".claude", "settings.json"),
	}
	gotDenylistFiles := listFiles(t, denylistInputs.ScratchDir)
	if len(gotDenylistFiles) != len(wantDenylistFiles) || gotDenylistFiles[0] != wantDenylistFiles[0] || gotDenylistFiles[1] != wantDenylistFiles[1] {
		t.Errorf("denylist probe files = %v; want %v", gotDenylistFiles, wantDenylistFiles)
	}
	denylistSettings, err := os.ReadFile(filepath.Join(denylistInputs.ScratchDir, settingsRelativePath))
	if err != nil {
		t.Fatalf("read denylist probe settings.json: %v", err)
	}
	if !hasExactDenySet(t, denylistSettings, []string{MCPName(probeDeniedTool), "Task"}) {
		t.Errorf("denylist probe settings.json does not deny exactly [%q, \"Task\"]: %s", MCPName(probeDeniedTool), denylistSettings)
	}

	if !allowlistInputs.HasServerDeclaration || !denylistInputs.HasServerDeclaration {
		t.Errorf("HasServerDeclaration = %t, %t; want both true", allowlistInputs.HasServerDeclaration, denylistInputs.HasServerDeclaration)
	}
}

func TestInstallSkill_CreatesDestinationTreeAndOverwrites(t *testing.T) {
	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "SKILL.md")
	if err := os.WriteFile(sourcePath, []byte("first version"), 0o644); err != nil {
		t.Fatalf("write source skill: %v", err)
	}

	destRoot := t.TempDir()
	destPath, err := InstallSkill(sourcePath, destRoot)
	if err != nil {
		t.Fatalf("InstallSkill(%q, %q) = _, %v; want nil error", sourcePath, destRoot, err)
	}
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read installed skill: %v", err)
	}
	if string(data) != "first version" {
		t.Errorf("installed skill contents = %q; want %q", data, "first version")
	}

	if err := os.WriteFile(sourcePath, []byte("second version"), 0o644); err != nil {
		t.Fatalf("rewrite source skill: %v", err)
	}
	if _, err := InstallSkill(sourcePath, destRoot); err != nil {
		t.Fatalf("second InstallSkill(%q, %q) = _, %v; want nil error", sourcePath, destRoot, err)
	}
	data, err = os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("re-read installed skill: %v", err)
	}
	if string(data) != "second version" {
		t.Errorf("installed skill after overwrite = %q; want %q", data, "second version")
	}
}

func TestInstallSkill_NeverWritesIntoAScratchDirectory(t *testing.T) {
	l := mustLoadLadderForSessions(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}
	inputs, err := PrepareRunSession(l, a0, 1, "/path/to/quarry-mcp", "/path/to/target")
	if err != nil {
		t.Fatalf("PrepareRunSession(l, a0, 1, ...) = _, %v; want nil error", err)
	}
	filesBefore := listFiles(t, inputs.ScratchDir)

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "SKILL.md")
	if err := os.WriteFile(sourcePath, []byte("skill body"), 0o644); err != nil {
		t.Fatalf("write source skill: %v", err)
	}
	destRoot := t.TempDir()
	if _, err := InstallSkill(sourcePath, destRoot); err != nil {
		t.Fatalf("InstallSkill(%q, %q) = _, %v; want nil error", sourcePath, destRoot, err)
	}

	filesAfter := listFiles(t, inputs.ScratchDir)
	if len(filesBefore) != len(filesAfter) {
		t.Errorf("scratch dir file list changed after InstallSkill: before %v, after %v", filesBefore, filesAfter)
	}
}

func TestLaunchCommand_OmitsServerFlagForBlindedIncludesForRung(t *testing.T) {
	blinded := SessionInputs{ScratchDir: "/scratch/a0-none-1", HasServerDeclaration: false}
	if got := LaunchCommand(blinded); strings.Contains(got, "--mcp-config") {
		t.Errorf("LaunchCommand(blinded) = %q; want no --mcp-config flag", got)
	}

	rung := SessionInputs{ScratchDir: "/scratch/b5-impact-1", HasServerDeclaration: true}
	got := LaunchCommand(rung)
	if !strings.Contains(got, "--mcp-config") {
		t.Errorf("LaunchCommand(rung) = %q; want a --mcp-config flag", got)
	}
	wantMCPPath := filepath.Join(rung.ScratchDir, serverDeclarationFilename)
	if !strings.Contains(got, wantMCPPath) {
		t.Errorf("LaunchCommand(rung) = %q; want it to name %q", got, wantMCPPath)
	}
	if !strings.Contains(got, launchSettingSources) {
		t.Errorf("LaunchCommand(rung) = %q; want it to name %q", got, launchSettingSources)
	}
}

// hasExactDenySet reads a settings document's JSON bytes and reports whether its permissions.deny is
// exactly want, independent of order.
func hasExactDenySet(t *testing.T, data []byte, want []string) bool {
	t.Helper()
	var doc SettingsDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal settings document: %v", err)
	}
	got := append([]string{}, doc.Permissions.Deny...)
	wantSorted := append([]string{}, want...)
	sort.Strings(got)
	sort.Strings(wantSorted)
	if len(got) != len(wantSorted) {
		return false
	}
	for i := range got {
		if got[i] != wantSorted[i] {
			return false
		}
	}
	return true
}

func containsQuarry(s string) bool {
	return strings.Contains(strings.ToLower(s), "quarry")
}
