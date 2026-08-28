// impact.go implements buildResult, the pure result-assembly seam, and the Impact entry point.

// Package impact composes query.Callers' verified caller set with toc's declaration ranges,
// answering "who calls this, and what declaration is each call site inside". It sits above both
// query and toc in the engine DAG, introduces no options type of its own, and issues no LSP
// request query.Callers does not already issue — every toc parse it performs is local, on-disk
// work over files query.Callers' references already named.
package impact

import (
	"context"
	"sort"

	"github.com/Knatte18/quarry/internal/quarryengine/query"
)

// buildResult assembles a Result from callers (query.Callers' verified reference set) and
// declaration (its definition-only result at the query position), resolving every entry's
// enclosing declaration through cache. It is separate from Impact for the same reason query's own
// callersFromClient seam is: the assembly is testable against a hand-built reference set with no
// LSP.
//
// Declaration exclusion: every entry in declaration is excluded from the reported callers, using
// the same set-membership rule internal/cli's filterUnexpectedCallers applies for
// assert-no-callers, re-implemented here because the seam guard bans the engine from importing any
// internal/*cli package. Only that set is excluded — a recursive call inside the target's own body
// is an ordinary caller whose enclosing symbol is the target itself, and is kept.
//
// Target and Definition: when declaration is empty, both are omitted and buildResult still returns
// the caller list with a nil error — their joint absence means "the language server returned no
// definition for the query position". Otherwise declaration[0] (the set is already sorted, so this
// is deterministic) supplies Definition's File and Line, and is run through the same enclosing
// lookup every caller uses, applying the three outcomes of the three-outcome-degradation-rule: a
// resolver error fills Definition.Error and leaves Target nil; no matching enclosing symbol leaves
// both the range fields and Error unset and Target nil; a match fills StartLine/SigEndLine/EndLine
// and builds Target from that same toc.Symbol plus the parsed file's declared package. Target's
// provenance is always that one toc.Symbol — never the query string and never an LSP candidate.
//
// Callers: the returned slice is initialized empty-but-non-nil so it marshals as "[]". For each
// surviving reference, in input order, File/CallSiteLine/CallSiteCharacter are set from the
// reference (query.Reference.File is already absolute — it is never re-resolved against a working
// directory), then its enclosing declaration is resolved through the same lookup: a resolver error
// sets the entry's Error and leaves EnclosingRange nil; no matching symbol leaves both unset; a
// match fills EnclosingRange and the identity fields. A file-scope entry still keeps Package, since
// the file parsed successfully and its declared package is known — only the symbol-level fields
// are absent there. One entry is emitted per call site: two calls to the target inside one
// enclosing function produce two entries with identical enclosing ranges.
//
// Cancellation: in the caller loop, immediately before resolving a reference whose file is not
// already cached, buildResult returns ctx.Err() if it is non-nil. The definition-side lookup is
// deliberately not cancellation-checked: it is a single parse of one file, performed once before
// the loop, whereas the loop's cost grows with the caller file count — the unbounded quantity the
// parse-loop-cancellation-scope Shared Decision scopes the check to. Adding a second check there
// would guard a bounded cost while implying a granularity the backend cannot deliver. buildResult
// never attempts to interrupt a parse already in flight, and threads no deadline into toc.TOCFile.
//
// The assembled entries are sorted by file, then line, then character, with a stable sort, so the
// ordering guarantee is local to this function rather than inherited from the caller.
func buildResult(ctx context.Context, callers, declaration []query.Reference, cache *fileCache) (Result, error) {
	declSet := make(map[query.Reference]bool, len(declaration))
	for _, d := range declaration {
		declSet[d] = true
	}

	result := Result{Callers: []Caller{}}
	if len(declaration) > 0 {
		def := declaration[0]
		definition := &Definition{File: def.File, Line: def.Line}
		fileTOC, err := cache.resolve(def.File)
		if err != nil {
			definition.Error = err.Error()
		} else if sym, ok := enclosingSymbol(fileTOC.Symbols, def.Line); ok {
			definition.StartLine = sym.Start
			definition.SigEndLine = sym.SigEnd
			definition.EndLine = sym.End
			result.Target = &Target{
				Kind:      sym.Kind,
				Name:      sym.Name,
				Owner:     sym.Owner,
				Package:   fileTOC.Package,
				Signature: sym.Signature,
			}
		}
		result.Definition = definition
	}

	for _, ref := range callers {
		if declSet[ref] {
			continue
		}

		// Check for cancellation only immediately before a cache miss is about to pay for a real
		// parse — a cache hit is free, and re-checking on every already-resolved reference would
		// guard nothing.
		if !cache.cached(ref.File) && ctx.Err() != nil {
			return Result{}, ctx.Err()
		}

		caller := Caller{File: ref.File, CallSiteLine: ref.Line, CallSiteCharacter: ref.Character}
		fileTOC, err := cache.resolve(ref.File)
		if err != nil {
			caller.Error = err.Error()
		} else {
			caller.Package = fileTOC.Package
			if sym, ok := enclosingSymbol(fileTOC.Symbols, ref.Line); ok {
				caller.EnclosingRange = &Range{StartLine: sym.Start, SigEndLine: sym.SigEnd, EndLine: sym.End}
				caller.Kind = sym.Kind
				caller.Name = sym.Name
				caller.Owner = sym.Owner
				caller.Signature = sym.Signature
			}
		}
		result.Callers = append(result.Callers, caller)
	}

	sort.SliceStable(result.Callers, func(i, j int) bool {
		a, b := result.Callers[i], result.Callers[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.CallSiteLine != b.CallSiteLine {
			return a.CallSiteLine < b.CallSiteLine
		}
		return a.CallSiteCharacter < b.CallSiteCharacter
	})

	return result, nil
}

// Impact resolves opts.Query against the language server for opts.TargetDir and returns the
// resolved declaration's identity and location alongside every verified caller, each annotated
// with its enclosing declaration when one exists.
//
// Impact never sets opts.SkipVerification — it passes opts through to query.Callers unchanged —
// but makes no claim that verification ran: per the
// verification-is-best-effort-and-resolution-means-what-it-means-on-refs Shared Decision,
// "resolution":"complete" on impact means exactly what it already means on refs, that the language
// server returned every reference for the query as given, and asserts nothing about per-caller
// verification having run or enclosing ranges having resolved.
//
// The returned caller list already excludes the resolved declaration's own site(s), and is not
// filtered by any directory scope.
func Impact(ctx context.Context, opts query.Options) (Result, error) {
	callers, declaration, err := query.Callers(ctx, opts)
	if err != nil {
		return Result{}, err
	}
	return buildResult(ctx, callers, declaration, newFileCache(nil))
}
