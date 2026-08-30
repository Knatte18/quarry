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
