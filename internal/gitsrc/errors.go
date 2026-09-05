// errors.go declares the sentinels and typed errors gitsrc's operations return, and the identity
// contract that lets a caller match either shape against the same value: errors.Is against the
// sentinel and errors.As against the typed error both succeed against the same returned value,
// because each typed error's Unwrap returns its matching sentinel.

package gitsrc

import (
	"errors"
	"fmt"
)

// ErrNotARepository is returned by Open when root does not name a path inside any git repository.
var ErrNotARepository = errors.New("gitsrc: not a git repository")

// ErrRootNotTopLevel is the sentinel RootNotTopLevelError wraps. It is returned, wrapped, by Open
// when root names a path inside a repository whose top-level directory is elsewhere.
var ErrRootNotTopLevel = errors.New("gitsrc: root is not the repository top level")

// ErrUnknownRevision is the sentinel UnknownRevisionError wraps. It is returned, wrapped, by
// (*Repo).VerifyRevision when the supplied revision does not resolve to a commit.
var ErrUnknownRevision = errors.New("gitsrc: unknown revision")

// RootNotTopLevelError is returned by Open when root names a path inside a repository whose
// top-level directory is elsewhere. Root and TopLevel are both absolute paths, spelled here as
// fields rather than folded into a message only, so the command-line layer can spell its own
// user-facing sentence from the value rather than by parsing a message.
type RootNotTopLevelError struct {
	// Root is the path Open was called with.
	Root string
	// TopLevel is the repository top level git reported for Root.
	TopLevel string
}

// Error implements error, spelling a full sentence naming both Root and TopLevel.
func (e *RootNotTopLevelError) Error() string {
	return fmt.Sprintf("gitsrc: root %q is not the repository top level (top level is %q)", e.Root, e.TopLevel)
}

// Unwrap returns ErrRootNotTopLevel, so errors.Is(err, ErrRootNotTopLevel) succeeds against a
// *RootNotTopLevelError exactly as errors.As(err, &rootErr) does.
func (e *RootNotTopLevelError) Unwrap() error {
	return ErrRootNotTopLevel
}

// UnknownRevisionError is returned by (*Repo).VerifyRevision when Rev does not resolve to a
// commit. Rev is spelled exactly as the caller gave it, as a field rather than folded into a
// message only, so the command-line layer can spell its own user-facing sentence from the value
// rather than by parsing a message.
type UnknownRevisionError struct {
	// Rev is the revision VerifyRevision was called with, spelled exactly as given.
	Rev string
}

// Error implements error, spelling a full sentence naming Rev.
func (e *UnknownRevisionError) Error() string {
	return fmt.Sprintf("gitsrc: unknown revision %q", e.Rev)
}

// Unwrap returns ErrUnknownRevision, so errors.Is(err, ErrUnknownRevision) succeeds against a
// *UnknownRevisionError exactly as errors.As(err, &revErr) does.
func (e *UnknownRevisionError) Unwrap() error {
	return ErrUnknownRevision
}
