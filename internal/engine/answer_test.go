// answer_test.go pins the emitted JSON shape (the closed key set's omitempty behaviour), the
// Depth and Symbols knobs, the failure entries, the extensionless-file header rule, and the
// gitignore-freshness guarantee — that nothing is cached on Repo, so an edit to a .gitignore
// between two TOC calls on the same Repo is reflected in the second.

package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// marshalToMap marshals v to JSON and unmarshals it back into a map, so a test can assert on key
// presence directly rather than pattern-matching the raw bytes.
func marshalToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal(%+v) failed: %v", v, err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal(%s) failed: %v", b, err)
	}
	return m
}

// TestAnswerJSON_DirsAbsentWhenEmpty asserts a directory with no subdirectory omits the "dirs" key
// entirely, rather than emitting an empty array.
func TestAnswerJSON_DirsAbsentWhenEmpty(t *testing.T) {
	r := openScratchRepo(t, "answer-json-dirs-absent", map[string]string{"foo.go": "package p\n"})
	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	m := marshalToMap(t, got)
	if _, ok := m["dirs"]; ok {
		t.Errorf("marshalled answer = %v; \"dirs\" key present, want absent for no subdirectories", m)
	}
}

// TestAnswerJSON_TestAndGeneratedAbsentWhenFalse asserts a plain, non-test, non-generated file
// entry omits both the "test" and "generated" keys rather than emitting them false.
func TestAnswerJSON_TestAndGeneratedAbsentWhenFalse(t *testing.T) {
	r := openScratchRepo(t, "answer-json-test-generated-absent", map[string]string{"foo.go": "package p\n\nfunc F() {}\n"})
	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	m := marshalToMap(t, got.Files[0])
	if _, ok := m["test"]; ok {
		t.Errorf("file entry = %v; \"test\" key present, want absent when false", m)
	}
	if _, ok := m["generated"]; ok {
		t.Errorf("file entry = %v; \"generated\" key present, want absent when false", m)
	}
}

// TestAnswerJSON_ErrorAndLossyAbsentOnHappyPath asserts a well-formed file entry omits both
// "error" and "lossy".
func TestAnswerJSON_ErrorAndLossyAbsentOnHappyPath(t *testing.T) {
	r := openScratchRepo(t, "answer-json-error-lossy-absent", map[string]string{"foo.go": "package p\n\nfunc F() {}\n"})
	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	m := marshalToMap(t, got.Files[0])
	if _, ok := m["error"]; ok {
		t.Errorf("file entry = %v; \"error\" key present, want absent on the happy path", m)
	}
	if _, ok := m["lossy"]; ok {
		t.Errorf("file entry = %v; \"lossy\" key present, want absent on the happy path", m)
	}
}

