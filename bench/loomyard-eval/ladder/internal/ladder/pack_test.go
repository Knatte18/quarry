// pack_test.go covers pack.go's three surfaces: RenderKickstartPack, the pure renderer that turns a
// batched resolve into the kick-start pack block; the sentinel-delimited read/write protocol,
// ExtractPackBlock/WritePackIntoCard/PackBlockSHA256, that puts a rendered pack into a card; and
// TestPack_*, which drives the Pack entry point itself against a real synthetic repository and a
// real pinned worktree through the production runner, reusing e2e_test.go's own fixture helpers --
// the synthetic quarry repository, the synthetic target repository, the worktree root and the fake
// claude binary.

package ladder

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// foundResult builds a fabricated found *quarry.ResolveResult carrying exactly one symbol, for
// driving RenderKickstartPack without a real repository -- the renderer is pure.
func foundResult(target, file string, start, end int, signature string) quarry.ResolveResult {
	return quarry.ResolveResult{
		Target: target,
		Status: quarry.StatusFound,
		Symbols: []quarry.Symbol{{
			ID:        target,
			Kind:      quarry.KindFunction,
			File:      file,
			Start:     start,
			End:       end,
			Signature: signature,
		}},
	}
}

func TestRenderKickstartPack_GoldenBlock(t *testing.T) {
	results := []quarry.ResolveResult{
		foundResult("pkg.Foo", "pkg/foo.go", 1, 3, "func Foo()"),
		foundResult("pkg.Bar", "pkg/bar.go", 5, 8, "func Bar(\n\ta int,\n) error"),
		foundResult("pkg.Baz", "pkg/baz.go", 10, 10, "const Baz = 1"),
	}
	want := strings.Join([]string{
		"pkg.Foo → pkg/foo.go 1-3",
		"    func Foo()",
		"pkg.Bar → pkg/bar.go 5-8",
		"    func Bar( a int, ) error",
		"pkg.Baz → pkg/baz.go 10-10",
		"    const Baz = 1",
	}, "\n")

	got, err := RenderKickstartPack(results)
	if err != nil {
		t.Fatalf("RenderKickstartPack() = %v; want no error", err)
	}
	if got != want {
		t.Errorf("RenderKickstartPack() = %q; want %q", got, want)
	}
}

func TestRenderKickstartPack_FatalStatuses(t *testing.T) {
	tests := []struct {
		name   string
		bad    quarry.ResolveResult
		target string
	}{
		{
			name:   "NotFound",
			bad:    quarry.ResolveResult{Target: "x.NotFound", Status: quarry.StatusNotFound},
			target: "x.NotFound",
		},
		{
			name:   "Ambiguous",
			bad:    quarry.ResolveResult{Target: "x.Amb", Status: quarry.StatusAmbiguous},
			target: "x.Amb",
		},
		{
			name:   "Multipart",
			bad:    quarry.ResolveResult{Target: "x.Multi", Status: quarry.StatusMultipart},
			target: "x.Multi",
		},
		{
			name:   "PreResolutionError",
			bad:    quarry.ResolveResult{Target: "x.Bad", Error: "not a valid glyph"},
			target: "x.Bad",
		},
		{
			name:   "FoundWithNoSymbols",
			bad:    quarry.ResolveResult{Target: "x.Empty", Status: quarry.StatusFound, Symbols: nil},
			target: "x.Empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := []quarry.ResolveResult{
				foundResult("pkg.Good", "pkg/good.go", 1, 2, "func Good()"),
				tt.bad,
			}
			got, err := RenderKickstartPack(results)
			if err == nil {
				t.Fatalf("RenderKickstartPack() = nil error; want one naming %q", tt.target)
			}
			if !strings.Contains(err.Error(), tt.target) {
				t.Errorf("RenderKickstartPack() error = %q; want it to name %q", err, tt.target)
			}
			if got != "" {
				t.Errorf("RenderKickstartPack() = %q; want no partial output on a fatal input", got)
			}
		})
	}
}

