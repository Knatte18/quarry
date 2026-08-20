// paths_test.go exercises resolveConfigPath and resolveStateDir's precedence chains, workspaceKey's
// determinism and collision-avoidance, and the socket-path-length guard the supervised daemon
// strategy depends on.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempUserConfigDir redirects userConfigDir at a fresh t.TempDir() for the duration of the
// test, restoring the original in a t.Cleanup.
func withTempUserConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	original := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = original })
	return dir
}

// withTempUserCacheDir redirects userCacheDir at a fresh t.TempDir() for the duration of the
// test, restoring the original in a t.Cleanup.
func withTempUserCacheDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	original := userCacheDir
	userCacheDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userCacheDir = original })
	return dir
}

func TestResolveConfigPath_Precedence(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		envValue  string
		wantFn    func(tempConfigDir string) string
	}{
		{
			name:      "flag beats env and default",
			flagValue: "/flag/servers.yaml",
			envValue:  "/env/servers.yaml",
			wantFn:    func(string) string { return "/flag/servers.yaml" },
		},
		{
			name:      "env beats default",
			flagValue: "",
			envValue:  "/env/servers.yaml",
			wantFn:    func(string) string { return "/env/servers.yaml" },
		},
		{
			name:      "default tier",
			flagValue: "",
			envValue:  "",
			wantFn: func(tempConfigDir string) string {
				return filepath.Join(tempConfigDir, "quarry", "servers.yaml")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempConfigDir := withTempUserConfigDir(t)
			t.Setenv("QUARRY_CONFIG", tt.envValue)

			got, err := resolveConfigPath(tt.flagValue)
			if err != nil {
				t.Fatalf("resolveConfigPath(%q) error = %v; want nil", tt.flagValue, err)
			}
			want := tt.wantFn(tempConfigDir)
			if got != want {
				t.Errorf("resolveConfigPath(%q) = %q; want %q", tt.flagValue, got, want)
			}
		})
	}
}

func TestResolveConfigPath_UserConfigDirError(t *testing.T) {
	t.Parallel()

	original := userConfigDir
	wantErr := os.ErrPermission
	userConfigDir = func() (string, error) { return "", wantErr }
	t.Cleanup(func() { userConfigDir = original })

	_, err := resolveConfigPath("")
	if err == nil {
		t.Fatal("resolveConfigPath(\"\") error = nil; want non-nil when userConfigDir() errors")
	}
}

func TestResolveStateDir_Precedence(t *testing.T) {
	const targetDir = "/some/project/dir"

	tests := []struct {
		name      string
		flagValue string
		envValue  string
		wantFn    func(tempCacheDir string) string
	}{
		{
			name:      "flag beats env and default",
			flagValue: "/flag/state",
			envValue:  "/env/state",
			wantFn:    func(string) string { return "/flag/state" },
		},
		{
			name:      "env beats default",
			flagValue: "",
			envValue:  "/env/state",
			wantFn:    func(string) string { return "/env/state" },
		},
		{
			name:      "default tier",
			flagValue: "",
			envValue:  "",
			wantFn: func(tempCacheDir string) string {
				return filepath.Join(tempCacheDir, "quarry", workspaceKey(targetDir))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempCacheDir := withTempUserCacheDir(t)
			t.Setenv("QUARRY_STATE_DIR", tt.envValue)

			got, err := resolveStateDir(tt.flagValue, targetDir)
			if err != nil {
				t.Fatalf("resolveStateDir(%q, %q) error = %v; want nil", tt.flagValue, targetDir, err)
			}
			want := tt.wantFn(tempCacheDir)
			if got != want {
				t.Errorf("resolveStateDir(%q, %q) = %q; want %q", tt.flagValue, targetDir, got, want)
			}
		})
	}
}

func TestResolveStateDir_UserCacheDirError(t *testing.T) {
	t.Parallel()

	original := userCacheDir
	wantErr := os.ErrPermission
	userCacheDir = func() (string, error) { return "", wantErr }
	t.Cleanup(func() { userCacheDir = original })

	_, err := resolveStateDir("", "/some/project")
	if err == nil {
		t.Fatal("resolveStateDir(\"\", ...) error = nil; want non-nil when userCacheDir() errors")
	}
}

// TestWorkspaceKey_DeterministicForSameAbsoluteDir verifies that repeated calls for the same
// absolute directory produce identical keys.
func TestWorkspaceKey_DeterministicForSameAbsoluteDir(t *testing.T) {
	t.Parallel()

	const dir = "/repos/quarry"
	first := workspaceKey(dir)
	second := workspaceKey(dir)
	if first != second {
		t.Errorf("workspaceKey(%q) = %q, then %q; want identical across repeated calls", dir, first, second)
	}
}

// TestWorkspaceKey_DistinctForDifferentDirsSharingBasename verifies that two different absolute
// directories that share a basename produce distinct keys.
func TestWorkspaceKey_DistinctForDifferentDirsSharingBasename(t *testing.T) {
	t.Parallel()

	first := workspaceKey("/repos/alice/app")
	second := workspaceKey("/repos/bob/app")
	if first == second {
		t.Errorf("workspaceKey() for two different dirs sharing basename %q; want distinct keys, got equal", first)
	}
}

// TestResolveStateDir_SocketPathStaysUnderSockaddrUnLimit guards a real pre-existing exposure:
// the supervised daemon's socket path is derived from the state directory, and
// os.UserCacheDir() plus a workspace key plus "<lang>/daemon.sock" can approach the 108-byte
// Linux sockaddr_un limit.
// The simulated cache root here is a realistic os.UserCacheDir() answer (a Linux
// "$HOME/.cache" path under a plausible username), not a t.TempDir() — a real test-harness
// temp directory name is far longer than any real machine's cache root and would make this
// assertion fail for a reason that has nothing to do with the exposure being guarded.
func TestResolveStateDir_SocketPathStaysUnderSockaddrUnLimit(t *testing.T) {
	original := userCacheDir
	const simulatedCacheDir = "/home/developer/.cache"
	userCacheDir = func() (string, error) { return simulatedCacheDir, nil }
	t.Cleanup(func() { userCacheDir = original })

	const sockaddrUnLimit = 108

	// A realistically deep target path: many nested directories under the same
	// home directory. Depth alone must not matter, because workspaceKey derives
	// its directory name from targetDir's basename plus a fixed-length hash,
	// discarding everything else about the path's shape.
	targetDir := filepath.Join("/home", "developer", "workspaces", "engineering", "monorepo-checkouts", "backend-service")

	stateDir, err := resolveStateDir("", targetDir)
	if err != nil {
		t.Fatalf("resolveStateDir(\"\", %q) error = %v; want nil", targetDir, err)
	}
	socketPath := filepath.Join(stateDir, "go", "daemon.sock")
	if len(socketPath) >= sockaddrUnLimit {
		t.Errorf("socket path %q is %d bytes; want under the Linux sockaddr_un limit of %d bytes", socketPath, len(socketPath), sockaddrUnLimit)
	}
	if !strings.HasSuffix(socketPath, filepath.Join("go", "daemon.sock")) {
		t.Errorf("socket path %q does not end with %q", socketPath, filepath.Join("go", "daemon.sock"))
	}
}
