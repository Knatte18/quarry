"""
The quarry-mcp capability ladder harness's entry point: builds and adopts
the two disposable task worktrees, builds the quarry-mcp server binary,
probes permission-deny semantics once before any paid run, executes the
sequential 42-run main matrix with resume and a three-attempt cap, then
runs the 3-run cold-daemon comparison cell last.

Every git call and every `claude` subprocess dispatch goes through one seam
each (`git=run_git`, `executor=launch_run`), so the pure planning and
resume logic -- run ordering, attempt accounting, the resume skip
decision, argv assembly -- is testable without invoking a model or
building a real worktree. The dispatch layer itself is exercised by
actually running the matrix, never by a mock.

Usage:
    python run_ladder.py bench/loomyard-eval/ladder/ladder.yaml \
        bench/loomyard-eval/ladder/results/2026-08-29 --stage all
"""
import json
import os
import re
import select
import shutil
import subprocess
import time
from dataclasses import dataclass
from pathlib import Path

from extract_usage import extract_usage, init_event, read_transcript, result_event
from gates import (
    GateFinding,
    daemon_state_file,
    gate_run_complete_artifacts,
    gate_worktree_neutralised,
    resolve_state_dir,
    run_gates,
    write_run_json,
)
from ladder_config import mcp_name, preamble_for, write_settings
from score_run import ScoringError, score_run


class HarnessError(Exception):
    """
    Raised when the harness cannot proceed safely: a stale worktree at a
    declared path, a worktree adopted at the wrong pin, a failed server
    build, a malformed or timed-out MCP call, a warm-up that left no
    daemon.json behind, a preflight probe whose denial did not block, a
    truncated run, or an exhausted attempt cap.
    """


""" TASK WORKTREE LIFECYCLE """


def run_git(args):
    """
    The single seam every git call in this module goes through: runs
    `git <args>` and returns its stdout. Every function below takes it as
    a `git=run_git` default parameter so tests can drive them against an
    injected runner without creating a real worktree.
    """
    result = subprocess.run(["git", *args], capture_output=True, text=True, check=True)
    return result.stdout


def neutralise_worktree(path):
    """
    Deletes CLAUDE.md, CONSTRAINTS.md, and .claude/ from the disposable
    worktree at path. This is a mutation of the disposable checkout only;
    the live source checkout is never touched.
    """
    path = Path(path)
    for name in ("CLAUDE.md", "CONSTRAINTS.md"):
        target = path / name
        if target.exists():
            target.unlink()
    claude_dir = path / ".claude"
    if claude_dir.exists():
        shutil.rmtree(claude_dir)


def build_worktree(ladder, path, sha, git=run_git):
    """
    Builds one disposable task worktree at path, pinned to sha, off
    ladder.source_repo: `git -C <source_repo> worktree add <path> <sha>`,
    then neutralise_worktree, then an assertion that
    gate_worktree_neutralised passes.

    Raises HarnessError when a directory already exists at path, so a
    stale worktree is never silently reused. ensure_task_worktrees is the
    idempotent caller; nothing else calls this directly.
    """
    path = Path(path)
    if path.exists():
        raise HarnessError(f"build_worktree: a directory already exists at {path} -- refusing to reuse a stale worktree")

    git(["-C", ladder.source_repo, "worktree", "add", str(path), sha])
    neutralise_worktree(path)

    findings = gate_worktree_neutralised(path)
    if findings:
        raise HarnessError(
            f"build_worktree: {path} failed gate_worktree_neutralised: {[f.message for f in findings]}"
        )


def restore_worktree(path, git=run_git):
    """
    Restores a task worktree to its pinned commit after a run: `git -C
    <path> reset --hard` followed by `git -C <path> clean -fdx`, then
    neutralise_worktree again, since `clean -fdx` restores the
    ambient-context files the neutralisation removed. Called
    unconditionally after every main-matrix run.
    """
    git(["-C", str(path), "reset", "--hard"])
    git(["-C", str(path), "clean", "-fdx"])
    neutralise_worktree(path)


