// Package repopath discovers a repository root and converts a caller-supplied target into a
// clean, forward-slash, repository-relative path. It has two callers, the CLI and the MCP server,
// and both format their own user-facing sentences from this package's exported sentinels rather
// than propagating this package's error strings, which are namespaced to this package and never
// user-visible on their own.
package repopath
