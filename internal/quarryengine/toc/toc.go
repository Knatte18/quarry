// toc.go implements the two entry points the toc package exports: TOCFile and TOCDir. Both resolve
// a language, drive treesitter.WithTree, dispatch to the resolved language's registered Strategy,
// and apply the two post-processing rules that belong to the entry point rather than to any
// strategy — first-paragraph header truncation (FirstParagraph) and, for TOCFile alone, sentence
// trimming of every symbol's docstring (FirstSentences). Putting both rules here rather than in the
// strategies gives each rule exactly one call site and keeps --doc-sentences from being threaded
// through every strategy.

package toc

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
	"github.com/Knatte18/quarry/internal/quarryengine/treesitter"
)

// resolveLanguage resolves the canonical language name for path, honoring langOverride when it is
// non-empty. A non-empty langOverride wins outright regardless of path's extension — a mismatch is
// not an error here, matching what --lang means on every existing verb. Validating langOverride
// against toc's own vocabulary (Implemented()) is the CLI's job, not this function's: a second
// validation here would drift from the flag's own error message over time.
//
// An extension that maps to no language, or a resolved language with no registered Strategy (a
// designed-but-unimplemented language), both return a wrapped quarryengine.ErrLanguageUnsupported.
// The unimplemented case is detected via StrategyFor rather than a second hard-coded language list,
// so the two checks can never drift apart.
func resolveLanguage(path, langOverride string) (string, error) {
	lang := langOverride
	if lang == "" {
		resolved, ok := registry.LanguageForExtension(filepath.Ext(path))
		if !ok {
			return "", fmt.Errorf("toc: %s: no language for extension %q: %w", path, filepath.Ext(path), quarryengine.ErrLanguageUnsupported)
		}
		lang = resolved
	}
	if _, ok := StrategyFor(lang); !ok {
		return "", fmt.Errorf("toc: %s: language %q has no toc strategy: %w", path, lang, quarryengine.ErrLanguageUnsupported)
	}
	return lang, nil
}

// TOCFile extracts a FileTOC from the file at path.
//
// Language resolution: when langOverride is non-empty it wins outright and path's extension is
// ignored — a mismatch is not an error. Otherwise the language is resolved from path's extension.
// See resolveLanguage's doc comment for the full rule and the ErrLanguageUnsupported cases.
//
// path is read with os.ReadFile; a read failure returns that error wrapped, with no sentinel of its
// own. Content that is not valid UTF-8 is rejected with a plain error naming path — this is the
// "never parsed at all" case, distinct from a lossy (Partial) parse.
//
// Header is FirstParagraph(strategy.Header(root, src)): the same truncation TOCDir applies, so a
// package-documentation file with zero symbols does not return its whole contents.
//
// opts.DocSentences governs every symbol's emitted Docstring: 0 clears it so the "docstring" JSON
// key is omitted, AllSentences leaves it whole, and any other N replaces it with
// FirstSentences(sym.Docstring, N). Start, SigEnd, and End are never touched by this policy — they
// always cover the whole docstring, so a truncated docstring field and a read of start–sigend or
// start–end stay consistent. That is what makes DocSentences: 0 a discovery mode rather than a
// lossy one.
func TOCFile(path string, langOverride string, opts Options) (FileTOC, error) {
	lang, err := resolveLanguage(path, langOverride)
	if err != nil {
		return FileTOC{}, err
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return FileTOC{}, fmt.Errorf("toc: read %s: %w", path, err)
	}
	if !utf8.Valid(src) {
		return FileTOC{}, fmt.Errorf("toc: %s: not valid UTF-8", path)
	}

	strategy, _ := StrategyFor(lang)

	var result FileTOC
	err = treesitter.WithTree(lang, src, func(root *ts.Node, partial bool) error {
		symbols := strategy.Symbols(root, src)
		applyDocSentences(symbols, opts.DocSentences)
		result = FileTOC{
			Language: lang,
			Package:  strategy.Package(root, src),
			Header:   FirstParagraph(strategy.Header(root, src)),
			Symbols:  symbols,
			Partial:  partial,
		}
		return nil
	})
	if err != nil {
		return FileTOC{}, err
	}
	return result, nil
}

