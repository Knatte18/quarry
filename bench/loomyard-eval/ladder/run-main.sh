#!/bin/sh
# Zero-argument entry point: runs the main ladder end to end. See run.sh for what that includes.
exec "$(dirname "$0")/run.sh" main
