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
		{"missing-verb", []string{}, "no verb given; expected: toc"},
		{"unknown-verb", []string{"resolve", "t"}, "unknown verb: resolve"},
		{"first-arg-is-flag", []string{"--depth", "3", "t"}, "no verb given; expected: toc"},
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
