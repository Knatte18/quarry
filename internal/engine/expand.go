// expand.go implements the exported expand verb: what a type consists of, across files. It holds
// Repo.Expand and its unexported worker, and NotATypeError, the one typed failure the verb returns.

package engine

import (
	"fmt"

	"github.com/Knatte18/quarry/glyph"
)

// NotATypeError is returned by Repo.Expand when target's glyph names a match set holding no
// KindType symbol — a function, a method, a const, a var, or the several-declaration "init" glyph,
// whatever its count. ID is the glyph's own String() form and Kind is the kind of the first match in
// file-then-line order.
//
// It is a struct rather than a bare sentinel because docs/rewrite-plan.md's expand rule — the glyph
// must name a type, and on any other kind the answer names the kind — requires the kind to be
// carried, and a caller mapping engine failures to a status word and an exit code needs it without
// parsing a message. That is the same argument that split ErrTargetOutsideRepo from
// ErrTargetNotFound in internal/engine/repo.go.
//
// NotATypeError is returned as Expand's own error and is never carried inside ExpandAnswer: an
// ok-plus-kind pair inside the payload would duplicate the envelope a later task owns, inside the
// data.
//
// NotATypeError is never returned under a unit collision, because the glyph does not unambiguously
// name anything there and naming a kind would be a claim the answer cannot support — a collision
// answers StatusAmbiguous instead, before the kind gate is ever reached.
type NotATypeError struct {
	// ID is the glyph's own String() form.
	ID string
	// Kind is the kind of the first match in file-then-line order.
	Kind Kind
}

// Error implements the error interface, naming the glyph and the kind it actually named.
func (e *NotATypeError) Error() string {
	return fmt.Sprintf("engine: expand %s: not a type, kind %s", e.ID, e.Kind)
}

// Expand answers target: the target type's own head, plus every member whose owner chain begins
// with it. Expand builds one unitMemo with newUnitMemo, returning that error if it fails, and
// delegates to the unexported expand — the same shell-plus-worker shape Resolve has, for the same
// reason: the memo is a local that dies with the call, and a test can construct one and pass it in.
func (r *Repo) Expand(target string) (ExpandAnswer, error) {
	m, err := newUnitMemo(r)
	if err != nil {
		return ExpandAnswer{}, err
	}
	return r.expand(target, m)
}

