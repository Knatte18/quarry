// annex.go is the "annex" arm of the ladder (HANDOFF.md §2a): quarry as mechanical pre-processing.
//
// An annex config grants the run agent no quarry tools at all. Instead, prepare-session runs the quarry
// CLI itself against the task's pinned worktree, before the session starts, and the resulting text is
// injected into the run prompt as a neutral "pre-computed analysis attachment". The agent's prompt is
// otherwise byte-identical to the control's, so the annex text is the only difference between an annex
// cell and its ladder's control -- what this arm measures is the value of the material, with zero
// agent-side tool cost and none of the tool-addressing friction the tool-granted rungs pay.
//
// Blinding: an annex cell is a blinded cell. The text reaches the transcript through the prompt, so it
// must never carry the word "quarry" or a path into the quarry repo; GenerateAnnex refuses to return
// such a text rather than let ingest's blinding gate fail an hour later.
package ladder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Annex kinds. Each names one shape of material the quarry CLI can produce mechanically for a task.
const (
	// AnnexTocDir is `toc dir` over Dirs: every source file with its package and header comment, one
	// listing per directory. The cheapest map of a scope.
	AnnexTocDir = "toc-dir"
	// AnnexTocFull is AnnexTocDir followed by `toc file` for every file those listings name: the full
	// symbol map of the scope, every declaration with its signature, docstring, and line range.
	AnnexTocFull = "toc-full"
	// AnnexTocFile is `toc file` over Files only: the symbol map of the named files and nothing else.
	AnnexTocFile = "toc-file"
	// AnnexImpact is `impact --in-file InFile [--within Within] Symbol`: the symbol's declaration and
	// every verified call site, each with its enclosing declaration. DropCallers > 0 deliberately removes
	// the last N callers from the result, to measure whether an agent verifies an annex or trusts it.
	AnnexImpact = "impact"
	// AnnexPlanPack is the implementer's pack for a plan of the shape "use symbols X and Y to edit symbol
	// Z": the declaration location of every Use symbol (`definition --in-file`), then the full impact of
	// every Change symbol (`impact --in-file`), so the agent can read everything it needs in one pass
	// instead of locating each symbol itself. DropCallers applies to every Change entry.
	AnnexPlanPack = "plan-pack"
)

// annexKinds is the closed set LoadLadder validates against.
var annexKinds = []string{AnnexTocDir, AnnexTocFull, AnnexTocFile, AnnexImpact, AnnexPlanPack}

// AnnexSpec is one named recipe under a task's `annexes:` map. A config selects one by name through its
// own `annex:` field; the recipe's targets belong to the task, not to the config, because they name
// things in the task's codebase.
type AnnexSpec struct {
	// Kind is one of the Annex* constants.
	Kind string `yaml:"kind"`
	// Dirs, for toc-dir and toc-full: directories relative to the worktree. A glob pattern (e.g.
	// "internal/*") is expanded at generation time and keeps only the directories it matches, sorted.
	Dirs []string `yaml:"dirs"`
	// Files, for toc-file: files relative to the worktree.
	Files []string `yaml:"files"`
	// Symbol, for impact: the bare symbol name resolved within InFile.
	Symbol string `yaml:"symbol"`
	// InFile, for impact: the file the symbol is declared in, relative to the worktree.
	InFile string `yaml:"in_file"`
	// Within, for impact, optional: restrict reported callers to this directory.
	Within string `yaml:"within"`
	// DropCallers, for impact and plan-pack, optional: remove the last N callers from each impact result
	// before injection.
	DropCallers int `yaml:"drop_callers"`
	// Compact, for the toc kinds: render the compact plain-text form (`toc ... --compact`) instead of
	// JSON -- the same map at roughly a third to a fifth of the bytes.
	Compact bool `yaml:"compact"`
	// Use, for plan-pack: the symbols the plan tells the implementer to use; their declaration
	// locations are injected.
	Use []SymbolRef `yaml:"use"`
	// Change, for plan-pack: the symbols the plan tells the implementer to edit; their declaration and
	// every call site are injected.
	Change []SymbolRef `yaml:"change"`
}

// SymbolRef names one symbol by its declaring file, the only unambiguous cheap address the CLI takes.
type SymbolRef struct {
	// Symbol is the bare name (no Type. qualifier).
	Symbol string `yaml:"symbol"`
	// InFile is the file it is declared in, relative to the worktree.
	InFile string `yaml:"in_file"`
	// Within, for Change entries only, optional: restrict reported callers to this directory.
	Within string `yaml:"within"`
}

