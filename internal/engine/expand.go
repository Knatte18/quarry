// expand.go implements the exported expand verb: what a type consists of, across files. It holds
// Repo.Expand and its unexported worker, and the two typed failures the verb returns:
// NotATypeError and SelfGlyphError.

package engine

import (
	"errors"
	"fmt"
	"sort"

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

// SelfGlyphError is returned by Repo.Expand when target parses as a self glyph — one naming its
// unit's whole directory or file rather than a member within it. ID is the glyph's own String()
// form.
//
// It is a struct, in the same shape as NotATypeError, rather than a bare sentinel, for the same
// reason NotATypeError is: a caller mapping engine failures to a status word and an exit code
// needs the glyph's identity carried with the error, without parsing a message.
//
// SelfGlyphError is a typed error rather than another Kind value: Kind is documented as the closed
// vocabulary a Symbol's kind is drawn from and the values toc emits, a self glyph is not a
// symbol, and widening a symbol vocabulary to describe a non-symbol would falsify that comment.
//
// It is also not answered as not_found: a self glyph matches no symbol, so the answer would fall
// out as not_found, which says the name is free when the truth is that the verb does not apply to
// a self glyph at all.
type SelfGlyphError struct {
	// ID is the glyph's own String() form.
	ID string
}

// Error implements the error interface, naming the glyph that named itself rather than a member.
func (e *SelfGlyphError) Error() string {
	return fmt.Sprintf("engine: expand %s: not a type, a self glyph names the unit or file itself", e.ID)
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

// expandOutcome is what one alphabet made of a target: its answer or typed failure, and its raw match set.
type expandOutcome struct {
	lang    glyph.Language
	answer  ExpandAnswer
	matches []Symbol
	err     error
}

// contributes reports whether the outcome is anything but not_found, a NotATypeError included.
func (o expandOutcome) contributes() bool {
	return o.err != nil || o.answer.Status != StatusNotFound
}

// expand is Expand's unexported worker.
// It takes the memo, rather than building its own, so a test can construct one, pass it in, and read extractions afterwards.
//
// It parses target under glyph.Go and glyph.CSharp.
// A target no alphabet accepts returns the zero ExpandAnswer and the parse error rejectionFor selects.
// The error is wrapped as "engine: expand <target>: <err>" with %w, so errors.As still reaches the *glyph.ParseError.
// This includes a target with no "#", which the grammar already rejects as its no-separator reason:
// every alphabet question is one glyph.Parse call, and a hand-rolled check would return a different error type for one rejection reason than for the rest.
// Unlike Resolve, expand answers one target, so there is no other answer to protect and no reason to move the failure into the payload.
//
// A self glyph returns the zero ExpandAnswer and a *SelfGlyphError once, before any alphabet answers,
// so no unit is parsed and no scan is built for a question that has no answer.
//
// Each accepting alphabet answers through expandIn, and the answers merge:
//
//   - One alphabet whose answer is not not_found, a NotATypeError included: its answer or error stands.
//   - Two such alphabets: ambiguous, with both raw match sets as candidates, each carrying its language.
//   - None: not_found, with Unit found when either alphabet found its unit.
//
// An error from expandIn other than a NotATypeError is returned as this call's own error.
func (r *Repo) expand(target string, m *unitMemo) (ExpandAnswer, error) {
	var parsed []glyph.Glyph
	var goErr, csharpErr error
	for _, lang := range []glyph.Language{glyph.Go, glyph.CSharp} {
		g, err := glyph.Parse(lang, target)
		switch {
		case err == nil:
			parsed = append(parsed, g)
		case lang == glyph.Go:
			goErr = err
		default:
			csharpErr = err
		}
	}
	if len(parsed) == 0 {
		return ExpandAnswer{}, fmt.Errorf("engine: expand %s: %w", target, rejectionFor(goErr, csharpErr))
	}
	if parsed[0].IsSelf() {
		return ExpandAnswer{}, &SelfGlyphError{ID: parsed[0].String()}
	}

	var outcomes []expandOutcome
	for _, g := range parsed {
		answer, matches, err := r.expandIn(g, m)
		var notAType *NotATypeError
		if err != nil && !errors.As(err, &notAType) {
			return ExpandAnswer{}, err
		}
		outcomes = append(outcomes, expandOutcome{lang: g.Lang, answer: answer, matches: matches, err: err})
	}

	var standing []expandOutcome
	for _, outcome := range outcomes {
		if outcome.contributes() {
			standing = append(standing, outcome)
		}
	}
	switch len(standing) {
	case 0:
		merged := outcomes[0].answer
		for _, outcome := range outcomes {
			if outcome.answer.Unit == StatusFound {
				merged.Unit = StatusFound
			}
		}
		return merged, nil
	case 1:
		return standing[0].answer, standing[0].err
	}

	merged := ExpandAnswer{ID: standing[0].answer.ID, Status: StatusAmbiguous}
	if merged.ID == "" {
		merged.ID = parsed[0].String()
	}
	for _, outcome := range standing {
		for _, sym := range outcome.matches {
			sym.Language = string(outcome.lang)
			merged.Candidates = append(merged.Candidates, sym)
		}
	}
	sort.SliceStable(merged.Candidates, func(i, j int) bool {
		if merged.Candidates[i].File != merged.Candidates[j].File {
			return merged.Candidates[i].File < merged.Candidates[j].File
		}
		return merged.Candidates[i].Start < merged.Candidates[j].Start
	})
	return merged, nil
}

// expandIn answers g, the parse of the target under one alphabet, and returns the alphabet's raw match set beside the answer.
// A Go unit is read through its directories and a C# unit through the call's namespace scan.
// The match set is returned even alongside a *NotATypeError, so a caller merging alphabets can offer it as candidates.
//
// The matches are g's owner chain, name and parameters read through matchesFor.
// statusForMatches decides the status, so expand can never disagree with Resolve about what a glyph resolves to.
// A kind gate sits between that function's zero-and-collision rows and its remaining ones, mapping the result onto ExpandAnswer in this order:
//
//  1. StatusNotFound: Status is StatusNotFound and Unit is StatusFound when the unit exists, StatusNotFound otherwise.
//     It is evaluated before the collision row, so a collision with no match answers not_found with unit found exactly as Resolve does.
//  2. StatusAmbiguous under a unit collision: Candidates is the match set, with no head and no members.
//     It is checked before the kind gate, so a *NotATypeError is never returned under a collision.
//  3. The kind gate: when no match is KindType, whatever the count, it returns the zero ExpandAnswer and a *NotATypeError
//     carrying the glyph's string form and the kind of the first match in file-then-line order.
//     This gives the several-declaration Go "init" glyph a defined answer, and a single non-type match too.
//  4. StatusMultipart with a type among the matches, which only a C# glyph can produce: Status is StatusMultipart,
//     Parts holds every part's head, and Members holds the members of all parts. There is no Head.
//  5. Any other status than StatusFound with a type in the set: Status is StatusAmbiguous and Candidates is the match set.
//     This covers build-tag duplicates and a mixed set naming a type and something else, where choosing would be a silent pick.
//  6. StatusFound with a type: Status is StatusFound, with the head and members below.
//
// A head is the matched type's own Symbol, copied by value, with Start and End substituted from HeadStart and HeadEnd.
// Every other field is the symbol's own, because one symbol entry is what toc, resolve and expand all return for a symbol.
// A type whose HeadStart is 0 is an invariant violation in the walk, reported as a plain error naming the glyph;
// a silent fallback to Start and End would hide it behind an answer that happens to be right for Go.
//
// A C# type's head is its declaration through its base list, a strict subset of its span,
// so the span of the whole declaration is recoverable only from resolve.
//
// The members are every symbol of the unit, not the match set, whose owner chain starts with g's owner chain followed by g's name.
// The type itself has a shorter chain and is never its own member.
// For Go the owner chain of a type is empty, so the rule selects the symbols whose first owner is the type's name.
// For a nested C# type it selects the nested type's members and never those of a top-level type of the same name,
// and it includes an explicit interface implementation whose minted chain passes through the target's chain.
// The unit's symbols are ordered by file then start line and the filter preserves that order.
// Members is nil when nothing matches, so a type with no members is found with a head and nothing else.
// Only the glyph's own unit is searched: the Go external test unit is a different unit and cannot declare members of this unit's types.
func (r *Repo) expandIn(g glyph.Glyph, m *unitMemo) (ExpandAnswer, []Symbol, error) {
	id := g.String()

	var symbols []Symbol
	var collision, unitFound bool
	if g.Lang == glyph.CSharp {
		scan, err := m.csharp()
		if err != nil {
			return ExpandAnswer{}, nil, err
		}
		symbols = scan.symbols[g.Unit]
		unitFound = scan.declares(g.Unit)
	} else {
		dirsRes := m.dirsOf(g.Unit)
		collision = dirsRes.collision
		unitFound = len(dirsRes.dirs) > 0
		var err error
		if symbols, err = m.symbolsOf(g.Unit); err != nil {
			return ExpandAnswer{}, nil, err
		}
	}

	matches := matchesFor(symbols, g)
	status := statusForMatches(g, matches, collision)

	if status == StatusNotFound {
		unit := StatusNotFound
		if unitFound {
			unit = StatusFound
		}
		return ExpandAnswer{ID: id, Status: StatusNotFound, Unit: unit}, matches, nil
	}
	if status == StatusAmbiguous && collision {
		return ExpandAnswer{ID: id, Status: StatusAmbiguous, Candidates: matches}, matches, nil
	}

	hasType := false
	for _, match := range matches {
		if match.Kind == KindType {
			hasType = true
			break
		}
	}
	if !hasType {
		return ExpandAnswer{}, matches, &NotATypeError{ID: id, Kind: matches[0].Kind}
	}
	if status != StatusFound && status != StatusMultipart {
		return ExpandAnswer{ID: id, Status: StatusAmbiguous, Candidates: matches}, matches, nil
	}

	heads := make([]Symbol, 0, len(matches))
	for _, match := range matches {
		if match.HeadStart == 0 {
			return ExpandAnswer{}, matches, fmt.Errorf("engine: expand %s: type symbol carries no head span", id)
		}
		head := match
		head.Start = match.HeadStart
		head.End = match.HeadEnd
		heads = append(heads, head)
	}

	chain := append(append([]string(nil), g.Owner...), g.Name)
	var members []Symbol
	for _, sym := range symbols {
		if ownerStartsWith(sym.Glyph.Owner, chain) {
			members = append(members, sym)
		}
	}

	if status == StatusMultipart {
		return ExpandAnswer{ID: id, Status: StatusMultipart, Parts: heads, Members: members}, matches, nil
	}
	return ExpandAnswer{ID: id, Status: StatusFound, Head: &heads[0], Members: members}, matches, nil
}

// ownerStartsWith reports whether owner begins with every element of prefix, in order.
func ownerStartsWith(owner, prefix []string) bool {
	if len(owner) < len(prefix) {
		return false
	}
	for i, name := range prefix {
		if owner[i] != name {
			return false
		}
	}
	return true
}
