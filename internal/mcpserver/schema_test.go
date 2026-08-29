package mcpserver

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

// fixtureEntry is a small local fixture entry type, standing in for a real tool's per-entry type.
type fixtureEntry struct {
	Target       string  `json:"target,omitempty"`
	OptionalName *string `json:"optionalName,omitempty"`
	Nested       *nested `json:"nested,omitempty"`
}

type nested struct {
	Line int `json:"line,omitempty"`
}

// fixtureCall is a small local fixture call type, standing in for a real tool's input type: a
// call-wide property plus the shared "targets" array.
type fixtureCall struct {
	Limit   int            `json:"limit,omitempty"`
	Targets []fixtureEntry `json:"targets"`
}

// fixtureResultEntry is a small local fixture per-entry result type, standing in for a real tool's
// output entry shape.
type fixtureResultEntry struct {
	Target string  `json:"target"`
	Status string  `json:"status"`
	Detail *string `json:"detail,omitempty"`
}

// fixtureOutput is a small local fixture output type, standing in for a real tool's output type.
type fixtureOutput struct {
	Results []fixtureResultEntry `json:"results"`
}

// fixtureDocSentences is a small local fixture type embedding docSentences, standing in for a real
// tool's toc-shaped entry.
type fixtureDocSentences struct {
	DocSentences docSentences `json:"docSentences,omitempty"`
}

func TestInputSchemaFor_TargetsArrayShape(t *testing.T) {
	s, err := inputSchemaFor[fixtureCall]()
	if err != nil {
		t.Fatalf("inputSchemaFor[fixtureCall]() error = %v", err)
	}

	targets, ok := s.Properties["targets"]
	if !ok {
		t.Fatal("inputSchemaFor[fixtureCall]() has no \"targets\" property")
	}
	if targets.Type != "array" {
		t.Errorf("targets.Type = %q; want \"array\"", targets.Type)
	}
	if slices.Contains(targets.Types, "null") {
		t.Errorf("targets.Types = %v; must not contain \"null\"", targets.Types)
	}
	if targets.MinItems == nil || *targets.MinItems != minTargets {
		t.Errorf("targets.MinItems = %v; want %d", targets.MinItems, minTargets)
	}
	if targets.MaxItems == nil || *targets.MaxItems != maxTargets {
		t.Errorf("targets.MaxItems = %v; want %d", targets.MaxItems, maxTargets)
	}
}

func TestInputSchemaFor_NoAdditionalPropertiesUnderTargets(t *testing.T) {
	s, err := inputSchemaFor[fixtureCall]()
	if err != nil {
		t.Fatalf("inputSchemaFor[fixtureCall]() error = %v", err)
	}

	items := s.Properties["targets"].Items
	if items.AdditionalProperties != nil {
		t.Error("targets.Items.AdditionalProperties is set; want nil")
	}
	nestedSchema := items.Properties["nested"]
	if nestedSchema == nil {
		t.Fatal("targets.Items.Properties has no \"nested\" property")
	}
	// nested is *nested, so inference wraps it as Types ["null", "object"] rather than Type
	// "object"; AdditionalProperties must still be cleared regardless of which field carries it.
	if nestedSchema.AdditionalProperties != nil {
		t.Error("targets.Items.Properties[\"nested\"].AdditionalProperties is set; want nil")
	}
}

func TestInputSchemaFor_CallWidePropertySurvives(t *testing.T) {
	s, err := inputSchemaFor[fixtureCall]()
	if err != nil {
		t.Fatalf("inputSchemaFor[fixtureCall]() error = %v", err)
	}

	limit, ok := s.Properties["limit"]
	if !ok {
		t.Fatal("inputSchemaFor[fixtureCall]() has no \"limit\" property")
	}
	if limit.Type != "integer" {
		t.Errorf("limit.Type = %q; want \"integer\" (call-wide inference must survive unpatched)", limit.Type)
	}
	// The call-wide AdditionalProperties on the top-level struct is deliberately untouched by
	// inputSchemaFor: only the targets item schema is patched, so a call-level violation stays a
	// whole-call failure.
	if s.AdditionalProperties == nil {
		t.Error("s.AdditionalProperties is nil; want the call-wide false-schema inference produced, unpatched")
	}
}

func TestInputSchemaFor_NoEntryPropertyIsRequired(t *testing.T) {
	s, err := inputSchemaFor[fixtureCall]()
	if err != nil {
		t.Fatalf("inputSchemaFor[fixtureCall]() error = %v", err)
	}

	items := s.Properties["targets"].Items
	if len(items.Required) != 0 {
		t.Errorf("targets.Items.Required = %v; want empty (every entry property must be optional)", items.Required)
	}
}