// validateAnnexSpec returns the rule an AnnexSpec breaks, or "" when it is well-formed.
func validateAnnexSpec(spec AnnexSpec) string {
	if !stringSliceContains(annexKinds, spec.Kind) {
		return fmt.Sprintf("kind %q is not one of %v", spec.Kind, annexKinds)
	}
	switch spec.Kind {
	case AnnexTocDir, AnnexTocFull:
		if len(spec.Dirs) == 0 {
			return "needs at least one entry in dirs"
		}
		if len(spec.Files) != 0 || spec.Symbol != "" || spec.InFile != "" || spec.Within != "" || spec.DropCallers != 0 {
			return "takes dirs only"
		}
	case AnnexTocFile:
		if len(spec.Files) == 0 {
			return "needs at least one entry in files"
		}
		if len(spec.Dirs) != 0 || spec.Symbol != "" || spec.InFile != "" || spec.Within != "" || spec.DropCallers != 0 {
			return "takes files only"
		}
	case AnnexImpact:
		if spec.Compact {
			return "compact applies to the toc kinds only"
		}
		if spec.Symbol == "" || spec.InFile == "" {
			return "needs symbol and in_file"
		}
		if len(spec.Dirs) != 0 || len(spec.Files) != 0 {
			return "takes symbol, in_file, within, drop_callers only"
		}
		if spec.DropCallers < 0 {
			return "drop_callers must not be negative"
		}
	case AnnexPlanPack:
		if spec.Compact {
			return "compact applies to the toc kinds only"
		}
		if len(spec.Use) == 0 && len(spec.Change) == 0 {
			return "needs at least one entry in use or change"
		}
		if len(spec.Dirs) != 0 || len(spec.Files) != 0 || spec.Symbol != "" || spec.InFile != "" || spec.Within != "" {
			return "takes use, change, drop_callers only"
		}
		if spec.DropCallers < 0 {
			return "drop_callers must not be negative"
		}
		for _, ref := range append(append([]SymbolRef{}, spec.Use...), spec.Change...) {
			if ref.Symbol == "" || ref.InFile == "" {
				return "every use/change entry needs symbol and in_file"
			}
		}
		for _, ref := range spec.Use {
			if ref.Within != "" {
				return "within applies to change entries only"
			}
		}
	}
	paths := append(append(append([]string{}, spec.Dirs...), spec.Files...), spec.InFile, spec.Within)
	for _, ref := range append(append([]SymbolRef{}, spec.Use...), spec.Change...) {
		paths = append(paths, ref.InFile, ref.Within)
	}
	for _, p := range paths {
		if filepath.IsAbs(p) {
			return fmt.Sprintf("path %q must be relative to the worktree", p)
		}
	}
	return ""
}

// IsControl reports whether c is its ladder's control: no tools granted and no annex injected. An annex
// config also grants no tools, which is exactly why the two conditions are checked together everywhere
// the harness needs to tell a control from a rung.
func IsControl(c LadderConfig) bool {
	return len(c.Allowed) == 0 && c.Annex == ""
}

// AnnexFor returns the AnnexSpec config selects, or an error when it selects none or an unknown name.
func AnnexFor(l *Ladder, config LadderConfig) (AnnexSpec, error) {
	if config.Annex == "" {
		return AnnexSpec{}, fmt.Errorf("config %q has no annex", config.ID)
	}
	task, ok := l.Tasks[config.Task]
	if !ok {
		return AnnexSpec{}, fmt.Errorf("config %q references unknown task %q", config.ID, config.Task)
	}
	spec, ok := task.Annexes[config.Annex]
	if !ok {
		return AnnexSpec{}, fmt.Errorf("config %q selects annex %q, which task %q does not define", config.ID, config.Annex, config.Task)
	}
	return spec, nil
}

// cliBinaryRelativePath is where BuildCLI puts the quarry CLI: under the gitignored .scratch tree, since
// the repo root already holds the facade package directory named quarry/.
const cliBinaryRelativePath = ".scratch/bin/quarry"

// BuildCLI builds the quarry CLI binary at <repoRoot>/.scratch/bin/quarry with CGO_ENABLED=1 (the toc
// verbs need the tree-sitter grammars), returning its absolute path. Same Builder seam as BuildServer.
func BuildCLI(repoRoot string, build Builder) (string, error) {
	binaryPath := filepath.Join(repoRoot, cliBinaryRelativePath)
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return "", fmt.Errorf("ladder: build cli: create %s: %w", filepath.Dir(binaryPath), err)
	}
	env := append(os.Environ(), "CGO_ENABLED=1")
	output, err := build(repoRoot, env, "go", "build", "-o", binaryPath, "./cmd/quarry")
	if err != nil {
		return "", &HarnessError{Message: fmt.Sprintf(
			"build_cli: go build ./cmd/quarry failed -- requires CGO_ENABLED=1 with a C toolchain:\n%s", output,
		)}
	}
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return "", fmt.Errorf("ladder: build cli: resolve absolute path for %s: %w", binaryPath, err)
	}
	return absPath, nil
}

