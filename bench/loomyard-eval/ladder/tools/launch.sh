#!/bin/sh
# launch.sh is the zero-argument way to launch whichever ladder session was most recently prepared: no
# path to type or remember. It reads the scratch-directory path out of the pointer file at
# .scratch/ladder-sessions/.current and hands it to ../launch-session.sh, which does the actual cd +
# claude launch.
#
# The pointer file is not written by ladderbench itself -- it is plain operational bookkeeping, updated
# by whoever last ran `ladderbench prepare-session` for a probe, run, or scoring session, immediately
# after that command printed its launch line. Run this script every time a new session is ready.

set -eu

# Resolved from this script's own location, never a hardcoded machine-specific checkout path -- this
# script sits at <repo_root>/bench/loomyard-eval/ladder/tools/launch.sh.
script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/../../../.." && pwd)"

pointer="$repo_root/.scratch/ladder-sessions/.current"
launcher="$repo_root/bench/loomyard-eval/ladder/launch-session.sh"

if [ ! -f "$pointer" ]; then
    echo "launch.sh: no current session recorded at $pointer -- ask for a session to be prepared first" >&2
    exit 1
fi

exec "$launcher" "$(cat "$pointer")"
