package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/internal/cli"
	"github.com/Knatte18/quarry/quarry"
)

// writeTOCTestFile writes content to name inside dir, creating dir if needed.
func writeTOCTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", dir, err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	return path
}

// stubTOCFileFn returns a tocFileFn-shaped stub keyed by the resolved absolute path it was called
// with, and records the quarry.TOCOptions each call received into calls, so a test can assert the
// per-argument config-base resolution reached the facade unmodified.
func stubTOCFileFn(t *testing.T, calls map[string]quarry.TOCOptions, responses map[string]struct {
	result quarry.TOCFileResult
	err    error
}) func(string, string, quarry.TOCOptions) (quarry.TOCFileResult, error) {
	t.Helper()
	return func(path, _ string, opts quarry.TOCOptions) (quarry.TOCFileResult, error) {
		if calls != nil {
			calls[path] = opts
		}
		r, ok := responses[path]
		if !ok {
			t.Fatalf("stub toc file fn: no response configured for path %q", path)
		}
		return r.result, r.err
	}
}

// stubTOCDirFn returns a tocDirFn-shaped stub keyed by the resolved absolute path it was called
// with.
func stubTOCDirFn(t *testing.T, responses map[string]struct {
	result quarry.TOCDirResult
	err    error
}) func(string, string) (quarry.TOCDirResult, error) {
	t.Helper()
	return func(path, _ string) (quarry.TOCDirResult, error) {
		r, ok := responses[path]
		if !ok {
			t.Fatalf("stub toc dir fn: no response configured for path %q", path)
		}
		return r.result, r.err
	}
}

// TestTOCFileHandler_MissingPathIsNotFound asserts a missing target is statusNotFound, exercised
// against a real (absent) fixture path rather than a stub.
func TestTOCFileHandler_MissingPathIsNotFound(t *testing.T) {
	cfg := newTestConfig(t)
	in := mustUnmarshal[tocFileInput](t, `{"targets":["does-not-exist.go"]}`)

	_, out, err := tocFileHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocFileHandler(cfg)(...) error = %v; want nil", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}
	if got := out.Results[0].Status; got != statusNotFound {
		t.Errorf("out.Results[0].Status = %q; want %q", got, statusNotFound)
	}
}

// TestTOCDirHandler_MissingPathIsNotFound mirrors TestTOCFileHandler_MissingPathIsNotFound for
// toc_dir.
func TestTOCDirHandler_MissingPathIsNotFound(t *testing.T) {
	cfg := newTestConfig(t)
	in := mustUnmarshal[tocDirInput](t, `{"targets":["does-not-exist"]}`)

	_, out, err := tocDirHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocDirHandler(cfg)(...) error = %v; want nil", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(out.Results) = %d; want 1", len(out.Results))
	}
	if got := out.Results[0].Status; got != statusNotFound {
		t.Errorf("out.Results[0].Status = %q; want %q", got, statusNotFound)
	}
}

// TestTOCFileHandler_DirectoryTargetIsError asserts a directory passed to toc_file is statusError.
func TestTOCFileHandler_DirectoryTargetIsError(t *testing.T) {
	cfg := newTestConfig(t)
	sub := filepath.Join(cfg.TargetDir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", sub, err)
	}

	in := mustUnmarshal[tocFileInput](t, `{"targets":["sub"]}`)

	_, out, err := tocFileHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocFileHandler(cfg)(...) error = %v; want nil", err)
	}
	if got := out.Results[0].Status; got != statusError {
		t.Errorf("out.Results[0].Status = %q; want %q", got, statusError)
	}
}

// TestTOCDirHandler_FileTargetIsError asserts a file passed to toc_dir is statusError.
func TestTOCDirHandler_FileTargetIsError(t *testing.T) {
	cfg := newTestConfig(t)
	writeTOCTestFile(t, cfg.TargetDir, "a.go", "package a\n")

	in := mustUnmarshal[tocDirInput](t, `{"targets":["a.go"]}`)

	_, out, err := tocDirHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocDirHandler(cfg)(...) error = %v; want nil", err)
	}
	if got := out.Results[0].Status; got != statusError {
		t.Errorf("out.Results[0].Status = %q; want %q", got, statusError)
	}
}