// CLIRunner runs the quarry CLI binary at bin with args, cwd dir, returning stdout. A non-zero exit is an
// error carrying stderr; stdout and stderr are kept apart because stdout is JSON the generator parses.
type CLIRunner func(bin, dir string, args ...string) (string, error)

// RunCLI is the real CLIRunner: the process environment is ScrubbedEnv, so the CLI sees the same
// QUARRY_* scrub every other quarry process launched by the harness sees.
func RunCLI(bin, dir string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = ScrubbedEnv()
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ladder: annex: %s %s: %w\nstdout: %s\nstderr: %s", filepath.Base(bin), strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

// Annex is one generated attachment: the text that goes into the prompt, and the record of how it was
// made, written beside it so a results root can say exactly what every annex run saw.
type Annex struct {
	// Text is the material as injected: CLI output with the worktree path relativised away.
	Text string `json:"-"`
	// Kind is the spec's kind.
	Kind string `json:"kind"`
	// Label is the neutral description of the material the prompt uses -- never a tool or verb name.
	Label string `json:"label"`
	// Compact records whether the toc material was rendered in the compact text form.
	Compact bool `json:"compact"`
	// Commands is every CLI argument vector run, in order, without the binary.
	Commands [][]string `json:"commands"`
	// Bytes is len(Text).
	Bytes int `json:"bytes"`
	// SHA256 is the hex digest of Text.
	SHA256 string `json:"sha256"`
	// DroppedCallers is how many callers an impact annex deliberately removed (0 for every other kind).
	DroppedCallers int `json:"dropped_callers"`
}

// annexLabels are the neutral names the prompt gives each kind's material.
var annexLabels = map[string]string{
	AnnexTocDir:   "directory listings of the in-scope packages: every source file with its package and header comment",
	AnnexTocFull:  "directory listings of the in-scope packages, followed by a table of contents for every file in them: each declaration with its signature, doc comment, and line range",
	AnnexTocFile:  "a table of contents for the named file(s): each declaration with its signature, doc comment, and line range",
	AnnexImpact:   "a static call-site analysis of the named symbol: its declaration, and every call site that resolves to it, each with the enclosing declaration",
	AnnexPlanPack: "a pack for the change described in the task: the declaration location of each symbol it involves, then, for the symbol being changed, its declaration and every call site that resolves to it, each with the enclosing declaration",
}

// GenerateAnnex runs spec against the worktree with the CLI at bin and returns the Annex. worktree must
// be absolute; every CLI call runs with cwd = worktree and relative arguments, so listing paths come out
// relative, and any absolute worktree path the CLI prints (impact's file fields) is relativised.
func GenerateAnnex(spec AnnexSpec, worktree, bin string, runCLI CLIRunner) (Annex, error) {
	if rule := validateAnnexSpec(spec); rule != "" {
		return Annex{}, &HarnessError{Message: "annex: invalid spec: " + rule}
	}
	if !filepath.IsAbs(worktree) {
		return Annex{}, fmt.Errorf("ladder: annex: worktree %q must be absolute", worktree)
	}

	annex := Annex{Kind: spec.Kind, Label: annexLabels[spec.Kind], Compact: spec.Compact}
	call := func(args ...string) (string, error) {
		annex.Commands = append(annex.Commands, args)
		return runCLI(bin, worktree, args...)
	}

	// tocArgs prefixes a toc verb's arguments with --compact when the spec asks for the compact form.
	tocArgs := func(verb string, paths ...string) []string {
		args := []string{"toc", verb}
		if spec.Compact {
			args = append(args, "--compact")
		}
		return append(args, paths...)
	}

	var parts []string
	switch spec.Kind {
	case AnnexTocDir, AnnexTocFull:
		dirs, err := expandDirs(worktree, spec.Dirs)
		if err != nil {
			return Annex{}, err
		}
		listing, err := call(tocArgs("dir", dirs...)...)
		if err != nil {
			return Annex{}, err
		}
		parts = append(parts, listing)
		if spec.Kind == AnnexTocFull {
			// File discovery always reads the JSON listing; in compact mode that is one extra call
			// whose output is not injected (it is still recorded in Commands).
			jsonListing := listing
			if spec.Compact {
				jsonListing, err = call(append([]string{"toc", "dir"}, dirs...)...)
				if err != nil {
					return Annex{}, err
				}
			}
			files, err := filesListedBy(jsonListing, len(dirs) > 1)
			if err != nil {
				return Annex{}, err
			}
			if len(files) > 0 {
				tocs, err := call(tocArgs("file", files...)...)
				if err != nil {
					return Annex{}, err
				}
				parts = append(parts, tocs)
			}
		}
	case AnnexTocFile:
		tocs, err := call(tocArgs("file", spec.Files...)...)
		if err != nil {
			return Annex{}, err
		}
		parts = append(parts, tocs)
	case AnnexImpact:
		args := []string{"impact", "--in-file", spec.InFile}
		if spec.Within != "" {
			args = append(args, "--within", spec.Within)
		}
		args = append(args, spec.Symbol)
		out, err := call(args...)
		if err != nil {
			return Annex{}, err
		}
		if spec.DropCallers > 0 {
			out, annex.DroppedCallers, err = dropLastCallers(out, spec.DropCallers)
			if err != nil {
				return Annex{}, err
			}
		}
		parts = append(parts, out)
	case AnnexPlanPack:
		for _, group := range groupByFile(spec.Use) {
			out, err := call(append([]string{"definition", "--in-file", group.file}, group.symbols...)...)
			if err != nil {
				return Annex{}, err
			}
			parts = append(parts, fmt.Sprintf("# declaration location of: %s (declared in %s)\n%s", strings.Join(group.symbols, ", "), group.file, out))
		}
		for _, ref := range spec.Change {
			args := []string{"impact", "--in-file", ref.InFile}
			if ref.Within != "" {
				args = append(args, "--within", ref.Within)
			}
			args = append(args, ref.Symbol)
			out, err := call(args...)
			if err != nil {
				return Annex{}, err
			}
			if spec.DropCallers > 0 {
				var dropped int
				out, dropped, err = dropLastCallers(out, spec.DropCallers)
				if err != nil {
					return Annex{}, err
				}
				annex.DroppedCallers += dropped
			}
			parts = append(parts, fmt.Sprintf("# declaration and every call site of: %s (declared in %s)\n%s", ref.Symbol, ref.InFile, out))
		}
	}

	text := strings.TrimSpace(strings.Join(parts, "\n"))
	text = strings.ReplaceAll(text, worktree+string(filepath.Separator), "")
	if leak := blindingLeak(text); leak != "" {
		return Annex{}, &HarnessError{Message: "annex: generated text would break blinding: " + leak}
	}

	sum := sha256.Sum256([]byte(text))
	annex.Text = text
	annex.Bytes = len(text)
	annex.SHA256 = hex.EncodeToString(sum[:])
	return annex, nil
}

// fileGroup is the symbols of one declaring file, in first-seen order, for one batched definition call.
type fileGroup struct {
	file    string
	symbols []string
}

// groupByFile batches refs by InFile, preserving first-seen order of files and of symbols within one.
func groupByFile(refs []SymbolRef) []fileGroup {
	var groups []fileGroup
	index := map[string]int{}
	for _, ref := range refs {
		i, ok := index[ref.InFile]
		if !ok {
			i = len(groups)
			index[ref.InFile] = i
			groups = append(groups, fileGroup{file: ref.InFile})
		}
		groups[i].symbols = append(groups[i].symbols, ref.Symbol)
	}
	return groups
}

// expandDirs resolves each pattern in dirs against worktree, keeping only existing directories, and
// returns them worktree-relative, in the order given with each glob's matches sorted. A pattern that
// matches nothing is an error: a silently empty annex would measure nothing.
func expandDirs(worktree string, dirs []string) ([]string, error) {
	var out []string
	for _, pattern := range dirs {
		matches, err := filepath.Glob(filepath.Join(worktree, pattern))
		if err != nil {
			return nil, fmt.Errorf("ladder: annex: bad dirs pattern %q: %w", pattern, err)
		}
		sort.Strings(matches)
		found := 0
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || !info.IsDir() {
				continue
			}
			rel, err := filepath.Rel(worktree, m)
			if err != nil {
				return nil, err
			}
			out = append(out, rel)
			found++
		}
		if found == 0 {
			return nil, &HarnessError{Message: fmt.Sprintf("annex: dirs entry %q matches no directory under the worktree", pattern)}
		}
	}
	return out, nil
}