func TestInputSchemaFor_MissingTargetsErrors(t *testing.T) {
	if _, err := inputSchemaFor[fixtureResultEntry](); err == nil {
		t.Error("inputSchemaFor[fixtureResultEntry]() error = nil; want an error naming the missing \"targets\" property")
	}
}

func TestOutputSchemaFor_NoAdditionalPropertiesAnywhere(t *testing.T) {
	s, err := outputSchemaFor[fixtureOutput]()
	if err != nil {
		t.Fatalf("outputSchemaFor[fixtureOutput]() error = %v", err)
	}

	if s.AdditionalProperties != nil {
		t.Error("outputSchemaFor top-level AdditionalProperties is set; want nil")
	}
	results := s.Properties["results"]
	if results == nil {
		t.Fatal("outputSchemaFor[fixtureOutput]() has no \"results\" property")
	}
	if results.Items.AdditionalProperties != nil {
		t.Error("outputSchemaFor results.Items.AdditionalProperties is set; want nil")
	}
}

func TestDocSentences_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    string
		wantOK  bool
		wantErr bool
	}{
		{"Number", `3`, "3", true, false},
		{"String", `"all"`, "all", true, false},
		{"Bool", `true`, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d docSentences
			err := json.Unmarshal([]byte(tt.json), &d)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("json.Unmarshal(%s) error = nil; want an error", tt.json)
				}
				return
			}
			if err != nil {
				t.Fatalf("json.Unmarshal(%s) error = %v", tt.json, err)
			}
			got, ok := d.value()
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("d.value() = (%q, %v); want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDocSentences_ZeroValue(t *testing.T) {
	var d docSentences
	if got, ok := d.value(); ok || got != "" {
		t.Errorf("zero docSentences.value() = (%q, %v); want (\"\", false)", got, ok)
	}
}

// TestInputSchemaFor_DocSentencesSchema exercises the TypeSchemas registration schemaOptions wires
// through jsonschema.For: fixtureDocSentences has no "targets" property, so inputSchemaFor itself
// errors, and the derivation is checked directly against jsonschema.For instead.
func TestInputSchemaFor_DocSentencesSchema(t *testing.T) {
	if _, err := inputSchemaFor[fixtureDocSentences](); err == nil {
		t.Fatal("inputSchemaFor[fixtureDocSentences]() error = nil; want an error (fixtureDocSentences has no \"targets\" property)")
	}

	full, err := jsonschema.For[fixtureDocSentences](schemaOptions)
	if err != nil {
		t.Fatalf("jsonschema.For[fixtureDocSentences](schemaOptions) error = %v", err)
	}
	ds := full.Properties["docSentences"]
	if ds == nil {
		t.Fatal("derived schema has no \"docSentences\" property")
	}
	if !slices.Contains(ds.Types, "integer") || !slices.Contains(ds.Types, "string") {
		t.Errorf("docSentences schema Types = %v; want both \"integer\" and \"string\"", ds.Types)
	}
}

func TestUnknownEntryKeys(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		allowed []string
		want    []string
	}{
		{"UnknownKeyReported", `{"target":"x","bogus":1}`, []string{"target"}, []string{"bogus"}},
		{"OnlyAllowedKeys", `{"target":"x"}`, []string{"target", "line"}, nil},
		{"NonObjectInput", `[1,2,3]`, []string{"target"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unknownEntryKeys(json.RawMessage(tt.raw), tt.allowed...)
			if len(got) != len(tt.want) {
				t.Fatalf("unknownEntryKeys(%s, %v) = %v; want %v", tt.raw, tt.allowed, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("unknownEntryKeys(%s, %v) = %v; want %v", tt.raw, tt.allowed, got, tt.want)
				}
			}
		})
	}
}

func TestDropEntryProperty(t *testing.T) {
	s, err := inputSchemaFor[fixtureCall]()
	if err != nil {
		t.Fatalf("inputSchemaFor[fixtureCall]() error = %v", err)
	}

	dropEntryProperty(s, "optionalName")

	items := s.Properties["targets"].Items
	if _, ok := items.Properties["optionalName"]; ok {
		t.Error("targets.Items.Properties still has \"optionalName\" after dropEntryProperty")
	}
	if slices.Contains(items.PropertyOrder, "optionalName") {
		t.Errorf("targets.Items.PropertyOrder = %v; must not contain \"optionalName\"", items.PropertyOrder)
	}
	if _, ok := items.Properties["target"]; !ok {
		t.Error("targets.Items.Properties lost \"target\" after dropping \"optionalName\"")
	}
	if _, ok := items.Properties["nested"]; !ok {
		t.Error("targets.Items.Properties lost \"nested\" after dropping \"optionalName\"")
	}
}
