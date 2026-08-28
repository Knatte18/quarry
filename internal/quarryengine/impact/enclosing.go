// enclosing.go implements the pure enclosing-symbol selection function impact uses for both the
// resolved target's own declaration site and every caller call site, plus the per-call parse cache
// that backs it: fileCache, newFileCache, (*fileCache).resolve, and (*fileCache).cached.

package impact

import "github.com/Knatte18/quarry/internal/quarryengine/toc"

// enclosingSymbol returns the symbol in symbols whose range [Start, End] covers line, both bounds
// inclusive. When more than one symbol's range covers line, the symbol with the greatest Start
// wins — not slice order. toc documents Symbols as source-ordered by Start, which makes "last
// match wins" and "greatest Start wins" equivalent today, but only the latter survives a future
// ordering change, so that is the contract this function commits to.
//
// The overlap case this rule exists for is Python and C# class-and-method nesting, where a
// KindType symbol spans every KindMethod symbol nested inside it: a line inside a method body is
// covered by both the class's range and the method's, and the method — the later, greater-Start
// declaration — is the intended answer. A grouped Go type declaration is not an overlap case:
// each spec inside a "type ( ... )" block has its own range computed from the spec itself, so
// sibling specs never overlap.
//
// ok is false when no symbol's range covers line, including when symbols is empty.
func enclosingSymbol(symbols []toc.Symbol, line int) (toc.Symbol, bool) {
	var best toc.Symbol
	found := false
	for _, sym := range symbols {
		if line < sym.Start || line > sym.End {
			continue
		}
		if !found || sym.Start > best.Start {
			best = sym
			found = true
		}
	}
	return best, found
}

// parseFunc is the injectable seam fileCache parses through: given an absolute file path, it
// returns that file's toc.FileTOC or an error. It exists so cards 6 and 7's tests can inject a
// counting parse function and assert the cache's one-parse-per-path guarantee without touching
// disk or an LSP.
type parseFunc func(path string) (toc.FileTOC, error)

// fileResult records the outcome of parsing one file: either a successful toc.FileTOC or the error
// that parse produced, never both meaningfully populated at once.
type fileResult struct {
	toc toc.FileTOC
	err error
}

// fileCache is a per-call, absolute-path-keyed cache of toc parse results. It parses each distinct
// path at most once per fileCache instance and remembers both the successful result and any
// per-file error, so a repeated lookup of a file that failed to parse does not retry the parse.
//
// fileCache is deliberately scoped to a single call: it must never be stored at package level or
// reused across calls, since toc itself is documented to spawn no daemon and cache nothing, and a
// longer-lived cache would go stale against on-disk edits between calls.
type fileCache struct {
	parse   parseFunc
	results map[string]fileResult
}

// newFileCache constructs a fileCache backed by parse. A nil parse selects the production parse,
// toc.TOCFile(path, "", toc.Options{DocSentences: 0}) — DocSentences: 0 is deliberate: impact never
// emits docstring text, and per toc.TOCFile's own contract Start, SigEnd, and End are never
// affected by that setting, so the omission costs nothing the range-based lookup needs.
func newFileCache(parse parseFunc) *fileCache {
	if parse == nil {
		parse = func(path string) (toc.FileTOC, error) {
			return toc.TOCFile(path, "", toc.Options{DocSentences: 0})
		}
	}
	return &fileCache{parse: parse, results: make(map[string]fileResult)}
}

// resolve returns the cached toc.FileTOC or error for path, parsing path through c.parse exactly
// once per distinct path and caching whichever outcome that call produced.
func (c *fileCache) resolve(path string) (toc.FileTOC, error) {
	if result, ok := c.results[path]; ok {
		return result.toc, result.err
	}
	fileTOC, err := c.parse(path)
	c.results[path] = fileResult{toc: fileTOC, err: err}
	return fileTOC, err
}

// cached reports whether path has already been parsed by this fileCache instance. It exists so the
// caller loop in impact.go can distinguish a cache hit (free) from a cache miss (about to pay for a
// parse) for its cancellation check.
func (c *fileCache) cached(path string) bool {
	_, ok := c.results[path]
	return ok
}
