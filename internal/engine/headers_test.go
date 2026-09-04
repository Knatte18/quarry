// headers_test.go drives HeaderForFile through one table per format it dispatches to, plus the
// base-name and no-match fallthrough cases HeaderForFile itself is responsible for.

package engine

import "testing"

func TestHeaderForFile(t *testing.T) {
	tests := []struct {
		name string
		base string
		src  string
		want string
	}{
		{
			name: "MarkdownATXHeadingWithParagraph",
			base: "README.md",
			src:  "# Title\n\nThis is the first paragraph.\n\nA second paragraph follows.\n",
			want: "Title\nThis is the first paragraph.",
		},
		{
			name: "MarkdownSetextHeading",
			base: "README.md",
			src:  "Title\n=====\n\nThis is the first paragraph.\n",
			want: "Title\nThis is the first paragraph.",
		},
		{
			name: "MarkdownParagraphFollowsBlankLineAfterHeading",
			base: "README.md",
			src:  "## Title\n\n\nThe paragraph, after an extra blank line.\n",
			want: "Title\nThe paragraph, after an extra blank line.",
		},
		{
			name: "MarkdownNoHeadingAtAll",
			base: "README.md",
			src:  "Just a plain paragraph with no heading above it.\n",
			want: "",
		},
		{
			name: "ShellScriptHashBlockAfterShebang",
			base: "run.sh",
			src:  "#!/usr/bin/env bash\n# Runs the build.\n# Exits non-zero on failure.\n\nset -e\n",
			want: "Runs the build.\nExits non-zero on failure.",
		},
		{
			name: "YAMLWithNoLeadingComment",
			base: "config.yaml",
			src:  "key: value\n",
			want: "",
		},
		{
			name: "HTMLLeadingComment",
			base: "index.html",
			src:  "<!-- This page renders the dashboard. -->\n<html></html>\n",
			want: "This page renders the dashboard.",
		},
		{
			name: "CSSLeadingComment",
			base: "style.css",
			src:  "/* Styles the dashboard layout. */\nbody { margin: 0; }\n",
			want: "Styles the dashboard layout.",
		},
		{
			name: "JavaScriptLineCommentBlock",
			base: "app.js",
			src:  "// Bootstraps the app.\n// Wires up the router.\n\nconst x = 1;\n",
			want: "Bootstraps the app.\nWires up the router.",
		},
		{
			name: "JavaScriptBlockComment",
			base: "app.js",
			src:  "/* Bootstraps the app. */\nconst x = 1;\n",
			want: "Bootstraps the app.",
		},
		{
			name: "MakefileResolvesThroughBaseNameTable",
			base: "Makefile",
			src:  "# Builds the project.\n# Runs the tests too.\n\nall:\n\tgo build ./...\n",
			want: "Builds the project.\nRuns the tests too.",
		},
		{
			name: "DockerfileResolvesThroughBaseNameTable",
			base: "Dockerfile",
			src:  "# Base image for the service.\n\nFROM golang:1\n",
			want: "Base image for the service.",
		},
		{
			name: "ExtensionlessFileInNeitherTable",
			base: "LICENSE",
			src:  "MIT License\n",
			want: "",
		},
		{
			name: "UnknownExtension",
			base: "data.proto",
			src:  "// Defines the wire format.\n",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HeaderForFile(tt.base, []byte(tt.src))
			if got != tt.want {
				t.Errorf("HeaderForFile(%q, ...) = %q; want %q", tt.base, got, tt.want)
			}
		})
	}
}
