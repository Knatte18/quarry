"""
Tests for score_run.py: the answer-redaction units (redact_text/
redact_answer/write_redacted) and the pinned, blinded scoring dispatch
(strip_fasit_meta/build_scorer_prompt/score_run).

Both units are pure once the model call is injected, so every test here
runs without a network call or a live model call -- the dispatch layer is
exercised by actually running the matrix in batch 8, never mocked here. The
injected `runner` mocks the subprocess boundary, not the model's judgement.

Usage:
    uv run --no-project --with pytest --with pyyaml python -m pytest \
        bench/loomyard-eval/ladder/tests/test_score_run.py -q
"""
import json
from pathlib import Path

import pytest
from ladder_config import QUARRY_TOOLS, Ladder, LadderConfig, ScorerConfig, TaskEntry
from score_run import (
    EXPLORATION_RULE,
    IMPACT_RULE,
    REDACTION_TOKEN,
    ScoringError,
    build_scorer_prompt,
    redact_answer,
    redact_text,
    score_run,
    strip_fasit_meta,
    write_redacted,
)

RESULTS_DIR = Path(__file__).resolve().parents[2] / "results" / "2026-08-28"
FASIT_EXPLORATION = RESULTS_DIR / "01-reed-geometry-exploration" / "c.json"
FASIT_IMPACT = RESULTS_DIR / "04-shedadapters-shuttle-impact" / "c.json"

# Impact-shaped answer whose evidence reads like the committed #006 A-arm
# phrasing (see FASIT_IMPACT): it names a quarry tool, invokes it as a CLI
# form, and separately uses "impact" as an ordinary English word.
IMPACT_ANSWER = {
    "callers_to_update": [
        {
            "file": "internal/shedadapters/singlellm.go",
            "line": 143,
            "evidence": (
                "ran quarry impact on singlellm.go:39:2 (--within internal/shedadapters), "
                "corroborated by mcp__quarry__impact. The impact of this change is that "
                "ctx must be threaded through the call chain."
            ),
        }
    ],
    "excluded_lookalikes": [
        {
            "file": "internal/shedadapters/burler.go",
            "line": 373,
            "reason": "Resolves to a different interface, confirmed with mcp__quarry__textDocument_definition, not quarry impact.",
        }
    ],
    "open_questions": ["Does mcp__quarry__workspace_symbol reach this via bouncer.go too?"],
    "confidence": "high",
}

EXPLORATION_ANSWER = {
    "relevant_files": ["internal/reedcli/attach.go", "internal/reedengine/attach.go"],
    "key_symbols": [
        {
            "name": "attachCmd",
            "file": "internal/reedcli/attach.go",
            "role": "Reads terminal size via mcp__quarry__toc_file and calls Engine.AttachArgv.",
        }
    ],
    "summary": "Uses quarry's toc_file and workspace_symbol tools plus /tmp/quarry-bench fixtures to trace geometry.",
    "confidence": "high",
    "open_questions": ["Does mcp__quarry__impact ever get called here?"],
}


""" REDACTION """


def test_redact_text_preserves_impact_as_common_noun():
    text = "the impact of this refactor is limited to one package"
    assert redact_text(text) == text


def test_redact_text_redacts_cli_invocation_form_of_impact():
    text = "ran quarry impact on singlellm.go:39:2 to confirm"
    redacted = redact_text(text)
    assert REDACTION_TOKEN in redacted
    assert "quarry" not in redacted.lower()
    assert "impact" not in redacted.lower()
    assert "singlellm.go:39:2" in redacted


def test_redact_text_collapses_adjacent_redaction_tokens():
    text = "mcp__quarry__toc_dir mcp__quarry__toc_file returned nothing"
    assert redact_text(text) == f"{REDACTION_TOKEN} returned nothing"


def test_redact_answer_preserves_common_noun_impact_and_strips_tool_provenance():
    redacted = redact_answer(IMPACT_ANSWER)
    evidence = redacted["callers_to_update"][0]["evidence"]

    assert "The impact of this change is that ctx must be threaded through the call chain." in evidence
    assert "mcp__quarry__impact" not in evidence
    assert "quarry" not in evidence.lower()
    assert REDACTION_TOKEN in evidence

    caller = redacted["callers_to_update"][0]
    assert caller["file"] == "internal/shedadapters/singlellm.go"
    assert caller["line"] == 143
    assert redacted["confidence"] == "high"


