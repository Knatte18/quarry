# Abandoned root — no data, do not read numbers from here

Two attempts at the compact ladder were started here on 2026-09-02 and stopped before the first
repetition ingested. Nothing was measured; `provenance.json` is only the pre-run stamp `runmatrix`
writes when it creates a root, and carries no per-rep `server_hashes`.

Left in place rather than deleted because two pieces of live state remain and are load-bearing as
evidence: the `.session-active` lock (gitignored) still names the stopped session, and the stopped
dispatch's subagent transcript still sits under the `~/.claude/projects` directory that the old
`session_dir_template` prefix (`compact-…`) mangles to. `LocateTranscript` hard-errors on two metadata
files sharing a description, so re-running against this root under that prefix would fail ingest on the
first repetition.

The real run therefore uses a fresh prefix (`compact2-…`, see `ladder-compact.yaml`) and a fresh root,
`results/2026-09-02-compact2/`. Read that one.
