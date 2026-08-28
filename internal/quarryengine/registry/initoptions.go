// initoptions.go implements RenderInitializationOptions, which turns an Entry's
// InitializationOptions template and a normalized build-tag set into the initializationOptions map
// the initialize request carries.

package registry

import (
	"strings"

	"github.com/Knatte18/quarry/internal/quarryengine"
)

// tagsPlaceholder is the literal substring RenderInitializationOptions replaces with the
// comma-joined tag set. It never matches against map keys, only string values.
const tagsPlaceholder = "{{tags}}"

// RenderInitializationOptions turns entry's InitializationOptions template and tags — the caller's
// already-normalized build-tag set — into the initializationOptions map the initialize request for
// lang carries.
//
// When tags is empty, RenderInitializationOptions returns (nil, nil) unconditionally, whatever
// entry.InitializationOptions holds: this is the back-compat path, where the caller sends no
// initializationOptions key at all and the initialize request stays byte-identical to today's.
//
// When tags is non-empty and no string value anywhere in entry.InitializationOptions contains the
// literal "{{tags}}", RenderInitializationOptions returns (nil, &quarryengine.ErrBuildTagsUnsupported{Language:
// lang}). An absent template, an empty template, and a non-empty placeholder-free template all take
// this branch identically — the predicate is the placeholder's absence, never the map's emptiness.
//
// When tags is non-empty and the placeholder is present, RenderInitializationOptions deep-copies
// entry.InitializationOptions and replaces every occurrence of "{{tags}}" in every string value with
// strings.Join(tags, ","), returning the copy; entry's own map is never mutated. Map keys are never
// substituted. The template's structure is walked recursively through nested map[string]any and
// slice values — a decoded yaml map may arrive as map[string]any and a decoded yaml list as []any, so
// both that shape and a plain []string are handled — and a list element containing the placeholder
// stays one element: there is no per-tag list expansion, only an in-place, comma-joined string
// substitution.
func RenderInitializationOptions(lang string, entry Entry, tags []string) (map[string]any, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	if !templateHasPlaceholder(entry.InitializationOptions) {
		return nil, &quarryengine.ErrBuildTagsUnsupported{Language: lang}
	}

	joined := strings.Join(tags, ",")
	rendered := renderValue(entry.InitializationOptions, joined).(map[string]any)
	return rendered, nil
}

// templateHasPlaceholder reports whether any string value reachable from v (recursing through
// map[string]any and []any/[]string) contains the "{{tags}}" placeholder. Map keys are never
// examined — only values.
func templateHasPlaceholder(v any) bool {
	switch value := v.(type) {
	case string:
		return strings.Contains(value, tagsPlaceholder)
	case map[string]any:
		for _, elem := range value {
			if templateHasPlaceholder(elem) {
				return true
			}
		}
		return false
	case []any:
		for _, elem := range value {
			if templateHasPlaceholder(elem) {
				return true
			}
		}
		return false
	case []string:
		for _, elem := range value {
			if strings.Contains(elem, tagsPlaceholder) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// renderValue deep-copies v, replacing every occurrence of the "{{tags}}" placeholder in every
// string value with joined. Map keys are copied verbatim, never substituted.
func renderValue(v any, joined string) any {
	switch value := v.(type) {
	case string:
		return strings.ReplaceAll(value, tagsPlaceholder, joined)
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, elem := range value {
			out[key] = renderValue(elem, joined)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, elem := range value {
			out[i] = renderValue(elem, joined)
		}
		return out
	case []string:
		out := make([]string, len(value))
		for i, elem := range value {
			out[i] = strings.ReplaceAll(elem, tagsPlaceholder, joined)
		}
		return out
	default:
		return value
	}
}
