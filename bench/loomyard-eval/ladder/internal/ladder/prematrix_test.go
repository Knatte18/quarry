// prematrix_test.go is the offline gate that runs before any real `claude -p` call is spent. It is
// keyed on the real committed ladder-toc.yaml, the real committed task files and the real committed
// fasits, so an authoring mistake in any of those is caught for free rather than after thirty real
// runs against the breadth matrix.

package ladder

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		prompt := RenderPrompt(content, prematrixPlaceholderDest, toolNames)

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

// prematrixFasitPath names a new task's fasit file, relative to this test's own directory, and the
// task id in ladder-toc.yaml whose PinnedSHA the fasit's _meta.pinned_sha is checked against.
type prematrixFasitPath struct {
	path   string
	taskID string
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
	l, err := LoadLadder("../../ladder-toc.yaml")
	if err != nil {
		t.Fatalf("LoadLadder(ladder-toc.yaml) = %v; want no error", err)
	}

	fasits := []prematrixFasitPath{
		{path: "../../../tasks/02-shedadapters-exploration.fasit.json", taskID: "02-shedadapters-exploration"},
		{path: "../../../tasks/06-loomyard-cold-start-orientation.fasit.json", taskID: "06-loomyard-cold-start-orientation"},
	}

	for _, ff := range fasits {
		t.Run(ff.taskID, func(t *testing.T) {
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
				t.Fatalf("task %q not found in ladder-toc.yaml", ff.taskID)
			}
			if pinnedSHA != task.PinnedSHA {
				t.Errorf("fasit %q _meta.pinned_sha = %q; want %q (ladder-toc.yaml's tasks.%s.pinned_sha)",
					ff.path, pinnedSHA, task.PinnedSHA, ff.taskID)
			}
		})
	}
}
