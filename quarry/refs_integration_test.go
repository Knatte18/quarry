//go:build lsp

// refs_integration_test.go exercises References against a real, held-open
// gopls subprocess — the one test in this package that actually launches a
// language server. The tag names its real precondition, a real
// language-server binary on $PATH, so this file is excluded from the plain
// `go test` verify and run separately with `-tags lsp` on a machine with
// gopls installed. Only the gopls-spawning subtest is guarded on
// exec.LookPath("gopls") (via t.Skip); the ErrServerNotFound subtest never
// launches gopls and always runs, even on a machine without it. This test
// spawns no git and needs no git-environment isolation — it spawns only
// gopls.

package quarry

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// funcDeclPattern matches top-level function declarations and captures the function name offset.
var funcDeclPattern = regexp.MustCompile(`^func (?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\(`)

// findFuncPosition returns the Position of a top-level function in a file.
func findFuncPosition(t *testing.T, file, funcName string) Position {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("findFuncPosition: read %s: %v", file, err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		m := funcDeclPattern.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		name := line[m[2]:m[3]]
		if name != funcName {
			continue
		}
		return Position{File: file, Line: i + 1, Character: m[2] + 1}
	}
	t.Fatalf("findFuncPosition: no top-level declaration of %q found in %s", funcName, file)
	return Position{}
}

