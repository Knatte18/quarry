# 2026-09-02 compact ladder — conclusion

`ladder-compact.yaml` via `run.sh compact <this root>`, **reps 2**, host Neve (WSL2), server built from
`f3d3caf` with a clean tracked tree (`provenance.json`). 6 runs, all ingested and scored, none excluded.
One distinct `server_hashes` value across all six repetitions — the binary did not change mid-run.

**This is a probe, not a dataset.** Two repetitions per cell were chosen deliberately to answer "does the
compact form obviously change what the agent does" cheaply. Every "separated" verdict below is two
non-overlapping two-point ranges; a single further repetition either way could erase any of them. Nothing
here is strong enough to flip a default on. Read the last section before acting.

The delivery mechanism worked exactly as designed: `toc_format` forced the form per cell, and the two
cells' `toc_dir` results are the two forms of the same map.

| cell | rep | `toc_dir` result | first bytes |
|---|---|---|---|
| a2-toc-dir | 1, 2 | 48 607 B | `{"results":[{"dirs":["render"],"files":[{…` |
| a9-toc-dir-compact | 1 | 12 472 B | `{"results":[{"compact":"# internal/reedengine (package reede…` |
| a9-toc-dir-compact | 2 | 14 386 B | `{"results":[{"compact":"# /tmp/loomyard-eval-01/internal/ree…` |

The two a9 repetitions differ because the agent passed a relative path in rep 1 and an absolute one in
rep 2; the header line echoes what it was given. Not a form defect.

## Numbers (median [min–max], per results root, this host)

| cell | n | turns | greps | cache_read | cache_creation | duration | recall | precision | Read chars | tool chars |
|---|---|---|---|---|---|---|---|---|---|---|
| a0-none | 2 | 7.5 [6–9] | 4.5 [4–5] | 123k [95–151] | 29k [29–30] | 56 s | 0.43 [0.40–0.45] | 0.94 [0.93–0.94] | 34k | 0 |
| a2-toc-dir | 2 | 4.5 [4–5] | 0 [0–0] | 99k [75–122] | 57k [51–63] | 54 s | 0.40 [0.35–0.45] | 0.96 [0.92–1.00] | 68k | 49k |
| a9-toc-dir-compact | 2 | 5.5 [4–7] | 0.5 [0–1] | 111k [64–158] | 29k [16–43] | 62 s | 0.36 [0.33–0.39] | 0.82 [0.78–0.86] | 49k | 13k |

"Read chars" is the total size of `Read` tool results in the transcript; "tool chars" the total size of
quarry tool results. `output_tokens` is unusable on the WSL2 hosts and is not shown. `denied_tool_attempts`
is 0 in every cell and is structurally incapable of being anything else — see the standing note on
`DenialShapePattern`; it is not evidence that no tool call was refused.

## a2 reproduces. a9 does not.

**a2-toc-dir is the established win again**, at n=2 on this host: turns 4.5 vs the control's 7.5
(separated), greps 0 vs 4.5, recall and precision unchanged (0.40 vs 0.43, 0.96 vs 0.94, neither
separated). One `toc_dir` call per repetition, no fallback to grep in either. Consistent with
`results/2026-09-02-toc` at n=4 and with 2026-08-30.

**a9-toc-dir-compact bought the bytes and lost the behaviour.** The saving is real and is the largest,
cleanest effect in the root:

- the toc result itself: 48.6 KB → 12.5 KB, a quarter, as predicted;
- total quarry tool bytes: 49k → 13k;
- `cache_creation`: 57k [51–63] → 29k [16–43], separated from a2 and level with the control, which never
  called a tool at all.

But every behavioural number moved the wrong way, and two of the three moves are separated:

- **precision 0.96 [0.92–1.00] → 0.82 [0.78–0.86]**, separated from a2 *and* separated from the control's
  0.935. The compact cell is the only cell in the root that is less precise than calling no tool.
