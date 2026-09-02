#!/bin/sh
# Zero-argument entry point: runs the compact-form ladder (toc as JSON vs compact text, as a tool and as
# an injected annex) end to end. See run.sh for what that includes.
exec "$(dirname "$0")/run.sh" compact
