"""
Keeps the scoring agent blind to which rung of the capability ladder it is
grading, and dispatches the pinned, three-input scoring call that turns one
run's answer into a `score.json`.

Blinding has two parts: `redact_text`/`redact_answer`/`write_redacted` strip
every trace of tool provenance -- client-side mcp__quarry__* names, bare
canonical tool names, the word "quarry", and CLI invocation forms -- out of
an answer's free-text fields before it ever reaches the scorer. `impact` is
the one canonical tool name excluded from the bare-name pass because it is
also an ordinary English word every Ladder-B answer's prose legitimately
uses; only its mcp__quarry__impact and CLI-invocation forms are redacted.

`build_scorer_prompt`/`score_run` then assemble a prompt from exactly three
inputs -- the redacted answer, the `_meta`-stripped fasit, and the task text
-- plus the fixed scoring rule for the task's schema, and dispatch it
through an injected `runner` so the unit tests never make a live model
call. The fasit's own free-text fields are left verbatim: it is one fixed
file identical across every rung of its task, so it cannot leak which
config is being graded, and its evidence/reason text is what the scoring
rules match a run's own entries against.

Usage:
    from score_run import score_run
    score = score_run(ladder, config, run_dir, task_text)
"""
import argparse
import copy
import json
import re
import subprocess
import sys
from pathlib import Path

from ladder_config import QUARRY_TOOLS, config_by_id, extract_fenced_json, load_ladder, mcp_name

# Placeholder every redacted tool-provenance mention is replaced with.
REDACTION_TOKEN = "<tool>"

# The canonical tool names redacted on their own, bare, unprefixed form.
# "impact" is deliberately excluded here -- see the module docstring and
# _REDACTION_PATTERN below.
_BARE_TOOL_NAMES_EXCEPT_IMPACT = tuple(tool for tool in QUARRY_TOOLS if tool != "impact")


def _build_redaction_pattern():
    """
    Builds the single compiled regex redact_text runs, as an alternation
    ordered from most to least specific so a more specific form (e.g. a full
    mcp__quarry__* name, or the "quarry <verb>" CLI shell form that also
    catches impact's CLI invocation) claims a match before the bare "quarry"
    fallback gets a chance to.
    """
    alternatives = [rf"\b{re.escape(mcp_name(tool))}\b" for tool in QUARRY_TOOLS]
    # "quarry <verb>" CLI shell form, e.g. "quarry impact ..." or
    # "quarry toc_file ...". Consumed as one unit, which is also the only
    # path that redacts impact's CLI-invocation form (impact itself is
    # excluded from the bare-name pass below).
    alternatives.append(r"\bquarry\s+[A-Za-z_]+\b")
    alternatives.extend(rf"\b{re.escape(tool)}\b" for tool in _BARE_TOOL_NAMES_EXCEPT_IMPACT)
    alternatives.append(re.escape("/tmp/quarry-bench"))
    alternatives.append(r"--target-dir(?:[= ]\S+)?")
    # The bare word "quarry" on its own, last so every more specific
    # alternative above gets first refusal at the same starting position.
    alternatives.append(r"\bquarry\b")
    return re.compile("|".join(alternatives), re.IGNORECASE)


_REDACTION_PATTERN = _build_redaction_pattern()

# Collapses a run of REDACTION_TOKENs produced by adjacent matches from one
# phrase (e.g. two adjacent mcp__quarry__* names) into a single token, so
# the redacted prose stays readable instead of stuttering.
_ADJACENT_TOKEN_RUN = re.compile(re.escape(REDACTION_TOKEN) + r"(?:\s+" + re.escape(REDACTION_TOKEN) + r")+")


def redact_text(text):
    """
    Replaces every case-insensitive occurrence of tool provenance in text
    with REDACTION_TOKEN: every mcp__quarry__* client-side name, every bare
    canonical quarry tool name except "impact", the word "quarry", and CLI
    invocation forms (the literal "/tmp/quarry-bench" path, a
    "quarry <verb>" shell form, and a "--target-dir"-style flag).

    A run of adjacent tokens produced by one phrase collapses into a single
    token.
    """
    redacted = _REDACTION_PATTERN.sub(REDACTION_TOKEN, text)
    return _ADJACENT_TOKEN_RUN.sub(REDACTION_TOKEN, redacted)


