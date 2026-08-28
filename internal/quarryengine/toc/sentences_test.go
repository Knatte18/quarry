// sentences_test.go covers FirstSentences.

package toc

import "testing"

func TestFirstSentences(t *testing.T) {
	tests := []struct {
		name string
		text string
		n    int
		want string
	}{
		{
			name: "DefaultOneSentence",
			text: "Foo does a thing. It returns an error.",
			n:    1,
			want: "Foo does a thing.",
		},
		{
			name: "AllSentences",
			text: "Foo does a thing. It returns an error.",
			n:    AllSentences,
			want: "Foo does a thing. It returns an error.",
		},
		{
			name: "NLargerThanSentenceCount",
			text: "Foo does a thing. It returns an error.",
			n:    5,
			want: "Foo does a thing. It returns an error.",
		},
		{
			name: "NLessOrEqualZero",
			text: "Foo does a thing.",
			n:    0,
			want: "",
		},
		{
			name: "EgAbbreviationNotSplit",
			text: "Foo does a thing, e.g. this one. Second sentence.",
			n:    1,
			want: "Foo does a thing, e.g. this one.",
		},
		{
			name: "IeAbbreviationNotSplit",
			text: "Foo does a thing, i.e. this one. Second sentence.",
			n:    1,
			want: "Foo does a thing, i.e. this one.",
		},
		{
			name: "SingleLetterInitialNotSplit",
			text: "A. Smith wrote this function. It works.",
			n:    1,
			want: "A. Smith wrote this function.",
		},
		{
			name: "BacktickDottedIdentifierNotSplit",
			text: "Call `pkg.Sub.Func` to do the thing. It returns an error.",
			n:    1,
			want: "Call `pkg.Sub.Func` to do the thing.",
		},
		{
			name: "MultiLineNewlinePreserved",
			text: "Foo does a thing.\nIt spans two lines. Third sentence trimmed away.",
			n:    2,
			want: "Foo does a thing.\nIt spans two lines.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FirstSentences(tt.text, tt.n)
			if got != tt.want {
				t.Errorf("FirstSentences(%q, %d) = %q; want %q", tt.text, tt.n, got, tt.want)
			}
		})
	}
}
