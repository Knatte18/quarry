#!/bin/sh
# launch-session.sh cds into one ladder session's scratch directory and launches claude with this suite's
# fixed session flags, so the operator never hand-types --setting-sources/--mcp-config/--permission-mode
# for every probe, run, and scoring session. This is a convenience wrapper only: the flag set it launches
# mirrors internal/ladder/session.go's own LaunchCommand exactly (--mcp-config only when the scratch
# directory actually carries a server declaration, e.g. never for a blinded "none" config), plus
# --permission-mode manual, which every session in this suite launches with so a live permission prompt
# surfaces normally instead of being silently blocked by Claude Code's auto-mode classifier.
#
# The initial prompt (default "/ladder-run") is passed as claude's own positional prompt argument. This
# still launches a fully interactive session -- claude only runs headless under -p/--print, which this
# script never passes -- it just submits that text as the session's first message automatically, so the
# operator never has to type it. The session is exactly as live and killable as one launched with no
# prompt at all: the operator is expected to watch it and can close the window at any point.
#
# Usage: launch-session.sh <scratch-dir> [initial-prompt]
# <scratch-dir> is exactly what `ladderbench prepare-session` printed on its own "scratch_dir: " line.
#
# tools/launch.sh is the zero-argument wrapper around this script for interactive use -- it reads
# whichever scratch directory was most recently prepared from a pointer file, so the operator never needs
# to know or type this path themselves.

set -eu

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
    echo "usage: $0 <scratch-dir> [initial-prompt]" >&2
    exit 1
fi

scratch_dir=$1
prompt=${2:-/ladder-run}

if [ ! -d "$scratch_dir" ]; then
    echo "launch-session.sh: $scratch_dir is not a directory -- run ladderbench prepare-session first" >&2
    exit 1
fi

cd "$scratch_dir"

if [ -f "$scratch_dir/.mcp.json" ]; then
    exec claude --setting-sources user,project --mcp-config "$scratch_dir/.mcp.json" --permission-mode manual "$prompt"
else
    exec claude --setting-sources user,project --permission-mode manual "$prompt"
fi