def remove_worktree(ladder, path, git=run_git):
    """Removes a disposable task worktree: `git -C <source_repo> worktree
    remove --force <path>`."""
    git(["-C", ladder.source_repo, "worktree", "remove", "--force", str(path)])


def ensure_task_worktrees(ladder, git=run_git):
    """
    Returns a mapping from task key to worktree path, idempotently,
    because the harness is re-invoked to resume and this runs on every
    invocation.

    For each task: when no directory exists at the declared path,
    build_worktree it. When one does exist, read `git -C <path>
    rev-parse HEAD` -- if it equals the task's declared pin, adopt the
    existing worktree by calling restore_worktree on it and continue; if
    it does not, raise HarnessError naming both SHAs, since a worktree at
    the wrong pin would silently benchmark a different codebase.
    """
    worktrees = {}
    for task_key, task in ladder.tasks.items():
        path = Path(task.worktree)
        if not path.exists():
            build_worktree(ladder, path, task.pinned_sha, git=git)
        else:
            head = git(["-C", str(path), "rev-parse", "HEAD"]).strip()
            if head != task.pinned_sha:
                raise HarnessError(
                    f"ensure_task_worktrees: {path} is at {head!r}, expected the declared pin {task.pinned_sha!r}"
                )
            restore_worktree(path, git=git)
        worktrees[task_key] = path
    return worktrees


""" SERVER BINARY, PER-RUN MCP CONFIG, AND WARM-UP """


def build_server(repo_root):
    """
    Builds the quarry-mcp server binary at <repo_root>/quarry-mcp with
    CGO_ENABLED=1, returning its absolute path.

    The warm-start path (a built binary) is used rather than the
    committed `go run ./cmd/quarry-mcp` form so a cold build cache cannot
    make a run's first connection exceed the client's connect timeout.
    Raises HarnessError with the compiler output when the build fails,
    naming the CGO toolchain requirement, since a missing C toolchain
    fails at compile time (the toc verbs' tree-sitter backend links C
    grammars).
    """
    repo_root = Path(repo_root)
    binary_path = repo_root / "quarry-mcp"
    build_env = dict(os.environ)
    build_env["CGO_ENABLED"] = "1"

    result = subprocess.run(
        ["go", "build", "-o", str(binary_path), "./cmd/quarry-mcp"],
        cwd=str(repo_root),
        env=build_env,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise HarnessError(
            "build_server: go build ./cmd/quarry-mcp failed -- requires CGO_ENABLED=1 "
            f"with a C toolchain:\n{result.stderr}"
        )
    return str(binary_path.resolve())


def mcp_config_document(server_path, target_dir):
    """
    The --mcp-config mapping declaring a single server named "quarry",
    whose command is the built binary's absolute path and whose args
    carry an explicit --target-dir <target_dir>.
    """
    return {
        "mcpServers": {
            "quarry": {
                "command": str(server_path),
                "args": ["--target-dir", str(target_dir)],
            }
        }
    }


def write_run_inputs(ladder, config, run_dir, target_dir, server_path):
    """
    Materialises one run's launch inputs into run_dir: its settings.json
    always, and its mcp.json only when config.allowed is non-empty.

    A config whose allowed set is empty gets no MCP config file and is
    launched with no --mcp-config at all: the quarry server is never
    declared to it, because a declared server named "quarry" exposing an
    mcp__quarry__* namespace is itself the structural leak the blinding
    forbids.
    """
    run_dir = Path(run_dir)
    run_dir.mkdir(parents=True, exist_ok=True)
    write_settings(ladder, config, run_dir / "settings.json")
    if config.allowed:
        with open(run_dir / "mcp.json", "w") as f:
            json.dump(mcp_config_document(server_path, target_dir), f, indent=2)
            f.write("\n")


_MCP_PROTOCOL_VERSION = "2024-11-05"


def _write_jsonrpc_message(stream, message):
    """Writes one JSON-RPC message as a single line, matching the
    newline-delimited framing cmd/quarry-mcp/main.go's mcp.StdioTransport{}
    speaks."""
    stream.write(json.dumps(message))
    stream.write("\n")
    stream.flush()


def _readline_with_timeout(stream, timeout_s):
    """
    Returns one line off stream, waiting at most timeout_s.

    Returns None when nothing became readable within timeout_s, and ""
    when the stream was readable but at EOF (the server closed stdout).
    """
    ready, _writable, _errored = select.select([stream], [], [], timeout_s)
    if not ready:
        return None
    return stream.readline()


def mcp_call(server_path, target_dir, tool, arguments, env, timeout_s):
    """
    A minimal MCP stdio client, standard library only.

    Spawns the server binary rooted at target_dir with pipes on stdin and
    stdout; writes a JSON-RPC "initialize" request, then an "initialized"
    notification, then a "tools/call" request naming tool and arguments,
    each as one JSON object followed by a newline; reads newline-delimited
    JSON objects back off stdout until the response whose id matches the
    request arrives; then closes stdin and terminates the process.

    Raises HarnessError on a JSON-RPC error response, on a malformed
    line, on the server closing stdout before a matching response
    arrives, or on timeout_s elapsing.
    """
    argv = [str(server_path), "--target-dir", str(target_dir)]
    proc = subprocess.Popen(
        argv, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=env
    )
    try:
        call_id = 2
        _write_jsonrpc_message(
            proc.stdin,
            {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": _MCP_PROTOCOL_VERSION,
                    "capabilities": {},
                    "clientInfo": {"name": "loomyard-eval-ladder", "version": "1.0"},
                },
            },
        )
        _write_jsonrpc_message(proc.stdin, {"jsonrpc": "2.0", "method": "notifications/initialized"})
        _write_jsonrpc_message(
            proc.stdin,
            {"jsonrpc": "2.0", "id": call_id, "method": "tools/call", "params": {"name": tool, "arguments": arguments}},
        )

        deadline = time.monotonic() + timeout_s
        while True:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise HarnessError(f"mcp_call: {tool} timed out after {timeout_s}s waiting for a response")
            line = _readline_with_timeout(proc.stdout, remaining)
            if line is None:
                continue
            if line == "":
                raise HarnessError(f"mcp_call: {tool}: server closed stdout before a matching response arrived")
            line = line.strip()
            if not line:
                continue
            try:
                message = json.loads(line)
            except json.JSONDecodeError as exc:
                raise HarnessError(f"mcp_call: malformed line from quarry-mcp server: {exc}") from exc
            if message.get("id") == call_id:
                if "error" in message:
                    raise HarnessError(f"mcp_call: {tool} returned a JSON-RPC error: {message['error']}")
                return message.get("result")
    finally:
        proc.stdin.close()
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait()


