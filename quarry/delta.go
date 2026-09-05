// delta.go holds the two delta methods — the pure Delta and the git convenience DeltaGit — and the
// one new type this facade declares rather than aliases: GitDeltaAnswer. Every other type in this
// package is an alias for an engine type; GitDeltaAnswer is not, because it carries the two revision
// strings DeltaGit computed its answer from, and the engine's own DeltaAnswer must stay ignorant of
// git — see GitDeltaAnswer's own doc comment for why.

package quarry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Knatte18/quarry/internal/gitsrc"
)

// GitDeltaAnswer wraps a DeltaAnswer with the two revision strings DeltaGit computed it from. The
// core's own DeltaAnswer carries no revision information at all: the engine knows nothing about git,
// and a field it can never populate would be a lie in its own type, so the echo lives here, exactly
// where the revisions are known.
//
// From and To are declared before the embedded DeltaAnswer so the emitted key order puts them
// first. To is a pointer so an absent revision — the after side is the working tree — marshals as
// an explicit JSON null rather than being omitted: the key's presence, with a null value, is itself
// the statement that the after side is the working tree.
type GitDeltaAnswer struct {
	// From is the before-side revision string, exactly as the caller gave it.
	From string `json:"from"`
	// To is the after-side revision string, exactly as the caller gave it, or nil for the working
	// tree.
	To *string `json:"to"`
	DeltaAnswer
}

// Delta compares two versions of a batch of files, exactly as the engine's own Delta does: it
// returns the engine's own answer and the engine's own error unchanged — no filtering, no
// re-shaping, no defaulting, exactly as the existing three query methods on this facade do.
func (r *Repo) Delta(entries []DeltaEntry) (DeltaAnswer, error) {
	return r.engine.Delta(entries)
}

// DeltaGit is the caller-facing convenience over Delta: it drives the git layer for paths and
// bytes, derives each side's glyph unit, assembles the batch, and calls Delta. target is already a
// repository-relative path when it arrives — this method performs no path arithmetic on it, and
// hands it to the git layer as the pathspec unchanged.
//
// to == "" means the after side is the working tree, selecting the one-revision form of the diff
// and the working-tree read path throughout; a non-empty to selects the two-revision form and reads
// both sides from git.
//
// DeltaGit opens the git layer against r.root, which is what performs the top-level verification
// (see gitsrc.Open's own doc comment), then verifies each supplied revision before any diff runs.
// Every error from the git layer is returned unchanged, so the facade's own aliased sentinels
// (ErrNotARepository, ErrRootNotTopLevel, ErrUnknownRevision) and typed errors (RootNotTopLevelError,
// UnknownRevisionError) survive the call untouched.
func (r *Repo) DeltaGit(from, to, target string) (GitDeltaAnswer, error) {
	gr, err := gitsrc.Open(r.root)
	if err != nil {
		return GitDeltaAnswer{}, err
	}
	if err := gr.VerifyRevision(from); err != nil {
		return GitDeltaAnswer{}, err
	}
	if to != "" {
		if err := gr.VerifyRevision(to); err != nil {
			return GitDeltaAnswer{}, err
		}
	}

	changes, err := gr.ChangedPaths(from, to, target)
	if err != nil {
		return GitDeltaAnswer{}, err
	}

	entries := make([]DeltaEntry, 0, len(changes))
	for _, c := range changes {
		entries = append(entries, r.deltaEntryForChange(gr, from, to, c))
	}

	// The untracked paths are added to the batch only when the after side is the working tree: a
	// revision-to-revision diff has no working tree to enumerate untracked files against.
	if to == "" {
		untracked, err := gr.UntrackedPaths(target)
		if err != nil {
			return GitDeltaAnswer{}, err
		}
		for _, path := range untracked {
			entries = append(entries, r.deltaEntryForUntracked(path))
		}
	}

	answer, err := r.Delta(entries)
	if err != nil {
		return GitDeltaAnswer{}, err
	}

	var toPtr *string
	if to != "" {
		toPtr = &to
	}
	return GitDeltaAnswer{From: from, To: toPtr, DeltaAnswer: answer}, nil
}

