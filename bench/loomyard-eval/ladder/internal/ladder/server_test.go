package ladder

import (
	"strings"
	"testing"
)

func TestMCPConfigDocument_NamesOneServerCarriesTargetDirAndEmptiesScrubbedKeys(t *testing.T) {
	document := MCPConfigDocument("/abs/quarry-mcp", "/tmp/target", "")

	servers, ok := document["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("document[%q] = %#v; want map[string]any", "mcpServers", document["mcpServers"])
	}
	if len(servers) != 1 {
		t.Fatalf("len(mcpServers) = %d; want 1", len(servers))
	}
	server, ok := servers["quarry"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers[%q] = %#v; want map[string]any", "quarry", servers["quarry"])
	}

	if server["command"] != "/abs/quarry-mcp" {
		t.Errorf("server[%q] = %v; want %q", "command", server["command"], "/abs/quarry-mcp")
	}

	args, ok := server["args"].([]string)
	if !ok {
		t.Fatalf("server[%q] = %#v; want []string", "args", server["args"])
	}
	wantArgs := []string{"--target-dir", "/tmp/target"}
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %v; want %v", args, wantArgs)
	}
	for i := range wantArgs {
		if args[i] != wantArgs[i] {
			t.Errorf("args[%d] = %q; want %q", i, args[i], wantArgs[i])
		}
	}

	env, ok := server["env"].(map[string]string)
	if !ok {
		t.Fatalf("server[%q] = %#v; want map[string]string", "env", server["env"])
	}
	if len(env) != 2 {
		t.Fatalf("env = %v; want exactly two keys, no other environment key mentioned", env)
	}
	if value, ok := env["QUARRY_STATE_DIR"]; !ok || value != "" {
		t.Errorf("env[%q] = %q, %v; want empty string, present", "QUARRY_STATE_DIR", value, ok)
	}
	if value, ok := env["QUARRY_BUILD_TAGS"]; !ok || value != "" {
		t.Errorf("env[%q] = %q, %v; want empty string, present", "QUARRY_BUILD_TAGS", value, ok)
	}
}

func TestBuildServer_FailurePropagatesBuilderOutputAsHarnessError(t *testing.T) {
	build := func(dir string, env []string, args ...string) (string, error) {
		return "compile error: undefined: cgo", &HarnessError{Message: "boom"}
	}

	_, err := BuildServer("/repo", build)
	if err == nil {
		t.Fatal("BuildServer() = nil error; want a *HarnessError on a build failure")
	}
	harnessErr, ok := err.(*HarnessError)
	if !ok {
		t.Fatalf("BuildServer() error type = %T; want *HarnessError", err)
	}
	if !strings.Contains(harnessErr.Message, "CGO_ENABLED=1") {
		t.Errorf("HarnessError.Message = %q; want it to name the CGO toolchain requirement", harnessErr.Message)
	}
	if !strings.Contains(harnessErr.Message, "undefined: cgo") {
		t.Errorf("HarnessError.Message = %q; want it to carry the builder's own output", harnessErr.Message)
	}
}

func TestBuildServer_InvokesBuilderWithRepoRootAndScrubbedCGOEnv(t *testing.T) {
	var gotDir string
	var gotEnv []string
	var gotArgs []string
	build := func(dir string, env []string, args ...string) (string, error) {
		gotDir = dir
		gotEnv = env
		gotArgs = args
		return "", nil
	}

	if _, err := BuildServer("/repo", build); err != nil {
		t.Fatalf("BuildServer() error = %v", err)
	}

	if gotDir != "/repo" {
		t.Errorf("builder dir = %q; want %q (the repository root)", gotDir, "/repo")
	}

	found := false
	for _, kv := range gotEnv {
		if kv == "CGO_ENABLED=1" {
			found = true
		}
	}
	if !found {
		t.Errorf("builder env = %v; want it to carry CGO_ENABLED=1", gotEnv)
	}

	wantArgs := []string{"go", "build", "-o", "/repo/quarry-mcp", "./cmd/quarry-mcp"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("builder args = %v; want %v", gotArgs, wantArgs)
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Errorf("builder args[%d] = %q; want %q", i, gotArgs[i], wantArgs[i])
		}
	}
}
