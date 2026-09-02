# 2026-09-02 follow-up matrix — conclusion

Re-run of `ladder-followup.yaml` (task 04, b0/b1/b2/b4, reps 2) via `run-followup.sh`, replacing the
invalid 2026-09-01 run (see `../2026-09-01-followup/conclusion.md`). `provenance.json`: server built from
`6ddf737` (the only dirty files were the harness/docs changes of the day, no server source), host
OSL-1033 (WSL2), Loomyard pin present. Every comparison below is within this root, rung vs control;
`duration_ms` is this machine's and not comparable to the August or 2026-09-01 roots.

**Short version:** the `within` fix (1592f4e) works and is used, and b1-symbol's cost now overlaps the
control. The description-steering fix (ee84d8d) does not remove the position-addressing friction on
b2/b4 and adds a new wrong turn. b4-lsp-trio lost recall in one rep (0.33) after four misfired tool
calls in a row, because the tool displaced the grep the control agent always runs.

| cell | quarry calls | turns | cache_read | duration (this host) | recall |
|---|---|---|---|---|---|
| b0-none | 0 | 5 [5-5] | 62k [57-68k] | 18.4 s [17.9-19.0] | 1.0 |
| b1-symbol | 1 [1-1] | 5.5 [4-7] | 106k [68-144k] | 29.9 s [18.5-41.3] | 1.0 |
| b2-definition | 4 [4-4] | 11 [9-13] | 179k [134-223k] | 58.7 s [54.8-62.6] | 1.0 |
| b4-lsp-trio | 4 [2-6] | 8.5 [7-10] | 165k [127-203k] | 39.4 s [35.9-42.9] | **0.67 [0.33-1.0]** |

Precision 1.0 everywhere. `output_tokens` is unusable on this host — see "Metric defect" below.

## b1-symbol — `within` verified

Both reps issued exactly one `workspace_symbol` call, both with `within: internal/shedadapters`, for the
two queries `Shuttle` and `Run`. The result was ~4.1k chars in each rep; the same two queries unscoped
produced ~25.6k in August. The noise the fix targeted is gone.

Rep 2 (4 turns, 18.5 s) was the cheapest tool-granted run of the day and inside the control's range.
Rep 1 (7 turns, 41 s) was slow for reasons unrelated to the server: a malformed `Read` call (the model
emitted invalid JSON, a client-side error) and whole-file reads of `burler.go` and `bouncer.go`. Both
reps preferred `Read` of entire files over grep, which is where the higher `cache_read` comes from,
not from tool output. Verdict per HANDOFF §1's criterion: cost overlaps the control at n=2, with
high variance driven by reading strategy, not by the tool.

## b2-definition — steering did not help; position hunting is inherent to the task

The task is "what does the `Run` at *this call site* resolve to". That is a call-site question, and the
symbol form answers a declaration question: `textDocument+symbol` searches the named file for a
*declaration* of `Run`. Rep 2 tried exactly what the new description recommends —
`{textDocument: bouncer.go, symbol: "Run", within: ...}` — and got `not_found`, because `Run` is not
declared in `bouncer.go`. It then fell back to the August pattern: an `awk`/`index()` column hunt to
compute 0-based characters, and three more positional calls. Rep 1 never tried the symbol form and
walked the column instead (characters 20, 22, 30, 34 across four calls until one landed on `Run`).

So the 4 calls per rep are: one or two mis-aimed positions, a failed symbol-form attempt, and the
correct positional call. The description change cannot fix this; only a tool that accepts a
*call-site* address without a column can — e.g. `textDocument + line + symbol` ("the `Run` on line
466"), or 1-based columns matching what grep prints. That is HANDOFF §5 item 5 (1-based positions)
plus a line+symbol form, which the description fix was a stand-in for.

## b4-lsp-trio — one perfect rep, one recall collapse from tool friction

Rep 1 (recall 1.0): a first `references` call at a mis-aimed position on the declaration line (char 38,
which sits on `shuttleengine.Result`, returned ~7.7k chars of `Result` references), then a corrected call
at char 1 that returned exactly the three real call sites (bouncer.go:466/580, singlellm.go:143) in
604 chars. Clean LSP resolution; the answer's evidence cites it.

Rep 2 (recall 0.33): six server calls, four of which misfired in sequence:

1. `references` with both `position` and `symbol` → server error "got both position and symbol".
2. `references` with a stray top-level `file_path` → MCP schema validation error.
3. `references` with `symbol: "Shuttle.Run"` → `not_found` (qualified name; the description says bare).
4. `definition` at char 15 on the call line → `definitions: []` (whitespace), reported as found/complete.
5. `definition` at char 26 → `singlellm.go:38`, the interface method. Correct, but a definition, not
   the caller set.

The agent then answered from that single definition, listing only `singlellm.go:143`, and never grepped
`.Run(` — the one command every control rep ran first and which lists bouncer.go:466/580 in 400
chars. `bash_grep_count` for this rep is 0. This is the mechanism behind "worse": the granted tool
displaces the cheap reliable search, and when the tool surface rejects four attempts the agent stops
short instead of falling back. Note that 4 is the ee84d8d `omitzero` fix working as intended (an
empty list is now visible) — the agent still read "found, complete" and moved on.

## What the two fixes did, net

- **1592f4e `within`**: verified. Should become default-on (HANDOFF §5 item 3) — both reps had to
  know to pass it; the description tells them to, and they did.
- **ee84d8d descriptions**: no measurable benefit on b2, and the symbol-form steering produced a
  `not_found` detour. The `omitzero` part is correct and visible in b4 rep 2. The steering text should
  be reworded to say the symbol form is for *declarations* and that call-site resolution needs a
  position, until a line+symbol form exists.
- The single biggest cost/correctness lever visible in these transcripts is input validation
  ergonomics: three of b4 rep 2's four failures were malformed calls the schema rejected with a terse
  message. A server that accepted `position`+`symbol` (using the position, or the symbol on that line)
  and stripped qualifiers from `Type.Method` would have turned that rep into rep 1.

## Metric defect: `output_tokens` on this host

Every cell reports absurd `output_tokens` (control median 15 for a 5-turn run). The transcripts this
Claude Code version (2.1.258) writes carry only streaming snapshots of `output_tokens` (max seen in a
whole control transcript: 5), so `perCallUsage`'s max-per-call reduction yields near zero. The
2026-09-01 task-05 root, run on this same host, shows the same defect (c3-references min 71, max 655
across reps). The 2026-08-30 and 2026-09-01-followup roots, from the Linux host, are sane (~1.5–3.5k).
Do not read `output_tokens` from any root produced on this host; `cache_read_input_tokens` and turns
are the cost signals here. Fixing it means either a different reduction in `usage.go` or accepting the
metric is version-dependent and recording the client version in `provenance.json`.

## Correctness, for the record

Recall 1.0 and precision 1.0 in 7 of 8 runs; the exception is b4-lsp-trio rep 2 above. The control was
perfect in both reps in 5 turns with 2–3 greps, as in every previous root.
