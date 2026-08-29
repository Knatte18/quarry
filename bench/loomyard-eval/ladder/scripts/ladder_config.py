"""
Loads and validates the quarry-mcp capability ladder's single declarative
source of truth, `ladder.yaml`: the 15-config, two-task, two-ladder matrix
plus the cold cell. Also derives, from a loaded config, everything the rest
of the suite needs to launch a run -- the per-config permissions deny-list,
the settings document a run is launched with, and the rung's prompt
preamble -- so no downstream module re-derives any of these from scratch.

Usage:
    from ladder_config import load_ladder, deny_list_for, preamble_for
    ladder = load_ladder("bench/loomyard-eval/ladder/ladder.yaml")
"""
import json
from dataclasses import dataclass
from pathlib import Path

import yaml

# Canonical seven client-side tool names quarry-mcp exposes, bare (without
# the mcp__quarry__ prefix). This is the VALIDATION constant: load_ladder
# checks the ladder file's own `quarry_tools:` against it, so a ladder that
# drifts from quarry's real surface is rejected at load rather than silently
# producing a wrong deny-list downstream.
QUARRY_TOOLS = (
    "toc_dir",
    "toc_file",
    "textDocument_definition",
    "textDocument_references",
    "workspace_symbol",
    "impact",
    "assert_no_callers",
)

MCP_PREFIX = "mcp__quarry__"

# toc_dir/toc_file are tree-sitter-backed and never start a daemon (see
# tocFileHandler/tocDirHandler in internal/mcpserver/tools_toc.go, which
# call effectiveTargetDir/tocPreflight directly and never resolveCall). Every
# other canonical tool routes through resolveCall/EnsureServer and can be
# used as a warmth signal for the cold cell.
DAEMON_BACKED_TOOLS = tuple(t for t in QUARRY_TOOLS if t not in ("toc_dir", "toc_file"))


def mcp_name(tool):
    """Returns the client-side mcp__quarry__* name for a bare tool name."""
    return f"{MCP_PREFIX}{tool}"


class LadderConfigError(Exception):
    """Raised when ladder.yaml fails validation. The message names the
    offending field/config id and the ladder file path."""


@dataclass(frozen=True)
class LadderConfig:
    """One row of the matrix: a single (ladder, task, tool-exposure) cell.

    Instance variables:
        id: unique config id, e.g. "a5-bundle".
        ladder: "a" or "b" -- which ladder this config belongs to.
        task: key into Ladder.tasks.
        allowed: tuple of bare quarry_tools names exposed to this config's
            agent; empty for a `none` control.
        cold: True only for the single cold-cell config.
        warm_counterpart: the warm config id this cold config is contrasted
            against, or None for every non-cold config.
    """

    id: str
    ladder: str
    task: str
    allowed: tuple
    cold: bool
    warm_counterpart: str = None


@dataclass(frozen=True)
class TaskEntry:
    """One entry of Ladder.tasks: everything needed to set up and score one
    of the two target tasks (task 01 exploration, task 04 impact)."""

    task_file: str
    pinned_sha: str
    worktree: str
    schema: str
    fasit: str


@dataclass(frozen=True)
class ScorerConfig:
    """The pinned scoring client parameters, shared by every config."""

    model: str = None
    effort: str = None


@dataclass(frozen=True)
class Ladder:
    """The fully loaded, validated contents of ladder.yaml.

    Instance variables:
        run_model: the pinned model id for all 45 runs; None until the
            operator sets it.
        reps: repetitions per config.
        max_turns: per-run turn ceiling, identical across all 45 runs.
        scorer: ScorerConfig with the pinned scoring model/effort.
        quarry_tools: the canonical seven tool names, as loaded (validated
            to equal QUARRY_TOOLS).
        tasks: mapping of task slug to TaskEntry.
        source_repo: path to the Loomyard checkout the pinned worktrees are
            built from.
        cold_worktree_template: per-repetition cold worktree path template.
        configs: tuple of all 15 LadderConfig rows.
    """

    run_model: str
    reps: int
    max_turns: int
    scorer: ScorerConfig
    quarry_tools: tuple
    tasks: dict
    source_repo: str
    cold_worktree_template: str
    configs: tuple


