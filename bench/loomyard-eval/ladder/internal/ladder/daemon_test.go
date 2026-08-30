package ladder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestScrubbedEnv(t *testing.T) {
	t.Setenv("QUARRY_STATE_DIR", "/should-be-scrubbed")
	t.Setenv("QUARRY_BUILD_TAGS", "integration")
	t.Setenv("QUARRY_CONFIG", "/keep/me")

	env := ScrubbedEnv()

	if value, ok := envLookup(env, "QUARRY_STATE_DIR"); !ok || value != "" {
		t.Errorf(`envLookup(ScrubbedEnv(), "QUARRY_STATE_DIR") = %q, %v; want "", true`, value, ok)
	}
	if value, ok := envLookup(env, "QUARRY_BUILD_TAGS"); !ok || value != "" {
		t.Errorf(`envLookup(ScrubbedEnv(), "QUARRY_BUILD_TAGS") = %q, %v; want "", true`, value, ok)
	}
	if value, ok := envLookup(env, "QUARRY_CONFIG"); !ok || value != "/keep/me" {
		t.Errorf(`envLookup(ScrubbedEnv(), "QUARRY_CONFIG") = %q, %v; want "/keep/me", true`, value, ok)
	}
}

func TestScrubbedEnv_ForcesAbsentKeysPresent(t *testing.T) {
	os.Unsetenv("QUARRY_STATE_DIR")
	os.Unsetenv("QUARRY_BUILD_TAGS")

	env := ScrubbedEnv()

	if value, ok := envLookup(env, "QUARRY_STATE_DIR"); !ok || value != "" {
		t.Errorf(`envLookup(ScrubbedEnv(), "QUARRY_STATE_DIR") = %q, %v; want "", true`, value, ok)
	}
	if value, ok := envLookup(env, "QUARRY_BUILD_TAGS"); !ok || value != "" {
		t.Errorf(`envLookup(ScrubbedEnv(), "QUARRY_BUILD_TAGS") = %q, %v; want "", true`, value, ok)
	}
}

func TestWorkspaceKey_DeterministicAndDiffersByPath(t *testing.T) {
	tmp := t.TempDir()
	first := filepath.Join(tmp, "app")
	second := filepath.Join(tmp, "nested", "app")
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatalf("MkdirAll(first): %v", err)
	}
	if err := os.MkdirAll(second, 0o755); err != nil {
		t.Fatalf("MkdirAll(second): %v", err)
	}

	firstKey, firstKeyAgain := WorkspaceKey(first), WorkspaceKey(first)
	if firstKey != firstKeyAgain {
		t.Error("WorkspaceKey(first) is not deterministic across calls")
	}
	if WorkspaceKey(first) == WorkspaceKey(second) {
		t.Error("WorkspaceKey(first) == WorkspaceKey(second); want distinct keys for distinct paths")
	}
}

func TestResolveStateDir(t *testing.T) {
	tmp := t.TempDir()

	t.Run("rejects a non-empty QUARRY_STATE_DIR", func(t *testing.T) {
		_, err := ResolveStateDir(tmp, tmp, []string{"QUARRY_STATE_DIR=/somewhere"})
		if err == nil {
			t.Fatal("ResolveStateDir() = _, nil; want a *GateError")
		}
		if _, ok := err.(*GateError); !ok {
			t.Errorf("ResolveStateDir() error type = %T; want *GateError", err)
		}
	})

	t.Run("rejects a non-empty QUARRY_BUILD_TAGS", func(t *testing.T) {
		_, err := ResolveStateDir(tmp, tmp, []string{"QUARRY_BUILD_TAGS=integration"})
		if err == nil {
			t.Fatal("ResolveStateDir() = _, nil; want a *GateError")
		}
		if _, ok := err.(*GateError); !ok {
			t.Errorf("ResolveStateDir() error type = %T; want *GateError", err)
		}
	})

	t.Run("accepts an absent key", func(t *testing.T) {
		got, err := ResolveStateDir(tmp, tmp, nil)
		if err != nil {
			t.Fatalf("ResolveStateDir() = _, %v; want nil error", err)
		}
		want := filepath.Join(tmp, "quarry", WorkspaceKey(tmp))
		if got != want {
			t.Errorf("ResolveStateDir() = %q; want %q", got, want)
		}
	})

	t.Run("accepts a key set to the empty string", func(t *testing.T) {
		got, err := ResolveStateDir(tmp, tmp, []string{"QUARRY_STATE_DIR=", "QUARRY_BUILD_TAGS="})
		if err != nil {
			t.Fatalf("ResolveStateDir() = _, %v; want nil error", err)
		}
		want := filepath.Join(tmp, "quarry", WorkspaceKey(tmp))
		if got != want {
			t.Errorf("ResolveStateDir() = %q; want %q", got, want)
		}
	})
}

func TestUserCacheDir(t *testing.T) {
	dir, err := UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir() = _, %v; want nil error", err)
	}
	if dir == "" {
		t.Error("UserCacheDir() = \"\"; want a non-empty path")
	}
}

