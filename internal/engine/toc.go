// toc.go implements the two entry points the engine package exports: TOCFile and TOCDir. Both
// resolve a language, drive treesitter.WithTree, dispatch to the resolved language's registered
// Strategy, and apply the one post-processing rule that belongs to the entry point rather than to
// any strategy — first-paragraph header truncation (FirstParagraph). Putting that rule here rather
// than in the strategies gives it exactly one call site shared by both entry points.

package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// resolveLanguage resolves the canonical language name for path, honoring langOverride when it is
// non-empty. A non-empty langOverride wins outright regardless of path's extension — a mismatch is
// not an error here.
//
// An extension that maps to no language, or an override naming a language with no registered
// Strategy, both return a wrapped ErrLanguageUnsupported.
func resolveLanguage(path, langOverride string) (string, error) {
	lang := langOverride
	if lang == "" {
		resolved, ok := LanguageForExtension(filepath.Ext(path))
		if !ok {
			return "", fmt.Errorf("toc: %s: no language for extension %q: %w", path, filepath.Ext(path), ErrLanguageUnsupported)
		}
		lang = resolved
	}
	if _, ok := StrategyFor(lang); !ok {
		return "", fmt.Errorf("toc: %s: language %q has no toc strategy: %w", path, lang, ErrLanguageUnsupported)
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
// A symbol's Docstring is always the complete, untrimmed docstring — TOCFile applies no
// sentence-count policy of its own.
func TOCFile(path string, langOverride string) (FileTOC, error) {
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
		result = FileTOC{
			Language: lang,
			Package:  strategy.Package(root, src),
			Header:   FirstParagraph(strategy.Header(root, src)),
			Symbols:  strategy.Symbols(root, src),
			Partial:  partial,
		}
		return nil
	})
	if err != nil {
		return FileTOC{}, err
	}
	return result, nil
}

// TOCDir extracts a DirTOC from exactly one directory level of dir. It never recurses and never
// descends into a subdirectory — a subdirectory entry contributes its bare name to Dirs and nothing
// else; TOCDir never reads, stats, or lists what that subdirectory itself contains.
//
// For each remaining (non-directory) entry, the language is resolved from its extension; an
// extension mapping to no language skips the file entirely, so a Markdown or YAML file never
// appears. When langOverride is non-empty, the file listing is restricted to that language's
// extensions via ExtensionsForLanguage, rather than reinterpreting every other file under
// the override. langOverride has no effect on Dirs: a subdirectory name carries no language, so
// there is nothing for the override to restrict.
//
// The surviving file entries are sorted lexicographically by base filename with an explicit
// sort.Slice, never left in os.ReadDir's own order; Dirs is sorted the same way, independently, so
// the two lists never influence each other's order.
//
// Error and Partial are mutually exclusive by construction: Partial is only ever set on the route
// that actually parsed the file through treesitter.WithTree. An unreadable file and invalid UTF-8
// both set Error instead and leave Header and Partial unset — the file is still listed, never
// skipped.
//
// A directory containing no file with a supported extension returns a DirTOC whose Files is an
// empty, non-nil slice and a nil error — a true answer to "what code is in here", not a failure.
// Dirs follows the identical empty-non-nil convention when the directory holds no subdirectory.
//
// TOCDir imposes no file-size cap: parse cost is linear and the runtime enforces its own work
// budgets, so a pathological file surfaces as a slow parse or as Partial, never as a special-cased
// refusal.
func TOCDir(dir string, langOverride string) (DirTOC, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return DirTOC{}, fmt.Errorf("toc: read dir %s: %w", dir, err)
	}

	var allowedExts map[string]bool
	if langOverride != "" {
		exts := ExtensionsForLanguage(langOverride)
		allowedExts = make(map[string]bool, len(exts))
		for _, ext := range exts {
			allowedExts[ext] = true
		}
	}

	result := DirTOC{Files: []DirEntry{}, Dirs: []string{}}
	for _, entry := range entries {
		if entry.IsDir() {
			result.Dirs = append(result.Dirs, entry.Name())
			continue
		}
		base := entry.Name()
		lang, ok := LanguageForExtension(filepath.Ext(base))
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
	sort.Strings(result.Dirs)

	return result, nil
}

// buildDirEntry builds one DirEntry for base (a file's base name inside dir), already resolved to
// lang, a language from the extension table and therefore one with a registered Strategy. See
// TOCDir's doc comment for the Error/Partial mutual-exclusion invariant this function upholds.
func buildDirEntry(dir, base, lang string) DirEntry {
	entry := DirEntry{Name: base, Language: lang}
	strategy, _ := StrategyFor(lang)

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
