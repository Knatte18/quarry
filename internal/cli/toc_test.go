// toc_test.go drives "toc file" and "toc dir" through the RunCLI/RunCLIIn seam, following
// cli_test.go's offline, spawn-free pattern: no subprocess, no language server, no t.Chdir.
// Fixture files live in a t.TempDir() reached via RunCLIIn's injected seam cwd, which is what
// proves toc honours that seam rather than reading the process working directory.

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

	"github.com/Knatte18/quarry/quarry"
)

// writeTOCFixture writes content to name inside dir and returns dir, so call sites can chain
// straight into a RunCLIIn call.
func writeTOCFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q, ...) failed: %v", path, err)
	}
}

// runTOCCLI runs RunCLIIn(cwd, args) and decodes the single-line JSON envelope it wrote, failing
// the test if the output is not exactly one line of valid JSON.
func runTOCCLI(t *testing.T, cwd string, args ...string) (int, map[string]any) {
	t.Helper()
	var out bytes.Buffer
	exitCode := RunCLIIn(cwd, &out, args)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("RunCLIIn(%v) output has %d lines; want exactly 1. output:\n%s", args, len(lines), out.String())
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &env); err != nil {
		t.Fatalf("RunCLIIn(%v) output is not valid JSON: %v; got: %q", args, err, lines[0])
	}
	return exitCode, env
}

const goFixture = `// Package fixture is a tiny Go fixture toc_test.go drives the CLI against.
package fixture

// Greet returns a greeting for name.
func Greet(name string) string {
	return "hello " + name
}
`

const typeAliasFixture = `package fixture

// ID names a fixture identifier.
type ID = string
`

const brokenFixture = "package fixture\n" +
	"\n" +
	"func Broken(\n" +
	"\n" +
	"func Recovered() {}\n"

const pythonFixtureUnderGoLang = `def greet(name):
    return "hello " + name
`

const rustFixture = "fn greet() -> &'static str {\n    \"hello\"\n}\n"

// TestRunCLI_Toc_BareShowsHelp verifies bare "quarry toc" prints help and exits 0, GroupRunE's
// bare-invocation contract.
func TestRunCLI_Toc_BareShowsHelp(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"toc"})

	if exitCode != 0 {
		t.Errorf("RunCLI(toc) = %d; want 0", exitCode)
	}
	if got := out.String(); !strings.Contains(got, "file") || !strings.Contains(got, "dir") {
		t.Errorf("RunCLI(toc) help output missing subcommand listing; got: %q", got)
	}
}

// TestRunCLI_Toc_UnknownSubcommand verifies "quarry toc badsub" emits the JSON error envelope
// naming the unknown subcommand, GroupRunE's other contract.
func TestRunCLI_Toc_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	exitCode, env := runTOCCLI(t, t.TempDir(), "toc", "badsub")
	if exitCode == 0 {
		t.Fatalf("RunCLI(toc badsub) = 0; want non-zero exit")
	}
	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("RunCLI(toc badsub) ok = true; want false")
	}
	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "badsub") {
		t.Errorf("RunCLI(toc badsub) error = %q; want it to name the unknown subcommand", errMsg)
	}
}

// TestRunCLI_TocFile_OnDirectory verifies "toc file" on a directory exits 1 with a message naming
// "toc dir" as the fix.
func TestRunCLI_TocFile_OnDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	exitCode, env := runTOCCLI(t, dir, "toc", "file", ".")
	if exitCode == 0 {
		t.Fatalf("RunCLIIn(toc file .) = 0; want non-zero exit for a directory argument")
	}
	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "toc dir") {
		t.Errorf("RunCLIIn(toc file .) error = %q; want it to name %q", errMsg, "toc dir")
	}
}

