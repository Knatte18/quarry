// flags_test.go pins parseArgs's behaviour and its usage errors, table-driven and with no
// filesystem access at all: parseArgs is pure over its argument slice.

package cli

import (
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
		{"missing-verb", []string{}, "no verb given; expected: toc, resolve, expand, delta, or name"},
		{"unknown-verb", []string{"bogus", "t"}, "unknown verb: bogus"},
		{"first-arg-is-flag", []string{"--depth", "3", "t"}, "no verb given; expected: toc, resolve, expand, delta, or name"},
		{"depth-not-valid-for-resolve", []string{"resolve", "--depth", "3", "t"}, "--depth is not valid for resolve"},
		{"depth-not-valid-for-resolve-bad-value", []string{"resolve", "--depth", "x", "t"}, "--depth is not valid for resolve"},
		{"symbols-not-valid-for-resolve", []string{"resolve", "--symbols", "t"}, "--symbols is not valid for resolve"},
		{"no-symbols-not-valid-for-resolve", []string{"resolve", "--no-symbols", "t"}, "--no-symbols is not valid for resolve"},
		{"depth-not-valid-for-expand", []string{"expand", "--depth", "3", "t#u"}, "--depth is not valid for expand"},
		{"symbols-not-valid-for-expand", []string{"expand", "--symbols", "t#u"}, "--symbols is not valid for expand"},
		{"no-symbols-not-valid-for-expand", []string{"expand", "--no-symbols", "t#u"}, "--no-symbols is not valid for expand"},
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

// TestParseArgs_FiveVerbGate pins that all five verbs are accepted and land in req.verb
// unchanged, with no target-shape rejection for the verbs that do not require one. The name is
// unchanged from when the gate accepted three, then four, verbs; the table below is what now
// asserts five.
func TestParseArgs_FiveVerbGate(t *testing.T) {
	tests := []struct {
		name string
		args []string
		verb string
	}{
		{"toc", []string{"toc", "t"}, "toc"},
		{"resolve", []string{"resolve", "t"}, "resolve"},
		{"expand", []string{"expand", "t#u"}, "expand"},
		{"delta", []string{"delta", "--from", "HEAD", "t"}, "delta"},
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
// --no-symbols, which are toc only, --from and --to, which are delta only, and --unit, which name
// alone requires.
func TestParseArgs_TextOnEveryVerbRootOnRepositoryVerbs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"toc-text", []string{"toc", "--text", "t"}},
		{"toc-root", []string{"toc", "--root", "/repo", "t"}},
		{"resolve-text", []string{"resolve", "--text", "t"}},
		{"resolve-root", []string{"resolve", "--root", "/repo", "t"}},
		{"expand-text", []string{"expand", "--text", "t#u"}},
		{"expand-root", []string{"expand", "--root", "/repo", "t#u"}},
		{"delta-text", []string{"delta", "--from", "HEAD", "--text", "t"}},
		{"delta-root", []string{"delta", "--from", "HEAD", "--root", "/repo", "t"}},
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
		{"delta-after-verb-no-target-no-revisions", []string{"delta", "--help"}},
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

// TestParseArgs_Delta covers the delta verb's own argument shape: both revision flags in both
// forms, the absent-to-revision default, an equals-sign-bearing value surviving verbatim, the
// missing-from-revision rejection, each revision flag given without a value, each revision flag
// given an explicitly empty value in the equals form, and the exactly-one-target rule.
func TestParseArgs_Delta(t *testing.T) {
	t.Run("BothRevisionsSpaceSeparated", func(t *testing.T) {
		got, err := parseArgs([]string{"delta", "--from", "abc123", "--to", "def456", "t"})
		if err != nil {
			t.Fatalf("parseArgs = _, %v; want nil error", err)
		}
		if got.from != "abc123" {
			t.Errorf("from = %q; want %q", got.from, "abc123")
		}
		if got.to != "def456" {
			t.Errorf("to = %q; want %q", got.to, "def456")
		}
	})

	t.Run("BothRevisionsEqualsForm", func(t *testing.T) {
		got, err := parseArgs([]string{"delta", "--from=abc123", "--to=def456", "t"})
		if err != nil {
			t.Fatalf("parseArgs = _, %v; want nil error", err)
		}
		if got.from != "abc123" {
			t.Errorf("from = %q; want %q", got.from, "abc123")
		}
		if got.to != "def456" {
			t.Errorf("to = %q; want %q", got.to, "def456")
		}
	})

	t.Run("AbsentToLeavesFieldEmpty", func(t *testing.T) {
		got, err := parseArgs([]string{"delta", "--from", "abc123", "t"})
		if err != nil {
			t.Fatalf("parseArgs = _, %v; want nil error", err)
		}
		if got.to != "" {
			t.Errorf("to = %q; want empty, which is how the working tree is spelled", got.to)
		}
	})

	t.Run("ValueContainingEqualsSignSurvivesVerbatim", func(t *testing.T) {
		got, err := parseArgs([]string{"delta", "--from=a=b", "t"})
		if err != nil {
			t.Fatalf("parseArgs = _, %v; want nil error", err)
		}
		if got.from != "a=b" {
			t.Errorf("from = %q; want %q", got.from, "a=b")
		}
	})

	t.Run("MissingFromRevisionRejected", func(t *testing.T) {
		_, err := parseArgs([]string{"delta", "t"})
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs error type = %T; want usageError", err)
		}
		if want := "delta requires --from"; string(ue) != want {
			t.Errorf("error = %q; want %q", string(ue), want)
		}
	})

	t.Run("FromWithoutValueRejected", func(t *testing.T) {
		_, err := parseArgs([]string{"delta", "--from"})
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs error type = %T; want usageError", err)
		}
		if want := "--from requires a value"; string(ue) != want {
			t.Errorf("error = %q; want %q", string(ue), want)
		}
	})

	t.Run("ToWithoutValueRejected", func(t *testing.T) {
		_, err := parseArgs([]string{"delta", "--from", "HEAD", "--to"})
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs error type = %T; want usageError", err)
		}
		if want := "--to requires a value"; string(ue) != want {
			t.Errorf("error = %q; want %q", string(ue), want)
		}
	})

	t.Run("FromExplicitlyEmptyRejected", func(t *testing.T) {
		_, err := parseArgs([]string{"delta", "--from=", "t"})
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs error type = %T; want usageError", err)
		}
		if want := "--from value must not be empty"; string(ue) != want {
			t.Errorf("error = %q; want %q", string(ue), want)
		}
	})

	t.Run("ToExplicitlyEmptyRejected", func(t *testing.T) {
		_, err := parseArgs([]string{"delta", "--from", "HEAD", "--to=", "t"})
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs error type = %T; want usageError", err)
		}
		if want := "--to value must not be empty"; string(ue) != want {
			t.Errorf("error = %q; want %q", string(ue), want)
		}
	})

	t.Run("ZeroTargetsRejectedWithCount", func(t *testing.T) {
		_, err := parseArgs([]string{"delta", "--from", "HEAD"})
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs error type = %T; want usageError", err)
		}
		if want := "delta takes exactly one target, got 0"; string(ue) != want {
			t.Errorf("error = %q; want %q", string(ue), want)
		}
	})

	t.Run("TwoTargetsRejectedWithCount", func(t *testing.T) {
		_, err := parseArgs([]string{"delta", "--from", "HEAD", "t", "u"})
		ue, ok := err.(usageError)
		if !ok {
			t.Fatalf("parseArgs error type = %T; want usageError", err)
		}
		if want := "delta takes exactly one target, got 2"; string(ue) != want {
			t.Errorf("error = %q; want %q", string(ue), want)
		}
	})
}