WARM_UP_TOOL = "workspace_symbol"

_WARM_UP_TIMEOUT_S = 60


def warm_daemon(server_path, target_dir, env, cache_dir):
    """
    Pre-warms the daemon for one main-matrix run: one mcp_call against
    WARM_UP_TOOL, then an assertion that a daemon.json now exists at the
    state directory gates.resolve_state_dir derives for target_dir.

    The query needs no match: workspace_symbol's handler calls
    resolveCall once for the whole call before resolving any target, so
    the daemon starts whether or not the query resolves -- which is why
    the post-condition is the state file's existence and not the result
    payload. WARM_UP_TOOL must be daemon-backed: toc_file and toc_dir
    reach the tree-sitter path directly and never EnsureServer, so
    warming with a toc call would start no daemon at all. Called by
    run_matrix immediately before each main-matrix run's dispatch, per
    run rather than once per worktree, since the daemon self-expires
    after its idle timeout; never called for a config with cold: true.
    """
    mcp_call(server_path, target_dir, WARM_UP_TOOL, {"targets": [{"query": "Run"}]}, env, timeout_s=_WARM_UP_TIMEOUT_S)
    state_dir = resolve_state_dir(target_dir, cache_dir, env)
    state_file = daemon_state_file(state_dir)
    if not os.path.exists(state_file):
        raise HarnessError(
            f"warm_daemon: no daemon.json at {state_file} after warming {target_dir} -- "
            "the warm-up call did not start a daemon"
        )


