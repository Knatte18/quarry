// resolve_test.go covers resolveContext, the function that replaced lookupContext's in-hub/
// out-of-hub branching once quarry dropped the lyx hub concept entirely: this is the end-to-end
// half of TDD candidates 1 and 2, wiring the two path-resolution tiers paths_test.go already
// covers in isolation (resolveConfigPath, resolveStateDir) into a real registry load.
// It needs no build tag -- unlike the hubforge fixture its Loomyard predecessor
// (cli_integration_test.go) built, nothing here spawns a process, touches the network, or shells
// out to git.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// TestResolveContext_ConfigPrecedence verifies an explicit --config value wins over
// $QUARRY_CONFIG and over the user config directory default.
func TestResolveContext_ConfigPrecedence(t *testing.T) {
	tempConfigDir := withIsolatedPathSeams(t)
	dir := t.TempDir()

	// Seed a servers.yaml at every one of the three tiers with a distinct "go" entry so the
	// test observes which tier actually got read, not merely that some file loaded.
	flagConfig := writeServersYAML(t, filepath.Join(t.TempDir(), "flag-servers.yaml"), "flag-cmd")
	envConfig := writeServersYAML(t, filepath.Join(t.TempDir(), "env-servers.yaml"), "env-cmd")
	writeServersYAML(t, filepath.Join(tempConfigDir, "quarry", "servers.yaml"), "default-cmd")

	t.Setenv("QUARRY_CONFIG", envConfig)
	t.Setenv("QUARRY_STATE_DIR", t.TempDir())

	registry, _, _, err := resolveContext(dir, flagConfig, "", nil)
	if err != nil {
		t.Fatalf("resolveContext(...) error = %v; want nil", err)
	}
	if got := registry["go"].Command[0]; got != "flag-cmd" {
		t.Errorf("resolveContext(...) registry[\"go\"].Command[0] = %q; want %q (the --config flag value, not $QUARRY_CONFIG or the default tier)", got, "flag-cmd")
	}
}

// TestResolveContext_ConfigEnvBeatsDefault verifies $QUARRY_CONFIG wins over the user config
// directory default when no --config flag is given.
func TestResolveContext_ConfigEnvBeatsDefault(t *testing.T) {
	tempConfigDir := withIsolatedPathSeams(t)
	dir := t.TempDir()

	envConfig := writeServersYAML(t, filepath.Join(t.TempDir(), "env-servers.yaml"), "env-cmd")
	writeServersYAML(t, filepath.Join(tempConfigDir, "quarry", "servers.yaml"), "default-cmd")

	t.Setenv("QUARRY_CONFIG", envConfig)
	t.Setenv("QUARRY_STATE_DIR", t.TempDir())

	registry, _, _, err := resolveContext(dir, "", "", nil)
	if err != nil {
		t.Fatalf("resolveContext(...) error = %v; want nil", err)
	}
	if got := registry["go"].Command[0]; got != "env-cmd" {
		t.Errorf("resolveContext(...) registry[\"go\"].Command[0] = %q; want %q ($QUARRY_CONFIG, not the default tier)", got, "env-cmd")
	}
}

// TestResolveContext_ConfigFileWholeReplacesBuiltinEntry verifies a servers.yaml written at the
// resolved path whole-replaces the built-in entry for that language in the returned registry.
func TestResolveContext_ConfigFileWholeReplacesBuiltinEntry(t *testing.T) {
	withIsolatedPathSeams(t)
	dir := t.TempDir()

	configPath := writeServersYAML(t, filepath.Join(t.TempDir(), "servers.yaml"), "overridden-gopls")

	registry, _, _, err := resolveContext(dir, configPath, "", nil)
	if err != nil {
		t.Fatalf("resolveContext(...) error = %v; want nil", err)
	}

	builtinGo := quarry.BuiltinRegistry()["go"]
	got := registry["go"]
	if got.Command[0] != "overridden-gopls" {
		t.Errorf("resolveContext(...) registry[\"go\"].Command = %v; want the overlay's [\"overridden-gopls\"]", got.Command)
	}
	if got.InstallHint == builtinGo.InstallHint {
		t.Errorf("resolveContext(...) registry[\"go\"].InstallHint = %q; want the overlay's install hint, not the built-in's %q -- whole-entry replacement, not a field-level merge", got.InstallHint, builtinGo.InstallHint)
	}

	// Every other built-in language entry survives untouched: the overlay only replaces the
	// language it names.
	if got, want := registry["python"], quarry.BuiltinRegistry()["python"]; got.Command[0] != want.Command[0] {
		t.Errorf("resolveContext(...) registry[\"python\"] = %+v; want the untouched built-in %+v", got, want)
	}
}