// TestRunCLI_TocDir_OnFile verifies "toc dir" on a file exits 1 with a message naming "toc file"
// as the fix.
func TestRunCLI_TocDir_OnFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "foo.go", goFixture)

	exitCode, env := runTOCCLI(t, dir, "toc", "dir", "foo.go")
	if exitCode == 0 {
		t.Fatalf("RunCLIIn(toc dir foo.go) = 0; want non-zero exit for a file argument")
	}
	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "toc file") {
		t.Errorf("RunCLIIn(toc dir foo.go) error = %q; want it to name %q", errMsg, "toc file")
	}
}

// TestRunCLI_Toc_NonexistentPath verifies a nonexistent path to either verb is an output.Err with
// exit 1.
func TestRunCLI_Toc_NonexistentPath(t *testing.T) {
	t.Parallel()

	for _, sub := range []string{"file", "dir"} {
		sub := sub
		t.Run(sub, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			exitCode, env := runTOCCLI(t, dir, "toc", sub, "does-not-exist")
			if exitCode == 0 {
				t.Fatalf("RunCLIIn(toc %s does-not-exist) = 0; want non-zero exit", sub)
			}
			if ok, _ := env["ok"].(bool); ok {
				t.Errorf("RunCLIIn(toc %s does-not-exist) ok = true; want false", sub)
			}
			if errMsg, _ := env["error"].(string); errMsg == "" {
				t.Errorf("RunCLIIn(toc %s does-not-exist) error field empty; want it set", sub)
			}
		})
	}
}

// TestRunCLI_TocFile_GoFixture verifies "toc file" on a real Go fixture returns ok:true, exit 0,
// and an envelope carrying "language", "package", "symbols", and a "header", with no "partial"
// key present, and each symbol carrying "start", "sigend", and "end".
func TestRunCLI_TocFile_GoFixture(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "foo.go", goFixture)

	exitCode, env := runTOCCLI(t, dir, "toc", "file", "foo.go")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc file foo.go) = %d; want 0. envelope: %v", exitCode, env)
	}
	if ok, _ := env["ok"].(bool); !ok {
		t.Fatalf("RunCLIIn(toc file foo.go) ok = false; want true. envelope: %v", env)
	}
	if env["language"] != "go" {
		t.Errorf("language = %v; want \"go\"", env["language"])
	}
	if env["package"] != "fixture" {
		t.Errorf("package = %v; want \"fixture\"", env["package"])
	}
	if _, ok := env["header"]; !ok {
		t.Errorf("header key missing; want it present for a file with a leading comment")
	}
	if _, ok := env["partial"]; ok {
		t.Errorf("partial key present = %v; want it omitted for a clean parse", env["partial"])
	}

	symbols, ok := env["symbols"].([]any)
	if !ok || len(symbols) == 0 {
		t.Fatalf("symbols = %v; want a non-empty array", env["symbols"])
	}
	sym, ok := symbols[0].(map[string]any)
	if !ok {
		t.Fatalf("symbols[0] = %v; want a JSON object", symbols[0])
	}
	for _, key := range []string{"start", "sigend", "end"} {
		if _, ok := sym[key]; !ok {
			t.Errorf("symbols[0] missing key %q; got: %v", key, sym)
		}
	}
}

// TestRunCLI_TocFile_TypeAliasHasNoSigEndKey verifies a fixture whose only symbol is a type alias
// has no "sigend" key at all on that symbol's entry, asserted against the decoded JSON since the
// omission is what a consumer sees.
func TestRunCLI_TocFile_TypeAliasHasNoSigEndKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "alias.go", typeAliasFixture)

	exitCode, env := runTOCCLI(t, dir, "toc", "file", "alias.go")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc file alias.go) = %d; want 0. envelope: %v", exitCode, env)
	}
	symbols, _ := env["symbols"].([]any)
	if len(symbols) != 1 {
		t.Fatalf("symbols = %v; want exactly one entry for the type-alias fixture", env["symbols"])
	}
	sym, _ := symbols[0].(map[string]any)
	if _, present := sym["sigend"]; present {
		t.Errorf("symbols[0] carries a \"sigend\" key = %v; want it omitted for a type alias", sym["sigend"])
	}
}