def test_redact_answer_preserves_relevant_files_and_redacts_summary():
    redacted = redact_answer(EXPLORATION_ANSWER)

    assert redacted["relevant_files"] == EXPLORATION_ANSWER["relevant_files"]
    assert redacted["summary"] != EXPLORATION_ANSWER["summary"]
    assert "quarry" not in redacted["summary"].lower()


def test_redact_answer_leaves_no_quarry_trace_anywhere():
    for redacted in (redact_answer(IMPACT_ANSWER), redact_answer(EXPLORATION_ANSWER)):
        blob = json.dumps(redacted).lower()
        assert "quarry" not in blob
        assert "mcp__quarry__" not in blob


def test_write_redacted_leaves_original_answer_byte_identical(tmp_path):
    original_bytes = json.dumps(IMPACT_ANSWER, indent=2).encode() + b"\n"
    (tmp_path / "answer.json").write_bytes(original_bytes)

    redacted = write_redacted(tmp_path)

    assert (tmp_path / "answer.json").read_bytes() == original_bytes
    on_disk = json.loads((tmp_path / "answer.redacted.json").read_text())
    assert on_disk == redacted
    assert "quarry" not in json.dumps(on_disk).lower()


""" SCORING DISPATCH """


@pytest.fixture
def ladder():
    tasks = {
        "01-reed-geometry-exploration": TaskEntry(
            task_file="bench/loomyard-eval/tasks/01-reed-geometry-exploration.md",
            pinned_sha="975578cda8d6f3a81580bd4e73725e060211b766",
            worktree="/tmp/loomyard-eval-01",
            schema="exploration",
            fasit=str(FASIT_EXPLORATION),
        ),
        "04-shedadapters-shuttle-impact": TaskEntry(
            task_file="bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md",
            pinned_sha="975578cda8d6f3a81580bd4e73725e060211b766",
            worktree="/tmp/loomyard-eval-04",
            schema="impact",
            fasit=str(FASIT_IMPACT),
        ),
    }
    configs = (
        LadderConfig(
            id="a5-bundle",
            ladder="a",
            task="01-reed-geometry-exploration",
            allowed=("toc_dir", "toc_file", "workspace_symbol"),
            cold=False,
        ),
        LadderConfig(
            id="b7-bundle",
            ladder="b",
            task="04-shedadapters-shuttle-impact",
            allowed=("impact", "assert_no_callers"),
            cold=False,
        ),
    )
    return Ladder(
        run_model="claude-opus-5",
        reps=3,
        max_turns=60,
        scorer=ScorerConfig(model="claude-opus-5", effort="high"),
        quarry_tools=QUARRY_TOOLS,
        tasks=tasks,
        source_repo="/home/knatte/Code/loomyard/wts/loomyard",
        cold_worktree_template="/tmp/loomyard-eval-01-cold-{n}",
        configs=configs,
    )


def _config(ladder, config_id):
    return next(candidate for candidate in ladder.configs if candidate.id == config_id)


def test_build_scorer_prompt_never_leaks_run_identity(ladder):
    config = _config(ladder, "b7-bundle")
    with open(FASIT_IMPACT) as f:
        fasit = json.load(f)
    redacted_answer = redact_answer(IMPACT_ANSWER)
    task_text = "Analyze the fallout of the Shuttle interface change across shedadapters."

    prompt = build_scorer_prompt(ladder, config, redacted_answer, fasit, task_text)

    assert "b7-bundle" not in prompt
    assert "assert_no_callers" not in prompt
    assert "ladder: b" not in prompt.lower()
    assert "ladder b" not in prompt.lower()
    assert task_text in prompt
    assert json.dumps(redacted_answer, indent=2) in prompt


def test_build_scorer_prompt_strips_fasit_meta_but_keeps_evidence_verbatim(ladder):
    config = _config(ladder, "b7-bundle")
    with open(FASIT_IMPACT) as f:
        fasit = json.load(f)
    redacted_answer = redact_answer(IMPACT_ANSWER)

    prompt = build_scorer_prompt(ladder, config, redacted_answer, fasit, "task text")

    assert "_meta" not in prompt
    assert "reference/fasit agent" not in prompt
    assert fasit["callers_to_update"][0]["evidence"] in prompt
    assert strip_fasit_meta(fasit) == json.loads(json.dumps(strip_fasit_meta(fasit)))
    assert "_meta" not in strip_fasit_meta(fasit)


