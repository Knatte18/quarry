// quarrydaemon_test.go tests the DaemonStateFile/DaemonLock path constructors as told-string path
// math over a single told-stateDir fixture — pure path arithmetic, no spawning, untagged (Tier 1).

package quarry

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDaemonStateFile(t *testing.T) {
	stateDir := t.TempDir()

	want := filepath.Join(stateDir, "go", "daemon.json")
	if got := DaemonStateFile(stateDir, "go"); got != want {
		t.Errorf("DaemonStateFile(%q, %q) = %q; want %q", stateDir, "go", got, want)
	}
}

func TestDaemonLock(t *testing.T) {
	stateDir := t.TempDir()

	want := filepath.Join(stateDir, "go", "daemon.lock")
	if got := DaemonLock(stateDir, "go"); got != want {
		t.Errorf("DaemonLock(%q, %q) = %q; want %q", stateDir, "go", got, want)
	}
}

// TestDaemonStateFile_NoLyxOrScoutSegment is a direct regression guard on card 13's told-stateDir
// rewrite: DaemonStateFile must never introduce a ".lyx" or "scout" path segment of its own — it
// joins only the language segment and the filename onto the directory it is told.
func TestDaemonStateFile_NoLyxOrScoutSegment(t *testing.T) {
	stateDir := t.TempDir()

	got := DaemonStateFile(stateDir, "go")
	for _, forbidden := range []string{".lyx", "scout"} {
		if containsPathSegment(got, forbidden) {
			t.Errorf("DaemonStateFile(%q, %q) = %q; contains forbidden segment %q", stateDir, "go", got, forbidden)
		}
	}
}

// TestDaemonLock_NoLyxOrScoutSegment is TestDaemonStateFile_NoLyxOrScoutSegment's DaemonLock
// counterpart.
func TestDaemonLock_NoLyxOrScoutSegment(t *testing.T) {
	stateDir := t.TempDir()

	got := DaemonLock(stateDir, "go")
	for _, forbidden := range []string{".lyx", "scout"} {
		if containsPathSegment(got, forbidden) {
			t.Errorf("DaemonLock(%q, %q) = %q; contains forbidden segment %q", stateDir, "go", got, forbidden)
		}
	}
}

// containsPathSegment reports whether path contains segment as one of its filepath.Separator-
// delimited components, anywhere in the path — not merely as a substring, so e.g. "scoutish" would
// not falsely match a "scout" segment check.
func containsPathSegment(path, segment string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == segment {
			return true
		}
	}
	return false
}

func TestDaemonStateFile_DistinctPerLanguage(t *testing.T) {
	stateDir := t.TempDir()

	goPath := DaemonStateFile(stateDir, "go")
	pythonPath := DaemonStateFile(stateDir, "python")
	if goPath == pythonPath {
		t.Errorf("DaemonStateFile(%q) and DaemonStateFile(%q) collided: both %q", "go", "python", goPath)
	}

	wantGo := filepath.Join(stateDir, "go", "daemon.json")
	if goPath != wantGo {
		t.Errorf("DaemonStateFile(%q) = %q; want %q", "go", goPath, wantGo)
	}
	wantPython := filepath.Join(stateDir, "python", "daemon.json")
	if pythonPath != wantPython {
		t.Errorf("DaemonStateFile(%q) = %q; want %q", "python", pythonPath, wantPython)
	}
}

func TestDaemonLock_DistinctPerLanguage(t *testing.T) {
	stateDir := t.TempDir()

	goPath := DaemonLock(stateDir, "go")
	pythonPath := DaemonLock(stateDir, "python")
	if goPath == pythonPath {
		t.Errorf("DaemonLock(%q) and DaemonLock(%q) collided: both %q", "go", "python", goPath)
	}

	wantGo := filepath.Join(stateDir, "go", "daemon.lock")
	if goPath != wantGo {
		t.Errorf("DaemonLock(%q) = %q; want %q", "go", goPath, wantGo)
	}
	wantPython := filepath.Join(stateDir, "python", "daemon.lock")
	if pythonPath != wantPython {
		t.Errorf("DaemonLock(%q) = %q; want %q", "python", pythonPath, wantPython)
	}
}
