# after/ — moved

The after-side goldens that used to live here are living test fixtures, not a frozen record:
`internal/cli/after_test.go` compares against them on every run and rewrites them under
`-update`. They now live in `internal/cli/testdata/`, next to the test that owns them, together
with their own `INDEX.md` — the before-to-after mapping, the per-invocation exit codes, and the
regeneration command.

The before-side files in this directory are unchanged and stay frozen.
