package ladder

import (
	"os"
	"path/filepath"
	"testing"
)

// writeRawFile writes contents verbatim to path, failing the test on error.
func writeRawFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mustWriteRunState writes s to dir via WriteRunState, failing the test on error.
func mustWriteRunState(t *testing.T, dir string, s RunState) {
	t.Helper()
	if err := WriteRunState(dir, s); err != nil {
		t.Fatalf("WriteRunState(%s) error = %v; want no error", dir, err)
	}
}

// mustMkdir creates dir, failing the test on error.
func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

// assertDirExists fails the test unless dir exists and is a directory.
func assertDirExists(t *testing.T, dir string) {
	t.Helper()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %s: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s exists but is not a directory", dir)
	}
}

// TestRepDir asserts the repetition directory path shape: <results-root>/raw/<cell>/<rep>.
func TestRepDir(t *testing.T) {
	got := RepDir("/results/root", "a0-none", 3)
	want := filepath.Join("/results/root", "raw", "a0-none", "3")
	if got != want {
		t.Errorf("RepDir() = %q; want %q", got, want)
	}
}

// TestWriteReadRunState_RoundTripsEveryField asserts that a written state round-trips every field,
// including the observations slice.
func TestWriteReadRunState_RoundTripsEveryField(t *testing.T) {
	dir := t.TempDir()

	want := RunState{
		State:            "complete",
		ConfigID:         "a1-quarry",
		Ladder:           "a",
		Task:             "reed-geometry",
		Allowed:          []string{"toc", "grep"},
		IsControl:        false,
		ControlForLadder: "",
		ServerName:       "quarry",
		MCPPrefix:        "mcp__quarry__",
		Rep:              2,
		Model:            "claude-opus",
		Effort:           "high",
		MaxTurns:         30,
		Scored:           true,
		ScoreSkipReason:  "",
		Observations: []Finding{
			{Gate: "worktree_dirtied", Fatal: false, Message: "dirty", Count: 0},
			{Gate: "target_origin_quarry_mention", Fatal: false, Message: "mentioned", Count: 2},
		},
		BlindingFailed: false,
		MaxTurnsHit:    false,
	}

	if err := WriteRunState(dir, want); err != nil {
		t.Fatalf("WriteRunState() error = %v; want no error", err)
	}

	got, err := ReadRunState(dir)
	if err != nil {
		t.Fatalf("ReadRunState() error = %v; want no error", err)
	}

	if got.State != want.State ||
		got.ConfigID != want.ConfigID ||
		got.Ladder != want.Ladder ||
		got.Task != want.Task ||
		got.IsControl != want.IsControl ||
		got.ControlForLadder != want.ControlForLadder ||
		got.ServerName != want.ServerName ||
		got.MCPPrefix != want.MCPPrefix ||
		got.Rep != want.Rep ||
		got.Model != want.Model ||
		got.Effort != want.Effort ||
		got.MaxTurns != want.MaxTurns ||
		got.Scored != want.Scored ||
		got.ScoreSkipReason != want.ScoreSkipReason ||
		got.BlindingFailed != want.BlindingFailed ||
		got.MaxTurnsHit != want.MaxTurnsHit {
		t.Errorf("ReadRunState() = %+v; want %+v", got, want)
	}

	if len(got.Allowed) != len(want.Allowed) {
		t.Fatalf("ReadRunState().Allowed = %v; want %v", got.Allowed, want.Allowed)
	}
	for i := range want.Allowed {
		if got.Allowed[i] != want.Allowed[i] {
			t.Errorf("ReadRunState().Allowed[%d] = %q; want %q", i, got.Allowed[i], want.Allowed[i])
		}
	}

	if len(got.Observations) != len(want.Observations) {
		t.Fatalf("ReadRunState().Observations = %+v; want %+v", got.Observations, want.Observations)
	}
	for i := range want.Observations {
		if got.Observations[i] != want.Observations[i] {
			t.Errorf("ReadRunState().Observations[%d] = %+v; want %+v", i, got.Observations[i], want.Observations[i])
		}
	}
}

// TestRepIsComplete covers every case the completeness predicate must distinguish: a missing file, a
// truncated file, a state that is not complete, a complete state whose blinding-failed flag is true --
// the case that must return false so a void repetition is re-attempted -- and a complete state whose
// blinding-failed flag is false.
func TestRepIsComplete(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string)
		want  bool
	}{
		{
			name:  "MissingFile",
			setup: func(dir string) {},
			want:  false,
		},
		{
			name: "TruncatedFile",
			setup: func(dir string) {
				writeRawFile(t, filepath.Join(dir, RunStateFile), "{not valid json")
			},
			want: false,
		},
		{
			name: "StateNotComplete",
			setup: func(dir string) {
				mustWriteRunState(t, dir, RunState{State: "in_progress", BlindingFailed: false})
			},
			want: false,
		},
		{
			name: "CompleteButBlindingFailed",
			setup: func(dir string) {
				mustWriteRunState(t, dir, RunState{State: "complete", BlindingFailed: true})
			},
			want: false,
		},
		{
			name: "CompleteAndNotBlindingFailed",
			setup: func(dir string) {
				mustWriteRunState(t, dir, RunState{State: "complete", BlindingFailed: false})
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			got := RepIsComplete(dir)
			if got != tt.want {
				t.Errorf("RepIsComplete() = %v; want %v", got, tt.want)
			}
		})
	}
}

// TestInvalidateRep asserts that repeated invalidation produces the first, second and third suffixed
// directories in order and reports the attempt count, and that the ceiling constant is three.
func TestInvalidateRep(t *testing.T) {
	if MaxAttempts != 3 {
		t.Fatalf("MaxAttempts = %d; want 3", MaxAttempts)
	}

	root := t.TempDir()
	dir := filepath.Join(root, "rep")
	mustMkdir(t, dir)

	attempts, err := InvalidateRep(dir)
	if err != nil {
		t.Fatalf("InvalidateRep() error = %v; want no error", err)
	}
	if attempts != 1 {
		t.Errorf("InvalidateRep() attempts = %d; want 1", attempts)
	}
	assertDirExists(t, dir+".invalid-1")

	mustMkdir(t, dir)
	attempts, err = InvalidateRep(dir)
	if err != nil {
		t.Fatalf("InvalidateRep() error = %v; want no error", err)
	}
	if attempts != 2 {
		t.Errorf("InvalidateRep() attempts = %d; want 2", attempts)
	}
	assertDirExists(t, dir+".invalid-2")

	mustMkdir(t, dir)
	attempts, err = InvalidateRep(dir)
	if err != nil {
		t.Fatalf("InvalidateRep() error = %v; want no error", err)
	}
	if attempts != 3 {
		t.Errorf("InvalidateRep() attempts = %d; want 3", attempts)
	}
	assertDirExists(t, dir+".invalid-3")
}
