package ladder

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// recordedCmd is one call recordingRunner.Run captured, stripped of its stream fields, so a test can
// compare it by value.
type recordedCmd struct {
	Dir  string
	Name string
	Args []string
	Env  map[string]string
}

// recordingRunner is a fake Runner: it records every Cmd it is asked to run, writes a canned
// key-mapped stdout string when one is configured for that Name+Args key, and fails every call once
// err is set.
type recordingRunner struct {
	calls   []recordedCmd
	outputs map[string]string
	err     error
}

func (r *recordingRunner) Run(_ context.Context, c Cmd) error {
	r.calls = append(r.calls, recordedCmd{
		Dir:  c.Dir,
		Name: c.Name,
		Args: append([]string(nil), c.Args...),
		Env:  c.Env,
	})
	if r.err != nil {
		return r.err
	}
	if r.outputs != nil && c.Stdout != nil {
		if out, ok := r.outputs[c.Name+" "+strings.Join(c.Args, " ")]; ok {
			_, _ = c.Stdout.Write([]byte(out))
		}
	}
	return nil
}

func TestPrepareWorktree_ArgumentVector(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "worktrees", "task-1")
	r := &recordingRunner{}

	if err := PrepareWorktree(context.Background(), r, "/repo", "task-1", "abc123", dest); err != nil {
		t.Fatalf("PrepareWorktree() = %v; want no error", err)
	}

	if len(r.calls) != 1 {
		t.Fatalf("PrepareWorktree() made %d calls; want 1", len(r.calls))
	}
	want := recordedCmd{Dir: "/repo", Name: "git", Args: []string{"worktree", "add", "--detach", dest, "abc123"}}
	if diff := diffRecordedCmd(want, r.calls[0]); diff != "" {
		t.Errorf("PrepareWorktree() call = %+v; want %+v (%s)", r.calls[0], want, diff)
	}
}

func TestPrepareWorktree_ExistingDestSkipsRunner(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "worktrees", "task-1")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	r := &recordingRunner{}

	if err := PrepareWorktree(context.Background(), r, "/repo", "task-1", "abc123", dest); err != nil {
		t.Fatalf("PrepareWorktree() = %v; want no error", err)
	}
	if len(r.calls) != 0 {
		t.Errorf("PrepareWorktree() on existing dest made %d calls; want 0", len(r.calls))
	}
}

func TestRestoreWorktree_ArgumentVectors(t *testing.T) {
	r := &recordingRunner{}
	if err := RestoreWorktree(context.Background(), r, "/dest"); err != nil {
		t.Fatalf("RestoreWorktree() = %v; want no error", err)
	}
	if len(r.calls) != 2 {
		t.Fatalf("RestoreWorktree() made %d calls; want 2", len(r.calls))
	}
	wantFirst := recordedCmd{Dir: "/dest", Name: "git", Args: []string{"reset", "--hard", "HEAD"}}
	wantSecond := recordedCmd{Dir: "/dest", Name: "git", Args: []string{"clean", "-fdx"}}
	if diff := diffRecordedCmd(wantFirst, r.calls[0]); diff != "" {
		t.Errorf("RestoreWorktree() first call = %+v; want %+v (%s)", r.calls[0], wantFirst, diff)
	}
	if diff := diffRecordedCmd(wantSecond, r.calls[1]); diff != "" {
		t.Errorf("RestoreWorktree() second call = %+v; want %+v (%s)", r.calls[1], wantSecond, diff)
	}
}

func TestWorktreeStatus_ArgumentVectorAndOutput(t *testing.T) {
	r := &recordingRunner{outputs: map[string]string{
		"git status --porcelain": " M dirty.txt\n",
	}}
	got, err := WorktreeStatus(context.Background(), r, "/dest")
	if err != nil {
		t.Fatalf("WorktreeStatus() = %v; want no error", err)
	}
	if got != " M dirty.txt\n" {
		t.Errorf("WorktreeStatus() = %q; want %q", got, " M dirty.txt\n")
	}
	if len(r.calls) != 1 {
		t.Fatalf("WorktreeStatus() made %d calls; want 1", len(r.calls))
	}
	want := recordedCmd{Dir: "/dest", Name: "git", Args: []string{"status", "--porcelain"}}
	if diff := diffRecordedCmd(want, r.calls[0]); diff != "" {
		t.Errorf("WorktreeStatus() call = %+v; want %+v (%s)", r.calls[0], want, diff)
	}
}

func diffRecordedCmd(want, got recordedCmd) string {
	if want.Dir != got.Dir {
		return "Dir differs"
	}
	if want.Name != got.Name {
		return "Name differs"
	}
	if len(want.Args) != len(got.Args) {
		return "Args length differs"
	}
	for i := range want.Args {
		if want.Args[i] != got.Args[i] {
			return "Args differ"
		}
	}
	return ""
}

