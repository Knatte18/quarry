"""
Per-run validation gates for the quarry-mcp capability ladder suite: pure
predicates over a parsed transcript, a filesystem state, and a run
directory. Each gate returns a list of `GateFinding`s rather than raising,
so one run can fail several gates and report all of them at once;
`run_gates` composes the individual predicates into one `GateReport`.
`run_ladder.py` calls `run_gates` once per run and writes `run.json` only
when the report passes -- it never re-implements a gate itself.

Usage:
    from gates import run_gates, GateReport
    report = run_gates(events, ladder, config, run_model, repo_root,
                        worktree, run_dir, cache_dir, env)
    if not report.passed:
        ...
"""
import hashlib
import json
import os
import re
import shutil
import subprocess
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

from ladder_config import DAEMON_BACKED_TOOLS, MCP_PREFIX, deny_list_for, mcp_name


class GateError(Exception):
    """
    Raised when a gate is asked to resolve state it cannot resolve safely:
    an environment mapping that still carries QUARRY_STATE_DIR or
    QUARRY_BUILD_TAGS (see resolve_state_dir), or a daemon pid that outlives
    the caller's own wait bound (see wait_for_daemon_exit).
    """


@dataclass(frozen=True)
class GateFinding:
    """
    One observation from a single gate predicate.

    Instance variables:
        gate: short name identifying which predicate produced this finding.
        message: human-readable detail.
        fatal: True when this finding alone should invalidate the run.
    """

    gate: str
    message: str
    fatal: bool


@dataclass(frozen=True)
class GateReport:
    """
    The composed result of every gate run_gates applied to one run.

    Instance variables:
        findings: tuple of every GateFinding produced, fatal and non-fatal
            alike.
    """

    findings: tuple

    @property
    def passed(self):
        """True only when no finding in this report is fatal."""
        return not any(finding.fatal for finding in self.findings)


""" TRANSCRIPT-DERIVED GATES """


def _tool_use_blocks(events):
    """Yields (tool_use_id, name, input_dict) for every tool_use block in
    every assistant event, in transcript order."""
    for event in events:
        if event.get("type") != "assistant":
            continue
        for block in event["message"]["content"]:
            if block.get("type") == "tool_use":
                yield block["id"], block["name"], block.get("input", {})


def _tool_results_by_id(events):
    """Maps tool_use_id to its matching tool_result block, across every
    user event's content."""
    results = {}
    for event in events:
        if event.get("type") != "user":
            continue
        for block in event.get("message", {}).get("content", []):
            if block.get("type") == "tool_result":
                results[block["tool_use_id"]] = block
    return results


def gate_denied_tools_not_used(events, denied_names):
    """
    Fatal when any tool_use block names a tool in denied_names and its
    matching tool_result did not error. A denied name that appears only as
    a rejected attempt -- surfaced in the result event's permission_denials,
    never as a tool_use block at all -- is not a violation; it is the
    denied_tool_attempts metric extract_usage.py already counts.
    """
    findings = []
    results_by_id = _tool_results_by_id(events)
    for tool_use_id, name, _tool_input in _tool_use_blocks(events):
        if name not in denied_names:
            continue
        result = results_by_id.get(tool_use_id)
        errored = bool(result and result.get("is_error"))
        if not errored:
            findings.append(
                GateFinding(
                    "denied_tools_not_used",
                    f"denied tool {name!r} was called and did not error",
                    fatal=True,
                )
            )
    return findings


def gate_no_target_override(events):
    """
    Fatal when any mcp__quarry__* tool call's input carries a targetDir or
    a buildTags key. A run that retargets breaks both the pinned-worktree
    constraint and the cold cell's daemon key.
    """
    findings = []
    for _tool_use_id, name, tool_input in _tool_use_blocks(events):
        if not name.startswith(MCP_PREFIX):
            continue
        for key in ("targetDir", "buildTags"):
            if key in tool_input:
                findings.append(
                    GateFinding(
                        "no_target_override",
                        f"{name} call carried a {key!r} key",
                        fatal=True,
                    )
                )
    return findings


# Strips a trailing bracketed context-window suffix, e.g. "[1m]", from a
# reported model id -- see gate_model_pinned.
_CONTEXT_WINDOW_SUFFIX_RE = re.compile(r"\[[^\]]*\]$")


