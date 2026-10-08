// enclose_golden_test.go pins the enclose verb's JSON payload bytes across every answer shape in
// one batch: found single line and range, not_found, and each rejection reason the batch can reach.
//
// The golden holds the payload bytes and nothing else, per the goldens-are-payload-bytes-only
// decision, and carries no absolute path because every location is repository-relative.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// encloseGoldenGoSource holds a documented function (lines 3-6), an interface with a method
// (lines 8-12), a struct type (lines 14-17) and a const group (lines 19-22).
const encloseGoldenGoSource = `package pkg

// Make builds a Widget.
func Make() Widget {
	return Widget{}
}

// Shape is implemented by drawable things.
type Shape interface {
	// Area reports the covered area.
	Area() float64
}

// Widget is a fixture struct.
type Widget struct {
	Name string
}

const (
	One = 1
	Two = 2
)
`

// encloseGoldenLocations is the fixed input list, one location per answer shape.
var encloseGoldenLocations = []string{
	"pkg/a.go:5",
	"pkg/a.go:6-8",
	"pkg/a.go:9-11",
	"pkg/a.go:19",
	"pkg/a.go:99",
	"pkg/missing.go:1",
	"docs/notes.md:1",
	"root.go:1",
	"not a location",
	"pkg/a.go:5-3",
}

func TestEncloseGolden(t *testing.T) {
	root := writeScratchTree(t, "enclose-golden", map[string]string{
		"pkg/a.go":      encloseGoldenGoSource,
		"docs/notes.md": "# Notes\n",
		"root.go":       "package root\n\nfunc Root() {}\n",
	})

	code, stdout, stderr := runCLIStdin(
		[]string{"enclose", "--root", root, "--stdin"},
		strings.Join(encloseGoldenLocations, "\n")+"\n",
	)
	if code != exitOK {
		t.Fatalf("run(enclose --stdin) code = %d; want %d (stderr %q)", code, exitOK, stderr)
	}
	compareEncloseGolden(t, "batch.json", stdout)
}

// compareEncloseGolden compares got byte for byte against testdata/enclose/name, or, under
// -update, writes got to that path, creating the directory if it does not yet exist.
func compareEncloseGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", "enclose", name)

	if *updateGoldens {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("compareEncloseGolden(%q): mkdir: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("compareEncloseGolden(%q): write: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("compareEncloseGolden(%q): read golden: %v", name, err)
	}
	if string(want) != got {
		t.Errorf("golden %q mismatch (-want +got):\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}
