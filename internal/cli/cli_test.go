// cli_test.go pins Run's request pipeline, the exit-code mapping, the CLI-authored error
// messages, and the failure envelope, per discussion.md's Testing block. Every case calls Run with
// bytes.Buffer sinks and an explicit --root pointing at a fixture tree built by writeScratchTree,
// so no test changes the process working directory and no test writes to a system temp directory.
//
// codeForTOCError's two sentinel branches are otherwise reachable only by a race — the target
// removed between Run's own Lstat and the engine's walk — which no test can provoke
// deterministically, so they are pinned here as a pure table against codeForTOCError directly
// instead.

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/quarry"
)

// newPipelineFixture builds the tree every Run test in this file shares: a directory (pkg) with a
// Go file carrying a package clause and a doc comment, a nested subdirectory (pkg/sub) whose own
// Go file gives --symbols something spellable to populate, a sibling subdirectory (pkg/other)
// declaring one free function, one type with a method, and one type with no members — so the
// resolve and expand verbs have something spellable to answer about — and a symlink (pkg-link) at
// the root pointing at pkg. It returns the fixture's absolute root.
//
// pkg/other sits beside pkg/sub rather than inside it or at the fixture root: adding a file inside
// pkg/sub would break TestRun_FlagPassThrough's assertion that that directory holds exactly one
// file entry, and a file directly under the fixture root would have the empty unit, which no
// glyph can spell.
func newPipelineFixture(t *testing.T) string {
	t.Helper()

	root := writeScratchTree(t, "pipeline-"+t.Name(), map[string]string{
		"pkg/doc.go": "// Package pkg is the pipeline fixture's own directory, carrying a header\n" +
			"// comment for the directory- and file-target render assertions.\n" +
			"package pkg\n",
		"pkg/sub/greet.go": "// Package sub holds one function so --symbols has something to\n" +
			"// populate.\n" +
			"package sub\n\n" +
			"// Greet returns a fixture greeting.\n" +
			"func Greet() string { return \"hello\" }\n",
		"pkg/other/other.go": "// Package other holds one free function, one type with a method, and\n" +
			"// one type with no members, so the resolve and expand verbs have something\n" +
			"// spellable to answer about.\n" +
			"package other\n\n" +
			"// Make returns a new fixture Widget.\n" +
			"func Make() Widget { return Widget{} }\n\n" +
			"// Widget is a fixture type carrying one method.\n" +
			"type Widget struct{}\n\n" +
			"// Value returns the fixture value.\n" +
			"func (w Widget) Value() int { return 42 }\n\n" +
			"// Empty is a fixture type with no members.\n" +
			"type Empty struct{}\n",
	})

	if err := os.Symlink(filepath.Join(root, "pkg"), filepath.Join(root, "pkg-link")); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}
	return root
}

// runCLI runs Run over args with fresh bytes.Buffer sinks and returns the exit code and both
// buffers' contents as strings.
func runCLI(args []string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	code = Run(args, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// failureEnvelope decodes stdout as the compact failure envelope and returns its error field.
func failureEnvelope(t *testing.T, stdout string) string {
	t.Helper()
	var env struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("failureEnvelope: json.Unmarshal(%q): %v", stdout, err)
	}
	if env.OK {
		t.Fatalf("failureEnvelope: ok = true; want false in %q", stdout)
	}
	return env.Error
}

func TestRun_ExitCodeMapping(t *testing.T) {
	root := newPipelineFixture(t)

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"directory-target", []string{"toc", "pkg", "--root", root}, exitOK},
		{"file-target", []string{"toc", "pkg/doc.go", "--root", root}, exitOK},
		{"missing-target-not-found", []string{"toc", "does-not-exist", "--root", root}, exitNegative},
		{"dot-dot-escapes-root", []string{"toc", "..", "--root", root}, exitNegative},
		{"unknown-flag", []string{"toc", "pkg", "--bogus", "--root", root}, exitUsage},
		{"missing-target-arg", []string{"toc", "--root", root}, exitUsage},
		{"two-targets", []string{"toc", "pkg", "pkg/sub", "--root", root}, exitUsage},
		{"unparseable-depth", []string{"toc", "pkg", "--depth", "x", "--root", root}, exitUsage},
		{"root-names-a-file", []string{"toc", "pkg", "--root", filepath.Join(root, "pkg", "doc.go")}, exitUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, _ := runCLI(tt.args)
			if code != tt.want {
				t.Errorf("Run(%v) code = %d; want %d", tt.args, code, tt.want)
			}
		})
	}
}