def test_build_scorer_prompt_selects_template_by_task_schema(ladder):
    exploration_config = _config(ladder, "a5-bundle")
    with open(FASIT_EXPLORATION) as f:
        exploration_fasit = json.load(f)
    exploration_prompt = build_scorer_prompt(
        ladder, exploration_config, redact_answer(EXPLORATION_ANSWER), exploration_fasit, "task text"
    )
    assert EXPLORATION_RULE.strip() in exploration_prompt
    assert IMPACT_RULE.strip() not in exploration_prompt

    impact_config = _config(ladder, "b7-bundle")
    with open(FASIT_IMPACT) as f:
        impact_fasit = json.load(f)
    impact_prompt = build_scorer_prompt(ladder, impact_config, redact_answer(IMPACT_ANSWER), impact_fasit, "task text")
    assert IMPACT_RULE.strip() in impact_prompt
    assert EXPLORATION_RULE.strip() not in impact_prompt


def test_score_run_records_pinned_scorer_model_effort_and_template(ladder, tmp_path):
    config = _config(ladder, "b7-bundle")
    (tmp_path / "answer.json").write_text(json.dumps(IMPACT_ANSWER))

    def runner(prompt, model, effort):
        return '```json\n{"recall": 1.0, "precision": 1.0, "decoy_admitted": false, "lookalikes_matched": 0}\n```'

    score = score_run(ladder, config, tmp_path, "task text", runner=runner)

    assert score["model"] == ladder.scorer.model
    assert score["effort"] == ladder.scorer.effort
    assert score["prompt_template"] == "impact"
    assert score["decoy_admitted"] is False
    assert score["lookalikes_matched"] == 0

    written = json.loads((tmp_path / "score.json").read_text())
    assert written == score


def test_score_run_reports_decoy_admitted_and_lookalikes_matched_outside_precision(ladder, tmp_path):
    config = _config(ladder, "b7-bundle")
    (tmp_path / "answer.json").write_text(json.dumps(IMPACT_ANSWER))

    def runner(prompt, model, effort):
        return '```json\n{"recall": 0.67, "precision": 1.0, "decoy_admitted": true, "lookalikes_matched": 2}\n```'

    score = score_run(ladder, config, tmp_path, "task text", runner=runner)

    assert "decoy_admitted" in score
    assert "lookalikes_matched" in score
    assert score["decoy_admitted"] is True
    assert score["lookalikes_matched"] == 2
    # decoy_admitted is its own field, never folded into precision.
    assert score["precision"] == 1.0


def test_score_run_passes_redacted_answer_not_original_to_runner(ladder, tmp_path):
    # The fasit is deliberately left verbatim (see strip_fasit_meta's
    # docstring) and its committed evidence text names "quarry" itself, so
    # this test asserts the *answer* section reaching the runner carries no
    # tool provenance, not that the word never appears anywhere in the
    # prompt.
    config = _config(ladder, "b7-bundle")
    (tmp_path / "answer.json").write_text(json.dumps(IMPACT_ANSWER))

    captured = {}

    def runner(prompt, model, effort):
        captured["prompt"] = prompt
        return '```json\n{"recall": 0.5, "precision": 0.5, "decoy_admitted": false, "lookalikes_matched": 0}\n```'

    score_run(ladder, config, tmp_path, "task text", runner=runner)

    answer_section = captured["prompt"].split("## Answer to score")[1]
    assert "mcp__quarry__impact" not in answer_section
    assert "quarry" not in answer_section.lower()


def test_score_run_raises_scoring_error_when_reply_has_no_fenced_json(ladder, tmp_path):
    config = _config(ladder, "b7-bundle")
    (tmp_path / "answer.json").write_text(json.dumps(IMPACT_ANSWER))

    def runner(prompt, model, effort):
        return "I refuse to answer."

    with pytest.raises(ScoringError):
        score_run(ladder, config, tmp_path, "task text", runner=runner)
