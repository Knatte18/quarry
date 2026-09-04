// ignore_test.go table-drives ignoreSet's newIgnoreSet/extend/trim/match over every pattern form
// ignore.go's doc comment lists, using fixture trees built at run time by writeScratchTree — see
// ignore.go's package doc and this task's fixture-split decision for why these trees are never
// committed under testdata/.

package engine

import "testing"

func TestIgnoreSet_Match(t *testing.T) {
	tests := []struct {
		name      string
		gitignore string
		checks    []struct {
			path  string
			isDir bool
			want  bool
		}
	}{
		{
			name:      "CommentLineIsNotAPattern",
			gitignore: "# ignore me\nfoo.txt\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"foo.txt", false, true},
				{"# ignore me", false, false},
			},
		},
		{
			name:      "BlankLineIsSkipped",
			gitignore: "\nfoo.txt\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"foo.txt", false, true},
			},
		},
		{
			name:      "BareNameMatchesAnyDepth",
			gitignore: "foo.txt\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"foo.txt", false, true},
				{"sub/foo.txt", false, true},
				{"bar.txt", false, false},
			},
		},
		{
			name:      "AnchoredSlashNameMatchesOnlyAtRoot",
			gitignore: "/foo.txt\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"foo.txt", false, true},
				{"sub/foo.txt", false, false},
			},
		},
		{
			name:      "DirectoryOnlyMatchesDirsOnly",
			gitignore: "build/\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"build", true, true},
				{"build", false, false},
				{"sub/build", true, true},
			},
		},
		{
			name:      "InteriorSlashPatternAnchoredAndDirOnly",
			gitignore: "src/build/\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"src/build", true, true},
				{"src/build", false, false},
				{"other/src/build", true, false},
			},
		},
		{
			name:      "StarAndQuestionWithinSegment",
			gitignore: "*.tmp\nfile?.log\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"a.tmp", false, true},
				{"sub/a.tmp", false, true},
				{"file1.log", false, true},
				{"file12.log", false, false},
			},
		},
		{
			name:      "DoubleStarAcrossSegments",
			gitignore: "**/logs\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"logs", true, true},
				{"a/b/logs", true, true},
				{"a/logsx", true, false},
			},
		},
		{
			name:      "NegationReincludesAPreviouslyExcludedPath",
			gitignore: "*.log\n!keep.log\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"a.log", false, true},
				{"keep.log", false, false},
			},
		},
		{
			name:      "QuarryOwnBinaryPlusReincludedPackagePair",
			gitignore: "/quarry\n!/quarry/\n",
			checks: []struct {
				path  string
				isDir bool
				want  bool
			}{
				{"quarry", false, true},
				{"quarry", true, false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeScratchTree(t, "ignoreset-"+tt.name, map[string]string{
				".gitignore": tt.gitignore,
			})

			s := newIgnoreSet(root)
			if _, err := s.extend(""); err != nil {
				t.Fatalf("extend(\"\") = %v", err)
			}

			for _, c := range tt.checks {
				if got := s.match(c.path, c.isDir); got != c.want {
					t.Errorf("match(%q, isDir=%v) = %v; want %v", c.path, c.isDir, got, c.want)
				}
			}
		})
	}
}

// TestIgnoreSet_GitAlwaysExcluded asserts that .git is excluded as a directory even when no
// .gitignore file exists at all — the rule is unconditional and precedes any pattern.
func TestIgnoreSet_GitAlwaysExcluded(t *testing.T) {
	root := writeScratchTree(t, "ignoreset-git-always-excluded", map[string]string{
		"README.md": "hello\n",
	})

	s := newIgnoreSet(root)
	if _, err := s.extend(""); err != nil {
		t.Fatalf("extend(\"\") = %v", err)
	}

	if !s.match(".git", true) {
		t.Errorf("match(\".git\", isDir=true) = false; want true with zero patterns present")
	}
	if s.match(".git", false) {
		t.Errorf("match(\".git\", isDir=false) = true; want false — .git is only unconditionally excluded as a directory")
	}
}

// TestIgnoreSet_ExtendMissingFileReturnsZero asserts that a directory with no .gitignore file is
// not an error and appends nothing.
func TestIgnoreSet_ExtendMissingFileReturnsZero(t *testing.T) {
	root := writeScratchTree(t, "ignoreset-extend-missing", map[string]string{
		"README.md": "hello\n",
	})

	s := newIgnoreSet(root)
	n, err := s.extend("")
	if err != nil {
		t.Fatalf("extend(\"\") = %v", err)
	}
	if n != 0 {
		t.Errorf("extend(\"\") = %d; want 0 with no .gitignore present", n)
	}
}

// TestIgnoreSet_TrimRestoresPreviousSet asserts that extending into a subdirectory and then
// trimming that extension's patterns leaves match reporting exactly what it reported before the
// walk descended — the frame the walk relies on when it leaves a directory.
func TestIgnoreSet_TrimRestoresPreviousSet(t *testing.T) {
	root := writeScratchTree(t, "ignoreset-trim-restores", map[string]string{
		".gitignore":     "*.log\n",
		"sub/.gitignore": "*.tmp\n",
		"sub/keep.log":   "x",
		"sub/keep.tmp":   "x",
	})

	s := newIgnoreSet(root)
	if _, err := s.extend(""); err != nil {
		t.Fatalf("extend(\"\") = %v", err)
	}

	before := s.match("sub/keep.tmp", false)
	if before {
		t.Fatalf("precondition: match(\"sub/keep.tmp\") = true before descending into sub/; want false")
	}

	n, err := s.extend("sub")
	if err != nil {
		t.Fatalf("extend(\"sub\") = %v", err)
	}
	if n != 1 {
		t.Fatalf("extend(\"sub\") appended %d patterns; want 1", n)
	}
	if !s.match("sub/keep.tmp", false) {
		t.Errorf("match(\"sub/keep.tmp\") = false after extend(\"sub\"); want true")
	}

	s.trim(n)

	if got := s.match("sub/keep.tmp", false); got != before {
		t.Errorf("match(\"sub/keep.tmp\") after trim = %v; want %v (the pre-extend value)", got, before)
	}
	if !s.match("keep.log", false) {
		t.Errorf("match(\"keep.log\") after trim = false; want true — the root pattern must survive the trim")
	}
}
