package ladder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testRepoRoot is the repository root, relative to this package directory
// (bench/loomyard-eval/ladder/internal/ladder), for tests that resolve a ladder-declared
// repo-root-relative path against a real committed file. A Go test binary's working directory is its
// own package directory, unlike pytest which happened to run from the repository root, so every such
// test must derive this path explicitly rather than assume the cwd.
const testRepoRoot = "../../../../.."

func TestTaskTextFor_ExtractsTask01BoundedTextWithoutFasitLeads(t *testing.T) {
	l := mustLoadLadder(t)

	text, err := TaskTextFor(l, testRepoRoot, "01-reed-geometry-exploration")
	if err != nil {
		t.Fatalf("TaskTextFor(01-reed-geometry-exploration) = _, %v; want nil error", err)
	}

	if !strings.HasPrefix(text, "Explain how a reed session's terminal geometry") {
		t.Errorf("TaskTextFor(01-reed-geometry-exploration) = %q; want a prefix of \"Explain how a reed session's terminal geometry\"", text)
	}
	if strings.Contains(text, "burler.go:373") {
		t.Errorf("TaskTextFor(01-reed-geometry-exploration) contains \"burler.go:373\"; want the fasit leads excluded")
	}
	if strings.Contains(strings.ToLower(text), "fasit") {
		t.Errorf("TaskTextFor(01-reed-geometry-exploration) contains \"fasit\"; want the fasit leads excluded")
	}
}

func TestTaskTextFor_ExtractsTask04BoundedTextWithoutScoringNotes(t *testing.T) {
	l := mustLoadLadder(t)

	text, err := TaskTextFor(l, testRepoRoot, "04-shedadapters-shuttle-impact")
	if err != nil {
		t.Fatalf("TaskTextFor(04-shedadapters-shuttle-impact) = _, %v; want nil error", err)
	}

	if !strings.HasPrefix(text, "You are about to change the `Shuttle` interface") {
		t.Errorf("TaskTextFor(04-shedadapters-shuttle-impact) = %q; want a prefix of \"You are about to change the `Shuttle` interface\"", text)
	}
	if strings.Contains(text, "burler.go:373") {
		t.Errorf("TaskTextFor(04-shedadapters-shuttle-impact) contains \"burler.go:373\"; want the scoring notes' decoy excluded")
	}
	if strings.Contains(strings.ToLower(text), "fasit") {
		t.Errorf("TaskTextFor(04-shedadapters-shuttle-impact) contains \"fasit\"; want the scoring notes excluded")
	}
}

func TestTaskTextFor_RaisesWhenHeadingAbsent(t *testing.T) {
	l := mustLoadLadder(t)

	bogusTaskFile := filepath.Join(t.TempDir(), "bogus.md")
	if err := os.WriteFile(bogusTaskFile, []byte("# No task text heading here\n"), 0o644); err != nil {
		t.Fatalf("write bogus task file: %v", err)
	}

	// TaskFile is written as an absolute path here, so joining it against testRepoRoot still resolves
	// correctly -- filepath.Join treats a second absolute-looking component as relative, but an
	// absolute bogusTaskFile combined with a relative repoRoot needs no repoRoot contribution at all
	// since Go's TaskTextFor always joins repoRoot with task.TaskFile.
	l.Tasks["bogus"] = TaskEntry{
		TaskFile:  bogusTaskFile,
		PinnedSHA: "deadbeef",
		Worktree:  filepath.Join(t.TempDir(), "wt"),
		Schema:    "exploration",
		Fasit:     "unused",
	}

	if _, err := TaskTextFor(l, "", "bogus"); err == nil {
		t.Errorf("TaskTextFor(bogus) = _, nil; want an error naming the absent heading")
	}
}

func TestTaskTextFor_RaisesWhenTaskUnknown(t *testing.T) {
	l := mustLoadLadder(t)

	if _, err := TaskTextFor(l, testRepoRoot, "no-such-task"); err == nil {
		t.Errorf("TaskTextFor(no-such-task) = _, nil; want an error naming the unknown task")
	}
}

func TestSchemaFor_ImpactSchemaComesFromTheImpactTasksOwnFile(t *testing.T) {
	l := mustLoadLadder(t)

	schema, err := SchemaFor(l, testRepoRoot, "04-shedadapters-shuttle-impact")
	if err != nil {
		t.Fatalf("SchemaFor(04-shedadapters-shuttle-impact) = _, %v; want nil error", err)
	}

	if !strings.HasPrefix(schema, "```json") || !strings.HasSuffix(schema, "```") {
		t.Errorf("SchemaFor(04-shedadapters-shuttle-impact) = %q; want a fenced json block with both fences", schema)
	}
	if !strings.Contains(schema, "callers_to_update") {
		t.Errorf("SchemaFor(04-shedadapters-shuttle-impact) = %q; want the impact schema's callers_to_update field", schema)
	}
}

func TestSchemaFor_ExplorationSchemaComesFromTheBenchmarkReadme(t *testing.T) {
	l := mustLoadLadder(t)

	schema, err := SchemaFor(l, testRepoRoot, "01-reed-geometry-exploration")
	if err != nil {
		t.Fatalf("SchemaFor(01-reed-geometry-exploration) = _, %v; want nil error", err)
	}

	if !strings.HasPrefix(schema, "```json") || !strings.HasSuffix(schema, "```") {
		t.Errorf("SchemaFor(01-reed-geometry-exploration) = %q; want a fenced json block with both fences", schema)
	}
	if !strings.Contains(schema, "relevant_files") {
		t.Errorf("SchemaFor(01-reed-geometry-exploration) = %q; want the exploration schema's relevant_files field", schema)
	}
	if strings.Contains(schema, "callers_to_update") {
		t.Errorf("SchemaFor(01-reed-geometry-exploration) = %q; want the exploration schema, not the impact schema", schema)
	}
}

func TestSchemaFor_RaisesOnUnknownSchema(t *testing.T) {
	l := mustLoadLadder(t)

	l.Tasks["bogus-schema"] = TaskEntry{
		TaskFile:  l.Tasks["01-reed-geometry-exploration"].TaskFile,
		PinnedSHA: "deadbeef",
		Worktree:  filepath.Join(t.TempDir(), "wt"),
		Schema:    "no-such-schema",
		Fasit:     "unused",
	}

	if _, err := SchemaFor(l, testRepoRoot, "bogus-schema"); err == nil {
		t.Errorf("SchemaFor(bogus-schema) = _, nil; want an error naming the unknown schema")
	}
}

func TestSchemaFor_RaisesOnMissingSection(t *testing.T) {
	l := mustLoadLadder(t)

	bogusTaskFile := filepath.Join(t.TempDir(), "bogus.md")
	if err := os.WriteFile(bogusTaskFile, []byte("# No output schema section here\n"), 0o644); err != nil {
		t.Fatalf("write bogus task file: %v", err)
	}
	l.Tasks["bogus-impact"] = TaskEntry{
		TaskFile:  bogusTaskFile,
		PinnedSHA: "deadbeef",
		Worktree:  filepath.Join(t.TempDir(), "wt"),
		Schema:    "impact",
		Fasit:     "unused",
	}

	if _, err := SchemaFor(l, "", "bogus-impact"); err == nil {
		t.Errorf("SchemaFor(bogus-impact) = _, nil; want an error naming the absent section")
	}
}
