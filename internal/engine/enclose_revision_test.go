// enclose_revision_test.go tests EncloseFrom against a map-backed revision fake.

package engine

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// fakeRevision is an in-memory RevisionFiles: paths to bytes, with optional injected failures.
type fakeRevision struct {
	files        map[string]string
	goFilesErr   error
	readErr      error
	readFailPath string
}

func (f fakeRevision) GoFiles(dirRel string) ([]string, error) {
	if f.goFilesErr != nil {
		return nil, f.goFilesErr
	}
	var out []string
	for rel := range f.files {
		dir, _ := splitDirBase(rel)
		if dir == dirRel && strings.HasSuffix(rel, ".go") {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (f fakeRevision) Read(rel string) ([]byte, error) {
	if f.readErr != nil && (f.readFailPath == "" || f.readFailPath == rel) {
		return nil, f.readErr
	}
	src, ok := f.files[rel]
	if !ok {
		return nil, fmt.Errorf("fake: no such file %s", rel)
	}
	return []byte(src), nil
}

// revisionWorkingTree is the working tree the revision tests deliberately disagree with.
func revisionWorkingTree() map[string]string {
	return map[string]string{
		"pkg/a.go": "package pkg\n\n\n\n\nfunc Moved() {}\n",
		"pkg/b.go": "package pkg\n\nfunc B() {}\n",
	}
}

func TestEncloseFrom_AnswersFromTheRevision(t *testing.T) {
	r := openScratchRepo(t, "enclose-rev-found", revisionWorkingTree())
	rev := fakeRevision{files: map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Old() {}\n",
		"pkg/b.go": "package pkg\n\nfunc B() {}\n",
	}}

	got, err := r.EncloseFrom(rev, []Location{loc("pkg/a.go", 3, 3)})
	if err != nil {
		t.Fatalf("EncloseFrom: %v", err)
	}
	assertFound(t, got[0], "pkg#Old")
	if got[0].Symbols[0].Start != 3 {
		t.Errorf("symbol Start = %d; want the revision's line 3", got[0].Symbols[0].Start)
	}
	if got[0].Symbols[0].File != "pkg/a.go" {
		t.Errorf("symbol File = %q; want pkg/a.go", got[0].Symbols[0].File)
	}
}

func TestEncloseFrom_Rejections(t *testing.T) {
	r := openScratchRepo(t, "enclose-rev-rejections", revisionWorkingTree())
	rev := fakeRevision{files: map[string]string{
		"pkg/a.go":   "package pkg\n\nfunc A() {}\n",
		"pkg/bad.go": "package pkg\n\xff\n",
		"root.go":    "package root\n\nfunc R() {}\n",
	}}
	tests := []struct {
		name    string
		loc     Location
		reason  string
		message string
	}{
		{"AbsentFromListing", loc("pkg/b.go", 3, 3), EncloseReasonMissingFile, "file not found: pkg/b.go"},
		{"InvalidUTF8", loc("pkg/bad.go", 1, 1), EncloseReasonUnreadable, "file unreadable: pkg/bad.go: engine: pkg/bad.go: not valid UTF-8"},
		{"RootLevelFile", loc("root.go", 3, 3), EncloseReasonUnaddressable, "file's glyph unit cannot be spelled: root.go"},
		{"UnsupportedLanguage", loc("pkg/README.md", 1, 1), EncloseReasonUnsupportedLanguage, "unsupported language: pkg/README.md"},
		{"PastEOF", loc("pkg/a.go", 3, 4), EncloseReasonPastEOF, "range ends past end of file: end 4, file has 3 lines"},
		{"PreRejected", Location{Target: "x", Reason: EncloseReasonBadLocation, Error: "not a location"}, EncloseReasonBadLocation, "not a location"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.EncloseFrom(rev, []Location{tt.loc})
			if err != nil {
				t.Fatalf("EncloseFrom: %v", err)
			}
			assertRejected(t, got[0], tt.reason, tt.message)
		})
	}
}

func TestEncloseFrom_RootLevelFileMatchesWorkingTree(t *testing.T) {
	files := map[string]string{"root.go": "package root\n\nfunc R() {}\n"}
	r := openScratchRepo(t, "enclose-rev-root", files)
	l := loc("root.go", 3, 3)

	atRev, err := r.EncloseFrom(fakeRevision{files: files}, []Location{l})
	if err != nil {
		t.Fatalf("EncloseFrom: %v", err)
	}
	inTree := encloseOne(t, r, l)
	if atRev[0].Reason != EncloseReasonUnaddressable || atRev[0].Error != inTree.Error || atRev[0].Reason != inTree.Reason {
		t.Errorf("revision = %+v, working tree = %+v; want the same unaddressable rejection", atRev[0], inTree)
	}
}

func TestEncloseFrom_UnitFollowsTheRevisionsVote(t *testing.T) {
	r := openScratchRepo(t, "enclose-rev-vote", map[string]string{
		"d/a.go":      "package alpha\n\nfunc A() {}\n",
		"d/a_test.go": "package alpha_test\n\nfunc TestA() {}\n",
	})
	rev := fakeRevision{files: map[string]string{
		"d/a.go":      "package beta\n\nfunc A() {}\n",
		"d/a_test.go": "package alpha_test\n\nfunc TestA() {}\n",
	}}
	got, err := r.EncloseFrom(rev, []Location{loc("d/a_test.go", 3, 3), loc("d/a.go", 3, 3)})
	if err != nil {
		t.Fatalf("EncloseFrom: %v", err)
	}
	assertFound(t, got[0], "d#TestA")
	assertFound(t, got[1], "d#A")
}

func TestEncloseFrom_WholeCallFailures(t *testing.T) {
	r := openScratchRepo(t, "enclose-rev-failures", revisionWorkingTree())
	files := map[string]string{
		"pkg/a.go": "package pkg\n\nfunc A() {}\n",
		"pkg/b.go": "package pkg\n\nfunc B() {}\n",
	}
	boom := errors.New("boom")
	tests := []struct {
		name string
		rev  fakeRevision
	}{
		{"GoFilesFailure", fakeRevision{files: files, goFilesErr: boom}},
		{"ReadFailure", fakeRevision{files: files, readErr: boom}},
		{"ReadFailureOfSibling", fakeRevision{files: files, readErr: boom, readFailPath: "pkg/b.go"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.EncloseFrom(tt.rev, []Location{loc("pkg/a.go", 3, 3)})
			if !errors.Is(err, boom) || got != nil {
				t.Errorf("EncloseFrom = %v, %v; want a nil slice and the injected error", got, err)
			}
		})
	}
}

func TestEncloseFrom_BatchShapeAndBuildCounts(t *testing.T) {
	files := map[string]string{
		"two/a.go": "package two\n\nfunc A() {}\n",
		"two/b.go": "package two\n\nfunc B() {}\n",
	}
	r := openScratchRepo(t, "enclose-rev-builds", files)
	rev := fakeRevision{files: files}

	for _, in := range [][]Location{nil, {}} {
		got, err := r.EncloseFrom(rev, in)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("EncloseFrom(%v) = %v, %v; want an empty non-nil slice and nil error", in, got, err)
		}
	}

	m := newFileMemo(r, true)
	locs := []Location{loc("two/a.go", 3, 3), loc("two/a.go", 1, 3), loc("two/b.go", 3, 3), loc("two/a.go", 3, 3)}
	got, err := r.encloseFrom(rev, locs, m)
	if err != nil {
		t.Fatalf("encloseFrom: %v", err)
	}
	assertFound(t, got[2], "two#B")
	assertBuilds(t, m.builds, []string{"two/a.go", "two/b.go"})
}
