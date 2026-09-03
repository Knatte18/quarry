// extension.go is the file-extension → language map: what language does this one file's extension
// name, with no directory context at all. extensionLanguages is the one definition of the extension
// set; LanguageForExtension, ExtensionsForLanguage, and ExtensionLanguages are all views over that
// one map. The two header-rule lookup tables below it are a separate concern with their own map —
// see their own comment for why.

package engine

import (
	"sort"
	"strings"
)

// extensionLanguages maps a lowercase, dot-prefixed file extension to the canonical language
// name it resolves to.
var extensionLanguages = map[string]string{
	".go": "go",
}

// LanguageForExtension maps a file extension to its canonical language name. ext is lowercased
// and tolerates a missing leading dot, so both ".GO" and "go" resolve to "go". An unknown
// extension returns ("", false).
func LanguageForExtension(ext string) (string, bool) {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	lang, ok := extensionLanguages[ext]
	return lang, ok
}

// ExtensionsForLanguage returns the sorted extensions that map to lang, or nil if lang names no
// language in the extension map.
func ExtensionsForLanguage(lang string) []string {
	var exts []string
	for ext, l := range extensionLanguages {
		if l == lang {
			exts = append(exts, ext)
		}
	}
	sort.Strings(exts)
	return exts
}

// ExtensionLanguages returns the sorted canonical language names the extension map defines.
func ExtensionLanguages() []string {
	seen := make(map[string]bool, len(extensionLanguages))
	for _, lang := range extensionLanguages {
		seen[lang] = true
	}
	names := make([]string, 0, len(seen))
	for lang := range seen {
		names = append(names, lang)
	}
	sort.Strings(names)
	return names
}

// extensionHeaderRules and baseNameHeaderRules are deliberately separate from extensionLanguages: an
// entry here gives a file a header, never a language, and never symbols. A file can appear in one
// table, both, or neither — a Markdown file has a header rule but no language, a Go file has a
// language but its header comes from the Go strategy rather than from either table here, and a JSON
// file has neither.
//
// extensionHeaderRules is keyed by a lowercase, dot-prefixed extension.
var extensionHeaderRules = map[string]headerRule{
	".md":   markdownHeader,
	".html": htmlCommentHeader,
	".htm":  htmlCommentHeader,
	".css":  cssCommentHeader,
	".js":   scriptCommentHeader,
	".mjs":  scriptCommentHeader,
	".ts":   scriptCommentHeader,
	".yaml": hashBlockHeader,
	".yml":  hashBlockHeader,
	".toml": hashBlockHeader,
	".sh":   hashBlockHeader,
	".bash": hashBlockHeader,
	".zsh":  hashBlockHeader,
}

// baseNameHeaderRules is keyed by an exact file base name and is consulted only when
// filepath.Ext(base) returns the empty string. A sentinel key inside extensionHeaderRules (say,
// "" or ".") would conflate "this file has no extension" with "this file's extension is unknown",
// and a key that reads like an extension but is not one would be a lie; an extensionless file — a
// Makefile, a Dockerfile — is a real, distinct case, so it gets its own table keyed by the whole
// base name instead.
var baseNameHeaderRules = map[string]headerRule{
	"Makefile":   hashBlockHeader,
	"Dockerfile": hashBlockHeader,
}
