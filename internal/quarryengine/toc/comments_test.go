// comments_test.go covers StripLineComment, StripComment, and FirstParagraph.

package toc

import "testing"

func TestStripLineComment(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		prefix string
		want   string
	}{
		{
			name:   "LineComment",
			text:   "// Foo does a thing.\n//\n// It returns an error.",
			prefix: "//",
			want:   "Foo does a thing.\n\nIt returns an error.",
		},
		{
			name:   "EmptyPrefixTrimsOnly",
			text:   "    Foo does a thing.\n\n    It returns an error.\n    ",
			prefix: "",
			want:   "Foo does a thing.\n\nIt returns an error.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripLineComment(tt.text, tt.prefix)
			if got != tt.want {
				t.Errorf("StripLineComment(%q, %q) = %q; want %q", tt.text, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestStripComment(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		prefix string
		want   string
	}{
		{
			name:   "LineCommentMatchesStripLineComment",
			text:   "// Foo does a thing.\n// It returns an error.",
			prefix: "//",
			want:   StripLineComment("// Foo does a thing.\n// It returns an error.", "//"),
		},
		{
			name:   "BlockCommentNoLeadingStar",
			text:   "/*\n Foo does a thing.\n It returns an error.\n*/",
			prefix: "",
			want:   "Foo does a thing.\nIt returns an error.",
		},
		{
			name:   "DoubleStarBlockWithLeadingStars",
			text:   "/**\n * Foo does a thing.\n * It returns an error.\n */",
			prefix: "",
			want:   "Foo does a thing.\nIt returns an error.",
		},
		{
			name:   "SingleLineBlock",
			text:   "/* one liner */",
			prefix: "",
			want:   "one liner",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripComment(tt.text, tt.prefix)
			if got != tt.want {
				t.Errorf("StripComment(%q, %q) = %q; want %q", tt.text, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestFirstParagraph(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "BlockWithBareSeparator",
			text: StripLineComment("// Foo does a thing.\n//\n// A second paragraph.", "//"),
			want: "Foo does a thing.",
		},
		{
			name: "HeaderWithNoBlankLine",
			text: "Foo does a thing across one whole paragraph with no break.",
			want: "Foo does a thing across one whole paragraph with no break.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FirstParagraph(tt.text)
			if got != tt.want {
				t.Errorf("FirstParagraph(%q) = %q; want %q", tt.text, got, tt.want)
			}
		})
	}
}