// cardWithSentinels builds a card carrying inner between the two pack sentinels, with surrounding
// prose before and after, for driving the sentinel-delimited protocol's happy path.
func cardWithSentinels(inner string) string {
	return strings.Join([]string{
		"# Title",
		"",
		"Some text before.",
		"",
		PackSentinelBegin,
		inner,
		PackSentinelEnd,
		"",
		"Some text after.",
		"",
	}, "\n")
}

func TestPackBlockRoundTrip(t *testing.T) {
	card := cardWithSentinels("")
	pack := "line one\nline two"

	written, err := WritePackIntoCard(card, pack)
	if err != nil {
		t.Fatalf("WritePackIntoCard() = %v; want no error", err)
	}

	if !strings.HasPrefix(written, "# Title\n\nSome text before.\n\n"+PackSentinelBegin+"\n") {
		t.Errorf("WritePackIntoCard() = %q; want the text before the sentinel preserved byte for byte", written)
	}
	if !strings.HasSuffix(written, "\n"+PackSentinelEnd+"\n\nSome text after.\n") {
		t.Errorf("WritePackIntoCard() = %q; want the text after the sentinel preserved byte for byte", written)
	}

	extracted, err := ExtractPackBlock(written)
	if err != nil {
		t.Fatalf("ExtractPackBlock() = %v; want no error", err)
	}
	if extracted != pack {
		t.Errorf("ExtractPackBlock() = %q; want exactly what was written, %q", extracted, pack)
	}

	writtenAgain, err := WritePackIntoCard(written, pack)
	if err != nil {
		t.Fatalf("second WritePackIntoCard() = %v; want no error", err)
	}
	if writtenAgain != written {
		t.Errorf("writing the same pack twice did not yield the same file:\nfirst:  %q\nsecond: %q", written, writtenAgain)
	}

	emptyExtracted, err := ExtractPackBlock(card)
	if err != nil {
		t.Fatalf("ExtractPackBlock(empty block) = %v; want no error", err)
	}
	if emptyExtracted != "" {
		t.Errorf("ExtractPackBlock(empty block) = %q; want the empty string", emptyExtracted)
	}
}

func TestPackBlockErrors(t *testing.T) {
	tests := []struct {
		name string
		card string
	}{
		{
			name: "NoSentinels",
			card: "no sentinels at all\njust text",
		},
		{
			name: "OnlyBegin",
			card: PackSentinelBegin + "\nonly begin",
		},
		{
			name: "OnlyEnd",
			card: "only end\n" + PackSentinelEnd,
		},
		{
			name: "SentinelTwice",
			card: PackSentinelBegin + "\n" + PackSentinelBegin + "\n" + PackSentinelEnd,
		},
		{
			name: "WrongOrder",
			card: PackSentinelEnd + "\n" + PackSentinelBegin,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ExtractPackBlock(tt.card); err == nil {
				t.Error("ExtractPackBlock() = nil error; want one rejecting the malformed sentinels")
			}
			if _, err := WritePackIntoCard(tt.card, "some pack"); err == nil {
				t.Error("WritePackIntoCard() = nil error; want one rejecting the malformed sentinels, not a silent append")
			}
		})
	}
}

func TestPackBlockSHA256_MatchesWrittenBlock(t *testing.T) {
	card := cardWithSentinels("")
	pack := "the pack block's own content"

	written, err := WritePackIntoCard(card, pack)
	if err != nil {
		t.Fatalf("WritePackIntoCard() = %v; want no error", err)
	}
	extracted, err := ExtractPackBlock(written)
	if err != nil {
		t.Fatalf("ExtractPackBlock() = %v; want no error", err)
	}

	got := PackBlockSHA256(extracted)
	want := PackBlockSHA256(pack)
	if got != want {
		t.Errorf("PackBlockSHA256(extracted) = %s; want PackBlockSHA256(written) = %s", got, want)
	}
}