// applyDocSentences rewrites every symbol's Docstring in place per docSentences: 0 clears the field
// so its JSON key is omitted, AllSentences leaves it unchanged, and any other value replaces it with
// FirstSentences(sym.Docstring, docSentences). A cleared field is the only mechanism used to omit
// the key — an empty-but-present docstring is never written.
func applyDocSentences(symbols []Symbol, docSentences int) {
	for i := range symbols {
		switch docSentences {
		case 0:
			symbols[i].Docstring = ""
		case AllSentences:
			// leave Docstring unchanged
		default:
			symbols[i].Docstring = FirstSentences(symbols[i].Docstring, docSentences)
		}
	}
}

// TOCDir extracts a DirTOC from exactly one directory level of dir. It never recurses and never
// descends into a subdirectory — a subdirectory entry is simply skipped.
//
// For each remaining entry, the language is resolved from its extension; an extension mapping to no
// language skips the file entirely, so a Markdown or YAML file never appears. When langOverride is
// non-empty, the listing is restricted to that language's extensions via
// registry.ExtensionsForLanguage, rather than reinterpreting every other file under the override.
//
// The surviving entries are sorted lexicographically by base filename with an explicit sort.Slice,
// never left in os.ReadDir's own order.
//
// Error and Partial are mutually exclusive by construction: Partial is only ever set on the route
// that actually parsed the file through treesitter.WithTree. A language with no registered
// Strategy, an unreadable file, and invalid UTF-8 all set Error instead and leave Header and Partial
// unset — the file is still listed, never skipped, and still counts as a code file for the
// empty-directory question: a directory holding only unimplemented-language files returns a
// non-empty Files, not an empty one.
//
// A directory containing no file with a supported extension returns a DirTOC whose Files is an
// empty, non-nil slice and a nil error — a true answer to "what code is in here", not a failure.
//
// TOCDir imposes no file-size cap: parse cost is linear and the runtime enforces its own work
// budgets, so a pathological file surfaces as a slow parse or as Partial, never as a special-cased
// refusal.
//
// TOCDir takes no Options: it emits headers, never docstrings, so the doc-sentences policy has
// nothing to affect here — that asymmetry with TOCFile is deliberate, not an oversight.
func TOCDir(dir string, langOverride string) (DirTOC, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return DirTOC{}, fmt.Errorf("toc: read dir %s: %w", dir, err)
	}

	var allowedExts map[string]bool
	if langOverride != "" {
		exts := registry.ExtensionsForLanguage(langOverride)
		allowedExts = make(map[string]bool, len(exts))
		for _, ext := range exts {
			allowedExts[ext] = true
		}
	}

	result := DirTOC{Files: []DirEntry{}}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		base := entry.Name()
		lang, ok := registry.LanguageForExtension(filepath.Ext(base))
		if !ok {
			continue
		}
		if allowedExts != nil && !allowedExts[filepath.Ext(base)] {
			continue
		}
		result.Files = append(result.Files, buildDirEntry(dir, base, lang))
	}

	sort.Slice(result.Files, func(i, j int) bool {
		return result.Files[i].Name < result.Files[j].Name
	})

	return result, nil
}

// buildDirEntry builds one DirEntry for base (a file's base name inside dir), already resolved to
// lang. See TOCDir's doc comment for the Error/Partial mutual-exclusion invariant this function
// upholds.
func buildDirEntry(dir, base, lang string) DirEntry {
	entry := DirEntry{Name: base, Language: lang}

	strategy, ok := StrategyFor(lang)
	if !ok {
		entry.Error = quarryengine.ErrLanguageUnsupported.Error()
		return entry
	}

	path := filepath.Join(dir, base)
	src, err := os.ReadFile(path)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	if !utf8.Valid(src) {
		entry.Error = fmt.Sprintf("toc: %s: not valid UTF-8", path)
		return entry
	}

	err = treesitter.WithTree(lang, src, func(root *ts.Node, partial bool) error {
		entry.Partial = partial
		entry.Package = strategy.Package(root, src)
		entry.Header = FirstParagraph(strategy.Header(root, src))
		if isTest, known := strategy.TestFile(base); known {
			entry.Test = &isTest
		}
		if generated, known := strategy.Generated(root, src); known {
			entry.Generated = &generated
		}
		return nil
	})
	if err != nil {
		entry.Error = err.Error()
	}
	return entry
}
