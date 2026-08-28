// facade_test.go asserts that quarry/facade.go is a pure re-export of internal/quarryengine
// rather than a wrapper — the one way it can break silently. Every aliased type must be a true
// `=` alias (an assignment across it, in either direction, compiles with no conversion), every
// re-exported sentinel must be the identical error value the engine holds (not a re-created
// error that merely has an equal message), and every delegating function's signature must not
// have drifted from what it forwards to. A mismatch found here is a real defect in batch 2 card
// 9's facade construction and must be fixed in facade.go, not papered over in this file.

package quarry

import (
	"context"
	"testing"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/query"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
	"github.com/Knatte18/quarry/internal/quarryengine/toc"
)

// The pairs below compile only if each facade type is a true `=` alias of its underlying engine
// type: assigning the engine value to the facade-typed variable, then assigning it straight back
// with no conversion, fails to compile the moment either side becomes a distinct defined type
// instead of an alias. Package-level variables are never subject to Go's "declared and not used"
// check, so no test body needs to read them — the compiler is the entire check.
var (
	aliasCheckEntryEngine                  registry.Entry
	aliasCheckEntryFacade                  Entry
	aliasCheckRegistryEngine               registry.Registry
	aliasCheckRegistryFacade               Registry
	aliasCheckPositionEngine               quarryengine.Position
	aliasCheckPositionFacade               Position
	aliasCheckQueryEngine                  query.Query
	aliasCheckQueryFacade                  Query
	aliasCheckInFileQueryEngine            query.InFileQuery
	aliasCheckInFileQueryFacade            InFileQuery
	aliasCheckOptionsEngine                query.Options
	aliasCheckOptionsFacade                Options
	aliasCheckReferenceEngine              query.Reference
	aliasCheckReferenceFacade              Reference
	aliasCheckSymbolMatchEngine            query.SymbolMatch
	aliasCheckSymbolMatchFacade            SymbolMatch
	aliasCheckErrServerNotFoundEngine      quarryengine.ErrServerNotFound
	aliasCheckErrServerNotFoundFacade      ErrServerNotFound
	aliasCheckErrSymbolNotFoundEngine      quarryengine.ErrSymbolNotFound
	aliasCheckErrSymbolNotFoundFacade      ErrSymbolNotFound
	aliasCheckErrAmbiguousSymbolEngine     quarryengine.ErrAmbiguousSymbol
	aliasCheckErrAmbiguousSymbolFacade     ErrAmbiguousSymbol
	aliasCheckErrResolverUnsupportedEngine quarryengine.ErrResolverUnsupported
	aliasCheckErrResolverUnsupportedFacade ErrResolverUnsupported
	aliasCheckErrServerTimeoutEngine       quarryengine.ErrServerTimeout
	aliasCheckErrServerTimeoutFacade       ErrServerTimeout
	aliasCheckErrServerSpawnTimeoutEngine  quarryengine.ErrServerSpawnTimeout
	aliasCheckErrServerSpawnTimeoutFacade  ErrServerSpawnTimeout
	aliasCheckTOCSymbolEngine              toc.Symbol
	aliasCheckTOCSymbolFacade              TOCSymbol
	aliasCheckTOCKindEngine                toc.Kind
	aliasCheckTOCKindFacade                TOCKind
	aliasCheckTOCFileResultEngine          toc.FileTOC
	aliasCheckTOCFileResultFacade          TOCFileResult
	aliasCheckTOCDirEntryEngine            toc.DirEntry
	aliasCheckTOCDirEntryFacade            TOCDirEntry
	aliasCheckTOCDirResultEngine           toc.DirTOC
	aliasCheckTOCDirResultFacade           TOCDirResult
	aliasCheckTOCOptionsEngine             toc.Options
	aliasCheckTOCOptionsFacade             TOCOptions
)