// packFakeClaudeOnce and packFakeClaudeBin cache the one build of testdata/fakeclaude this file's
// three TestPack_* top-level tests share. Each is its own top-level test, so none of them can lend
// the shared binary its own t.TempDir()'s lifetime the way TestE2E's own buildFakeClaudeOnce call
// does: the binary lives in a directory of its own instead, deliberately outside testing's
// TempDir cleanup, so whichever TestPack_* test builds it first does not invalidate the path for
// the ones that run after it.
var (
	packFakeClaudeOnce sync.Once
	packFakeClaudeBin  string
)

// buildPackFakeClaude builds testdata/fakeclaude the first time any TestPack_* subtest calls it and
// returns that same path on every call thereafter -- see packFakeClaudeOnce's own comment for why
// this cannot simply delegate to e2e_test.go's buildFakeClaudeOnce.
func buildPackFakeClaude(t *testing.T) string {
	t.Helper()
	packFakeClaudeOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ladder-pack-fakeclaude-")
		if err != nil {
			t.Fatalf("mkdir temp dir for fake claude: %v", err)
		}
		packFakeClaudeBin = filepath.Join(dir, "fakeclaude")
		cmd := exec.Command("go", "build", "-o", packFakeClaudeBin, "./testdata/fakeclaude")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build ./testdata/fakeclaude: %v\n%s", err, out)
		}
	})
	if packFakeClaudeBin == "" {
		t.Fatal("fake claude binary was never built")
	}
	return packFakeClaudeBin
}

// writePackFixturePackage writes a small Go package with two exported declarations under
// targetRepoRoot/pkg. A repository root-level file's glyph unit is unspellable, so the package this
// pack fixture resolves against lives one directory down, mirroring
// TestRepoTOC_GoFileWithSymbols's own fixture placement in the engine package.
func writePackFixturePackage(t *testing.T, targetRepoRoot string) {
	t.Helper()
	dir := filepath.Join(targetRepoRoot, "pkg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	src := "package pkg\n\n" +
		"// Alpha does the first thing.\n" +
		"func Alpha() int {\n\treturn 1\n}\n\n" +
		"// Beta does the second thing.\n" +
		"func Beta() int {\n\treturn 2\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "pkg.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write pkg.go: %v", err)
	}
}

// newPackE2EEnv wires up a synthetic quarry repository, a synthetic target repository holding
// writePackFixturePackage's own package, a worktree root and a task/fasit pair -- everything
// TestPack_* needs except the results root, which each test creates fresh so a call against it
// exercises the provenance mkdir fix from a second call site. It returns the environment alongside
// the pinned commit that carries the package files.
func newPackE2EEnv(t *testing.T, claudeBinPath string) (*e2eEnv, string) {
	t.Helper()

	quarryRepoRoot := initGitRepo(t)
	commitReadme(t, quarryRepoRoot)

	targetRepoPath := initGitRepo(t)
	writePackFixturePackage(t, targetRepoPath)
	pinnedSHA := commitReadme(t, targetRepoPath)

	worktreeRoot := t.TempDir()
	t.Setenv("LADDER_WORKTREE_ROOT", worktreeRoot)
	t.Setenv("LADDER_LOOMYARD_REPO", targetRepoPath)

	taskDir := t.TempDir()
	taskFilePath := filepath.Join(taskDir, "task.md")
	syntheticTaskFile(t, taskFilePath)
	fasitPath := filepath.Join(taskDir, "fasit.json")
	syntheticFasit(t, fasitPath)

	return &e2eEnv{
		quarryRepoRoot: quarryRepoRoot,
		targetRepoPath: targetRepoPath,
		worktreeRoot:   worktreeRoot,
		claudeBinPath:  claudeBinPath,
		taskFilePath:   taskFilePath,
		fasitPath:      fasitPath,
	}, pinnedSHA
}

