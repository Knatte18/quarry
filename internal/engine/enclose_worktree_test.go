// enclose_worktree_test.go tests Enclose against scratch working trees.

package engine

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// encloseA is the main fixture file; the comments mark the lines the tests locate.
var encloseA = strings.Join([]string{
	"package enc",             // 1
	"",                        // 2
	"// F does f.",            // 3
	"func F() {",              // 4
	"\tg := func() {",         // 5
	"\t\t_ = 1",               // 6
	"\t}",                     // 7
	"\tg()",                   // 8
	"}",                       // 9
	"",                        // 10
	"type I interface {",      // 11
	"\t// A does a.",          // 12
	"\tA()",                   // 13
	"\tB()",                   // 14
	"}",                       // 15
	"",                        // 16
	"const (",                 // 17
	"\tX = 1",                 // 18
	")",                       // 19
	"",                        // 20
	"type J interface{ M() }", // 21
	"",
}, "\n")

// encloseFixture is the tree most worktree tests run over.
func encloseFixture() map[string]string {
	return map[string]string{
		"enc/a.go":      encloseA,
		"enc/imp.go":    "package enc\n\nimport (\n\t\"fmt\"\n)\n\nfunc H() { fmt.Println() }\n",
		"enc/x_test.go": "package enc_test\n\nfunc TestX() {}\n",
		"enc/lossy.go":  "package enc\n\nfunc Broken(\n\nfunc Recovered() {}\n",
		"enc/empty.go":  "",
		"enc/README.md": "# enc\n",
		"root.go":       "package root\n\nfunc R() {}\n",
	}
}

// loc builds a normalised location whose target echoes file:start-end.
func loc(file string, start, end int) Location {
	return Location{Target: file + ":" + strconv.Itoa(start) + "-" + strconv.Itoa(end), File: file, Start: start, End: end}
}

// encloseOne runs Enclose over one location and returns its single answer.
func encloseOne(t *testing.T, r *Repo, l Location) EncloseResult {
	t.Helper()
	got, err := r.Enclose([]Location{l})
	if err != nil {
		t.Fatalf("Enclose: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Enclose returned %d results; want 1", len(got))
	}
	return got[0]
}

// symbolIDs returns the IDs of res.Symbols in order.
func symbolIDs(res EncloseResult) []string {
	ids := make([]string, 0, len(res.Symbols))
	for _, s := range res.Symbols {
		ids = append(ids, s.ID)
	}
	return ids
}

// assertFound fails unless res is found with exactly the wanted symbol IDs.
func assertFound(t *testing.T, res EncloseResult, want ...string) {
	t.Helper()
	if res.Status != StatusFound {
		t.Fatalf("result = %+v; want found", res)
	}
	if got := strings.Join(symbolIDs(res), ","); got != strings.Join(want, ",") {
		t.Errorf("symbols = %s; want %s", got, strings.Join(want, ","))
	}
	if res.Unit != "" {
		t.Errorf("Unit = %q; want none on found", res.Unit)
	}
}

// assertRejected fails unless res is a rejection with reason and message.
func assertRejected(t *testing.T, res EncloseResult, reason, message string) {
	t.Helper()
	if !res.Rejected() || res.Reason != reason || res.Error != message {
		t.Errorf("result = %+v; want rejection %s: %q", res, reason, message)
	}
	if res.Unit != "" || len(res.Symbols) != 0 {
		t.Errorf("rejection carries Unit %q and Symbols %v; want neither", res.Unit, res.Symbols)
	}
}

func TestEnclose_MemberSelection(t *testing.T) {
	r := openScratchRepo(t, "enclose-members", encloseFixture())
	tests := []struct {
		name       string
		start, end int
		want       []string
	}{
		{"SingleLine", 8, 8, []string{"enc#F"}},
		{"ClosureLine", 6, 6, []string{"enc#F"}},
		{"DocCommentLine", 3, 3, []string{"enc#F"}},
		{"TwoTopLevelMembers", 9, 11, []string{"enc#F", "enc#I"}},
		{"InterfaceMethodLine", 14, 14, []string{"enc#I.B"}},
		{"InterfaceMethodDocLine", 12, 12, []string{"enc#I.A"}},
		{"InterfaceHeadPlusMethod", 11, 13, []string{"enc#I", "enc#I.A"}},
		{"ConstSpecLine", 18, 18, []string{"enc#X"}},
		{"OneLineInterface", 21, 21, []string{"enc#J", "enc#J.M"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := encloseOne(t, r, loc("enc/a.go", tt.start, tt.end))
			assertFound(t, res, tt.want...)
			if res.File != "enc/a.go" || res.Start != tt.start || res.End != tt.end {
				t.Errorf("result = %+v; want file and range echoed", res)
			}
		})
	}
}

