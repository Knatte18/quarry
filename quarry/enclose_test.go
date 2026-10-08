// enclose_test.go covers the location parser, the four file-free rejection checks, path
// normalisation, and Enclose and EncloseAt against a scratch tree and a git fixture.

package quarry

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseLocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		target             string
		wantPath           string
		wantStart, wantEnd int
		wantOK             bool
	}{
		{"Line", "a.go:12", "a.go", 12, 12, true},
		{"Range", "a.go:12-20", "a.go", 12, 20, true},
		{"Column", "a.go:12:5", "a.go", 12, 12, true},
		{"OverflowingColumn", "a.go:12:99999999999999999999999", "a.go", 12, 12, true},
		{"WindowsPath", `C:\x\a.go:12`, `C:\x\a.go`, 12, 12, true},
		{"EmptyPath", ":12", "", 0, 0, false},
		{"MissingLine", "a.go", "", 0, 0, false},
		{"NonNumericLine", "a.go:x", "", 0, 0, false},
		{"OverflowingLine", "a.go:99999999999999999999999", "", 0, 0, false},
		{"OverflowingEnd", "a.go:1-99999999999999999999999", "", 0, 0, false},
		{"DanglingDash", "a.go:12-", "", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path, start, end, ok := parseLocation(tt.target)
			if ok != tt.wantOK || path != tt.wantPath || start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("parseLocation(%q) = %q, %d, %d, %v; want %q, %d, %d, %v",
					tt.target, path, start, end, ok, tt.wantPath, tt.wantStart, tt.wantEnd, tt.wantOK)
			}
		})
	}
}

func TestEnclose_FileFreeRejections(t *testing.T) {
	root := writeScratchTree(t, "enclose-rejections", map[string]string{
		"a/b.go": "package a\n\nfunc F() {}\n",
	})
	r := openDeltaRepo(t, root)
	outside := filepath.Join(filepath.Dir(root), "elsewhere", "x.go") + ":1"

	tests := []struct {
		name     string
		target   string
		reason   string
		wantFile string
		wantErr  string
	}{
		{"NoLine", "a/b.go", EncloseReasonBadLocation, "", "not a location: want path:line, path:line-line or path:line:col: a/b.go"},
		{"NonNumeric", "a/b.go:x", EncloseReasonBadLocation, "", "not a location: want path:line, path:line-line or path:line:col: a/b.go:x"},
		{"ZeroStart", "a/b.go:0", EncloseReasonBadRange, "", "bad range: start 0 must be at least 1 and not after end 0"},
		{"ReversedRange", "a/b.go:5-2", EncloseReasonBadRange, "", "bad range: start 5 must be at least 1 and not after end 2"},
		{"EscapingRelative", "../x.go:1", EncloseReasonOutsideRoot, "", "path outside repository: ../x.go"},
		{"AbsoluteOutside", outside, EncloseReasonOutsideRoot, "", "path outside repository: " + filepath.ToSlash(outside[:len(outside)-2])},
		{"RootItself", ".:1", EncloseReasonUnaddressable, ".", "path cannot be spelled as a glyph unit: ."},
		{"SpaceInSegment", "a b/c.go:1", EncloseReasonUnaddressable, "a b/c.go", "path cannot be spelled as a glyph unit: a b/c.go"},
		{"HashInSegment", "a#b/c.go:1", EncloseReasonUnaddressable, "", "path cannot be spelled as a glyph unit: a#b/c.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Enclose([]string{tt.target})
			if err != nil {
				t.Fatalf("Enclose(%q) error = %v", tt.target, err)
			}
			if len(got) != 1 {
				t.Fatalf("Enclose(%q) returned %d answers; want 1", tt.target, len(got))
			}
			res := got[0]
			if res.Target != tt.target || res.Reason != tt.reason || res.File != tt.wantFile || res.Status != "" {
				t.Errorf("Enclose(%q) = %+v; want reason %q, file %q, no status", tt.target, res, tt.reason, tt.wantFile)
			}
			if res.Error != tt.wantErr {
				t.Errorf("Enclose(%q).Error = %q; want %q", tt.target, res.Error, tt.wantErr)
			}
		})
	}
}

func TestEnclose_NormalisesPaths(t *testing.T) {
	root := writeScratchTree(t, "enclose-normalise", map[string]string{
		"a/b.go": "package a\n\nfunc F() {}\n",
	})
	r := openDeltaRepo(t, root)

	for _, target := range []string{
		filepath.Join(root, "a", "b.go") + ":3",
		"./a/../a/b.go:3",
	} {
		got, err := r.Enclose([]string{target})
		if err != nil {
			t.Fatalf("Enclose(%q) error = %v", target, err)
		}
		if got[0].File != "a/b.go" || got[0].Status != StatusFound {
			t.Errorf("Enclose(%q) = %+v; want found with file a/b.go", target, got[0])
		}
	}
}

func TestEnclose_EqualsEncloseAtEmptyRevision(t *testing.T) {
	root := writeScratchTree(t, "enclose-equals", map[string]string{
		"a/b.go": "package a\n\nfunc F() {}\n",
	})
	r := openDeltaRepo(t, root)
	targets := []string{"a/b.go:3", "a/b.go:1", "nope", "a/missing.go:1"}

	want, err := r.EncloseAt("", targets)
	if err != nil {
		t.Fatalf("EncloseAt(\"\") error = %v", err)
	}
	got, err := r.Enclose(targets)
	if err != nil {
		t.Fatalf("Enclose() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Enclose() = %+v; want %+v", got, want)
	}

	empty, err := r.Enclose(nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("Enclose(nil) = %#v, %v; want an empty non-nil slice", empty, err)
	}
}