// packLadderFile builds a one-cell, one-pack-target-pair ladder file against env and pinnedSHA,
// writing its pack cell's card under env.quarryRepoRoot so the card's own repository-relative path
// is a real relative string rather than an operator's absolute path. It returns the ladder file's
// own path and the card's repository-relative path.
func packLadderFile(t *testing.T, env *e2eEnv, pinnedSHA string) (ladderPath, cardRelPath string) {
	t.Helper()

	l := baseLadder(env, pinnedSHA, 1, "pack-task")
	cardRelPath = "card.md"
	cardPath := filepath.Join(env.quarryRepoRoot, cardRelPath)
	if err := os.WriteFile(cardPath, []byte(cardWithSentinels("")), 0o644); err != nil {
		t.Fatalf("write card: %v", err)
	}
	l.PackTargets = []string{"pkg#Alpha", "pkg#Beta"}
	l.Configs = []Config{{ID: "pack-cell", Ladder: "k", Task: "pack-task", Allowed: nil, Card: cardRelPath, Pack: true}}

	ladderPath = filepath.Join(t.TempDir(), "ladder.yaml")
	writeSyntheticLadderFile(t, ladderPath, l)
	return ladderPath, cardRelPath
}

// packOpts builds the PackOptions common to this file's TestPack_* subtests.
func packOpts(env *e2eEnv, ladderFilePath, resultsRoot string) PackOptions {
	return PackOptions{
		LadderFilePath:  ladderFilePath,
		ResultsRoot:     resultsRoot,
		ClaudeBinPath:   env.claudeBinPath,
		QuarryRepoStart: env.quarryRepoRoot,
		Runner:          ExecRunner{},
	}
}

// assertNoPackLockFile asserts that env's worktree root carries no advisory lock file -- the
// release half TestPack_EndToEnd and TestPack_UnresolvableGlyphIsFatal each assert inline, since a
// pack that leaves its lock behind would deadlock every run and pack against the same worktree root
// afterwards.
func assertNoPackLockFile(t *testing.T, env *e2eEnv) {
	t.Helper()
	worktreeRoot, err := ResolveWorktreeRoot(env.quarryRepoRoot)
	if err != nil {
		t.Fatalf("ResolveWorktreeRoot() = %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktreeRoot, lockFileName)); !os.IsNotExist(err) {
		t.Errorf("lock file still exists under %s after Pack() returned: %v", worktreeRoot, err)
	}
}

func TestPack_EndToEnd(t *testing.T) {
	fakeBinPath := buildPackFakeClaude(t)
	env, pinnedSHA := newPackE2EEnv(t, fakeBinPath)
	ladderPath, cardRelPath := packLadderFile(t, env, pinnedSHA)

	resultsRoot := filepath.Join(t.TempDir(), "root")
	if _, err := os.Stat(resultsRoot); !os.IsNotExist(err) {
		t.Fatalf("results root %s already exists before Pack(); want it fresh", resultsRoot)
	}

	if err := Pack(context.Background(), packOpts(env, ladderPath, resultsRoot)); err != nil {
		t.Fatalf("Pack() = %v; want no error", err)
	}
	assertNoPackLockFile(t, env)

	cardPath := filepath.Join(env.quarryRepoRoot, cardRelPath)
	cardData, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read card %s: %v", cardPath, err)
	}
	block, err := ExtractPackBlock(string(cardData))
	if err != nil {
		t.Fatalf("ExtractPackBlock() = %v; want no error", err)
	}
	if !strings.Contains(block, "pkg#Alpha") || !strings.Contains(block, "pkg#Beta") {
		t.Errorf("pack block = %q; want both glyphs rendered between the sentinels", block)
	}

	resolvePath := filepath.Join(resultsRoot, PackResolveFile)
	if _, err := os.Stat(resolvePath); err != nil {
		t.Errorf("resolve json %s was not written: %v", resolvePath, err)
	}

	prov, err := ReadProvenance(resultsRoot)
	if err != nil {
		t.Fatalf("ReadProvenance() = %v; want no error", err)
	}
	if prov == nil {
		t.Fatal("ReadProvenance() = nil; want a provenance record")
	}
	if prov.KickstartPack == nil {
		t.Fatal("provenance carries no kickstart_pack block")
	}
	if want := PackBlockSHA256(block); prov.KickstartPack.PackSHA256 != want {
		t.Errorf("KickstartPack.PackSHA256 = %s; want %s (the hash of the card's extracted block)", prov.KickstartPack.PackSHA256, want)
	}
	wantTargets := []string{"pkg#Alpha", "pkg#Beta"}
	if !stringSlicesEqual(prov.KickstartPack.Targets, wantTargets) {
		t.Errorf("KickstartPack.Targets = %v; want %v", prov.KickstartPack.Targets, wantTargets)
	}
	if prov.KickstartPack.CardFile != cardRelPath {
		t.Errorf("KickstartPack.CardFile = %q; want the ladder file's own repository-relative card value %q", prov.KickstartPack.CardFile, cardRelPath)
	}
	if len(prov.Invocations) != 1 {
		t.Errorf("len(prov.Invocations) = %d; want exactly 1", len(prov.Invocations))
	}
}

