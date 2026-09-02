#!/bin/sh
# run.sh is the one command that runs a whole ladder file: preflight, every run session (warm and cold),
# the scoring session, cold-cell finalisation, summarize, provenance.json, and a per-cell table. Nothing
# else needs to be typed, copied, or remembered. It is resumable: re-running it against the same results
# root skips every step that root already records.
#
# Usage:
#   bench/loomyard-eval/ladder/run-<main|followup|task05|toc|annex|compact>.sh (zero arguments)
#   bench/loomyard-eval/ladder/run.sh <ladder> [results-root]      (any ladder yaml / chosen root)
#
# <ladder> is a shortname -- main, followup, task05, toc, annex, compact -- or a path to any ladder yaml.
# [results-root] defaults to results/<today>[-<shortname>] under this directory; pass one explicitly to
# resume an older root or to start a second root on the same day.
#
# Machine-specific configuration is exactly one value, the Loomyard checkout, read from
# $LADDER_LOOMYARD_REPO -- or, when that is unset, from <repo-root>/.scratch/ladder.env (gitignored),
# a one-line file:  LADDER_LOOMYARD_REPO=/path/to/loomyard
#
# Everything actually launched is still a live, watchable claude session inside the "ladder-run" tmux
# session (`tmux attach -t ladder-run`); this script only removes the operator from the sequencing.
# The two permission probes (`ladderbench prepare-session --probe ...`) are not part of it -- they are
# a one-off per harness change, not per matrix run.

set -eu

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
ladder_dir="$repo_root/bench/loomyard-eval/ladder"

usage() {
    echo "usage: $0 <main|followup|task05|toc|annex|compact|path/to/ladder.yaml> [results-root]" >&2
    exit 2
}

[ $# -ge 1 ] && [ $# -le 2 ] || usage

case "$1" in
    main)     ladder="$ladder_dir/ladder.yaml" ;;
    followup) ladder="$ladder_dir/ladder-followup.yaml" ;;
    task05)   ladder="$ladder_dir/ladder-task05.yaml" ;;
    toc)      ladder="$ladder_dir/ladder-toc.yaml" ;;
    annex)    ladder="$ladder_dir/ladder-annex.yaml" ;;
    compact)  ladder="$ladder_dir/ladder-compact.yaml" ;;
    -h|--help) usage ;;
    *)        ladder="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")" ;;
esac
[ -f "$ladder" ] || { echo "run.sh: no such ladder file: $ladder" >&2; exit 1; }

results_root=""
if [ $# -eq 2 ]; then
    results_root="$2"
    case "$results_root" in
        /*) ;;
        *)  results_root="$PWD/$results_root" ;;
    esac
fi

fail=0
need() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "run.sh: missing prerequisite on PATH: $1 -- $2" >&2
        fail=1
    fi
}
need go     "builds ladderbench and quarry-mcp"
need cc     "quarry-mcp needs CGO_ENABLED=1 (tree-sitter grammars); install a C toolchain"
need claude "every run and scoring session is a live Claude Code session"
need tmux   "each session is launched detached inside tmux session 'ladder-run'"
need git    "worktree lifecycle"

env_file="$repo_root/.scratch/ladder.env"
if [ -z "${LADDER_LOOMYARD_REPO:-}" ] && [ -f "$env_file" ]; then
    # shellcheck disable=SC1090
    . "$env_file"
    export LADDER_LOOMYARD_REPO
fi
if [ -z "${LADDER_LOOMYARD_REPO:-}" ]; then
    echo "run.sh: LADDER_LOOMYARD_REPO is not set. Either export it, or create $env_file containing:" >&2
    echo "    LADDER_LOOMYARD_REPO=/path/to/your/loomyard/checkout" >&2
    fail=1
elif ! git -C "$LADDER_LOOMYARD_REPO" rev-parse --show-toplevel >/dev/null 2>&1; then
    echo "run.sh: LADDER_LOOMYARD_REPO=$LADDER_LOOMYARD_REPO is not a git checkout" >&2
    fail=1
fi

if tmux has-session -t ladder-run 2>/dev/null; then
    echo "run.sh: a tmux session named 'ladder-run' is already running -- another matrix is in progress, or a" >&2
    echo "        previous one was interrupted. Attach with 'tmux attach -t ladder-run' to inspect, or" >&2
    echo "        'tmux kill-session -t ladder-run' to discard it, then re-run." >&2
    fail=1
fi

[ "$fail" -eq 0 ] || exit 1

cd "$repo_root"
echo "== run.sh: ladder=$ladder"
echo "== run.sh: LADDER_LOOMYARD_REPO=$LADDER_LOOMYARD_REPO"
echo "== run.sh: watch sessions with: tmux attach -t ladder-run"
if [ -n "$(git -C "$repo_root" status --porcelain)" ]; then
    echo "== run.sh: NOTE: the quarry working tree is dirty. quarry-mcp is rebuilt from it before every repetition,"
    echo "           so do not edit quarry source while this runs; provenance.json records each build's hash."
fi

if [ -n "$results_root" ]; then
    exec go run ./bench/loomyard-eval/ladder/tools/runmatrix --all --ladder "$ladder" --results-root "$results_root"
else
    exec go run ./bench/loomyard-eval/ladder/tools/runmatrix --all --ladder "$ladder"
fi
