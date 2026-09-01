package ladder

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// ladderYAMLPath is the committed ladder.yaml this package's tests load and mutate a copy of.
const ladderYAMLPath = "../../ladder.yaml"

// rawLadder loads ladder.yaml as a generic map, for tests that mutate one field and re-dump it to a
// temp file to exercise a specific validation rejection -- mirroring the Python suite's
// _raw_ladder_dict/_write_ladder helpers.
func rawLadder(t *testing.T) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(ladderYAMLPath)
	if err != nil {
		t.Fatalf("read %s: %v", ladderYAMLPath, err)
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal %s: %v", ladderYAMLPath, err)
	}
	return raw
}

// writeLadder serialises raw as YAML to a fresh temp file and returns its path.
func writeLadder(t *testing.T, raw map[string]interface{}) string {
	t.Helper()
	data, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal raw ladder: %v", err)
	}
	path := filepath.Join(t.TempDir(), "ladder.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// configsSlice returns raw's "configs" entry as a []interface{}.
func configsSlice(t *testing.T, raw map[string]interface{}) []interface{} {
	t.Helper()
	list, ok := raw["configs"].([]interface{})
	if !ok {
		t.Fatalf("raw[\"configs\"] is not a []interface{}: %T", raw["configs"])
	}
	return list
}

// configByRawID returns the config map with the given id, or fails the test.
func configByRawID(t *testing.T, raw map[string]interface{}, id string) map[string]interface{} {
	t.Helper()
	for _, item := range configsSlice(t, raw) {
		m := item.(map[string]interface{})
		if m["id"] == id {
			return m
		}
	}
	t.Fatalf("no config with id %q in raw ladder", id)
	return nil
}

// deepCopyRawConfig returns an independent copy of a raw config map, via a YAML round trip.
func deepCopyRawConfig(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("marshal raw config: %v", err)
	}
	var out map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal raw config copy: %v", err)
	}
	return out
}

func TestLoadLadder_SucceedsWith15Configs(t *testing.T) {
	l, err := LoadLadder(ladderYAMLPath)
	if err != nil {
		t.Fatalf("LoadLadder(%q) = _, %v; want nil error", ladderYAMLPath, err)
	}
	if len(l.Configs) != 15 {
		t.Errorf("LoadLadder(%q) loaded %d configs; want 15", ladderYAMLPath, len(l.Configs))
	}
}

func TestLoadLadder_RejectsInvalidLadders(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, raw map[string]interface{})
	}{
		{
			name: "duplicate_config_id",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				list := configsSlice(t, raw)
				list[1].(map[string]interface{})["id"] = list[0].(map[string]interface{})["id"]
			},
		},
		{
			name: "bad_ladder_value",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				configsSlice(t, raw)[0].(map[string]interface{})["ladder"] = "c"
			},
		},
		{
			name: "unknown_task_key",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				configsSlice(t, raw)[0].(map[string]interface{})["task"] = "not-a-real-task"
			},
		},
		{
			name: "allowed_entry_outside_quarry_tools",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				configsSlice(t, raw)[0].(map[string]interface{})["allowed"] = []interface{}{"not_a_real_tool"}
			},
		},
		{
			name: "non_canonical_quarry_tools",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				tools := raw["quarry_tools"].([]interface{})
				raw["quarry_tools"] = tools[:len(tools)-1]
			},
		},
		{
			name: "ladder_a_has_no_control",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				list := configsSlice(t, raw)
				filtered := make([]interface{}, 0, len(list))
				for _, item := range list {
					if item.(map[string]interface{})["id"] != "a0-none" {
						filtered = append(filtered, item)
					}
				}
				raw["configs"] = filtered
			},
		},
		{
			name: "ladder_a_has_two_controls",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				extra := deepCopyRawConfig(t, configByRawID(t, raw, "a0-none"))
				extra["id"] = "a0-none-again"
				raw["configs"] = append(configsSlice(t, raw), extra)
			},
		},
		{
			name: "second_cold_config",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				extra := deepCopyRawConfig(t, configByRawID(t, raw, "a5-bundle-cold"))
				extra["id"] = "a5-bundle-cold-again"
				raw["configs"] = append(configsSlice(t, raw), extra)
			},
		},
		{
			name: "warm_counterpart_on_non_cold_config",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				configByRawID(t, raw, "a1-toc-file")["warm_counterpart"] = "a5-bundle"
			},
		},
		{
			name: "cold_config_missing_warm_counterpart",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				delete(configByRawID(t, raw, "a5-bundle-cold"), "warm_counterpart")
			},
		},
		{
			name: "warm_counterpart_naming_unknown_id",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				configByRawID(t, raw, "a5-bundle-cold")["warm_counterpart"] = "not-a-real-id"
			},
		},
		{
			// The only way to reach the "names a cold config" branch without first tripping the
			// "more than one cold config" check is a single cold config naming itself -- there is
			// no way to construct a ladder with exactly one cold: true config whose warm_counterpart
			// names a *different* cold config, since that would require a second cold: true entry.
			name: "warm_counterpart_naming_the_cold_config_itself",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				configByRawID(t, raw, "a5-bundle-cold")["warm_counterpart"] = "a5-bundle-cold"
			},
		},
		{
			name: "absolute_session_dir_template",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				raw["session_dir_template"] = "/home/someone/quarry/.scratch/ladder-sessions/{config_id}-{n}"
			},
		},
		{
			name: "hardcoded_source_repo_path",
			mutate: func(t *testing.T, raw map[string]interface{}) {
				raw["source_repo"] = "/home/someone/Code/loomyard/wts/loomyard"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := rawLadder(t)
			tt.mutate(t, raw)
			path := writeLadder(t, raw)

			_, err := LoadLadder(path)
			if err == nil {
				t.Errorf("LoadLadder(%q) = _, nil; want a *ConfigError", path)
				return
			}
			if _, ok := err.(*ConfigError); !ok {
				t.Errorf("LoadLadder(%q) = _, %v (%T); want a *ConfigError", path, err, err)
			}
		})
	}
}

