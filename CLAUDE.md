# quarry repo conventions

## Go only — no Python

Do not introduce Python scripts or tooling anywhere in this repo, including
disposable bench/test harness code under `bench/`. Use Go.

Reason: on the user's Windows 11 work machine, PowerShell 5 takes ~1s to
start and loading Python on top of that adds another ~0.8s — every single
Python invocation (e.g. a pytest verify command run per batch, or any script
called repeatedly) pays ~1.8s of pure interpreter/shell startup tax before
doing any real work. This is tolerable on Linux with `uv`, but Windows/PWSH5
is where the user actually works day to day, and there it is unworkable.
A compiled Go binary has no equivalent startup cost. This is a hard
performance constraint, not a stylistic preference.

`bench/loomyard-eval/scripts/gen_compact_toc.py` is a pre-existing exception,
not a sanctioned precedent — do not extend it or copy its pattern.

`bench/loomyard-eval/ladder/` (added by task #008, `mcp-capability-bench`) is
a grandfathered Python exception: 5 of its 9 batches were already
approved/committed when this rule was raised mid-task, so it was allowed to
finish rather than being rewritten in flight. Task `#009
port-ladder-bench-to-go` tracks porting it to Go once #008 is fully done.