// expand is Expand's unexported worker. It takes the memo, rather than building its own, so a test
// can construct one, pass it in, and read parses afterwards; Expand itself never exposes it.
//
// expand parses target with glyph.Parse(glyph.Go, target) and, on failure, returns the zero
// ExpandAnswer and the parse error wrapped as "engine: expand <target>: <err>" with %w, so errors.As
// still reaches the *glyph.ParseError. This includes a target with no "#", which the grammar already
// rejects as its no-separator reason: expand writes no separator check of its own, because every
// alphabet question is one glyph.Parse call and a hand-rolled check would return a different error
// type for one of the grammar's rejection reasons than for all the others. Unlike Resolve, expand
// answers one target, so there is no other answer to protect and no reason to move the failure into
// the payload.
//
// It sets id to the parsed glyph's String(), reads m.dirsOf(g.Unit) for the directory list and the
// collision flag, reads m.symbolsOf(g.Unit) — an error from which is returned as this call's own
// error — and filters with matchesFor.
//
// It then calls statusForMatches(g, matches, dirs.collision), the one place a match set becomes a
// status, so expand can never disagree with Resolve about what a glyph resolves to, and adds a kind
// gate between that function's zero-and-collision rows and its remaining ones, mapping the four
// returned values onto ExpandAnswer in exactly this order:
//
//  1. StatusNotFound: Status is StatusNotFound, Unit is StatusFound when the memoised directory list
//     is non-empty and StatusNotFound when it is empty, and the answer carries no head, members or
//     candidates. This is statusForMatches' own first row, evaluated before its collision row, which
//     is why a collision with no match in either directory answers not_found with unit: found here
//     exactly as it does in Resolve.
//  2. StatusAmbiguous and the collision flag is set: Status is StatusAmbiguous, Candidates is the
//     match set, and there is no head and no members. Checked before the kind gate, so a single type
//     match under a collision answers ambiguous here exactly as it does in Resolve, and
//     *NotATypeError is never returned under a collision — the glyph does not unambiguously name
//     anything, so naming a kind would be a claim the answer cannot support.
//  3. The kind gate, applied to whatever statusForMatches returned other than the two rows above —
//     StatusFound, StatusMultipart, or a non-collision StatusAmbiguous. When no match in the set is
//     KindType, whatever the count, expand returns the zero ExpandAnswer and a *NotATypeError
//     carrying the glyph's string form and the kind of the first match in file-then-line order.
//     Keying the gate on no match being a type rather than on the match count is what gives the
//     several-declaration init glyph a defined answer: statusForMatches calls that set
//     StatusMultipart, a status ExpandAnswer does not admit, and the gate catches it because init's
//     kind is function. It also catches a single non-type match, which statusForMatches calls
//     StatusFound.
//  4. The set holds at least one type and statusForMatches did not return StatusFound: Status is
//     StatusAmbiguous and Candidates is the match set. This covers both several type declarations
//     under one glyph — docs/glyph.md §5's build-tag duplicates, the only way a Go type multiplies,
//     since docs/rewrite-plan.md says a Go type never splits and only init does — and a mixed set
//     naming a type and something else, where choosing between them would be a silent pick.
//  5. StatusFound and the single match is a type: Status is StatusFound, with the head and members
//     below.
//
// The head is the matched type's own Symbol, copied by value, with two substitutions and nothing
// else: Start becomes the symbol's HeadStart and End becomes its HeadEnd. Every other field — the
// id, kind, file, sigend, signature and doc — is the symbol's own, because one symbol entry is what
// all three verbs return for a symbol and expand emits no shape of its own. Assign its address to
// Head. If the matched KindType symbol has HeadStart == 0, expand returns the zero ExpandAnswer and a
// plain fmt.Errorf naming the glyph's string form and stating that a type symbol carries no head
// span: that is an invariant violation in the walk, and a silent fallback to Start and End would hide
// it behind an answer that happens to be right for Go.
//
// Contract gap, closed by nobody: substituting Start and End with the head span means the type's
// full declaration span is not recoverable from an ExpandAnswer for a language whose head is a
// strict subset — while docs/rewrite-plan.md's three-queries section says the whole class is the
// type symbol's own start-end and "is available, never the default". For Go the two spans are
// identical, so nothing is lost and the gap is unreachable; the first language whose head is a
// strict subset has to decide whether expand carries both spans, and that decision belongs to that
// language's task, against a repository that needs it. docs/rewrite-plan.md's "the class span minus
// its member spans" — its phrase, in the three-queries section, not docs/glyph.md's — describes what
// a reader ends up reading, not arithmetic this verb performs: for a Go struct the subtraction is
// empty, and for an interface the answer already carries every member's start and end, so a consumer
// wanting only the non-member lines has what it needs and the engine emits one contiguous head entry
// rather than a discontiguous span type the closed symbol shape does not have.
//
// The span is read from the head fields rather than re-derived because, for Go, the two pairs are
// identical, so nothing observable changes today; the head field is what a language whose head is a
// strict subset of its declaration would read instead of re-deriving the rule here.
//
// The members are every symbol in the unit's symbol slice — not the match set — with
// len(s.Glyph.Owner) > 0 && s.Glyph.Owner[0] == g.Name. The type symbol itself has no owner and is
// excluded by that filter, so it is never both head and member. The slice symbolsOf returns is
// already ordered by file then start line and the filter preserves it, so no second sort is needed
// and none is written. Members is left nil when the filter selects nothing, so omitempty drops the
// key: a type with no members is found with a head and nothing else, not an error and not a
// not_found — the type exists and consists of its head.
//
// Matching on the first owner element rather than the whole chain is the general form: in Go the
// chain is at most one element because the grammar rejects a deeper member outright, so the two are
// the same today and the general form is what a nested-type language needs. An interface method is a
// member by this same rule with no special case, since the walk gives a method element the
// interface's own type name as its owner, and it sorts into place inside the head range by file and
// line. Only the glyph's own unit is searched: the external test unit is a different unit and cannot
// declare methods on this unit's types.
func (r *Repo) expand(target string, m *unitMemo) (ExpandAnswer, error) {
	g, err := glyph.Parse(glyph.Go, target)
	if err != nil {
		return ExpandAnswer{}, fmt.Errorf("engine: expand %s: %w", target, err)
	}
	id := g.String()

	dirsRes := m.dirsOf(g.Unit)
	symbols, err := m.symbolsOf(g.Unit)
	if err != nil {
		return ExpandAnswer{}, err
	}

	matches := matchesFor(symbols, g)
	status := statusForMatches(g, matches, dirsRes.collision)

	if status == StatusNotFound {
		unit := StatusNotFound
		if len(dirsRes.dirs) > 0 {
			unit = StatusFound
		}
		return ExpandAnswer{ID: id, Status: StatusNotFound, Unit: unit}, nil
	}
	if status == StatusAmbiguous && dirsRes.collision {
		return ExpandAnswer{ID: id, Status: StatusAmbiguous, Candidates: matches}, nil
	}

	hasType := false
	for _, match := range matches {
		if match.Kind == KindType {
			hasType = true
			break
		}
	}
	if !hasType {
		return ExpandAnswer{}, &NotATypeError{ID: id, Kind: matches[0].Kind}
	}
	if status != StatusFound {
		return ExpandAnswer{ID: id, Status: StatusAmbiguous, Candidates: matches}, nil
	}

	typeSym := matches[0]
	if typeSym.HeadStart == 0 {
		return ExpandAnswer{}, fmt.Errorf("engine: expand %s: type symbol carries no head span", id)
	}
	head := typeSym
	head.Start = typeSym.HeadStart
	head.End = typeSym.HeadEnd

	var members []Symbol
	for _, sym := range symbols {
		if len(sym.Glyph.Owner) > 0 && sym.Glyph.Owner[0] == g.Name {
			members = append(members, sym)
		}
	}

	return ExpandAnswer{ID: id, Status: StatusFound, Head: &head, Members: members}, nil
}