func TestResolveWorktreeRoot(t *testing.T) {
	repoRoot := t.TempDir()

	tests := []struct {
		name          string
		override      string
		xdgCacheHome  string
		wantErrSubstr string
	}{
		{
			name:          "OverrideUnderRepoRoot_Refused",
			override:      filepath.Join(repoRoot, "cache"),
			wantErrSubstr: "LADDER_WORKTREE_ROOT",
		},
		{
			name:          "OverrideContainsQuarryAnyCasing_Refused",
			override:      "/tmp/some-QuArry-cache",
			wantErrSubstr: "LADDER_WORKTREE_ROOT",
		},
		{
			name:         "CacheDerivedDefault_Accepted",
			xdgCacheHome: t.TempDir(),
		},
		{
			name:     "ExplicitOverrideOutsideRepoWithoutToken_Accepted",
			override: filepath.Join(t.TempDir(), "eval-root"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LADDER_WORKTREE_ROOT", tt.override)
			t.Setenv("XDG_CACHE_HOME", tt.xdgCacheHome)

			got, err := ResolveWorktreeRoot(repoRoot)
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("ResolveWorktreeRoot(%s) = %s, nil; want error naming %s", repoRoot, got, tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("ResolveWorktreeRoot(%s) error = %q; want it to name %q", repoRoot, err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveWorktreeRoot(%s) = %v; want no error", repoRoot, err)
			}
			if got == "" {
				t.Errorf("ResolveWorktreeRoot(%s) = %q; want a non-empty path", repoRoot, got)
			}
		})
	}
}

func TestResolveLoomyardRepo(t *testing.T) {
	t.Run("ProcessVariableWins", func(t *testing.T) {
		quarryRoot := t.TempDir()
		t.Setenv("LADDER_LOOMYARD_REPO", "/from/env/var")

		got, err := ResolveLoomyardRepo(quarryRoot)
		if err != nil {
			t.Fatalf("ResolveLoomyardRepo() = %v; want no error", err)
		}
		if got != "/from/env/var" {
			t.Errorf("ResolveLoomyardRepo() = %q; want %q", got, "/from/env/var")
		}
	})

	t.Run("EnvFileParsedWhenVariableUnset", func(t *testing.T) {
		quarryRoot := t.TempDir()
		t.Setenv("LADDER_LOOMYARD_REPO", "")
		scratchDir := filepath.Join(quarryRoot, ".scratch")
		if err := os.MkdirAll(scratchDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		contents := "# a comment\n\nLADDER_LOOMYARD_REPO=/from/file\n"
		if err := os.WriteFile(filepath.Join(scratchDir, "ladder.env"), []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		got, err := ResolveLoomyardRepo(quarryRoot)
		if err != nil {
			t.Fatalf("ResolveLoomyardRepo() = %v; want no error", err)
		}
		if got != "/from/file" {
			t.Errorf("ResolveLoomyardRepo() = %q; want %q", got, "/from/file")
		}
	})

	t.Run("NeitherSet_ErrorNamesBoth", func(t *testing.T) {
		quarryRoot := t.TempDir()
		t.Setenv("LADDER_LOOMYARD_REPO", "")

		_, err := ResolveLoomyardRepo(quarryRoot)
		if err == nil {
			t.Fatal("ResolveLoomyardRepo() = nil error; want an error naming the variable and the file")
		}
		if !strings.Contains(err.Error(), "LADDER_LOOMYARD_REPO") {
			t.Errorf("ResolveLoomyardRepo() error = %q; want it to name LADDER_LOOMYARD_REPO", err)
		}
		if !strings.Contains(err.Error(), filepath.Join(quarryRoot, ".scratch", "ladder.env")) {
			t.Errorf("ResolveLoomyardRepo() error = %q; want it to name the env file path", err)
		}
	})
}

func TestAcquireRunLock(t *testing.T) {
	worktreeRoot := t.TempDir()

	release, err := AcquireRunLock(worktreeRoot, "/results/root")
	if err != nil {
		t.Fatalf("AcquireRunLock() = %v; want no error", err)
	}

	lockPath := filepath.Join(worktreeRoot, ".ladder.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file %s not created: %v", lockPath, err)
	}

	// The lock path must be the worktree root's own direct child, asserted by string equality --
	// never inside a task worktree -- so a future refactor cannot move it there unnoticed.
	wantLockPath := worktreeRoot + string(filepath.Separator) + ".ladder.lock"
	if lockPath != wantLockPath {
		t.Errorf("lock path = %q; want %q", lockPath, wantLockPath)
	}

	_, err = AcquireRunLock(worktreeRoot, "/other/results")
	if err == nil {
		t.Fatal("second AcquireRunLock() = nil error; want an error naming the first holder")
	}
	pid := strconv.Itoa(os.Getpid())
	if !strings.Contains(err.Error(), "pid "+pid) {
		t.Errorf("second AcquireRunLock() error = %q; want it to carry pid %s", err, pid)
	}
	if !strings.Contains(err.Error(), "/results/root") {
		t.Errorf("second AcquireRunLock() error = %q; want it to carry the first holder's results root", err)
	}

	if err := release(); err != nil {
		t.Fatalf("release() = %v; want no error", err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("lock file %s still exists after release", lockPath)
	}
}