func TestEnclose_NotFoundCarriesSelfGlyph(t *testing.T) {
	r := openScratchRepo(t, "enclose-notfound", encloseFixture())
	tests := []struct {
		name       string
		file       string
		start, end int
		unit       string
	}{
		{"ConstGroupHead", "enc/a.go", 17, 17, "enc/a.go#"},
		{"ConstGroupClose", "enc/a.go", 19, 19, "enc/a.go#"},
		{"ImportBlock", "enc/imp.go", 4, 4, "enc/imp.go#"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := encloseOne(t, r, loc(tt.file, tt.start, tt.end))
			if res.Status != StatusNotFound || res.Unit != tt.unit || len(res.Symbols) != 0 {
				t.Errorf("result = %+v; want not_found with unit %q", res, tt.unit)
			}
		})
	}
}

func TestEnclose_LossyFileAnswersWithFlag(t *testing.T) {
	r := openScratchRepo(t, "enclose-lossy", encloseFixture())
	res := encloseOne(t, r, loc("enc/lossy.go", 5, 5))
	if res.Rejected() || !res.Lossy {
		t.Errorf("result = %+v; want an answer with Lossy set", res)
	}
}

func TestEnclose_ExternalTestUnit(t *testing.T) {
	r := openScratchRepo(t, "enclose-xtest", encloseFixture())
	assertFound(t, encloseOne(t, r, loc("enc/x_test.go", 3, 3)), "enc_test#TestX")
}

func TestEnclose_Rejections(t *testing.T) {
	root := writeScratchTree(t, "enclose-rejections", encloseFixture())
	if err := os.Symlink("a.go", filepath.Join(root, "enc", "link.go")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "enc", "bad.go"), []byte("package enc\n\xff\n"), 0o644); err != nil {
		t.Fatalf("write invalid UTF-8 file: %v", err)
	}
	r := openRepo(t, root)

	tests := []struct {
		name    string
		loc     Location
		reason  string
		message string
	}{
		{"PastEOF", loc("enc/a.go", 21, 22), EncloseReasonPastEOF, "range ends past end of file: end 22, file has 21 lines"},
		{"EmptyFile", loc("enc/empty.go", 1, 1), EncloseReasonPastEOF, "range ends past end of file: end 1, file has 0 lines"},
		{"UnsupportedLanguage", loc("enc/README.md", 1, 1), EncloseReasonUnsupportedLanguage, "unsupported language: enc/README.md"},
		{"MissingFile", loc("enc/nope.go", 1, 1), EncloseReasonMissingFile, "file not found: enc/nope.go"},
		{"FileAsDirectorySegment", loc("enc/a.go/b.go", 1, 1), EncloseReasonMissingFile, "file not found: enc/a.go/b.go"},
		{"Directory", loc("enc", 1, 1), EncloseReasonUnsupportedLanguage, "unsupported language: enc"},
		{"Symlink", loc("enc/link.go", 1, 1), EncloseReasonUnreadable, "file unreadable: enc/link.go: symlink"},
		{"InvalidUTF8", loc("enc/bad.go", 1, 1), EncloseReasonUnreadable, "file unreadable: enc/bad.go: engine: enc/bad.go: not valid UTF-8"},
		{"RootLevelFile", loc("root.go", 3, 3), EncloseReasonUnaddressable, "file's glyph unit cannot be spelled: root.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := encloseOne(t, r, tt.loc)
			assertRejected(t, res, tt.reason, tt.message)
			if res.File != tt.loc.File || res.Start != tt.loc.Start || res.End != tt.loc.End || res.Target != tt.loc.Target {
				t.Errorf("rejection = %+v; want Target, File and range carried", res)
			}
		})
	}
}

func TestEnclose_PreRejectedPassesThrough(t *testing.T) {
	r := openScratchRepo(t, "enclose-prerejected", encloseFixture())
	in := Location{Target: "x:0", File: "enc/a.go", Start: 0, End: 0, Reason: EncloseReasonBadRange, Error: "bad range: start 0 must be at least 1 and not after end 0"}
	res := encloseOne(t, r, in)
	want := EncloseResult{Target: in.Target, File: in.File, Reason: in.Reason, Error: in.Error}
	if res.Target != want.Target || res.File != want.File || res.Reason != want.Reason || res.Error != want.Error || res.Status != "" {
		t.Errorf("result = %+v; want %+v", res, want)
	}
}

