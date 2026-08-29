// schema.go implements the schema-derivation-and-patching Shared Decision: input and output
// schemas are inferred from Go types with jsonschema.For[T], then patched so a per-entry violation
// stays a per-entry error instead of failing the whole call. See the plan's
// schema-derivation-and-patching decision for the three mandatory patches this file applies.

package mcpserver

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/google/jsonschema-go/jsonschema"
)

// docSentences accepts either a JSON number or a JSON string for a tool's "docSentences"
// parameter, so {"docSentences": 3} and {"docSentences": "all"} both reach
// cli.ParseDocSentences as a string. Its zero value (an empty raw message) means "not supplied".
type docSentences struct {
	raw json.RawMessage
}

// UnmarshalJSON accepts a JSON number or a JSON string and rejects anything else — in particular a
// JSON boolean, which would otherwise silently decode into neither form value() recognizes.
func (d *docSentences) UnmarshalJSON(data []byte) error {
	var asNumber json.Number
	if err := json.Unmarshal(data, &asNumber); err == nil {
		d.raw = append(d.raw[:0], data...)
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		d.raw = append(d.raw[:0], data...)
		return nil
	}

	return fmt.Errorf("mcpserver: docSentences must be a JSON number or string, got %s", string(data))
}

// value returns the decimal string form of a number, the string unchanged for a string, and false
// for the zero value (raw is empty, meaning the field was not supplied).
func (d docSentences) value() (string, bool) {
	if len(d.raw) == 0 {
		return "", false
	}

	var asString string
	if err := json.Unmarshal(d.raw, &asString); err == nil {
		return asString, true
	}

	// Not a JSON string: UnmarshalJSON only accepts a number as the other case, so this is the
	// number's own JSON text, already the decimal string form ParseDocSentences expects.
	return string(d.raw), true
}

// docSentencesSchema is the schema registered for docSentences via jsonschema.ForOptions —
// inference would otherwise reduce the type to a property-less object, since docSentences has no
// exported fields of its own.
var docSentencesSchema = &jsonschema.Schema{Types: []string{"integer", "string"}}

// schemaOptions is the one jsonschema.ForOptions value every call to jsonschema.For in this
// package uses — inputSchemaFor and outputSchemaFor alike, never nil — so the two helpers name one
// call form. Harmless for outputSchemaFor callers, since no output type embeds docSentences.
var schemaOptions = &jsonschema.ForOptions{
	TypeSchemas: map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[docSentences](): docSentencesSchema,
	},
}

// clearAdditionalProperties walks s's Properties, Items, and AdditionalProperties recursively and
// sets each visited object schema's AdditionalProperties to nil, so a stray or wrong-tool property
// on one entry never fails the whole call. It guards against a nil schema and against revisiting a
// schema pointer twice, since jsonschema.For can share one *Schema across multiple reachable paths.
func clearAdditionalProperties(s *jsonschema.Schema) {
	clearAdditionalPropertiesVisited(s, make(map[*jsonschema.Schema]bool))
}

func clearAdditionalPropertiesVisited(s *jsonschema.Schema, visited map[*jsonschema.Schema]bool) {
	if s == nil || visited[s] {
		return
	}
	visited[s] = true

	additionalProperties := s.AdditionalProperties
	s.AdditionalProperties = nil

	for _, prop := range s.Properties {
		clearAdditionalPropertiesVisited(prop, visited)
	}
	clearAdditionalPropertiesVisited(s.Items, visited)
	clearAdditionalPropertiesVisited(additionalProperties, visited)
}

// inputSchemaFor derives T's input schema via jsonschema.For, then applies the targets-property
// patches every tool's input schema needs: Types is cleared and Type is set to "array" (slice
// inference emits Types: ["null", "array"], which would let {"targets": null} bypass minItems
// entirely), MinItems/MaxItems are set to minTargets/maxTargets, and clearAdditionalProperties is
// applied to the targets item schema only — the call-wide properties keep whatever inference
// produced, so a call-level violation stays a whole-call failure per
// input-schema-strictness.
func inputSchemaFor[T any]() (*jsonschema.Schema, error) {
	s, err := jsonschema.For[T](schemaOptions)
	if err != nil {
		return nil, fmt.Errorf("mcpserver: derive input schema for %T: %w", *new(T), err)
	}

	targets, ok := s.Properties["targets"]
	if !ok {
		return nil, fmt.Errorf("mcpserver: derive input schema for %T: no %q property", *new(T), "targets")
	}

	targets.Types = nil
	targets.Type = "array"
	targets.MinItems = jsonschema.Ptr(minTargets)
	targets.MaxItems = jsonschema.Ptr(maxTargets)
	clearAdditionalProperties(targets.Items)

	return s, nil
}

// outputSchemaFor derives T's output schema via jsonschema.For, using the same schemaOptions value
// inputSchemaFor uses, then applies clearAdditionalProperties to the whole tree, so a mixed batch
// (some entries succeeding, some failing) never fails output validation over a property one entry
// type carries and another does not.
func outputSchemaFor[T any]() (*jsonschema.Schema, error) {
	s, err := jsonschema.For[T](schemaOptions)
	if err != nil {
		return nil, fmt.Errorf("mcpserver: derive output schema for %T: %w", *new(T), err)
	}

	clearAdditionalProperties(s)
	return s, nil
}

// dropEntryProperty removes name from the targets item schema's Properties map and from its
// PropertyOrder, so a tool whose entry Go type carries a field it does not accept never advertises
// that field in its published schema.
func dropEntryProperty(s *jsonschema.Schema, name string) {
	targets, ok := s.Properties["targets"]
	if !ok {
		return
	}
	items := targets.Items
	if items == nil {
		return
	}

	delete(items.Properties, name)

	order := items.PropertyOrder[:0]
	for _, p := range items.PropertyOrder {
		if p != name {
			order = append(order, p)
		}
	}
	items.PropertyOrder = order
}

// unknownEntryKeys unmarshals raw into a map[string]json.RawMessage and returns the sorted key
// names absent from allowed, or nil when raw is not a JSON object.
//
// This exists because clearing additionalProperties on entry schemas is what makes a wrong-tool
// property a per-entry error instead of a whole-call failure — the SDK no longer rejects the call
// for it, so the handler needs its own detection point to still report it as a per-entry error.
func unknownEntryKeys(raw json.RawMessage, allowed ...string) []string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}

	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}

	var unknown []string
	for k := range obj {
		if !allowedSet[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	return unknown
}