// deltaEntryForChange maps one gitsrc.Change to its DeltaEntry, following the total status-letter
// table: "A" gives nil before bytes and the after bytes; "M" gives both sides' bytes; "D" gives the
// before bytes and a nil after; "T" (a typechange, file <-> symlink) gives both sides' bytes, where a
// side that is now a symlink yields its link text rather than the bytes it points to — link text is
// not parseable source and therefore contributes no symbols on that side, exactly as any other
// unparseable content does; "U" (unmerged) is refused before extraction, since a conflicted path's
// working-tree content is conflict markers and extracting it as source would be a silent lie; and
// any other letter is refused with a message naming the letter verbatim. "R" and "C" cannot appear
// because rename and copy detection are disabled, but this catch-all row covers them anyway rather
// than relying on that argument.
//
// A read failure on either side — a blob read failing, or a working-tree file being unreadable
// during assembly — is a pre-set refusal naming the side and the underlying error, never a failure
// of the whole call: a disk-read failure during assembly is neither a failed git command nor grounds
// to fail a batch, so it is one entry's problem reported as that entry's refusal.
func (r *Repo) deltaEntryForChange(gr *gitsrc.Repo, from, to string, c gitsrc.Change) DeltaEntry {
	switch c.Status {
	case "A":
		after, err := r.readAfterSide(gr, to, c.Path)
		if err != nil {
			return readRefusal(c.Path, "after", err)
		}
		return DeltaEntry{Path: c.Path, After: after}
	case "M", "T":
		before, err := gr.ReadBlob(from, c.Path)
		if err != nil {
			return readRefusal(c.Path, "before", err)
		}
		after, err := r.readAfterSide(gr, to, c.Path)
		if err != nil {
			return readRefusal(c.Path, "after", err)
		}
		return DeltaEntry{Path: c.Path, Before: before, After: after}
	case "D":
		before, err := gr.ReadBlob(from, c.Path)
		if err != nil {
			return readRefusal(c.Path, "before", err)
		}
		return DeltaEntry{Path: c.Path, Before: before}
	case "U":
		return DeltaEntry{Path: c.Path, Refusal: "unmerged path"}
	default:
		return DeltaEntry{Path: c.Path, Refusal: fmt.Sprintf("%s: unrecognised git status %q", c.Path, c.Status)}
	}
}

// deltaEntryForUntracked builds the DeltaEntry for one untracked, on-disk-only path: nil before
// bytes and its on-disk bytes after, so it reads as DispositionAdded exactly like a staged new file.
func (r *Repo) deltaEntryForUntracked(path string) DeltaEntry {
	after, err := r.readWorkingTreeBytes(path)
	if err != nil {
		return readRefusal(path, "after", err)
	}
	return DeltaEntry{Path: path, After: after}
}

// readAfterSide reads the after side of path: from disk when to == "" (the after side is the
// working tree), otherwise from the git layer at revision to.
func (r *Repo) readAfterSide(gr *gitsrc.Repo, to, path string) ([]byte, error) {
	if to == "" {
		return r.readWorkingTreeBytes(path)
	}
	return gr.ReadBlob(to, path)
}

// readWorkingTreeBytes reads path, repository-relative, from disk under r.root. A symlink yields its
// link text via os.Readlink, matching the bytes a git blob holds for the same symlink (git's blob
// content for a symlink is its link target text, never the bytes of whatever it points to), so the
// two read paths this file uses agree on what a symlink's "content" is.
func (r *Repo) readWorkingTreeBytes(path string) ([]byte, error) {
	full := filepath.Join(r.root, filepath.FromSlash(path))
	info, err := os.Lstat(full)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(full)
		if err != nil {
			return nil, err
		}
		return []byte(target), nil
	}
	return os.ReadFile(full)
}

// readRefusal builds the DeltaEntry a read failure on side ("before" or "after") produces for path:
// a pre-set Refusal naming both, so the core skips the entry entirely rather than failing the batch.
func readRefusal(path, side string, err error) DeltaEntry {
	return DeltaEntry{Path: path, Refusal: fmt.Sprintf("%s: %s side: %v", path, side, err)}
}
