package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRestoreWorktree_RefusesTheColdConfig(t *testing.T) {
	l := mustLoadLadderFixture(t)

	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		return "", nil
	}

	var out bytes.Buffer
	err := runRestoreWorktree(&out, l, "a5-bundle-cold", git)
	if err == nil {
		t.Fatal("runRestoreWorktree() = nil; want an error for the cold config")
	}
	if !strings.Contains(err.Error(), "cold") {
		t.Errorf("runRestoreWorktree() error = %v; want it to name the cold config", err)
	}
	if len(calls) != 0 {
		t.Errorf("git calls = %v; want none, since the cold config must be refused before touching git", calls)
	}
}

func TestRunRestoreWorktree_InvokesResetCleanForAWarmConfig(t *testing.T) {
	l := mustLoadLadderFixture(t)
	config := l.Tasks["01-reed-geometry-exploration"]

	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		return "", nil
	}

	var out bytes.Buffer
	if err := runRestoreWorktree(&out, l, "a0-none", git); err != nil {
		t.Fatalf("runRestoreWorktree() error = %v; want nil", err)
	}

	wantCalls := [][]string{
		{"-C", config.Worktree, "reset", "--hard"},
		{"-C", config.Worktree, "clean", "-fdx"},
	}
	if len(calls) != len(wantCalls) {
		t.Fatalf("git calls = %v; want %v", calls, wantCalls)
	}
	for i := range wantCalls {
		if strings.Join(calls[i], " ") != strings.Join(wantCalls[i], " ") {
			t.Errorf("git call[%d] = %v; want %v", i, calls[i], wantCalls[i])
		}
	}
	if !strings.Contains(out.String(), config.Worktree) {
		t.Errorf("runRestoreWorktree() output = %q; want it to name the restored worktree", out.String())
	}
}

func TestRunRestoreWorktree_UnknownConfigErrors(t *testing.T) {
	l := mustLoadLadderFixture(t)
	git := func(args ...string) (string, error) { return "", nil }
	var out bytes.Buffer
	if err := runRestoreWorktree(&out, l, "does-not-exist", git); err == nil {
		t.Error("runRestoreWorktree() = nil; want an error for an unknown config id")
	}
}
