// toc.go implements the engine package's one entry point, Repo.TOC. It validates the target
// through resolveTarget, builds a fresh ignoreSet extended along the chain from the repository
// root down to the target's parent, and dispatches to walkDir for a directory target or to the
// enclosing directory's own listing for a file target.

package engine

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// TOC answers a table-of-contents query for target, a repository-relative path (or "" / "." for
// the repository root), per opts.
//
// target is validated through resolveTarget: an absolute or root-escaping target returns
// ErrTargetOutsideRepo, and a missing target returns ErrTargetNotFound. TOC then builds a fresh
// ignoreSet for the repository root and extends it along the chain from the root down to the
// target's parent — never the target's own directory, which walkDir (for a directory target) or
// this function's own file-target path extends itself on entry.
//
// A directory target is answered by walkDir. opts.Depth of 0 fills the target's own files and
// lists its direct subdirectories as identity-plus-doc answers; N fills the files of
// subdirectories N levels down, each level's leaf dirs again identity-plus-doc; DepthAll recurses
// to the bottom.
//
// A file target — including a target that is itself a symlink — is answered as the enclosing
// directory's dir, package, language and doc, with files holding exactly that one entry and no
// dirs. Those four facts are the directory's, so a file target reads every file in the enclosing
// directory exactly as a directory target does: parsing the target alone and emitting its own
// clause as the directory's package would be wrong whenever the directory holds a package-clause
// deviation, which is the very case the tie-break exists for. opts.Depth is ignored for a file
// target — there is nothing below a file to fill — and a non-zero depth with a file target is not
// an error.
//
// opts.Symbols nil means true for a file target and false for a directory target; a non-nil value
// wins for every file entry at every depth.
//
// TOC builds a fresh record memo, so each file is parsed at most once per call.
func (r *Repo) TOC(target string, opts TOCOptions) (DirAnswer, error) {
	return r.toc(target, opts, newFileMemo(r, false))
}

// toc is TOC's memo-aware worker: it answers exactly as TOC does, reading and parsing each file
// through m. Resolve's self path calls it with its own call-wide memo so a file already built
// earlier in the call is not parsed again.
func (r *Repo) toc(target string, opts TOCOptions, m *fileMemo) (DirAnswer, error) {
	rel, info, err := r.resolveTarget(target)
	if err != nil {
		return DirAnswer{}, err
	}

	walkTarget := rel
	if !info.IsDir() {
		walkTarget, _ = splitDirBase(rel)
	}

	ig := newIgnoreSet(r.root)
	for _, ancestor := range ancestorChain(walkTarget) {
		if _, err := ig.extend(ancestor); err != nil {
			return DirAnswer{}, fmt.Errorf("engine: read .gitignore for %q: %w", ancestor, err)
		}
	}

	if info.IsDir() {
		wantSymbols := opts.Symbols != nil && *opts.Symbols
		return r.walkDir(rel, ig, opts.Depth, wantSymbols, false, m)
	}

	_, base := splitDirBase(rel)
	wantSymbols := opts.Symbols == nil || *opts.Symbols
	return r.fileTargetAnswer(walkTarget, base, ig, wantSymbols, m)
}

// splitDirBase splits a repository-relative, forward-slash path rel into its enclosing directory
// (or "." when rel names a file directly under the repository root) and its base name.
func splitDirBase(rel string) (dir, base string) {
	idx := strings.LastIndexByte(rel, '/')
	if idx < 0 {
		return ".", rel
	}
	return rel[:idx], rel[idx+1:]
}

// ancestorChain returns every proper ancestor directory of dir, from the repository root down to
// dir's own parent, exclusive of dir itself: dir extends its own patterns on entry to walkDir or
// fileTargetAnswer, so appending it here as well would apply it twice.
func ancestorChain(dir string) []string {
	if dir == "." {
		return nil
	}
	segments := strings.Split(dir, "/")
	chain := []string{"."}
	cur := "."
	for i := 0; i < len(segments)-1; i++ {
		if cur == "." {
			cur = segments[i]
		} else {
			cur = cur + "/" + segments[i]
		}
		chain = append(chain, cur)
	}
	return chain
}

// fileTargetAnswer builds the DirAnswer for a file target: dirRel's own package, language and doc
// — computed by reading every file in dirRel, exactly as a directory target would — with Files
// holding exactly the one entry named targetBase and no Dirs.
//
// targetBase is included in the answer even when the directory's own .gitignore would otherwise
// exclude it: resolveTarget's validation deliberately does not consult the ignore set, since the
// filter exists so a listing is not noise, not to make an explicitly named file unaddressable. The
// explicitly named target joins the package tie-break even when gitignored, because it is kept in
// the entries the vote reads; every other gitignored file stays out of the vote, matching walkDir.
func (r *Repo) fileTargetAnswer(dirRel, targetBase string, ig *ignoreSet, wantSymbols bool, m *fileMemo) (DirAnswer, error) {
	n, err := ig.extend(dirRel)
	if err != nil {
		return DirAnswer{}, fmt.Errorf("engine: read .gitignore for %q: %w", dirRel, err)
	}
	defer ig.trim(n)

	osEntries, err := os.ReadDir(r.absDir(dirRel))
	if err != nil {
		return DirAnswer{}, fmt.Errorf("engine: read dir %q: %w", dirRel, err)
	}

	var fileEntries []os.DirEntry
	var targetEntry os.DirEntry
	targetIgnored := false
	for _, entry := range osEntries {
		if entry.IsDir() {
			continue
		}
		isTarget := entry.Name() == targetBase
		if isTarget {
			targetEntry = entry
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			// A symlink never votes in the tie-break and is never parsed; if it is the target
			// itself it is handled below as a name-only entry.
			continue
		}
		childRel := joinRel(dirRel, entry.Name())
		ignored := ig.match(childRel, false)
		if isTarget {
			targetIgnored = ignored
		}
		if !isTarget && ignored {
			continue
		}
		fileEntries = append(fileEntries, entry)
	}
	if targetEntry == nil {
		return DirAnswer{}, fmt.Errorf("engine: target %q no longer exists in directory %q", targetBase, dirRel)
	}

	bases := make([]string, 0, len(fileEntries))
	for _, entry := range fileEntries {
		bases = append(bases, entry.Name())
	}
	recs := m.dirRecords(dirRel, bases, func(base string) bool { return base == targetBase && wantSymbols })
	v := m.dirVote(dirRel, recs, !targetIgnored)

	answer := DirAnswer{Dir: dirRel}
	if v.pkg != "" {
		answer.Package = v.pkg
		answer.Language = v.lang
	}

	docs := make(map[string]string)
	for base, rec := range recs {
		if rec.packageDoc != "" {
			docs[base] = rec.packageDoc
		}
	}
	var targetFileEntry FileEntry
	if targetEntry.Type()&fs.ModeSymlink == 0 {
		targetFileEntry = r.fileEntry(dirRel, targetBase, recs[targetBase], v, wantSymbols, make(map[string]bool))
	} else {
		targetFileEntry = FileEntry{Name: targetBase}
	}

	answer.Doc = r.dirDoc(v.clauses, docs, v.pkg)
	answer.Files = []FileEntry{targetFileEntry}
	return answer, nil
}
