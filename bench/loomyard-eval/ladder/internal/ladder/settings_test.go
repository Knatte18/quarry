package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestDenyListFor_NoneControlsDenyAllSevenQuarryNames(t *testing.T) {
	l := mustLoadLadder(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}
	b0, err := ConfigByID(l, "b0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "b0-none", err)
	}

	expected := make([]string, 0, len(QuarryTools))
	for _, tool := range QuarryTools {
		expected = append(expected, MCPName(tool))
	}
	sort.Strings(expected)

	if got := DenyListFor(l, a0); !reflect.DeepEqual(got, expected) {
		t.Errorf("DenyListFor(l, a0) = %v; want %v", got, expected)
	}
	if got := DenyListFor(l, b0); !reflect.DeepEqual(got, expected) {
		t.Errorf("DenyListFor(l, b0) = %v; want %v", got, expected)
	}
}

func TestDenyListFor_FullBundlesDenyNoQuarryName(t *testing.T) {
	l := mustLoadLadder(t)
	a5, err := ConfigByID(l, "a5-bundle")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a5-bundle", err)
	}
	b7, err := ConfigByID(l, "b7-bundle")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "b7-bundle", err)
	}

	if got := DenyListFor(l, a5); len(got) != 0 {
		t.Errorf("DenyListFor(l, a5) = %v; want empty", got)
	}
	if got := DenyListFor(l, b7); len(got) != 0 {
		t.Errorf("DenyListFor(l, b7) = %v; want empty", got)
	}
}

func TestDenyListFor_B5ImpactDeniesExactlySix(t *testing.T) {
	l := mustLoadLadder(t)
	b5, err := ConfigByID(l, "b5-impact")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "b5-impact", err)
	}
	if got := len(DenyListFor(l, b5)); got != 6 {
		t.Errorf("len(DenyListFor(l, b5)) = %d; want 6", got)
	}
}

func TestDenyListFor_TracksAMutatedQuarryTools(t *testing.T) {
	// Post-load mutation on purpose: LoadLadder rejects a quarry_tools list that is not exactly the
	// canonical seven, so this drift can only be expressed on an already-loaded Ladder. It proves
	// DenyListFor derives from l.QuarryTools with no per-config edit -- not that the loader accepts
	// an eighth tool.
	l := mustLoadLadder(t)
	l.QuarryTools = append(append([]string{}, l.QuarryTools...), "eighth_tool")

	for _, config := range l.Configs {
		deny := DenyListFor(l, config)
		if !stringSliceContains(config.Allowed, "eighth_tool") {
			if !stringSliceContains(deny, MCPName("eighth_tool")) {
				t.Errorf("DenyListFor(l, %q) = %v; want it to contain %q", config.ID, deny, MCPName("eighth_tool"))
			}
		}
	}
}

func TestSettingsDocumentFor_NoDocumentContainsTask(t *testing.T) {
	l := mustLoadLadder(t)
	for _, config := range l.Configs {
		settings := SettingsDocumentFor(l, config)
		if stringSliceContains(settings.Permissions.Deny, "Task") {
			t.Errorf("SettingsDocumentFor(l, %q).Permissions.Deny = %v; want it not to contain \"Task\" -- a session-wide Task deny leaves the operator's own live session unable to dispatch the run agent at all", config.ID, settings.Permissions.Deny)
		}
	}
}

// TestSettingsDocumentFor_AllowIncludesConfigsOwnGrantedQuarryNames asserts Allow is exactly the fixed
// Read/Grep/Glob/Bash set, plus config's own granted quarry tool names appended in l.QuarryTools' own
// order -- mirroring SettingsDocumentFor's own algorithm structurally, the same way
// TestDenyListFor_NoneControlsDenyAllSevenQuarryNames mirrors DenyListFor's. This is what lets a real run
// call its own granted quarry tools without an interactive permission prompt for every single call --
// prompting there would corrupt this suite's own duration measurements.
func TestSettingsDocumentFor_AllowIncludesConfigsOwnGrantedQuarryNames(t *testing.T) {
	l := mustLoadLadder(t)
	base := []string{"Read", "Grep", "Glob", "Bash"}
	for _, config := range l.Configs {
		settings := SettingsDocumentFor(l, config)
		want := append([]string{}, base...)
		for _, tool := range l.QuarryTools {
			if stringSliceContains(config.Allowed, tool) {
				want = append(want, MCPName(tool))
			}
		}
		if !reflect.DeepEqual(settings.Permissions.Allow, want) {
			t.Errorf("SettingsDocumentFor(l, %q).Permissions.Allow = %v; want %v", config.ID, settings.Permissions.Allow, want)
		}
	}
}

