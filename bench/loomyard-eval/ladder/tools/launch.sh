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

pointer="/home/knatte/Code/quarry/wts/quarry/.scratch/ladder-sessions/.current"
launcher="/home/knatte/Code/quarry/wts/quarry/bench/loomyard-eval/ladder/launch-session.sh"

if [ ! -f "$pointer" ]; then
    echo "launch.sh: no current session recorded at $pointer -- ask for a session to be prepared first" >&2
    exit 1
fi

exec "$launcher" "$(cat "$pointer")"
