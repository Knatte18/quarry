package ladder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeMCPServerEnvVar, when set to "1" in this test binary's own environment, makes TestMain run this
// binary as a fake MCP stdio server instead of the Go test suite. Warm spawns exec.Command(serverPath,
// ...) and passes it env as the child's own environment (see Warm's cmd.Env = env); passing os.Args[0]
// as serverPath with this variable set in env re-execs the test binary itself as the fake server,
// mirroring the standard library's own TestHelperProcess self-exec pattern -- this is the fake this
// batch's plan requires for testing Warm's post-condition failure path without spawning a real
// quarry-mcp binary.
const fakeMCPServerEnvVar = "LADDER_FAKE_MCP_SERVER"

func TestMain(m *testing.M) {
	if os.Getenv(fakeMCPServerEnvVar) == "1" {
		runFakeMCPServerThatStartsNoDaemon()
		return
	}
	os.Exit(m.Run())
}

// runFakeMCPServerThatStartsNoDaemon answers WarmUpTool's call successfully but writes no daemon.json
// anywhere -- Warm's post-condition check is what must then fail, not the call itself.
func runFakeMCPServerThatStartsNoDaemon() {
	server := mcp.NewServer(&mcp.Implementation{Name: "fake-quarry-mcp", Version: "0.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: WarmUpTool, Description: "fake warm-up tool that starts no daemon"},
		func(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{}, nil, nil
		},
	)
	_ = server.Run(context.Background(), &mcp.StdioTransport{})
}

func TestWarm_ReturnsHarnessErrorWhenNoDaemonJSONAppearsAfterTheCall(t *testing.T) {
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	targetDir := t.TempDir()
	cacheDir := t.TempDir()
	env := append(os.Environ(), fakeMCPServerEnvVar+"=1")

	err = Warm(testBinary, targetDir, env, cacheDir)
	if err == nil {
		t.Fatal("Warm() = nil error; want a *HarnessError when the warm-up call starts no daemon")
	}
	harnessErr, ok := err.(*HarnessError)
	if !ok {
		t.Fatalf("Warm() error type = %T; want *HarnessError", err)
	}
	if !strings.Contains(harnessErr.Message, "daemon.json") {
		t.Errorf("HarnessError.Message = %q; want it to name the missing daemon.json", harnessErr.Message)
	}

	stateDir, err := ResolveStateDir(targetDir, cacheDir, env)
	if err != nil {
		t.Fatalf("ResolveStateDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, daemonLang, "daemon.json")); !os.IsNotExist(err) {
		t.Errorf("daemon.json exists at the resolved state dir; want the fake server to have started none")
	}
}
