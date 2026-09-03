package ladder

import (
	"errors"
	"strings"
	"testing"
)

const twoBlocksText = "before\n```json\n{\"a\": 1}\n```\nbetween\n```json\n{\"b\": 2}\n```\nafter"

func TestExtractFencedJSON_Selector(t *testing.T) {
	tests := []struct {
		name      string
		which     string
		wantInner string
	}{
		{"first", "first", `{"a": 1}`},
		{"last", "last", `{"b": 2}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, inner, err := ExtractFencedJSON(twoBlocksText, tt.which)
			if err != nil {
				t.Fatalf("ExtractFencedJSON(_, %q) returned error: %v", tt.which, err)
			}
			if inner != tt.wantInner {
				t.Errorf("ExtractFencedJSON(_, %q) inner = %q; want %q", tt.which, inner, tt.wantInner)
			}
			if !strings.HasPrefix(block, "```json") || !strings.HasSuffix(block, "```") {
				t.Errorf("ExtractFencedJSON(_, %q) block = %q; want fences on both ends", tt.which, block)
			}
			if strings.Contains(inner, "```") {
				t.Errorf("ExtractFencedJSON(_, %q) inner = %q; want no fence markers", tt.which, inner)
			}
		})
	}
}

func TestExtractFencedJSON_NoFence(t *testing.T) {
	_, _, err := ExtractFencedJSON("no fenced block here at all", "first")
	if !errors.Is(err, ErrNoFencedJSONBlock) {
		t.Errorf("ExtractFencedJSON(no fence) error = %v; want ErrNoFencedJSONBlock", err)
	}
}

func TestExtractFencedJSON_UnrecognisedSelector(t *testing.T) {
	_, _, err := ExtractFencedJSON(twoBlocksText, "last-ish")
	if err == nil {
		t.Fatal("ExtractFencedJSON with an unrecognised selector = nil error; want an error")
	}
	if errors.Is(err, ErrNoFencedJSONBlock) {
		t.Errorf("ExtractFencedJSON with an unrecognised selector returned ErrNoFencedJSONBlock; want a selector-specific error, not a silent fallthrough to \"last\"")
	}
}

func TestExtractFencedJSON_MultilineBodyCapturedWhole(t *testing.T) {
	text := "```json\n{\n  \"a\": 1,\n  \"b\": {\n    \"c\": [1, 2, 3]\n  }\n}\n```"
	wantInner := "{\n  \"a\": 1,\n  \"b\": {\n    \"c\": [1, 2, 3]\n  }\n}"

	block, inner, err := ExtractFencedJSON(text, "first")
	if err != nil {
		t.Fatalf("ExtractFencedJSON returned error: %v", err)
	}
	if inner != wantInner {
		t.Errorf("ExtractFencedJSON multiline inner = %q; want %q", inner, wantInner)
	}
	if block != text {
		t.Errorf("ExtractFencedJSON multiline block = %q; want %q", block, text)
	}
}
