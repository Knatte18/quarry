// daemon.go ports the environment-dependent half of scripts/gates.py: the environment scrub, the
// workspace-key and state-directory resolution the cold cell's entire argument rests on, the daemon
// state file and liveness helpers, state-directory clearing, the bounded daemon-exit wait, and the two
// cold-cell gates. Every function here that resolves a state directory takes an explicit environment
// slice rather than reading the process environment, so ScrubbedEnv is applied at an explicit call site
// rather than implicitly -- this harness no longer owns the process that spawns the server, so an
// implicit scrub would have nowhere to apply.

package ladder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// scrubbedEnvKeys is the fixed set of environment variable names ScrubbedEnv forces to the empty
// string, in the order they are considered throughout this file.
var scrubbedEnvKeys = []string{"QUARRY_STATE_DIR", "QUARRY_BUILD_TAGS"}

// ScrubbedEnv returns os.Environ() with QUARRY_STATE_DIR and QUARRY_BUILD_TAGS each forced to the
// empty string rather than removed, and QUARRY_CONFIG left untouched.
//
// Both scrubbed variables take precedence over the workspace key outright and a non-empty tag set
// appends a "tags-<hex>" segment at every tier, so either one, if inherited from the harness's own
// process, would move the resolved state directory off the per-path key the cold cell depends on.
// QUARRY_CONFIG is deliberately not scrubbed: it selects the servers.yaml overlay naming the
// language-server command, and clearing it on a machine that needs an overlay would stop the server
// starting at all.
func ScrubbedEnv() []string {
	environ := os.Environ()
	result := make([]string, 0, len(environ)+len(scrubbedEnvKeys))
	seen := make(map[string]bool, len(scrubbedEnvKeys))
	for _, kv := range environ {
		key, _, _ := envKeyValue(kv)
		if isScrubbedEnvKey(key) {
			result = append(result, key+"=")
			seen[key] = true
			continue
		}
		result = append(result, kv)
	}
	// A scrubbed key absent from os.Environ() is still forced present, empty -- ResolveStateDir
	// treats an absent key and a key set to the empty string identically (both pass), so this loop
	// exists only to keep ScrubbedEnv's own contract ("each forced to the empty string") true
	// regardless of the harness process's own environment.
	for _, key := range scrubbedEnvKeys {
		if !seen[key] {
			result = append(result, key+"=")
		}
	}
	return result
}

// isScrubbedEnvKey reports whether key is one ScrubbedEnv forces to the empty string.
func isScrubbedEnvKey(key string) bool {
	for _, scrubbed := range scrubbedEnvKeys {
		if key == scrubbed {
			return true
		}
	}
	return false
}

// envKeyValue splits one os.Environ()-style "KEY=VALUE" entry into its key and value.
func envKeyValue(kv string) (key, value string, ok bool) {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return kv[:i], kv[i+1:], true
		}
	}
	return kv, "", false
}

// envLookup returns the value of key in env (an os.Environ()-style slice) and whether it was present
// at all. A key present with an empty value returns ("", true), distinct from a key absent entirely,
// which returns ("", false) -- this is what lets ResolveStateDir tell "the key was set to empty" apart
// from "the key was never set", matching resolve_state_dir's own env.get(...) truthiness check.
func envLookup(env []string, key string) (value string, ok bool) {
	for _, kv := range env {
		k, v, hasEquals := envKeyValue(kv)
		if k != key {
			continue
		}
		if !hasEquals {
			return "", true
		}
		return v, true
	}
	return "", false
}

// WorkspaceKey re-derives quarry's own workspace key (internal/cli/paths.go's unexported
// workspaceKey): targetDir's base name, a hyphen, then the first 12 hex characters of the SHA-256
// digest of the cleaned absolute path.
func WorkspaceKey(targetDir string) string {
	return workspaceKey(targetDir)
}

// workspaceKey is WorkspaceKey's implementation. It is factored out unexported so ResolveStateDir can
// call it without going through the exported wrapper, mirroring internal/cli/paths.go's own
// workspaceKey/ResolveStateDir split.
func workspaceKey(targetDir string) string {
	// filepath.Abs's error is ignored here, mirroring quarry's own workspaceKey
	// (internal/cli/paths.go): this function's signature returns a bare string, and an unresolved
	// absolute path simply yields a hash computed against the (still-deterministic) input Abs was
	// given, rather than failing a function with nothing to return an error through.
	abs, _ := filepath.Abs(targetDir)
	digest := sha256.Sum256([]byte(filepath.Clean(abs)))
	return filepath.Base(targetDir) + "-" + hex.EncodeToString(digest[:6])
}

// UserCacheDir resolves the base directory quarry's own default state-dir tier resolves to. It is
// built directly on os.UserCacheDir, which already implements the $XDG_CACHE_HOME-when-set-and-
// non-empty, otherwise-~/.cache precedence this suite's production cache_dir argument is derived
// from -- no other function in this package hand-writes that path.
func UserCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("ladder: resolve user cache dir: %w", err)
	}
	return dir, nil
}