// filesListedBy extracts the "path" of every file entry in a `toc dir` listing, skipping entries the
// CLI marked with an error. batch selects the {"results":[...]} envelope over the single-dir shape.
func filesListedBy(listing string, batch bool) ([]string, error) {
	type fileEntry struct {
		Path  string `json:"path"`
		Error string `json:"error"`
	}
	type dirResult struct {
		Files []fileEntry `json:"files"`
	}
	var results []dirResult
	if batch {
		var envelope struct {
			Results []dirResult `json:"results"`
		}
		if err := json.Unmarshal([]byte(listing), &envelope); err != nil {
			return nil, fmt.Errorf("ladder: annex: parse toc dir batch output: %w", err)
		}
		results = envelope.Results
	} else {
		var single dirResult
		if err := json.Unmarshal([]byte(listing), &single); err != nil {
			return nil, fmt.Errorf("ladder: annex: parse toc dir output: %w", err)
		}
		results = []dirResult{single}
	}
	var files []string
	for _, r := range results {
		for _, f := range r.Files {
			if f.Error == "" && f.Path != "" {
				files = append(files, f.Path)
			}
		}
	}
	return files, nil
}

// dropLastCallers removes the last n entries of an impact envelope's "callers" array and re-marshals
// the document, returning it with the count actually removed. Fewer than n callers is an error: a
// degraded annex that ends up empty measures something else.
func dropLastCallers(impactJSON string, n int) (string, int, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(impactJSON), &doc); err != nil {
		return "", 0, fmt.Errorf("ladder: annex: parse impact output: %w", err)
	}
	var callers []json.RawMessage
	if raw, ok := doc["callers"]; ok {
		if err := json.Unmarshal(raw, &callers); err != nil {
			return "", 0, fmt.Errorf("ladder: annex: parse impact callers: %w", err)
		}
	}
	if len(callers) <= n {
		return "", 0, &HarnessError{Message: fmt.Sprintf("annex: drop_callers %d but the impact result has only %d callers", n, len(callers))}
	}
	kept, err := json.Marshal(callers[:len(callers)-n])
	if err != nil {
		return "", 0, err
	}
	doc["callers"] = kept
	out, err := json.Marshal(doc)
	if err != nil {
		return "", 0, err
	}
	return string(out), n, nil
}