// writeDaemonStateFile writes a daemon.json for targetDir/cacheDir/lang recording pid, mirroring
// test_gates.py's own _write_daemon_state helper.
func writeDaemonStateFile(t *testing.T, targetDir, cacheDir string, pid int, lang string) {
	t.Helper()
	stateDir, err := ResolveStateDir(targetDir, cacheDir, nil)
	if err != nil {
		t.Fatalf("ResolveStateDir() = _, %v; want nil error", err)
	}
	path := DaemonStateFile(stateDir, lang)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	body := fmt.Sprintf(`{"pid":%d,"address":"unix;x","protocol_version":"1","started_at":"now"}`, pid)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// spawnAndWaitForExit spawns a process that exits immediately and returns its now-dead pid, mirroring
// test_gates.py's own _spawn_and_wait_for_exit helper.
func spawnAndWaitForExit(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sh", "-c", "true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("spawn short-lived process: %v", err)
	}
	return cmd.Process.Pid
}

func TestPidAlive(t *testing.T) {
	if !pidAlive(os.Getpid()) {
		t.Error("pidAlive(os.Getpid()) = false; want true")
	}
	if pidAlive(spawnAndWaitForExit(t)) {
		t.Error("pidAlive(<dead pid>) = true; want false")
	}
}

func TestReadDaemonState(t *testing.T) {
	t.Run("absent state file returns nil", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if state := readDaemonState(targetDir, cacheDir, nil, daemonLang); state != nil {
			t.Errorf("readDaemonState() = %+v; want nil", state)
		}
	})

	t.Run("well-formed state file parses", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		writeDaemonStateFile(t, targetDir, cacheDir, os.Getpid(), daemonLang)

		state := readDaemonState(targetDir, cacheDir, nil, daemonLang)
		if state == nil || state.PID != os.Getpid() {
			t.Errorf("readDaemonState() = %+v; want PID %d", state, os.Getpid())
		}
	})

	t.Run("malformed state file panics", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		stateDir, err := ResolveStateDir(targetDir, cacheDir, nil)
		if err != nil {
			t.Fatalf("ResolveStateDir() = _, %v; want nil error", err)
		}
		path := DaemonStateFile(stateDir, daemonLang)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		defer func() {
			if recover() == nil {
				t.Error("readDaemonState() on a malformed file did not panic; want a panic mirroring the Python port's uncaught JSONDecodeError")
			}
		}()
		readDaemonState(targetDir, cacheDir, nil, daemonLang)
	})
}

func TestDaemonAlive(t *testing.T) {
	t.Run("true when the recorded pid is alive", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		writeDaemonStateFile(t, targetDir, cacheDir, os.Getpid(), daemonLang)
		if !DaemonAlive(targetDir, cacheDir, nil) {
			t.Error("DaemonAlive() = false; want true")
		}
	})

	t.Run("false when no state file exists", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if DaemonAlive(targetDir, cacheDir, nil) {
			t.Error("DaemonAlive() = true; want false")
		}
	})

	t.Run("false when the recorded pid is dead", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		writeDaemonStateFile(t, targetDir, cacheDir, spawnAndWaitForExit(t), daemonLang)
		if DaemonAlive(targetDir, cacheDir, nil) {
			t.Error("DaemonAlive() = true; want false")
		}
	})
}

func TestDaemonPID(t *testing.T) {
	tmp := t.TempDir()
	targetDir := filepath.Join(tmp, "worktree")
	cacheDir := filepath.Join(tmp, "cache")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if _, ok := DaemonPID(targetDir, cacheDir, nil); ok {
		t.Error("DaemonPID() ok = true before any state file exists; want false")
	}

	writeDaemonStateFile(t, targetDir, cacheDir, os.Getpid(), daemonLang)
	pid, ok := DaemonPID(targetDir, cacheDir, nil)
	if !ok || pid != os.Getpid() {
		t.Errorf("DaemonPID() = %d, %v; want %d, true", pid, ok, os.Getpid())
	}
}

func TestClearStateDir(t *testing.T) {
	t.Run("removes the resolved directory and leaves a sibling untouched", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		siblingDir := filepath.Join(tmp, "sibling-worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.MkdirAll(siblingDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		writeDaemonStateFile(t, targetDir, cacheDir, os.Getpid(), daemonLang)
		writeDaemonStateFile(t, siblingDir, cacheDir, os.Getpid(), daemonLang)

		if err := ClearStateDir(targetDir, cacheDir, nil); err != nil {
			t.Fatalf("ClearStateDir() = %v; want nil error", err)
		}

		targetStateDir, err := ResolveStateDir(targetDir, cacheDir, nil)
		if err != nil {
			t.Fatalf("ResolveStateDir(targetDir): %v", err)
		}
		siblingStateDir, err := ResolveStateDir(siblingDir, cacheDir, nil)
		if err != nil {
			t.Fatalf("ResolveStateDir(siblingDir): %v", err)
		}
		if _, err := os.Stat(targetStateDir); !os.IsNotExist(err) {
			t.Errorf("os.Stat(targetStateDir) err = %v; want IsNotExist", err)
		}
		if _, err := os.Stat(siblingStateDir); err != nil {
			t.Errorf("os.Stat(siblingStateDir) err = %v; want nil", err)
		}
	})

	t.Run("is not an error when the directory is already absent", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := ClearStateDir(targetDir, cacheDir, nil); err != nil {
			t.Errorf("ClearStateDir() on an absent directory = %v; want nil error", err)
		}
	})
}

