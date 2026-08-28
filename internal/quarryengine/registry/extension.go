// extension.go adds a file-extension language map to package registry, alongside — never
// replacing — the marker-based DetectLanguage in detect.go. DetectLanguage matches marker files
// against a *directory* with a fixed precedence order, which resolves a .ts file inside a Go
// module to "go" — correct for the LSP verbs, wrong for a path-scoped lookup. The extension map
// answers a different question: what language does *this one file's extension* name, with no
// directory context at all. The toc verbs need that second question answered, so both live here
// side by side.

package registry

import (
	"sort"
	"strings"
)

// extensionLanguages maps a lowercase, dot-prefixed file extension to the canonical language
// name it resolves to. It is the one definition of the extension set; LanguageForExtension,
// ExtensionsForLanguage, and ExtensionLanguages are all views over this single map.
var extensionLanguages = map[string]string{
	".go":  "go",
	".py":  "python",
	".cs":  "csharp",
	".ts":  "typescript",
	".tsx": "typescript",
	".rs":  "rust",
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