def _normalise_model_id(model_id):
    """Drops a trailing bracketed context-window suffix from a model id, so
    a pinned "claude-opus-5" matches a reported "claude-opus-5[1m]"."""
    return _CONTEXT_WINDOW_SUFFIX_RE.sub("", model_id)


def gate_model_pinned(events, run_model):
    """
    Fatal when the init event's model does not match the pinned run_model,
    after normalising away a trailing bracketed context-window suffix on
    the reported id.
    """
    init = next(
        event for event in events if event.get("type") == "system" and event.get("subtype") == "init"
    )
    reported_model = init.get("model")
    if _normalise_model_id(reported_model) != run_model:
        return [
            GateFinding(
                "model_pinned",
                f"init event reported model {reported_model!r}, pinned run_model is {run_model!r}",
                fatal=True,
            )
        ]
    return []


def _redact_tool_result_content(events):
    """
    Returns a copy of events with every tool_result block's content
    replaced by a placeholder, so gate_blinding can distinguish a token
    that appears only inside a tool_result payload from one that appears
    anywhere else in the transcript.
    """
    redacted_events = []
    for event in events:
        if event.get("type") != "user":
            redacted_events.append(event)
            continue
        message = dict(event.get("message", {}))
        content = []
        for block in message.get("content", []):
            if block.get("type") == "tool_result":
                block = dict(block)
                block["content"] = "REDACTED"
            content.append(block)
        message["content"] = content
        redacted_event = dict(event)
        redacted_event["message"] = message
        redacted_events.append(redacted_event)
    return redacted_events


def gate_blinding(events, repo_root):
    """
    Applies only to a config whose allowed is empty (the caller's job to
    decide -- see run_gates). Fatal when the transcript contains an
    mcp__quarry__ tool name, the literal "/tmp/quarry-bench", or any
    filesystem path into repo_root.

    A bare "quarry" mention confined to a tool_result payload is not
    fatal: it records a non-fatal target_origin_quarry_mention finding
    instead, because the target codebase mentions quarry in its own
    tracked files and a bare-string gate would halt the matrix over the
    target's own prose. A "quarry" mention found anywhere else in the
    transcript -- outside a tool_result -- is fatal, since nothing besides
    a tool_result should ever have surfaced that word to a blinded agent.
    """
    findings = []
    full_text = json.dumps(events)

    if "mcp__quarry__" in full_text:
        findings.append(
            GateFinding("blinding", "transcript contains an mcp__quarry__ tool name", fatal=True)
        )
    if "/tmp/quarry-bench" in full_text:
        findings.append(
            GateFinding("blinding", "transcript contains the literal /tmp/quarry-bench", fatal=True)
        )
    repo_root_str = str(repo_root)
    if repo_root_str and repo_root_str in full_text:
        findings.append(
            GateFinding(
                "blinding",
                f"transcript contains a filesystem path into repo_root ({repo_root_str})",
                fatal=True,
            )
        )
    if any(finding.fatal for finding in findings):
        return findings

    if "quarry" in full_text.lower():
        non_tool_result_text = json.dumps(_redact_tool_result_content(events))
        if "quarry" in non_tool_result_text.lower():
            findings.append(
                GateFinding("blinding", "quarry mentioned outside a tool_result payload", fatal=True)
            )
        else:
            findings.append(
                GateFinding(
                    "target_origin_quarry_mention",
                    "bare quarry mention confined to a tool_result payload",
                    fatal=False,
                )
            )
    return findings


""" FILESYSTEM AND DAEMON-STATE GATES """


def workspace_key(target_dir):
    """
    Re-derives quarry's own workspaceKey (internal/cli/paths.go): the
    target directory's base name, a hyphen, then the first 12 hex
    characters of the SHA-256 digest of the cleaned absolute path.
    """
    cleaned_abs = os.path.normpath(os.path.abspath(target_dir))
    digest = hashlib.sha256(cleaned_abs.encode()).digest()[:6].hex()
    return f"{os.path.basename(target_dir)}-{digest}"


def user_cache_dir():
    """
    Reproduces Go's os.UserCacheDir semantics explicitly -- the base
    directory quarry's own default state-dir tier resolves to: $XDG_CACHE_HOME
    when set and non-empty, otherwise ~/.cache. This is the only place the
    production cache_dir argument is derived; no other function in this
    suite hand-writes that path.
    """
    xdg_cache_home = os.environ.get("XDG_CACHE_HOME")
    if xdg_cache_home:
        return xdg_cache_home
    return os.path.join(os.path.expanduser("~"), ".cache")