func TestWaitForDaemonExit(t *testing.T) {
	t.Run("returns immediately when no daemon.json exists", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := WaitForDaemonExit(targetDir, cacheDir, nil, time.Second, daemonLang); err != nil {
			t.Errorf("WaitForDaemonExit() = %v; want nil error", err)
		}
	})

	t.Run("returns once a live pid exits", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		cmd := exec.Command("sh", "-c", "sleep 0.3")
		if err := cmd.Start(); err != nil {
			t.Fatalf("Start(): %v", err)
		}
		writeDaemonStateFile(t, targetDir, cacheDir, cmd.Process.Pid, daemonLang)
		// A direct child stays a zombie -- still "alive" to a signal-zero probe -- until reaped, so a
		// concurrent goroutine reaps it while the poll loop runs, mirroring test_gates.py's own reaper
		// thread.
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()

		start := time.Now()
		if err := WaitForDaemonExit(targetDir, cacheDir, nil, 5*time.Second, daemonLang); err != nil {
			t.Errorf("WaitForDaemonExit() = %v; want nil error", err)
		}
		if elapsed := time.Since(start); elapsed >= 5*time.Second {
			t.Errorf("WaitForDaemonExit() took %s; want well under the 5s timeout", elapsed)
		}
		<-done
	})

	t.Run("raises on timeout against a pid that stays alive", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		cmd := exec.Command("sh", "-c", "sleep 5")
		if err := cmd.Start(); err != nil {
			t.Fatalf("Start(): %v", err)
		}
		defer func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}()
		writeDaemonStateFile(t, targetDir, cacheDir, cmd.Process.Pid, daemonLang)

		err := WaitForDaemonExit(targetDir, cacheDir, nil, 300*time.Millisecond, daemonLang)
		if err == nil {
			t.Fatal("WaitForDaemonExit() = nil; want a *GateError")
		}
		if _, ok := err.(*GateError); !ok {
			t.Errorf("WaitForDaemonExit() error type = %T; want *GateError", err)
		}
	})
}

func TestGateColdBefore(t *testing.T) {
	t.Run("passes on an empty cache dir", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if findings := GateColdBefore(targetDir, cacheDir, nil); findings != nil {
			t.Errorf("GateColdBefore() = %+v; want nil", findings)
		}
	})

	t.Run("fails when a daemon is alive", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		writeDaemonStateFile(t, targetDir, cacheDir, os.Getpid(), daemonLang)
		findings := GateColdBefore(targetDir, cacheDir, nil)
		if len(findings) != 1 || !findings[0].Fatal {
			t.Errorf("GateColdBefore() = %+v; want exactly one fatal finding", findings)
		}
	})

	t.Run("passes when daemon.json is present but the pid is dead", func(t *testing.T) {
		// The state a previous failed attempt leaves behind -- daemon.json is never removed on exit,
		// only daemon.sock is.
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		writeDaemonStateFile(t, targetDir, cacheDir, spawnAndWaitForExit(t), daemonLang)
		if findings := GateColdBefore(targetDir, cacheDir, nil); findings != nil {
			t.Errorf("GateColdBefore() = %+v; want nil", findings)
		}
	})
}

func TestGateColdAfter(t *testing.T) {
	t.Run("non-fatal observation when no daemon-backed call was made", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		records, err := ReadTranscript("testdata/cold-native-fallback.jsonl")
		if err != nil {
			t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
		}

		findings := GateColdAfter(records, targetDir, cacheDir, nil)
		if len(findings) != 1 || findings[0].Fatal || findings[0].Gate != "cold_no_daemon_backed_call" {
			t.Errorf("GateColdAfter() = %+v; want exactly one non-fatal cold_no_daemon_backed_call finding", findings)
		}
	})

	t.Run("fatal when a daemon-backed tool was used and no daemon.json exists", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
		if err != nil {
			t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
		}

		findings := GateColdAfter(records, targetDir, cacheDir, nil)
		if len(findings) != 1 || !findings[0].Fatal || findings[0].Gate != "cold_after" {
			t.Errorf("GateColdAfter() = %+v; want exactly one fatal cold_after finding", findings)
		}
	})

	t.Run("passes when a daemon-backed tool was used and daemon.json exists", func(t *testing.T) {
		tmp := t.TempDir()
		targetDir := filepath.Join(tmp, "worktree")
		cacheDir := filepath.Join(tmp, "cache")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		writeDaemonStateFile(t, targetDir, cacheDir, os.Getpid(), daemonLang)
		records, err := ReadTranscript("testdata/bundle-mixed-tools.jsonl")
		if err != nil {
			t.Fatalf("ReadTranscript() = _, %v; want nil error", err)
		}

		if findings := GateColdAfter(records, targetDir, cacheDir, nil); findings != nil {
			t.Errorf("GateColdAfter() = %+v; want nil", findings)
		}
	})
}
