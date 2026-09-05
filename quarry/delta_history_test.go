// delta_history_test.go runs (*Repo).DeltaGit against this repository's own history, pinned to the
// commit pair _mill/discussion.md's "Real-history pin" section names: d413ceb (parent) to 49304ca
// ("Glyph self-form and the resolve contract", C1), scoped separately to glyph/ and
// internal/engine/ — the two directories that commit touches.
//
// Every assertion below is presence-only over the in-scope subset the discussion records — "these
// symbols appear in this array" — never exact-set equality, because the pin's own sets are
// explicitly partial: two of the commit's deletions (internal/repopath#RepoRelPath and
// internal/cli#isGlyphTarget) fall outside both scoped directories and are never asserted here.
//
// TestDeltaRealHistory skips cleanly, rather than failing, whenever either pinned revision is
// unreachable in this checkout -- a shallow clone is exactly that case, and it is a normal state for
// this test to be run against, following the skip-versus-fail asymmetry
// internal/engine/loomyard_test.go's own loomyardRepo already establishes for an unreachable
// checkout. The repository root is resolved from this file's own location, never from an
// environment variable, since the repository under test is this one.

package quarry

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"
)

// realHistoryFrom and realHistoryTo are the commit pair the discussion's "Real-history pin" section
// names: d413ceb is 49304ca's parent.
const (
	realHistoryFrom = "d413ceb"
	realHistoryTo   = "49304ca"
)

// realHistoryRepoRoot resolves this repository's own absolute root from this test file's location.
// thisFile is .../quarry/delta_history_test.go; quarry/ sits one directory below the module root,
// the same one-step-up rule scratchtree_test.go's own writeScratchTree already documents for this
// package.
func realHistoryRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("realHistoryRepoRoot: runtime.Caller(0) failed to resolve this file's path")
	}
	return filepath.Dir(filepath.Dir(thisFile))
}

// deltaGitRealHistoryOrSkip opens root and runs DeltaGit(realHistoryFrom, realHistoryTo, target)
// against it, skipping the whole test -- with the reason, never failing it -- when either pinned
// revision does not resolve in this checkout. Any other error fails the test: a machine that can
// see this history and disagrees with the pin is a real failure, not a normal state.
func deltaGitRealHistoryOrSkip(t *testing.T, root, target string) GitDeltaAnswer {
	t.Helper()

	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	answer, err := r.DeltaGit(realHistoryFrom, realHistoryTo, target)
	if err != nil {
		var revErr *UnknownRevisionError
		if errors.As(err, &revErr) {
			t.Skipf("revision %q is unreachable in this checkout (%v); a shallow clone is the normal case this test skips for", revErr.Rev, err)
		}
		t.Fatalf("DeltaGit(%q, %q, %q) returned error: %v", realHistoryFrom, realHistoryTo, target, err)
	}
	return answer
}

// findRenameCandidateEntry returns the RenameCandidateEntry in entries whose ID equals id, and
// whether one was found.
func findRenameCandidateEntry(entries []RenameCandidateEntry, id string) (RenameCandidateEntry, bool) {
	for _, e := range entries {
		if e.ID == id {
			return e, true
		}
	}
	return RenameCandidateEntry{}, false
}

// findRenameCandidate returns the RenameCandidate in candidates whose ID equals id, and whether one
// was found.
func findRenameCandidate(candidates []RenameCandidate, id string) (RenameCandidate, bool) {
	for _, c := range candidates {
		if c.ID == id {
			return c, true
		}
	}
	return RenameCandidate{}, false
}

// renamedContainsQuarry reports whether pairs holds a RenamedPair whose From.ID equals fromID and
// To.ID equals toID. Named distinctly from internal/engine/delta_rename_test.go's own
// renamedContains, which this file cannot import: Go test helpers are not importable across
// packages.
func renamedContainsQuarry(pairs []RenamedPair, fromID, toID string) bool {
	for _, p := range pairs {
		if p.From.ID == fromID && p.To.ID == toID {
			return true
		}
	}
	return false
}

// TestDeltaRealHistory pins (*Repo).DeltaGit against this repository's own history over the
// realHistoryFrom..realHistoryTo commit pair, scoped separately to glyph/ and internal/engine/, per
// this file's own header comment.
func TestDeltaRealHistory(t *testing.T) {
	root := realHistoryRepoRoot(t)

	t.Run("Glyph", func(t *testing.T) {
		ans := deltaGitRealHistoryOrSkip(t, root, "glyph")

		for _, id := range []string{"glyph#Self", "glyph#Glyph.IsSelf"} {
			if !hasSymbolID(ans.Created, id) {
				t.Errorf("Created is missing %q; Created = %+v", id, ans.Created)
			}
		}
		if len(ans.Modified) == 0 {
			t.Error("Modified is empty; want at least one modified entry in glyph over this commit pair")
		}
	})

	t.Run("Engine", func(t *testing.T) {
		ans := deltaGitRealHistoryOrSkip(t, root, "internal/engine")

		for _, id := range []string{
			"internal/engine#SelfGlyphError",
			"internal/engine#SelfGlyphError.Error",
			"internal/engine#Repo.resolveSelfTarget",
		} {
			if !hasSymbolID(ans.Created, id) {
				t.Errorf("Created is missing %q; Created = %+v", id, ans.Created)
			}
		}

		const deletedID = "internal/engine#Repo.resolvePathTarget"
		if !hasSymbolID(ans.Deleted, deletedID) {
			t.Errorf("Deleted is missing %q; Deleted = %+v", deletedID, ans.Deleted)
		}
		if len(ans.Modified) == 0 {
			t.Error("Modified is empty; want at least one modified entry in internal/engine over this commit pair")
		}

		// The load-bearing assertion: resolvePathTarget -> resolveSelfTarget is the one genuine
		// evidence-tier rename this history exercises. It must be present as a reported candidate,
		// never as an asserted pair -- the parameter itself was renamed (target -> unit), not merely
		// the function, so the signature signal is false.
		const renamedID = "internal/engine#Repo.resolveSelfTarget"
		if renamedContainsQuarry(ans.Renamed, deletedID, renamedID) {
			t.Errorf("Renamed = %+v; want the pair absent -- it is reported by the evidence tier, never asserted by the exact tier", ans.Renamed)
		}
		entry, ok := findRenameCandidateEntry(ans.RenameCandidates, deletedID)
		if !ok {
			t.Fatalf("RenameCandidates is missing an entry for %q; RenameCandidates = %+v", deletedID, ans.RenameCandidates)
		}
		cand, ok := findRenameCandidate(entry.Candidates, renamedID)
		if !ok {
			t.Fatalf("Candidates for %q is missing %q; Candidates = %+v", deletedID, renamedID, entry.Candidates)
		}
		if cand.Signals.SignatureIdenticalModuloName {
			t.Error("SignatureIdenticalModuloName = true; want false -- the parameter itself was renamed (target -> unit), not only the function identifier")
		}
	})
}