// ResolveStateDir resolves <cacheDir>/quarry/<WorkspaceKey(targetDir)>, matching the third
// precedence tier of quarry's own ResolveStateDir (internal/cli/paths.go). The suite never passes
// --state-dir and always clears QUARRY_STATE_DIR via ScrubbedEnv, so the two higher tiers are
// deliberately not modelled; it also models no tags-<hex> segment, because the suite clears
// QUARRY_BUILD_TAGS and never sets buildTags on a call.
//
// ResolveStateDir takes the environment it is resolving for as an explicit env argument rather than
// reading os.Environ(), and returns a *GateError when that slice carries a non-empty value for either
// QUARRY_STATE_DIR or QUARRY_BUILD_TAGS -- not merely a present key, matching resolve_state_dir's own
// truthiness check and quarry's own treatment of an empty value as unset in internal/cli/paths.go.
// This is what stops the gate being asked about an environment the runs were not launched with, which
// would silently resolve a key that is not the one in use. The harness's own process may legitimately
// have either variable exported; that is not what this function resolves against.
func ResolveStateDir(targetDir, cacheDir string, env []string) (string, error) {
	if value, ok := envLookup(env, "QUARRY_STATE_DIR"); ok && value != "" {
		return "", &GateError{Message: "resolve_state_dir: env carries QUARRY_STATE_DIR, which the suite always clears"}
	}
	if value, ok := envLookup(env, "QUARRY_BUILD_TAGS"); ok && value != "" {
		return "", &GateError{Message: "resolve_state_dir: env carries QUARRY_BUILD_TAGS, which the suite always clears"}
	}
	return filepath.Join(cacheDir, "quarry", workspaceKey(targetDir)), nil
}

// DaemonStateFile returns <stateDir>/<lang>/daemon.json, mirroring quarry's own DaemonStateFile
// (internal/quarryengine/daemon/daemonstate.go).
func DaemonStateFile(stateDir, lang string) string {
	return filepath.Join(stateDir, lang, "daemon.json")
}

// daemonState is the JSON shape read from a daemon state file. Only PID is read by anything in this
// file; the remaining fields are carried so a malformed-but-present file still round-trips its other
// data when a future caller needs it, mirroring the Python port's own plain-dict read.
type daemonState struct {
	PID             int    `json:"pid"`
	Address         string `json:"address"`
	ProtocolVersion string `json:"protocol_version"`
	StartedAt       string `json:"started_at"`
}

// pidAlive reports whether a process with this pid exists, per the same os.kill(pid, 0) semantics the
// Python port's _pid_alive used: ESRCH means gone, EPERM means it exists but is owned by someone else,
// anything else propagates as "alive" rather than misclassifying an unexpected errno as "gone". It
// uses a signal-zero probe directly rather than parsing a process listing, so it costs one syscall and
// carries no dependency on any particular ps output format.
func pidAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if err == syscall.ESRCH {
		return false
	}
	return true
}

// readDaemonState returns the parsed daemon.json mapping at the resolved location for lang, or nil
// when it does not exist. It panics if env carries a scrubbed variable ResolveStateDir rejects, or if
// the state file exists but fails to parse as JSON -- both mirror the Python port's own uncaught
// GateError / json.JSONDecodeError propagation out of _read_daemon_state, which this function's
// callers (DaemonAlive, DaemonPID, GateColdBefore, GateColdAfter) are documented as never catching
// either.
func readDaemonState(targetDir, cacheDir string, env []string, lang string) *daemonState {
	stateDir, err := ResolveStateDir(targetDir, cacheDir, env)
	if err != nil {
		panic(err)
	}
	path := DaemonStateFile(stateDir, lang)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		panic(fmt.Errorf("ladder: read daemon state file %s: %w", path, err))
	}
	var state daemonState
	if err := json.Unmarshal(data, &state); err != nil {
		panic(fmt.Errorf("ladder: unmarshal daemon state file %s: %w", path, err))
	}
	return &state
}

// daemonLang is the language segment every gate and daemon helper in this suite resolves against --
// the ladder harness only ever supervises the Go language server, so this is the suite's own default
// for the lang parameter the Python port carried as lang="go" on daemon_alive, daemon_pid, and
// wait_for_daemon_exit alike. DaemonAlive and DaemonPID apply it implicitly; WaitForDaemonExit's own
// signature (card 20) takes lang explicitly instead, since its caller already threads one through.
const daemonLang = "go"

// DaemonAlive is true only when a daemon.json exists at the resolved location and the pid it records
// is alive. This is the suite's definition of "a daemon is running", mirroring quarry's own
// daemonStale, which likewise treats a state file whose pid is dead as stale rather than as a live
// daemon.
func DaemonAlive(targetDir, cacheDir string, env []string) bool {
	state := readDaemonState(targetDir, cacheDir, env, daemonLang)
	return state != nil && pidAlive(state.PID)
}