def resolve_state_dir(target_dir, cache_dir, env):
    """
    <cache_dir>/quarry/<workspace_key>, matching the third precedence tier
    of quarry's ResolveStateDir. The suite never passes --state-dir and
    always clears QUARRY_STATE_DIR, so the two higher tiers are
    deliberately not modelled; it also models no tags-<hex> segment,
    because the suite clears QUARRY_BUILD_TAGS and never sets buildTags on
    a call.

    Takes the environment it is resolving for as an explicit env argument
    rather than reading os.environ, and raises GateError when that mapping
    carries either variable -- this is what stops the gate being asked
    about an environment the runs were not launched with, which would
    silently resolve a key that is not the one in use. The harness's own
    process may legitimately have either variable exported; that is not
    what this function resolves against.
    """
    if env.get("QUARRY_STATE_DIR"):
        raise GateError("resolve_state_dir: env carries QUARRY_STATE_DIR, which the suite always clears")
    if env.get("QUARRY_BUILD_TAGS"):
        raise GateError("resolve_state_dir: env carries QUARRY_BUILD_TAGS, which the suite always clears")
    return os.path.join(cache_dir, "quarry", workspace_key(target_dir))


def daemon_state_file(state_dir, lang="go"):
    """<state_dir>/<lang>/daemon.json, mirroring quarry's DaemonStateFile."""
    return os.path.join(state_dir, lang, "daemon.json")


def _pid_alive(pid):
    """True when a process with this pid exists, per os.kill(pid, 0)
    semantics: ESRCH means gone, EPERM means it exists but is owned by
    someone else, anything else propagates."""
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    return True


def _read_daemon_state(target_dir, cache_dir, env, lang="go"):
    """Returns the parsed daemon.json mapping at the resolved location, or
    None when it does not exist."""
    state_dir = resolve_state_dir(target_dir, cache_dir, env)
    path = daemon_state_file(state_dir, lang)
    if not os.path.exists(path):
        return None
    with open(path) as f:
        return json.load(f)


def daemon_alive(target_dir, cache_dir, env, lang="go"):
    """
    True only when a daemon.json exists at the resolved location and the
    pid it records is alive. This is the suite's definition of "a daemon
    is running", mirroring quarry's own daemonStale, which likewise treats
    a state file whose pid is dead as stale rather than as a live daemon.
    """
    state = _read_daemon_state(target_dir, cache_dir, env, lang)
    return state is not None and _pid_alive(state["pid"])


def gate_cold_before(target_dir, cache_dir, env):
    """
    Fatal when daemon_alive is True before a cold run starts, since the
    daemon is already warm and the run cannot be reported as cold.

    Keys on liveness rather than on daemon.json's mere existence, because
    neither daemon.json nor the state directory is removed when a daemon
    exits -- only daemon.sock is, and only by the next spawn's stale-socket
    cleanup. A presence check would make every retry at the same worktree
    path fail deterministically once a prior attempt left a state file
    behind; card 21's per-attempt state-directory clear makes the
    precondition deterministic, and this liveness definition is what keeps
    the gate correct even when a stale file survives it.
    """
    if daemon_alive(target_dir, cache_dir, env):
        return [
            GateFinding(
                "cold_before",
                "a daemon is already alive at this worktree's state directory before the cold run started",
                fatal=True,
            )
        ]
    return []


def used_daemon_backed_tool(events):
    """
    True when the transcript contains at least one tool_use block whose
    name is mcp_name(t) for a t in DAEMON_BACKED_TOOLS. toc_file and
    toc_dir are deliberately excluded: their handlers reach the
    tree-sitter path directly and never EnsureServer, so a toc call starts
    no daemon and writes no state.
    """
    daemon_backed_names = {mcp_name(tool) for tool in DAEMON_BACKED_TOOLS}
    return any(name in daemon_backed_names for _tool_use_id, name, _tool_input in _tool_use_blocks(events))