def redact_answer(answer):
    """
    Returns a deep copy of an answer with redact_text applied to every
    free-text field a scorer would otherwise read tool provenance out of.

    Exploration answers get `summary`, `open_questions`, and each
    `key_symbols[].role` redacted. Impact answers get `open_questions`,
    each `callers_to_update[].evidence`, and each `excluded_lookalikes[].reason`
    redacted. Structural fields -- `relevant_files`, every `file`, every
    `line`, every `name`, and `confidence` -- are returned untouched.
    """
    redacted = copy.deepcopy(answer)

    if "summary" in redacted:
        redacted["summary"] = redact_text(redacted["summary"])
    if "open_questions" in redacted:
        redacted["open_questions"] = [redact_text(question) for question in redacted["open_questions"]]
    if "key_symbols" in redacted:
        for symbol in redacted["key_symbols"]:
            symbol["role"] = redact_text(symbol["role"])
    if "callers_to_update" in redacted:
        for caller in redacted["callers_to_update"]:
            caller["evidence"] = redact_text(caller["evidence"])
    if "excluded_lookalikes" in redacted:
        for lookalike in redacted["excluded_lookalikes"]:
            lookalike["reason"] = redact_text(lookalike["reason"])

    return redacted


def write_redacted(run_dir):
    """
    Reads answer.json from run_dir, writes the redacted copy beside it as
    answer.redacted.json, and leaves the original byte-identical.

    Returns the redacted answer mapping.
    """
    run_dir = Path(run_dir)
    with open(run_dir / "answer.json") as f:
        answer = json.load(f)

    redacted = redact_answer(answer)

    with open(run_dir / "answer.redacted.json", "w") as f:
        json.dump(redacted, f, indent=2)
        f.write("\n")

    return redacted


""" SCORING RULES AND DISPATCH """


class ScoringError(Exception):
    """
    Raised when the scoring client exits non-zero, or its reply carries no
    parseable fenced json block.
    """


# Reproduced from the committed benchmark README's Scoring section, adapted
# from the three-arm A/B/C labelling to the blind fasit/answer pairing this
# scorer actually sees.
EXPLORATION_RULE = """Exploration scoring rule:

recall = (the fasit's relevant_files/key_symbols also present in the
answer's) / (the fasit's total); precision = (the answer's entries
corroborated by the fasit) / (the answer's total). Also judge qualitatively
whether the answer's summary describes the same actual mechanism the fasit
found, not just whether file names overlap.

Reply with ONLY a fenced json code block, no other trailing prose after it:
```json
{"recall": <float 0.0-1.0>, "precision": <float 0.0-1.0>, "summary_matches": <true|false>}
```
"""

IMPACT_RULE = """Impact-analysis scoring rule:

recall = (the fasit's callers_to_update entries matched on file AND line --
a line must denote the same call site, not merely the same file -- also
present in the answer's) / (the fasit's total); precision = (the answer's
callers_to_update entries corroborated by the fasit) / (the answer's
total). decoy_admitted is true when the answer's callers_to_update contains
a call site the fasit lists under excluded_lookalikes -- report this as its
own field, never folded into precision. lookalikes_matched is the count of
the answer's excluded_lookalikes the fasit also names -- credited, never
required, so an answer naming none loses no points for it.

Reply with ONLY a fenced json code block, no other trailing prose after it:
```json
{"recall": <float 0.0-1.0>, "precision": <float 0.0-1.0>, "decoy_admitted": <true|false>, "lookalikes_matched": <int>}
```
"""

_RULE_BY_SCHEMA = {
    "exploration": EXPLORATION_RULE,
    "impact": IMPACT_RULE,
}


def strip_fasit_meta(fasit):
    """
    Returns a copy of a loaded fasit mapping with its top-level `_meta`
    block removed.

    `_meta` is identical across every run of a task and carries no
    per-config signal, but its `role`/`see_also` text names quarry and
    scorecard.md -- scoring-irrelevant, so it is dropped rather than left to
    sit unexplained next to an answer deliberately redacted of the same
    words. Every other field, including the free-text evidence/reason
    strings the scoring rules match a run's entries against, is left
    verbatim: the fasit is one fixed file, identical across all 45 runs of
    its task, so it cannot tell the scorer which config it is grading, and
    redacting it would damage the very fields recall/precision are computed
    from.
    """
    stripped = dict(fasit)
    stripped.pop("_meta", None)
    return stripped


