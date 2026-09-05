// after_test.go pins the twenty testdata/ golden files as goldens: one real
// invocation of Run, in-process, against a Loomyard checkout at loomyardPin, compared byte for
// byte against the committed golden. These twenty files are also the task's committed evidence —
// the golden fixture and the regression gate are the same artifact. The table spans four verbs —
// toc, glyphs, resolve, and expand, the fourth a frozen preset over the first — and each row
// carries its own expected exit code: the expected code lives here, in the table, and in
// testdata/INDEX.md, never in a trailer inside a golden file itself.
//
// TestAfterGoldens is named so that "-run TestAfter -update" (testdata/INDEX.md's
// own regeneration instructions) matches it by prefix: a differently named function would make that
// -update run a silent no-op that produces no files at all, which is why the name and the
// regeneration command are load-bearing on each other and must not be changed independently.
//
// Each case gives its target repository-relative, never absolute, because --root rebases target
// interpretation against the repository root rather than the test process's own working directory
// — without that rule the target would resolve outside the Loomyard root and return exitNegative
// instead of a golden. This dependency on repoRelTarget's --root handling is the second reason this
// file and loomyard_test.go / cli.go must not be changed independently of each other.
//
// This file opens a deliberate red window with the batch it ships in: it compares against
// testdata/toc-dir.txt and the other eleven golden files before the card that creates the eight new
// ones has run, so on a machine with the pinned checkout "go test ./internal/cli/..." fails on a
// missing golden between this card's commit and the next one's. That window is expected and
// bounded to within the batch — the batch's own verify only runs once every card has landed — and
// it is not papered over by skipping on a missing golden: after the next card, a missing golden is
// a real regression (a deleted or unstaged fixture), and a skip there would hide exactly the
// failure this gate exists to catch.

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// afterGoldenCase is one row of the after/ golden table: the golden file name, the verb it
// invokes, the CLI arguments following that verb (used both to build Run's argv and to spell the
// invocation line each golden file records), the repository-relative target those arguments end
// with, and the exit code Run is expected to return for this row.
type afterGoldenCase struct {
	golden     string
	verb       string
	invocation string
	verbArgs   []string
	exitCode   int
}