func TestSettingsDocumentFor_NoNonQuarryNameAppearsInAnyDenyList(t *testing.T) {
	l := mustLoadLadder(t)
	quarryNames := map[string]bool{}
	for _, tool := range QuarryTools {
		quarryNames[MCPName(tool)] = true
	}
	for _, config := range l.Configs {
		settings := SettingsDocumentFor(l, config)
		for _, name := range settings.Permissions.Deny {
			if !quarryNames[name] {
				t.Errorf("SettingsDocumentFor(l, %q).Permissions.Deny contains %q; want only quarry names", config.ID, name)
			}
		}
	}
}

// TestSettingsDocumentFor_EnabledMcpjsonServersMatchesServerDeclaration asserts EnabledMcpjsonServers is
// exactly ["quarry"] for a config that declares a server (Allowed non-empty) and nil for one that doesn't
// (a blinded config) -- the same condition PrepareRunSession itself uses for HasServerDeclaration, so a
// session's own settings.json is never left silently un-pre-approved for a server it actually connects
// to, and never names "quarry" in a blinded session's own settings.json.
func TestSettingsDocumentFor_EnabledMcpjsonServersMatchesServerDeclaration(t *testing.T) {
	l := mustLoadLadder(t)
	for _, config := range l.Configs {
		settings := SettingsDocumentFor(l, config)
		if len(config.Allowed) == 0 {
			if settings.EnabledMcpjsonServers != nil {
				t.Errorf("SettingsDocumentFor(l, %q).EnabledMcpjsonServers = %v; want nil for a blinded config", config.ID, settings.EnabledMcpjsonServers)
			}
			continue
		}
		want := []string{"quarry"}
		if !reflect.DeepEqual(settings.EnabledMcpjsonServers, want) {
			t.Errorf("SettingsDocumentFor(l, %q).EnabledMcpjsonServers = %v; want %v", config.ID, settings.EnabledMcpjsonServers, want)
		}
	}
}

func TestSettingsDocumentFor_BlindedConfigDeniesNothing(t *testing.T) {
	l := mustLoadLadder(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}
	settings := SettingsDocumentFor(l, a0)
	want := []string{}
	if !reflect.DeepEqual(settings.Permissions.Deny, want) {
		t.Errorf("SettingsDocumentFor(l, a0).Permissions.Deny = %v; want %v", settings.Permissions.Deny, want)
	}
}

func TestSettingsDocumentFor_RungConfigDeniesExactlyItsDenyList(t *testing.T) {
	l := mustLoadLadder(t)
	b5, err := ConfigByID(l, "b5-impact")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "b5-impact", err)
	}
	settings := SettingsDocumentFor(l, b5)
	want := DenyListFor(l, b5)
	if !reflect.DeepEqual(settings.Permissions.Deny, want) {
		t.Errorf("SettingsDocumentFor(l, b5).Permissions.Deny = %v; want %v", settings.Permissions.Deny, want)
	}
}

func TestWriteSettings_SerialisesTheSettingsDocument(t *testing.T) {
	l := mustLoadLadder(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}
	outPath := filepath.Join(t.TempDir(), "settings.json")
	if err := WriteSettings(l, a0, outPath); err != nil {
		t.Fatalf("WriteSettings(l, a0, %q) = %v; want nil error", outPath, err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read %s: %v", outPath, err)
	}
	var written SettingsDocument
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("unmarshal %s: %v", outPath, err)
	}

	want := SettingsDocumentFor(l, a0)
	if !reflect.DeepEqual(written, want) {
		t.Errorf("WriteSettings wrote %+v; want %+v", written, want)
	}
}
