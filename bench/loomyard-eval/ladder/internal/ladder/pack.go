// pack.go generates the kick-start pack a pack cell's card carries: RenderKickstartPack turns one
// batched resolve into the two-line-per-target block a card's sentinel-delimited region holds, and
// the sentinel-delimited read/write protocol below is what lets run's pre-rep-1 gate verify the pack
// sitting in the prompt is the pack the provenance record says it is.
//
// The sentinels mark exactly the substitutable region of a pack cell's card: PackBlockSHA256 is the
// one hash function both the writer here and the run-time gate call, so the two can never drift into
// computing the hash two different ways.

package ladder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Knatte18/quarry/quarry"
)

// RenderKickstartPack renders results -- one batched (*quarry.Repo).Resolve call's answer, in the
// slice's own order, which is the ladder file's pack_targets order since the facade answers
// positionally -- into the kick-start pack block: two lines per result,
//
//	<target> → <file> <start>-<end>
//	    <signature>
//
// The first line's fields are the result's own Target verbatim, its single symbol's
// repository-relative file, and that symbol's start and end lines joined by a hyphen. The second
// line is indented by exactly four spaces and carries the symbol's signature with its internal
// newlines collapsed: split on newlines, each resulting line trimmed of its surrounding whitespace,
// then joined with a single space. The rendered block ends with no trailing newline.
//
// The docstring is never emitted, and neither is the signature-end line: this is the pack's
// treatment definition, not a formatting preference. Emitting the docstring would turn "the agent
// knows where things are" into "the agent knows where things are and has a slab of prose", which
// would make a measured win uninterpretable. The existing text-view renderer in quarry/text.go emits
// the docstring, which is exactly why this function exists instead of a call to it.
//
// Any result whose status is not StatusFound, and any result carrying a non-empty pre-resolution
// Error, is fatal: RenderKickstartPack returns an error naming the offending target and what came
// back for it, with no partial output -- a pack missing one glyph is a different treatment from the
// one the cards describe. A found result carries exactly one symbol; a found result with no symbols
// is the same class of fatal error, checked explicitly rather than indexing into an empty slice.
func RenderKickstartPack(results []quarry.ResolveResult) (string, error) {
	lines := make([]string, 0, len(results)*2)
	for _, r := range results {
		if r.Error != "" {
			return "", fmt.Errorf("render kickstart pack: target %q: pre-resolution error: %s", r.Target, r.Error)
		}
		if r.Status != quarry.StatusFound {
			return "", fmt.Errorf("render kickstart pack: target %q: status %q, want %q", r.Target, r.Status, quarry.StatusFound)
		}
		if len(r.Symbols) != 1 {
			return "", fmt.Errorf("render kickstart pack: target %q: found result carries %d symbols, want exactly 1", r.Target, len(r.Symbols))
		}
		sym := r.Symbols[0]
		lines = append(lines, fmt.Sprintf("%s → %s %d-%d", r.Target, sym.File, sym.Start, sym.End))
		lines = append(lines, "    "+collapseSignature(sym.Signature))
	}
	return strings.Join(lines, "\n"), nil
}

// collapseSignature splits sig on newlines, trims each resulting line's surrounding whitespace, and
// joins the result with a single space -- the treatment RenderKickstartPack's doc comment describes
// for a multi-line signature.
func collapseSignature(sig string) string {
	parts := strings.Split(sig, "\n")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return strings.Join(parts, " ")
}

// PackSentinelBegin marks the start of a pack cell's card's substitutable region. It is spelled as a
// markdown comment line so it renders invisibly in a rendered card while remaining a literal,
// greppable line in the prompt text handed to the model. It must appear on its own line, exactly
// once, before PackSentinelEnd.
const PackSentinelBegin = "<!-- KICKSTART-PACK:BEGIN -->"

// PackSentinelEnd marks the end of a pack cell's card's substitutable region, for the same reason and
// under the same one-line, exactly-once, after-PackSentinelBegin rule PackSentinelBegin's doc comment
// states.
const PackSentinelEnd = "<!-- KICKSTART-PACK:END -->"

// ExtractPackBlock returns the text strictly between cardText's two sentinel lines -- neither
// sentinel line included -- with leading and trailing blank lines removed and no trailing newline. A
// card missing either sentinel, carrying either more than once, or carrying them in the wrong order,
// is an error naming which condition failed. A card whose block is empty is not an error: that is the
// state an authored card starts in, before the pack is generated into it.
func ExtractPackBlock(cardText string) (string, error) {
	lines := strings.Split(cardText, "\n")
	beginIdx, endIdx, err := findPackSentinels(lines)
	if err != nil {
		return "", fmt.Errorf("extract pack block: %w", err)
	}
	return trimBlankLines(lines[beginIdx+1 : endIdx]), nil
}

