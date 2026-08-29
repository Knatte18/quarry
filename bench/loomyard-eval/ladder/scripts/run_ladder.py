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
import select
import shutil
import subprocess
import time
from pathlib import Path

from gates import daemon_state_file, gate_worktree_neutralised, resolve_state_dir
from ladder_config import write_settings


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