// blindingLeak names the first thing in text that would fail the blinding gate, or "".
func blindingLeak(text string) string {
	if strings.Contains(text, MCPPrefix) {
		return "contains an " + MCPPrefix + " tool name"
	}
	if strings.Contains(strings.ToLower(text), "quarry") {
		return `contains the word "quarry"`
	}
	return ""
}

// Filenames the annex is materialised under, in the session scratch directory and again in the run's
// results directory after ingest.
const (
	AnnexTextFilename = "annex.txt"
	AnnexMetaFilename = "annex.meta.json"
)

// WriteAnnex writes annex's text and metadata into dir.
func WriteAnnex(dir string, annex Annex) error {
	if err := os.WriteFile(filepath.Join(dir, AnnexTextFilename), []byte(annex.Text), 0o644); err != nil {
		return fmt.Errorf("ladder: write annex: %w", err)
	}
	return writeJSONDocument(filepath.Join(dir, AnnexMetaFilename), annex)
}

// ReadAnnex returns the annex prepare-session wrote into dir (text and metadata), or an error naming
// the missing file -- next-run must never build an annex config's prompt without it.
func ReadAnnex(dir string) (Annex, error) {
	text, err := os.ReadFile(filepath.Join(dir, AnnexTextFilename))
	if err != nil {
		return Annex{}, fmt.Errorf("ladder: read annex: %w (run prepare-session for this config and rep first)", err)
	}
	meta, err := os.ReadFile(filepath.Join(dir, AnnexMetaFilename))
	if err != nil {
		return Annex{}, fmt.Errorf("ladder: read annex: %w", err)
	}
	var annex Annex
	if err := json.Unmarshal(meta, &annex); err != nil {
		return Annex{}, fmt.Errorf("ladder: read annex: parse %s: %w", AnnexMetaFilename, err)
	}
	annex.Text = string(text)
	return annex, nil
}

// AnnexBlock renders the annex as the prompt paragraph PreambleFor inserts between the control body and
// the task text. Neutral on purpose: it names what the material is, not what produced it, and it grants
// the agent no instruction beyond "start here" -- the same wording for a correct and a degraded annex,
// so a degraded cell measures whether the agent verifies under neutral wording.
func AnnexBlock(targetDir string, annex Annex) string {
	return fmt.Sprintf(`Attached below is a pre-computed analysis of the codebase at %s, generated
mechanically from the code before this session started, at the same commit
you are working on. It is %s.
It is static-analysis output, not an opinion. Use it as your starting point;
you may still read or search the code to confirm or extend it.

--- BEGIN ATTACHMENT ---
%s
--- END ATTACHMENT ---`, targetDir, annex.Label, annex.Text)
}