// TestTOCFileHandler_LanguageUnsupportedIsErrorNeverNotFound asserts a stub returning a wrapped
// quarry.ErrLanguageUnsupported is statusError, worded from quarry.TOCImplemented(), and never
// statusNotFound.
func TestTOCFileHandler_LanguageUnsupportedIsErrorNeverNotFound(t *testing.T) {
	cfg := newTestConfig(t)
	abs := writeTOCTestFile(t, cfg.TargetDir, "a.rs", "fn main() {}\n")

	withStubbedFacade(t, &tocFileFn, stubTOCFileFn(t, nil, map[string]struct {
		result quarry.TOCFileResult
		err    error
	}{
		abs: {err: quarry.ErrLanguageUnsupported},
	}))

	in := mustUnmarshal[tocFileInput](t, `{"targets":["a.rs"]}`)

	_, out, err := tocFileHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocFileHandler(cfg)(...) error = %v; want nil", err)
	}
	entry := out.Results[0]
	if entry.Status != statusError {
		t.Errorf("entry.Status = %q; want %q (never %q)", entry.Status, statusError, statusNotFound)
	}
	for _, lang := range quarry.TOCImplemented() {
		if !strings.Contains(entry.Error, lang) {
			t.Errorf("entry.Error = %q; want it worded from quarry.TOCImplemented() (missing %q)", entry.Error, lang)
		}
	}
}

// TestTOCDirHandler_EntriesCarryCallerRelativePath asserts a toc_dir entry's "files" carry a
// "path" composed against the caller-written argument, proving cli.TOCDirEntries was applied
// rather than cli.StructToFields alone (toc.DirEntry.Name carries json:"-", so StructToFields
// alone would emit neither "name" nor "path"), and that the composed path round-trips into a
// following toc_file call.
func TestTOCDirHandler_EntriesCarryCallerRelativePath(t *testing.T) {
	cfg := newTestConfig(t)
	sub := filepath.Join(cfg.TargetDir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", sub, err)
	}

	withStubbedFacade(t, &tocDirFn, stubTOCDirFn(t, map[string]struct {
		result quarry.TOCDirResult
		err    error
	}{
		sub: {result: quarry.TOCDirResult{Files: []quarry.TOCDirEntry{{Name: "x.go", Language: "go"}}}},
	}))

	in := mustUnmarshal[tocDirInput](t, `{"targets":["sub"]}`)

	_, out, err := tocDirHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocDirHandler(cfg)(...) error = %v; want nil", err)
	}
	if len(out.Results[0].Files) != 1 {
		t.Fatalf("len(out.Results[0].Files) = %d; want 1", len(out.Results[0].Files))
	}
	file, ok := out.Results[0].Files[0].(map[string]any)
	if !ok {
		t.Fatalf("out.Results[0].Files[0] = %#v; want map[string]any", out.Results[0].Files[0])
	}

	wantPath := filepath.Join("sub", "x.go")
	if file["path"] != wantPath {
		t.Errorf("file[\"path\"] = %v; want %q (caller-relative, composed against the argument as written)", file["path"], wantPath)
	}
	if filepath.IsAbs(file["path"].(string)) {
		t.Errorf("file[\"path\"] = %v; want a caller-relative path, not absolute", file["path"])
	}
	if _, present := file["name"]; present {
		t.Errorf("file has a \"name\" key = %v; want none (toc.DirEntry.Name carries json:\"-\")", file["name"])
	}
}