// TestAnswerJSON_SymbolsThreeStates covers the "symbols" key's three states: absent when not
// requested, present as "[]" when requested of a file with no declaration, and present and
// populated otherwise.
func TestAnswerJSON_SymbolsThreeStates(t *testing.T) {
	t.Run("AbsentOnDirectoryQueryByDefault", func(t *testing.T) {
		r := openScratchRepo(t, "answer-json-symbols-absent", map[string]string{"foo.go": "package p\n\nfunc F() {}\n"})
		got, err := r.TOC(".", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		m := marshalToMap(t, got.Files[0])
		if _, ok := m["symbols"]; ok {
			t.Errorf("file entry = %v; \"symbols\" key present, want absent — a directory query defaults Symbols to false", m)
		}
	})

	t.Run("PresentEmptyForFileWithNoDeclaration", func(t *testing.T) {
		r := openScratchRepo(t, "answer-json-symbols-empty", map[string]string{"foo.go": "package p\n"})
		got, err := r.TOC("foo.go", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		m := marshalToMap(t, got.Files[0])
		raw, ok := m["symbols"]
		if !ok {
			t.Fatal("\"symbols\" key absent; want present as [] for a file target with no declaration")
		}
		arr, ok := raw.([]any)
		if !ok || len(arr) != 0 {
			t.Errorf("symbols = %v; want an empty array", raw)
		}
	})

	t.Run("PresentAndPopulated", func(t *testing.T) {
		r := openScratchRepo(t, "answer-json-symbols-populated", map[string]string{"foo.go": "package p\n\nfunc F() {}\n"})
		got, err := r.TOC("foo.go", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		m := marshalToMap(t, got.Files[0])
		raw, ok := m["symbols"]
		if !ok {
			t.Fatal("\"symbols\" key absent; want present and populated")
		}
		arr, ok := raw.([]any)
		if !ok || len(arr) == 0 {
			t.Errorf("symbols = %v; want a populated array", raw)
		}
	})
}

// TestAnswerKnobs_Depth covers Depth 0, 1, and DepthAll over the committed tree/ fixture.
func TestAnswerKnobs_Depth(t *testing.T) {
	r := openModuleRepo(t)

	t.Run("DepthZeroListsDirectSubdirectoriesIdentityOnly", func(t *testing.T) {
		got, err := r.TOC("internal/engine/testdata/tree", TOCOptions{Depth: 0})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		if len(got.Files) == 0 {
			t.Error("Files is empty; want the tree/ directory's own top-level files")
		}
		for _, d := range got.Dirs {
			m := marshalToMap(t, d)
			if _, ok := m["files"]; ok {
				t.Errorf("%s: \"files\" key present; want absent at the Depth-0 cut", d.Dir)
			}
			if _, ok := m["dirs"]; ok {
				t.Errorf("%s: \"dirs\" key present; want absent at the Depth-0 cut", d.Dir)
			}
		}
	})

	t.Run("DepthOneFillsOneLevelLeafIdentityOnly", func(t *testing.T) {
		got, err := r.TOC("internal/engine/testdata/tree", TOCOptions{Depth: 1})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		var sub *DirAnswer
		for i := range got.Dirs {
			if got.Dirs[i].Dir == "internal/engine/testdata/tree/sub" {
				sub = &got.Dirs[i]
			}
		}
		if sub == nil {
			t.Fatal("no \"sub\" entry in Dirs")
		}
		if len(sub.Files) == 0 {
			t.Error("sub.Files is empty; want sub's own files filled at Depth 1")
		}
		if len(sub.Dirs) != 1 || sub.Dirs[0].Dir != "internal/engine/testdata/tree/sub/deep" {
			t.Fatalf("sub.Dirs = %+v; want exactly the identity-only \"deep\" entry", sub.Dirs)
		}
		if len(sub.Dirs[0].Files) != 0 {
			t.Errorf("sub/deep.Files = %+v; want absent at the depth cut one level further down", sub.Dirs[0].Files)
		}
	})

	t.Run("DepthAllRecursesToTheBottom", func(t *testing.T) {
		got, err := r.TOC("internal/engine/testdata/tree", TOCOptions{Depth: DepthAll})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		var sub *DirAnswer
		for i := range got.Dirs {
			if got.Dirs[i].Dir == "internal/engine/testdata/tree/sub" {
				sub = &got.Dirs[i]
			}
		}
		if sub == nil {
			t.Fatal("no \"sub\" entry in Dirs")
		}
		if len(sub.Dirs) != 1 || len(sub.Dirs[0].Files) == 0 {
			t.Errorf("sub/deep = %+v; want its own Files filled under DepthAll", sub.Dirs)
		}
	})
}

// TestAnswerKnobs_SymbolsDefaultsAndOverrides covers Symbols defaulting per target kind and both
// explicit overrides winning.
func TestAnswerKnobs_SymbolsDefaultsAndOverrides(t *testing.T) {
	files := map[string]string{"foo.go": "package p\n\nfunc F() {}\n"}

	t.Run("FileTargetDefaultsTrue", func(t *testing.T) {
		r := openScratchRepo(t, "answer-symbols-file-default", files)
		got, err := r.TOC("foo.go", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		if got.Files[0].Symbols == nil {
			t.Error("Symbols == nil; want non-nil by default for a file target")
		}
	})

	t.Run("DirectoryTargetDefaultsFalse", func(t *testing.T) {
		r := openScratchRepo(t, "answer-symbols-dir-default", files)
		got, err := r.TOC(".", TOCOptions{})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		if got.Files[0].Symbols != nil {
			t.Error("Symbols != nil; want nil by default for a directory target")
		}
	})

	t.Run("DirectoryTargetExplicitTrueWins", func(t *testing.T) {
		r := openScratchRepo(t, "answer-symbols-dir-override-true", files)
		got, err := r.TOC(".", TOCOptions{Symbols: boolPtr(true)})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		if got.Files[0].Symbols == nil {
			t.Error("Symbols == nil; want non-nil when explicitly requested for a directory target")
		}
	})

	t.Run("FileTargetExplicitFalseWins", func(t *testing.T) {
		r := openScratchRepo(t, "answer-symbols-file-override-false", files)
		got, err := r.TOC("foo.go", TOCOptions{Symbols: boolPtr(false)})
		if err != nil {
			t.Fatalf("TOC returned error: %v", err)
		}
		if got.Files[0].Symbols != nil {
			t.Error("Symbols != nil; want nil when explicitly suppressed for a file target")
		}
	})
}

// TestAnswerKnobs_FileTargetShapeAndDepthIgnored asserts a file target answers as a one-entry
// directory answer carrying the enclosing directory's facts and no Dirs, and that Depth changes
// nothing about that answer.
func TestAnswerKnobs_FileTargetShapeAndDepthIgnored(t *testing.T) {
	r := openModuleRepo(t)

	gotDepthZero, err := r.TOC("internal/engine/testdata/tree/pkg/alpha.go", TOCOptions{Depth: 0})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if gotDepthZero.Dir != "internal/engine/testdata/tree/pkg" {
		t.Errorf("Dir = %q; want the enclosing directory", gotDepthZero.Dir)
	}
	if gotDepthZero.Package != "pkg" {
		t.Errorf("Package = %q; want %q, the enclosing directory's package", gotDepthZero.Package, "pkg")
	}
	if len(gotDepthZero.Files) != 1 || gotDepthZero.Files[0].Name != "alpha.go" {
		t.Fatalf("Files = %+v; want exactly alpha.go", gotDepthZero.Files)
	}
	if len(gotDepthZero.Dirs) != 0 {
		t.Errorf("Dirs = %+v; want none for a file target", gotDepthZero.Dirs)
	}

	gotDepthAll, err := r.TOC("internal/engine/testdata/tree/pkg/alpha.go", TOCOptions{Depth: DepthAll})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if gotDepthAll.Dir != gotDepthZero.Dir || gotDepthAll.Package != gotDepthZero.Package ||
		len(gotDepthAll.Files) != len(gotDepthZero.Files) || len(gotDepthAll.Dirs) != len(gotDepthZero.Dirs) {
		t.Errorf("Depth changed a file target's answer: Depth=0 -> %+v, Depth=DepthAll -> %+v", gotDepthZero, gotDepthAll)
	}
}

// TestAnswerFailureEntries_BrokenFixture covers the broken/ fixture's three files: the well-formed
// one, the syntax error (Lossy, no Error), and the invalid-UTF-8 file (Error, no Lossy) — all
// still listed, never skipped.
func TestAnswerFailureEntries_BrokenFixture(t *testing.T) {
	r := openModuleRepo(t)

	got, err := r.TOC("internal/engine/testdata/broken", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if len(got.Files) != 3 {
		t.Fatalf("len(Files) = %d; want 3 — every file listed regardless of failure", len(got.Files))
	}

	ok := entryByName(t, got.Files, "ok.go")
	if ok.Error != "" || ok.Lossy {
		t.Errorf("ok.go = %+v; want neither Error nor Lossy set", ok)
	}

	syntax := entryByName(t, got.Files, "syntax.go")
	if !syntax.Lossy {
		t.Error("syntax.go Lossy = false; want true")
	}
	if syntax.Error != "" {
		t.Errorf("syntax.go Error = %q; want empty", syntax.Error)
	}

	invalid := entryByName(t, got.Files, "invalid.go")
	if invalid.Error == "" {
		t.Error("invalid.go Error is empty; want it set")
	}
	if invalid.Lossy {
		t.Error("invalid.go Lossy = true; want false")
	}
}

// TestAnswerFailureEntries_UnreadableFile covers an unreadable file, built at run time since a
// committed file cannot be unreadable. Skipped when running as a user for whom chmod 0000 has no
// effect (e.g. root).
func TestAnswerFailureEntries_UnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as a privileged user for whom chmod 0000 does not block reads")
	}
	root := writeScratchTree(t, "answer-unreadable-file", map[string]string{
		"unreadable.go": "package p\n",
	})
	path := filepath.Join(root, "unreadable.go")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("os.Chmod(%q, 0000) failed: %v", path, err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("chmod 0000 did not block reads in this environment")
	}
	r := openRepo(t, root)

	got, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	entry := entryByName(t, got.Files, "unreadable.go")
	if entry.Error == "" {
		t.Error("Error is empty; want it set for an unreadable file")
	}
	if entry.Lossy {
		t.Error("Lossy = true; want false")
	}
}

// TestAnswerExtensionlessFileHeaderRule covers Makefile's hash-block header and notes.rst's
// no-rule-at-all case over the committed tree/ fixture.
func TestAnswerExtensionlessFileHeaderRule(t *testing.T) {
	r := openModuleRepo(t)

	got, err := r.TOC("internal/engine/testdata/tree", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}

	makefile := entryByName(t, got.Files, "Makefile")
	if makefile.Header == "" {
		t.Error("Makefile Header is empty; want the hash-block header rule to apply")
	}

	notes := entryByName(t, got.Files, "notes.rst")
	if notes.Header != "" {
		t.Errorf("notes.rst Header = %q; want empty — no rule covers this extension", notes.Header)
	}
}

// TestAnswerIgnoreSetFreshness asserts two TOC calls on one Repo see the .gitignore as it stands
// at the time of each call: nothing is cached on Repo, including the ignore set.
func TestAnswerIgnoreSetFreshness(t *testing.T) {
	root := writeScratchTree(t, "answer-ignore-freshness", map[string]string{
		".gitignore": "",
		"a.go":       "package p\n",
		"b.go":       "package p\n",
	})
	r := openRepo(t, root)

	before, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	if len(before.Files) != 3 {
		t.Fatalf("before: len(Files) = %d; want 3 (.gitignore, a.go, b.go)", len(before.Files))
	}

	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("b.go\n"), 0o644); err != nil {
		t.Fatalf("rewriting .gitignore failed: %v", err)
	}

	after, err := r.TOC(".", TOCOptions{})
	if err != nil {
		t.Fatalf("TOC returned error: %v", err)
	}
	for _, f := range after.Files {
		if f.Name == "b.go" {
			t.Errorf("after: Files = %+v; b.go should now be excluded by the rewritten .gitignore", after.Files)
		}
	}
	if len(after.Files) != 2 {
		t.Errorf("after: len(Files) = %d; want 2 (.gitignore, a.go)", len(after.Files))
	}
}