def gate_cold_after(events, target_dir, cache_dir, env):
    """
    Three outcomes, not two:
      - used_daemon_backed_tool is False: the gate does not apply; returns
        a single non-fatal cold_no_daemon_backed_call observation, and the
        run stands as valid while carrying no warmth signal.
      - used_daemon_backed_tool is True and daemon.json exists: passes --
        the positive confirmation the connection was supervised.
      - used_daemon_backed_tool is True and no daemon.json exists: fatal --
        the native fallback was taken, on which path the shared daemon
        address is not a function of the state directory at all, so the
        run is invalidated rather than reported as cold.
    """
    if not used_daemon_backed_tool(events):
        return [
            GateFinding(
                "cold_no_daemon_backed_call",
                "no daemon-backed tool call observed; cold cell carries no warmth signal for this run",
                fatal=False,
            )
        ]
    state = _read_daemon_state(target_dir, cache_dir, env)
    if state is not None:
        return []
    return [
        GateFinding(
            "cold_after",
            "a daemon-backed tool was used but no daemon.json exists; the native fallback was taken",
            fatal=True,
        )
    ]


def daemon_pid(target_dir, cache_dir, env, lang="go"):
    """The pid field of the resolved daemon.json, or None when the file is
    absent."""
    state = _read_daemon_state(target_dir, cache_dir, env, lang)
    return None if state is None else state["pid"]


def clear_state_dir(target_dir, cache_dir, env):
    """
    Removes the resolved state directory entirely. Called by the cold
    driver before each attempt so the before-and-after assertions read a
    directory this suite put in a known state, rather than one carrying a
    previous attempt's leftovers.
    """
    state_dir = resolve_state_dir(target_dir, cache_dir, env)
    shutil.rmtree(state_dir, ignore_errors=True)


# Poll interval for wait_for_daemon_exit's liveness loop -- short enough to
# keep the suite's own wait bounded tightly to the daemon's actual exit,
# without busy-spinning.
_DAEMON_EXIT_POLL_INTERVAL_S = 0.1


def wait_for_daemon_exit(target_dir, cache_dir, env, timeout_s, lang="go"):
    """
    Polls that pid's liveness with os.kill(pid, 0) until the process is
    gone, returning immediately when daemon_pid is None. Raises GateError
    naming the pid and the bound when timeout_s elapses first.

    The pid is the liveness signal because neither daemon.json nor the
    state directory is removed on exit -- only daemon.sock is -- so file
    presence says nothing about whether the daemon is still running, while
    daemon.json's recorded pid is exactly what quarry's own staleness
    check reads. Callers pass a bound derived from the daemon's own
    10-minute idle timeout plus a margin.
    """
    pid = daemon_pid(target_dir, cache_dir, env, lang)
    if pid is None:
        return
    deadline = time.monotonic() + timeout_s
    while _pid_alive(pid):
        if time.monotonic() >= deadline:
            raise GateError(f"daemon pid {pid} did not exit within {timeout_s}s")
        time.sleep(_DAEMON_EXIT_POLL_INTERVAL_S)


def gate_worktree_neutralised(worktree):
    """Fatal when CLAUDE.md, CONSTRAINTS.md, or .claude/ exists in the task
    worktree."""
    findings = []
    worktree = Path(worktree)
    for name in ("CLAUDE.md", "CONSTRAINTS.md", ".claude"):
        if (worktree / name).exists():
            findings.append(
                GateFinding(
                    "worktree_neutralised",
                    f"{name} is present in the task worktree",
                    fatal=True,
                )
            )
    return findings


def observe_worktree_dirtied(worktree):
    """
    Returns a non-fatal finding carrying True or False from
    `git -C <worktree> status --porcelain` being non-empty. Recorded,
    never gated. Called from inside run_gates, which runs before the
    worktree is restored; an observation taken after restore_worktree
    would be False for every run, since the restore is precisely what
    erases the evidence.
    """
    result = subprocess.run(
        ["git", "-C", str(worktree), "status", "--porcelain"],
        capture_output=True,
        text=True,
        check=True,
    )
    dirtied = bool(result.stdout.strip())
    return GateFinding("worktree_dirtied", f"worktree dirtied: {dirtied}", fatal=False)


""" RUN-STATE, INVALIDATION, AND THE COMPOSED GATE REPORT """


def run_dir(results_root, config_id, n):
    """<results_root>/raw/<config_id>/<n>/"""
    return Path(results_root) / "raw" / config_id / str(n)