// TestRunCLI_TocFile_SyntaxErrorSetsPartialTrue verifies the same fixture shape with a syntax
// error reports "partial":true and still exits 0.
func TestRunCLI_TocFile_SyntaxErrorSetsPartialTrue(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "broken.go", brokenFixture)

	exitCode, env := runTOCCLI(t, dir, "toc", "file", "broken.go")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc file broken.go) = %d; want 0 even for a partial parse. envelope: %v", exitCode, env)
	}
	if partial, _ := env["partial"].(bool); !partial {
		t.Errorf("partial = %v; want true for a deliberately broken fixture", env["partial"])
	}
}

// TestRunCLI_TocFile_RustFixtureNamesUnsupportedLanguage verifies "toc file" on a .rs fixture
// returns the explicit classifyTOCError message rather than an empty result.
func TestRunCLI_TocFile_RustFixtureNamesUnsupportedLanguage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "lib.rs", rustFixture)

	exitCode, env := runTOCCLI(t, dir, "toc", "file", "lib.rs")
	if exitCode == 0 {
		t.Fatalf("RunCLIIn(toc file lib.rs) = 0; want non-zero exit for an unimplemented language")
	}
	errMsg, _ := env["error"].(string)
	wantStatus, wantMsg := classifyTOCError(fmt.Errorf("toc: lib.rs: language %q has no toc strategy: %w", "rust", quarry.ErrLanguageUnsupported))
	if wantStatus != statusError {
		t.Fatalf("classifyTOCError sanity check: status = %v; want statusError", wantStatus)
	}
	if errMsg != wantMsg {
		t.Errorf("RunCLIIn(toc file lib.rs) error = %q; want the classifyTOCError message %q", errMsg, wantMsg)
	}
}

// TestRunCLI_TocFile_RustFixtureBatchMatchesSingleArg verifies the same .rs argument inside a
// multi-argument batch carries the identical error message as the single-argument call, proving
// the batch driver routes through classifyTOCError instead of re-deriving the message.
func TestRunCLI_TocFile_RustFixtureBatchMatchesSingleArg(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "lib.rs", rustFixture)
	writeTOCFixture(t, dir, "foo.go", goFixture)

	_, singleEnv := runTOCCLI(t, dir, "toc", "file", "lib.rs")
	singleMsg, _ := singleEnv["error"].(string)
	if singleMsg == "" {
		t.Fatalf("single-argument toc file lib.rs produced no error message")
	}

	_, batchEnv := runTOCCLI(t, dir, "toc", "file", "lib.rs", "foo.go")
	results, _ := batchEnv["results"].([]any)
	entry := findBatchEntry(t, results, "lib.rs")
	batchMsg, _ := entry["error"].(string)
	if batchMsg != singleMsg {
		t.Errorf("batch entry error = %q; want it identical to the single-argument message %q", batchMsg, singleMsg)
	}
	if entry["status"] != string(statusError) {
		t.Errorf("batch entry status = %v; want %q", entry["status"], statusError)
	}
}

// TestClassifyTOCError_SentinelReachedThroughWrap verifies classifyTOCError's ErrLanguageUnsupported
// branch is reached through a wrap, since every real caller wraps the sentinel — a == comparison
// rather than errors.Is would pass this test's unwrapped equivalent but fail against the wrapped
// error every real caller actually produces.
func TestClassifyTOCError_SentinelReachedThroughWrap(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("toc: some/path.rs: language %q has no toc strategy: %w", "rust", quarry.ErrLanguageUnsupported)
	if !errors.Is(wrapped, quarry.ErrLanguageUnsupported) {
		t.Fatalf("errors.Is(wrapped, ErrLanguageUnsupported) = false; test fixture itself is wrong")
	}

	status, msg := classifyTOCError(wrapped)
	if status != statusError {
		t.Errorf("classifyTOCError(wrapped).status = %v; want statusError", status)
	}
	if !strings.Contains(msg, "not yet supported") {
		t.Errorf("classifyTOCError(wrapped).msg = %q; want it to name the unsupported-language situation", msg)
	}
	for _, lang := range quarry.TOCImplemented() {
		if !strings.Contains(msg, lang) {
			t.Errorf("classifyTOCError(wrapped).msg = %q; want it to name implemented language %q", msg, lang)
		}
	}

	otherErr := errors.New("some other unrelated failure")
	status, msg = classifyTOCError(otherErr)
	if status != statusError {
		t.Errorf("classifyTOCError(otherErr).status = %v; want statusError", status)
	}
	if msg != otherErr.Error() {
		t.Errorf("classifyTOCError(otherErr).msg = %q; want the error's own unmodified message %q", msg, otherErr.Error())
	}
}

