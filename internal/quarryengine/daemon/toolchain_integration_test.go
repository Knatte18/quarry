//go:build lsp

// toolchain_integration_test.go exercises resolveGoToolchain against a real
// `go install`, mirroring refs_integration_test.go's header-comment style:
// the tag names its real precondition, a real language-server binary on
// $PATH, so this file is excluded from the plain `go test` verify and run
// separately with `-tags lsp`. Unlike refs_integration_test.go's
// gopls-presence skip gate, this test has no natural skip condition — it
// needs only the go toolchain itself, which is guaranteed present in any
// environment where `go test -tags lsp` can even run — so it runs
// unconditionally under the tag; there is deliberately no t.Skip here.
// This test spawns `go install` and the freshly installed gopls binary but
// no git, so it needs no git-environment isolation.

package daemon

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine/registry"
)

func TestResolveGoToolchain_Integration(t *testing.T) {
	dir := t.TempDir()
	original := UserCacheDir
	UserCacheDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { UserCacheDir = original })

	// InstallGoToolchain stays at its production value (runGoInstall) here:
	// this test's entire point is proving the real `go install` path
	// produces a working gopls binary, not exercising the mocked seam
	// toolchain_test.go already covers.

	// 120 seconds is generous enough for a real module-proxy fetch and
	// build of gopls on a cold module cache.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	binPath, err := resolveGoToolchain(ctx, registry.BuiltinRegistry()["go"].PinnedVersion)
	if err != nil {
		t.Fatalf("resolveGoToolchain() error = %v; want nil", err)
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v; want the installed gopls binary to exist", binPath, err)
	}

	// This file's own exec.Command call below is a direct subprocess spawn
	// in test code, proving the installed binary is actually a working
	// gopls, not just a file that happens to exist at the expected path.
	if err := exec.Command(binPath, "version").Run(); err != nil {
		t.Fatalf("exec.Command(%q, %q).Run() error = %v; want the freshly installed gopls to report its version", binPath, "version", err)
	}
}
