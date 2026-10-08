// enclose_test.go pins the enclose vocabulary, result shape and innermost-member selection.

package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEnclose_ReasonCompleteness pins EncloseReasons to exactly the eight reason constants, each once.
func TestEnclose_ReasonCompleteness(t *testing.T) {
	t.Parallel()
	want := map[string]bool{
		EncloseReasonBadLocation:         true,
		EncloseReasonBadRange:            true,
		EncloseReasonOutsideRoot:         true,
		EncloseReasonUnaddressable:       true,
		EncloseReasonUnsupportedLanguage: true,
		EncloseReasonMissingFile:         true,
		EncloseReasonUnreadable:          true,
		EncloseReasonPastEOF:             true,
	}
	if len(EncloseReasons) != len(want) {
		t.Fatalf("len(EncloseReasons) = %d; want %d", len(EncloseReasons), len(want))
	}
	seen := make(map[string]bool, len(EncloseReasons))
	for _, r := range EncloseReasons {
		if seen[r] {
			t.Errorf("EncloseReasons contains %q more than once", r)
		}
		seen[r] = true
		if !want[r] {
			t.Errorf("EncloseReasons contains unexpected value %q", r)
		}
	}
	for r := range want {
		if !seen[r] {
			t.Errorf("EncloseReasons is missing %q", r)
		}
	}
}

// TestEncloseResult_Rejected asserts Rejected is true exactly when Status is empty.
func TestEncloseResult_Rejected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"Empty", "", true},
		{"Found", StatusFound, false},
		{"NotFound", StatusNotFound, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := (EncloseResult{Status: tt.status}).Rejected(); got != tt.want {
				t.Errorf("EncloseResult{Status: %q}.Rejected() = %v; want %v", tt.status, got, tt.want)
			}
		})
	}
}

// TestEncloseResult_JSONKeyOrder asserts a fully populated result emits its keys in declared order.
func TestEncloseResult_JSONKeyOrder(t *testing.T) {
	t.Parallel()
	res := EncloseResult{
		Target: "a/b.go:3", File: "a/b.go", Start: 3, End: 3, Status: StatusFound,
		Symbols: []Symbol{{ID: "a#F"}}, Unit: "a/b.go#", Lossy: true, Error: "e", Reason: "r",
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	last := -1
	for _, key := range []string{"target", "file", "start", "end", "status", "symbols", "unit", "lossy", "error", "reason"} {
		idx := strings.Index(string(raw), `"`+key+`":`)
		if idx < 0 {
			t.Fatalf("key %q missing from %s", key, raw)
		}
		if idx < last {
			t.Errorf("key %q out of order in %s", key, raw)
		}
		last = idx
	}
}
