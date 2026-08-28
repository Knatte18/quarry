// paths.go resolves the three path axes internal/cli owns: the optional servers.yaml config file,
// the per-workspace state directory the supervised daemon uses for its state file, lock file, and
// socket, and the optional per-directory toc config file.
// Every resolution produces a path only — none reads nor creates anything on disk. An absent
// config file is not an error at this layer; LoadRegistry (for servers.yaml) and loadTOCConfig
// (for the toc config file) are what fall back to their own built-in defaults. The toolchain cache
// (goToolchainCacheDir in the engine's toolchain.go) is a separate, engine-owned path axis and is
// not resolved here.

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Knatte18/quarry/quarry"
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

// resolveBuildTags resolves the normalized build-tag set, in precedence order: flagValue (the
// --build-tags flag), then $QUARRY_BUILD_TAGS, then empty. Whichever raw comma-separated string
// wins is passed to quarry.NormalizeBuildTags, so a raw value that normalizes away entirely —
// "", ",", " , " — yields nil from either source, which every downstream consumer (the
// initializationOptions render, the tags-<hex> state-directory segment, and nativeArgv) treats as
// "no build tags", per the empty-tag-set-is-a-uniform-no-op Shared Decision.
// It shares resolveConfigPath's and resolveStateDir's flag-then-environment-then-default
// resolution shape, which is why it lives beside them rather than in cli.go, even though it is
// the one function in this file that imports beyond the standard library.
func resolveBuildTags(flagValue string) []string {
	raw := flagValue
	if raw == "" {
		raw = os.Getenv("QUARRY_BUILD_TAGS")
	}
	return quarry.NormalizeBuildTags(raw)
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

// buildTagsSegment derives the "tags-<hex>" path segment for a build-tag set: the literal
// "tags-" followed by the first 12 hex characters of the SHA-256 digest of the normalized tags
// joined with ",". It shares workspaceKey's hash-and-6-byte-truncate convention above, so this
// file carries one hashing convention rather than two, and it is factored out here specifically
// so resolveStateDir can call it at all three of its precedence tiers.
// The re-normalization here (via quarry.NormalizeBuildTags, defensive of a caller that has
// already normalized) is what makes two tag sets differing only in input order — []string{"a",
// "b"} and []string{"b", "a"} — hash to the same segment.
func buildTagsSegment(buildTags []string) string {
	sum := sha256.Sum256([]byte(strings.Join(quarry.NormalizeBuildTags(buildTags...), ",")))
	return "tags-" + hex.EncodeToString(sum[:6])
}

// resolveStateDir resolves the leaf state directory for targetDir, in precedence order:
// flagValue (the --state-dir flag), then $QUARRY_STATE_DIR, then
// filepath.Join(userCacheDir(), "quarry", workspaceKey(targetDir)).
// The returned directory is the leaf the engine is told; it carries no "<lang>" segment, because
// DaemonStateFile/DaemonLock join that themselves.
//
// buildTags is the normalized build-tag set (already run through quarry.NormalizeBuildTags by the
// caller). An empty buildTags returns exactly what this function returned before this parameter
// existed, byte for byte, at all three tiers — the empty-tag-set-is-a-uniform-no-op Shared
// Decision. A non-empty buildTags appends buildTagsSegment(buildTags) as one further path segment
// onto the resolved leaf, at every tier including the explicit --state-dir and $QUARRY_STATE_DIR
// tiers, which bypass workspaceKey entirely.
//
// The segment is applied to the resolved leaf here, rather than folded into workspaceKey itself,
// because the first two tiers never call workspaceKey: folding the tag segment in there would
// silently collide two different tag sets onto one socket for an operator who pins --state-dir or
// $QUARRY_STATE_DIR, which is exactly the case this keying exists to prevent.
func resolveStateDir(flagValue, targetDir string, buildTags []string) (string, error) {
	var leaf string
	switch {
	case flagValue != "":
		leaf = flagValue
	case os.Getenv("QUARRY_STATE_DIR") != "":
		// Read once per branch rather than caching to a local first: os.Getenv
		// is cheap, and this keeps each precedence tier self-contained,
		// mirroring resolveConfigPath's own two-read shape for QUARRY_CONFIG.
		leaf = os.Getenv("QUARRY_STATE_DIR")
	default:
		dir, err := userCacheDir()
		if err != nil {
			return "", fmt.Errorf("cli: resolve user cache dir: %w", err)
		}
		leaf = filepath.Join(dir, "quarry", workspaceKey(targetDir))
	}

	if len(buildTags) == 0 {
		return leaf, nil
	}
	return filepath.Join(leaf, buildTagsSegment(buildTags)), nil
}

// resolveTOCConfigPath resolves the optional toc config file path for targetDir, in precedence
// order: $QUARRY_TOC_CONFIG when set, otherwise filepath.Join(targetDir, ".quarry.yaml").
// It resolves a path only and never reads the file — an absent file is not an error at this
// layer, mirroring resolveConfigPath above for the servers.yaml overlay. Unlike resolveConfigPath
// it returns no error, because it never consults a machine-global directory (no os.UserConfigDir
// equivalent) and so has nothing to fail at.
//
// The lookup happens in targetDir and nowhere else — no walk up the directory tree, no
// repository-root search, no project detection. toc has no repository-root concept and never
// detects a project the way the LSP verbs do; an upward search would introduce exactly that
// concept.
//
// For "toc file", targetDir is the file's parent directory, so the resolution is per-directory
// rather than per-invocation. "toc dir" never calls this function: it emits headers only, never
// docstrings, so no setting in this file applies to it. The signature is directory-shaped
// regardless, because that is the natural shape of a per-directory lookup — the "toc dir" case is
// reserved but currently unused, so no caller should be assumed for it.
//
// This file is not servers.yaml. It has its own name and its own environment variable precisely
// so toc never touches the language-server registry, which it needs no part of.
func resolveTOCConfigPath(targetDir string) string {
	if env := os.Getenv("QUARRY_TOC_CONFIG"); env != "" {
		return env
	}
	return filepath.Join(targetDir, ".quarry.yaml")
}