// repoRoot returns this worktree's module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("repoRoot: could not determine scoutengine source directory location")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func TestReferences_Integration(t *testing.T) {
	t.Run("live gopls references for a known high-fan-in symbol", func(t *testing.T) {
		if _, err := exec.LookPath("gopls"); err != nil {
			t.Skip(builtins()["go"].InstallHint)
		}

		root := repoRoot(t)
		detectFile := filepath.Join(root, "quarry", "detect.go")
		pos := findFuncPosition(t, detectFile, "DetectLanguage")

		// Since the engine-supervised-flip batch, Go's registry entry
		// dispatches through ensureServer -> ensureSupervised, which spawns a
		// quarry-owned daemon that teardownConnection's connKindSupervised
		// branch deliberately never kills. StateDir is required (Options no
		// longer derives it), so an isolated t.TempDir() plus a
		// state-file-driven reap is what keeps this test from leaking the
		// daemon.
		stateDir := t.TempDir()
		statePath := DaemonStateFile(stateDir, "go")
		t.Cleanup(func() { killRecordedDaemon(t, statePath) })

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		refs, err := References(ctx, Options{
			Registry:  builtins(),
			TargetDir: root,
			StateDir:  stateDir,
			Lang:      "go",
			Query:     Query{Pos: &pos},
			Timeout:   30 * time.Second,
		})
		if err != nil {
			t.Fatalf("References(DetectLanguage) returned unexpected error: %v", err)
		}
		if len(refs) == 0 {
			t.Fatal("References(DetectLanguage) returned zero references; want the declaration site plus its call sites")
		}

		foundDeclSite := false
		for _, ref := range refs {
			if filepath.Clean(ref.File) == filepath.Clean(detectFile) && ref.Line == pos.Line {
				foundDeclSite = true
				break
			}
		}
		if !foundDeclSite {
			t.Errorf("References(DetectLanguage) = %+v; want it to include the declaration site %s:%d", refs, detectFile, pos.Line)
		}
	})

	t.Run("non-existent server binary yields ErrServerNotFound", func(t *testing.T) {
		root := repoRoot(t)
		reg := Registry{
			"go": {
				Markers:     []string{"go.mod"},
				Match:       "any",
				Command:     []string{"quarry-nonexistent-binary-xyz"},
				InstallHint: "this binary is intentionally fake for the test",
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := References(ctx, Options{
			Registry:  reg,
			TargetDir: root,
			Lang:      "go",
			Query:     Query{Symbol: "Resolve"},
			Timeout:   5 * time.Second,
		})
		if !errors.Is(err, ErrServerNotFoundSentinel) {
			t.Errorf("References() with a non-existent server binary err = %v; want errors.Is(err, ErrServerNotFoundSentinel)", err)
		}
	})
}

// writeAmbiguousModule writes a minimal, self-contained Go module to dir: a
// go.mod plus one .go file declaring the same method name ("Open") on two
// distinct types. This is what makes a name-only, per-file
// textDocument/documentSymbol lookup for "Open" within that file resolve to
// more than one candidate — the case that distinguishes InFile's exhaustive
// per-file search from a single hit, exercised by
// TestReferences_InFile_Integration's "same-name-in-two-types ambiguity"
// subtest.
func writeAmbiguousModule(t *testing.T, dir string) {
	t.Helper()
	goMod := "module ambiguousmodule\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) failed: %v", err)
	}
	src := `package main

type FileHandle struct{}

func (h FileHandle) Open() error { return nil }

type SocketHandle struct{}

func (h SocketHandle) Open() error { return nil }

func main() {}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("os.WriteFile(main.go) failed: %v", err)
	}
}

// TestReferences_InFile_Integration proves the Query.InFile resolve path — documentSymbol ->
// position -> textDocument/references — end to end against a real gopls, the InFile analogue of
// TestReferences_Integration's Query.Pos coverage above.
// Both subcases route through ensureServer's now- live supervised dispatch (builtins()'s Go entry
// has HasNativeDaemon: true), which spawns a quarry-owned daemon that teardownConnection's
// connKindSupervised branch deliberately never kills — each subcase gives its own isolated
// t.TempDir() as StateDir and reaps the spawned daemon in t.Cleanup, exactly like
// TestEnsureServer_Integration_ SupervisedDispatch (ensureserver_integration_test.go) and
// supervised_integration_test.go already do.
func TestReferences_InFile_Integration(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip(builtins()["go"].InstallHint)
	}

	t.Run("single-match resolve", func(t *testing.T) {
		root := repoRoot(t)
		detectFile := filepath.Join(root, "quarry", "detect.go")
		pos := findFuncPosition(t, detectFile, "DetectLanguage")

		// TargetDir stays the real repo root (correct indexing), but StateDir
		// is an isolated temp dir so the supervised daemon this call spawns
		// records its state there, never anywhere under the real repo.
		stateDir := t.TempDir()
		statePath := DaemonStateFile(stateDir, "go")
		t.Cleanup(func() { killRecordedDaemon(t, statePath) })

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		refs, err := References(ctx, Options{
			Registry:  builtins(),
			TargetDir: root,
			StateDir:  stateDir,
			Lang:      "go",
			Query:     Query{InFile: &InFileQuery{File: detectFile, Name: "DetectLanguage"}},
			Timeout:   30 * time.Second,
		})
		if err != nil {
			t.Fatalf("References(InFile DetectLanguage) returned unexpected error: %v", err)
		}
		if len(refs) == 0 {
			t.Fatal("References(InFile DetectLanguage) returned zero references; want the declaration site plus its call sites")
		}

		foundDeclSite := false
		for _, ref := range refs {
			if filepath.Clean(ref.File) == filepath.Clean(detectFile) && ref.Line == pos.Line {
				foundDeclSite = true
				break
			}
		}
		if !foundDeclSite {
			t.Errorf("References(InFile DetectLanguage) = %+v; want it to include the declaration site %s:%d", refs, detectFile, pos.Line)
		}
	})

	t.Run("same-name-in-two-types ambiguity", func(t *testing.T) {
		modRoot := t.TempDir()
		writeAmbiguousModule(t, modRoot)

		stateDir := t.TempDir()
		statePath := DaemonStateFile(stateDir, "go")
		t.Cleanup(func() { killRecordedDaemon(t, statePath) })

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		_, err := References(ctx, Options{
			Registry:  builtins(),
			TargetDir: modRoot,
			StateDir:  stateDir,
			Lang:      "go",
			Query:     Query{InFile: &InFileQuery{File: filepath.Join(modRoot, "main.go"), Name: "Open"}},
			Timeout:   30 * time.Second,
		})
		if !errors.Is(err, ErrAmbiguousSymbolSentinel) {
			t.Errorf("References(InFile Open, two types) err = %v; want errors.Is(err, ErrAmbiguousSymbolSentinel)", err)
		}
	})
}