def run_env():
    """
    The environment every subprocess this module launches inherits, with
    QUARRY_STATE_DIR and QUARRY_BUILD_TAGS removed.

    Both would move the resolved state directory off the per-path key the
    cold cell depends on: the first two take precedence over
    workspaceKey outright, and a non-empty tag set appends a
    "tags-<hex>" segment at every tier. QUARRY_CONFIG is deliberately
    not scrubbed: it selects the servers.yaml overlay naming the
    language-server command, and clearing it on a machine that needs an
    overlay would stop the server starting at all.
    """
    env = dict(os.environ)
    env.pop("QUARRY_STATE_DIR", None)
    env.pop("QUARRY_BUILD_TAGS", None)
    return env


""" THE PREFLIGHT DENIAL PROBE """

# The one tool the probe denies -- an arbitrary daemon-backed choice, since
# what is being established is whether permissions.deny blocks at all, not
# whether this particular tool is safe.
_PROBE_DENIED_TOOL = "impact"


def _denied_call_succeeded(events, denied_name):
    """
    True when some tool_use block named denied_name has a matching
    tool_result that did not error -- i.e. the denial did not block it.
    """
    results_by_id = {}
    for event in events:
        if event.get("type") != "user":
            continue
        for block in event.get("message", {}).get("content", []):
            if block.get("type") == "tool_result":
                results_by_id[block["tool_use_id"]] = block

    for event in events:
        if event.get("type") != "assistant":
            continue
        for block in event["message"]["content"]:
            if block.get("type") == "tool_use" and block["name"] == denied_name:
                result = results_by_id.get(block["id"])
                if result and not result.get("is_error"):
                    return True
    return False


def run_probe(ladder, repo_root, results_root, target_dir, server_path):
    """
    Executes one throwaway claude -p run, before any matrix run, that
    declares the quarry server with mcp__quarry__impact denied and asks
    the agent to call it and report what happened.

    Writes probe.json into results_root recording denial_blocks (whether
    the denied call failed to succeed -- the load-bearing premise of the
    whole suite), denied_tools_advertised (whether the denied name
    appears in the init event's tools array), advertised_tools,
    session_id, the resolved model, the resolved QUARRY_CONFIG value, and
    the probe's own transcript path (captured under
    <results_root>/raw/probe/, the one subtree .gitignore covers).

    Raises HarnessError when denial_blocks is False -- the matrix halts
    before a single paid run, because every rung would silently be the
    full bundle. The probe runs with the same --setting-sources "",
    --strict-mcp-config, --max-turns, and non-interactive permission mode
    as every matrix run, so what it establishes is true of the matrix and
    not of a differently-configured invocation.
    """
    del repo_root  # accepted for signature symmetry with the rest of this module's dispatch functions; unused here.
    results_root = Path(results_root)
    probe_dir = results_root / "raw" / "probe"
    probe_dir.mkdir(parents=True, exist_ok=True)
    transcript_path = probe_dir / "transcript.jsonl"

    denied_name = mcp_name(_PROBE_DENIED_TOOL)
    settings_path = probe_dir / "settings.json"
    with open(settings_path, "w") as f:
        json.dump({"permissions": {"allow": ["Read", "Grep", "Glob", "Bash"], "deny": [denied_name, "Task"]}}, f, indent=2)
        f.write("\n")

    mcp_config_path = probe_dir / "mcp.json"
    with open(mcp_config_path, "w") as f:
        json.dump(mcp_config_document(server_path, target_dir), f, indent=2)
        f.write("\n")

    prompt = (
        f"Call the {denied_name} tool once, with any plausible arguments for it, and "
        "report exactly what happened when you called it -- whether it succeeded, was "
        "denied, or errored, quoting the tool result you received."
    )

    argv = [
        "claude", "-p", prompt,
        "--model", ladder.run_model,
        "--output-format", "stream-json",
        "--verbose",
        "--setting-sources", "",
        "--strict-mcp-config",
        "--settings", str(settings_path),
        "--permission-mode", "dontAsk",
        "--max-turns", str(ladder.max_turns),
        "--mcp-config", str(mcp_config_path),
    ]

    env = run_env()
    launch_run(argv, cwd=target_dir, env=env, transcript_path=transcript_path)
    events = read_transcript(transcript_path)

    init = init_event(events)
    advertised_tools = init.get("tools") or []
    denial_blocks = not _denied_call_succeeded(events, denied_name)

    probe_record = {
        "denial_blocks": denial_blocks,
        "denied_tools_advertised": denied_name in advertised_tools,
        "advertised_tools": advertised_tools,
        "session_id": init.get("session_id"),
        "model": init.get("model"),
        "quarry_config": env.get("QUARRY_CONFIG"),
        "transcript": str(transcript_path),
    }
    with open(results_root / "probe.json", "w") as f:
        json.dump(probe_record, f, indent=2)
        f.write("\n")

    if not denial_blocks:
        raise HarnessError(
            "run_probe: the denied tool call did not error -- permissions.deny does not "
            "block; halting before any matrix run"
        )
    return probe_record