func TestEncloseAt_Revision(t *testing.T) {
	t.Run("MovedMemberAnswersAtCommittedLines", func(t *testing.T) {
		f := newDeltaFixture(t)
		f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc A() {}\n\nfunc B() {}\n", "base")
		f.write("pkg/a.go", "package pkg\n\n\n\n\n\nfunc B() {}\n\nfunc A() {}\n")
		r := openDeltaRepo(t, f.root)

		atRev, err := r.EncloseAt("HEAD", []string{"pkg/a.go:3"})
		if err != nil {
			t.Fatalf("EncloseAt(HEAD) error = %v", err)
		}
		if atRev[0].Status != StatusFound || len(atRev[0].Symbols) != 1 || atRev[0].Symbols[0].ID != "pkg#A" {
			t.Errorf("EncloseAt(HEAD, a.go:3) = %+v; want found pkg#A", atRev[0])
		}
		now, err := r.Enclose([]string{"pkg/a.go:3"})
		if err != nil {
			t.Fatalf("Enclose() error = %v", err)
		}
		if now[0].Status != StatusNotFound {
			t.Errorf("Enclose(a.go:3) = %+v; want not_found in the moved working tree", now[0])
		}
	})

	t.Run("FileAbsentAtRevision", func(t *testing.T) {
		f := newDeltaFixture(t)
		f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc A() {}\n", "base")
		f.write("pkg/new.go", "package pkg\n\nfunc N() {}\n")
		r := openDeltaRepo(t, f.root)

		got, err := r.EncloseAt("HEAD", []string{"pkg/new.go:3"})
		if err != nil {
			t.Fatalf("EncloseAt(HEAD) error = %v", err)
		}
		if got[0].Reason != EncloseReasonMissingFile {
			t.Errorf("EncloseAt(HEAD, new.go:3) = %+v; want reason %q", got[0], EncloseReasonMissingFile)
		}
	})

	t.Run("UnknownRevisionFailsTheCall", func(t *testing.T) {
		f := newDeltaFixture(t)
		f.writeAndCommit("pkg/a.go", "package pkg\n\nfunc A() {}\n", "base")
		r := openDeltaRepo(t, f.root)

		got, err := r.EncloseAt("no-such-rev", []string{"pkg/a.go:3"})
		if !errors.Is(err, ErrUnknownRevision) {
			t.Errorf("EncloseAt(no-such-rev) error = %v; want errors.Is(err, ErrUnknownRevision)", err)
		}
		if got != nil {
			t.Errorf("EncloseAt(no-such-rev) = %+v; want nil answers", got)
		}
	})

	t.Run("DominantClauseComesFromTheRevision", func(t *testing.T) {
		f := newDeltaFixture(t)
		f.write("pkg/a.go", "package foo\n\nfunc A() {}\n")
		f.write("pkg/b.go", "package foo\n\nfunc B() {}\n")
		f.write("pkg/c.go", "package foo_test\n\nfunc C() {}\n")
		f.commit("base")
		f.write("pkg/a.go", "package foo_test\n\nfunc A() {}\n")
		f.write("pkg/b.go", "package foo_test\n\nfunc B() {}\n")
		r := openDeltaRepo(t, f.root)

		atRev, err := r.EncloseAt("HEAD", []string{"pkg/c.go:3"})
		if err != nil {
			t.Fatalf("EncloseAt(HEAD) error = %v", err)
		}
		if atRev[0].Status != StatusFound || atRev[0].Symbols[0].ID != "pkg_test#C" {
			t.Errorf("EncloseAt(HEAD, c.go:3) = %+v; want found pkg_test#C", atRev[0])
		}
		now, err := r.Enclose([]string{"pkg/c.go:3"})
		if err != nil {
			t.Fatalf("Enclose() error = %v", err)
		}
		if now[0].Status != StatusFound || now[0].Symbols[0].ID != "pkg#C" {
			t.Errorf("Enclose(c.go:3) = %+v; want found pkg#C", now[0])
		}
	})

	t.Run("RootLevelFileIsUnaddressable", func(t *testing.T) {
		f := newDeltaFixture(t)
		f.writeAndCommit("root.go", "package main\n\nfunc R() {}\n", "base")
		r := openDeltaRepo(t, f.root)

		atRev, err := r.EncloseAt("HEAD", []string{"root.go:3"})
		if err != nil {
			t.Fatalf("EncloseAt(HEAD) error = %v", err)
		}
		now, err := r.Enclose([]string{"root.go:3"})
		if err != nil {
			t.Fatalf("Enclose() error = %v", err)
		}
		if atRev[0].Reason != EncloseReasonUnaddressable || !reflect.DeepEqual(atRev, now) {
			t.Errorf("EncloseAt(HEAD) = %+v, Enclose() = %+v; want equal unaddressable answers", atRev, now)
		}
		if atRev[0].Error != "file's glyph unit cannot be spelled: root.go" {
			t.Errorf("Error = %q; want the unit-half text", atRev[0].Error)
		}
	})
}
