// prematrix_test.go is the offline gate that runs before any real `claude -p` call is spent. It is
// keyed on the real committed ladder-toc.yaml, the real committed task files and the real committed
// fasits, so an authoring mistake in any of those is caught for free rather than after thirty real
// runs against the breadth matrix.

package ladder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// prematrixPlaceholderRepoRoot stands in for the pinned quarry repository root in
// TestPreMatrix_ControlPromptsAreBlind. It is a fixed placeholder string, not a real path on this
// machine, since this is a tracked test file.
const prematrixPlaceholderRepoRoot = "/placeholder/quarry-repo-root"

// prematrixPlaceholderDest stands in for the pinned target worktree path in the same test.
const prematrixPlaceholderDest = "/placeholder/target-worktree"

// prematrixRepoRootFromPackageDir joins onto a ladder file's repository-relative task_file path to
// resolve it from this package's own directory: five levels up, past internal, ladder, loomyard-eval
// and bench, reaches the repository root, and task.TaskFile is already repository-relative from
// there. run.go's resolveRepoRelative does the equivalent join against the process's real
// quarryRepoRoot; this test cannot use that helper because it has no real pinned worktree to point
// it at.
const prematrixRepoRootFromPackageDir = "../../../../../"

// TestPreMatrix_ControlPromptsAreBlind loads the real, tracked ladder-toc.yaml and asserts, for every
// control cell it declares, that the fully rendered prompt run.go's runCellRepetition would send
// passes CheckRenderedControlPrompt. This is the only pre-matrix check that catches a new prompt
// carrying the bare token "toc" or "quarry" before it voids a control cell for the whole matrix: a
// void control repetition is written by writeVoidRepetition, is flagged blinding_failed, produces no
// invalid_reason.txt at all, and does not abort the run, so every invocation re-attempts and
// re-fails it deterministically while the paired rung cell spends five real calls against a control
// that can never complete.
func TestPreMatrix_ControlPromptsAreBlind(t *testing.T) {
	l, err := LoadLadder("../../ladder-toc.yaml")
	if err != nil {
		t.Fatalf("LoadLadder(ladder-toc.yaml) = %v; want no error", err)
	}

	// Every control the amended file declares, not only the three cells this breadth matrix runs, so
	// the gate covers the whole file and a0-none does not sit unguarded.
	controlIDs := []string{"a0-none", "b0-none", "c0-none", "d0-none"}

	for _, id := range controlIDs {
		cfg, ok := l.ConfigByID(id)
		if !ok {
			t.Errorf("config %q not found in ladder-toc.yaml", id)
			continue
		}
		task, ok := l.Tasks[cfg.Task]
		if !ok {
			t.Errorf("task %q for config %q not found in ladder-toc.yaml", cfg.Task, id)
			continue
		}

		taskFilePath := filepath.Join(prematrixRepoRootFromPackageDir, task.TaskFile)
		content, err := LoadTaskFile(taskFilePath)
		if err != nil {
			t.Errorf("LoadTaskFile(%q) for config %q = %v; want no error", taskFilePath, id, err)
			continue
		}

		toolNames := grantedToolNames(l, cfg)
		prompt := RenderPrompt(content, prematrixPlaceholderDest, toolNames, "")

		in := BlindingInput{
			MCPPrefix:      l.MCPPrefix(),
			ServerName:     l.ServerName(),
			QuarryRepoRoot: prematrixPlaceholderRepoRoot,
		}
		if f := CheckRenderedControlPrompt(prompt, in, l.QuarryTools); f != nil {
			t.Errorf("CheckRenderedControlPrompt for control %q = %+v; want nil (%s)", id, f, f.Message)
		}
	}
}