// afterGoldenCases is testdata/INDEX.md's own table: which golden file each
// invocation produces, the exact invocation-line suffix it records, and the exit code Run is
// expected to return. Each row spells its own suffix literally, rather than deriving it from
// verbArgs, so a machine-specific --root cannot leak into a committed golden by construction.
var afterGoldenCases = []afterGoldenCase{
	{
		golden:     "toc-dir.txt",
		verb:       "toc",
		invocation: "internal/logger",
		verbArgs:   []string{"internal/logger"},
		exitCode:   exitOK,
	},
	{
		golden:     "toc-file.txt",
		verb:       "toc",
		invocation: "internal/logger/logger.go",
		verbArgs:   []string{"internal/logger/logger.go"},
		exitCode:   exitOK,
	},
	{
		golden:     "toc-dir-text.txt",
		verb:       "toc",
		invocation: "--text internal/logger",
		verbArgs:   []string{"--text", "internal/logger"},
		exitCode:   exitOK,
	},
	{
		golden:     "toc-file-text.txt",
		verb:       "toc",
		invocation: "--text internal/logger/logger.go",
		verbArgs:   []string{"--text", "internal/logger/logger.go"},
		exitCode:   exitOK,
	},
	{
		golden:     "resolve-glyph.txt",
		verb:       "resolve",
		invocation: "internal/logger#stderrHandlerSnapshot",
		verbArgs:   []string{"internal/logger#stderrHandlerSnapshot"},
		exitCode:   exitOK,
	},
	{
		golden:     "resolve-glyph-text.txt",
		verb:       "resolve",
		invocation: "--text internal/logger#stderrHandlerSnapshot",
		verbArgs:   []string{"--text", "internal/logger#stderrHandlerSnapshot"},
		exitCode:   exitOK,
	},
	{
		golden:     "resolve-method.txt",
		verb:       "resolve",
		invocation: "internal/logger#dualHandler.Handle",
		verbArgs:   []string{"internal/logger#dualHandler.Handle"},
		exitCode:   exitOK,
	},
	{
		golden:     "resolve-not-found.txt",
		verb:       "resolve",
		invocation: "internal/logger#noSuchSymbol",
		verbArgs:   []string{"internal/logger#noSuchSymbol"},
		exitCode:   exitNegative,
	},
	{
		golden:     "resolve-self-file.txt",
		verb:       "resolve",
		invocation: "internal/logger/logger.go#",
		verbArgs:   []string{"internal/logger/logger.go#"},
		exitCode:   exitOK,
	},
	{
		golden:     "resolve-self-dir.txt",
		verb:       "resolve",
		invocation: "internal/logger#",
		verbArgs:   []string{"internal/logger#"},
		exitCode:   exitOK,
	},
	{
		golden:     "resolve-self-file-text.txt",
		verb:       "resolve",
		invocation: "--text internal/logger/logger.go#",
		verbArgs:   []string{"--text", "internal/logger/logger.go#"},
		exitCode:   exitOK,
	},
	{
		golden:     "resolve-self-dir-text.txt",
		verb:       "resolve",
		invocation: "--text internal/logger#",
		verbArgs:   []string{"--text", "internal/logger#"},
		exitCode:   exitOK,
	},
	{
		golden:     "expand-type.txt",
		verb:       "expand",
		invocation: "internal/logger#dualHandler",
		verbArgs:   []string{"internal/logger#dualHandler"},
		exitCode:   exitOK,
	},
	{
		golden:     "expand-type-text.txt",
		verb:       "expand",
		invocation: "--text internal/logger#dualHandler",
		verbArgs:   []string{"--text", "internal/logger#dualHandler"},
		exitCode:   exitOK,
	},
	{
		golden:     "expand-not-a-type.txt",
		verb:       "expand",
		invocation: "internal/logger#newDualHandler",
		verbArgs:   []string{"internal/logger#newDualHandler"},
		exitCode:   exitNegative,
	},
	{
		golden:     "glyphs-dir.txt",
		verb:       "glyphs",
		invocation: "internal/logger",
		verbArgs:   []string{"internal/logger"},
		exitCode:   exitOK,
	},
	{
		golden:     "glyphs-dir-text.txt",
		verb:       "glyphs",
		invocation: "--text internal/logger",
		verbArgs:   []string{"--text", "internal/logger"},
		exitCode:   exitOK,
	},
	{
		golden:     "glyphs-file.txt",
		verb:       "glyphs",
		invocation: "internal/logger/logger.go",
		verbArgs:   []string{"internal/logger/logger.go"},
		exitCode:   exitOK,
	},
	{
		golden:     "glyphs-file-text.txt",
		verb:       "glyphs",
		invocation: "--text internal/logger/logger.go",
		verbArgs:   []string{"--text", "internal/logger/logger.go"},
		exitCode:   exitOK,
	},
	// toc-view-glyphs-depth.txt is a "toc" invocation, not a "glyphs" one, because the preset is
	// frozen at --depth all and rejects --depth; a depth-cut view is only reachable by spelling
	// "toc --view glyphs" directly. internal/boardengine is the target because it is the smallest
	// of the three directories in the pinned checkout whose own subdirectory itself holds Go files
	// (internal/boardengine/boardtest) — the property the "flat projection crosses directory
	// boundaries" assertion below needs — and internal/logger, the other four rows' target, has no
	// such subdirectory at this pin. The row carries no --symbols token, and that omission is
	// deliberate and load-bearing: the view supplies that default itself, and a non-empty symbol
	// list in this golden *is* the assertion that the default works. A version of this row that
	// spelled --symbols would pass whether the default works or not.
	{
		golden:     "toc-view-glyphs-depth.txt",
		verb:       "toc",
		invocation: "--text --view glyphs --depth 1 internal/boardengine",
		verbArgs:   []string{"--text", "--view", "glyphs", "--depth", "1", "internal/boardengine"},
		exitCode:   exitOK,
	},
}

// TestAfterGoldens runs every case in afterGoldenCases against loomyardRepo(t) and compares the
// assembled bytes against the committed golden, or rewrites it under -update.
func TestAfterGoldens(t *testing.T) {
	repo := loomyardRepo(t)

	for _, tc := range afterGoldenCases {
		t.Run(tc.golden, func(t *testing.T) {
			args := append([]string{tc.verb, "--root", repo}, tc.verbArgs...)

			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code != tc.exitCode {
				t.Fatalf("Run(%v) code = %d, stderr = %q; want %d", args, code, stderr.String(), tc.exitCode)
			}
			if tc.exitCode == exitOK && stderr.Len() != 0 {
				t.Fatalf("Run(%v) stderr = %q; want empty", args, stderr.String())
			}

			got := "$ quarry " + tc.verb + " " + tc.invocation + "\n\n" + stdout.String()
			compareAfterGolden(t, tc.golden, got)
		})
	}
}

// compareAfterGolden compares got byte for byte against testdata/name, or — under -update —
// writes got to that path, creating the testdata/ directory if it does not yet exist.
func compareAfterGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)

	if *updateGoldens {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("compareAfterGolden(%q): mkdir: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("compareAfterGolden(%q): write: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("compareAfterGolden(%q): read golden: %v", name, err)
	}
	if string(want) != got {
		t.Errorf("golden %q mismatch (-want +got):\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}