// TestParseArgs_DeltaFlagValidity covers the validity matrix in both directions: --from and --to
// rejected for each of the other three verbs, and --depth, --symbols and --no-symbols each
// rejected for delta, all with the same "%s is not valid for %s" message shape.
func TestParseArgs_DeltaFlagValidity(t *testing.T) {
	revisionFlagCases := []struct {
		name string
		args []string
		want string
	}{
		{"from-not-valid-for-toc", []string{"toc", "--from", "HEAD", "t"}, "--from is not valid for toc"},
		{"to-not-valid-for-toc", []string{"toc", "--to", "HEAD", "t"}, "--to is not valid for toc"},
		{"from-not-valid-for-resolve", []string{"resolve", "--from", "HEAD", "t"}, "--from is not valid for resolve"},
		{"to-not-valid-for-resolve", []string{"resolve", "--to", "HEAD", "t"}, "--to is not valid for resolve"},
		{"from-not-valid-for-expand", []string{"expand", "--from", "HEAD", "t#u"}, "--from is not valid for expand"},
		{"to-not-valid-for-expand", []string{"expand", "--to", "HEAD", "t#u"}, "--to is not valid for expand"},
	}
	for _, tt := range revisionFlagCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			ue, ok := err.(usageError)
			if !ok {
				t.Fatalf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
			}
			if string(ue) != tt.want {
				t.Errorf("parseArgs(%v) error = %q; want %q", tt.args, string(ue), tt.want)
			}
		})
	}

	tocOnlyFlagCases := []struct {
		name string
		args []string
		want string
	}{
		{"depth-not-valid-for-delta", []string{"delta", "--from", "HEAD", "--depth", "3", "t"}, "--depth is not valid for delta"},
		{"symbols-not-valid-for-delta", []string{"delta", "--from", "HEAD", "--symbols", "t"}, "--symbols is not valid for delta"},
		{"no-symbols-not-valid-for-delta", []string{"delta", "--from", "HEAD", "--no-symbols", "t"}, "--no-symbols is not valid for delta"},
	}
	for _, tt := range tocOnlyFlagCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			ue, ok := err.(usageError)
			if !ok {
				t.Fatalf("parseArgs(%v) error type = %T; want usageError", tt.args, err)
			}
			if string(ue) != tt.want {
				t.Errorf("parseArgs(%v) error = %q; want %q", tt.args, string(ue), tt.want)
			}
		})
	}
}