- **recall 0.36 [0.33–0.39]**, separated below the control's 0.425. Not separated from a2's 0.40 [0.35–0.45].
- **turns 5.5 [4–7]: no longer separated from the control.** a2's turn halving is the whole reason toc_dir
  ships; a9 does not reproduce it. a9 also fell back to a Bash grep in one repetition; a2 never did.
- duration 62 s vs a2's 54 s and the control's 56 s, separated from both. Comparable here because it is
  within one root on one host.

## The mechanism is not the one §3 predicted

HANDOFF §3 anticipated that if the 120-char one-sentence-per-file cap cost the agent information, the
symptom would be **Read volume up** — the agent compensating by opening files the JSON form would have
described. The opposite happened:

| cell | Read chars | quarry chars | total tool chars |
|---|---|---|---|
| a0-none | 34k | 0 | 44k |
| a2-toc-dir | 68k | 49k | 117k |
| a9-toc-dir-compact | 49k | 13k | 62k |

a9 read **less** than a2 (49k vs 68k), not more. It did not notice the thinner map and go looking; it
answered from it. That is why precision fell rather than cost rising: the compact form did not make the
agent work harder for the same answer, it made it confident on less. The JSON form's per-file first
paragraph — the 69% of its bytes that is header-comment prose, the part the compact form truncates to one
sentence — appears to be doing work, not padding.

This is §3's third bullet ("recall down → do not adopt until understood"), reached by a route §3 did not
list.

## Verdict

**Do not flip the defaults.** The §3 plan — compact by default in the MCP server, a `quarry map` verb,
`docs/when-to-use-quarry.md` pointed at it — is blocked. Adopting a form that halves the bill and takes
precision from 0.96 to 0.82 would trade the one measured win for a cheaper wrong answer.

**Do not discard the compact form either.** n=2 cannot carry this. The byte saving is a deterministic
property of the form and is not in question; what is in question is a behavioural difference resting on
two observations per cell, in a root whose within-cell spread is wide (a9's own two repetitions are 4 vs 7
turns and 64k vs 158k cache_read).

Next, in order:

1. **Re-run this ladder at `reps: 5`**, into a fresh root. Same three cells, no changes. That is the
   cheapest thing that can settle it, and it is the same 25 minutes the toc ladder cost. If precision
   holds at ~0.82 across five repetitions the form is rejected on evidence; if it converges on a2's 0.96
   this probe was noise and the flip proceeds.
2. Only if it is rejected: the knob to try before abandoning the idea is the sentence cap. `leadMaxRunes`
   at 120 is what produced the thin lines; the compact form at two or three sentences per file would still
   be far below 48 KB. Cell `a10-toc-dir-compact-wide` against a9 is the shape.
3. §6 item 6 (compact output for impact/refs) inherits this result — it was queued behind toc's compact
   form being adopted, and is now queued behind step 1.

## Harness note — `server_vcs_modified` cannot read false

`provenance.json` here records `quarry_dirty: false` and `server_vcs_modified: "true"` at the same time.
HANDOFF §3's pre-read checklist asks the operator to confirm `server_vcs_modified` is false; **that check
can never pass, on any run, regardless of operator discipline.**

The two fields are sampled at different moments against different definitions. `newProvenance`
(`tools/runmatrix/main.go:666`) runs `git status --porcelain` when the root object is constructed, before
the results directory exists. Go stamps `vcs.modified` into the binary at build time — after `runmatrix`
has created the results root, which is untracked, which makes `git status --porcelain` non-empty from that
point on. Verified directly: with the compact2 root as the only entry in `git status --porcelain`, a fresh
`go build ./cmd/quarry-mcp` stamps `vcs.modified=true`. `results/2026-09-02-toc/provenance.json` carries
the same `"true"`.

So `server_vcs_modified` is structurally true and carries no information. The fields that do carry it are
`quarry_dirty` (tracked-tree state before the run) and `server_hashes` (one distinct value here, which is
the real "the binary did not change mid-run" evidence). This is the same shape as the standing
`denied_tool_attempts` note: a gate that reads a constant is worse than no gate, because it is read as
having passed. §3's checklist should drop it.
