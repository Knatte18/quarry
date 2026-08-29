"""
Tests for run_ladder.py: the pure planning and resume logic -- run
ordering, attempt accounting, the resume skip decision, argv assembly,
environment scrubbing, and task-text/schema extraction with their section
boundaries. Every test drives an injected `git=` or `executor=` seam; none
launches a subprocess, builds a real worktree, or makes a model call.
"""
