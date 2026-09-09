// quarry_test.go covers the vocabulary vars quarry.go re-exports from the engine: that Statuses and
// NameReasons are the engine's own slices, not copies, and that the Known and Rejected methods added
// to the engine's Status and ResolveResult types are reachable through this package's own aliased
// spellings without importing the engine.

package quarry

import (
	"testing"

	"github.com/Knatte18/quarry/internal/engine"
)

// TestFacadeVocabulariesAreEngineValues asserts three things: Statuses and NameReasons are each the
// engine's own slice rather than a copy, and Status.Known and ResolveResult.Rejected are reachable
// through the quarry package's own aliased Status and ResolveResult spellings.
func TestFacadeVocabulariesAreEngineValues(t *testing.T) {
	t.Run("StatusesIsEngineStatuses", func(t *testing.T) {
		if len(Statuses) != len(engine.Statuses) {
			t.Fatalf("len(Statuses) = %d; want %d", len(Statuses), len(engine.Statuses))
		}
		if len(Statuses) == 0 {
			t.Fatal("Statuses is empty; want a non-empty slice to compare addresses against")
		}
		if &Statuses[0] != &engine.Statuses[0] {
			t.Error("&Statuses[0] != &engine.Statuses[0]; want the engine's own slice, not a copy")
		}
	})

	t.Run("NameReasonsIsEngineNameReasons", func(t *testing.T) {
		if len(NameReasons) != len(engine.NameReasons) {
			t.Fatalf("len(NameReasons) = %d; want %d", len(NameReasons), len(engine.NameReasons))
		}
		if len(NameReasons) == 0 {
			t.Fatal("NameReasons is empty; want a non-empty slice to compare addresses against")
		}
		if &NameReasons[0] != &engine.NameReasons[0] {
			t.Error("&NameReasons[0] != &engine.NameReasons[0]; want the engine's own slice, not a copy")
		}
	})

	t.Run("MethodsReachableThroughAliasedTypes", func(t *testing.T) {
		// This is a compile-level assertion first: Status("found").Known() and
		// ResolveResult{}.Rejected() only compile if the aliased types carry the engine's methods.
		known := Status("found").Known()
		rejected := ResolveResult{}.Rejected()
		if !known {
			t.Errorf("Status(%q).Known() = %v; want true", "found", known)
		}
		if !rejected {
			t.Errorf("ResolveResult{}.Rejected() = %v; want true", rejected)
		}
	})
}
