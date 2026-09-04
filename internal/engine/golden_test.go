// golden_test.go pins Repo.TOC's output against two real targets in a Loomyard checkout at
// loomyardPin, taken from the rewrite plan's own §4 examples: the render directory as a directory
// target with default options, and that directory's layout.go source file as a file target. Both
// cases are gated by loomyardRepo (loomyard_test.go) and skip together with the rest of the
// Loomyard suite on a machine with no checkout, or one at the wrong commit fails loudly instead —
// see that helper's own doc comment for why the two are not symmetric.
//
// Under the "-update" flag (also declared in loomyard_test.go) each case rewrites its own committed
// golden from the current checkout instead of comparing against it. The two committed goldens can
// only be produced this way, against a checkout at loomyardPin: a hand-written or invented golden
// would pin the wrong bytes and pass forever, which is exactly the failure this comment warns the
// next -update run against.

package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGolden_LoomyardRenderDir compares Repo.TOC("internal/reedengine/render", TOCOptions{}) — a
// directory target with default options — against the committed render-dir.json.
func TestGolden_LoomyardRenderDir(t *testing.T) {
	repoRoot := loomyardRepo(t)
	r := openRepo(t, repoRoot)

	got, err := r.TOC("internal/reedengine/render", TOCOptions{})
	if err != nil {
		t.Fatalf(`TOC("internal/reedengine/render", TOCOptions{}) failed: %v`, err)
	}
	compareGolden(t, "render-dir.json", got)
}

// TestGolden_LoomyardRenderLayoutFile compares
// Repo.TOC("internal/reedengine/render/layout.go", TOCOptions{}) — a file target, whose one Files
// entry therefore carries Symbols — against the committed render-layout-file.json.
func TestGolden_LoomyardRenderLayoutFile(t *testing.T) {
	repoRoot := loomyardRepo(t)
	r := openRepo(t, repoRoot)

	got, err := r.TOC("internal/reedengine/render/layout.go", TOCOptions{})
	if err != nil {
		t.Fatalf(`TOC("internal/reedengine/render/layout.go", TOCOptions{}) failed: %v`, err)
	}
	compareGolden(t, "render-layout-file.json", got)
}

// TestGolden_LoomyardParentDirDepthZero asserts the shape of a subdirectory entry at depth zero,
// rather than comparing against a committed golden: the point here is a shape, not prose, per the
// rewrite plan's second §4 example. It queries the render directory's own parent,
// "internal/reedengine", at TOCOptions{} default depth (0), finds the "render" entry in the
// result's Dirs, and asserts on that one entry's own marshalled JSON that it carries "dir",
// "package" and "doc" and no other key — so an accidentally-populated "files" or "language" fails
// the test rather than silently passing.
func TestGolden_LoomyardParentDirDepthZero(t *testing.T) {
	repoRoot := loomyardRepo(t)
	r := openRepo(t, repoRoot)

	got, err := r.TOC("internal/reedengine", TOCOptions{})
	if err != nil {
		t.Fatalf(`TOC("internal/reedengine", TOCOptions{}) failed: %v`, err)
	}

	var renderEntry *DirAnswer
	for i := range got.Dirs {
		if got.Dirs[i].Dir == "internal/reedengine/render" {
			renderEntry = &got.Dirs[i]
			break
		}
	}
	if renderEntry == nil {
		t.Fatalf(`no "internal/reedengine/render" entry in Dirs; want one at depth 0`)
	}

	entryJSON, err := json.Marshal(renderEntry)
	if err != nil {
		t.Fatalf("marshal render subdirectory entry: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(entryJSON, &raw); err != nil {
		t.Fatalf("unmarshal render subdirectory entry: %v", err)
	}

	wantKeys := map[string]bool{"dir": true, "package": true, "doc": true}
	for key := range raw {
		if !wantKeys[key] {
			t.Errorf("render subdirectory entry carries unexpected key %q: %s", key, entryJSON)
		}
	}
	for key := range wantKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("render subdirectory entry is missing key %q: %s", key, entryJSON)
		}
	}
}

// compareGolden marshals got to indented JSON and either compares it byte for byte against the
// committed golden testdata/loomyard/name, or — under "-update" — rewrites that golden from got.
// The indentation and trailing newline are fixed here once, so a golden produced by one run of
// -update compares equal to itself on every later run that made no source change.
func compareGolden(t *testing.T, name string, got DirAnswer) {
	t.Helper()

	gotJSON, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal golden %q: %v", name, err)
	}
	gotJSON = append(gotJSON, '\n')

	path := filepath.Join("testdata", "loomyard", name)
	if *updateGoldens {
		if err := os.WriteFile(path, gotJSON, 0o644); err != nil {
			t.Fatalf("write golden %q: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %q: %v", path, err)
	}
	if !bytes.Equal(want, gotJSON) {
		t.Errorf("golden %q mismatch (-want +got):\n--- want ---\n%s\n--- got ---\n%s", name, want, gotJSON)
	}
}