// TestResolveContext_AbsentFileAtEveryTierReturnsBuiltinRegistry verifies an absent file at every
// precedence tier returns the built-in registry with no error.
func TestResolveContext_AbsentFileAtEveryTierReturnsBuiltinRegistry(t *testing.T) {
	withIsolatedPathSeams(t)
	dir := t.TempDir()

	registry, _, _, err := resolveContext(dir, "", "", nil)
	if err != nil {
		t.Fatalf("resolveContext(...) error = %v; want nil", err)
	}

	if got, want := registry["go"], quarry.BuiltinRegistry()["go"]; got.Command[0] != want.Command[0] || got.InstallHint != want.InstallHint {
		t.Errorf("resolveContext(...) registry[\"go\"] = %+v; want the built-in entry %+v", got, want)
	}
}

// TestResolveContext_MalformedFileReturnsError verifies a malformed config file returns an error
// rather than degrading to the built-in registry -- the one case where the loader does not
// swallow the failure.
func TestResolveContext_MalformedFileReturnsError(t *testing.T) {
	withIsolatedPathSeams(t)
	dir := t.TempDir()

	malformed := filepath.Join(t.TempDir(), "servers.yaml")
	if err := os.WriteFile(malformed, []byte("go: [this is not a valid entry mapping"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", malformed, err)
	}

	_, _, _, err := resolveContext(dir, malformed, "", nil)
	if err == nil {
		t.Fatal("resolveContext(...) error = nil; want non-nil for a malformed servers.yaml")
	}
}

// TestResolveContext_StateDirPrecedence verifies an explicit --state-dir value wins over
// $QUARRY_STATE_DIR and over the user cache directory default.
func TestResolveContext_StateDirPrecedence(t *testing.T) {
	withIsolatedPathSeams(t)
	dir := t.TempDir()

	flagStateDir := t.TempDir()
	t.Setenv("QUARRY_STATE_DIR", t.TempDir())

	_, _, stateDir, err := resolveContext(dir, "", flagStateDir, nil)
	if err != nil {
		t.Fatalf("resolveContext(...) error = %v; want nil", err)
	}
	if stateDir != flagStateDir {
		t.Errorf("resolveContext(...) stateDir = %q; want the --state-dir flag value %q", stateDir, flagStateDir)
	}
}

// TestResolveContext_StateDirEnvBeatsDefault verifies $QUARRY_STATE_DIR wins over the user cache
// directory default when no --state-dir flag is given.
func TestResolveContext_StateDirEnvBeatsDefault(t *testing.T) {
	tempCacheDir := withIsolatedPathSeams(t)
	dir := t.TempDir()

	envStateDir := t.TempDir()
	t.Setenv("QUARRY_STATE_DIR", envStateDir)

	_, _, stateDir, err := resolveContext(dir, "", "", nil)
	if err != nil {
		t.Fatalf("resolveContext(...) error = %v; want nil", err)
	}
	if stateDir != envStateDir {
		t.Errorf("resolveContext(...) stateDir = %q; want $QUARRY_STATE_DIR's value %q, not a directory under %q", stateDir, envStateDir, tempCacheDir)
	}
}

// TestResolveContext_NonEmptyBuildTagsChangesOnlyStateDir verifies a non-empty build-tag set
// changes the returned state directory while leaving the returned registry and absolute target
// directory unchanged.
func TestResolveContext_NonEmptyBuildTagsChangesOnlyStateDir(t *testing.T) {
	withIsolatedPathSeams(t)
	dir := t.TempDir()

	untaggedRegistry, untaggedTargetDir, untaggedStateDir, err := resolveContext(dir, "", "", nil)
	if err != nil {
		t.Fatalf("resolveContext(..., nil) error = %v; want nil", err)
	}

	taggedRegistry, taggedTargetDir, taggedStateDir, err := resolveContext(dir, "", "", []string{"integration"})
	if err != nil {
		t.Fatalf("resolveContext(..., [integration]) error = %v; want nil", err)
	}

	if taggedStateDir == untaggedStateDir {
		t.Errorf("resolveContext(..., [integration]) stateDir = %q; want distinct from the untagged stateDir %q", taggedStateDir, untaggedStateDir)
	}
	if taggedTargetDir != untaggedTargetDir {
		t.Errorf("resolveContext(..., [integration]) targetDir = %q; want unchanged from the untagged targetDir %q", taggedTargetDir, untaggedTargetDir)
	}
	if taggedRegistry["go"].Command[0] != untaggedRegistry["go"].Command[0] {
		t.Errorf("resolveContext(..., [integration]) registry[\"go\"] = %+v; want unchanged from the untagged registry %+v", taggedRegistry["go"], untaggedRegistry["go"])
	}
}

// TestResolveContext_TargetDirIsAbsoluteFormOfDirArgument verifies the returned target directory
// is the absolute form of the dir argument.
func TestResolveContext_TargetDirIsAbsoluteFormOfDirArgument(t *testing.T) {
	withIsolatedPathSeams(t)
	dir := t.TempDir()

	t.Setenv("QUARRY_STATE_DIR", t.TempDir())

	_, targetDir, _, err := resolveContext(dir, "", "", nil)
	if err != nil {
		t.Fatalf("resolveContext(...) error = %v; want nil", err)
	}

	wantAbs, absErr := filepath.Abs(dir)
	if absErr != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", dir, absErr)
	}
	if targetDir != wantAbs {
		t.Errorf("resolveContext(...) targetDir = %q; want %q (filepath.Abs(dir))", targetDir, wantAbs)
	}
}

