// paths.go resolves the two path axes internal/cli owns: the optional servers.yaml config file and
// the per-workspace state directory the supervised daemon uses for its state file, lock file, and
// socket.
// Both resolutions produce a path only — neither reads nor creates anything on disk. An absent
// config file is not an error at this layer; LoadRegistry is what falls back to the built-in
// registry. The toolchain cache (goToolchainCacheDir in the engine's toolchain.go) is a separate,
// engine-owned path axis and is not resolved here.

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// userConfigDir is the seam resolveConfigPath calls through instead of os.UserConfigDir()
// directly, so tests can redirect resolution at a t.TempDir() without ever touching the real
// machine-global config directory. Production leaves it at the stdlib default.
var userConfigDir = os.UserConfigDir

// userCacheDir is the seam resolveStateDir calls through instead of os.UserCacheDir() directly,
// so tests can redirect resolution at a t.TempDir() without ever touching the real machine-global
// cache directory. Production leaves it at the stdlib default.
var userCacheDir = os.UserCacheDir

// resolveConfigPath resolves the servers.yaml config file path, in precedence order: flagValue
// (the --config flag), then $QUARRY_CONFIG, then filepath.Join(userConfigDir(), "quarry",
// "servers.yaml").
// It resolves a path only and never reads the file — an absent file is not an error at this
// layer, because LoadRegistry is what falls back to the built-in registry.
func resolveConfigPath(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := os.Getenv("QUARRY_CONFIG"); env != "" {
		return env, nil
	}
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("cli: resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "quarry", "servers.yaml"), nil
}

// workspaceKey derives a stable, human-legible directory name for targetDir: the target
// directory's base name, followed by the first 12 hex characters of the SHA-256 digest of the
// cleaned absolute targetDir.
// The hash component is what keeps two different directories that happen to share a basename
// (e.g. two repos both named "app" cloned into different parents) from colliding on the same
// state directory; the basename prefix is what keeps the directory human-legible in a directory
// listing.
func workspaceKey(targetDir string) string {
	// filepath.Abs's error is ignored here, mirroring goToolchainCacheDir's own
	// treatment of userCacheDir()'s error in toolchain.go: this function's
	// signature returns a bare string, and an unresolved absolute path simply
	// yields a hash computed against the (still-deterministic) input Abs was
	// given, rather than failing a function with nothing to return an error
	// through.
	abs, _ := filepath.Abs(targetDir)
	sum := sha256.Sum256([]byte(filepath.Clean(abs)))
	return filepath.Base(targetDir) + "-" + hex.EncodeToString(sum[:6])
}

// resolveStateDir resolves the leaf state directory for targetDir, in precedence order:
// flagValue (the --state-dir flag), then $QUARRY_STATE_DIR, then
// filepath.Join(userCacheDir(), "quarry", workspaceKey(targetDir)).
// The returned directory is the leaf the engine is told; it carries no "<lang>" segment, because
// DaemonStateFile/DaemonLock join that themselves.
func resolveStateDir(flagValue, targetDir string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := os.Getenv("QUARRY_STATE_DIR"); env != "" {
		return env, nil
	}
	dir, err := userCacheDir()
	if err != nil {
		return "", fmt.Errorf("cli: resolve user cache dir: %w", err)
	}
	return filepath.Join(dir, "quarry", workspaceKey(targetDir)), nil
}
