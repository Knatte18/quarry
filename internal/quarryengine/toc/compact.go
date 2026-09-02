// compact.go renders a FileTOC or DirTOC as the compact text form: the same map, without the JSON
// envelope, without the per-entry keys that repeat on every line, and with prose cut to one sentence.
//
// Measured on Loomyard at the ladder pin (2026-09-02): the JSON form of a 25-file directory listing is
// 10.9 KB, of which 69% is header-comment prose and 22% is repeated keys/booleans; the compact form is
// 3.1 KB. A 13-symbol file's JSON toc is 5.0 KB, of which 43% is docstrings and 31% keys, names, and
// numbers; the compact form is 2.6 KB, or 1.2 KB with docstrings off. The compact form exists so a caller that is going to read
// the map, not parse it -- an agent, or a prompt pre-processor -- pays for the map and nothing else.
//
// Every line number is the same 1-based inclusive number the JSON form carries, so a "start-end" range
// on a compact line can be read from the source file directly.

package toc

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CompactFile renders f as compact text. path is the file as the caller wrote it and is used verbatim
// on the header line. The header line is "# <path> (package <pkg>): <first sentence of header>" with
// "[partial]" appended when the parse was lossy; each symbol is "<start>-<end>: <signature>", followed
// by " -- <docstring>" when the symbol has one (the docstring is whatever the caller's DocSentences
// option left, collapsed to one line and cut at leadMaxRunes). A file with no symbols says so on its
// header line.
func CompactFile(path string, f FileTOC) string {
	var b strings.Builder
	b.WriteString("# " + path)
	if f.Package != "" {
		b.WriteString(" (package " + f.Package + ")")
	}
	if f.Partial {
		b.WriteString(" [partial]")
	}
	if head := lead(f.Header); head != "" {
		b.WriteString(": " + head)
	}
	if len(f.Symbols) == 0 {
		b.WriteString(" (no symbols)")
	}
	for _, s := range f.Symbols {
		fmt.Fprintf(&b, "\n%d-%d: %s", s.Start, s.End, oneLine(s.Signature))
		if doc := lead(s.Docstring); doc != "" {
			b.WriteString(" -- " + doc)
		}
	}
	return b.String()
}

// CompactDir renders d as compact text. dirArg is the directory as the caller wrote it; each file line
// is filepath.Join(dirArg, entry.Name), exactly the "path" the JSON form composes, so a line's path
// pastes straight into a following "toc file" call. The header line names the directory, the dominant
// package (the most common one among the files), and the file count. Each file line is
// "<path>[ [test]][ [generated]][ [partial]][ (package <pkg>)]: <first sentence of header>", where the
// package is shown only when it differs from the dominant one (a _test package, typically) and the
// colon and sentence are omitted for a file with no header. A file that could not be parsed is
// "<path>: error: <message>". A trailing "dirs: a, b" line lists direct subdirectories when there are
// any.
func CompactDir(dirArg string, d DirTOC) string {
	dominant := dominantPackage(d.Files)

	var b strings.Builder
	b.WriteString("# " + dirArg)
	if dominant != "" {
		b.WriteString(" (package " + dominant + ")")
	}
	fmt.Fprintf(&b, ", %d files", len(d.Files))

	for _, e := range d.Files {
		b.WriteString("\n" + filepath.Join(dirArg, e.Name))
		if e.Error != "" {
			b.WriteString(": error: " + oneLine(e.Error))
			continue
		}
		if e.Test != nil && *e.Test {
			b.WriteString(" [test]")
		}
		if e.Generated != nil && *e.Generated {
			b.WriteString(" [generated]")
		}
		if e.Partial {
			b.WriteString(" [partial]")
		}
		if e.Package != "" && e.Package != dominant {
			b.WriteString(" (package " + e.Package + ")")
		}
		if head := lead(e.Header); head != "" {
			b.WriteString(": " + head)
		}
	}
	if len(d.Dirs) > 0 {
		b.WriteString("\ndirs: " + strings.Join(d.Dirs, ", "))
	}
	return b.String()
}

// dominantPackage returns the most common non-empty Package among files, ties broken by first
// occurrence, or "" when no file declares one.
func dominantPackage(files []DirEntry) string {
	counts := map[string]int{}
	var order []string
	for _, e := range files {
		if e.Package == "" || e.Error != "" {
			continue
		}
		if counts[e.Package] == 0 {
			order = append(order, e.Package)
		}
		counts[e.Package]++
	}
	best := ""
	for _, pkg := range order {
		if counts[pkg] > counts[best] {
			best = pkg
		}
	}
	return best
}

// leadMaxRunes caps the prose a compact line carries. Loomyard's first sentences run to 200+
// characters; the first 120 are enough to decide whether to open the file or symbol, which is the only
// decision the compact form exists to support.
const leadMaxRunes = 120

// lead is the first sentence of s, collapsed to one line and cut at leadMaxRunes on a word boundary
// with a trailing ellipsis.
func lead(s string) string {
	line := oneLine(FirstSentences(s, 1))
	runes := []rune(line)
	if len(runes) <= leadMaxRunes {
		return line
	}
	cut := string(runes[:leadMaxRunes])
	if i := strings.LastIndex(cut, " "); i > leadMaxRunes/2 {
		cut = cut[:i]
	}
	return cut + "…"
}

// oneLine collapses every run of whitespace in s (newlines included) to a single space and trims it.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
