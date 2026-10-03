// memo_test.go pins that each file is built at most once per Resolve call, whatever mix of self and
// member targets reaches its directory, and that sharing the memo never changes an answer.

package engine

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// assertBuilds fails unless every path in want was built exactly once and builds holds no other path.
// The expected set is written out from the fixture, never read back from what the memo saw requested.
// The second half pins the lazy rule: a file no target reaches is never parsed.
func assertBuilds(t *testing.T, builds map[string]int, want []string) {
	t.Helper()
	wantSet := make(map[string]bool, len(want))
	for _, path := range want {
		wantSet[path] = true
		if builds[path] != 1 {
			t.Errorf("builds[%q] = %d; want 1", path, builds[path])
		}
	}
	var extra []string
	for path := range builds {
		if !wantSet[path] {
			extra = append(extra, path)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Errorf("built files outside the expected set: %s", strings.Join(extra, ", "))
	}
}

// memoFixture is the directory tree shared by the builds-once and equivalence tests.
func memoFixture() map[string]string {
	return map[string]string{
		"pkg/a.go":            "package pkg\n\nfunc A() {}\n",
		"pkg/b.go":            "package pkg\n\nfunc B() {}\n",
		"pkg/b_test.go":       "package pkg_test\n\nfunc TestB() {}\n",
		"pkg/README.md":       "# pkg\n\nNotes.\n",
		"pkg/sub/s.go":        "// Package sub is a subdirectory.\npackage sub\n",
		"pkg/sub/deeper/d.go": "package deeper\n",
		"other/o.go":          "package other\n\nfunc O() {}\n",
		"untouched/u.go":      "package untouched\n",
		"mem/m.go":            "package mem\n\nfunc M() {}\n",
		"mem/notes.txt":       "member-only directory\n",
	}
}

// memoTargets mixes self file, directory self and member targets over memoFixture.
func memoTargets() []string {
	return []string{
		"pkg/a.go#", "pkg/b.go#", "pkg/README.md#", "pkg#",
		"pkg#A", "pkg#B", "pkg_test#TestB", "other/o.go#", "mem#M",
	}
}

// TestResolve_BuildsEachFileOnce resolves self and member targets over one fixture in a single call
// and asserts every reached file is built exactly once.
// The expected set is each self file target's directory, each member glyph's unitDirs directories,
// and the directory self target's own directory plus its direct subdirectories
// (pkg/sub is reached for its identity and no deeper).
// mem exists because no self target reaches it: its two files appear in m.files.builds only if the
// member path builds into the call's shared memo.
// pkg/README.md and pkg/b_test.go are first built by self targets and then requested with symbols by
// the member path, so a count of 1 for them pins that the member path reuses records the self path built.
func TestResolve_BuildsEachFileOnce(t *testing.T) {
	r := openScratchRepo(t, "memo-builds-once", memoFixture())
	m, err := newUnitMemo(r)
	if err != nil {
		t.Fatalf("newUnitMemo: %v", err)
	}

	results, err := r.resolve(memoTargets(), m)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	for _, res := range results {
		if res.Status != StatusFound {
			t.Errorf("resolve(%q).Status = %q; want %q", res.Target, res.Status, StatusFound)
		}
	}

	assertBuilds(t, m.files.builds, []string{
		"pkg/a.go", "pkg/b.go", "pkg/b_test.go", "pkg/README.md", "pkg/sub/s.go",
		"other/o.go", "mem/m.go", "mem/notes.txt",
	})
}

// TestResolve_MemoAnswersMatchFreshCalls checks that answers from a shared memo equal answers from
// fresh single-purpose calls, and that target order does not change any answer.
// It guards memo keying and sharing, not extraction: both sides run the same extraction code.
func TestResolve_MemoAnswersMatchFreshCalls(t *testing.T) {
	r := openScratchRepo(t, "memo-equivalence", memoFixture())
	targets := memoTargets()

	forward, err := r.Resolve(targets)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	for _, res := range forward {
		if strings.HasSuffix(res.Target, "#") {
			unit := strings.TrimSuffix(res.Target, "#")
			want, err := r.TOC(unit, TOCOptions{Symbols: boolPtr(false)})
			if err != nil {
				t.Fatalf("TOC(%q): %v", unit, err)
			}
			if res.Listing == nil || !reflect.DeepEqual(*res.Listing, want) {
				t.Errorf("Resolve(%q).Listing = %+v; want %+v", res.Target, res.Listing, want)
			}
			continue
		}
		g, err := glyph.Parse(glyph.Go, res.Target)
		if err != nil {
			t.Fatalf("glyph.Parse(%q): %v", res.Target, err)
		}
		want, err := r.SpansOf(g)
		if err != nil {
			t.Fatalf("SpansOf(%q): %v", res.Target, err)
		}
		if !reflect.DeepEqual(res.Symbols, want) {
			t.Errorf("Resolve(%q).Symbols = %+v; want %+v", res.Target, res.Symbols, want)
		}
	}

	reversed := make([]string, len(targets))
	for i, target := range targets {
		reversed[len(targets)-1-i] = target
	}
	backward, err := r.Resolve(reversed)
	if err != nil {
		t.Fatalf("Resolve(reversed): %v", err)
	}
	for i, res := range backward {
		want := forward[len(targets)-1-i]
		if !reflect.DeepEqual(res, want) {
			t.Errorf("reversed Resolve(%q) = %+v; want %+v", res.Target, res, want)
		}
	}
}

// TestResolve_ClauselessFileKeepsPerConsumerRule pins that a file with no package clause stays in the
// walk consumer's unit listing but is excluded from the member consumer's symbols,
// even though both read the one shared record.
func TestResolve_ClauselessFileKeepsPerConsumerRule(t *testing.T) {
	r := openScratchRepo(t, "memo-clauseless", map[string]string{
		"cl/a.go": "package cl\n\nfunc A() {}\n",
		// tree-sitter recovers func Lost but records no clause, so the entry is lossy.
		"cl/broken.go": "packag cl\n\nfunc Lost() {}\n",
	})

	walk, err := r.TOC("cl", TOCOptions{Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf("TOC(cl): %v", err)
	}
	broken := entryByName(t, walk.Files, "broken.go")
	if !broken.Lossy {
		t.Errorf("broken.go Lossy = false; want true")
	}
	foundLost := false
	if broken.Symbols != nil {
		for _, sym := range *broken.Symbols {
			if sym.ID == "cl#Lost" {
				foundLost = true
			}
		}
	}
	if !foundLost {
		t.Fatalf("broken.go Symbols = %+v; want one with ID %q", broken.Symbols, "cl#Lost")
	}

	m, err := newUnitMemo(r)
	if err != nil {
		t.Fatalf("newUnitMemo: %v", err)
	}
	results, err := r.resolve([]string{"cl/broken.go#", "cl#Lost", "cl#A"}, m)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if results[1].Status != StatusNotFound || results[1].Unit != StatusFound {
		t.Errorf("resolve(cl#Lost) status/unit = %q/%q; want %q/%q",
			results[1].Status, results[1].Unit, StatusNotFound, StatusFound)
	}
	if results[2].Status != StatusFound {
		t.Errorf("resolve(cl#A).Status = %q; want %q", results[2].Status, StatusFound)
	}
	want, err := r.TOC("cl/broken.go", TOCOptions{Symbols: boolPtr(false)})
	if err != nil {
		t.Fatalf("TOC(cl/broken.go): %v", err)
	}
	if results[0].Listing == nil || !reflect.DeepEqual(*results[0].Listing, want) {
		t.Errorf("resolve(cl/broken.go#).Listing = %+v; want %+v", results[0].Listing, want)
	}

	assertBuilds(t, m.files.builds, []string{"cl/a.go", "cl/broken.go"})
}

// TestResolve_GitignoredExplicitTargetVote pins that an explicitly named, gitignored file joins only
// its own target's directory vote.
// secret.go declares package aaa, so its own target's vote ties aaa against g and picks aaa,
// while every other target's vote excludes it and picks g.
// It catches memoising the extended vote under the plain directory key,
// which would leak aaa into other targets or g into the named file's own answer.
func TestResolve_GitignoredExplicitTargetVote(t *testing.T) {
	r := openScratchRepo(t, "memo-gitignored", map[string]string{
		"g/.gitignore": "secret.go\n",
		"g/a.go":       "package g\n\nfunc A() {}\n",
		"g/secret.go":  "package aaa\n\nfunc S() {}\n",
	})
	orders := map[string][]string{
		"forward":  {"g/a.go#", "g/secret.go#", "g#A"},
		"reversed": {"g#A", "g/secret.go#", "g/a.go#"},
	}
	wantA, err := r.TOC("g/a.go", TOCOptions{Symbols: boolPtr(false)})
	if err != nil {
		t.Fatalf("TOC(g/a.go): %v", err)
	}
	wantSecret, err := r.TOC("g/secret.go", TOCOptions{Symbols: boolPtr(false)})
	if err != nil {
		t.Fatalf("TOC(g/secret.go): %v", err)
	}

	for name, targets := range orders {
		t.Run(name, func(t *testing.T) {
			m, err := newUnitMemo(r)
			if err != nil {
				t.Fatalf("newUnitMemo: %v", err)
			}
			results, err := r.resolve(targets, m)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			byTarget := make(map[string]ResolveResult, len(results))
			for _, res := range results {
				byTarget[res.Target] = res
			}

			a := byTarget["g/a.go#"]
			if a.Listing == nil || !reflect.DeepEqual(*a.Listing, wantA) || a.Listing.Package != "g" {
				t.Errorf("resolve(g/a.go#).Listing = %+v; want %+v with Package g", a.Listing, wantA)
			}
			secret := byTarget["g/secret.go#"]
			if secret.Listing == nil || !reflect.DeepEqual(*secret.Listing, wantSecret) || secret.Listing.Package != "aaa" {
				t.Errorf("resolve(g/secret.go#).Listing = %+v; want %+v with Package aaa", secret.Listing, wantSecret)
			}
			if got := byTarget["g#A"].Status; got != StatusFound {
				t.Errorf("resolve(g#A).Status = %q; want %q", got, StatusFound)
			}

			assertBuilds(t, m.files.builds, []string{"g/.gitignore", "g/a.go", "g/secret.go"})
		})
	}
}
