// Package toc is the toc orchestration layer: the per-language extraction strategies (Strategy,
// registered per language) and the TOCFile / TOCDir entry points that walk a parsed tree into the
// Symbol, FileTOC, and DirTOC results below, and the file-extension → language map they resolve
// through. It imports the engine root (github.com/Knatte18/quarry/internal/quarryengine) for the
// shared error sentinel and treesitter for the parse-and-release seam — nothing else.
//
// This package returns typed results and typed errors only. It never emits JSON, never decides an
// exit code, and never resolves a caller's cwd.
//
// # The sentence-boundary rule
//
// FirstSentences (sentences.go) trims an already delimiter-stripped docstring to its first N
// sentences. A sentence ends at a '.', '!', or '?' followed by whitespace or end-of-string, except
// when that terminator belongs to one of three excluded shapes: a known abbreviation from the closed
// list "e.g.", "i.e.", "cf.", "vs.", "etc.", "resp.", "approx." (matched case-insensitively), a
// single-letter initial (a '.' whose preceding token is one letter, e.g. "A."), or a terminator
// inside a backtick-quoted span. The abbreviation list is kept short and explicit — most notably
// "e.g." and "i.e." — because both are common in Go doc comments and, left unhandled, split a
// sentence in two.
package toc
