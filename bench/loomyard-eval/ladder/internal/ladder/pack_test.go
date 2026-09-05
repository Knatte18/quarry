// pack_test.go covers pack.go's two pure surfaces: RenderKickstartPack, the renderer that turns a
// batched resolve into the kick-start pack block, and the sentinel-delimited read/write protocol,
// ExtractPackBlock/WritePackIntoCard/PackBlockSHA256, that puts a rendered pack into a card. The Pack
// entry point itself, which needs a real repository and worktree, is covered separately in
// TestPack_* alongside the end-to-end suite.

package ladder

import (
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// foundResult builds a fabricated found *quarry.ResolveResult carrying exactly one symbol, for
// driving RenderKickstartPack without a real repository -- the renderer is pure.
func foundResult(target, file string, start, end int, signature string) quarry.ResolveResult {
	return quarry.ResolveResult{
		Target: target,
		Status: quarry.StatusFound,
		Symbols: []quarry.Symbol{{
			ID:        target,
			Kind:      quarry.KindFunction,
			File:      file,
			Start:     start,
			End:       end,
			Signature: signature,
		}},
	}
}

func TestRenderKickstartPack_GoldenBlock(t *testing.T) {
	results := []quarry.ResolveResult{
		foundResult("pkg.Foo", "pkg/foo.go", 1, 3, "func Foo()"),
		foundResult("pkg.Bar", "pkg/bar.go", 5, 8, "func Bar(\n\ta int,\n) error"),
		foundResult("pkg.Baz", "pkg/baz.go", 10, 10, "const Baz = 1"),
	}
	want := strings.Join([]string{
		"pkg.Foo → pkg/foo.go 1-3",
		"    func Foo()",
		"pkg.Bar → pkg/bar.go 5-8",
		"    func Bar( a int, ) error",
		"pkg.Baz → pkg/baz.go 10-10",
		"    const Baz = 1",
	}, "\n")

	got, err := RenderKickstartPack(results)
	if err != nil {
		t.Fatalf("RenderKickstartPack() = %v; want no error", err)
	}
	if got != want {
		t.Errorf("RenderKickstartPack() = %q; want %q", got, want)
	}
}

func TestRenderKickstartPack_FatalStatuses(t *testing.T) {
	tests := []struct {
		name   string
		bad    quarry.ResolveResult
		target string
	}{
		{
			name:   "NotFound",
			bad:    quarry.ResolveResult{Target: "x.NotFound", Status: quarry.StatusNotFound},
			target: "x.NotFound",
		},
		{
			name:   "Ambiguous",
			bad:    quarry.ResolveResult{Target: "x.Amb", Status: quarry.StatusAmbiguous},
			target: "x.Amb",
		},
		{
			name:   "Multipart",
			bad:    quarry.ResolveResult{Target: "x.Multi", Status: quarry.StatusMultipart},
			target: "x.Multi",
		},
		{
			name:   "PreResolutionError",
			bad:    quarry.ResolveResult{Target: "x.Bad", Error: "not a valid glyph"},
			target: "x.Bad",
		},
		{
			name:   "FoundWithNoSymbols",
			bad:    quarry.ResolveResult{Target: "x.Empty", Status: quarry.StatusFound, Symbols: nil},
			target: "x.Empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := []quarry.ResolveResult{
				foundResult("pkg.Good", "pkg/good.go", 1, 2, "func Good()"),
				tt.bad,
			}
			got, err := RenderKickstartPack(results)
			if err == nil {
				t.Fatalf("RenderKickstartPack() = nil error; want one naming %q", tt.target)
			}
			if !strings.Contains(err.Error(), tt.target) {
				t.Errorf("RenderKickstartPack() error = %q; want it to name %q", err, tt.target)
			}
			if got != "" {
				t.Errorf("RenderKickstartPack() = %q; want no partial output on a fatal input", got)
			}
		})
	}
}