// TestPreMatrix_KickstartCellPromptsAreBlind loads the real, tracked ladder-kickstart.yaml and
// asserts, for every one of its three cells -- not only the control, since D2's decoupling of
// IsControl from GrantsTools means the blinding gate now covers every cell that grants no tools --
// that the fully rendered prompt run.go's runCellRepetition would send, card included, passes
// CheckRenderedControlPrompt. This is the only pre-matrix check that catches a card carrying the
// blinded token "quarry" before it voids a cell for the whole matrix.
func TestPreMatrix_KickstartCellPromptsAreBlind(t *testing.T) {
	l, err := LoadLadder("../../ladder-kickstart.yaml")
	if err != nil {
		t.Fatalf("LoadLadder(ladder-kickstart.yaml) = %v; want no error", err)
	}

	cellIDs := []string{"e0-names", "e1-pack", "e2-files"}

	for _, id := range cellIDs {
		cfg, ok := l.ConfigByID(id)
		if !ok {
			t.Errorf("config %q not found in ladder-kickstart.yaml", id)
			continue
		}
		if cfg.GrantsTools() {
			t.Errorf("config %q grants tools %v; want none -- every cell in this ladder file is tool-less", id, cfg.Allowed)
			continue
		}
		task, ok := l.Tasks[cfg.Task]
		if !ok {
			t.Errorf("task %q for config %q not found in ladder-kickstart.yaml", cfg.Task, id)
			continue
		}

		taskFilePath := filepath.Join(prematrixRepoRootFromPackageDir, task.TaskFile)
		content, err := LoadTaskFile(taskFilePath)
		if err != nil {
			t.Errorf("LoadTaskFile(%q) for config %q = %v; want no error", taskFilePath, id, err)
			continue
		}

		var card string
		if cfg.Card != "" {
			cardPath := filepath.Join(prematrixRepoRootFromPackageDir, cfg.Card)
			card, err = LoadCardFile(cardPath)
			if err != nil {
				t.Errorf("LoadCardFile(%q) for config %q = %v; want no error", cardPath, id, err)
				continue
			}
		}

		toolNames := grantedToolNames(l, cfg)
		prompt := RenderPrompt(content, prematrixPlaceholderDest, toolNames, card)

		in := BlindingInput{
			MCPPrefix:      l.MCPPrefix(),
			ServerName:     l.ServerName(),
			QuarryRepoRoot: prematrixPlaceholderRepoRoot,
		}
		if f := CheckRenderedControlPrompt(prompt, in, l.QuarryTools); f != nil {
			t.Errorf("CheckRenderedControlPrompt for cell %q = %+v; want nil (%s)", id, f, f.Message)
		}
	}
}

// prematrixFasitPath names a new task's fasit file, relative to this test's own directory, the
// ladder file (relative to the same directory) whose Tasks entry names it, and the task id within
// that file whose PinnedSHA the fasit's _meta.pinned_sha is checked against.
type prematrixFasitPath struct {
	path       string
	ladderFile string
	taskID     string
}

