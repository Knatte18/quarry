// Package repopath discovers a repository root and converts a caller-supplied target into a
// clean, forward-slash, repository-relative path. Its callers
// format their own user-facing sentences from the engine's target sentinels rather than
// propagating this package's error strings, which are namespaced to this package and never
// user-visible on their own.
package repopath
