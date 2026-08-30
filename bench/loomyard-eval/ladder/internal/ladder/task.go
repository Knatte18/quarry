// task.go ports the prompt-input extraction half of scripts/run_ladder.py: pulling a task's
// task-text block and output schema out of the committed task files and the benchmark README, so the
// preamble is assembled from the tracked source rather than from a copy.

package ladder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TaskTextHeading is the exact heading text both task files use to introduce their identical
// task-text block. The section boundary this heading anchors is load-bearing rather than tidiness:
// task 01's very next section carries its fasit leads, and task 04's carries its scoring notes naming
// the real callers and the burler.go:373 decoy outright -- an extractor that reads past this boundary
// would paste the answer key into every run's prompt.
const TaskTextHeading = "## `<TASK TEXT>` (identical for A, B, C)"

// TaskTextFor extracts taskKey's task-text block from its committed task file, resolved against
// repoRoot -- every ladder-declared path (the task file here, the schema and answer-key reads
// elsewhere) is repo-root-relative and routes through this same parameter rather than deriving a root
// of its own, since the Python original only worked because pytest happened to run from the
// repository root while a Go test binary's working directory is its own package directory.
//
// It starts at the line whose heading text is TaskTextHeading, takes every following line up to but
// not including the next line beginning with "## ", strips a leading "> " or ">" from each line, and
// trims surrounding blank lines. Returns an error when the heading is absent or the extracted body is
// empty.
func TaskTextFor(l *Ladder, repoRoot, taskKey string) (string, error) {
	task, ok := l.Tasks[taskKey]
	if !ok {
		return "", fmt.Errorf("task text for: unknown task %q", taskKey)
	}

	taskFile := filepath.Join(repoRoot, task.TaskFile)
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return "", fmt.Errorf("task text for: read %s: %w", taskFile, err)
	}
	lines := strings.Split(string(data), "\n")

	start := -1
	for i, line := range lines {
		if line == TaskTextHeading {
			start = i
			break
		}
	}
	if start == -1 {
		return "", fmt.Errorf("task text for: %s has no %q heading", taskFile, TaskTextHeading)
	}

	var bodyLines []string
	for _, line := range lines[start+1:] {
		if strings.HasPrefix(line, "## ") {
			break
		}
		switch {
		case strings.HasPrefix(line, "> "):
			bodyLines = append(bodyLines, line[2:])
		case strings.HasPrefix(line, ">"):
			bodyLines = append(bodyLines, line[1:])
		default:
			bodyLines = append(bodyLines, line)
		}
	}

	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
		bodyLines = bodyLines[1:]
	}
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[len(bodyLines)-1]) == "" {
		bodyLines = bodyLines[:len(bodyLines)-1]
	}

	if len(bodyLines) == 0 {
		return "", fmt.Errorf("task text for: %s extracted an empty task-text body", taskFile)
	}

	return strings.Join(bodyLines, "\n"), nil
}

// firstFencedJSONBlock returns the first ```json ... ``` fenced block found in text, fences included --
// the extracted schema is embedded verbatim into the preamble as measured stimulus, and the fences are
// part of that text. Returns an error when none is present.
func firstFencedJSONBlock(text string) (string, error) {
	block, _, err := ExtractFencedJSON(text, "first")
	if err != nil {
		return "", fmt.Errorf("schema for: no fenced json block found in the expected section: %w", err)
	}
	return block, nil
}

// section returns the text between heading (exclusive) and the next line starting with "## "
// (exclusive), or to the end of text when there is no next "## " line. Returns an error when heading is
// absent.
func section(text, heading string) (string, error) {
	idx := strings.Index(text, heading)
	if idx == -1 {
		return "", fmt.Errorf("schema for: no %q section found", heading)
	}
	body := text[idx+len(heading):]
	if nextHeadingIdx := strings.Index(body, "\n## "); nextHeadingIdx != -1 {
		body = body[:nextHeadingIdx]
	}
	return body, nil
}

// impactSchemaHeading is the heading task 04's own task file uses to introduce its output schema.
const impactSchemaHeading = "## Output schema (impact-analysis tasks)"

// explorationSchemasHeading is the heading the benchmark README uses to introduce every task type's
// output schema, including the exploration schema task 01's own task file has no section for.
const explorationSchemasHeading = "## Output schemas"

// explorationSchemaMarker is the bold marker under explorationSchemasHeading that introduces the
// exploration schema specifically, among the README's other schema families.
const explorationSchemaMarker = "**Exploration tasks:**"

// benchmarkReadmePath is the repo-root-relative path to the benchmark README the exploration schema is
// read from.
const benchmarkReadmePath = "bench/loomyard-eval/README.md"

// SchemaFor returns taskKey's output schema (a fenced ```json ... ``` block, with fences), from the
// source that actually holds it -- which differs per task and is not uniform. repoRoot resolves both
// the task file's and the benchmark README's repo-root-relative paths, for the same reason TaskTextFor
// takes it.
//
// The impact schema is in the task's own file, under impactSchemaHeading. Task 01 has no schema section
// at all, so the exploration schema comes from the benchmark README's explorationSchemasHeading section,
// under its explorationSchemaMarker. Selection is driven by the task's declared Schema field, never by
// the task key.
//
// Returns an error when the task or schema is unknown, or when the named section is absent.
func SchemaFor(l *Ladder, repoRoot, taskKey string) (string, error) {
	task, ok := l.Tasks[taskKey]
	if !ok {
		return "", fmt.Errorf("schema for: unknown task %q", taskKey)
	}

	switch task.Schema {
	case "impact":
		taskFile := filepath.Join(repoRoot, task.TaskFile)
		data, err := os.ReadFile(taskFile)
		if err != nil {
			return "", fmt.Errorf("schema for: read %s: %w", taskFile, err)
		}
		impactSection, err := section(string(data), impactSchemaHeading)
		if err != nil {
			return "", err
		}
		return firstFencedJSONBlock(impactSection)

	case "exploration":
		readmePath := filepath.Join(repoRoot, benchmarkReadmePath)
		data, err := os.ReadFile(readmePath)
		if err != nil {
			return "", fmt.Errorf("schema for: read %s: %w", readmePath, err)
		}
		schemasSection, err := section(string(data), explorationSchemasHeading)
		if err != nil {
			return "", err
		}
		markerIdx := strings.Index(schemasSection, explorationSchemaMarker)
		if markerIdx == -1 {
			return "", fmt.Errorf("schema for: %s's %q section has no %q marker", readmePath, explorationSchemasHeading, explorationSchemaMarker)
		}
		return firstFencedJSONBlock(schemasSection[markerIdx:])

	default:
		return "", fmt.Errorf("schema for: unknown schema %q for task %q", task.Schema, taskKey)
	}
}
