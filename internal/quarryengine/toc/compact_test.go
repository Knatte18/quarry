package toc

import (
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestCompactFile(t *testing.T) {
	f := FileTOC{
		Header:   "singlellm.go implements SingleLLMProducer,\nthe adapter. It also does more.",
		Language: "go",
		Package:  "shedadapters",
		Symbols: []Symbol{
			{Kind: KindType, Name: "Shuttle", Signature: "type Shuttle interface", Docstring: "Shuttle is the seam\nSingleLLMProducer drives.", Start: 22, SigEnd: 38, End: 41},
			{Kind: KindMethod, Name: "Call", Owner: "P", Signature: "func (p *P) Call(ctx context.Context) error", Start: 88, SigEnd: 92, End: 151},
		},
	}
	got := CompactFile("internal/x/singlellm.go", f)
	want := "# internal/x/singlellm.go (package shedadapters): singlellm.go implements SingleLLMProducer, the adapter.\n" +
		"22-41: type Shuttle interface -- Shuttle is the seam SingleLLMProducer drives.\n" +
		"88-151: func (p *P) Call(ctx context.Context) error"
	if got != want {
		t.Errorf("CompactFile() =\n%s\nwant\n%s", got, want)
	}

	empty := CompactFile("e.go", FileTOC{Language: "go", Partial: true, Symbols: []Symbol{}})
	if empty != "# e.go [partial] (no symbols)" {
		t.Errorf("empty = %q", empty)
	}
}

func TestCompactDir(t *testing.T) {
	d := DirTOC{
		Files: []DirEntry{
			{Name: "archive.go", Language: "go", Package: "shed", Header: "archive.go implements Xyz.\nMore prose here.", Test: boolPtr(false), Generated: boolPtr(false)},
			{Name: "archive_test.go", Language: "go", Package: "shed", Test: boolPtr(true), Generated: boolPtr(false)},
			{Name: "ext_test.go", Language: "go", Package: "shed_test", Header: "External tests.", Test: boolPtr(true)},
			{Name: "gen.go", Language: "go", Package: "shed", Generated: boolPtr(true), Header: "Code generated. DO NOT EDIT."},
			{Name: "unreadable.go", Language: "go", Error: "open unreadable.go: permission denied"},
		},
		Dirs: []string{"render", "sub"},
	}
	got := CompactDir("internal/shed", d)
	want := strings.Join([]string{
		"# internal/shed (package shed), 5 files",
		"internal/shed/archive.go: archive.go implements Xyz.",
		"internal/shed/archive_test.go [test]",
		"internal/shed/ext_test.go [test] (package shed_test): External tests.",
		"internal/shed/gen.go [generated]: Code generated.",
		"internal/shed/unreadable.go: error: open unreadable.go: permission denied",
		"dirs: render, sub",
	}, "\n")
	if got != want {
		t.Errorf("CompactDir() =\n%s\nwant\n%s", got, want)
	}
	if !strings.HasPrefix(CompactDir("x", DirTOC{Files: []DirEntry{}, Dirs: []string{}}), "# x, 0 files") {
		t.Error("empty dir header")
	}
}

func TestLeadCapsLongProseOnWordBoundary(t *testing.T) {
	long := strings.Repeat("word ", 40) + "end."
	got := lead(long)
	if r := []rune(got); len(r) > leadMaxRunes+1 || !strings.HasSuffix(got, "…") || strings.HasSuffix(got, " …") {
		t.Errorf("lead() = %q (%d runes)", got, len(r))
	}
	if lead("Short. Second sentence.") != "Short." {
		t.Errorf("lead() did not keep the first sentence: %q", lead("Short. Second sentence."))
	}
}
