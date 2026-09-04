// after_test.go pins the four docs/research/output-formats/after/ files as goldens: one real
// invocation of Run, in-process, against a Loomyard checkout at loomyardPin, compared byte for
// byte against the committed golden. These four files are also the task's committed evidence — the
// golden fixture and the regression gate are the same artifact.
//
// TestAfterGoldens is named so that "-run TestAfter -update" (docs/research/output-formats/after's
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
// after/toc-dir.txt and the other three golden files before the card that creates them has run, so
// on a machine with the pinned checkout "go test ./internal/cli/..." fails on a missing golden
// between this card's commit and the next one's. That window is expected and bounded to within the
// batch — the batch's own verify only runs once every card has landed — and it is not papered over
// by skipping on a missing golden: after the next card, a missing golden is a real regression (a
// deleted or unstaged fixture), and a skip there would hide exactly the failure this gate exists to
// catch.

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// afterGoldenCase is one row of the after/ golden table: the golden file name, the CLI arguments
// following the "toc" verb (used both to build Run's argv and to spell the invocation line each
// golden file records), and the repository-relative target those arguments end with.
type afterGoldenCase struct {
	golden     string
	invocation string
	verbArgs   []string
}

// afterGoldenCases is docs/research/output-formats/after/'s own table: which golden file each
// invocation produces, and the exact invocation-line suffix it records. Each row spells its own
// suffix literally, rather than deriving it from verbArgs, so a machine-specific --root cannot leak
// into a committed golden by construction.
var afterGoldenCases = []afterGoldenCase{
	{
		golden:     "toc-dir.txt",
		invocation: "internal/logger",
		verbArgs:   []string{"internal/logger"},
	},
	{
		golden:     "toc-file.txt",
		invocation: "internal/logger/logger.go",
		verbArgs:   []string{"internal/logger/logger.go"},
	},
	{
		golden:     "toc-dir-text.txt",
		invocation: "--text internal/logger",
		verbArgs:   []string{"--text", "internal/logger"},
	},
	{
		golden:     "toc-file-text.txt",
		invocation: "--text internal/logger/logger.go",
		verbArgs:   []string{"--text", "internal/logger/logger.go"},
	},
}

// TestAfterGoldens runs every case in afterGoldenCases against loomyardRepo(t) and compares the
// assembled bytes against the committed golden, or rewrites it under -update.
func TestAfterGoldens(t *testing.T) {
	repo := loomyardRepo(t)

	for _, tc := range afterGoldenCases {
		t.Run(tc.golden, func(t *testing.T) {
			args := append([]string{"toc", "--root", repo}, tc.verbArgs...)

			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("Run(%v) code = %d, stderr = %q; want %d", args, code, stderr.String(), exitOK)
			}
			if stderr.Len() != 0 {
				t.Fatalf("Run(%v) stderr = %q; want empty", args, stderr.String())
			}

			got := "$ quarry toc " + tc.invocation + "\n\n" + stdout.String()
			compareAfterGolden(t, tc.golden, got)
		})
	}
}

// compareAfterGolden compares got byte for byte against
// docs/research/output-formats/after/name, or — under -update — writes got to that path,
// creating the after/ directory if it does not yet exist.
func compareAfterGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("..", "..", "docs", "research", "output-formats", "after", name)

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