""" EXECUTE ONE RUN END TO END """

# The exact heading text both task files use to introduce their identical
# task-text block -- the section boundary is load-bearing, not tidiness
# (see task_text_for's docstring).
TASK_TEXT_HEADING = "## `<TASK TEXT>` (identical for A, B, C)"


def task_text_for(ladder, task_key):
    """
    Extracts the task's task-text block from its committed task file, so
    the preamble is assembled from the tracked source rather than from a
    copy.

    Starts at the line whose heading text is TASK_TEXT_HEADING, takes
    every following line up to but not including the next line beginning
    with "## ", strips a leading "> " or ">" from each line, and strips
    surrounding blank lines. Over-reading this boundary is the worst
    failure this harness can have: task 01's very next section is its
    fasit leads, and task 04's is followed by its scoring notes naming
    the real callers and the burler.go:373 decoy outright -- an extractor
    that ran past the boundary would paste the answer key into every
    run's prompt.

    Raises HarnessError when the heading is absent or the extracted body
    is empty.
    """
    task = ladder.tasks[task_key]
    with open(task.task_file) as f:
        lines = f.read().splitlines()

    try:
        start = lines.index(TASK_TEXT_HEADING)
    except ValueError as exc:
        raise HarnessError(f"task_text_for: {task.task_file!r} has no {TASK_TEXT_HEADING!r} heading") from exc

    body_lines = []
    for line in lines[start + 1:]:
        if line.startswith("## "):
            break
        if line.startswith("> "):
            body_lines.append(line[2:])
        elif line.startswith(">"):
            body_lines.append(line[1:])
        else:
            body_lines.append(line)

    while body_lines and not body_lines[0].strip():
        body_lines.pop(0)
    while body_lines and not body_lines[-1].strip():
        body_lines.pop()

    if not body_lines:
        raise HarnessError(f"task_text_for: {task.task_file!r} extracted an empty task-text body")

    return "\n".join(body_lines)


_FENCED_JSON_BLOCK_RE = re.compile(r"```json\n.*?\n```", re.DOTALL)


def _first_fenced_json_block(text):
    """Returns the first ```json ... ``` fenced block found in text, fences
    included. Raises HarnessError when none is present."""
    match = _FENCED_JSON_BLOCK_RE.search(text)
    if match is None:
        raise HarnessError("schema_for: no fenced json block found in the expected section")
    return match.group(0)


def _section(text, heading):
    """Returns the text between heading (exclusive) and the next line
    starting with "## " (exclusive), or to the end of text when there is
    no next "## " line. Raises HarnessError when heading is absent."""
    idx = text.find(heading)
    if idx == -1:
        raise HarnessError(f"schema_for: no {heading!r} section found")
    section = text[idx + len(heading):]
    next_heading = section.find("\n## ")
    if next_heading != -1:
        section = section[:next_heading]
    return section


_IMPACT_SCHEMA_HEADING = "## Output schema (impact-analysis tasks)"
_EXPLORATION_SCHEMAS_HEADING = "## Output schemas"
_EXPLORATION_SCHEMA_MARKER = "**Exploration tasks:**"
_BENCHMARK_README_PATH = "bench/loomyard-eval/README.md"


