package ladder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireSessionLock_SecondAcquireForADifferentLabelFails(t *testing.T) {
	resultsRoot := t.TempDir()
	if err := AcquireSessionLock(resultsRoot, "session-a"); err != nil {
		t.Fatalf("AcquireSessionLock(root, %q) = %v; want nil error", "session-a", err)
	}
	if err := AcquireSessionLock(resultsRoot, "session-b"); err == nil {
		t.Error("AcquireSessionLock(root, \"session-b\") = nil; want an error while session-a holds the lock")
	}
}

func TestAcquireSessionLock_SucceedsAfterRelease(t *testing.T) {
	resultsRoot := t.TempDir()
	if err := AcquireSessionLock(resultsRoot, "session-a"); err != nil {
		t.Fatalf("AcquireSessionLock(root, %q) = %v; want nil error", "session-a", err)
	}
	if err := ReleaseSessionLock(resultsRoot); err != nil {
		t.Fatalf("ReleaseSessionLock(root) = %v; want nil error", err)
	}
	if err := AcquireSessionLock(resultsRoot, "session-b"); err != nil {
		t.Errorf("AcquireSessionLock(root, %q) after release = %v; want nil error", "session-b", err)
	}
}

func TestReleaseSessionLock_AbsentLockIsNotAnError(t *testing.T) {
	resultsRoot := t.TempDir()
	if err := ReleaseSessionLock(resultsRoot); err != nil {
		t.Errorf("ReleaseSessionLock(root) with no lock present = %v; want nil error", err)
	}
}

func TestAcquireSessionLock_ReacquiringTheSameLabelIsIdempotent(t *testing.T) {
	resultsRoot := t.TempDir()
	if err := AcquireSessionLock(resultsRoot, "session-a"); err != nil {
		t.Fatalf("AcquireSessionLock(root, %q) = %v; want nil error", "session-a", err)
	}
	if err := AcquireSessionLock(resultsRoot, "session-a"); err != nil {
		t.Errorf("re-AcquireSessionLock(root, %q) = %v; want nil error (idempotent)", "session-a", err)
	}

	data, err := os.ReadFile(filepath.Join(resultsRoot, sessionLockFilename))
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	if string(data) != "session-a\n" {
		t.Errorf("lockfile contents = %q; want %q", string(data), "session-a\n")
	}
}