def _fail(path, message):
    raise LadderConfigError(f"{path}: {message}")


def load_ladder(path):
    """
    Loads and validates ladder.yaml, returning a Ladder.

    Raises LadderConfigError on any of: a duplicate or non-unique config id;
    a `ladder` value outside a/b; a `task` key absent from `tasks:`; an
    `allowed` entry not in `quarry_tools`; a `quarry_tools` list that is not
    exactly the canonical seven; a ladder with zero or more than one config
    whose `allowed` is empty (the control); more than one `cold: true`
    config; a `warm_counterpart` on a non-cold config; a cold config with no
    `warm_counterpart`; and a `warm_counterpart` naming an unknown id, a cold
    config, or the cold config itself.
    """
    path = Path(path)
    with open(path) as f:
        raw = yaml.safe_load(f)

    quarry_tools = tuple(raw["quarry_tools"])
    if quarry_tools != QUARRY_TOOLS:
        _fail(path, f"quarry_tools must be exactly the canonical seven {QUARRY_TOOLS}, got {quarry_tools}")

    tasks = {
        slug: TaskEntry(
            task_file=entry["task_file"],
            pinned_sha=entry["pinned_sha"],
            worktree=entry["worktree"],
            schema=entry["schema"],
            fasit=entry["fasit"],
        )
        for slug, entry in raw["tasks"].items()
    }

    scorer_raw = raw["scorer"]
    scorer = ScorerConfig(model=scorer_raw.get("model"), effort=scorer_raw.get("effort"))

    configs = []
    seen_ids = set()
    for entry in raw["configs"]:
        config_id = entry["id"]
        if config_id in seen_ids:
            _fail(path, f"duplicate config id {config_id!r}")
        seen_ids.add(config_id)

        if entry["ladder"] not in ("a", "b"):
            _fail(path, f"config {config_id!r} has ladder {entry['ladder']!r}, must be 'a' or 'b'")

        if entry["task"] not in tasks:
            _fail(path, f"config {config_id!r} references unknown task {entry['task']!r}")

        allowed = tuple(entry["allowed"])
        for tool in allowed:
            if tool not in quarry_tools:
                _fail(path, f"config {config_id!r} allows unknown tool {tool!r}")

        configs.append(
            LadderConfig(
                id=config_id,
                ladder=entry["ladder"],
                task=entry["task"],
                allowed=allowed,
                cold=bool(entry.get("cold", False)),
                warm_counterpart=entry.get("warm_counterpart"),
            )
        )

    for ladder_name in ("a", "b"):
        controls = [c for c in configs if c.ladder == ladder_name and len(c.allowed) == 0]
        if len(controls) == 0:
            _fail(path, f"ladder {ladder_name!r} has no control config (empty allowed)")
        if len(controls) > 1:
            _fail(path, f"ladder {ladder_name!r} has {len(controls)} control configs (empty allowed); must have exactly one")

    cold_configs = [c for c in configs if c.cold]
    if len(cold_configs) > 1:
        _fail(path, f"found {len(cold_configs)} configs with cold: true; must have at most one")

    by_id = {c.id: c for c in configs}
    for config in configs:
        if config.warm_counterpart is not None and not config.cold:
            _fail(path, f"config {config.id!r} sets warm_counterpart but is not cold")
        if config.cold and config.warm_counterpart is None:
            _fail(path, f"cold config {config.id!r} has no warm_counterpart")
        if config.cold:
            target = by_id.get(config.warm_counterpart)
            if target is None:
                _fail(path, f"cold config {config.id!r} names unknown warm_counterpart {config.warm_counterpart!r}")
            elif target.cold:
                _fail(path, f"cold config {config.id!r} names a cold config as warm_counterpart: {config.warm_counterpart!r}")

    return Ladder(
        run_model=raw.get("run_model"),
        reps=raw["reps"],
        max_turns=raw.get("max_turns"),
        scorer=scorer,
        quarry_tools=quarry_tools,
        tasks=tasks,
        source_repo=raw["source_repo"],
        cold_worktree_template=raw["cold_worktree_template"],
        configs=tuple(configs),
    )