def schema_for(ladder, task_key):
    """
    Returns the task's output schema (a fenced ```json ... ``` block, with
    fences), from the source that actually holds it -- which differs per
    task and is not uniform.

    The impact schema is in task 04's own "## Output schema
    (impact-analysis tasks)" section. Task 01 has no schema section at
    all, so the exploration schema comes from the "## Output schemas"
    section of the benchmark README, under its "Exploration tasks:"
    marker. Selection is driven by the task's declared schema field,
    never by guessing which file to read.

    Raises HarnessError when the named section is absent.
    """
    task = ladder.tasks[task_key]

    if task.schema == "impact":
        with open(task.task_file) as f:
            text = f.read()
        return _first_fenced_json_block(_section(text, _IMPACT_SCHEMA_HEADING))

    if task.schema == "exploration":
        with open(_BENCHMARK_README_PATH) as f:
            text = f.read()
        schemas_section = _section(text, _EXPLORATION_SCHEMAS_HEADING)
        marker_idx = schemas_section.find(_EXPLORATION_SCHEMA_MARKER)
        if marker_idx == -1:
            raise HarnessError(
                f"schema_for: {_BENCHMARK_README_PATH!r}'s {_EXPLORATION_SCHEMAS_HEADING!r} section "
                f"has no {_EXPLORATION_SCHEMA_MARKER!r} marker"
            )
        return _first_fenced_json_block(schemas_section[marker_idx:])

    raise HarnessError(f"schema_for: unknown schema {task.schema!r} for task {task_key!r}")


def build_argv(ladder, config, run_dir, target_dir):
    """
    Builds the full claude argv for one matrix run: -p, the generated
    preamble as the prompt, --model the pinned run model, --output-format
    stream-json, --verbose, --setting-sources "", --strict-mcp-config,
    --settings <run_dir>/settings.json, --permission-mode dontAsk,
    --max-turns from the ladder's max_turns field, and --mcp-config
    <run_dir>/mcp.json only when config.allowed is non-empty.

    Never includes --state-dir: the suite always resolves the state
    directory through the scrubbed environment, never the launch flag.
    """
    task_text = task_text_for(ladder, config.task)
    schema_json = schema_for(ladder, config.task)
    prompt = preamble_for(ladder, config, target_dir, task_text, schema_json)

    run_dir = Path(run_dir)
    argv = [
        "claude", "-p", prompt,
        "--model", ladder.run_model,
        "--output-format", "stream-json",
        "--verbose",
        "--setting-sources", "",
        "--strict-mcp-config",
        "--settings", str(run_dir / "settings.json"),
        "--permission-mode", "dontAsk",
        "--max-turns", str(ladder.max_turns),
    ]
    if config.allowed:
        argv += ["--mcp-config", str(run_dir / "mcp.json")]
    return argv


def launch_run(argv, cwd, env, transcript_path):
    """
    The default executor: the one place this module starts a claude
    subprocess.

    Captures the subprocess's stdout stream directly to transcript_path
    as it runs -- the OS pipes the child's writes straight to the open
    file, so a run that dies mid-way still leaves a diagnosable
    transcript. Exists as a named seam so tests drive an injected
    executor rather than a model.
    """
    with open(transcript_path, "w") as transcript_file:
        subprocess.run(argv, cwd=str(cwd), env=env, stdout=transcript_file, stderr=subprocess.PIPE, text=True)


_FENCED_JSON_ANSWER_RE = re.compile(r"```json\s*(.*?)\s*```", re.DOTALL)


def _extract_answer(result_payload):
    """
    Parses the final fenced json block out of the result event's "result"
    text field. Raises HarnessError when none is present or it does not
    parse.
    """
    text = result_payload.get("result", "")
    matches = _FENCED_JSON_ANSWER_RE.findall(text)
    if not matches:
        raise HarnessError("execute_run: result event's result field carried no fenced json block")
    try:
        return json.loads(matches[-1])
    except json.JSONDecodeError as exc:
        raise HarnessError(f"execute_run: final fenced json block did not parse: {exc}") from exc


