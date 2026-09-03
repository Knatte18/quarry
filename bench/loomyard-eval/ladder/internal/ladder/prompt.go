// prompt.go extracts a task file's two prompt-facing pieces, the task text and the output schema,
// and renders the one preamble every cell receives around them.

package ladder

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TaskContent holds the two things a task file contributes to a rendered prompt: the exercise text
// dedented from its blockquote, and the output-schema fenced JSON block the answer must satisfy.
type TaskContent struct {
	TaskText    string
	SchemaBlock string
}

// taskTextHeadingMarker is the literal, backtick-wrapped token that marks the task-text heading. It
// appears verbatim, with any trailing parenthetical, in every task file's task-text heading line.
const taskTextHeadingMarker = "`<TASK TEXT>`"

// outputSchemaHeadingPrefix is the exact, case-sensitive prefix every task file's output-schema
// heading begins with. The parenthetical that follows differs per task file's schema kind and is not
// checked here -- that distinction lives in the ladder file's own schema key, not in this heading's
// text.
const outputSchemaHeadingPrefix = "## Output schema"

// outputSchemaHeadingPattern matches the output-schema heading at the start of a line.
var outputSchemaHeadingPattern = regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(outputSchemaHeadingPrefix))

// LoadTaskFile reads the task file at path and extracts exactly two things from it: the task text
// quoted under the "## `<TASK TEXT>`" heading, dedented, and the first fenced JSON block following the
// "## Output schema" heading. Extraction is strictly inclusion-based: it takes those two blocks and
// nothing else. In particular no code here ever looks for, matches, or excludes the answer-key
// heading, which is spelt differently across task files -- an exclusion-based extractor keyed on one
// spelling would leak the answer key from the others. A file missing either heading is a hard load
// error naming the file; this function never returns an empty schema.
func LoadTaskFile(path string) (TaskContent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TaskContent{}, fmt.Errorf("load task file %s: %w", path, err)
	}
	text := string(data)

	taskText, err := extractTaskText(text)
	if err != nil {
		return TaskContent{}, fmt.Errorf("load task file %s: %w", path, err)
	}

	schemaBlock, err := extractSchemaBlock(text)
	if err != nil {
		return TaskContent{}, fmt.Errorf("load task file %s: %w", path, err)
	}

	return TaskContent{TaskText: taskText, SchemaBlock: schemaBlock}, nil
}

// extractTaskText locates the task-text heading and returns its blockquote, dedented. Lines beginning
// with ">" are collected until the first line that is neither a blockquote line nor blank; blank lines
// inside the quote are preserved as blank, and a leading ">" plus at most one following space is
// stripped from every collected line so an indented sub-list keeps its relative indentation.
func extractTaskText(text string) (string, error) {
	lines := strings.Split(text, "\n")

	headingIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") && strings.Contains(line, taskTextHeadingMarker) {
			headingIdx = i
			break
		}
	}
	if headingIdx == -1 {
		return "", fmt.Errorf("no task-text heading (%q) found", taskTextHeadingMarker)
	}

	// The heading is followed by a blank separator line before the blockquote itself begins; skip
	// past that separator without collecting it.
	i := headingIdx + 1
	for i < len(lines) && lines[i] == "" {
		i++
	}

	var collected []string
	for ; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			collected = append(collected, "")
			continue
		}
		if !strings.HasPrefix(line, ">") {
			break
		}
		collected = append(collected, dedentQuoteLine(line))
	}

	return strings.TrimRight(strings.Join(collected, "\n"), "\n"), nil
}

// dedentQuoteLine strips a leading ">" and at most one following space from a blockquote line, so an
// indented sub-list keeps its relative indentation.
func dedentQuoteLine(line string) string {
	rest := strings.TrimPrefix(line, ">")
	return strings.TrimPrefix(rest, " ")
}

// extractSchemaBlock locates the first "## Output schema" heading and returns the first fenced JSON
// block found after it, fences included. A file with no such heading is a hard load error; this
// function never returns an empty schema.
func extractSchemaBlock(text string) (string, error) {
	loc := outputSchemaHeadingPattern.FindStringIndex(text)
	if loc == nil {
		return "", fmt.Errorf("no %q heading found", outputSchemaHeadingPrefix)
	}

	block, _, err := ExtractFencedJSON(text[loc[0]:], "first")
	if err != nil {
		return "", fmt.Errorf("no fenced json block found after %q heading: %w", outputSchemaHeadingPrefix, err)
	}
	return block, nil
}