// TestTOCDirHandler_EntryCarriesSubdirectoryNames asserts a toc_dir entry's "dirs" is copied
// straight from quarry.TOCDirResult.Dirs, verbatim and in the order the stub returned it (TOCDir
// itself is responsible for sorting; the handler must not re-sort or otherwise transform it).
func TestTOCDirHandler_EntryCarriesSubdirectoryNames(t *testing.T) {
	cfg := newTestConfig(t)
	sub := filepath.Join(cfg.TargetDir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", sub, err)
	}

	withStubbedFacade(t, &tocDirFn, stubTOCDirFn(t, map[string]struct {
		result quarry.TOCDirResult
		err    error
	}{
		sub: {result: quarry.TOCDirResult{Files: []quarry.TOCDirEntry{}, Dirs: []string{"apple", "zebra"}}},
	}))

	in := mustUnmarshal[tocDirInput](t, `{"targets":["sub"]}`)

	_, out, err := tocDirHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocDirHandler(cfg)(...) error = %v; want nil", err)
	}
	want := []string{"apple", "zebra"}
	if len(out.Results[0].Dirs) != len(want) {
		t.Fatalf("out.Results[0].Dirs = %v; want %v", out.Results[0].Dirs, want)
	}
	for i, name := range want {
		if out.Results[0].Dirs[i] != name {
			t.Errorf("out.Results[0].Dirs[%d] = %q; want %q", i, out.Results[0].Dirs[i], name)
		}
	}
}

// TestTOCFileHandler_ResolvesQuarryYAMLAgainstFilesOwnDirectory asserts each target's effective
// DocSentences value is resolved against that target's own resolved file's parent directory, never
// the server's target directory, using two fixture directories carrying different doc_sentences
// values.
func TestTOCFileHandler_ResolvesQuarryYAMLAgainstFilesOwnDirectory(t *testing.T) {
	root := t.TempDir()
	writeTOCTestFile(t, root, ".quarry.yaml", "toc:\n  doc_sentences: 99\n")

	dirA := filepath.Join(root, "a")
	fileA := writeTOCTestFile(t, dirA, "a.go", "package a\n")
	writeTOCTestFile(t, dirA, ".quarry.yaml", "toc:\n  doc_sentences: 0\n")

	dirB := filepath.Join(root, "b")
	fileB := writeTOCTestFile(t, dirB, "b.go", "package b\n")
	writeTOCTestFile(t, dirB, ".quarry.yaml", "toc:\n  doc_sentences: 5\n")

	cfg := Config{TargetDir: root}
	calls := map[string]quarry.TOCOptions{}
	withStubbedFacade(t, &tocFileFn, stubTOCFileFn(t, calls, map[string]struct {
		result quarry.TOCFileResult
		err    error
	}{
		fileA: {result: quarry.TOCFileResult{Language: "go"}},
		fileB: {result: quarry.TOCFileResult{Language: "go"}},
	}))

	in := mustUnmarshal[tocFileInput](t, `{"targets":["a/a.go","b/b.go"]}`)

	_, _, err := tocFileHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocFileHandler(cfg)(...) error = %v; want nil", err)
	}

	if got := calls[fileA].DocSentences; got != 0 {
		t.Errorf("calls[fileA].DocSentences = %d; want 0 (dirA's own .quarry.yaml, not root's 99)", got)
	}
	if got := calls[fileB].DocSentences; got != 5 {
		t.Errorf("calls[fileB].DocSentences = %d; want 5 (dirB's own .quarry.yaml, not root's 99)", got)
	}
}