// DaemonPID returns the pid field of the resolved daemon.json and true, or (0, false) when the file
// is absent.
func DaemonPID(targetDir, cacheDir string, env []string) (int, bool) {
	state := readDaemonState(targetDir, cacheDir, env, daemonLang)
	if state == nil {
		return 0, false
	}
	return state.PID, true
}

// ClearStateDir removes the resolved state directory entirely, ignoring an already-absent directory.
// Called by the cold driver before each attempt so the before-and-after assertions read a directory
// this suite put in a known state, rather than one carrying a previous attempt's leftovers.
func ClearStateDir(targetDir, cacheDir string, env []string) error {
	stateDir, err := ResolveStateDir(targetDir, cacheDir, env)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(stateDir); err != nil {
		return fmt.Errorf("ladder: clear state dir %s: %w", stateDir, err)
	}
	return nil
}

// daemonExitPollInterval is the poll interval for WaitForDaemonExit's liveness loop -- short enough
// to keep the suite's own wait bounded tightly to the daemon's actual exit, without busy-spinning.
const daemonExitPollInterval = 100 * time.Millisecond

// DaemonExitTimeout is the bound callers pass to WaitForDaemonExit, derived from the daemon's own
// 10-minute idle timeout plus a margin.
const DaemonExitTimeout = 660 * time.Second

// WaitForDaemonExit polls the resolved daemon's pid liveness with the same signal-zero probe pidAlive
// uses, until the process is gone, returning immediately when no daemon.json is present. It returns a
// *GateError naming the pid and the bound when timeout elapses first.
//
// The pid is the liveness signal because neither daemon.json nor the state directory is removed on
// exit -- only daemon.sock is -- so file presence says nothing about whether the daemon is still
// running, while daemon.json's recorded pid is exactly what quarry's own staleness check reads.
// Callers pass a bound derived from the daemon's own 10-minute idle timeout plus a margin (see
// DaemonExitTimeout).
func WaitForDaemonExit(targetDir, cacheDir string, env []string, timeout time.Duration, lang string) error {
	pid, ok := readDaemonPID(targetDir, cacheDir, env, lang)
	if !ok {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for pidAlive(pid) {
		if time.Now().After(deadline) {
			return &GateError{Message: fmt.Sprintf("daemon pid %d did not exit within %s", pid, timeout)}
		}
		time.Sleep(daemonExitPollInterval)
	}
	return nil
}

// readDaemonPID returns the pid field of the resolved daemon.json for lang and true, or (0, false)
// when the file is absent. It is DaemonPID's lang-parametrized sibling: DaemonPID always resolves
// against daemonLang, while WaitForDaemonExit (card 20) takes lang explicitly and reads through this
// function instead.
func readDaemonPID(targetDir, cacheDir string, env []string, lang string) (int, bool) {
	state := readDaemonState(targetDir, cacheDir, env, lang)
	if state == nil {
		return 0, false
	}
	return state.PID, true
}

// GateColdBefore is fatal when DaemonAlive is true before a cold run starts, since the daemon is
// already warm and the run cannot be reported as cold.
//
// It keys on liveness rather than on daemon.json's mere existence, because neither daemon.json nor
// the state directory is removed when a daemon exits -- only daemon.sock is, and only by the next
// spawn's stale-socket cleanup. A presence check would make every retry at the same worktree path
// fail deterministically once a prior attempt left a state file behind; ClearStateDir's per-attempt
// state-directory clear makes the precondition deterministic, and this liveness definition is what
// keeps the gate correct even when a stale file survives it.
func GateColdBefore(targetDir, cacheDir string, env []string) []GateFinding {
	if DaemonAlive(targetDir, cacheDir, env) {
		return []GateFinding{{
			Gate:    "cold_before",
			Fatal:   true,
			Message: "a daemon is already alive at this worktree's state directory before the cold run started",
		}}
	}
	return nil
}

// GateColdAfter has three outcomes, not two:
//   - UsedDaemonBackedTool is false: the gate does not apply; returns a single non-fatal
//     cold_no_daemon_backed_call observation, and the run stands as valid while carrying no warmth
//     signal.
//   - UsedDaemonBackedTool is true and daemon.json exists: passes -- the positive confirmation the
//     connection was supervised.
//   - UsedDaemonBackedTool is true and no daemon.json exists: fatal -- the native fallback was taken,
//     on which path the shared daemon address is not a function of the state directory at all, so the
//     run is invalidated rather than reported as cold.
func GateColdAfter(records []Record, targetDir, cacheDir string, env []string) []GateFinding {
	if !UsedDaemonBackedTool(records) {
		return []GateFinding{{
			Gate:    "cold_no_daemon_backed_call",
			Fatal:   false,
			Message: "no daemon-backed tool call observed; cold cell carries no warmth signal for this run",
		}}
	}
	state := readDaemonState(targetDir, cacheDir, env, daemonLang)
	if state != nil {
		return nil
	}
	return []GateFinding{{
		Gate:    "cold_after",
		Fatal:   true,
		Message: "a daemon-backed tool was used but no daemon.json exists; the native fallback was taken",
	}}
}