def is_complete(a_run_dir):
    """
    True only when run.json exists in that directory and parses with
    state == "complete". A directory holding answer.json and usage.json
    but no score.json is by construction not complete, because run.json is
    written last.
    """
    run_json_path = Path(a_run_dir) / "run.json"
    if not run_json_path.exists():
        return False
    with open(run_json_path) as f:
        payload = json.load(f)
    return payload.get("state") == "complete"


def invalidate(a_run_dir):
    """
    Deletes run.json, then moves the directory aside to a sibling
    <n>.invalid-<k>/ with k the lowest unused index, leaving the discarded
    attempt inspectable. Returns the new path.
    """
    a_run_dir = Path(a_run_dir)
    run_json_path = a_run_dir / "run.json"
    if run_json_path.exists():
        run_json_path.unlink()

    k = 1
    while True:
        candidate = a_run_dir.parent / f"{a_run_dir.name}.invalid-{k}"
        if not candidate.exists():
            break
        k += 1
    a_run_dir.rename(candidate)
    return candidate


def write_run_json(a_run_dir, payload):
    """
    Writes the terminal-state marker to run.json: payload (the config id,
    the repetition index, the resolved run model, and the gate report's
    non-fatal observations, as assembled by the caller) plus a stamped
    state: "complete" and a UTC timestamp. Called only after the answer
    parsed, usage.json was extracted, every fatal gate passed, and
    score.json exists.
    """
    record = dict(payload)
    record["state"] = "complete"
    record["timestamp"] = datetime.now(timezone.utc).isoformat()

    a_run_dir = Path(a_run_dir)
    a_run_dir.mkdir(parents=True, exist_ok=True)
    with open(a_run_dir / "run.json", "w") as f:
        json.dump(record, f, indent=2)
        f.write("\n")
    return record


def run_gates(events, ladder, config, run_model, repo_root, worktree, a_run_dir, cache_dir, env):
    """
    Composes the transcript gates and the filesystem/daemon-state gates
    into one GateReport.

    Takes ladder because gate_denied_tools_not_used needs that config's
    denied names and ladder_config.deny_list_for(ladder, config) is the
    suite's only derivation of them; passing a precomputed list instead
    would invite a second derivation site, which is exactly what the
    single-source deny-list decision exists to prevent. Takes env because
    resolve_state_dir resolves against the environment the run was
    launched with, not the harness's own.

    Applies gate_blinding only when config.allowed is empty, and the
    cold-cell gate (gate_cold_after) only when config.cold is true --
    gate_cold_before is a separate precondition the caller checks before
    starting an attempt, not part of this composed report.

    a_run_dir is accepted for signature symmetry with
    gate_run_complete_artifacts's own run_dir argument, but is not
    inspected here: this function runs before scoring, so it requires only
    answer.json and usage.json on disk and never asserts on a scoring
    artifact -- that is gate_run_complete_artifacts's job, invoked
    separately after scoring completes.
    """
    del a_run_dir  # see docstring: accepted for signature symmetry only.

    denied_names = deny_list_for(ladder, config)
    findings = []
    findings.extend(gate_denied_tools_not_used(events, denied_names))
    findings.extend(gate_no_target_override(events))
    findings.extend(gate_model_pinned(events, run_model))
    if not config.allowed:
        findings.extend(gate_blinding(events, repo_root))
    findings.extend(gate_worktree_neutralised(worktree))
    findings.append(observe_worktree_dirtied(worktree))
    if config.cold:
        findings.extend(gate_cold_after(events, worktree, cache_dir, env))

    return GateReport(findings=tuple(findings))


def gate_run_complete_artifacts(a_run_dir):
    """
    A separate, later gate requiring all four of answer.json,
    answer.redacted.json, usage.json, and score.json. Deliberately not
    part of run_gates: two of those files are written by the scorer, which
    runs after run_gates, so folding it in would make every run fail a
    fatal gate before the scorer had a chance to write them. The caller
    invokes this one after scoring and immediately before write_run_json.
    """
    a_run_dir = Path(a_run_dir)
    findings = []
    for name in ("answer.json", "answer.redacted.json", "usage.json", "score.json"):
        if not (a_run_dir / name).exists():
            findings.append(
                GateFinding(
                    "run_complete_artifacts",
                    f"{name} missing from {a_run_dir}",
                    fatal=True,
                )
            )
    return findings
