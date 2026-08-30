package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func TestRunWarm_RefusesTheColdConfig(t *testing.T) {
	l := mustLoadLadderFixture(t)

	buildCalls := 0
	build := func(dir string, env []string, args ...string) (string, error) {
		buildCalls++
		return "", nil
	}
	warmCalls := 0
	warmFn := func(serverPath, targetDir string, env []string, cacheDir string) error {
		warmCalls++
		return nil
	}

	var out bytes.Buffer
	err := runWarm(&out, l, "a5-bundle-cold", t.TempDir(), build, warmFn)
	if err == nil {
		t.Fatal("runWarm() = nil; want an error for the cold config")
	}
	if !strings.Contains(err.Error(), "cold") {
		t.Errorf("runWarm() error = %v; want it to name the cold config", err)
	}
	if buildCalls != 0 || warmCalls != 0 {
		t.Errorf("build calls = %d, warm calls = %d; want both 0, since the cold config must be refused before either", buildCalls, warmCalls)
	}
}

func TestRunWarm_PostConditionFailureProducesANonNilError(t *testing.T) {
	l := mustLoadLadderFixture(t)

	build := func(dir string, env []string, args ...string) (string, error) {
		return "/built/quarry-mcp", nil
	}
	warmFn := func(serverPath, targetDir string, env []string, cacheDir string) error {
		return &ladder.HarnessError{Message: "warm_daemon: no daemon.json after warming"}
	}

	var out bytes.Buffer
	err := runWarm(&out, l, "a0-none", t.TempDir(), build, warmFn)
	if err == nil {
		t.Fatal("runWarm() = nil; want the post-condition failure propagated as a non-nil error")
	}
	if !strings.Contains(err.Error(), "daemon.json") {
		t.Errorf("runWarm() error = %v; want it to carry the warm-up post-condition failure", err)
	}
}

func TestRunWarm_SuccessPrintsConfirmation(t *testing.T) {
	l := mustLoadLadderFixture(t)

	build := func(dir string, env []string, args ...string) (string, error) {
		return "/built/quarry-mcp", nil
	}
	warmFn := func(serverPath, targetDir string, env []string, cacheDir string) error {
		return nil
	}

	var out bytes.Buffer
	if err := runWarm(&out, l, "a0-none", t.TempDir(), build, warmFn); err != nil {
		t.Fatalf("runWarm() error = %v; want nil", err)
	}
	if !strings.Contains(out.String(), "a0-none") {
		t.Errorf("runWarm() output = %q; want it to name the warmed config", out.String())
	}
}

func TestRunWarm_UnknownConfigErrors(t *testing.T) {
	l := mustLoadLadderFixture(t)
	var out bytes.Buffer
	build := func(dir string, env []string, args ...string) (string, error) { return "", nil }
	warmFn := func(serverPath, targetDir string, env []string, cacheDir string) error { return nil }

	if err := runWarm(&out, l, "does-not-exist", t.TempDir(), build, warmFn); err == nil {
		t.Error("runWarm() = nil; want an error for an unknown config id")
	}
}
