# 2026-09-02 compact ladder — conclusion

`ladder-compact.yaml` via `run.sh compact <this root>`, **reps 2**, host Neve (WSL2), server built from
`f3d3caf` with a clean tracked tree (`provenance.json`). 6 runs, all ingested and scored, none excluded.
One distinct `server_hashes` value across all six repetitions — the binary did not change mid-run.

**Two repetitions per cell, deliberately.** The question was whether the compact form is *free* — same
behaviour, fewer bytes — and a claim of no effect only needs enough resolution to see the effect it claims
to preserve. This root has that resolution: a2-toc-dir reproduces the established turn halving at n=2.
Every "separated" verdict below is still two non-overlapping two-point ranges, so treat single metrics
with care; the verdict rests on the pattern, not on any one of them. See the last section.

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

## Verdict — the compact form as built is rejected. Do not re-run this ladder.

**Do not flip the defaults.** The §3 plan — compact by default in the MCP server, a `quarry map` verb,
`docs/when-to-use-quarry.md` pointed at it — is dropped, not deferred. Adopting a form that cuts the bill
and takes precision from 0.96 to 0.82 would trade the one measured win for a cheaper wrong answer.

**Two repetitions are enough to decide this, because the claim under test was that the form is free.** A
null-effect claim does not need the resolution a small-effect claim needs; it needs only enough resolution
to see the effect it claims to preserve. This root has that: a2-toc-dir reproduced the established turn
halving at n=2 (4.5 vs the control's 7.5, separated, zero greps vs 4.5). The effect is visible here at two
repetitions. a9 did not show it at the same n, under the same task, on the same host, in the same root.
And precision went 2/2 in the same direction — 0.78 and 0.86, below every one of a2's observations (0.92,
1.00) and below both control observations (0.93, 0.94). Spending another 25 minutes to confirm that
something advertised as free is not free is not worth it.

What this does **not** establish: that a compact map is a bad idea. It rejects *this* form — one sentence
per file, `leadMaxRunes` at 120 — whose truncation is the most likely cause of the agent answering from a
thinner map. A two- or three-sentence-per-file form would still be far below 48 KB and is a different
form, not a re-run of this one. If anyone builds it, it is a new cell (`a10-toc-dir-compact-wide` against
a9) and a new question; nothing in the current programme depends on it.

**§6 item 6 (compact output for impact/refs) loses its premise.** It was queued behind toc's compact form
being adopted. It is not blocked on more data; it is unmotivated until some compact form is shown to be
free, which this one is not.

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
