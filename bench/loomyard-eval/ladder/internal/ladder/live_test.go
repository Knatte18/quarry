// live_test.go is the one test in this package that talks to the real claude CLI. It is guarded by
// the environment variable LADDER_LIVE_TEST and skips immediately when that variable is not set to
// "1", so the offline suite (everything else in this package) stays free, deterministic and network-
// free. When enabled, it runs one repetition of the migrated ladder file's control cell in a freshly
// created worktree and asserts three things no offline test can: that the advertised tool list is
// exactly the four built-ins for a control cell, that no MCP server loaded, and that a granted tool
// actually executes from that fresh directory rather than being silently denied.

package ladder

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// liveTestClaudeBinEnv names the environment variable this test reads for the claude binary path,
// defaulting to "claude" -- the same default main.go's --claude-bin flag uses.
const liveTestClaudeBinEnv = "LADDER_LIVE_CLAUDE_BIN"

// readOnlyBashCommandPattern matches a leading command word this test considers unambiguously
// read-only, so a permission denial naming one of them is unambiguous evidence of a degraded Bash
// grant rather than a legitimate refusal of something mutating.
var readOnlyBashCommandPattern = regexp.MustCompile(
	`^(ls|cat|grep|rg|find|head|tail|pwd|echo|wc|git\s+(status|log|diff|show)|go\s+(build|vet|test))\b`,
)

// permissionDenialProbe is the minimal shape this test decodes out of one raw permission-denial
// entry: the tool name and, for a Bash denial, the attempted command.
type permissionDenialProbe struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

func TestLive_FreshWorktreeGrantsExactlyBuiltins(t *testing.T) {
	if os.Getenv("LADDER_LIVE_TEST") != "1" {
		t.Skip("set LADDER_LIVE_TEST=1 to run the guarded live smoke test against the real claude CLI")
	}

	ctx := context.Background()
	r := ExecRunner{}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() = %v", err)
	}
	quarryRepoRoot, err := ResolveQuarryRepoRoot(cwd)
	if err != nil {
		t.Fatalf("ResolveQuarryRepoRoot() = %v", err)
	}

	ladderFilePath := filepath.Join(quarryRepoRoot, "bench", "loomyard-eval", "ladder", "ladder-toc.yaml")
	l, err := LoadLadder(ladderFilePath)
	if err != nil {
		t.Fatalf("LoadLadder(%s) = %v", ladderFilePath, err)
	}
	cfg, ok := l.ConfigByID("a0-none")
	if !ok {
		t.Fatalf("ladder file %s declares no config %q", ladderFilePath, "a0-none")
	}
	task, ok := l.Tasks[cfg.Task]
	if !ok {
		t.Fatalf("ladder file %s declares no task %q", ladderFilePath, cfg.Task)
	}

	targetRepoPath, err := ResolveLoomyardRepo(quarryRepoRoot)
	if err != nil {
		t.Fatalf("ResolveLoomyardRepo() = %v", err)
	}
	worktreeRoot, err := ResolveWorktreeRoot(quarryRepoRoot)
	if err != nil {
		t.Fatalf("ResolveWorktreeRoot() = %v", err)
	}

	dest := TaskWorktreePath(worktreeRoot, cfg.Task)
	removeExistingWorktree(t, ctx, r, targetRepoPath, dest)

	if err := PrepareWorktree(ctx, r, targetRepoPath, cfg.Task, task.PinnedSHA, dest); err != nil {
		t.Fatalf("PrepareWorktree() = %v", err)
	}
	t.Cleanup(func() {
		_ = RestoreWorktree(ctx, r, dest)
	})

	content, err := LoadTaskFile(resolveRepoRelative(quarryRepoRoot, task.TaskFile))
	if err != nil {
		t.Fatalf("LoadTaskFile() = %v", err)
	}
	toolNames := grantedToolNames(l, cfg)
	prompt := RenderPrompt(content, dest, toolNames)

	mcpDoc, err := MCPConfigDocument(l, cfg, "", dest)
	if err != nil {
		t.Fatalf("MCPConfigDocument() = %v", err)
	}
	mcpConfigPath, err := WriteMCPConfig(quarryRepoRoot, "live-a0-none.json", mcpDoc)
	if err != nil {
		t.Fatalf("WriteMCPConfig() = %v", err)
	}

	claudeBin := os.Getenv(liveTestClaudeBinEnv)
	if claudeBin == "" {
		claudeBin = "claude"
	}
	opts := RunOptions{ClaudeBinPath: claudeBin, Runner: r}

	transcript, err := invokeMeasuredProcess(ctx, opts, l, cfg, prompt, dest, mcpConfigPath, t.TempDir())
	if err != nil {
		t.Fatalf("invokeMeasuredProcess() = %v", err)
	}

	if transcript.Init == nil {
		t.Fatal("transcript carries no session-init record")
	}

	// First: a control cell's advertised tool list must be exactly the four built-ins, sorted -- a
	// silently degraded grant in a new directory would void every metric from such a run.
	wantTools := append([]string(nil), BuiltinTools...)
	sort.Strings(wantTools)
	gotTools := append([]string(nil), transcript.Init.Tools...)
	sort.Strings(gotTools)
	if !stringSlicesEqual(gotTools, wantTools) {
		t.Errorf("session-init tools = %v; want exactly %v", gotTools, wantTools)
	}

	// Second: the advertised server list must be empty, proving strict configuration mode held and
	// the operator's own personal servers did not load.
	if len(transcript.Init.MCPServers) != 0 {
		t.Errorf("session-init mcp_servers = %v; want none", transcript.Init.MCPServers)
	}

	// Third: only an executed call proves the grant works from a fresh directory. At least one Bash
	// tool use must have a matching tool result carrying output.
	assertAtLeastOneBashUseProducedOutput(t, transcript)

	// And no permission denial may name an unambiguously read-only command -- the advertised list
	// says what was offered, this is the only assertion that proves the grant actually works.
	if transcript.Result != nil {
		for _, raw := range transcript.Result.PermissionDenials {
			var probe permissionDenialProbe
			if err := json.Unmarshal(raw, &probe); err != nil {
				continue
			}
			if probe.ToolName == "Bash" && readOnlyBashCommandPattern.MatchString(probe.ToolInput.Command) {
				t.Errorf("permission denial refused a read-only Bash command: %q", probe.ToolInput.Command)
			}
		}
	}
}