// TestPreMatrix_NewFasitsAreWellFormed asserts both new fasit files decode as JSON and carry the
// exploration schema's scored keys, non-empty relevant_files and key_symbols, well-formed key_symbols
// entries, and a _meta.pinned_sha matching the pin the loaded ladder file records for that task -- the
// machine-checkable half of the degenerate-fasit guard, since ExplorationRule computes recall and
// precision against exactly relevant_files and key_symbols, and an empty one deflates recall
// uniformly across the control and the rung and hides the separation this matrix exists to find. The
// judgement half -- whether the answer is substantive rather than merely present -- is the
// fasit-authoring cards' own and is not attempted here.
func TestPreMatrix_NewFasitsAreWellFormed(t *testing.T) {
	ladders := map[string]*Ladder{}
	for _, path := range []string{"../../ladder-toc.yaml", "../../ladder-kickstart.yaml"} {
		l, err := LoadLadder(path)
		if err != nil {
			t.Fatalf("LoadLadder(%s) = %v; want no error", path, err)
		}
		ladders[path] = l
	}

	fasits := []prematrixFasitPath{
		{path: "../../../tasks/02-shedadapters-exploration.fasit.json", ladderFile: "../../ladder-toc.yaml", taskID: "02-shedadapters-exploration"},
		{path: "../../../tasks/06-loomyard-cold-start-orientation.fasit.json", ladderFile: "../../ladder-toc.yaml", taskID: "06-loomyard-cold-start-orientation"},
		{path: "../../../tasks/07-fabric-merge-state-tracing.fasit.json", ladderFile: "../../ladder-kickstart.yaml", taskID: "07-fabric-merge-state-tracing"},
	}

	for _, ff := range fasits {
		t.Run(ff.taskID, func(t *testing.T) {
			l := ladders[ff.ladderFile]
			data, err := os.ReadFile(ff.path)
			if err != nil {
				t.Fatalf("ReadFile(%q) = %v; want no error", ff.path, err)
			}

			var fasit map[string]any
			if err := json.Unmarshal(data, &fasit); err != nil {
				t.Fatalf("json.Unmarshal(%q) = %v; want valid JSON", ff.path, err)
			}

			for _, key := range []string{"relevant_files", "key_symbols", "summary", "confidence", "open_questions"} {
				if _, ok := fasit[key]; !ok {
					t.Errorf("fasit %q missing key %q", ff.path, key)
				}
			}

			relevantFiles, ok := fasit["relevant_files"].([]any)
			if !ok || len(relevantFiles) == 0 {
				t.Errorf("fasit %q relevant_files = %v; want a non-empty array", ff.path, fasit["relevant_files"])
			}

			keySymbols, ok := fasit["key_symbols"].([]any)
			if !ok || len(keySymbols) == 0 {
				t.Errorf("fasit %q key_symbols = %v; want a non-empty array", ff.path, fasit["key_symbols"])
			}
			for i, raw := range keySymbols {
				sym, ok := raw.(map[string]any)
				if !ok {
					t.Errorf("fasit %q key_symbols[%d] = %v; want an object", ff.path, i, raw)
					continue
				}
				for _, field := range []string{"name", "file", "role"} {
					s, ok := sym[field].(string)
					if !ok || s == "" {
						t.Errorf("fasit %q key_symbols[%d].%s = %v; want a non-empty string", ff.path, i, field, sym[field])
					}
				}
			}

			meta, ok := fasit["_meta"].(map[string]any)
			if !ok {
				t.Fatalf("fasit %q _meta = %v; want an object", ff.path, fasit["_meta"])
			}
			pinnedSHA, ok := meta["pinned_sha"].(string)
			if !ok {
				t.Fatalf("fasit %q _meta.pinned_sha = %v; want a string", ff.path, meta["pinned_sha"])
			}

			task, ok := l.Tasks[ff.taskID]
			if !ok {
				t.Fatalf("task %q not found in %s", ff.taskID, ff.ladderFile)
			}
			if pinnedSHA != task.PinnedSHA {
				t.Errorf("fasit %q _meta.pinned_sha = %q; want %q (%s's tasks.%s.pinned_sha)",
					ff.path, pinnedSHA, task.PinnedSHA, ff.ladderFile, ff.taskID)
			}
		})
	}
}

// kickstartCardPaths names the three kick-start card files, relative to this test's own directory,
// in the order the ladder file's pack_targets and each card's own Uses: list carry the glyphs.
var kickstartCardPaths = []string{
	"../../../cards/07-e0-names.md",
	"../../../cards/07-e1-pack.md",
	"../../../cards/07-e2-files.md",
}