// mustLoadLadder loads the committed ladder.yaml or fails the test.
func mustLoadLadder(t *testing.T) *Ladder {
	t.Helper()
	l, err := LoadLadder(ladderYAMLPath)
	if err != nil {
		t.Fatalf("LoadLadder(%q) = _, %v; want nil error", ladderYAMLPath, err)
	}
	return l
}

func TestConfigByID(t *testing.T) {
	l := mustLoadLadder(t)

	config, err := ConfigByID(l, "a5-bundle")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v; want nil error", "a5-bundle", err)
	}
	if config.ID != "a5-bundle" {
		t.Errorf("ConfigByID(l, %q).ID = %q; want %q", "a5-bundle", config.ID, "a5-bundle")
	}

	if _, err := ConfigByID(l, "not-a-real-id"); err == nil {
		t.Error("ConfigByID(l, \"not-a-real-id\") = _, nil; want an error")
	}
}

func TestControlFor_ResolvesWithinEachLadder(t *testing.T) {
	l := mustLoadLadder(t)

	a5, err := ConfigByID(l, "a5-bundle")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a5-bundle", err)
	}
	b5, err := ConfigByID(l, "b5-impact")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "b5-impact", err)
	}

	aControl, err := ControlFor(l, a5)
	if err != nil {
		t.Fatalf("ControlFor(l, a5) = _, %v; want nil error", err)
	}
	if aControl.ID != "a0-none" {
		t.Errorf("ControlFor(l, a5).ID = %q; want %q", aControl.ID, "a0-none")
	}

	bControl, err := ControlFor(l, b5)
	if err != nil {
		t.Fatalf("ControlFor(l, b5) = _, %v; want nil error", err)
	}
	if bControl.ID != "b0-none" {
		t.Errorf("ControlFor(l, b5).ID = %q; want %q", bControl.ID, "b0-none")
	}
}

func TestWarmCounterpartFor_ResolvesTheColdConfigsWarmCell(t *testing.T) {
	l := mustLoadLadder(t)

	cold, err := ConfigByID(l, "a5-bundle-cold")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a5-bundle-cold", err)
	}
	warm, err := WarmCounterpartFor(l, cold)
	if err != nil {
		t.Fatalf("WarmCounterpartFor(l, cold) = _, %v; want nil error", err)
	}
	if warm.ID != "a5-bundle" {
		t.Errorf("WarmCounterpartFor(l, cold).ID = %q; want %q", warm.ID, "a5-bundle")
	}
}

func TestRequirePins(t *testing.T) {
	pinned := func() *string { s := "claude-opus-5"; return &s }
	turns := func() *int { n := 60; return &n }

	tests := []struct {
		name    string
		mutate  func(l *Ladder)
		wantErr bool
	}{
		{
			name:    "all_pins_set",
			mutate:  func(l *Ladder) {},
			wantErr: false,
		},
		{
			name:    "run_model_unset",
			mutate:  func(l *Ladder) { l.RunModel = nil },
			wantErr: true,
		},
		{
			name:    "max_turns_unset",
			mutate:  func(l *Ladder) { l.MaxTurns = nil },
			wantErr: true,
		},
		{
			name:    "run_effort_unset",
			mutate:  func(l *Ladder) { l.RunEffort = "" },
			wantErr: true,
		},
		{
			name:    "scorer_model_unset",
			mutate:  func(l *Ladder) { l.Scorer.Model = "" },
			wantErr: true,
		},
		{
			name:    "scorer_effort_unset",
			mutate:  func(l *Ladder) { l.Scorer.Effort = "" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := mustLoadLadder(t)
			l.RunModel = pinned()
			l.MaxTurns = turns()
			// Set explicitly rather than relying on ladder.yaml's own value, so the baseline
			// "all pins set" case is genuinely fully pinned regardless of the file's committed state.
			l.RunEffort = "medium"
			tt.mutate(l)

			err := RequirePins(l)
			if tt.wantErr && err == nil {
				t.Error("RequirePins(l) = nil; want an error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("RequirePins(l) = %v; want nil error", err)
			}
		})
	}
}

func TestRequireSessionPins(t *testing.T) {
	l := mustLoadLadder(t)
	l.MaxTurns = nil // session preparation must not depend on MaxTurns
	l.RunModel = nil // exercise the unset-RunModel branch below regardless of ladder.yaml's own value
	// Set explicitly rather than relying on ladder.yaml's own value, matching TestRequirePins.
	l.RunEffort = "medium"
	l.SessionDirTemplate = "/tmp/ladder-session-{config_id}-{n}"

	if err := RequireSessionPins(l, ""); err == nil {
		t.Error("RequireSessionPins(l, \"\") = nil; want an error, RunModel is unset and no override was given")
	}

	if err := RequireSessionPins(l, "claude-opus-5"); err != nil {
		t.Errorf("RequireSessionPins(l, %q) = %v; want nil error with MaxTurns unset", "claude-opus-5", err)
	}

	pinnedModel := "claude-opus-5"
	l.RunModel = &pinnedModel
	if err := RequireSessionPins(l, ""); err != nil {
		t.Errorf("RequireSessionPins(l, \"\") = %v; want nil error once RunModel is set", err)
	}
}
