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
import shutil
import subprocess
from pathlib import Path

from gates import gate_worktree_neutralised


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