func TestRun_RequestPipelineOrdering(t *testing.T) {
	root := newPipelineFixture(t)

	t.Run("escape-checked-before-stat", func(t *testing.T) {
		code, stdout, _ := runCLI([]string{"toc", "..", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		got := failureEnvelope(t, stdout)
		if want := "target outside repository: .."; got != want {
			t.Errorf("error = %q; want %q", got, want)
		}
	})

	t.Run("not-found-names-repo-relative-path", func(t *testing.T) {
		code, stdout, _ := runCLI([]string{"toc", "pkg/nope", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		got := failureEnvelope(t, stdout)
		if want := "target not found: pkg/nope"; got != want {
			t.Errorf("error = %q; want %q", got, want)
		}
	})

	t.Run("symlink-to-directory-renders-as-file", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"toc", "pkg-link", "--text", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		// A directory-form render of the same symlink target would read
		// ".. , 1 file\npkg-link\n"; the file-form's single "pkg-link\n" line is the proof that
		// targetIsFile came from Lstat rather than Stat.
		if want := "pkg-link\n"; stdout != want {
			t.Errorf("stdout = %q; want %q (the file form)", stdout, want)
		}
	})
}

func TestRun_ExitInternal(t *testing.T) {
	root := newPipelineFixture(t)

	t.Run("stat-error-not-IsNotExist", func(t *testing.T) {
		// pkg/doc.go is a regular file; naming a path below it makes the Lstat fail with ENOTDIR,
		// which os.IsNotExist reports false for, so the pipeline must take the exitInternal branch
		// rather than misreporting it as exitNegative.
		probe := filepath.Join(root, "pkg", "doc.go", "inner")
		if _, err := os.Lstat(probe); os.IsNotExist(err) {
			t.Skip("this platform's Lstat reports IsNotExist for a path below a file; ENOTDIR case not reachable here")
		}

		code, stdout, _ := runCLI([]string{"toc", "pkg/doc.go/inner", "--root", root})
		if code != exitInternal {
			t.Fatalf("code = %d; want %d", code, exitInternal)
		}
		got := failureEnvelope(t, stdout)
		if !strings.HasPrefix(got, "internal error: ") {
			t.Errorf("error = %q; want it to start with %q", got, "internal error: ")
		}
	})

	t.Run("failing-stdout-writer", func(t *testing.T) {
		var stderr bytes.Buffer
		code := Run([]string{"toc", "pkg", "--root", root}, failingWriter{}, &stderr)
		if code != exitInternal {
			t.Errorf("code = %d; want %d", code, exitInternal)
		}
		if stderr.Len() == 0 {
			t.Errorf("stderr is empty; want the internal-error sentence")
		}
	})
}

// failingWriter is an io.Writer whose Write always fails, used to exercise Run's render-path
// write-failure branch. fail() also writes the failure envelope to this same writer and cannot
// report there either, which is why the failing-stdout-writer case above asserts only the
// returned code and stderr, never stdout.
type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, errors.New("simulated write failure")
}

func TestRun_ErrorMessageDerivation(t *testing.T) {
	root := newPipelineFixture(t)

	tests := []struct {
		name string
		args []string
		code int
	}{
		{"exit-1-not-found", []string{"toc", "nope", "--root", root}, exitNegative},
		{"exit-1-outside-root", []string{"toc", "..", "--root", root}, exitNegative},
		{"exit-2-unknown-flag", []string{"toc", "pkg", "--bogus", "--root", root}, exitUsage},
		{"exit-3-enotdir", []string{"toc", "pkg/doc.go/inner", "--root", root}, exitInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code == exitInternal {
				probe := filepath.Join(root, "pkg", "doc.go", "inner")
				if _, err := os.Lstat(probe); os.IsNotExist(err) {
					t.Skip("this platform's Lstat reports IsNotExist for a path below a file")
				}
			}

			code, stdout, stderr := runCLI(tt.args)
			if code != tt.code {
				t.Fatalf("code = %d; want %d", code, tt.code)
			}
			envErr := failureEnvelope(t, stdout)
			firstLine, _, _ := strings.Cut(stderr, "\n")
			if firstLine != envErr {
				t.Errorf("stderr first line = %q; want it byte-identical to envelope error %q", firstLine, envErr)
			}
			if tt.code == exitNegative || tt.code == exitUsage {
				if strings.Contains(envErr, "internal/engine") {
					t.Errorf("error = %q; must not contain %q", envErr, "internal/engine")
				}
			}
			if tt.code == exitInternal && !strings.HasPrefix(envErr, "internal error: ") {
				t.Errorf("error = %q; want it to start with %q", envErr, "internal error: ")
			}
		})
	}
}

func TestRun_FailureEnvelope(t *testing.T) {
	root := newPipelineFixture(t)

	cases := []struct {
		name string
		args []string
	}{
		{"exit-1", []string{"toc", "nope", "--root", root}},
		{"exit-2", []string{"toc", "pkg", "--bogus", "--root", root}},
	}
	for _, tc := range cases {
		for _, withText := range []bool{false, true} {
			name := tc.name
			args := append([]string{}, tc.args...)
			if withText {
				name += "-text"
				args = append(args, "--text")
			}
			t.Run(name, func(t *testing.T) {
				code, stdout, stderr := runCLI(args)
				if code == exitOK {
					t.Fatalf("code = %d; want non-zero", code)
				}
				envErr := failureEnvelope(t, stdout)
				want := fmt.Sprintf(`{"ok":false,"error":%q}`+"\n", envErr)
				if stdout != want {
					t.Errorf("stdout = %q; want %q", stdout, want)
				}
				if stderr == "" {
					t.Errorf("stderr is empty; want non-empty")
				}
			})
		}
	}
}

func TestRun_UsageTextPlacement(t *testing.T) {
	root := newPipelineFixture(t)

	code, stdout, stderr := runCLI([]string{"toc", "pkg", "--bogus", "--root", root})
	if code != exitUsage {
		t.Fatalf("code = %d; want %d", code, exitUsage)
	}
	envErr := failureEnvelope(t, stdout)
	wantStderr := envErr + "\n" + usageText
	if stderr != wantStderr {
		t.Errorf("stderr = %q; want %q", stderr, wantStderr)
	}
	if strings.Contains(stdout, usageText) {
		t.Errorf("stdout = %q; must not carry usage text", stdout)
	}
}

// TestRun_TOCTargetHasSeparator pins card 26's usage-error mapping: a toc target carrying the
// glyph grammar's "#" separator is exit 2, with the separator sentence followed by the full usage
// block on stderr -- both parts are asserted, since the sentence alone would pass a substring
// check while pinning the wrong bytes.
func TestRun_TOCTargetHasSeparator(t *testing.T) {
	root := newPipelineFixture(t)

	code, stdout, stderr := runCLI([]string{"toc", "pkg/other#Make", "--root", root})
	if code != exitUsage {
		t.Fatalf("code = %d; want %d", code, exitUsage)
	}
	envErr := failureEnvelope(t, stdout)
	want := `target contains the glyph separator "#": pkg/other#Make`
	if envErr != want {
		t.Errorf("error = %q; want %q", envErr, want)
	}
	wantStderr := envErr + "\n" + usageText
	if stderr != wantStderr {
		t.Errorf("stderr = %q; want %q", stderr, wantStderr)
	}
}

// TestRun_ExpandAndResolveAgreeOnBarePath pins the two-verb agreement the grammar-only
// classification buys: the same bare path given to expand and to resolve both exit 1, asserted in
// one test so their agreement is what the test is about rather than a coincidence of two separate
// rows. expand's full sentence is asserted, not the bare reason word.
func TestRun_ExpandAndResolveAgreeOnBarePath(t *testing.T) {
	root := newPipelineFixture(t)
	barePath := "pkg/other/other.go"

	expandCode, expandStdout, expandStderr := runCLI([]string{"expand", barePath, "--root", root})
	if expandCode != exitNegative {
		t.Fatalf("expand code = %d; want %d", expandCode, exitNegative)
	}
	_, parseErr := glyph.Parse(glyph.Go, barePath)
	var pe *glyph.ParseError
	if !errors.As(parseErr, &pe) {
		t.Fatalf("glyph.Parse(%q) error = %v; want a *glyph.ParseError", barePath, parseErr)
	}
	wantExpandErr := "expand: " + pe.Error()
	expandErr := failureEnvelope(t, expandStdout)
	if expandErr != wantExpandErr {
		t.Errorf("expand error = %q; want %q", expandErr, wantExpandErr)
	}
	if expandStderr != wantExpandErr+"\n" {
		t.Errorf("expand stderr = %q; want %q", expandStderr, wantExpandErr+"\n")
	}

	resolveCode, resolveStdout, resolveStderr := runCLI([]string{"resolve", barePath, "--root", root})
	if resolveCode != exitNegative {
		t.Fatalf("resolve code = %d; want %d", resolveCode, exitNegative)
	}
	if resolveStderr != "" {
		t.Errorf("resolve stderr = %q; want empty", resolveStderr)
	}
	var result quarry.ResolveResult
	if err := json.Unmarshal([]byte(resolveStdout), &result); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", resolveStdout, err)
	}
	if result.Reason != string(glyph.ReasonNoSeparator) {
		t.Errorf("resolve reason = %q; want %q", result.Reason, glyph.ReasonNoSeparator)
	}
}

func TestRun_Help(t *testing.T) {
	tests := [][]string{
		{"--help"},
		{"-h"},
		{"toc", "--help"},
		{"toc", "-h"},
		{"toc", "pkg", "--help"},
		{"--help", "toc", "pkg"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			code, stdout, stderr := runCLI(args)
			if code != exitOK {
				t.Errorf("code = %d; want %d", code, exitOK)
			}
			if stdout != usageText {
				t.Errorf("stdout = %q; want usageText", stdout)
			}
			if stderr != "" {
				t.Errorf("stderr = %q; want empty", stderr)
			}
		})
	}
}

