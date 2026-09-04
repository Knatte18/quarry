// decls.go carries the package-level const and var shapes glyph_test.go's const/var coverage
// exercises: the ungrouped, single-spec-grouped, several-names-per-spec, and bare-iota forms.

package glyphs

// UngroupedConst is a plain ungrouped const declaration.
const UngroupedConst = 1

// UngroupedVar is a plain ungrouped var declaration.
var UngroupedVar int

const (
	// GroupedConst is the sole spec in a single-spec group.
	GroupedConst = 2
)

var (
	// GroupedVar is the sole spec in a single-spec group.
	GroupedVar string
)

const (
	// GroupedMultiA and GroupedMultiB share one spec.
	GroupedMultiA, GroupedMultiB = 3, 4
)

// Weekday enumerates days by iota, so Tuesday is a bare spec with no type and no value of its own.
type Weekday int

const (
	Monday Weekday = iota
	Tuesday
)
