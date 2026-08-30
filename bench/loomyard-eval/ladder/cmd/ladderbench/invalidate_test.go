package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func TestRunInvalidate_PrintsNextAttemptIndex(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	runDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", runDir, err)
	}

	var out bytes.Buffer
	if err := runInvalidate(&out, l, resultsRoot, "a0-none", 1); err != nil {
		t.Fatalf("runInvalidate() error = %v; want nil", err)
	}
	if !strings.Contains(out.String(), "next attempt is 2") {
		t.Errorf("runInvalidate() output = %q; want it to report next attempt is 2", out.String())
	}
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Errorf("runInvalidate() left %s in place; want it renamed aside", runDir)
	}
	if _, err := os.Stat(runDir + ".invalid-1"); err != nil {
		t.Errorf("runInvalidate() did not rename %s to its .invalid-1 sibling: %v", runDir, err)
	}
}

func TestRunInvalidate_ErrorsOnceAttemptCeilingIsExhausted(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	runDir := ladder.RunDirPath(resultsRoot, "a0-none", 1)
	for k := 1; k < ladder.MaxAttempts; k++ {
		if err := os.MkdirAll(fmt.Sprintf("%s.invalid-%d", runDir, k), 0o755); err != nil {
			t.Fatalf("mkdir invalid sibling %d: %v", k, err)
		}
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", runDir, err)
	}

	var out bytes.Buffer
	err := runInvalidate(&out, l, resultsRoot, "a0-none", 1)
	if err == nil {
		t.Fatal("runInvalidate() error = nil; want an error once MaxAttempts is exhausted")
	}
}

func TestRunInvalidate_UnknownConfigErrors(t *testing.T) {
	l := mustLoadLadderFixture(t)
	resultsRoot := t.TempDir()

	var out bytes.Buffer
	if err := runInvalidate(&out, l, resultsRoot, "does-not-exist", 1); err == nil {
		t.Error("runInvalidate() = nil; want an error for an unknown config id")
	}
}