func TestRun_SuccessOutput(t *testing.T) {
	root := newPipelineFixture(t)

	t.Run("json", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"toc", "pkg", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		if strings.Contains(stdout, `"ok"`) {
			t.Errorf("stdout contains an \"ok\" key on the success path: %q", stdout)
		}
		if !strings.HasSuffix(stdout, "\n") || strings.HasSuffix(stdout, "\n\n") {
			t.Errorf("stdout = %q; want exactly one trailing newline", stdout)
		}
		var answer quarry.DirAnswer
		if err := json.Unmarshal([]byte(stdout), &answer); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
	})

	t.Run("text", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"toc", "pkg", "--text", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		if !strings.HasPrefix(stdout, "pkg") {
			t.Errorf("stdout = %q; want it to start with the directory's own line", stdout)
		}
	})
}

// TestRun_Resolve pins the resolve verb's own pipeline: the glyph-versus-path classification, the
// no-stat rule, the code and message pairing per target/status, and the target-echo asymmetry
// between a glyph target and a path target.
func TestRun_Resolve(t *testing.T) {
	root := newPipelineFixture(t)

	t.Run("glyph-found-free-function", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg/other#Make", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stdout = %q, stderr = %q; want %d", code, stdout, stderr, exitOK)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Status != quarry.StatusFound {
			t.Errorf("status = %q; want %q", result.Status, quarry.StatusFound)
		}
	})

	t.Run("glyph-unit-found-member-missing", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg/other#Nope", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Status != quarry.StatusNotFound {
			t.Errorf("status = %q; want %q", result.Status, quarry.StatusNotFound)
		}
		if result.Unit != quarry.StatusFound {
			t.Errorf("unit = %q; want %q", result.Unit, quarry.StatusFound)
		}
	})

	t.Run("glyph-unit-not-found", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg/missing#Foo", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Status != quarry.StatusNotFound {
			t.Errorf("status = %q; want %q", result.Status, quarry.StatusNotFound)
		}
		if result.Unit != quarry.StatusNotFound {
			t.Errorf("unit = %q; want %q", result.Unit, quarry.StatusNotFound)
		}
	})

	t.Run("glyph-grammar-rejection", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg/other#1bad", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty — a grammar rejection is a payload, not the failure envelope", stderr)
		}
		if strings.Contains(stdout, `"ok"`) {
			t.Errorf("stdout = %q; must not be the failure envelope", stdout)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Error == "" {
			t.Errorf("error = %q; want non-empty", result.Error)
		}
		if result.Reason == "" {
			t.Errorf("reason = %q; want non-empty", result.Reason)
		}
	})

	t.Run("self-glyph-directory", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg#", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Status != quarry.StatusFound {
			t.Errorf("status = %q; want %q", result.Status, quarry.StatusFound)
		}
		if result.Listing == nil {
			t.Fatalf("listing = nil; want a directory answer")
		}
	})

	t.Run("self-glyph-file", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg/doc.go#", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Status != quarry.StatusFound {
			t.Errorf("status = %q; want %q", result.Status, quarry.StatusFound)
		}
		if result.Listing == nil {
			t.Fatalf("listing = nil; want a directory answer")
		}
		if result.Listing.Dir != "pkg" {
			t.Errorf("listing.Dir = %q; want %q", result.Listing.Dir, "pkg")
		}
		if len(result.Listing.Files) != 1 {
			t.Fatalf("listing.Files = %+v; want exactly one file entry", result.Listing.Files)
		}
	})

	t.Run("self-glyph-missing-file", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg/missing.go#", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Status != quarry.StatusNotFound {
			t.Errorf("status = %q; want %q", result.Status, quarry.StatusNotFound)
		}
		if result.Unit != quarry.StatusNotFound {
			t.Errorf("unit = %q; want %q", result.Unit, quarry.StatusNotFound)
		}
	})

	// self-glyph-external-test-unit pins the same collision unitDirs's own doc comment records: no
	// "pkg_test" directory exists on disk, but unitDirs strips the "_test" suffix and finds "pkg",
	// so the self glyph reports not_found with unit: found.
	t.Run("self-glyph-external-test-unit", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg_test#", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Status != quarry.StatusNotFound {
			t.Errorf("status = %q; want %q", result.Status, quarry.StatusNotFound)
		}
		if result.Unit != quarry.StatusFound {
			t.Errorf("unit = %q; want %q", result.Unit, quarry.StatusFound)
		}
	})

	// bare-path-rejected-missing-name pins that a bare path given to resolve is a grammar
	// rejection now, not an engine not-found answer: the grammar is the only classifier, so a
	// string with no "#" never reaches the engine at all.
	t.Run("bare-path-rejected-missing-name", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "nope", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Reason != string(glyph.ReasonNoSeparator) {
			t.Errorf("reason = %q; want %q", result.Reason, glyph.ReasonNoSeparator)
		}
	})

	// bare-path-rejected-dot-dot pins the same rejection for a target that used to escape the
	// root through path arithmetic: ".." carries no "#" either, so it is rejected by the grammar
	// before any path conversion is attempted, and the engine's own outside-repository rule is
	// never reached from resolve any more.
	t.Run("bare-path-rejected-dot-dot", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "..", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Reason != string(glyph.ReasonNoSeparator) {
			t.Errorf("reason = %q; want %q", result.Reason, glyph.ReasonNoSeparator)
		}
	})

	// two-separator-target pins that a target with more than one "#" is a payload-carried
	// rejection, exit 1, exactly like every other grammar rejection.
	t.Run("two-separator-target", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "a#b#c", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var result quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if result.Reason != string(glyph.ReasonMultipleSeparators) {
			t.Errorf("reason = %q; want %q", result.Reason, glyph.ReasonMultipleSeparators)
		}
	})

	t.Run("text-flag", func(t *testing.T) {
		cases := []struct {
			name string
			args []string
		}{
			{"glyph", []string{"resolve", "pkg/other#Make", "--root", root}},
			{"self-glyph-directory", []string{"resolve", "pkg#", "--root", root}},
			{"self-glyph-file", []string{"resolve", "pkg/doc.go#", "--root", root}},
			{"self-glyph-missing-file", []string{"resolve", "pkg/missing.go#", "--root", root}},
			{"self-glyph-external-test-unit", []string{"resolve", "pkg_test#", "--root", root}},
			{"bare-path-rejected", []string{"resolve", "nope", "--root", root}},
			{"two-separator-target", []string{"resolve", "a#b#c", "--root", root}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				jsonCode, jsonOut, _ := runCLI(tc.args)
				var result quarry.ResolveResult
				if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
					t.Fatalf("json.Unmarshal(%q): %v", jsonOut, err)
				}
				want := quarry.RenderResolveText(result)

				textArgs := append(append([]string{}, tc.args...), "--text")
				code, stdout, stderr := runCLI(textArgs)
				if code != jsonCode {
					t.Fatalf("code = %d, stderr = %q; want %d (the same code the JSON run returned)", code, stderr, jsonCode)
				}
				if stdout != want {
					t.Errorf("stdout = %q; want %q", stdout, want)
				}
			})
		}
	})

	// target-field-always-verbatim replaces the old target-echo-asymmetry test: card 23 removed
	// the relativisation that used to contrast a path target, echoed as its repository-relative
	// form, against a glyph target, echoed verbatim. Both now echo the argument verbatim, so this
	// asserts a rule rather than a contrast.
	t.Run("target-field-always-verbatim", func(t *testing.T) {
		abs := filepath.Join(root, "pkg")
		code, stdout, stderr := runCLI([]string{"resolve", abs, "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitNegative)
		}
		var pathResult quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &pathResult); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if pathResult.Target != abs {
			t.Errorf("target = %q; want %q, the absolute argument as given", pathResult.Target, abs)
		}
		if pathResult.Reason != string(glyph.ReasonNoSeparator) {
			t.Errorf("reason = %q; want %q", pathResult.Reason, glyph.ReasonNoSeparator)
		}

		code, stdout, stderr = runCLI([]string{"resolve", "pkg/other#Make", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		var glyphResult quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &glyphResult); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if glyphResult.Target != "pkg/other#Make" {
			t.Errorf("target = %q; want it to echo the argument verbatim", glyphResult.Target)
		}
	})
}