// TestResolveContext_StateDirCarriesNoLyxOrScoutSegment verifies the returned state directory
// carries no ".lyx" and no "scout" segment -- quarry has no hub, and "scout" is dead vocabulary
// once the tool is called quarry.
func TestResolveContext_StateDirCarriesNoLyxOrScoutSegment(t *testing.T) {
	withIsolatedPathSeams(t)
	dir := t.TempDir()

	_, _, stateDir, err := resolveContext(dir, "", "", nil)
	if err != nil {
		t.Fatalf("resolveContext(...) error = %v; want nil", err)
	}

	if want := ".lyx"; containsPathSegment(stateDir, want) {
		t.Errorf("resolveContext(...) stateDir = %q; want no %q segment", stateDir, want)
	}
	if want := "scout"; containsPathSegment(stateDir, want) {
		t.Errorf("resolveContext(...) stateDir = %q; want no %q segment", stateDir, want)
	}
}

// containsPathSegment reports whether path contains segment as one of its "/"-delimited
// components, after normalising path's separators via filepath.ToSlash so this works identically
// on linux and windows.
func containsPathSegment(path, segment string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == segment {
			return true
		}
	}
	return false
}

// writeServersYAML writes a minimal valid servers.yaml overlay at path, overriding only the "go"
// entry with a distinct Command[0] so a test can tell which config tier actually got read, and
// returns path.
func writeServersYAML(t *testing.T, path, goCommand string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	body := "go:\n" +
		"  markers:\n" +
		"    - go.mod\n" +
		"  match: any\n" +
		"  command:\n" +
		"    - " + goCommand + "\n" +
		"  install_hint: install " + goCommand + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}
