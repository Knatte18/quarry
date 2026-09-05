MILL_REVIEW_BEGIN
# Review: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a)

```yaml
duration_s: 212.4
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic); exact build not self-verifiable
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] `--view glyphs` interaction with `--symbols` undecided
**Section:** `preset-expansion` + Testing (golden list)
**Issue:** The decisions cover `--symbols` only inside the frozen preset; nothing decides `toc --view glyphs` without `--symbols` (nil ⇒ false for a directory target, `internal/engine/answer.go:257`) or with `--no-symbols`, both of which produce an empty index — the wrong-negative failure `incomplete-is-explicit` exists to prevent. The discussion's own golden row `toc-view-glyphs-depth.txt` — `toc --view glyphs --depth 1 internal/logger` — omits `--symbols` and would therefore pin an empty symbol list, contradicting its stated point ("a nested level's symbols appear in the flat list").
**Fix:** Decide the interaction (reject `--no-symbols` under `--view glyphs`, force `Symbols: true`, or accept the empty answer as honest) and respell the depth golden's invocation to match.

### [BLOCKING:design] Post-rewrite usage errors name `toc`, not `glyphs`
**Section:** `rewrite-mechanism` + Testing (`parseArgs` cases)
**Issue:** After the argv rewrite the parser's `verb` is `"toc"`, so `internal/cli/flags.go:166` emits `toc takes exactly one target, got 2` for `quarry glyphs a b`, and `--unit is not valid for toc` for a `glyphs` invocation — while `preset-is-frozen` promises rejections naming `glyphs`. The testing plan lists the zero-target and two-target `glyphs` cases but states no expected message.
**Fix:** State which verb name post-rewrite errors carry, and where the target-count/unknown-flag checks run relative to the rewrite.

### [NIT:consistency] `unknown verb` message claim contradicts the source
**Demoted-from:** BLOCKING
**Section:** Technical context ("The CLI") + Testing
**Issue:** The claim that `"no verb given"` and `"unknown verb"` "both enumerate the verbs" is false: `flags.go:74` is `unknown verb: %s` with no enumeration (only the two `"no verb given"` literals at `:66`/`:71` enumerate), and `flags_test.go:157` pins that exact string. The testing plan then requires "the `unknown verb` and `no verb given` messages now naming five verbs", mandating a change to an existing message under the Constraints' "no existing … verb behaviour changes".
**Fix:** Correct the technical-context claim and state explicitly whether `unknown verb` gains an enumeration (and if so, that the existing test expectation changes) or stays as-is.

### [NIT:scope] Two required artefacts named only in prose
**Section:** Scope "In" vs Technical context
**Issue:** The `docs/roadmap.md` 2a removal, the new `internal/cli/testdata/INDEX.md` rows, its "fifteen files" counts (`INDEX.md:105`, `after_test.go:1`) and the sentence distinguishing the new view from the retired compact view are all required by Technical context but absent from the Scope "In" list a plan writer enumerates from.
**Fix:** Add both artefacts to Scope "In".

### [NIT:design] Empty-answer wire shape unstated
**Section:** `glyphs-answer-shape` + `text-line-shape`
**Issue:** `Symbols []Symbol` carries `json:"symbols"` with no `omitempty`, so a zero-symbol answer emits `"symbols": null`, not `[]`, and the JSON testing item asserts the key set but never the empty value. On the text side the "empty string" rule is claimed to match `RenderText`'s contract, which is "ends with exactly one `\n`" (`quarry/text.go:19`) — the empty string does not satisfy it.
**Fix:** State the empty-answer JSON value and give the glyphs text renderer its own stated contract rather than borrowing `RenderText`'s.

### [NIT:design] `Target` echo is unnormalised across surfaces
**Section:** `glyphs-answer-shape` + `facade-shape`
**Issue:** `Target` echoes the caller's string verbatim, but `Repo.TOC` accepts both `""` and `"."` for the root (`quarry/repo.go:30`), so `Glyphs("")` emits `"target": ""` where the CLI's `RepoRelTarget` yields `"."` — two spellings of one query.
**Fix:** State the normalisation rule, or that `Target` is a verbatim echo and may differ between CLI and facade.

## Verdict

REQUEST_CHANGES
Symbols interaction, post-rewrite error verb, and a false source claim need deciding first.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