// TestRun_Expand pins the expand verb's own pipeline: the errors.As-based error classification and
// its exact quarry-authored sentences, the found/not-found dispositions, and the failure envelope
// discipline that holds even under --text.
func TestRun_Expand(t *testing.T) {
	root := newPipelineFixture(t)

	t.Run("type-with-members", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg/other#Widget", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stdout = %q, stderr = %q; want %d", code, stdout, stderr, exitOK)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var answer quarry.ExpandAnswer
		if err := json.Unmarshal([]byte(stdout), &answer); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if answer.Status != quarry.StatusFound {
			t.Errorf("status = %q; want %q", answer.Status, quarry.StatusFound)
		}
		if answer.Head == nil {
			t.Errorf("head = nil; want a head symbol")
		}
		if len(answer.Members) == 0 {
			t.Errorf("members = %+v; want at least one member", answer.Members)
		}
	})

	t.Run("type-with-no-members", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg/other#Empty", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stdout = %q, stderr = %q; want %d", code, stdout, stderr, exitOK)
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if raw["status"] != "found" {
			t.Errorf(`payload["status"] = %v; want "found"`, raw["status"])
		}
		if _, ok := raw["members"]; ok {
			t.Errorf("payload = %v; want no %q key for a type with no members", raw, "members")
		}
	})

	t.Run("unit-found-member-missing", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg/other#Nope", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var answer quarry.ExpandAnswer
		if err := json.Unmarshal([]byte(stdout), &answer); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if answer.Status != quarry.StatusNotFound {
			t.Errorf("status = %q; want %q", answer.Status, quarry.StatusNotFound)
		}
		if answer.Unit != quarry.StatusFound {
			t.Errorf("unit = %q; want %q", answer.Unit, quarry.StatusFound)
		}
	})

	t.Run("not-a-type", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg/other#Make", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		envErr := failureEnvelope(t, stdout)
		want := "expand pkg/other#Make: not a type, kind function"
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

	// grammar-rejection-with-separator pins that the message carries the grammar's full sentence,
	// not the bare reason word: the regression card 27 guards against is exactly a message that
	// ends in the raw reason constant instead.
	t.Run("grammar-rejection-with-separator", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg/other#1bad", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		_, parseErr := glyph.Parse(glyph.Go, "pkg/other#1bad")
		var pe *glyph.ParseError
		if !errors.As(parseErr, &pe) {
			t.Fatalf("glyph.Parse(%q) error = %v; want a *glyph.ParseError", "pkg/other#1bad", parseErr)
		}
		want := "expand: " + pe.Error()
		envErr := failureEnvelope(t, stdout)
		if envErr != want {
			t.Errorf("error = %q; want %q", envErr, want)
		}
		if envErr == "expand pkg/other#1bad: "+string(glyph.ReasonMemberNotIdentifier) {
			t.Errorf("error = %q; must not be the bare reason word form card 27 deleted", envErr)
		}
		if strings.Contains(stderr, usageText) {
			t.Errorf("stderr = %q; must not carry usage text", stderr)
		}
	})

	// bare-path-rejected pins that expand no longer has a usage gate of its own: a target with no
	// "#" now reaches the facade, and glyph.Parse rejects it exactly as it would for resolve. The
	// grammar's own full sentence is what runExpand carries, not the bare reason word.
	t.Run("bare-path-rejected", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "no-separator", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		_, parseErr := glyph.Parse(glyph.Go, "no-separator")
		var pe *glyph.ParseError
		if !errors.As(parseErr, &pe) {
			t.Fatalf("glyph.Parse(%q) error = %v; want a *glyph.ParseError", "no-separator", parseErr)
		}
		want := "expand: " + pe.Error()
		envErr := failureEnvelope(t, stdout)
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

	// two-separator-target pins the changed message for a target the grammar rejects with more
	// than one "#": the full sentence, not the bare reason word.
	t.Run("two-separator-target", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "a#b#c", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		_, parseErr := glyph.Parse(glyph.Go, "a#b#c")
		var pe *glyph.ParseError
		if !errors.As(parseErr, &pe) {
			t.Fatalf("glyph.Parse(%q) error = %v; want a *glyph.ParseError", "a#b#c", parseErr)
		}
		want := "expand: " + pe.Error()
		envErr := failureEnvelope(t, stdout)
		if envErr != want {
			t.Errorf("error = %q; want %q", envErr, want)
		}
		if stderr != want+"\n" {
			t.Errorf("stderr = %q; want %q", stderr, want+"\n")
		}
	})

	// self-glyph pins the *quarry.SelfGlyphError branch runExpand gained: a self glyph exits 1
	// with the failure envelope on stdout and the message on stderr, spelled from the value's ID
	// field rather than the engine's own error text.
	t.Run("self-glyph", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg#", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		envErr := failureEnvelope(t, stdout)
		want := "expand pkg#: not a type, self"
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

	t.Run("text-flag-over-found", func(t *testing.T) {
		_, jsonOut, _ := runCLI([]string{"expand", "pkg/other#Widget", "--root", root})
		var answer quarry.ExpandAnswer
		if err := json.Unmarshal([]byte(jsonOut), &answer); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", jsonOut, err)
		}
		want := quarry.RenderExpandText(answer)

		code, stdout, stderr := runCLI([]string{"expand", "pkg/other#Widget", "--text", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		if stdout != want {
			t.Errorf("stdout = %q; want %q", stdout, want)
		}
		if !strings.Contains(stdout, "\n\n") {
			t.Errorf("stdout = %q; want a blank line between the head and the members", stdout)
		}
	})

	t.Run("text-flag-over-failure", func(t *testing.T) {
		code, stdout, _ := runCLI([]string{"expand", "pkg/other#Make", "--text", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		envErr := failureEnvelope(t, stdout)
		want := fmt.Sprintf(`{"ok":false,"error":%q}`+"\n", envErr)
		if stdout != want {
			t.Errorf("stdout = %q; want %q (the JSON failure envelope, even under --text)", stdout, want)
		}
	})
}

// TestRun_GlyphsView covers runTOC's glyphs-view branch with the newPipelineFixture tree and no
// Loomyard checkout: this is not a duplicate of TestGlyphsIsByteIdenticalToItsExpansion or of
// batch 3's goldens, both of which skip when LADDER_LOOMYARD_REPO is unset. Without the cases
// below, nothing on a machine with no Loomyard checkout asserts that this branch renders the
// glyphs view at all.
func TestRun_GlyphsView(t *testing.T) {
	root := newPipelineFixture(t)

	// assertGlyphsShape asserts the four properties every glyphs-view invocation below must have:
	// exitOK, empty stderr, and — for JSON — the glyphs envelope's key set, or — for text — the
	// documented line grammar with no complete-view lines mixed in.
	assertGlyphsShape := func(t *testing.T, args []string, wantText bool) {
		t.Helper()
		code, stdout, stderr := runCLI(args)
		if code != exitOK {
			t.Fatalf("Run(%v) code = %d, stdout = %q, stderr = %q; want %d", args, code, stdout, stderr, exitOK)
		}
		if stderr != "" {
			t.Errorf("Run(%v) stderr = %q; want empty", args, stderr)
		}
		if !wantText {
			var raw map[string]any
			if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
				t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
			}
			for _, key := range []string{"target", "symbols"} {
				if _, ok := raw[key]; !ok {
					t.Errorf("Run(%v) payload = %v; want key %q", args, raw, key)
				}
			}
			for _, key := range []string{"dirs", "files", "signature"} {
				if _, ok := raw[key]; ok {
					t.Errorf("Run(%v) payload = %v; want no key %q", args, raw, key)
				}
			}
			return
		}
		for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "[incomplete] ") {
				continue
			}
			fields := strings.SplitN(line, " ", 3)
			if len(fields) != 3 || !strings.Contains(fields[0], ":") || !strings.Contains(fields[0], "-") {
				t.Errorf("Run(%v) stdout line = %q; want the <file>:<start>-<end> <kind> <id> shape", args, line)
			}
		}
		if strings.Contains(stdout, "(sig ") {
			t.Errorf("Run(%v) stdout = %q; want no signature line", args, stdout)
		}
	}

	t.Run("preset-directory", func(t *testing.T) {
		assertGlyphsShape(t, []string{"glyphs", "pkg", "--root", root}, false)
		assertGlyphsShape(t, []string{"glyphs", "pkg", "--text", "--root", root}, true)
	})

	t.Run("preset-file", func(t *testing.T) {
		assertGlyphsShape(t, []string{"glyphs", "pkg/doc.go", "--root", root}, false)
		assertGlyphsShape(t, []string{"glyphs", "pkg/doc.go", "--text", "--root", root}, true)
	})

	t.Run("explicit-directory", func(t *testing.T) {
		assertGlyphsShape(t, []string{"toc", "pkg", "--view", "glyphs", "--depth", "all", "--symbols", "--root", root}, false)
		assertGlyphsShape(t, []string{"toc", "pkg", "--view", "glyphs", "--depth", "all", "--symbols", "--text", "--root", root}, true)
	})

	t.Run("explicit-file", func(t *testing.T) {
		assertGlyphsShape(t, []string{"toc", "pkg/doc.go", "--view", "glyphs", "--depth", "all", "--symbols", "--root", root}, false)
		assertGlyphsShape(t, []string{"toc", "pkg/doc.go", "--view", "glyphs", "--depth", "all", "--symbols", "--text", "--root", root}, true)
	})

	// no-symbols-flag-still-populates is the machine-independent counterpart of the depth
	// golden's own assertion that the view's symbols default works: pkg/other carries a free
	// function, a type with a method, and a type with no members, so a non-empty symbol list here
	// proves --view glyphs's default reached the query with no --symbols flag on the command line.
	t.Run("no-symbols-flag-still-populates", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"toc", "pkg/other", "--view", "glyphs", "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		var payload struct {
			Symbols []quarry.Symbol `json:"symbols"`
		}
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if len(payload.Symbols) == 0 {
			t.Errorf("symbols = %+v; want a non-empty list", payload.Symbols)
		}
	})
}