// WritePackIntoCard returns cardText with the region between its two sentinels replaced by pack. It
// preserves everything outside the sentinels byte for byte, and it is idempotent: writing the same
// pack twice yields the same text. A card missing its sentinels is an error, never a silent append --
// appending would produce a card whose prompt text disagrees with the hash provenance records, which
// is the one failure this whole mechanism exists to make impossible.
func WritePackIntoCard(cardText, pack string) (string, error) {
	lines := strings.Split(cardText, "\n")
	beginIdx, endIdx, err := findPackSentinels(lines)
	if err != nil {
		return "", fmt.Errorf("write pack into card: %w", err)
	}
	newLines := make([]string, 0, beginIdx+1+1+len(lines)-endIdx)
	newLines = append(newLines, lines[:beginIdx+1]...)
	newLines = append(newLines, pack)
	newLines = append(newLines, lines[endIdx:]...)
	return strings.Join(newLines, "\n"), nil
}

// PackBlockSHA256 returns the hex sha256 of block's bytes, computed by calling the package's existing
// sha256Hex helper rather than by spelling the same computation a second time. Both WritePackIntoCard's
// caller and the run-time gate call this one function, so the two cannot drift.
func PackBlockSHA256(block string) string {
	return sha256Hex(block)
}

// findPackSentinels scans lines for exactly one PackSentinelBegin line and exactly one
// PackSentinelEnd line, in that order, and returns their indices. It returns an error naming which
// condition failed -- missing, duplicated, or out of order -- otherwise.
func findPackSentinels(lines []string) (beginIdx, endIdx int, err error) {
	beginIdx, endIdx = -1, -1
	beginCount, endCount := 0, 0
	for i, line := range lines {
		switch line {
		case PackSentinelBegin:
			beginCount++
			beginIdx = i
		case PackSentinelEnd:
			endCount++
			endIdx = i
		}
	}
	switch {
	case beginCount == 0:
		return 0, 0, fmt.Errorf("missing sentinel %q", PackSentinelBegin)
	case endCount == 0:
		return 0, 0, fmt.Errorf("missing sentinel %q", PackSentinelEnd)
	case beginCount > 1:
		return 0, 0, fmt.Errorf("sentinel %q appears more than once", PackSentinelBegin)
	case endCount > 1:
		return 0, 0, fmt.Errorf("sentinel %q appears more than once", PackSentinelEnd)
	case beginIdx > endIdx:
		return 0, 0, fmt.Errorf("sentinel %q appears after %q", PackSentinelBegin, PackSentinelEnd)
	}
	return beginIdx, endIdx, nil
}

// trimBlankLines joins lines with "\n" after dropping leading and trailing lines that are empty or
// hold only whitespace.
func trimBlankLines(lines []string) string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}

// PackResolveFile is the file name a results root's pack-time resolve output is written under, under
// the results root -- the sibling of ProvenanceFile card 28's summary file and card 29's table file
// each declare their own constant for, in the file that writes them.
const PackResolveFile = "pack-resolve.json"

// PackOptions carries everything Pack needs to drive one pack-generation invocation: the operator's
// own ladder file path and results root, the claude binary path, a starting point Pack resolves up to
// the quarry repository root from, and the Runner every external process goes through. It mirrors
// RunOptions's own field set minus SelectedCells and RepsOverride, which have no meaning here: Pack
// always resolves the file's whole glyph list against its one pack cell, never a caller-chosen subset
// or repetition count.
type PackOptions struct {
	// LadderFilePath is the operator's own ladder file path.
	LadderFilePath string
	// ResultsRoot is the results root Pack writes pack-resolve.json and the provenance record to.
	ResultsRoot string
	// ClaudeBinPath is the claude binary path CollectInvocation probes for its version. Pack takes
	// this as an option, rather than a constant, because it must be able to name the same binary the
	// run this pack precedes will use.
	ClaudeBinPath string
	// QuarryRepoStart is a path inside the quarry repository Pack resolves up from to the
	// repository's own root via ResolveQuarryRepoRoot.
	QuarryRepoStart string
	// Runner is the seam every external process -- claude, git -- runs through.
	Runner Runner
}

