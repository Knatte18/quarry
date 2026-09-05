// flags_test.go pins parseArgs's behaviour and its usage errors, table-driven and with no
// filesystem access at all: parseArgs is pure over its argument slice.

package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

func TestParseArgs_Depth(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    int
		wantErr bool
	}{
		{"zero", []string{"toc", "--depth", "0", "t"}, 0, false},
		{"three", []string{"toc", "--depth", "3", "t"}, 3, false},
		{"all", []string{"toc", "--depth", "all", "t"}, quarry.DepthAll, false},
		{"equals-all", []string{"toc", "--depth=all", "t"}, quarry.DepthAll, false},
		{"negative", []string{"toc", "--depth", "-1", "t"}, 0, true},
		{"non-integer", []string{"toc", "--depth", "x", "t"}, 0, true},
		{"empty-value", []string{"toc", "--depth=", "t"}, 0, true},
		{"default", []string{"toc", "t"}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseArgs(%v) = _, nil; want error", tt.args)
				}
				if _, ok := err.(usageError); !ok {
					t.Errorf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
			if got.depth != tt.want {
				t.Errorf("parseArgs(%v).depth = %d; want %d", tt.args, got.depth, tt.want)
			}
		})
	}
}

// TestParseArgs_View pins --view's closed full|glyphs vocabulary, both flag forms, and the two
// usage errors: a missing value and a value outside the closed set.
func TestParseArgs_View(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{"full", []string{"toc", "--view", "full", "t"}, "full", ""},
		{"glyphs", []string{"toc", "--view", "glyphs", "t"}, "glyphs", ""},
		{"equals-glyphs", []string{"toc", "--view=glyphs", "t"}, "glyphs", ""},
		{"viewless", []string{"toc", "t"}, "", ""},
		{"no-value", []string{"toc", "--view"}, "", "--view requires a value"},
		{"bogus", []string{"toc", "--view", "bogus", "t"}, "", `--view must be "full" or "glyphs", got "bogus"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseArgs(%v) = _, nil; want error %q", tt.args, tt.wantErr)
				}
				ue, ok := err.(usageError)
				if !ok {
					t.Fatalf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
				}
				if string(ue) != tt.wantErr {
					t.Errorf("parseArgs(%v) error = %q; want %q", tt.args, string(ue), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
			if got.view != tt.want {
				t.Errorf("parseArgs(%v).view = %q; want %q", tt.args, got.view, tt.want)
			}
		})
	}
}

func TestParseArgs_Symbols(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantNil   bool
		wantValue bool
		wantErr   bool
	}{
		{"symbols", []string{"toc", "--symbols", "t"}, false, true, false},
		{"no-symbols", []string{"toc", "--no-symbols", "t"}, false, false, false},
		{"neither", []string{"toc", "t"}, true, false, false},
		{"both-symbols-first", []string{"toc", "--symbols", "--no-symbols", "t"}, false, false, true},
		{"both-no-symbols-first", []string{"toc", "--no-symbols", "--symbols", "t"}, false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseArgs(%v) = _, nil; want error", tt.args)
				}
				if _, ok := err.(usageError); !ok {
					t.Errorf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
			if tt.wantNil {
				if got.symbols != nil {
					t.Errorf("parseArgs(%v).symbols = %v; want nil", tt.args, *got.symbols)
				}
				return
			}
			if got.symbols == nil {
				t.Fatalf("parseArgs(%v).symbols = nil; want non-nil", tt.args)
			}
			if *got.symbols != tt.wantValue {
				t.Errorf("parseArgs(%v).symbols = %v; want %v", tt.args, *got.symbols, tt.wantValue)
			}
		})
	}
}

// TestParseArgs_ViewGlyphsSymbols pins that --view glyphs defaults symbols to true and rejects
// --no-symbols, in either flag order, while --view full and a viewless toc are both untouched by
// the default.
func TestParseArgs_ViewGlyphsSymbols(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantNil   bool
		wantValue bool
		wantErr   string
	}{
		{"view-glyphs-defaults-true", []string{"toc", "--view", "glyphs", "t"}, false, true, ""},
		{"view-then-symbols", []string{"toc", "--view", "glyphs", "--symbols", "t"}, false, true, ""},
		{"symbols-then-view", []string{"toc", "--symbols", "--view", "glyphs", "t"}, false, true, ""},
		{"view-then-no-symbols", []string{"toc", "--view", "glyphs", "--no-symbols", "t"}, false, false, "--no-symbols is not valid with --view glyphs"},
		{"no-symbols-then-view", []string{"toc", "--no-symbols", "--view", "glyphs", "t"}, false, false, "--no-symbols is not valid with --view glyphs"},
		{"view-full-no-symbols-untouched", []string{"toc", "--view", "full", "--no-symbols", "t"}, false, false, ""},
		{"viewless-no-symbols-untouched", []string{"toc", "--no-symbols", "t"}, false, false, ""},
		{"view-full-nil", []string{"toc", "--view", "full", "t"}, true, false, ""},
		{"viewless-nil", []string{"toc", "t"}, true, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseArgs(%v) = _, nil; want error %q", tt.args, tt.wantErr)
				}
				ue, ok := err.(usageError)
				if !ok {
					t.Fatalf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
				}
				if string(ue) != tt.wantErr {
					t.Errorf("parseArgs(%v) error = %q; want %q", tt.args, string(ue), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
			if tt.wantNil {
				if got.symbols != nil {
					t.Errorf("parseArgs(%v).symbols = %v; want nil", tt.args, *got.symbols)
				}
				return
			}
			if got.symbols == nil {
				t.Fatalf("parseArgs(%v).symbols = nil; want non-nil", tt.args)
			}
			if *got.symbols != tt.wantValue {
				t.Errorf("parseArgs(%v).symbols = %v; want %v", tt.args, *got.symbols, tt.wantValue)
			}
		})
	}
}

func TestParseArgs_Text(t *testing.T) {
	got, err := parseArgs([]string{"toc", "--text", "t"})
	if err != nil {
		t.Fatalf("parseArgs = _, %v; want nil error", err)
	}
	if !got.text {
		t.Errorf("parseArgs(--text).text = false; want true")
	}

	got, err = parseArgs([]string{"toc", "t"})
	if err != nil {
		t.Fatalf("parseArgs = _, %v; want nil error", err)
	}
	if got.text {
		t.Errorf("parseArgs().text = true; want false (default)")
	}
}

func TestParseArgs_Root(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{"absolute", []string{"toc", "--root", "/tmp/repo", "t"}, "/tmp/repo", false},
		{"relative", []string{"toc", "--root", "../repo", "t"}, "../repo", false},
		{"no-value", []string{"toc", "--root"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseArgs(%v) = _, nil; want error", tt.args)
				}
				if _, ok := err.(usageError); !ok {
					t.Errorf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
			if got.root != tt.want {
				t.Errorf("parseArgs(%v).root = %q; want %q", tt.args, got.root, tt.want)
			}
		})
	}
}

func TestParseArgs_UsageErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{"unknown-flag", []string{"toc", "--depht", "t"}, "unknown flag: --depht"},
		{"missing-target", []string{"toc"}, "toc takes exactly one target, got 0"},
		{"two-targets", []string{"toc", "a", "b"}, "toc takes exactly one target, got 2"},
		{"missing-verb", []string{}, "no verb given; expected: toc, glyphs, resolve, expand, or name"},
		{"unknown-verb", []string{"bogus", "t"}, "unknown verb: bogus"},
		{"first-arg-is-flag", []string{"--depth", "3", "t"}, "no verb given; expected: toc, glyphs, resolve, expand, or name"},
		{"depth-not-valid-for-resolve", []string{"resolve", "--depth", "3", "t"}, "--depth is not valid for resolve"},
		{"depth-not-valid-for-resolve-bad-value", []string{"resolve", "--depth", "x", "t"}, "--depth is not valid for resolve"},
		{"symbols-not-valid-for-resolve", []string{"resolve", "--symbols", "t"}, "--symbols is not valid for resolve"},
		{"no-symbols-not-valid-for-resolve", []string{"resolve", "--no-symbols", "t"}, "--no-symbols is not valid for resolve"},
		{"view-not-valid-for-resolve", []string{"resolve", "--view", "glyphs", "t"}, "--view is not valid for resolve"},
		{"depth-not-valid-for-expand", []string{"expand", "--depth", "3", "t#u"}, "--depth is not valid for expand"},
		{"symbols-not-valid-for-expand", []string{"expand", "--symbols", "t#u"}, "--symbols is not valid for expand"},
		{"no-symbols-not-valid-for-expand", []string{"expand", "--no-symbols", "t#u"}, "--no-symbols is not valid for expand"},
		{"view-not-valid-for-expand", []string{"expand", "--view", "glyphs", "t#u"}, "--view is not valid for expand"},
		{"view-not-valid-for-name", []string{"name", "--unit", "u", "--view", "glyphs", "t"}, "--view is not valid for name"},
		{"resolve-missing-target", []string{"resolve"}, "resolve takes exactly one target, got 0"},
		{"resolve-two-targets", []string{"resolve", "a", "b"}, "resolve takes exactly one target, got 2"},
		{"expand-missing-target", []string{"expand"}, "expand takes exactly one target, got 0"},
		{"expand-two-targets", []string{"expand", "a#b", "c#d"}, "expand takes exactly one target, got 2"},
		{"unit-not-valid-for-toc", []string{"toc", "--unit", "u", "t"}, "--unit is not valid for toc"},
		{"unit-not-valid-for-resolve", []string{"resolve", "--unit", "u", "t"}, "--unit is not valid for resolve"},
		{"unit-not-valid-for-expand", []string{"expand", "--unit", "u", "t#u"}, "--unit is not valid for expand"},
		{"root-not-valid-for-name", []string{"name", "--unit", "u", "--root", "/repo", "t"}, "--root is not valid for name"},
		{"depth-not-valid-for-name", []string{"name", "--unit", "u", "--depth", "3", "t"}, "--depth is not valid for name"},
		{"symbols-not-valid-for-name", []string{"name", "--unit", "u", "--symbols", "t"}, "--symbols is not valid for name"},
		{"no-symbols-not-valid-for-name", []string{"name", "--unit", "u", "--no-symbols", "t"}, "--no-symbols is not valid for name"},
		{"unit-no-value", []string{"name", "--unit"}, "--unit requires a value"},
		{"name-missing-unit", []string{"name", "t"}, "--unit is required for name"},
		{"name-missing-target", []string{"name", "--unit", "u"}, "name takes exactly one target, got 0"},
		{"name-two-targets", []string{"name", "--unit", "u", "a", "b"}, "name takes exactly one target, got 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			if err == nil {
				t.Fatalf("parseArgs(%v) = _, nil; want error", tt.args)
			}
			ue, ok := err.(usageError)
			if !ok {
				t.Fatalf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
			}
			if string(ue) != tt.wantMsg {
				t.Errorf("parseArgs(%v) error = %q; want %q", tt.args, string(ue), tt.wantMsg)
			}
		})
	}
}

func TestParseArgs_SingleDashHole(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{"post-verb-dash-x", []string{"toc", "-x"}, "unknown flag: -x"},
		{"bare-dash", []string{"toc", "-"}, "unknown flag: -"},
		{"bare-double-dash", []string{"toc", "--"}, "unknown flag: --"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			if err == nil {
				t.Fatalf("parseArgs(%v) = _, nil; want error", tt.args)
			}
			ue, ok := err.(usageError)
			if !ok {
				t.Fatalf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
			}
			if string(ue) != tt.wantMsg {
				t.Errorf("parseArgs(%v) error = %q; want %q", tt.args, string(ue), tt.wantMsg)
			}
			if strings.Contains(string(ue), "target not found") {
				t.Errorf("parseArgs(%v) leaked to target-not-found path: %q", tt.args, string(ue))
			}
		})
	}
}

// TestParseArgs_FiveVerbGate pins that all five verbs are accepted, with no target-shape
// rejection for the verbs that do not require one. Unlike the other four rows, whose req.verb is
// the verb they were given, the "glyphs" row's req.verb is "toc" after the rewrite — stated here
// so a reader comparing the rows does not mistake it for a bug.
func TestParseArgs_FiveVerbGate(t *testing.T) {
	tests := []struct {
		name string
		args []string
		verb string
	}{
		{"toc", []string{"toc", "t"}, "toc"},
		{"glyphs", []string{"glyphs", "t"}, "toc"}, // rewritten: glyphs' own req.verb is "toc".
		{"resolve", []string{"resolve", "t"}, "resolve"},
		{"expand", []string{"expand", "t#u"}, "expand"},
		{"name", []string{"name", "--unit", "u", "t"}, "name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
			if got.verb != tt.verb {
				t.Errorf("parseArgs(%v).verb = %q; want %q", tt.args, got.verb, tt.verb)
			}
		})
	}
}

// TestParseArgs_ExpandAcceptsHashBearingTarget pins that a "#"-bearing target passes the parser
// for expand: the grammar's own rejection of a malformed glyph belongs to a later stage, not
// here.
func TestParseArgs_ExpandAcceptsHashBearingTarget(t *testing.T) {
	got, err := parseArgs([]string{"expand", "pkg/file.go#Foo"})
	if err != nil {
		t.Fatalf("parseArgs(expand, hash-bearing target) = _, %v; want nil error", err)
	}
	if got.target != "pkg/file.go#Foo" {
		t.Errorf("parseArgs(expand, hash-bearing target).target = %q; want %q", got.target, "pkg/file.go#Foo")
	}
}

// TestParseArgs_TextOnEveryVerbRootOnRepositoryVerbs pins that --text is accepted on every verb,
// while --root is accepted on the four repository verbs only, unlike --depth, --symbols and
// --no-symbols, which are toc only.
func TestParseArgs_TextOnEveryVerbRootOnRepositoryVerbs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"toc-text", []string{"toc", "--text", "t"}},
		{"toc-root", []string{"toc", "--root", "/repo", "t"}},
		{"glyphs-text", []string{"glyphs", "--text", "t"}},
		{"glyphs-root", []string{"glyphs", "--root", "/repo", "t"}},
		{"resolve-text", []string{"resolve", "--text", "t"}},
		{"resolve-root", []string{"resolve", "--root", "/repo", "t"}},
		{"expand-text", []string{"expand", "--text", "t#u"}},
		{"expand-root", []string{"expand", "--root", "/repo", "t#u"}},
		{"name-text", []string{"name", "--unit", "u", "--text", "t"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseArgs(tt.args); err != nil {
				t.Errorf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
		})
	}
}

// TestParseArgs_NameUnitAndTarget pins that a name invocation's --unit and target land in the
// request unchanged, in both the space-separated and the --unit=value equals forms.
func TestParseArgs_NameUnitAndTarget(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantUnit   string
		wantTarget string
	}{
		{"space-separated", []string{"name", "--unit", "pkg", "func F() error"}, "pkg", "func F() error"},
		{"equals-form", []string{"name", "--unit=pkg", "func F() error"}, "pkg", "func F() error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
			if got.unit != tt.wantUnit {
				t.Errorf("parseArgs(%v).unit = %q; want %q", tt.args, got.unit, tt.wantUnit)
			}
			if got.target != tt.wantTarget {
				t.Errorf("parseArgs(%v).target = %q; want %q", tt.args, got.target, tt.wantTarget)
			}
		})
	}
}

func TestParseArgs_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"alone", []string{"--help"}},
		{"alone-short", []string{"-h"}},
		{"before-verb", []string{"--help", "toc"}},
		{"after-verb", []string{"toc", "--help"}},
		{"after-target", []string{"toc", "t", "--help"}},
		{"alongside-invalid-flag", []string{"toc", "--bogus", "-h"}},
		{"resolve-after-verb", []string{"resolve", "--help"}},
		{"resolve-alongside-invalid-flag", []string{"resolve", "--depth", "-h"}},
		{"expand-after-verb", []string{"expand", "--help"}},
		{"expand-alongside-invalid-flag", []string{"expand", "--symbols", "-h"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs(%v) = _, %v; want nil error", tt.args, err)
			}
			if !got.help {
				t.Errorf("parseArgs(%v).help = false; want true", tt.args)
			}
		})
	}
}

// TestParseArgs_Glyphs pins the glyphs verb's own pre-scan: which flags it accepts, which it
// rejects and by what message, and the target-count rule, all before any rewrite ever happens.
func TestParseArgs_Glyphs(t *testing.T) {
	t.Run("parses", func(t *testing.T) {
		if _, err := parseArgs([]string{"glyphs", "x"}); err != nil {
			t.Fatalf("parseArgs(glyphs, x) = _, %v; want nil error", err)
		}
	})

	t.Run("text-sets-text", func(t *testing.T) {
		got, err := parseArgs([]string{"glyphs", "--text", "x"})
		if err != nil {
			t.Fatalf("parseArgs(glyphs, --text, x) = _, %v; want nil error", err)
		}
		if !got.text {
			t.Errorf("parseArgs(glyphs, --text, x).text = false; want true")
		}
	})

	t.Run("root-space-separated", func(t *testing.T) {
		got, err := parseArgs([]string{"glyphs", "--root", "/repo", "x"})
		if err != nil {
			t.Fatalf("parseArgs(glyphs, --root, /repo, x) = _, %v; want nil error", err)
		}
		if got.root != "/repo" {
			t.Errorf("root = %q; want %q", got.root, "/repo")
		}
		if got.target != "x" {
			t.Errorf("target = %q; want %q", got.target, "x")
		}
	})

	t.Run("root-equals-form", func(t *testing.T) {
		got, err := parseArgs([]string{"glyphs", "--root=/repo", "x"})
		if err != nil {
			t.Fatalf("parseArgs(glyphs, --root=/repo, x) = _, %v; want nil error", err)
		}
		if got.root != "/repo" {
			t.Errorf("root = %q; want %q", got.root, "/repo")
		}
		if got.target != "x" {
			t.Errorf("target = %q; want %q", got.target, "x")
		}
	})

	rejectedFlags := []struct {
		name string
		args []string
		want string
	}{
		{"view", []string{"glyphs", "--view", "glyphs", "x"}, "--view is not valid for glyphs"},
		{"depth", []string{"glyphs", "--depth", "1", "x"}, "--depth is not valid for glyphs"},
		{"symbols", []string{"glyphs", "--symbols", "x"}, "--symbols is not valid for glyphs"},
		{"no-symbols", []string{"glyphs", "--no-symbols", "x"}, "--no-symbols is not valid for glyphs"},
		{"unit", []string{"glyphs", "--unit", "u", "x"}, "--unit is not valid for glyphs"},
	}
	for _, tt := range rejectedFlags {
		t.Run("rejects-"+tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			if err == nil {
				t.Fatalf("parseArgs(%v) = _, nil; want error", tt.args)
			}
			ue, ok := err.(usageError)
			if !ok {
				t.Fatalf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
			}
			if string(ue) != tt.want {
				t.Errorf("parseArgs(%v) error = %q; want %q", tt.args, string(ue), tt.want)
			}
		})
	}

	t.Run("unknown-flag", func(t *testing.T) {
		_, err := parseArgs([]string{"glyphs", "--nope", "x"})
		if err == nil {
			t.Fatalf("parseArgs(glyphs, --nope, x) = _, nil; want error")
		}
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs(glyphs, --nope, x) error type = %T; want usageError", err)
		}
		if string(ue) != "unknown flag: --nope" {
			t.Errorf("error = %q; want %q", string(ue), "unknown flag: --nope")
		}
	})

	t.Run("zero-targets", func(t *testing.T) {
		_, err := parseArgs([]string{"glyphs"})
		if err == nil {
			t.Fatalf("parseArgs(glyphs) = _, nil; want error")
		}
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs(glyphs) error type = %T; want usageError", err)
		}
		if string(ue) != "glyphs takes exactly one target, got 0" {
			t.Errorf("error = %q; want %q", string(ue), "glyphs takes exactly one target, got 0")
		}
	})

	t.Run("two-targets", func(t *testing.T) {
		_, err := parseArgs([]string{"glyphs", "x", "y"})
		if err == nil {
			t.Fatalf("parseArgs(glyphs, x, y) = _, nil; want error")
		}
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs(glyphs, x, y) error type = %T; want usageError", err)
		}
		if string(ue) != "glyphs takes exactly one target, got 2" {
			t.Errorf("error = %q; want %q", string(ue), "glyphs takes exactly one target, got 2")
		}
	})

	// root-no-value pins that a missing --root value is reported as its own usage error, not
	// misreported as a target-count message: falling through to the count would say "got 0" for an
	// invocation that did supply a target, just not one --root could consume.
	t.Run("root-no-value", func(t *testing.T) {
		_, err := parseArgs([]string{"glyphs", "--root"})
		if err == nil {
			t.Fatalf("parseArgs(glyphs, --root) = _, nil; want error")
		}
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs(glyphs, --root) error type = %T; want usageError", err)
		}
		if string(ue) != "--root requires a value" {
			t.Errorf("error = %q; want %q", string(ue), "--root requires a value")
		}
	})

	t.Run("help-scan-runs-before-verb-gate", func(t *testing.T) {
		got, err := parseArgs([]string{"glyphs", "--help", "x"})
		if err != nil {
			t.Fatalf("parseArgs(glyphs, --help, x) = _, %v; want nil error", err)
		}
		if !got.help {
			t.Errorf("parseArgs(glyphs, --help, x).help = false; want true")
		}
	})
}

// TestParseArgs_GlyphsIsTheFrozenTOCExpansion is the load-bearing case: parseArgs of the glyphs
// verb and parseArgs of its documented toc expansion must produce a deep-equal request. The
// symbols pointer is compared by dereferencing, since two distinct pointers to true are not
// reflect.DeepEqual-equal as pointers.
func TestParseArgs_GlyphsIsTheFrozenTOCExpansion(t *testing.T) {
	glyphsReq, err := parseArgs([]string{"glyphs", "x"})
	if err != nil {
		t.Fatalf("parseArgs(glyphs, x) = _, %v; want nil error", err)
	}
	tocReq, err := parseArgs([]string{"toc", "--view", "glyphs", "--depth", "all", "--symbols", "x"})
	if err != nil {
		t.Fatalf("parseArgs(toc, --view, glyphs, --depth, all, --symbols, x) = _, %v; want nil error", err)
	}

	if glyphsReq.symbols == nil || tocReq.symbols == nil {
		t.Fatalf("symbols = %v, %v; want both non-nil", glyphsReq.symbols, tocReq.symbols)
	}
	if *glyphsReq.symbols != *tocReq.symbols {
		t.Errorf("symbols = %v; want %v", *glyphsReq.symbols, *tocReq.symbols)
	}

	// Compare the rest of the struct with the pointers zeroed, since reflect.DeepEqual would
	// otherwise compare the two distinct pointers themselves rather than what they point to.
	glyphsReq.symbols = nil
	tocReq.symbols = nil
	if !reflect.DeepEqual(glyphsReq, tocReq) {
		t.Errorf("parseArgs(glyphs, x) = %+v; want deep-equal to parseArgs(toc expansion) = %+v", glyphsReq, tocReq)
	}
}