// TestRun_ViewFullIsByteIdenticalToViewless pins the view-vocabulary promise that no other card
// asserts: an absent --view means "full", so "toc --view full <target>" must produce byte-identical
// stdout to a viewless "toc <target>", for both a file and a directory target, in both formats.
func TestRun_ViewFullIsByteIdenticalToViewless(t *testing.T) {
	root := newPipelineFixture(t)

	targets := []string{"pkg", "pkg/doc.go"}
	for _, target := range targets {
		for _, withText := range []bool{false, true} {
			name := target
			if withText {
				name += "-text"
			}
			t.Run(name, func(t *testing.T) {
				viewlessArgs := []string{"toc", target, "--root", root}
				viewFullArgs := []string{"toc", target, "--view", "full", "--root", root}
				if withText {
					viewlessArgs = append(viewlessArgs, "--text")
					viewFullArgs = append(viewFullArgs, "--text")
				}

				viewlessCode, viewlessOut, viewlessErr := runCLI(viewlessArgs)
				fullCode, fullOut, fullErr := runCLI(viewFullArgs)

				if viewlessCode != fullCode {
					t.Errorf("code = %d; want %d (viewless)", fullCode, viewlessCode)
				}
				if viewlessOut != fullOut {
					t.Errorf("stdout = %q; want %q (viewless, byte-identical)", fullOut, viewlessOut)
				}
				if viewlessErr != fullErr {
					t.Errorf("stderr = %q; want %q (viewless, byte-identical)", fullErr, viewlessErr)
				}
			})
		}
	}
}

func TestRun_FlagPassThrough(t *testing.T) {
	root := newPipelineFixture(t)

	t.Run("depth-all-reaches-nested-dirs", func(t *testing.T) {
		_, stdout, _ := runCLI([]string{"toc", ".", "--depth", "all", "--root", root})
		var answer quarry.DirAnswer
		if err := json.Unmarshal([]byte(stdout), &answer); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if findDir(answer, "pkg/sub") == nil {
			t.Errorf("answer = %+v; want a nested %q dir entry reachable with --depth all", answer, "pkg/sub")
		}
	})

	t.Run("symbols-reaches-named-subdirectory", func(t *testing.T) {
		// pkg/sub, not the fixture root: a-symbols-key-is-never-guaranteed-by---symbols means a Go
		// file directly under the root has the empty, unspellable unit and never carries symbols
		// even with --symbols.
		_, stdout, _ := runCLI([]string{"toc", "pkg/sub", "--symbols", "--root", root})
		var answer quarry.DirAnswer
		if err := json.Unmarshal([]byte(stdout), &answer); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if len(answer.Files) != 1 || answer.Files[0].Symbols == nil {
			t.Fatalf("answer = %+v; want exactly one file entry with a non-nil Symbols", answer)
		}
		if len(*answer.Files[0].Symbols) == 0 {
			t.Errorf("answer.Files[0].Symbols = %+v; want at least one symbol", *answer.Files[0].Symbols)
		}
	})
}

// findDir searches a's tree, depth-first, for a DirAnswer whose Dir equals dir, returning nil
// when none is found.
func findDir(a quarry.DirAnswer, dir string) *quarry.DirAnswer {
	if a.Dir == dir {
		return &a
	}
	for _, child := range a.Dirs {
		if found := findDir(child, dir); found != nil {
			return found
		}
	}
	return nil
}

