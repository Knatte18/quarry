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