// Pack generates one ladder file's kick-start pack: it loads the file, finds its single pack cell,
// prepares that cell's pinned worktree, makes exactly one batched (*quarry.Repo).Resolve call over
// the file's whole glyph list through the facade, renders the pack, writes it into the pack cell's
// card between its sentinels, records the resolve output and a provenance invocation carrying the
// pack's own kickstart_pack block, and returns.
//
// Pack acquires the same advisory run lock Run does, against the same worktree root and results root,
// with the release deferred, so a pack and a run can never touch one pinned worktree concurrently. The
// lock is exclusive-create and is never reaped automatically, so a pack that dies leaves the same
// operator-cleared stale lock a dead run does -- the existing, documented behaviour, and not something
// this command works around.
//
// Pack writes a real, complete invocation -- naming every config id in the file as its selected
// cells and the file's own reps as its effective repetition count -- rather than a pack-only stub that
// leaves those at zero, because MergeProvenance refuses a record whose selected cells or effective
// repetition count differs from the existing one, and Run additionally refuses a root whose effective
// repetition count differs from its own. A stub would fail both checks on the very first run, before
// rep 1, every time.
//
// This carries two consequences, accepted deliberately: the record carries one invocation that ran no
// repetitions, and the effective repetition count is pinned from the pack onward, so a later per-run
// repetition override against the same root is refused -- which is the correct behaviour under a
// locked n, not a limitation to work around.
//
// Pack does not restore the pinned worktree when it finishes. It only reads through the facade, so it
// dirties nothing, and restoring would discard state that some other holder of the worktree put there
// rather than state Pack caused. The run lock, held for the whole command, is what keeps a pack and a
// run off one pinned worktree at the same time.
func Pack(ctx context.Context, opts PackOptions) error {
	l, err := LoadLadder(opts.LadderFilePath)
	if err != nil {
		return err
	}

	quarryRepoRoot, err := ResolveQuarryRepoRoot(opts.QuarryRepoStart)
	if err != nil {
		return err
	}
	targetRepoPath, err := ResolveLoomyardRepo(quarryRepoRoot)
	if err != nil {
		return err
	}
	worktreeRoot, err := ResolveWorktreeRoot(quarryRepoRoot)
	if err != nil {
		return err
	}

	release, err := AcquireRunLock(worktreeRoot, opts.ResultsRoot)
	if err != nil {
		return err
	}
	defer func() { _ = release() }()

	packCfg, err := findPackConfig(l, opts.LadderFilePath)
	if err != nil {
		return err
	}

	task, ok := l.Tasks[packCfg.Task]
	if !ok {
		return fmt.Errorf("pack: cell %s references unknown task %q", packCfg.ID, packCfg.Task)
	}
	dest := TaskWorktreePath(worktreeRoot, packCfg.Task)
	if err := PrepareWorktree(ctx, opts.Runner, targetRepoPath, packCfg.Task, task.PinnedSHA, dest); err != nil {
		return err
	}

	selectedIDs := make([]string, len(l.Configs))
	for i, c := range l.Configs {
		selectedIDs[i] = c.ID
	}
	inv, err := CollectInvocation(ctx, opts.Runner, CollectInput{
		QuarryRepoRoot: quarryRepoRoot,
		LadderFilePath: opts.LadderFilePath,
		TargetRepoPath: targetRepoPath,
		ServerName:     l.ServerName(),
		SelectedCells:  selectedIDs,
		RepsEffective:  l.Reps,
		ClaudeBinPath:  opts.ClaudeBinPath,
	})
	if err != nil {
		return err
	}

	repo, err := quarry.Open(dest)
	if err != nil {
		return fmt.Errorf("pack: open pinned worktree %s: %w", dest, err)
	}
	results, err := repo.Resolve(l.PackTargets)
	if err != nil {
		return fmt.Errorf("pack: resolve pack targets: %w", err)
	}

	pack, err := RenderKickstartPack(results)
	if err != nil {
		return err
	}

	cardPath := resolveRepoRelative(quarryRepoRoot, packCfg.Card)
	cardData, err := os.ReadFile(cardPath)
	if err != nil {
		return fmt.Errorf("pack: read card %s: %w", cardPath, err)
	}
	newCard, err := WritePackIntoCard(string(cardData), pack)
	if err != nil {
		return fmt.Errorf("pack: write pack into card %s: %w", cardPath, err)
	}
	if err := os.WriteFile(cardPath, []byte(newCard), 0o644); err != nil {
		return fmt.Errorf("pack: write card %s: %w", cardPath, err)
	}

	resolveData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("pack: marshal resolve results: %w", err)
	}
	if err := os.MkdirAll(opts.ResultsRoot, 0o755); err != nil {
		return fmt.Errorf("pack: create results root %s: %w", opts.ResultsRoot, err)
	}
	resolvePath := filepath.Join(opts.ResultsRoot, PackResolveFile)
	if err := os.WriteFile(resolvePath, resolveData, 0o644); err != nil {
		return fmt.Errorf("pack: write %s: %w", resolvePath, err)
	}

	existing, err := ReadProvenance(opts.ResultsRoot)
	if err != nil {
		return err
	}
	prov, err := MergeProvenance(existing, inv)
	if err != nil {
		return err
	}
	prov.KickstartPack = &KickstartPack{
		GeneratedAt:    inv.WrittenAt,
		QuarryCommit:   inv.QuarryCommit,
		QuarryDirty:    inv.QuarryDirty,
		LoomyardCommit: inv.LoomyardCommit,
		Targets:        append([]string(nil), l.PackTargets...),
		PackSHA256:     PackBlockSHA256(pack),
		ResolveSHA256:  sha256Hex(string(resolveData)),
		CardFile:       packCfg.Card,
	}
	if err := WriteProvenance(opts.ResultsRoot, prov); err != nil {
		return err
	}

	return nil
}

// findPackConfig returns the single config in l whose Pack flag is set, erroring naming
// ladderFilePath when there is none. validate guarantees at most one such config exists.
func findPackConfig(l *Ladder, ladderFilePath string) (Config, error) {
	for _, c := range l.Configs {
		if c.Pack {
			return c, nil
		}
	}
	return Config{}, fmt.Errorf("pack: ladder file %s declares no config with pack: true", ladderFilePath)
}