func TestCodeForTOCError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, exitOK},
		{"wrapped-not-found", fmt.Errorf("engine: resolve target %q: %w", "x", quarry.ErrTargetNotFound), exitNegative},
		{"wrapped-outside-repo", fmt.Errorf("engine: resolve target %q: %w", "x", quarry.ErrTargetOutsideRepo), exitNegative},
		{"arbitrary-error", errors.New("boom"), exitInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeForTOCError(tt.err); got != tt.want {
				t.Errorf("codeForTOCError(%v) = %d; want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestCodeForResolveResult(t *testing.T) {
	tests := []struct {
		name string
		r    quarry.ResolveResult
		want int
	}{
		{"found", quarry.ResolveResult{Status: quarry.StatusFound}, exitOK},
		{"multipart", quarry.ResolveResult{Status: quarry.StatusMultipart}, exitOK},
		{"not-found", quarry.ResolveResult{Status: quarry.StatusNotFound}, exitNegative},
		{"ambiguous", quarry.ResolveResult{Status: quarry.StatusAmbiguous}, exitNegative},
		{"empty-status-pre-resolution-rejection", quarry.ResolveResult{Status: "", Error: "boom"}, exitNegative},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeForResolveResult(tt.r); got != tt.want {
				t.Errorf("codeForResolveResult(%+v) = %d; want %d", tt.r, got, tt.want)
			}
		})
	}
}

func TestCodeForExpandAnswer(t *testing.T) {
	tests := []struct {
		name string
		a    quarry.ExpandAnswer
		want int
	}{
		{"found", quarry.ExpandAnswer{Status: quarry.StatusFound}, exitOK},
		{"not-found", quarry.ExpandAnswer{Status: quarry.StatusNotFound}, exitNegative},
		{"ambiguous", quarry.ExpandAnswer{Status: quarry.StatusAmbiguous}, exitNegative},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeForExpandAnswer(tt.a); got != tt.want {
				t.Errorf("codeForExpandAnswer(%+v) = %d; want %d", tt.a, got, tt.want)
			}
		})
	}
}

// deltaCLIFixture is a throwaway git repository, built fresh under t.TempDir(), that TestRun_Delta
// and its neighbours drive Run's own entry point against. This is a deliberate per-package copy of
// quarry/delta_test.go's own deltaFixture, because Go test helpers are not importable across
// packages.
type deltaCLIFixture struct {
	t    *testing.T
	root string
}

// newDeltaCLIFixture initialises a repository under a fresh temporary directory, with a fixed
// identity and a fixed default branch name so no machine's global git configuration can change the
// fixture's behaviour. It skips the whole test, with the reason, when no git binary is available.
func newDeltaCLIFixture(t *testing.T) *deltaCLIFixture {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found on this machine")
	}

	f := &deltaCLIFixture{t: t, root: t.TempDir()}
	f.git("init", "--quiet", "--initial-branch=main")
	f.git("config", "user.name", "quarry-cli-delta-fixture")
	f.git("config", "user.email", "quarry-cli-delta-fixture@example.com")
	return f
}

// git runs one git invocation against the fixture's root, failing the test immediately on error: a
// fixture-construction step that fails is a broken test, never a normal state worth skipping.
func (f *deltaCLIFixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", f.root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// write writes content to path, repository-relative, creating parent directories as needed. It
// does not stage the write.
func (f *deltaCLIFixture) write(path, content string) {
	f.t.Helper()
	full := filepath.Join(f.root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		f.t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		f.t.Fatalf("write %q: %v", full, err)
	}
}

// commit stages every pending change and commits it with message, returning the resulting
// commit's identifier.
func (f *deltaCLIFixture) commit(message string) string {
	f.git("add", "-A")
	f.git("commit", "--quiet", "-m", message)
	return f.git("rev-parse", "HEAD")
}

// writeAndCommit writes content to path and commits it in one step, returning the resulting
// commit's identifier.
func (f *deltaCLIFixture) writeAndCommit(path, content, message string) string {
	f.write(path, content)
	return f.commit(message)
}

// removeUnstaged deletes path from the working tree without staging the deletion, leaving it
// present in the index but absent from disk.
func (f *deltaCLIFixture) removeUnstaged(path string) {
	f.t.Helper()
	full := filepath.Join(f.root, filepath.FromSlash(path))
	if err := os.Remove(full); err != nil {
		f.t.Fatalf("remove %q: %v", full, err)
	}
}

// unmergedPath produces a conflicted path reachable on the working-tree side during a merge,
// returning the base commit it built the conflict from: it commits a base version of path,
// changes it on a side branch, changes it again on main, then attempts to merge the side branch
// into main and expects, rather than resolves, the resulting conflict.
func (f *deltaCLIFixture) unmergedPath(path string) string {
	f.t.Helper()
	base := f.writeAndCommit(path, "package pkg\n\nfunc Base() {}\n", "base commit for "+path)
	f.git("checkout", "--quiet", "-b", "delta-fixture-conflict")
	f.writeAndCommit(path, "package pkg\n\nfunc Side() {}\n", "conflict branch change")
	f.git("checkout", "--quiet", "main")
	f.writeAndCommit(path, "package pkg\n\nfunc Main() {}\n", "main branch change")

	cmd := exec.Command("git", "-C", f.root, "merge", "--quiet", "--no-edit", "delta-fixture-conflict")
	// The merge is expected to fail with a conflict on path; a merge that succeeds cleanly, leaving
	// no unmerged path at all, is the fixture-construction failure worth stopping the test over.
	if err := cmd.Run(); err == nil {
		f.t.Fatalf("merge of delta-fixture-conflict into main succeeded cleanly; wanted a conflict on %q", path)
	}
	return base
}

// deltaFileByPath returns the DeltaFile in files whose Path equals path, and whether it was found.
func deltaFileByPath(files []quarry.DeltaFile, path string) (quarry.DeltaFile, bool) {
	for _, df := range files {
		if df.Path == path {
			return df, true
		}
	}
	return quarry.DeltaFile{}, false
}

// hasSymbolIDCLI reports whether syms contains a Symbol whose ID equals id. This is a deliberate
// per-package copy of quarry/delta_test.go's own hasSymbolID, for the same reason deltaCLIFixture
// is.
func hasSymbolIDCLI(syms []quarry.Symbol, id string) bool {
	for _, s := range syms {
		if s.ID == id {
			return true
		}
	}
	return false
}

// decodeGitDeltaAnswer decodes stdout as a quarry.GitDeltaAnswer, failing the test on a decode
// error.
func decodeGitDeltaAnswer(t *testing.T, stdout string) quarry.GitDeltaAnswer {
	t.Helper()
	var answer quarry.GitDeltaAnswer
	if err := json.Unmarshal([]byte(stdout), &answer); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
	}
	return answer
}

