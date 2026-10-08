// enclose_test.go pins the enclose verb's pipeline through run: location sources (arguments and
// standard input), usage rejections, per-item answers carried with exit 0, the root-relative path
// frame, and the revision answer path.

package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// encloseItem is the subset of one enclose answer these tests assert on.
type encloseItem struct {
	Target  string `json:"target"`
	File    string `json:"file"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Error   string `json:"error"`
	Symbols []struct {
		ID string `json:"id"`
	} `json:"symbols"`
}

// encloseGoSource declares two free functions: A on lines 3 to 6 including its doc line, B on line 8.
const encloseGoSource = "package pkg\n\n// A does a.\nfunc A() {\n\t_ = 1\n}\n\nfunc B() {}\n"

// runCLIStdin is runCLI with the given text as standard input.
func runCLIStdin(args []string, stdin string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	code = run(args, strings.NewReader(stdin), &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// decodeEnclose decodes stdout as the enclose answer array.
func decodeEnclose(t *testing.T, stdout string) []encloseItem {
	t.Helper()
	var items []encloseItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", stdout, err)
	}
	return items
}

func TestRunEnclose_PositionalLocations(t *testing.T) {
	t.Parallel()
	root := writeScratchTree(t, "enclose-positional", map[string]string{"pkg/a.go": encloseGoSource})

	code, stdout, stderr := runCLIStdin([]string{"enclose", "--root", root, "pkg/a.go:5", "pkg/a.go:8"}, "")
	if code != exitOK {
		t.Fatalf("code = %d; want %d (stderr %q)", code, exitOK, stderr)
	}
	items := decodeEnclose(t, stdout)
	if len(items) != 2 {
		t.Fatalf("len(items) = %d; want 2 in %q", len(items), stdout)
	}
	for i, wantID := range []string{"pkg#A", "pkg#B"} {
		if items[i].Status != "found" || len(items[i].Symbols) != 1 || items[i].Symbols[0].ID != wantID {
			t.Errorf("items[%d] = %+v; want found with symbol %q", i, items[i], wantID)
		}
	}
}

func TestRunEnclose_Stdin(t *testing.T) {
	t.Parallel()
	root := writeScratchTree(t, "enclose-stdin", map[string]string{"pkg/a.go": encloseGoSource})

	t.Run("BlankLinesAndCRLF", func(t *testing.T) {
		t.Parallel()
		code, stdout, stderr := runCLIStdin([]string{"enclose", "--root", root, "--stdin"},
			"pkg/a.go:5\r\n\r\n  \n\tpkg/a.go:8  \r\n")
		if code != exitOK {
			t.Fatalf("code = %d; want %d (stderr %q)", code, exitOK, stderr)
		}
		items := decodeEnclose(t, stdout)
		if len(items) != 2 || items[0].Target != "pkg/a.go:5" || items[1].Target != "pkg/a.go:8" {
			t.Errorf("items = %+v; want the two trimmed targets in order", items)
		}
	})

	t.Run("EmptyInputAnswersEmptyArray", func(t *testing.T) {
		t.Parallel()
		code, stdout, _ := runCLIStdin([]string{"enclose", "--root", root, "--stdin"}, "")
		if code != exitOK {
			t.Fatalf("code = %d; want %d", code, exitOK)
		}
		if stdout != "[]\n" {
			t.Errorf("stdout = %q; want %q", stdout, "[]\n")
		}
	})
}

func TestRunEnclose_UsageErrors(t *testing.T) {
	t.Parallel()
	root := writeScratchTree(t, "enclose-usage", map[string]string{"pkg/a.go": encloseGoSource})

	tests := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{"BothLocationsAndStdin", []string{"enclose", "--root", root, "--stdin", "pkg/a.go:5"}, "pkg/a.go:5\n", "enclose takes locations as arguments or --stdin, not both"},
		{"Neither", []string{"enclose", "--root", root}, "", "enclose requires at least one location or --stdin"},
		{"Text", []string{"enclose", "--root", root, "--text", "pkg/a.go:5"}, "", "--text is not valid for enclose"},
		{"RevOnToc", []string{"toc", "--root", root, "--rev", "HEAD", "pkg"}, "", "--rev is not valid for toc"},
		{"EmptyRev", []string{"enclose", "--root", root, "--rev", "", "pkg/a.go:5"}, "", "--rev value must not be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, stdout, stderr := runCLIStdin(tt.args, tt.stdin)
			if code != exitUsage {
				t.Fatalf("code = %d; want %d", code, exitUsage)
			}
			if got := failureEnvelope(t, stdout); got != tt.want {
				t.Errorf("envelope error = %q; want %q", got, tt.want)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Errorf("stderr = %q; want it to contain %q", stderr, tt.want)
			}
		})
	}
}

func TestRunEnclose_PerItemAnswersExitZero(t *testing.T) {
	t.Parallel()
	root := writeScratchTree(t, "enclose-peritem", map[string]string{"pkg/a.go": encloseGoSource})

	t.Run("BareFileNameIsMissingFile", func(t *testing.T) {
		t.Parallel()
		code, stdout, _ := runCLIStdin([]string{"enclose", "--root", root, "foo_test.go:42"}, "")
		if code != exitOK {
			t.Fatalf("code = %d; want %d", code, exitOK)
		}
		items := decodeEnclose(t, stdout)
		if len(items) != 1 || items[0].Reason != "missing_file" {
			t.Errorf("items = %+v; want one missing_file rejection", items)
		}
	})

	t.Run("LineAndColumn", func(t *testing.T) {
		t.Parallel()
		code, stdout, _ := runCLIStdin([]string{"enclose", "--root", root, "pkg/a.go:5:2"}, "")
		if code != exitOK {
			t.Fatalf("code = %d; want %d", code, exitOK)
		}
		items := decodeEnclose(t, stdout)
		if len(items) != 1 || items[0].Status != "found" || items[0].Start != 5 || items[0].End != 5 ||
			len(items[0].Symbols) != 1 || items[0].Symbols[0].ID != "pkg#A" {
			t.Errorf("items = %+v; want found A over line 5", items)
		}
	})
}

// TestRunEnclose_SubdirectoryResolvesAgainstRoot cannot run in parallel: t.Chdir changes the
// process working directory.
func TestRunEnclose_SubdirectoryResolvesAgainstRoot(t *testing.T) {
	f := newDeltaCLIFixture(t)
	f.writeAndCommit("pkg/a.go", encloseGoSource, "base")
	t.Chdir(f.root + "/pkg")

	code, stdout, stderr := runCLIStdin([]string{"enclose", "pkg/a.go:5"}, "")
	if code != exitOK {
		t.Fatalf("code = %d; want %d (stderr %q)", code, exitOK, stderr)
	}
	items := decodeEnclose(t, stdout)
	if len(items) != 1 || items[0].File != "pkg/a.go" || items[0].Status != "found" {
		t.Errorf("items = %+v; want found in pkg/a.go resolved against the root, not the cwd", items)
	}
}

func TestRunEnclose_Revision(t *testing.T) {
	t.Parallel()

	t.Run("UnknownRevision", func(t *testing.T) {
		t.Parallel()
		f := newDeltaCLIFixture(t)
		f.writeAndCommit("pkg/a.go", encloseGoSource, "base")

		code, stdout, _ := runCLIStdin([]string{"enclose", "--root", f.root, "--rev", "no-such-rev", "pkg/a.go:5"}, "")
		if code != exitUsage {
			t.Fatalf("code = %d; want %d", code, exitUsage)
		}
		if got, want := failureEnvelope(t, stdout), "enclose: unknown revision no-such-rev"; got != want {
			t.Errorf("envelope error = %q; want %q", got, want)
		}
	})

	t.Run("KnownRevisionAnswersAtCommittedLines", func(t *testing.T) {
		t.Parallel()
		f := newDeltaCLIFixture(t)
		rev := f.writeAndCommit("pkg/a.go", encloseGoSource, "base")
		f.write("pkg/a.go", "package pkg\n\n\n\n\n\nfunc B() {}\n\n// A does a.\nfunc A() {\n\t_ = 1\n}\n")

		code, stdout, stderr := runCLIStdin([]string{"enclose", "--root", f.root, "--rev", rev, "pkg/a.go:5"}, "")
		if code != exitOK {
			t.Fatalf("code = %d; want %d (stderr %q)", code, exitOK, stderr)
		}
		items := decodeEnclose(t, stdout)
		if len(items) != 1 || len(items[0].Symbols) != 1 || items[0].Symbols[0].ID != "pkg#A" {
			t.Errorf("items = %+v; want A at the committed line 5", items)
		}

		code, stdout, _ = runCLIStdin([]string{"enclose", "--root", f.root, "pkg/a.go:5"}, "")
		if code != exitOK {
			t.Fatalf("working-tree code = %d; want %d", code, exitOK)
		}
		if items := decodeEnclose(t, stdout); len(items) != 1 || items[0].Status != "not_found" {
			t.Errorf("working-tree items = %+v; want line 5 outside every member", items)
		}
	})
}
