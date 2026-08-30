// lock.go ports the cross-session guard scripts/run_ladder.py never needed: because the Go harness now
// dispatches through one supervised interactive session per (config, repetition) rather than a single
// subprocess this module owns end to end, nothing here can serialise concurrent sessions in-process --
// AcquireSessionLock/ReleaseSessionLock give the operator a filesystem-visible signal instead.

package ladder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sessionLockFilename is the fixed name of the cross-session lockfile at a results root.
const sessionLockFilename = ".session-active"

// sessionLockPath returns the lockfile path AcquireSessionLock/ReleaseSessionLock read and write at
// resultsRoot.
func sessionLockPath(resultsRoot string) string {
	return filepath.Join(resultsRoot, sessionLockFilename)
}

// AcquireSessionLock takes the cross-session lock at resultsRoot for label, writing a ".session-active"
// file naming label. Acquiring refuses with an error naming the other session's label while a lock
// naming a different label already exists; re-acquiring with the same label the existing lock already
// names is idempotent and succeeds without rewriting the file.
//
// This is a guard over operator discipline, not a proof: an operator who deletes the lockfile by hand,
// or launches a second session from a second checkout against the same results root, defeats it
// entirely. A pid-based lock is not usable here in its place, because a session outlives every process
// this harness itself starts -- there is no pid of a harness-owned process whose death would mean the
// session is over.
func AcquireSessionLock(resultsRoot, label string) error {
	path := sessionLockPath(resultsRoot)
	data, err := os.ReadFile(path)
	if err == nil {
		existing := strings.TrimSpace(string(data))
		if existing != label {
			return fmt.Errorf(
				"ladder: acquire session lock: %s is held by session %q, refusing to acquire it for %q",
				path, existing, label,
			)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("ladder: acquire session lock: read %s: %w", path, err)
	}

	if err := os.MkdirAll(resultsRoot, 0o755); err != nil {
		return fmt.Errorf("ladder: acquire session lock: create results root %s: %w", resultsRoot, err)
	}
	if err := os.WriteFile(path, []byte(label+"\n"), 0o644); err != nil {
		return fmt.Errorf("ladder: acquire session lock: write %s: %w", path, err)
	}
	return nil
}

// ReleaseSessionLock removes the cross-session lock at resultsRoot. Releasing an absent lock is not an
// error, since a resumed invocation may call this having never itself acquired the lock this run.
func ReleaseSessionLock(resultsRoot string) error {
	path := sessionLockPath(resultsRoot)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ladder: release session lock: remove %s: %w", path, err)
	}
	return nil
}