func TestEnclose_BadItemsDoNotFailTheCall(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 000 does not restrict root")
	}
	root := writeScratchTree(t, "enclose-unreadable", map[string]string{
		"good/a.go":          "package good\n\nfunc G() {}\n",
		"locked/a.go":        "package locked\n\nfunc L() {}\n",
		"ignored/a.go":       "package ignored\n\nfunc I() {}\n",
		"ignored/.gitignore": "*.tmp\n",
	})
	lockedDir := filepath.Join(root, "locked")
	ignoreFile := filepath.Join(root, "ignored", ".gitignore")
	if err := os.Chmod(lockedDir, 0); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := os.Chmod(ignoreFile, 0); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedDir, 0o755)
		_ = os.Chmod(ignoreFile, 0o644)
	})
	r := openRepo(t, root)

	got, err := r.Enclose([]Location{loc("locked/a.go", 3, 3), loc("good/a.go", 3, 3), loc("ignored/a.go", 3, 3)})
	if err != nil {
		t.Fatalf("Enclose: %v", err)
	}
	if got[0].Reason != EncloseReasonUnreadable {
		t.Errorf("locked directory result = %+v; want unreadable", got[0])
	}
	assertFound(t, got[1], "good#G")
	if got[2].Reason != EncloseReasonUnreadable {
		t.Errorf("unreadable .gitignore result = %+v; want unreadable", got[2])
	}
}

// TestEnclose_ExplicitIgnoredFileVotesWithItself pins that the unit comes from the DirAnswer of the
// call that answered the file, whichever order the locations arrive in.
func TestEnclose_ExplicitIgnoredFileVotesWithItself(t *testing.T) {
	files := map[string]string{
		"ign/.gitignore": "extra.go\n",
		"ign/a.go":       "package alpha\n\nfunc A() {}\n",
		"ign/x_test.go":  "package alpha_test\n\nfunc TestX() {}\n",
		"ign/extra.go":   "package aaa\n\nfunc E() {}\n",
	}
	tests := []struct {
		name  string
		order []string
	}{
		{"IgnoredAlone", []string{"ign/extra.go"}},
		{"IgnoredAfterSibling", []string{"ign/x_test.go", "ign/extra.go"}},
		{"IgnoredBeforeSibling", []string{"ign/extra.go", "ign/x_test.go"}},
	}
	want := map[string]string{"ign/extra.go": "ign#E", "ign/x_test.go": "ign_test#TestX"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := openScratchRepo(t, "enclose-ignored-"+tt.name, files)
			var locs []Location
			for _, file := range tt.order {
				locs = append(locs, loc(file, 3, 3))
			}
			got, err := r.Enclose(locs)
			if err != nil {
				t.Fatalf("Enclose: %v", err)
			}
			for i, file := range tt.order {
				assertFound(t, got[i], want[file])
			}
		})
	}
}

func TestEnclose_BatchShape(t *testing.T) {
	r := openScratchRepo(t, "enclose-shape", encloseFixture())

	for _, in := range [][]Location{nil, {}} {
		got, err := r.Enclose(in)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("Enclose(%v) = %v, %v; want an empty non-nil slice and nil error", in, got, err)
		}
	}

	dup := loc("enc/a.go", 8, 8)
	got, err := r.Enclose([]Location{dup, dup})
	if err != nil || len(got) != 2 {
		t.Fatalf("Enclose(duplicate) = %v, %v; want two answers", got, err)
	}
	assertFound(t, got[0], "enc#F")
	assertFound(t, got[1], "enc#F")
}

func TestEnclose_BuildsEachFileOnce(t *testing.T) {
	r := openScratchRepo(t, "enclose-builds", map[string]string{
		"two/a.go": "package two\n\nfunc A() {}\n",
		"two/b.go": "package two\n\nfunc B() {}\n",
	})
	m := newFileMemo(r, true)
	locs := []Location{loc("two/a.go", 3, 3), loc("two/a.go", 1, 3), loc("two/b.go", 3, 3), loc("two/a.go", 3, 3)}
	if _, err := r.enclose(locs, m); err != nil {
		t.Fatalf("enclose: %v", err)
	}
	assertBuilds(t, m.builds, []string{"two/a.go", "two/b.go"})
}

func TestVerifyEncloseCoverage_Panics(t *testing.T) {
	t.Parallel()
	locs := []Location{{Target: "a"}, {Target: "b"}}
	tests := []struct {
		name    string
		results []EncloseResult
	}{
		{"LengthMismatch", []EncloseResult{{Target: "a"}}},
		{"TargetMismatch", []EncloseResult{{Target: "a"}, {Target: "c"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if recover() == nil {
					t.Error("verifyEncloseCoverage did not panic")
				}
			}()
			verifyEncloseCoverage(locs, tt.results)
		})
	}
}
