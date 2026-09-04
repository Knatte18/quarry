// loomyard_test.go is this package's own copy of internal/engine/loomyard_test.go's environment
// gate: loomyardRepo resolves a Loomyard checkout, skips when this machine has none, and fails
// when this machine has the wrong one. It also declares the "-update" flag the after/ goldens use.
//
// This is a deliberate copy, not a shared helper: Go test helpers are not importable across
// packages, and the overview's no-file-under-internal-engine-is-modified decision forbids
// exporting internal/engine's own copy to make it reachable from here.
//
// The skip-versus-fail asymmetry below is deliberate, not an oversight: "this machine has no
// Loomyard" is a normal state every other machine and CI are entitled to be in, so it skips. "This
// machine has the wrong Loomyard" is a checkout silently answering for the wrong commit; a skip
// there would let this task's own done-criterion pass without the byte-for-byte comparison it
// exists to make ever running, so it fails instead.

package cli

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// updateGoldens is "-update", checked by after_test.go's cases to decide whether to compare against
// the committed golden or to rewrite it from the current LADDER_LOOMYARD_REPO checkout. It is this
// package's own flag.Bool, distinct from internal/engine's flag of the same name: flag.Bool panics
// on a duplicate name only within one binary, and each package's tests build their own binary.
var updateGoldens = flag.Bool("update", false, "regenerate the after/ goldens under docs/research/output-formats/after from the current LADDER_LOOMYARD_REPO checkout")

// loomyardPin is the commit the rewrite plan's after/ outputs were taken at, identical to
// internal/engine's own loomyardPin.
const loomyardPin = "72c23d9"

// loomyardRepo returns the absolute path to a Loomyard checkout pinned at loomyardPin, read from
// the LADDER_LOOMYARD_REPO environment variable — never from a tracked file, because no tracked
// file may carry a machine-specific path.
//
// It skips, with the reason, when:
//   - the variable is unset or names a path that is not an existing directory ("this machine has no
//     Loomyard", the normal case on most machines);
//   - the "git" binary is missing or the rev-parse below errors ("no usable checkout" — a checkout
//     git cannot introspect is exactly as unusable as no checkout at all).
//
// It fails, rather than skips, when git succeeds but the checkout's HEAD does not start with
// loomyardPin: see internal/engine/loomyard_test.go's loomyardRepo doc comment for why this
// asymmetry is deliberate.
func loomyardRepo(t *testing.T) string {
	t.Helper()

	repo := os.Getenv("LADDER_LOOMYARD_REPO")
	if repo == "" {
		t.Skip("LADDER_LOOMYARD_REPO is unset; this machine has no Loomyard checkout")
	}
	info, err := os.Stat(repo)
	if err != nil || !info.IsDir() {
		t.Skipf("LADDER_LOOMYARD_REPO %q is not an existing directory; this machine has no Loomyard checkout", repo)
	}

	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Skipf("git rev-parse HEAD in %q failed (%v); no usable Loomyard checkout on this machine", repo, err)
	}

	head := strings.TrimSpace(string(out))
	if !strings.HasPrefix(head, loomyardPin) {
		t.Fatalf("LADDER_LOOMYARD_REPO %q is at %s; want a checkout pinned at %s, the commit the after/ goldens were taken from", repo, head, loomyardPin)
	}

	return repo
}
