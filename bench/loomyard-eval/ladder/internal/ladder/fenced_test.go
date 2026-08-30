package ladder

import (
	"errors"
	"testing"
)

func TestExtractFencedJSON_FirstSelector(t *testing.T) {
	text := "prose before\n```json\n{\"a\": 1}\n```\nmiddle prose\n```json\n{\"b\": 2}\n```\ntrailing prose"

	block, inner, err := ExtractFencedJSON(text, "first")
	if err != nil {
		t.Fatalf("ExtractFencedJSON(text, \"first\") = _, _, %v; want nil error", err)
	}
	wantInner := `{"a": 1}`
	wantBlock := "```json\n{\"a\": 1}\n```"
	if inner != wantInner {
		t.Errorf("inner = %q; want %q", inner, wantInner)
	}
	if block != wantBlock {
		t.Errorf("block = %q; want %q", block, wantBlock)
	}
}

func TestExtractFencedJSON_LastSelector(t *testing.T) {
	text := "prose before\n```json\n{\"a\": 1}\n```\nmiddle prose\n```json\n{\"b\": 2}\n```\ntrailing prose"

	block, inner, err := ExtractFencedJSON(text, "last")
	if err != nil {
		t.Fatalf("ExtractFencedJSON(text, \"last\") = _, _, %v; want nil error", err)
	}
	wantInner := `{"b": 2}`
	wantBlock := "```json\n{\"b\": 2}\n```"
	if inner != wantInner {
		t.Errorf("inner = %q; want %q", inner, wantInner)
	}
	if block != wantBlock {
		t.Errorf("block = %q; want %q", block, wantBlock)
	}
}

func TestExtractFencedJSON_MultipleBlocksSelectsCorrectlyByPosition(t *testing.T) {
	text := "```json\n{\"first\": true}\n```\n```json\n{\"second\": true}\n```\n```json\n{\"third\": true}\n```"

	firstBlock, firstInner, err := ExtractFencedJSON(text, "first")
	if err != nil {
		t.Fatalf("ExtractFencedJSON(text, \"first\") = _, _, %v; want nil error", err)
	}
	if firstInner != `{"first": true}` {
		t.Errorf("first inner = %q; want %q", firstInner, `{"first": true}`)
	}
	if firstBlock != "```json\n{\"first\": true}\n```" {
		t.Errorf("first block = %q; want the first fenced block verbatim", firstBlock)
	}

	lastBlock, lastInner, err := ExtractFencedJSON(text, "last")
	if err != nil {
		t.Fatalf("ExtractFencedJSON(text, \"last\") = _, _, %v; want nil error", err)
	}
	if lastInner != `{"third": true}` {
		t.Errorf("last inner = %q; want %q", lastInner, `{"third": true}`)
	}
	if lastBlock != "```json\n{\"third\": true}\n```" {
		t.Errorf("last block = %q; want the third fenced block verbatim", lastBlock)
	}
}

func TestExtractFencedJSON_NoBlockReturnsSentinelError(t *testing.T) {
	block, inner, err := ExtractFencedJSON("no fenced block anywhere in this text", "first")
	if !errors.Is(err, ErrNoFencedJSONBlock) {
		t.Fatalf("ExtractFencedJSON(text, \"first\") error = %v; want ErrNoFencedJSONBlock", err)
	}
	if block != "" {
		t.Errorf("block = %q; want empty on no-block error", block)
	}
	if inner != "" {
		t.Errorf("inner = %q; want empty on no-block error", inner)
	}
}

func TestExtractFencedJSON_UnrecognisedSelectorReturnsAnError(t *testing.T) {
	text := "```json\n{}\n```"
	block, inner, err := ExtractFencedJSON(text, "penultimate")
	if err == nil {
		t.Fatal("ExtractFencedJSON(text, \"penultimate\") = _, _, nil; want an error")
	}
	if errors.Is(err, ErrNoFencedJSONBlock) {
		t.Error("unrecognised-selector error must not be ErrNoFencedJSONBlock -- that sentinel means no block was found")
	}
	if block != "" || inner != "" {
		t.Errorf("block, inner = %q, %q; want both empty on an unrecognised-selector error", block, inner)
	}
}

func TestExtractFencedJSON_FenceBodySpansMultipleLines(t *testing.T) {
	text := "```json\n{\n  \"a\": 1,\n  \"b\": 2\n}\n```"
	_, inner, err := ExtractFencedJSON(text, "first")
	if err != nil {
		t.Fatalf("ExtractFencedJSON(text, \"first\") = _, _, %v; want nil error", err)
	}
	want := "{\n  \"a\": 1,\n  \"b\": 2\n}"
	if inner != want {
		t.Errorf("inner = %q; want %q", inner, want)
	}
}
