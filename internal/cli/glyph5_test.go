// glyph5_test.go is the machine-independent proof that all four resolution statuses — found,
// not_found, ambiguous, and multipart — answer correctly through the command line, on a fixture
// tree this file builds itself rather than a pinned external checkout. It exists because the
// evidence goldens in internal/cli/testdata/ cannot carry two of those cases: the
// golden test skips wherever LADDER_LOOMYARD_REPO is unset, and the pinned Loomyard checkout is
// not known to contain either a build-tag-duplicated declaration or a several-declaration
// initialiser, so this file is what runs everywhere, including on a build machine with no
// checkout at all.

package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// newGlyph5Fixture builds the tree every case in TestRun_AllFourStatuses shares: one package
// directory, "pkg" — never the fixture root, since a file directly under the root has the empty
// unit that no glyph can spell — holding:
//
//   - one free function, Alpha, declared exactly once, for the found row;
//   - one type with a method, Beta, declared exactly once, for the expand found row, distinct
//     from the duplicated type below;
//   - a second, differently named free function, Gamma, declared twice across two files under
//     different build tags, for the resolve ambiguous row — not the found row's function, since
//     two declarations of one name is what makes that row ambiguous rather than found;
//   - a second, differently named type, Delta, declared twice across two files under different
//     build tags, for the expand ambiguous row, distinct from Beta for the same reason;
//   - a file declaring two package-level init functions, so one glyph — "init" — matches several
//     parts of one symbol.
//
// Every duplicated declaration is distinct from every singly-declared one. It returns the
// fixture's absolute root.
func newGlyph5Fixture(t *testing.T) string {
	t.Helper()

	return writeScratchTree(t, "glyph5-"+t.Name(), map[string]string{
		"pkg/alpha.go": "// Package pkg is the all-four-statuses fixture's own directory.\n" +
			"package pkg\n\n" +
			"// Alpha is the fixture's singly-declared free function, spelled by the found row.\n" +
			"func Alpha() {}\n\n" +
			"// Beta is the fixture's singly-declared type with a method, spelled by the expand found\n" +
			"// row.\n" +
			"type Beta struct{}\n\n" +
			"// Method is Beta's own method.\n" +
			"func (b Beta) Method() {}\n",
		"pkg/gamma_a.go": "//go:build a\n\n" +
			"package pkg\n\n" +
			"// Gamma is one of two build-tag-duplicated declarations of this name, spelled by the\n" +
			"// resolve ambiguous row.\n" +
			"func Gamma() {}\n",
		"pkg/gamma_b.go": "//go:build b\n\n" +
			"package pkg\n\n" +
			"// Gamma is the second of two build-tag-duplicated declarations of this name.\n" +
			"func Gamma() {}\n",
		"pkg/delta_a.go": "//go:build a\n\n" +
			"package pkg\n\n" +
			"// Delta is one of two build-tag-duplicated declarations of this name, spelled by the\n" +
			"// expand ambiguous row.\n" +
			"type Delta struct{}\n",
		"pkg/delta_b.go": "//go:build b\n\n" +
			"package pkg\n\n" +
			"// Delta is the second of two build-tag-duplicated declarations of this name.\n" +
			"type Delta struct{}\n",
		"pkg/init.go": "package pkg\n\n" +
			"// init is the first of two package-level initialisers this file declares, spelled by\n" +
			"// the multipart row.\n" +
			"func init() {}\n\n" +
			"// init is the second.\n" +
			"func init() {}\n",
	})
}

// decodeGlyph5Payload decodes stdout as a generic JSON object, so a case can assert both a
// field's value and a key's presence or absence without importing the engine.
func decodeGlyph5Payload(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
	}
	return payload
}

