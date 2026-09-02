#!/bin/sh
# Zero-argument entry point: runs the annex ladder (quarry as mechanical pre-processing, injected into
# the prompt) end to end. See run.sh for what that includes.
exec "$(dirname "$0")/run.sh" annex