// TestRun_Delta pins the delta verb's own pipeline: the exit-code contract that has no negative
// answer, the two path-taking-verb rejections it shares with toc, the three usage dispositions the
// facade's git-error identity maps to quarry's own sentences, the text view, and the no-stat rule
// that lets it report a path that no longer exists but did at the from revision.
func TestRun_Delta(t *testing.T) {
	t.Run("EmptyDeltaIsSuccess", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		from := f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc Foo() {}\n", "add Foo")

		code, stdout, stderr := runCLI([]string{"delta", ".", "--from", from, "--root", f.root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		answer := decodeGitDeltaAnswer(t, stdout)
		if len(answer.Files) != 0 {
			t.Errorf("Files = %+v; want empty", answer.Files)
		}
	})

	t.Run("ErrorDispositionEntryIsStillSuccess", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		base := f.unmergedPath("unmerged.go")

		code, stdout, stderr := runCLI([]string{"delta", ".", "--from", base, "--root", f.root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		answer := decodeGitDeltaAnswer(t, stdout)
		df, ok := deltaFileByPath(answer.Files, "unmerged.go")
		if !ok {
			t.Fatalf("Files = %+v; want an entry for %q", answer.Files, "unmerged.go")
		}
		if df.Disposition != quarry.DispositionError {
			t.Errorf("Disposition = %q; want %q", df.Disposition, quarry.DispositionError)
		}
		if df.Error == "" {
			t.Error("Error is empty; want it set for the unmerged path")
		}
	})

	t.Run("UnresolvableFromRevisionIsUsageError", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		f.writeAndCommit("a.go", "package pkg\n\nfunc A() {}\n", "base")

		code, stdout, stderr := runCLI([]string{"delta", ".", "--from", "does-not-exist-rev", "--root", f.root})
		if code != exitUsage {
			t.Fatalf("code = %d; want %d", code, exitUsage)
		}
		envErr := failureEnvelope(t, stdout)
		want := "delta: unknown revision does-not-exist-rev"
		if envErr != want {
			t.Errorf("error = %q; want %q", envErr, want)
		}
		if !strings.Contains(stderr, usageText) {
			t.Errorf("stderr = %q; want it to carry the usage text", stderr)
		}
		if strings.Contains(envErr, "gitsrc:") {
			t.Errorf("error = %q; must not carry git's own message", envErr)
		}
	})

	t.Run("RootIsSubdirectoryOfRepositoryIsUsageError", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		f.writeAndCommit("sub/a.go", "package sub\n\nfunc A() {}\n", "base")
		subRoot := filepath.Join(f.root, "sub")

		code, stdout, stderr := runCLI([]string{"delta", ".", "--from", "HEAD", "--root", subRoot})
		if code != exitUsage {
			t.Fatalf("code = %d; want %d", code, exitUsage)
		}
		envErr := failureEnvelope(t, stdout)
		if !strings.HasPrefix(envErr, "delta: root ") || !strings.Contains(envErr, "is not the repository top level") {
			t.Errorf("error = %q; want quarry's own root-not-top-level sentence", envErr)
		}
		if strings.HasPrefix(envErr, "internal error: ") {
			t.Errorf("error = %q; must not be the internal code with git's own message", envErr)
		}
		if !strings.Contains(stderr, usageText) {
			t.Errorf("stderr = %q; want it to carry the usage text", stderr)
		}
	})

	t.Run("RootOutsideAnyRepositoryIsUsageError", func(t *testing.T) {
		// A directory outside any git repository, by definition ruling out newDeltaCLIFixture's own
		// tree: t.TempDir() is the same precedent quarry/delta_test.go's own ErrorIdentity case sets
		// for exactly this need.
		root := t.TempDir()

		code, stdout, stderr := runCLI([]string{"delta", ".", "--from", "HEAD", "--root", root})
		if code != exitUsage {
			t.Fatalf("code = %d; want %d", code, exitUsage)
		}
		envErr := failureEnvelope(t, stdout)
		want := "delta: root is not a git repository: " + root
		if envErr != want {
			t.Errorf("error = %q; want %q", envErr, want)
		}
		if !strings.Contains(stderr, usageText) {
			t.Errorf("stderr = %q; want it to carry the usage text", stderr)
		}
	})

	t.Run("TargetEscapingRootIsNegative", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		f.writeAndCommit("a.go", "package pkg\n\nfunc A() {}\n", "base")

		code, stdout, _ := runCLI([]string{"delta", "..", "--from", "HEAD", "--root", f.root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		envErr := failureEnvelope(t, stdout)
		if want := "target outside repository: .."; envErr != want {
			t.Errorf("error = %q; want %q", envErr, want)
		}
	})

	t.Run("TargetWithSeparatorIsUsageError", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc A() {}\n", "base")

		code, stdout, stderr := runCLI([]string{"delta", "pkg/other#Make", "--from", "HEAD", "--root", f.root})
		if code != exitUsage {
			t.Fatalf("code = %d; want %d", code, exitUsage)
		}
		envErr := failureEnvelope(t, stdout)
		want := `target contains the glyph separator "#": pkg/other#Make`
		if envErr != want {
			t.Errorf("error = %q; want %q", envErr, want)
		}
		if !strings.Contains(stderr, usageText) {
			t.Errorf("stderr = %q; want it to carry the usage text", stderr)
		}
	})

	t.Run("TextFlagProducesTextView", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		from := f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc Foo() {}\n", "add Foo")

		jsonCode, jsonOut, _ := runCLI([]string{"delta", ".", "--from", from, "--root", f.root})
		answer := decodeGitDeltaAnswer(t, jsonOut)
		want := quarry.RenderDeltaText(answer)

		code, stdout, stderr := runCLI([]string{"delta", ".", "--from", from, "--text", "--root", f.root})
		if code != jsonCode {
			t.Fatalf("code = %d, stderr = %q; want %d (the same code the JSON run returned)", code, stderr, jsonCode)
		}
		if stdout != want {
			t.Errorf("stdout = %q; want %q", stdout, want)
		}
	})

	t.Run("PathspecMatchingNothingIsSuccessWithEmptyDelta", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		from := f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc Foo() {}\n", "add Foo")

		code, stdout, stderr := runCLI([]string{"delta", "no-such-dir", "--from", from, "--root", f.root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		answer := decodeGitDeltaAnswer(t, stdout)
		if len(answer.Files) != 0 {
			t.Errorf("Files = %+v; want empty for a pathspec matching nothing", answer.Files)
		}
	})

	// NoStatOnDeletedTarget proves this verb performs no stat: sub/gone.go exists at the from
	// revision and is removed, unstaged, from the working tree before Run is ever called, so a stat
	// on the target itself would report it missing. The success code and the path's own symbol in
	// the deleted array are what show DeltaGit answered from git history rather than from a stat.
	t.Run("NoStatOnDeletedTarget", func(t *testing.T) {
		f := newDeltaCLIFixture(t)
		from := f.writeAndCommit("sub/gone.go", "package sub\n\nfunc Gone() {}\n", "add Gone")
		f.removeUnstaged("sub/gone.go")

		code, stdout, stderr := runCLI([]string{"delta", "sub/gone.go", "--from", from, "--root", f.root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		answer := decodeGitDeltaAnswer(t, stdout)
		df, ok := deltaFileByPath(answer.Files, "sub/gone.go")
		if !ok {
			t.Fatalf("Files = %+v; want an entry for %q", answer.Files, "sub/gone.go")
		}
		if df.Disposition != quarry.DispositionRemoved {
			t.Errorf("Disposition = %q; want %q", df.Disposition, quarry.DispositionRemoved)
		}
		if !hasSymbolIDCLI(answer.Deleted, "sub#Gone") {
			t.Errorf("Deleted = %+v; want it to contain %q", answer.Deleted, "sub#Gone")
		}
	})

	// GitFailureForNonUsageReasonIsInternal covers the one remaining case discussion.md's own
	// command-line list names: a git invocation failing for a reason that is not a usage error. Only
	// the from-side blob's own loose object file is made unreadable, rather than the whole object
	// store, so the top-level and revision checks -- and the name-status diff itself, which compares
	// tree entries and never opens blob content -- have already passed by the time the blob read
	// fails. Skipped when the process can read the path regardless, which is the normal state when
	// tests run as a privileged user.
	t.Run("GitFailureForNonUsageReasonIsInternal", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as a privileged user for whom chmod 0000 does not block reads")
		}
		f := newDeltaCLIFixture(t)
		from := f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc Foo() {}\n", "add Foo")
		f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc Foo() {}\n\nfunc Bar() {}\n", "add Bar")

		hash := f.git("rev-parse", from+":pkg/a.go")
		obj := filepath.Join(f.root, ".git", "objects", hash[:2], hash[2:])
		if err := os.Chmod(obj, 0o000); err != nil {
			t.Fatalf("os.Chmod(%q, 0000) failed: %v", obj, err)
		}
		t.Cleanup(func() { _ = os.Chmod(obj, 0o644) })
		if err := exec.Command("git", "-C", f.root, "cat-file", "-p", hash).Run(); err == nil {
			t.Skip("chmod 0000 did not block reads in this environment")
		}

		code, stdout, stderr := runCLI([]string{"delta", ".", "--from", from, "--root", f.root})
		if code != exitInternal {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitInternal)
		}
		envErr := failureEnvelope(t, stdout)
		if !strings.HasPrefix(envErr, "internal error: ") {
			t.Errorf("error = %q; want it to start with %q", envErr, "internal error: ")
		}
		if envErr == "internal error: " {
			t.Errorf("error = %q; want git's own message carried whole behind the prefix", envErr)
		}
	})
}

