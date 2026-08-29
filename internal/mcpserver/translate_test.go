package mcpserver

import "testing"

func TestStripFileURI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"FileScheme", "file:///abs/path", "/abs/path"},
		{"PlainPathPassthrough", "/abs/path", "/abs/path"},
		{"RelativePassthrough", "relative/path", "relative/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFileURI(tt.in)
			if got != tt.want {
				t.Errorf("stripFileURI(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveEntryFile(t *testing.T) {
	tests := []struct {
		name      string
		targetDir string
		raw       string
		want      string
	}{
		{"AbsoluteInput", "/target", "/abs/file.go", "/abs/file.go"},
		{"RelativeInput", "/target", "rel/file.go", "/target/rel/file.go"},
		{"FileURIInput", "/target", "file:///abs/file.go", "/abs/file.go"},
		{"FileURIRelativeAfterStrip", "/target", "file://rel/file.go", "/target/rel/file.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveEntryFile(tt.targetDir, tt.raw)
			if got != tt.want {
				t.Errorf("resolveEntryFile(%q, %q) = %q; want %q", tt.targetDir, tt.raw, got, tt.want)
			}
		})
	}
}

func TestToOneBased(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"Zero", 0, 1},
		{"Positive", 41, 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toOneBased(tt.in)
			if got != tt.want {
				t.Errorf("toOneBased(%d) = %d; want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestToZeroBased(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"One", 1, 0},
		{"Positive", 42, 41},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toZeroBased(tt.in)
			if got != tt.want {
				t.Errorf("toZeroBased(%d) = %d; want %d", tt.in, got, tt.want)
			}
		})
	}
}

// TestToOneBased_ToZeroBased_RoundTrip pins that the two conversions are exact inverses of one
// another for an arbitrary value, since a value returned by toZeroBased must round-trip straight
// back into a following call through toOneBased.
func TestToOneBased_ToZeroBased_RoundTrip(t *testing.T) {
	for _, v := range []int{0, 1, 7, 41, 1000} {
		if got := toZeroBased(toOneBased(v)); got != v {
			t.Errorf("toZeroBased(toOneBased(%d)) = %d; want %d", v, got, v)
		}
		if got := toOneBased(toZeroBased(v)); got != v {
			t.Errorf("toOneBased(toZeroBased(%d)) = %d; want %d", v, got, v)
		}
	}
}

// TestToOneBased_NonASCIINaiveBehaviour pins that toOneBased/toZeroBased perform a naive "+1"/"-1"
// with no UTF-16 or byte-width accounting at all, deliberately reproducing
// internal/quarryengine/query/refs.go's naive outbound conversion rather than fixing it locally: a
// character offset that already sits past a multi-byte rune (e.g. character 5 on a line starting
// with "héllo, wörld") converts by simple arithmetic, not by re-deriving a UTF-16 or byte offset
// from the line's content — this layer has no file content to consult in the first place.
func TestToOneBased_NonASCIINaiveBehaviour(t *testing.T) {
	// "héllo" — a character index counted past the multi-byte "é" is still just "+1" here,
	// because neither function reads a file or knows a line even exists.
	const nonASCIICharacter = 5
	if got := toOneBased(nonASCIICharacter); got != 6 {
		t.Errorf("toOneBased(%d) = %d; want 6 (naive +1, no UTF-16 accounting)", nonASCIICharacter, got)
	}
	if got := toZeroBased(nonASCIICharacter); got != 4 {
		t.Errorf("toZeroBased(%d) = %d; want 4 (naive -1, no UTF-16 accounting)", nonASCIICharacter, got)
	}
}