func TestPack_UnresolvableGlyphIsFatal(t *testing.T) {
	fakeBinPath := buildPackFakeClaude(t)
	env, pinnedSHA := newPackE2EEnv(t, fakeBinPath)
	ladderPath, cardRelPath := packLadderFile(t, env, pinnedSHA)

	l, err := LoadLadder(ladderPath)
	if err != nil {
		t.Fatalf("LoadLadder() = %v; want no error", err)
	}
	l.PackTargets = append(l.PackTargets, "pkg#Gamma")
	writeSyntheticLadderFile(t, ladderPath, l)

	cardPath := filepath.Join(env.quarryRepoRoot, cardRelPath)
	before, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read card %s: %v", cardPath, err)
	}

	resultsRoot := filepath.Join(t.TempDir(), "root")
	err = Pack(context.Background(), packOpts(env, ladderPath, resultsRoot))
	if err == nil {
		t.Fatal("Pack() = nil error; want one naming the unresolvable glyph")
	}
	if !strings.Contains(err.Error(), "pkg#Gamma") {
		t.Errorf("Pack() error = %q; want it to name pkg#Gamma", err)
	}
	assertNoPackLockFile(t, env)

	after, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read card %s: %v", cardPath, err)
	}
	if string(after) != string(before) {
		t.Error("the card was modified despite Pack() failing; want it left byte for byte unchanged")
	}

	prov, err := ReadProvenance(resultsRoot)
	if err != nil {
		t.Fatalf("ReadProvenance() = %v; want no error", err)
	}
	if prov != nil {
		t.Errorf("provenance record %+v was written despite Pack() failing; want none", prov)
	}
}

func TestPack_LockHeld(t *testing.T) {
	fakeBinPath := buildPackFakeClaude(t)
	env, pinnedSHA := newPackE2EEnv(t, fakeBinPath)
	ladderPath, cardRelPath := packLadderFile(t, env, pinnedSHA)
	cardPath := filepath.Join(env.quarryRepoRoot, cardRelPath)
	before, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read card %s: %v", cardPath, err)
	}

	worktreeRoot, err := ResolveWorktreeRoot(env.quarryRepoRoot)
	if err != nil {
		t.Fatalf("ResolveWorktreeRoot() = %v", err)
	}
	release, err := AcquireRunLock(worktreeRoot, "/first/holder/results")
	if err != nil {
		t.Fatalf("AcquireRunLock() = %v; want no error", err)
	}
	t.Cleanup(func() { _ = release() })

	resultsRoot := filepath.Join(t.TempDir(), "root")
	err = Pack(context.Background(), packOpts(env, ladderPath, resultsRoot))
	if err == nil {
		t.Fatal("Pack() against an already-locked worktree root = nil error; want one naming the first holder")
	}
	pid := strconv.Itoa(os.Getpid())
	if !strings.Contains(err.Error(), "pid "+pid) {
		t.Errorf("Pack() error = %q; want it to carry pid %s", err, pid)
	}
	if !strings.Contains(err.Error(), "/first/holder/results") {
		t.Errorf("Pack() error = %q; want it to carry the first holder's own results root", err)
	}

	after, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read card %s: %v", cardPath, err)
	}
	if string(after) != string(before) {
		t.Error("the card was modified despite the lock being held; want Pack() to fail before touching it")
	}
}