def build_scorer_prompt(ladder, config, redacted_answer, fasit, task_text):
    """
    Assembles the scorer's prompt from exactly three inputs plus the fixed
    rule for the task's schema: the redacted answer, the `_meta`-stripped
    fasit, and the task text.

    Never embeds config.id, config.ladder, config.allowed, the transcript,
    or any other run's answer -- the scorer must not learn which rung it is
    grading. The task text is included because the exploration rule cannot
    judge the summary without it, and it is identical across a ladder's
    rungs.
    """
    task = ladder.tasks[config.task]
    rule = _RULE_BY_SCHEMA[task.schema]
    stripped_fasit = strip_fasit_meta(fasit)

    return f"""{rule}
## Task

{task_text}

## Reference fasit

```json
{json.dumps(stripped_fasit, indent=2)}
```

## Answer to score

```json
{json.dumps(redacted_answer, indent=2)}
```
"""


def _extract_fenced_json(reply):
    """Parses the first fenced json code block out of a scorer reply.
    Raises ScoringError when none is present or it does not parse."""
    found = extract_fenced_json(reply, which="first")
    if found is None:
        raise ScoringError("scorer reply carried no fenced json block")
    _block_text, inner_text = found
    try:
        return json.loads(inner_text)
    except json.JSONDecodeError as exc:
        raise ScoringError(f"scorer reply's fenced json block did not parse: {exc}") from exc


def run_scorer_client(prompt, model, effort):
    """
    Dispatches prompt to `claude -p`, pinned to model/effort, with the same
    blinding flags every other dispatch in the suite uses:
    `--setting-sources ""` and `--strict-mcp-config` (no ambient settings or
    MCP source can leak in) and `--permission-mode dontAsk` (a scoring call
    that reached for a tool would otherwise block on a prompt no one is
    there to answer, stalling the matrix mid-run).

    Returns the client's reply text. Raises ScoringError on a non-zero
    exit or an unparseable `--output-format json` envelope.
    """
    result = subprocess.run(
        [
            "claude",
            "-p",
            prompt,
            "--setting-sources",
            "",
            "--strict-mcp-config",
            "--permission-mode",
            "dontAsk",
            "--model",
            model,
            "--effort",
            effort,
            "--output-format",
            "json",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        raise ScoringError(f"scorer client exited {result.returncode}: {result.stderr}")
    try:
        envelope = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise ScoringError(f"scorer client produced a non-JSON --output-format json envelope: {exc}") from exc
    return envelope.get("result", "")


def score_run(ladder, config, run_dir, task_text, runner=run_scorer_client):
    """
    Scores one run's answer against its task's fasit, blind to which rung
    produced it.

    Writes answer.redacted.json (via write_redacted) and score.json into
    run_dir, and returns the parsed score mapping. score.json additionally
    carries the resolved scorer `model`, `effort`, and `prompt_template`
    (the task schema the template was chosen from), so a drifting scorer
    prompt is visible in the record.

    Args:
        runner: called as runner(prompt, model, effort) -> reply text.
            Defaults to run_scorer_client, which shells out to `claude -p`.
            Tests inject a fake runner so no unit test makes a live call.
    """
    run_dir = Path(run_dir)
    task = ladder.tasks[config.task]

    redacted_answer = write_redacted(run_dir)

    with open(task.fasit) as f:
        fasit = json.load(f)

    prompt = build_scorer_prompt(ladder, config, redacted_answer, fasit, task_text)
    reply = runner(prompt, ladder.scorer.model, ladder.scorer.effort)
    score = _extract_fenced_json(reply)

    score["model"] = ladder.scorer.model
    score["effort"] = ladder.scorer.effort
    score["prompt_template"] = task.schema

    with open(run_dir / "score.json", "w") as f:
        json.dump(score, f, indent=2)
        f.write("\n")

    return score


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Score one run's answer against its task's fasit through the pinned, blinded scorer.")
    parser.add_argument("--ladder", required=True, help="path to ladder.yaml")
    parser.add_argument("--config-id", required=True, help="config id whose run is being scored")
    parser.add_argument("--run-dir", required=True, help="run directory holding answer.json")
    args = parser.parse_args()

    loaded_ladder = load_ladder(args.ladder)
    run_config = config_by_id(loaded_ladder, args.config_id)
    run_task = loaded_ladder.tasks[run_config.task]
    with open(run_task.task_file) as f:
        run_task_text = f.read()

    run_score = score_run(loaded_ladder, run_config, args.run_dir, run_task_text)
    json.dump(run_score, sys.stdout, indent=2)
    sys.stdout.write("\n")