// removeExistingWorktree removes any worktree already registered at dest, on a best-effort basis, so
// the worktree PrepareWorktree creates next is genuinely new.
func removeExistingWorktree(t *testing.T, ctx context.Context, r Runner, targetRepoPath, dest string) {
	t.Helper()
	if _, err := os.Stat(dest); err != nil {
		return
	}
	_ = r.Run(ctx, Cmd{Dir: targetRepoPath, Name: "git", Args: []string{"worktree", "remove", "--force", dest}})
	_ = os.RemoveAll(dest)
	_ = r.Run(ctx, Cmd{Dir: targetRepoPath, Name: "git", Args: []string{"worktree", "prune"}})
}

// assertAtLeastOneBashUseProducedOutput scans t's records for a Bash tool_use block and its matching
// tool_result block, failing unless at least one such pair carries non-empty result text.
func assertAtLeastOneBashUseProducedOutput(t *testing.T, transcript *Transcript) {
	t.Helper()

	bashUseIDs := map[string]bool{}
	for _, record := range transcript.Records {
		for _, block := range record.Message.Content {
			if block.Type == "tool_use" && block.Name == "Bash" {
				bashUseIDs[block.ID] = true
			}
		}
	}
	if len(bashUseIDs) == 0 {
		t.Fatal("transcript carries no Bash tool use at all")
	}

	for _, record := range transcript.Records {
		for _, block := range record.Message.Content {
			if block.Type != "tool_result" || !bashUseIDs[block.ToolUseID] {
				continue
			}
			if toolResultText(block) != "" {
				return
			}
		}
	}
	t.Error("no Bash tool use has a matching tool result carrying output")
}