def config_by_id(ladder, config_id):
    """Returns the LadderConfig with the given id, or raises KeyError."""
    for config in ladder.configs:
        if config.id == config_id:
            return config
    raise KeyError(f"no config with id {config_id!r}")


def control_for(ladder, config):
    """
    Returns the `none` control for config's ladder -- the config on the
    same ladder whose `allowed` is empty. Resolved by field lookup, never by
    parsing config.id.
    """
    for candidate in ladder.configs:
        if candidate.ladder == config.ladder and len(candidate.allowed) == 0:
            return candidate
    raise LadderConfigError(f"no control config found for ladder {config.ladder!r}")


def warm_counterpart_for(ladder, config):
    """Returns the warm config a cold config's `warm_counterpart` field
    names, resolved through config_by_id rather than an id-suffix
    convention."""
    return config_by_id(ladder, config.warm_counterpart)


def require_pins(ladder):
    """
    Raises LadderConfigError naming the offending field when any pinned
    value the matrix depends on is unset: run_model, max_turns,
    scorer.model, or scorer.effort. Only run_model ships null by design; the
    other three ship with values, so this check exists to catch an edit that
    blanks one of them before the matrix starts, rather than reaching
    --model/--max-turns/--effort as a null on the command line.
    """
    if ladder.run_model is None:
        raise LadderConfigError("ladder.yaml: run_model is unset -- set it to the pinned model id before starting the matrix")
    if ladder.max_turns is None:
        raise LadderConfigError("ladder.yaml: max_turns is unset")
    if ladder.scorer.model is None:
        raise LadderConfigError("ladder.yaml: scorer.model is unset")
    if ladder.scorer.effort is None:
        raise LadderConfigError("ladder.yaml: scorer.effort is unset")


""" DENY-LIST AND SETTINGS DERIVATION """


def deny_list_for(ladder, config):
    """
    Returns the sorted list of client-side mcp__quarry__* names for every
    canonical tool in ladder.quarry_tools not in config.allowed. Always
    prefixed via mcp_name -- never assembled from a literal -- so a config's
    deny-list tracks ladder.quarry_tools even for a deliberately mutated,
    already-loaded Ladder (see the drift-guard test).
    """
    return sorted(mcp_name(tool) for tool in ladder.quarry_tools if tool not in config.allowed)


def settings_document_for(ladder, config):
    """
    Returns the full settings mapping a run is launched with:
    permissions.allow is fixed to Read/Grep/Glob/Bash (prompt-avoidance
    only, per the plan's Shared Decision -- never treated as an allowlist
    anywhere in this suite), and permissions.deny is config's quarry
    deny-list plus "Task", denied uniformly across all 45 runs so a
    dispatched subagent's tool calls can never produce an undercounted
    transcript.
    """
    deny = sorted(deny_list_for(ladder, config) + ["Task"])
    return {
        "permissions": {
            "allow": ["Read", "Grep", "Glob", "Bash"],
            "deny": deny,
        }
    }


def write_settings(ladder, config, path):
    """Serialises settings_document_for(ladder, config) as JSON to path."""
    with open(path, "w") as f:
        json.dump(settings_document_for(ladder, config), f, indent=2)
        f.write("\n")


""" PREAMBLE GENERATION """

# Copied byte-for-byte from bench/loomyard-eval/README.md's committed
# preambles, which every rung's prompt reuses verbatim.
PARALLEL_OPENING = """USE PARALLEL TOOL CALLS. Whenever you have more than one independent thing to
read or check, issue ALL of those tool calls together in the SAME turn --
never one at a time across separate turns. This is not optional."""

PARALLEL_BLOCK = """<use_parallel_tool_calls>
For maximum efficiency, whenever you need to perform multiple independent
operations, invoke all relevant tools simultaneously rather than
sequentially. Prioritize calling tools in parallel whenever possible. For
example, once you know several independent locations to read or check (e.g.
a list of caller locations from a single lookup), issue all of those Read or
Bash calls together in one turn rather than one at a time across separate
turns -- each turn costs a full round of model latency regardless of how
fast the underlying tool executes, so batching directly cuts wall-clock and
token cost. Err on the side of maximizing parallel tool calls rather than
running too many tools sequentially. Only batch tool calls that are
independent of each other -- two Read calls at two locations you already
know about are never dependent on each other.
</use_parallel_tool_calls>"""

