//go:build lsp

// buildtags_lsp_test.go exercises References against a real, held-open gopls subprocess to prove
// the build-tag visibility defect issue #2 describes: a reference behind a "//go:build" constraint
// is invisible until --build-tags names that constraint. This is gopls's own behaviour, not this
// wrapper's wiring, so it is not reproducible against a fake server — the same reason
// refs_integration_test.go's own live subtests exist under this build tag. Only the gopls-spawning
// subtest is guarded on exec.LookPath("gopls") (via t.Skip).

package query

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/daemon/daemontest"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
)

// TestReferences_BuildTagVisibility_Integration proves that a reference gated behind a
// "//go:build sometag" constraint is absent from the untagged References answer and present once
// --build-tags names that tag — the whole defect issue #2 describes.
func TestReferences_BuildTagVisibility_Integration(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip(registry.BuiltinRegistry()["go"].InstallHint)
	}

	root := repoRoot(t)
	fixtureRoot := filepath.Join(root, "testdata", "buildtagfixture")
	libFile := filepath.Join(fixtureRoot, "lib", "lib.go")
	plainFile := filepath.Join(fixtureRoot, "consumer", "plain.go")
	taggedFile := filepath.Join(fixtureRoot, "consumer", "tagged.go")

	pos := findFuncPosition(t, libFile, "QuarryBuildTagProbe")

	untaggedRefs := queryBuildTagFixtureReferences(t, fixtureRoot, pos, nil)
	foundPlain, foundTagged := classifyBuildTagFixtureRefs(untaggedRefs, plainFile, taggedFile)
	if !foundPlain {
		t.Errorf("References(QuarryBuildTagProbe, no --build-tags) = %+v; want it to include the untagged call site %s", untaggedRefs, plainFile)
	}
	if foundTagged {
		t.Errorf("References(QuarryBuildTagProbe, no --build-tags) = %+v; want it to exclude the tag-gated call site %s", untaggedRefs, taggedFile)
	}

	taggedRefs := queryBuildTagFixtureReferences(t, fixtureRoot, pos, []string{"sometag"})
	foundPlain, foundTagged = classifyBuildTagFixtureRefs(taggedRefs, plainFile, taggedFile)
	if !foundPlain {
		t.Errorf("References(QuarryBuildTagProbe, --build-tags sometag) = %+v; want it to still include the untagged call site %s", taggedRefs, plainFile)
	}
	if !foundTagged {
		t.Errorf("References(QuarryBuildTagProbe, --build-tags sometag) = %+v; want it to include the tag-gated call site %s", taggedRefs, taggedFile)
	}
}

// queryBuildTagFixtureReferences calls References against the buildtagfixture module at pos with
// buildTags, using its own isolated t.TempDir() state directory (so two calls with different tag
// sets are never served by the same warm daemon view) and reaping the supervised daemon it spawns
// in t.Cleanup — Go's registry entry has HasNativeDaemon true, so every call here routes through
// the supervised strategy, whose teardown branch deliberately never kills the daemon it spawns.
func queryBuildTagFixtureReferences(t *testing.T, fixtureRoot string, pos quarryengine.Position, buildTags []string) []Reference {
	t.Helper()

	stateDir := t.TempDir()
	statePath := daemontest.StateFile(stateDir, "go")
	t.Cleanup(func() { daemontest.KillRecordedDaemon(t, statePath) })

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	refs, err := References(ctx, Options{
		Registry:  registry.BuiltinRegistry(),
		TargetDir: fixtureRoot,
		StateDir:  stateDir,
		Lang:      "go",
		Query:     Query{Pos: &pos},
		Timeout:   30 * time.Second,
		BuildTags: buildTags,
	})
	if err != nil {
		t.Fatalf("References(QuarryBuildTagProbe, BuildTags=%v) returned unexpected error: %v", buildTags, err)
	}
	return refs
}

// classifyBuildTagFixtureRefs reports whether refs contains the plain and tagged consumer call
// sites, respectively.
func classifyBuildTagFixtureRefs(refs []Reference, plainFile, taggedFile string) (foundPlain, foundTagged bool) {
	for _, ref := range refs {
		switch filepath.Clean(ref.File) {
		case filepath.Clean(plainFile):
			foundPlain = true
		case filepath.Clean(taggedFile):
			foundTagged = true
		}
	}
	return foundPlain, foundTagged
}