// TestRun_DeltaTargetResolution pins the shared rule runDelta's own doc comment states: a lone dot
// given to this verb from a subdirectory scopes to that subdirectory, producing the same scope the
// table-of-contents verb's lone dot produces from the same directory.
func TestRun_DeltaTargetResolution(t *testing.T) {
	f := newDeltaCLIFixture(t)
	from := f.writeAndCommit("pkg/sub/a.go", "package sub\n\nfunc A() {}\n", "base")
	f.writeAndCommit("pkg/sub/a.go", "package sub\n\nfunc A() {}\n\nfunc B() {}\n", "add B")

	subdir := filepath.Join(f.root, "pkg", "sub")

	// Neither call passes --root: base must come from Run's own cwd discovery, which is the one
	// thing this test exists to pin. Passing --root would set base to the fixture root regardless of
	// cwd and defeat the point.
	var deltaOut, tocOut bytes.Buffer
	var deltaErr, tocErr bytes.Buffer
	code := runFromDir(t, subdir, []string{"delta", ".", "--from", from}, &deltaOut, &deltaErr)
	if code != exitOK {
		t.Fatalf("delta code = %d, stderr = %q; want %d", code, deltaErr.String(), exitOK)
	}
	answer := decodeGitDeltaAnswer(t, deltaOut.String())
	df, ok := deltaFileByPath(answer.Files, "pkg/sub/a.go")
	if !ok {
		t.Fatalf("Files = %+v; want an entry for %q, proving the lone dot scoped to pkg/sub", answer.Files, "pkg/sub/a.go")
	}
	if df.Disposition != quarry.DispositionChanged {
		t.Errorf("Disposition = %q; want %q", df.Disposition, quarry.DispositionChanged)
	}

	tocCode := runFromDir(t, subdir, []string{"toc", "."}, &tocOut, &tocErr)
	if tocCode != exitOK {
		t.Fatalf("toc code = %d, stderr = %q; want %d", tocCode, tocErr.String(), exitOK)
	}
	var tocAnswer quarry.DirAnswer
	if err := json.Unmarshal(tocOut.Bytes(), &tocAnswer); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", tocOut.String(), err)
	}
	if tocAnswer.Dir != "pkg/sub" {
		t.Errorf("toc's lone-dot Dir = %q; want %q, the same scope delta's lone dot produced", tocAnswer.Dir, "pkg/sub")
	}
}

// runFromDir runs Run with the process working directory changed to dir for the duration of the
// call, restoring it afterwards. This is the one test in this file that changes the working
// directory, which is why it is isolated in its own helper rather than folded into runCLI: every
// other test in this package passes an explicit --root so Run's own os.Getwd call never matters.
func runFromDir(t *testing.T, dir string, args []string, stdout, stderr *bytes.Buffer) int {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("os.Chdir(%q) (restore): %v", cwd, err)
		}
	})
	return Run(args, stdout, stderr)
}

func TestCodeForDeltaError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, exitOK},
		{"unknown-revision", &quarry.UnknownRevisionError{Rev: "x"}, exitUsage},
		{"wrapped-unknown-revision", fmt.Errorf("wrap: %w", &quarry.UnknownRevisionError{Rev: "x"}), exitUsage},
		{"not-a-repository", quarry.ErrNotARepository, exitUsage},
		{"wrapped-not-a-repository", fmt.Errorf("wrap: %w", quarry.ErrNotARepository), exitUsage},
		{"root-not-top-level", &quarry.RootNotTopLevelError{Root: "/a", TopLevel: "/b"}, exitUsage},
		{"wrapped-root-not-top-level", fmt.Errorf("wrap: %w", &quarry.RootNotTopLevelError{Root: "/a", TopLevel: "/b"}), exitUsage},
		{"arbitrary-error", errors.New("boom"), exitInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeForDeltaError(tt.err); got != tt.want {
				t.Errorf("codeForDeltaError(%v) = %d; want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestCodeForExpandError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, exitOK},
		{"not-a-type", &quarry.NotATypeError{ID: "x#Y", Kind: quarry.KindFunction}, exitNegative},
		{
			"wrapped-not-a-type",
			fmt.Errorf("wrap: %w", &quarry.NotATypeError{ID: "x#Y", Kind: quarry.KindFunction}),
			exitNegative,
		},
		{"parse-error", &glyph.ParseError{Reason: glyph.ReasonNoSeparator}, exitNegative},
		{
			"wrapped-parse-error",
			fmt.Errorf("wrap: %w", &glyph.ParseError{Reason: glyph.ReasonNoSeparator}),
			exitNegative,
		},
		{"self-glyph-error", &quarry.SelfGlyphError{ID: "x#"}, exitNegative},
		{
			"wrapped-self-glyph-error",
			fmt.Errorf("wrap: %w", &quarry.SelfGlyphError{ID: "x#"}),
			exitNegative,
		},
		{"plain-formatted-error-stands-in-for-invariant-failure", errors.New("engine: expand x#Y: type symbol carries no head span"), exitInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeForExpandError(tt.err); got != tt.want {
				t.Errorf("codeForExpandError(%v) = %d; want %d", tt.err, got, tt.want)
			}
		})
	}
}
