package main

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommand_RegistersWithoutError(t *testing.T) {
	cmd := Command()
	cmd.SetArgs([]string{"--help"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command().Execute() with --help = %v; want nil error", err)
	}

	names := make(map[string]bool, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"prepare-session", "next-run", "warm", "restore-worktree"} {
		if !names[want] {
			t.Errorf("Command().Commands() is missing %q", want)
		}
	}
}

func TestResolveLadder_SurfacesALoadFailureRatherThanSwallowingIt(t *testing.T) {
	cmd := Command()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	missing := filepath.Join(t.TempDir(), "missing-ladder.yaml")
	cmd.SetArgs([]string{"next-run", "--ladder", missing, "--results-root", t.TempDir(), "--config-id", "a0-none"})
	cmd.SetOut(io.Discard)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() = nil; want an error for a nonexistent ladder file")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("Execute() error = %v; want it to name the missing ladder path %q", err, missing)
	}
}