// TestRunCLI_Toc_UnrecognisedLangNamesValidSet verifies --lang with an unrecognised value on both
// verbs errors, naming the valid set.
func TestRunCLI_Toc_UnrecognisedLangNamesValidSet(t *testing.T) {
	t.Parallel()

	for _, sub := range []string{"file", "dir"} {
		sub := sub
		t.Run(sub, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTOCFixture(t, dir, "foo.go", goFixture)
			arg := "foo.go"
			if sub == "dir" {
				arg = "."
			}
			exitCode, env := runTOCCLI(t, dir, "toc", sub, arg, "--lang", "bogus")
			if exitCode == 0 {
				t.Fatalf("RunCLIIn(toc %s %s --lang bogus) = 0; want non-zero exit", sub, arg)
			}
			errMsg, _ := env["error"].(string)
			for _, lang := range quarry.TOCLanguages() {
				if !strings.Contains(errMsg, lang) {
					t.Errorf("RunCLIIn(toc %s --lang bogus) error = %q; want it to name valid language %q", sub, errMsg, lang)
				}
			}
		})
	}
}

// TestRunCLI_TocFile_LangGoOnPyFixtureParsesAsGo verifies "--lang go" on a .py fixture parses with
// the Go grammar and does not error on the extension mismatch.
func TestRunCLI_TocFile_LangGoOnPyFixtureParsesAsGo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "script.py", pythonFixtureUnderGoLang)

	exitCode, env := runTOCCLI(t, dir, "toc", "file", "script.py", "--lang", "go")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc file script.py --lang go) = %d; want 0 — a mismatch is not an error. envelope: %v", exitCode, env)
	}
	if env["language"] != "go" {
		t.Errorf("language = %v; want \"go\" (the --lang override), not the .py extension's own language", env["language"])
	}
}

// TestRunCLI_TocDir_LangRustOnRsFilesListsPerFileErrors verifies "toc dir --lang rust" on a
// directory holding .rs files returns ok:true, exit 0, and a per-file "error" entry for each —
// not a top-level error, since the flag selects which files to list and an unimplemented language
// is a reported limitation rather than a failure of the listing.
func TestRunCLI_TocDir_LangRustOnRsFilesListsPerFileErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "lib.rs", rustFixture)

	exitCode, env := runTOCCLI(t, dir, "toc", "dir", ".", "--lang", "rust")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc dir . --lang rust) = %d; want 0. envelope: %v", exitCode, env)
	}
	if ok, _ := env["ok"].(bool); !ok {
		t.Fatalf("RunCLIIn(toc dir . --lang rust) ok = false; want true. envelope: %v", env)
	}
	if _, present := env["error"]; present {
		t.Errorf("top-level error key present = %v; want no top-level error for a per-file limitation", env["error"])
	}
	files, _ := env["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("files = %v; want exactly one entry for lib.rs", env["files"])
	}
	entry, _ := files[0].(map[string]any)
	if errMsg, _ := entry["error"].(string); errMsg == "" {
		t.Errorf("files[0].error is empty; want it set for the unimplemented rust strategy")
	}
}

