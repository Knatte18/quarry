// Package gitsrc is read-only git plumbing: it opens a repository root, verifies it and a
// revision, lists changed paths between two revisions or against the working tree, lists
// untracked paths, reads one path's bytes at a revision, and enumerates a directory's immediate
// children on either side of a diff.
//
// It runs no command that writes: no checkout, no stash, no index write, no config write. It
// returns paths, bytes and errors only, and knows nothing about symbols, clauses or units — that
// derivation happens above this package, in the engine's own parse seam, which is the engine's job
// and never this package's.
//
// It never uses git's own rename or copy detection. Detection is a similarity threshold, and the
// two-tier delta contract this package serves classifies a rename as a delete plus an add through
// its own table comparison instead; letting git's heuristic run would mean the delta's answer
// silently inherited the exact heuristic the two-tier design exists to replace.
package gitsrc
