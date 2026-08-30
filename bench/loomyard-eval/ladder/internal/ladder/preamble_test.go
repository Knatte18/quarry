package ladder

import (
	"strings"
	"testing"
)

const (
	testTargetDir  = "/tmp/loomyard-eval-01"
	testTaskText   = "Explain how the thing works."
	testSchemaJSON = `{"summary": "string"}`
)

// forbiddenLiterals must never appear in a generated preamble: binary paths, CLI-verb syntax, and
// shell-flag syntax have no place in a call-shaped MCP preamble.
var forbiddenLiterals = []string{"/tmp/quarry-bench", "quarry toc dir", "quarry refs", "--target-dir"}

func TestPreambleFor_ControlReproducesTheCommittedBPreambleShape(t *testing.T) {
	l := mustLoadLadder(t)
	a0, err := ConfigByID(l, "a0-none")
	if err != nil {
		t.Fatalf("ConfigByID(l, %q) = _, %v", "a0-none", err)
	}

	prompt := PreambleFor(l, a0, testTargetDir, testTaskText, testSchemaJSON)

	if !strings.Contains(prompt, PARALLEL_OPENING) {
		t.Error("preamble does not contain PARALLEL_OPENING")
	}
	if !strings.Contains(prompt, PARALLEL_BLOCK) {
		t.Error("preamble does not contain PARALLEL_BLOCK")
	}
	if !strings.Contains(prompt, testTargetDir) {
		t.Errorf("preamble does not contain the target dir %q", testTargetDir)
	}
	if !strings.Contains(prompt, testTaskText) {
		t.Errorf("preamble does not contain the task text %q", testTaskText)
	}
	if !strings.Contains(prompt, "standard tools: Read, Grep, Bash, Glob") {
		t.Error("preamble does not contain the committed Agent B tool sentence")
	}
}

func TestPreambleFor_NoneControlNeverMentionsQuarry(t *testing.T) {
	l := mustLoadLadder(t)
	for _, config := range l.Configs {
		if len(config.Allowed) != 0 {
			continue
		}
		t.Run(config.ID, func(t *testing.T) {
			prompt := PreambleFor(l, config, testTargetDir, testTaskText, testSchemaJSON)
			if strings.Contains(strings.ToLower(prompt), "quarry") {
				t.Errorf("PreambleFor(l, %q, ...) mentions \"quarry\": %s", config.ID, prompt)
			}
		})
	}
}

func TestPreambleFor_RungListsExactlyItsAllowedToolsInCanonicalOrder(t *testing.T) {
	l := mustLoadLadder(t)
	for _, config := range l.Configs {
		if len(config.Allowed) == 0 {
			continue
		}
		t.Run(config.ID, func(t *testing.T) {
			prompt := PreambleFor(l, config, testTargetDir, testTaskText, testSchemaJSON)

			// Every allowed tool's mcp__quarry__* name must appear, and no other canonical tool's
			// name may appear.
			for _, tool := range QuarryTools {
				name := MCPName(tool)
				if stringSliceContains(config.Allowed, tool) {
					if !strings.Contains(prompt, name) {
						t.Errorf("PreambleFor(l, %q, ...) does not contain allowed tool %q", config.ID, name)
					}
				} else if strings.Contains(prompt, name) {
					t.Errorf("PreambleFor(l, %q, ...) contains disallowed tool %q", config.ID, name)
				}
			}

			// The allowed tool names must appear in canonical QuarryTools order.
			lastIndex := -1
			for _, tool := range QuarryTools {
				if !stringSliceContains(config.Allowed, tool) {
					continue
				}
				index := strings.Index(prompt, MCPName(tool))
				if index < lastIndex {
					t.Errorf("PreambleFor(l, %q, ...) lists %q out of canonical order", config.ID, MCPName(tool))
				}
				lastIndex = index
			}

			for _, literal := range forbiddenLiterals {
				if strings.Contains(prompt, literal) {
					t.Errorf("PreambleFor(l, %q, ...) contains forbidden literal %q", config.ID, literal)
				}
			}
			if !strings.Contains(prompt, "Never set buildTags on any of these calls") {
				t.Errorf("PreambleFor(l, %q, ...) does not contain the buildTags warning", config.ID)
			}
			if !strings.Contains(prompt, PARALLEL_OPENING) {
				t.Errorf("PreambleFor(l, %q, ...) does not contain PARALLEL_OPENING", config.ID)
			}
			if !strings.Contains(prompt, PARALLEL_BLOCK) {
				t.Errorf("PreambleFor(l, %q, ...) does not contain PARALLEL_BLOCK", config.ID)
			}
		})
	}
}
