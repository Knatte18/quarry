// loomyard_test.go is the environment gate the Loomyard-dependent goldens and round trip run
// through: loomyardRepo resolves the checkout, skips when this machine has none, and fails when
// this machine has the wrong one. It also declares the "-update" flag those goldens use, so exactly
// one file in this package owns it.

package engine

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// updateGoldens is "-update", checked by golden_test.go's cases to decide whether to compare
// against the committed golden or to rewrite it from the current LADDER_LOOMYARD_REPO checkout.
// Declared here, once, so no other file needs its own flag.Bool for the same name.
var updateGoldens = flag.Bool("update", false, "regenerate the Loomyard goldens under testdata/loomyard from the current LADDER_LOOMYARD_REPO checkout")

// loomyardPin is the commit the rewrite plan's §4 examples were taken at. The Loomyard-dependent
// goldens and round trip only mean anything against this exact commit — an example transcribed by
// hand from a later or earlier checkout would silently drift from the source it claims to reproduce.
const loomyardPin = "72c23d9"

// loomyardRepo returns the absolute path to a Loomyard checkout pinned at loomyardPin, read from
// the LADDER_LOOMYARD_REPO environment variable — never from a tracked file, because no tracked
// file may carry a machine-specific path. The per-machine value lives in the gitignored
// .scratch/ladder.env, recreated per machine, and is not this repository's to track.
//
// It skips, with the reason, when:
//   - the variable is unset or names a path that is not an existing directory ("this machine has no
//     Loomyard", the normal case on most machines);
//   - the "git" binary is missing or the rev-parse below errors ("no usable checkout" — a checkout
//     git cannot introspect is exactly as unusable as no checkout at all).
//
// It fails, rather than skips, when git succeeds but the checkout's HEAD does not start with
// loomyardPin. This asymmetry is deliberate: "this machine has no Loomyard" is a normal, expected
// state that every other machine and CI are entitled to be in, but "this machine has the wrong
// Loomyard" is a checkout present and reachable yet silently answering for the wrong commit — a
// skip there would let this task's own done-criterion pass without the byte-for-byte comparison it
// exists to make ever actually running.
//
// The pin is read by running "git -C <repo> rev-parse HEAD" rather than by reading the checkout's
// own .git/HEAD file directly, because the checkout this task's goldens were produced against is a
// git *worktree*: a worktree's ".git" is a file containing a "gitdir:" pointer into the parent
// repository's worktrees directory, not a ref file itself, so reading it directly would either
// misparse that pointer or, at best, resolve nothing at all. Asking git resolves this correctly
// regardless of whether the checkout is a worktree or an ordinary clone.
func loomyardRepo(t testing.TB) string {
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
		t.Fatalf("LADDER_LOOMYARD_REPO %q is at %s; want a checkout pinned at %s, the commit the rewrite plan's §4 examples were taken from", repo, head, loomyardPin)
	}

	return repo
}
