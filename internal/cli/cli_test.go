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

	t.Run("path-directory", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg", "--root", root})
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

	t.Run("path-file", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "pkg/doc.go", "--root", root})
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

	t.Run("path-not-found", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"resolve", "nope", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want empty", stderr)
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if _, ok := raw["unit"]; ok {
			t.Errorf("payload = %v; want no %q key for a path result", raw, "unit")
		}
		if raw["status"] != "not_found" {
			t.Errorf(`payload["status"] = %v; want "not_found"`, raw["status"])
		}
	})

	t.Run("path-escapes-root", func(t *testing.T) {
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
		// This is the one test anywhere in the plan that pins this exact string: the engine's own
		// doubled "engine: ...: engine: ..." wording, carried verbatim into the payload's error
		// field per the overview's payload-error-text Shared Decision.
		want := `engine: resolve target "..": engine: target outside repository`
		if result.Error != want {
			t.Errorf("error = %q; want %q", result.Error, want)
		}
	})

	t.Run("text-flag", func(t *testing.T) {
		cases := []struct {
			name string
			args []string
		}{
			{"glyph", []string{"resolve", "pkg/other#Make", "--root", root}},
			{"directory-path", []string{"resolve", "pkg", "--root", root}},
			{"file-path", []string{"resolve", "pkg/doc.go", "--root", root}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, jsonOut, _ := runCLI(tc.args)
				var result quarry.ResolveResult
				if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
					t.Fatalf("json.Unmarshal(%q): %v", jsonOut, err)
				}
				want := quarry.RenderResolveText(result)

				textArgs := append(append([]string{}, tc.args...), "--text")
				code, stdout, stderr := runCLI(textArgs)
				if code != exitOK {
					t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
				}
				if stdout != want {
					t.Errorf("stdout = %q; want %q", stdout, want)
				}
			})
		}
	})

	t.Run("target-echo-asymmetry", func(t *testing.T) {
		abs := filepath.Join(root, "pkg")
		code, stdout, stderr := runCLI([]string{"resolve", abs, "--root", root})
		if code != exitOK {
			t.Fatalf("code = %d, stderr = %q; want %d", code, stderr, exitOK)
		}
		var pathResult quarry.ResolveResult
		if err := json.Unmarshal([]byte(stdout), &pathResult); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
		}
		if pathResult.Target != "pkg" {
			t.Errorf("target = %q; want %q, the repository-relative form", pathResult.Target, "pkg")
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

	t.Run("grammar-rejection-with-separator", func(t *testing.T) {
		code, stdout, stderr := runCLI([]string{"expand", "pkg/other#1bad", "--root", root})
		if code != exitNegative {
			t.Fatalf("code = %d; want %d", code, exitNegative)
		}
		envErr := failureEnvelope(t, stdout)
		if !strings.HasSuffix(envErr, string(glyph.ReasonMemberNotIdentifier)) {
			t.Errorf("error = %q; want it to end in the reason word %q", envErr, glyph.ReasonMemberNotIdentifier)
		}
		if strings.Contains(stderr, usageText) {
			t.Errorf("stderr = %q; must not carry usage text", stderr)
		}
	})

	t.Run("no-separator", func(t *testing.T) {
		// The parser rejects this before any repository is opened, so no --root is given.
		code, stdout, stderr := runCLI([]string{"expand", "no-separator"})
		if code != exitUsage {
			t.Fatalf("code = %d; want %d", code, exitUsage)
		}
		envErr := failureEnvelope(t, stdout)
		wantStderr := envErr + "\n" + usageText
		if stderr != wantStderr {
			t.Errorf("stderr = %q; want %q", stderr, wantStderr)
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