@dataclass(frozen=True)
class RunOutcome:
    """
    The result of execute_run's per-run pipeline.

    Instance variables:
        status: "complete", "truncated", or "failed".
        config_id: the config this outcome belongs to.
        n: 1-based repetition index.
        run_json: the written run.json record, present only when status
            == "complete".
        findings: tuple of GateFinding-shaped details explaining a
            "truncated" or "failed" outcome; empty for "complete".
    """

    status: str
    config_id: str
    n: int
    run_json: dict = None
    findings: tuple = ()


def execute_run(ladder, config, n, run_dir, target_dir, server_path, repo_root, cache_dir, executor=launch_run):
    """
    The per-run pipeline: materialise inputs; launch the subprocess
    through executor, capturing its stdout stream directly to
    transcript.jsonl as it runs; parse the final fenced json block out of
    the result event's result field into answer.json; extract
    usage.json; run run_gates; invoke score_run; run
    gate_run_complete_artifacts; and only then write_run_json.

    A run whose result event reports a non-success subtype -- the shape a
    run stopped by --max-turns leaves, with no fenced json block to parse
    -- returns a "truncated" RunOutcome rather than the generic
    unparseable-answer failure, because the two call for opposite
    responses. Any failing fatal gate, an unparseable answer from a run
    that did finish, or a scoring failure returns a "failed" RunOutcome
    carrying the findings, and no run.json is written in any of these
    cases.
    """
    run_dir = Path(run_dir)
    run_dir.mkdir(parents=True, exist_ok=True)
    write_run_inputs(ladder, config, run_dir, target_dir, server_path)

    argv = build_argv(ladder, config, run_dir, target_dir)
    transcript_path = run_dir / "transcript.jsonl"
    env = run_env()

    start = time.monotonic()
    executor(argv, cwd=target_dir, env=env, transcript_path=transcript_path)
    wall_clock_ms = int((time.monotonic() - start) * 1000)

    events = read_transcript(transcript_path)
    result = result_event(events)

    if result.get("subtype") != "success":
        return RunOutcome(
            status="truncated",
            config_id=config.id,
            n=n,
            findings=(
                GateFinding(
                    "truncated",
                    f"config {config.id!r} rep {n} did not finish successfully "
                    f"(subtype={result.get('subtype')!r}); max_turns={ladder.max_turns}",
                    fatal=True,
                ),
            ),
        )

    try:
        answer = _extract_answer(result)
    except HarnessError as exc:
        return RunOutcome(
            status="failed", config_id=config.id, n=n,
            findings=(GateFinding("unparseable_answer", str(exc), fatal=True),),
        )

    with open(run_dir / "answer.json", "w") as f:
        json.dump(answer, f, indent=2)
        f.write("\n")

    usage = extract_usage(events, wall_clock_ms=wall_clock_ms, transcript_path=str(transcript_path))
    with open(run_dir / "usage.json", "w") as f:
        json.dump(usage, f, indent=2)
        f.write("\n")

    gate_report = run_gates(events, ladder, config, ladder.run_model, repo_root, target_dir, run_dir, cache_dir, env)
    if not gate_report.passed:
        return RunOutcome(status="failed", config_id=config.id, n=n, findings=gate_report.findings)

    task_text = task_text_for(ladder, config.task)
    try:
        score_run(ladder, config, run_dir, task_text)
    except ScoringError as exc:
        return RunOutcome(
            status="failed", config_id=config.id, n=n,
            findings=(GateFinding("scoring_failed", str(exc), fatal=True),),
        )

    artifact_findings = gate_run_complete_artifacts(run_dir)
    if artifact_findings:
        return RunOutcome(status="failed", config_id=config.id, n=n, findings=tuple(artifact_findings))

    payload = {
        "config_id": config.id,
        "n": n,
        "model": ladder.run_model,
        "observations": [{"gate": finding.gate, "message": finding.message} for finding in gate_report.findings],
    }
    written = write_run_json(run_dir, payload)
    return RunOutcome(status="complete", config_id=config.id, n=n, run_json=written)