// TestRun_AllFourStatuses proves all four resolution statuses — found, not_found, ambiguous, and
// multipart — answer correctly through the command line on the fixture newGlyph5Fixture builds,
// plus the expand verb's not-a-type failure and its own ambiguous row, asserting both the exit
// code and the decoded payload for every case. Every assertion on candidates and symbols checks
// the key's presence or absence, not only the status word: the separate key is the signal that
// nothing was chosen, and a test reading only the status would pass against an implementation
// that emitted the wrong one.
func TestRun_AllFourStatuses(t *testing.T) {
	root := newGlyph5Fixture(t)

	t.Run("found", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg#Alpha", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		payload := decodeGlyph5Payload(t, stdout)
		if payload["status"] != "found" {
			t.Errorf(`payload["status"] = %v; want "found"`, payload["status"])
		}
		symbols, ok := payload["symbols"].([]any)
		if !ok || len(symbols) != 1 {
			t.Errorf("payload = %v; want exactly one symbol", payload)
		}
	})

	t.Run("not_found, unit found", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg#NoSuchThing", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitNegative)
		}
		payload := decodeGlyph5Payload(t, stdout)
		if payload["status"] != "not_found" {
			t.Errorf(`payload["status"] = %v; want "not_found"`, payload["status"])
		}
		if payload["unit"] != "found" {
			t.Errorf(`payload["unit"] = %v; want "found"`, payload["unit"])
		}
	})

	t.Run("not_found, unit not_found", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg/missing#Something", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitNegative)
		}
		payload := decodeGlyph5Payload(t, stdout)
		if payload["status"] != "not_found" {
			t.Errorf(`payload["status"] = %v; want "not_found"`, payload["status"])
		}
		if payload["unit"] != "not_found" {
			t.Errorf(`payload["unit"] = %v; want "not_found"`, payload["unit"])
		}
	})

	t.Run("ambiguous", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg#Gamma", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitNegative)
		}
		payload := decodeGlyph5Payload(t, stdout)
		if payload["status"] != "ambiguous" {
			t.Errorf(`payload["status"] = %v; want "ambiguous"`, payload["status"])
		}
		candidates, ok := payload["candidates"].([]any)
		if !ok || len(candidates) != 2 {
			t.Errorf("payload = %v; want exactly two candidates", payload)
		}
		if _, ok := payload["symbols"]; ok {
			t.Errorf("payload = %v; want no %q key on an ambiguous result", payload, "symbols")
		}
	})

	t.Run("multipart", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg#init", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		payload := decodeGlyph5Payload(t, stdout)
		if payload["status"] != "multipart" {
			t.Errorf(`payload["status"] = %v; want "multipart"`, payload["status"])
		}
		symbols, ok := payload["symbols"].([]any)
		if !ok || len(symbols) != 2 {
			t.Errorf("payload = %v; want every part present in the symbols list", payload)
		}
	})

	t.Run("not a type", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg#Alpha", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		envErr := failureEnvelope(t, stdout)
		want := "expand pkg#Alpha: not a type, kind function"
		if envErr != want {
			t.Errorf("error = %q; want %q", envErr, want)
		}
		if stderr != want+"\n" {
			t.Errorf("stderr = %q; want %q", stderr, want+"\n")
		}
		if strings.Contains(stderr, usageText) {
			t.Errorf("stderr = %q; must not carry usage text", stderr)
		}
	})

	t.Run("expand, ambiguous", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg#Delta", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitNegative)
		}
		payload := decodeGlyph5Payload(t, stdout)
		if payload["status"] != "ambiguous" {
			t.Errorf(`payload["status"] = %v; want "ambiguous"`, payload["status"])
		}
		candidates, ok := payload["candidates"].([]any)
		if !ok || len(candidates) != 2 {
			t.Errorf("payload = %v; want exactly two candidates", payload)
		}
		if _, ok := payload["head"]; ok {
			t.Errorf("payload = %v; want no %q key on an ambiguous answer", payload, "head")
		}
		if _, ok := payload["members"]; ok {
			t.Errorf("payload = %v; want no %q key on an ambiguous answer", payload, "members")
		}
	})
}
