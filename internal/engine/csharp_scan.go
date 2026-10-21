// csharp_scan.go resolves C# units, which are namespaces and so have no directory to look up.
// A scan of every .cs file under the root finds the namespaces and symbols a C# glyph can name,
// and the self-glyph answer lists the directories those symbols sit in.
// A scan is a local of one exported call and dies with it.

package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// csharpGlobalUnit is the unit of a declaration outside any namespace.
const csharpGlobalUnit = "global"

// namespaceDeclarer is implemented by a strategy whose language spells units by declaration.
type namespaceDeclarer interface {
	// Namespaces returns every namespace the file declares and every prefix of it.
	Namespaces(root *ts.Node, src []byte) []string
}

// csharpScan is everything one walk over the tree's .cs files yields.
type csharpScan struct {
	// symbols holds each unit's symbols ordered by file, then start line.
	symbols map[string][]Symbol
	// declared holds every namespace some file declares or is a prefix of, plus "global" when any .cs file exists.
	declared map[string]struct{}
	// dirFiles holds, per unit, the base names of the files with a symbol in it, keyed by their repository-relative directory.
	dirFiles map[string]map[string][]string
}

// declares reports whether some file declares unit.
func (s *csharpScan) declares(unit string) bool {
	_, ok := s.declared[unit]
	return ok
}

// scanCSharp walks the tree from the root and collects the namespaces and symbols of its .cs files.
// It applies the ignore rules walkDir applies and never follows a symlink.
// Every .cs file is requested through m, so a file is parsed at most once per call, and a file that declares no namespace is read like any other.
// ig must carry the root's own patterns and nothing below them, as newUnitMemo's does.
func (r *Repo) scanCSharp(ig *ignoreSet, m *fileMemo) (*csharpScan, error) {
	scan := &csharpScan{
		symbols:  make(map[string][]Symbol),
		declared: make(map[string]struct{}),
		dirFiles: make(map[string]map[string][]string),
	}
	if err := r.scanCSharpEntries(".", ig, m, scan); err != nil {
		return nil, err
	}
	for _, symbols := range scan.symbols {
		sort.SliceStable(symbols, func(i, j int) bool {
			if symbols[i].File != symbols[j].File {
				return symbols[i].File < symbols[j].File
			}
			return symbols[i].Start < symbols[j].Start
		})
	}
	return scan, nil
}

// scanCSharpDir scans the subdirectory dirRel, extending ig with its patterns for the length of the scan.
func (r *Repo) scanCSharpDir(dirRel string, ig *ignoreSet, m *fileMemo, scan *csharpScan) error {
	n, err := ig.extend(dirRel)
	if err != nil {
		return fmt.Errorf("engine: read .gitignore for %q: %w", dirRel, err)
	}
	defer ig.trim(n)
	return r.scanCSharpEntries(dirRel, ig, m, scan)
}

// scanCSharpEntries scans the entries of dirRel, whose ignore patterns ig already carries.
func (r *Repo) scanCSharpEntries(dirRel string, ig *ignoreSet, m *fileMemo, scan *csharpScan) error {
	osEntries, err := os.ReadDir(r.absDir(dirRel))
	if err != nil {
		return fmt.Errorf("engine: read dir %q: %w", dirRel, err)
	}
	for _, entry := range osEntries {
		childRel := joinRel(dirRel, entry.Name())
		if ig.match(childRel, entry.IsDir()) || entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		if entry.IsDir() {
			if err := r.scanCSharpDir(childRel, ig, m, scan); err != nil {
				return err
			}
			continue
		}
		if lang, ok := LanguageForExtension(filepath.Ext(entry.Name())); !ok || lang != "csharp" {
			continue
		}
		scan.add(dirRel, entry.Name(), m.record(dirRel, entry.Name(), true))
	}
	return nil
}

// add records the file base inside dirRel, as read into rec.
func (s *csharpScan) add(dirRel, base string, rec *fileRecord) {
	s.declared[csharpGlobalUnit] = struct{}{}
	for _, namespace := range rec.namespaces {
		s.declared[namespace] = struct{}{}
	}
	listed := make(map[string]struct{})
	for _, sym := range stampSymbols(rec.symbols, "", joinRel(dirRel, base)) {
		unit := sym.Glyph.Unit
		s.symbols[unit] = append(s.symbols[unit], sym)
		if _, done := listed[unit]; done {
			continue
		}
		listed[unit] = struct{}{}
		if s.dirFiles[unit] == nil {
			s.dirFiles[unit] = make(map[string][]string)
		}
		s.dirFiles[unit][dirRel] = append(s.dirFiles[unit][dirRel], base)
	}
}

// csharp returns the call's C# scan, building it on the first call.
// A call with no C# target never calls it.
func (m *unitMemo) csharp() (*csharpScan, error) {
	if m.scan != nil {
		return m.scan, nil
	}
	m.scans++
	scan, err := m.repo.scanCSharp(m.ig, m.files)
	if err != nil {
		return nil, err
	}
	m.scan = scan
	return scan, nil
}

// csharpSelfTarget answers a C# self glyph: found exactly when some file declares unit.
// A found answer carries one listing per directory holding a file with a symbol in unit, in path order.
// Each listing is that directory's depth-0, symbols-off answer reduced to those files and without subdirectories.
// A declared unit with no such file carries an empty list.
func (r *Repo) csharpSelfTarget(unit string, m *unitMemo) (ResolveResult, error) {
	scan, err := m.csharp()
	if err != nil {
		return ResolveResult{}, err
	}
	if !scan.declares(unit) {
		return ResolveResult{Target: unit, Status: StatusNotFound}, nil
	}

	dirs := make([]string, 0, len(scan.dirFiles[unit]))
	for dirRel := range scan.dirFiles[unit] {
		dirs = append(dirs, dirRel)
	}
	sort.Strings(dirs)

	symbolsOff := false
	listings := make([]DirAnswer, 0, len(dirs))
	for _, dirRel := range dirs {
		answer, err := r.toc(dirRel, TOCOptions{Depth: 0, Symbols: &symbolsOff}, m.files)
		if err != nil {
			return ResolveResult{}, err
		}
		keep := make(map[string]struct{}, len(scan.dirFiles[unit][dirRel]))
		for _, base := range scan.dirFiles[unit][dirRel] {
			keep[base] = struct{}{}
		}
		files := make([]FileEntry, 0, len(keep))
		for _, entry := range answer.Files {
			if _, ok := keep[entry.Name]; ok {
				files = append(files, entry)
			}
		}
		answer.Files = files
		answer.Dirs = nil
		listings = append(listings, answer)
	}
	return ResolveResult{Target: unit, Status: StatusFound, Listings: &listings}, nil
}