// init performs the actual round-trip assignment for each of the twenty aliased types: engine
// value into the facade-typed variable, then straight back into the engine-typed variable, with
// no conversion on either side.
func init() {
	aliasCheckEntryFacade = aliasCheckEntryEngine
	aliasCheckEntryEngine = aliasCheckEntryFacade

	aliasCheckRegistryFacade = aliasCheckRegistryEngine
	aliasCheckRegistryEngine = aliasCheckRegistryFacade

	aliasCheckPositionFacade = aliasCheckPositionEngine
	aliasCheckPositionEngine = aliasCheckPositionFacade

	aliasCheckQueryFacade = aliasCheckQueryEngine
	aliasCheckQueryEngine = aliasCheckQueryFacade

	aliasCheckInFileQueryFacade = aliasCheckInFileQueryEngine
	aliasCheckInFileQueryEngine = aliasCheckInFileQueryFacade

	aliasCheckOptionsFacade = aliasCheckOptionsEngine
	aliasCheckOptionsEngine = aliasCheckOptionsFacade

	aliasCheckReferenceFacade = aliasCheckReferenceEngine
	aliasCheckReferenceEngine = aliasCheckReferenceFacade

	aliasCheckSymbolMatchFacade = aliasCheckSymbolMatchEngine
	aliasCheckSymbolMatchEngine = aliasCheckSymbolMatchFacade

	aliasCheckErrServerNotFoundFacade = aliasCheckErrServerNotFoundEngine
	aliasCheckErrServerNotFoundEngine = aliasCheckErrServerNotFoundFacade

	aliasCheckErrSymbolNotFoundFacade = aliasCheckErrSymbolNotFoundEngine
	aliasCheckErrSymbolNotFoundEngine = aliasCheckErrSymbolNotFoundFacade

	aliasCheckErrAmbiguousSymbolFacade = aliasCheckErrAmbiguousSymbolEngine
	aliasCheckErrAmbiguousSymbolEngine = aliasCheckErrAmbiguousSymbolFacade

	aliasCheckErrResolverUnsupportedFacade = aliasCheckErrResolverUnsupportedEngine
	aliasCheckErrResolverUnsupportedEngine = aliasCheckErrResolverUnsupportedFacade

	aliasCheckErrServerTimeoutFacade = aliasCheckErrServerTimeoutEngine
	aliasCheckErrServerTimeoutEngine = aliasCheckErrServerTimeoutFacade

	aliasCheckErrServerSpawnTimeoutFacade = aliasCheckErrServerSpawnTimeoutEngine
	aliasCheckErrServerSpawnTimeoutEngine = aliasCheckErrServerSpawnTimeoutFacade

	aliasCheckTOCSymbolFacade = aliasCheckTOCSymbolEngine
	aliasCheckTOCSymbolEngine = aliasCheckTOCSymbolFacade

	aliasCheckTOCKindFacade = aliasCheckTOCKindEngine
	aliasCheckTOCKindEngine = aliasCheckTOCKindFacade

	aliasCheckTOCFileResultFacade = aliasCheckTOCFileResultEngine
	aliasCheckTOCFileResultEngine = aliasCheckTOCFileResultFacade

	aliasCheckTOCDirEntryFacade = aliasCheckTOCDirEntryEngine
	aliasCheckTOCDirEntryEngine = aliasCheckTOCDirEntryFacade

	aliasCheckTOCDirResultFacade = aliasCheckTOCDirResultEngine
	aliasCheckTOCDirResultEngine = aliasCheckTOCDirResultFacade

	aliasCheckTOCOptionsFacade = aliasCheckTOCOptionsEngine
	aliasCheckTOCOptionsEngine = aliasCheckTOCOptionsFacade
}

// The twelve blank-identifier assignments below reference every delegating function in
// facade.go, each against the exact func type its engine counterpart demands: a signature drift
// on either side fails the build.
var (
	_ func() Registry                                         = BuiltinRegistry
	_ func(string) (Registry, error)                          = LoadRegistry
	_ func(string, Registry, string) (string, Entry, error)   = DetectLanguage
	_ func(string, string) string                             = DaemonStateFile
	_ func(string, string) string                             = DaemonLock
	_ func(context.Context, Options) ([]Reference, error)     = References
	_ func(context.Context, Options) ([]Reference, error)     = Definition
	_ func(context.Context, Options) ([]SymbolMatch, error)   = Symbol
	_ func(string, string, TOCOptions) (TOCFileResult, error) = TOCFile
	_ func(string, string) (TOCDirResult, error)              = TOCDir
	_ func() []string                                         = TOCLanguages
	_ func() []string                                         = TOCImplemented
)

// TestFacadeSentinels_Identity verifies each of the eight re-exported sentinel error values is
// the identical value the engine holds, not merely a re-created error with an equal message —
// errors.Is/== comparisons against the facade's sentinel would silently stop matching engine
// errors if a re-export were ever replaced with a fresh errors.New call.
func TestFacadeSentinels_Identity(t *testing.T) {
	tests := []struct {
		name   string
		facade error
		engine error
	}{
		{"ErrNoLanguage", ErrNoLanguage, quarryengine.ErrNoLanguage},
		{"ErrServerNotFoundSentinel", ErrServerNotFoundSentinel, quarryengine.ErrServerNotFoundSentinel},
		{"ErrSymbolNotFoundSentinel", ErrSymbolNotFoundSentinel, quarryengine.ErrSymbolNotFoundSentinel},
		{"ErrAmbiguousSymbolSentinel", ErrAmbiguousSymbolSentinel, quarryengine.ErrAmbiguousSymbolSentinel},
		{"ErrResolverUnsupportedSentinel", ErrResolverUnsupportedSentinel, quarryengine.ErrResolverUnsupportedSentinel},
		{"ErrServerTimeoutSentinel", ErrServerTimeoutSentinel, quarryengine.ErrServerTimeoutSentinel},
		{"ErrServerSpawnTimeoutSentinel", ErrServerSpawnTimeoutSentinel, quarryengine.ErrServerSpawnTimeoutSentinel},
		{"ErrLanguageUnsupported", ErrLanguageUnsupported, quarryengine.ErrLanguageUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.facade != tt.engine {
				t.Errorf("quarry.%s = %v; want the identical quarryengine sentinel value %v, got a distinct error (re-created via errors.New?)", tt.name, tt.facade, tt.engine)
			}
		})
	}
}
