// fenced.go extracts a ```json ... ``` fenced block from free-form text, the shape a run's final
// answer, a task schema block, and a scorer reply all share.

package ladder

import (
	"errors"
	"fmt"
	"regexp"
)

// fencedJSONPattern is the one ```json ... ``` regex this package compiles. Every call site (the
// schema extractor, the answer parser, the scorer-reply parser) selects a different match by
// position via ExtractFencedJSON's which argument, rather than re-deriving this pattern per call
// site. Go's regexp package has no DOTALL flag; the (?s) inline flag lets the fence body span lines
// instead.
var fencedJSONPattern = regexp.MustCompile(`(?s)` + "```json" + `\s*(.*?)\s*` + "```")

// ErrNoFencedJSONBlock is returned by ExtractFencedJSON when text carries no fenced json block at all.
// Every call site wraps this with its own contextually-worded error rather than surfacing it directly,
// so no context is lost.
var ErrNoFencedJSONBlock = errors.New("no fenced json block found")

// ExtractFencedJSON finds every ```json ... ``` fenced block in text and returns the one selected by
// which -- "first" or "last" -- as (block, inner). block includes the fences and is what the schema
// extractor embeds into the preamble as measured stimulus; inner is the decode-ready content between
// the fences that the answer parser and the scorer-reply parser consume. Both halves are returned
// because both are load-bearing and neither is cheaply re-derivable from the other.
//
// It returns ErrNoFencedJSONBlock when text carries no fenced json block, and a plain error for a
// selector other than "first"/"last". This deliberately differs from the Python original in two ways:
// the Python function returns a nil result on no match, leaving each caller to raise its own
// contextually-worded error, and it silently treats any selector other than "first" as "last". A
// silent fallthrough on a typo'd selector would pick the wrong block with nothing reporting it, so the
// Go version reports both cases as errors instead.
func ExtractFencedJSON(text, which string) (block, inner string, err error) {
	if which != "first" && which != "last" {
		return "", "", fmt.Errorf("extract fenced json: unrecognised selector %q, want \"first\" or \"last\"", which)
	}

	matches := fencedJSONPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return "", "", ErrNoFencedJSONBlock
	}

	match := matches[0]
	if which == "last" {
		match = matches[len(matches)-1]
	}

	block = text[match[0]:match[1]]
	inner = text[match[2]:match[3]]
	return block, inner, nil
}