// TestRunCLI_TocDir_NoSupportedFilesReturnsEmptyFiles verifies "toc dir" on a directory with no
// supported files returns "files":[] and exit 0.
func TestRunCLI_TocDir_NoSupportedFilesReturnsEmptyFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "README.md", "# nothing toc reads\n")

	exitCode, env := runTOCCLI(t, dir, "toc", "dir", ".")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc dir .) = %d; want 0 for an empty listing. envelope: %v", exitCode, env)
	}
	files, ok := env["files"].([]any)
	if !ok || len(files) != 0 {
		t.Errorf("files = %v; want an empty array", env["files"])
	}
}

// TestRunCLI_TocDir_PathCompositionUsesArgumentAsWritten verifies every entry's "path" is the
// directory argument as written joined with the filename, by passing a relative argument through
// RunCLIIn and checking the emitted prefix is the relative form, not the absolute one.
func TestRunCLI_TocDir_PathCompositionUsesArgumentAsWritten(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("os.Mkdir(%q) failed: %v", sub, err)
	}
	writeTOCFixture(t, sub, "foo.go", goFixture)

	exitCode, env := runTOCCLI(t, root, "toc", "dir", "sub")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc dir sub) = %d; want 0. envelope: %v", exitCode, env)
	}
	files, _ := env["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("files = %v; want exactly one entry", env["files"])
	}
	entry, _ := files[0].(map[string]any)
	wantPath := filepath.Join("sub", "foo.go")
	if entry["path"] != wantPath {
		t.Errorf("files[0].path = %v; want the relative form %q, not the absolutised path", entry["path"], wantPath)
	}
}

// TestRunCLI_TocDir_UnreadableFileStillListedRestUnaffected verifies an invalid-UTF-8 file inside
// a "toc dir" listing is still listed, with "error" set, no "header", no "partial", and the rest
// of the directory unaffected.
func TestRunCLI_TocDir_UnreadableFileStillListedRestUnaffected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "readable.go", goFixture)
	writeTOCFixture(t, dir, "bad.go", "package fixture\n\xff\n")

	exitCode, env := runTOCCLI(t, dir, "toc", "dir", ".")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc dir .) = %d; want 0 — a bad file never fails the listing. envelope: %v", exitCode, env)
	}
	files, _ := env["files"].([]any)
	var bad, readable map[string]any
	for _, raw := range files {
		f, _ := raw.(map[string]any)
		switch f["path"] {
		case "bad.go":
			bad = f
		case "readable.go":
			readable = f
		}
	}
	if bad == nil {
		t.Fatalf("files = %v; want an entry for bad.go", files)
	}
	if errMsg, _ := bad["error"].(string); errMsg == "" {
		t.Error("bad.go entry has no error; want it set for invalid UTF-8")
	}
	if _, present := bad["header"]; present {
		t.Errorf("bad.go entry carries a header %v; want it omitted", bad["header"])
	}
	if _, present := bad["partial"]; present {
		t.Errorf("bad.go entry carries a partial key %v; want it omitted", bad["partial"])
	}
	if readable == nil {
		t.Fatalf("files = %v; want readable.go still listed", files)
	}
	if _, present := readable["error"]; present {
		t.Errorf("readable.go entry carries an error %v; want none, unaffected by its bad sibling", readable["error"])
	}
}

// TestRunCLI_TocBatch_KeyIsPathNotSymbol verifies batch mode with two or more arguments on both
// verbs keys each entry by "path", not "symbol".
func TestRunCLI_TocBatch_KeyIsPathNotSymbol(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "a.go", goFixture)
	writeTOCFixture(t, dir, "b.go", goFixture)

	for _, sub := range []string{"file", "dir"} {
		sub := sub
		t.Run(sub, func(t *testing.T) {
			t.Parallel()
			args := []string{"toc", sub, "a.go", "b.go"}
			if sub == "dir" {
				args = []string{"toc", sub, ".", "."}
			}
			_, env := runTOCCLI(t, dir, args...)
			results, ok := env["results"].([]any)
			if !ok || len(results) == 0 {
				t.Fatalf("results = %v; want a non-empty array", env["results"])
			}
			for _, raw := range results {
				entry, _ := raw.(map[string]any)
				if _, present := entry["path"]; !present {
					t.Errorf("batch entry %v missing \"path\" key", entry)
				}
				if _, present := entry["symbol"]; present {
					t.Errorf("batch entry %v carries a \"symbol\" key; want \"path\" only", entry)
				}
			}
		})
	}
}