// extractUsesList reads the "Uses:" heading in cardText and returns the glyph names listed under
// it, one per "- " prefixed line, in order, stopping at the first blank line or the end of the
// text. It errors when no "Uses:" heading is found at all -- every kick-start card carries one by
// construction, so a missing heading here means the card itself is malformed, not that the glyph
// list is legitimately empty.
func extractUsesList(cardText string) ([]string, error) {
	lines := strings.Split(cardText, "\n")
	headingIdx := -1
	for i, line := range lines {
		if line == "Uses:" {
			headingIdx = i
			break
		}
	}
	if headingIdx == -1 {
		return nil, fmt.Errorf(`no "Uses:" heading found`)
	}
	var glyphs []string
	for _, line := range lines[headingIdx+1:] {
		if line == "" {
			break
		}
		glyphs = append(glyphs, strings.TrimPrefix(line, "- "))
	}
	return glyphs, nil
}

// TestPreMatrix_KickstartCardsShareOneUsesList reads the three kick-start card files and asserts
// their Uses: sections are byte-identical. The three lists being identical is the property that
// makes the three arms differ only in the dimension under test, and no other code enforces it --
// this is the mechanical backstop to the eye check the batch's design note describes.
func TestPreMatrix_KickstartCardsShareOneUsesList(t *testing.T) {
	var first []string
	var firstPath string
	for _, path := range kickstartCardPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) = %v; want no error", path, err)
		}
		uses, err := extractUsesList(string(data))
		if err != nil {
			t.Fatalf("extractUsesList(%q) = %v; want no error", path, err)
		}
		if first == nil {
			first = uses
			firstPath = path
			continue
		}
		if len(uses) != len(first) {
			t.Fatalf("%q Uses: list has %d entries; want %d, matching %q", path, len(uses), len(first), firstPath)
		}
		for i := range uses {
			if uses[i] != first[i] {
				t.Errorf("%q Uses:[%d] = %q; want %q, matching %q", path, i, uses[i], first[i], firstPath)
			}
		}
	}
}

// TestPreMatrix_KickstartUsesListMatchesPackTargets parses each kick-start card's Uses: entries and
// asserts they equal the loaded ladder file's own pack_targets list, element for element, in order.
// This is a different invariant from TestPreMatrix_KickstartCardsShareOneUsesList: three cards can
// agree with each other while all three disagree with the ladder file, which is exactly the state
// the glyph substitution procedure risks producing, since it edits the ladder file's list and the
// three cards' lists as separate steps.
func TestPreMatrix_KickstartUsesListMatchesPackTargets(t *testing.T) {
	l, err := LoadLadder("../../ladder-kickstart.yaml")
	if err != nil {
		t.Fatalf("LoadLadder(ladder-kickstart.yaml) = %v; want no error", err)
	}

	for _, path := range kickstartCardPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) = %v; want no error", path, err)
		}
		uses, err := extractUsesList(string(data))
		if err != nil {
			t.Fatalf("extractUsesList(%q) = %v; want no error", path, err)
		}
		if len(uses) != len(l.PackTargets) {
			t.Fatalf("%q Uses: list has %d entries; want %d, matching ladder-kickstart.yaml's pack_targets", path, len(uses), len(l.PackTargets))
		}
		for i := range uses {
			if uses[i] != l.PackTargets[i] {
				t.Errorf("%q Uses:[%d] = %q; want %q (ladder-kickstart.yaml's pack_targets[%d])", path, i, uses[i], l.PackTargets[i], i)
			}
		}
	}
}

// TestPreMatrix_KickstartPackCellCardHasSentinels asserts the treatment card's block extracts
// cleanly through ExtractPackBlock -- the authoring-time half of run's pre-rep-1 verification, which
// otherwise first fails at matrix start rather than at authoring time.
func TestPreMatrix_KickstartPackCellCardHasSentinels(t *testing.T) {
	data, err := os.ReadFile("../../../cards/07-e1-pack.md")
	if err != nil {
		t.Fatalf("ReadFile(07-e1-pack.md) = %v; want no error", err)
	}
	if _, err := ExtractPackBlock(string(data)); err != nil {
		t.Errorf("ExtractPackBlock(07-e1-pack.md) = %v; want no error", err)
	}
}