# The committed Agent B preamble's own body, between PARALLEL_OPENING and
# PARALLEL_BLOCK. Ends at the <TARGET_DIR> paragraph and excludes the
# committed template's own <TASK TEXT> placeholder line -- preamble_for
# appends the task text itself, so carrying the placeholder through would
# emit both the placeholder and the text into every control run's prompt.
B_PREAMBLE_BODY = """You are working on a code task in the codebase at <TARGET_DIR>. You have
standard tools: Read, Grep, Bash, Glob. Explore as needed to answer
thoroughly and correctly."""

_CLOSING_SENTENCE = """When you are completely done, end your final message with ONLY a fenced json
code block matching the schema below -- no other trailing prose after it."""

# One-line description of each canonical tool's job, for the generated
# MCP-shaped preamble body. Written without any binary path, shell verb
# syntax, or --prefixed flag -- every rung's tool is a call, not a CLI verb.
_TOOL_DESCRIPTIONS = {
    "toc_dir": "lists every source file directly in a directory (not recursive) with its package, header comment, and test/generated flags",
    "toc_file": "returns the table of contents for one file: every function, method, and type with its signature, docstring, and precise line range",
    "textDocument_definition": "jumps to a symbol's definition, LSP-resolved",
    "textDocument_references": "finds every reference to a symbol, LSP-resolved -- including interface-dispatched calls a text search cannot see",
    "workspace_symbol": "searches for a symbol by name across the whole target codebase",
    "impact": "resolves a symbol, finds its callers, and reports each caller's full enclosing-function line range",
    "assert_no_callers": "fails if the symbol has callers outside its declaration",
}


def _mcp_preamble_body(ladder, config, target_dir):
    """Builds the body of a freshly generated MCP-shaped preamble for a
    quarry rung: names config's allowed tools by their client-side name and
    nothing else, then carries over the three exposure-independent
    instructions from the committed Agent A template."""
    tool_lines = "\n".join(
        f"- {mcp_name(tool)} -- {_TOOL_DESCRIPTIONS[tool]}" for tool in ladder.quarry_tools if tool in config.allowed
    )
    return f"""You are working on a code task in the codebase at {target_dir}. You have
access to the following code-navigation tools, each a call taking a
call-wide input with a `targets` array:

{tool_lines}

Never set targetDir or buildTags on any of these calls -- the server is
already rooted at the correct target codebase.

Use these tools as your PRIMARY tool for anything they cover: symbol
lookups, "who calls this / where is this defined", file/directory surveys,
and caller-impact analysis. Do NOT reach for grep/ripgrep as a reflex, and
do NOT use it to "double-check" a question one of these tools has already
answered -- that defeats the point of having it and just spends tokens
re-deriving what you already know.

If you already know a symbol's declaration line -- e.g. from an earlier
table-of-contents call, which gives you every symbol's exact line range up
front -- call the matching tool with that `file:line:character` position
directly instead of the bare symbol name. A bare name triggers a
project-wide symbol search that is often genuinely ambiguous, and costs you
a second round trip to disambiguate with the position you already had."""


def preamble_for(ladder, config, target_dir, task_text, schema_json):
    """
    Returns the full prompt string for one run of config against target_dir.

    When config.allowed is empty, reproduces the committed Agent B preamble
    exactly. Otherwise generates a freshly written MCP-shaped preamble
    naming config's allowed tools by their mcp__quarry__* client-side names.
    Both shapes share PARALLEL_OPENING, PARALLEL_BLOCK, the closing
    schema-only-output sentence, and schema_json.
    """
    if config.allowed:
        body = _mcp_preamble_body(ladder, config, target_dir)
    else:
        body = B_PREAMBLE_BODY.replace("<TARGET_DIR>", target_dir)

    return "\n\n".join([PARALLEL_OPENING, body, task_text, PARALLEL_BLOCK, _CLOSING_SENTENCE, schema_json])