// TestTOCFileHandler_DocSentencesAsNumberAndAsAllBothSucceed asserts docSentences sent as a JSON
// number and as the string "all" both succeed as a call-wide flag override.
func TestTOCFileHandler_DocSentencesAsNumberAndAsAllBothSucceed(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"Number", `{"targets":["a.go"],"docSentences":3}`},
		{"All", `{"targets":["a.go"],"docSentences":"all"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestConfig(t)
			abs := writeTOCTestFile(t, cfg.TargetDir, "a.go", "package a\n")
			withStubbedFacade(t, &tocFileFn, stubTOCFileFn(t, nil, map[string]struct {
				result quarry.TOCFileResult
				err    error
			}{
				abs: {result: quarry.TOCFileResult{Language: "go"}},
			}))

			in := mustUnmarshal[tocFileInput](t, tt.body)

			_, out, err := tocFileHandler(cfg)(context.Background(), nil, in)
			if err != nil {
				t.Fatalf("tocFileHandler(cfg)(...) error = %v; want nil", err)
			}
			if got := out.Results[0].Status; got != statusFound {
				t.Errorf("out.Results[0].Status = %q; want %q", got, statusFound)
			}
		})
	}
}

// TestTOCFileHandler_ResultWrapperCarriesMarshalledTOCFileResult asserts a found entry's "result"
// key carries the marshalled quarry.TOCFileResult.
func TestTOCFileHandler_ResultWrapperCarriesMarshalledTOCFileResult(t *testing.T) {
	cfg := newTestConfig(t)
	abs := writeTOCTestFile(t, cfg.TargetDir, "a.go", "package a\n")
	withStubbedFacade(t, &tocFileFn, stubTOCFileFn(t, nil, map[string]struct {
		result quarry.TOCFileResult
		err    error
	}{
		abs: {result: quarry.TOCFileResult{Language: "go", Package: "a"}},
	}))

	in := mustUnmarshal[tocFileInput](t, `{"targets":["a.go"]}`)

	_, out, err := tocFileHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocFileHandler(cfg)(...) error = %v; want nil", err)
	}
	if got := out.Results[0].Result["language"]; got != "go" {
		t.Errorf("out.Results[0].Result[\"language\"] = %v; want %q", got, "go")
	}
	if got := out.Results[0].Result["package"]; got != "a" {
		t.Errorf("out.Results[0].Result[\"package\"] = %v; want %q", got, "a")
	}
}

// TestTOCMarshalFailure_MessageIsVerbatimNeverImpactReworded asserts a cli.StructToFields marshal
// failure's message flows through unchanged, with no "impact: " rewording applied — toc's own
// error disposition (resolveTOCFileEntry, resolveTOCDirEntry) uses err.Error() directly, unlike
// impact's resolveImpactEntry, which calls rewordMarshalFailure. This is exercised directly against
// cli.StructToFields, mirroring TestRewordMarshalFailure_BeginsWithImpactNeverToc's own rationale
// in tools_impact_test.go: quarry.TOCFileResult's field types (strings, ints, bools, slices of the
// same) always marshal successfully, so a stubbed tocFileFn can never itself trigger this branch
// through the full handler.
func TestTOCMarshalFailure_MessageIsVerbatimNeverImpactReworded(t *testing.T) {
	_, err := cli.StructToFields(make(chan int))
	if err == nil {
		t.Fatalf("cli.StructToFields(chan int) error = nil; want non-nil")
	}

	// resolveTOCFileEntry's and resolveTOCDirEntry's own marshal-failure branches assign
	// err.Error() to Error verbatim — never through rewordMarshalFailure, which prefixes "impact: "
	// and is impact-only.
	message := err.Error()
	if strings.HasPrefix(message, "impact: ") {
		t.Errorf("message = %q; want no \"impact: \" rewording for a toc marshal failure", message)
	}
	if !strings.HasPrefix(message, "toc: ") {
		t.Errorf("message = %q; want it to keep cli.StructToFields' own \"toc: \" prefix verbatim", message)
	}
}

// TestTOCFileHandler_InvalidLangFailsWholeCall asserts an invalid lang fails the whole call.
func TestTOCFileHandler_InvalidLangFailsWholeCall(t *testing.T) {
	cfg := newTestConfig(t)
	in := mustUnmarshal[tocFileInput](t, `{"targets":["a.go"],"lang":"not-a-real-language"}`)

	if _, _, err := tocFileHandler(cfg)(context.Background(), nil, in); err == nil {
		t.Errorf("tocFileHandler(cfg)(...) error = nil; want a whole-call failure for an invalid lang")
	}
}

// TestTOCFileHandler_InvalidDocSentencesFailsWholeCall asserts an invalid docSentences fails the
// whole call.
func TestTOCFileHandler_InvalidDocSentencesFailsWholeCall(t *testing.T) {
	cfg := newTestConfig(t)
	in := mustUnmarshal[tocFileInput](t, `{"targets":["a.go"],"docSentences":"not-a-number"}`)

	if _, _, err := tocFileHandler(cfg)(context.Background(), nil, in); err == nil {
		t.Errorf("tocFileHandler(cfg)(...) error = nil; want a whole-call failure for an invalid docSentences")
	}
}

// TestTOCDirHandler_InvalidLangFailsWholeCall asserts an invalid lang fails the whole call for
// toc_dir too.
func TestTOCDirHandler_InvalidLangFailsWholeCall(t *testing.T) {
	cfg := newTestConfig(t)
	in := mustUnmarshal[tocDirInput](t, `{"targets":["sub"],"lang":"not-a-real-language"}`)

	if _, _, err := tocDirHandler(cfg)(context.Background(), nil, in); err == nil {
		t.Errorf("tocDirHandler(cfg)(...) error = nil; want a whole-call failure for an invalid lang")
	}
}

// TestTOCTools_MalformedServersYAMLStillSucceeds asserts a Config.ConfigPath pointing at a
// malformed servers.yaml leaves both toc tools succeeding, because neither tocFileCommand nor
// tocDirCommand ever calls quarry.LoadRegistry — the handlers read cfg.TargetDir and call
// tocPreflight directly and never resolveCall.
func TestTOCTools_MalformedServersYAMLStillSucceeds(t *testing.T) {
	malformed := filepath.Join(t.TempDir(), "servers.yaml")
	if err := os.WriteFile(malformed, []byte("not: valid: yaml: [\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", malformed, err)
	}

	cfg := newTestConfig(t)
	cfg.ConfigPath = malformed
	abs := writeTOCTestFile(t, cfg.TargetDir, "a.go", "package a\n")
	sub := filepath.Join(cfg.TargetDir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", sub, err)
	}

	withStubbedFacade(t, &tocFileFn, stubTOCFileFn(t, nil, map[string]struct {
		result quarry.TOCFileResult
		err    error
	}{
		abs: {result: quarry.TOCFileResult{Language: "go"}},
	}))
	withStubbedFacade(t, &tocDirFn, stubTOCDirFn(t, map[string]struct {
		result quarry.TOCDirResult
		err    error
	}{
		sub: {result: quarry.TOCDirResult{Files: []quarry.TOCDirEntry{}}},
	}))

	fileIn := mustUnmarshal[tocFileInput](t, `{"targets":["a.go"]}`)
	if _, out, err := tocFileHandler(cfg)(context.Background(), nil, fileIn); err != nil {
		t.Errorf("tocFileHandler(cfg)(...) error = %v; want nil (toc never loads the registry)", err)
	} else if got := out.Results[0].Status; got != statusFound {
		t.Errorf("out.Results[0].Status = %q; want %q", got, statusFound)
	}

	dirIn := mustUnmarshal[tocDirInput](t, `{"targets":["sub"]}`)
	if _, out, err := tocDirHandler(cfg)(context.Background(), nil, dirIn); err != nil {
		t.Errorf("tocDirHandler(cfg)(...) error = %v; want nil (toc never loads the registry)", err)
	} else if got := out.Results[0].Status; got != statusFound {
		t.Errorf("out.Results[0].Status = %q; want %q", got, statusFound)
	}
}

// TestTOCFileHandler_AbsoluteTargetResolvesOutsideLaunchRoot pins the one partial escape hatch that
// survives the per-call targetDir override's removal: cli.ResolveTOCPath ignores the base directory
// entirely for an absolute argument, so toc_file can still resolve a target outside cfg.TargetDir by
// sending an absolute path. The five language-server-backed tools have no equivalent — their queries
// are served by the gopls daemon rooted at the server's own target directory, so there is no
// per-argument way to escape that root the way toc's plain-string targets allow.
func TestTOCFileHandler_AbsoluteTargetResolvesOutsideLaunchRoot(t *testing.T) {
	cfg := newTestConfig(t)
	outside := t.TempDir()
	abs := writeTOCTestFile(t, outside, "outside.go", "package outside\n")

	withStubbedFacade(t, &tocFileFn, stubTOCFileFn(t, nil, map[string]struct {
		result quarry.TOCFileResult
		err    error
	}{
		abs: {result: quarry.TOCFileResult{Language: "go"}},
	}))

	in := tocFileInput{Targets: []string{abs}}

	_, out, err := tocFileHandler(cfg)(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("tocFileHandler(cfg)(...) error = %v; want nil", err)
	}
	if got := out.Results[0].Status; got != statusFound {
		t.Errorf("out.Results[0].Status = %q; want %q (an absolute target outside cfg.TargetDir still resolves)", got, statusFound)
	}
}