// TestRunCLI_TocDirBatch_EveryFileEntryCarriesItsOwnPath verifies batch mode on "toc dir" with two
// directory arguments composes every element of each entry's "files" array with its own "path",
// derived from that entry's own argument as written. This is the assertion that fails if the
// batch closure skips tocDirEntries.
func TestRunCLI_TocDirBatch_EveryFileEntryCarriesItsOwnPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, name := range []string{"one", "two"} {
		sub := filepath.Join(root, name)
		if err := os.Mkdir(sub, 0o755); err != nil {
			t.Fatalf("os.Mkdir(%q) failed: %v", sub, err)
		}
		writeTOCFixture(t, sub, "foo.go", goFixture)
	}

	_, env := runTOCCLI(t, root, "toc", "dir", "one", "two")
	results, _ := env["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("results = %v; want exactly 2 entries", env["results"])
	}
	for _, raw := range results {
		entry, _ := raw.(map[string]any)
		argPath, _ := entry["path"].(string)
		files, _ := entry["files"].([]any)
		if len(files) != 1 {
			t.Fatalf("entry %v files = %v; want exactly one file", argPath, entry["files"])
		}
		file, _ := files[0].(map[string]any)
		wantPath := filepath.Join(argPath, "foo.go")
		if file["path"] != wantPath {
			t.Errorf("entry %q file path = %v; want %q", argPath, file["path"], wantPath)
		}
	}
}

// TestRunCLI_TocBatch_WorstStatusSetsExitCode verifies a batch mixing a found, a not-found, and an
// error argument sets the exit code to the worst rank present.
func TestRunCLI_TocBatch_WorstStatusSetsExitCode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "ok.go", goFixture)
	writeTOCFixture(t, dir, "bad.rs", rustFixture)

	exitCode, env := runTOCCLI(t, dir, "toc", "file", "ok.go", "missing.go", "bad.rs")
	if exitCode != statusRank[statusError] {
		t.Errorf("exit code = %d; want the worst rank %d (statusError)", exitCode, statusRank[statusError])
	}
	results, _ := env["results"].([]any)
	if len(results) != 3 {
		t.Fatalf("results = %v; want exactly 3 entries", env["results"])
	}
}

// TestRunCLI_TocBatch_PartialTrueStaysExitZero verifies a batch containing one "partial":true file
// still exits 0 — partial degrading the status would poison the exit code of any batch containing
// a single mid-edit file.
func TestRunCLI_TocBatch_PartialTrueStaysExitZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTOCFixture(t, dir, "broken.go", brokenFixture)
	writeTOCFixture(t, dir, "clean.go", goFixture)

	exitCode, env := runTOCCLI(t, dir, "toc", "file", "broken.go", "clean.go")
	if exitCode != 0 {
		t.Fatalf("RunCLIIn(toc file broken.go clean.go) = %d; want 0. envelope: %v", exitCode, env)
	}
	results, _ := env["results"].([]any)
	entry := findBatchEntry(t, results, "broken.go")
	if partial, _ := entry["partial"].(bool); !partial {
		t.Errorf("broken.go batch entry partial = %v; want true", entry["partial"])
	}
	if entry["status"] != string(statusFound) {
		t.Errorf("broken.go batch entry status = %v; want %q — partial must not degrade status", entry["status"], statusFound)
	}
}

// findBatchEntry returns the entry in results whose "path" equals path, failing the test if none
// matches.
func findBatchEntry(t *testing.T, results []any, path string) map[string]any {
	t.Helper()
	for _, raw := range results {
		entry, _ := raw.(map[string]any)
		if entry["path"] == path {
			return entry
		}
	}
	t.Fatalf("no batch entry found with path %q in %v", path, results)
	return nil
}