// cardWithSentinels builds a card carrying inner between the two pack sentinels, with surrounding
// prose before and after, for driving the sentinel-delimited protocol's happy path.
func cardWithSentinels(inner string) string {
	return strings.Join([]string{
		"# Title",
		"",
		"Some text before.",
		"",
		PackSentinelBegin,
		inner,
		PackSentinelEnd,
		"",
		"Some text after.",
		"",
	}, "\n")
}

func TestPackBlockRoundTrip(t *testing.T) {
	card := cardWithSentinels("")
	pack := "line one\nline two"

	written, err := WritePackIntoCard(card, pack)
	if err != nil {
		t.Fatalf("WritePackIntoCard() = %v; want no error", err)
	}

	if !strings.HasPrefix(written, "# Title\n\nSome text before.\n\n"+PackSentinelBegin+"\n") {
		t.Errorf("WritePackIntoCard() = %q; want the text before the sentinel preserved byte for byte", written)
	}
	if !strings.HasSuffix(written, "\n"+PackSentinelEnd+"\n\nSome text after.\n") {
		t.Errorf("WritePackIntoCard() = %q; want the text after the sentinel preserved byte for byte", written)
	}

	extracted, err := ExtractPackBlock(written)
	if err != nil {
		t.Fatalf("ExtractPackBlock() = %v; want no error", err)
	}
	if extracted != pack {
		t.Errorf("ExtractPackBlock() = %q; want exactly what was written, %q", extracted, pack)
	}

	writtenAgain, err := WritePackIntoCard(written, pack)
	if err != nil {
		t.Fatalf("second WritePackIntoCard() = %v; want no error", err)
	}
	if writtenAgain != written {
		t.Errorf("writing the same pack twice did not yield the same file:\nfirst:  %q\nsecond: %q", written, writtenAgain)
	}

	emptyExtracted, err := ExtractPackBlock(card)
	if err != nil {
		t.Fatalf("ExtractPackBlock(empty block) = %v; want no error", err)
	}
	if emptyExtracted != "" {
		t.Errorf("ExtractPackBlock(empty block) = %q; want the empty string", emptyExtracted)
	}
}

func TestPackBlockErrors(t *testing.T) {
	tests := []struct {
		name string
		card string
	}{
		{
			name: "NoSentinels",
			card: "no sentinels at all\njust text",
		},
		{
			name: "OnlyBegin",
			card: PackSentinelBegin + "\nonly begin",
		},
		{
			name: "OnlyEnd",
			card: "only end\n" + PackSentinelEnd,
		},
		{
			name: "SentinelTwice",
			card: PackSentinelBegin + "\n" + PackSentinelBegin + "\n" + PackSentinelEnd,
		},
		{
			name: "WrongOrder",
			card: PackSentinelEnd + "\n" + PackSentinelBegin,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ExtractPackBlock(tt.card); err == nil {
				t.Error("ExtractPackBlock() = nil error; want one rejecting the malformed sentinels")
			}
			if _, err := WritePackIntoCard(tt.card, "some pack"); err == nil {
				t.Error("WritePackIntoCard() = nil error; want one rejecting the malformed sentinels, not a silent append")
			}
		})
	}
}

func TestPackBlockSHA256_MatchesWrittenBlock(t *testing.T) {
	card := cardWithSentinels("")
	pack := "the pack block's own content"

	written, err := WritePackIntoCard(card, pack)
	if err != nil {
		t.Fatalf("WritePackIntoCard() = %v; want no error", err)
	}
	extracted, err := ExtractPackBlock(written)
	if err != nil {
		t.Fatalf("ExtractPackBlock() = %v; want no error", err)
	}

	got := PackBlockSHA256(extracted)
	want := PackBlockSHA256(pack)
	if got != want {
		t.Errorf("PackBlockSHA256(extracted) = %s; want PackBlockSHA256(written) = %s", got, want)
	}
}
