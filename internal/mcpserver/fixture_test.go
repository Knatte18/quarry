// fixture_test.go declares the committed fixture repository this batch's tests all read, and
// connectedClient, the client-session helper every test file in this batch calls to wire a server
// for that repository over the SDK's in-memory transport.
//
// The fixture repository under testdata/repo is a committed tree, following
// internal/engine/testdata/tree's precedent, rather than this package's own writeScratchTree
// (root_test.go's per-package copy of the repository's standing scratch-tree convention). Card 13's
// six goldens are only a stable gate when the bytes they were produced from are themselves stable
// across every future test run, not just the run that generated them, and a committed tree is what
// makes that true — a tree built fresh by each test run would drift with the toolchain or the
// filesystem in ways a committed tree cannot.

package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Knatte18/quarry/quarry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fixtureRepoRoot returns the absolute path to this package's committed fixture repository under
// testdata/repo, resolved from runtime.Caller(0) so it does not depend on the working directory the
// test binary happens to run from. It fails the test if the directory is missing.
func fixtureRepoRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("fixtureRepoRoot: runtime.Caller(0) failed to resolve this file's path")
	}
	root := filepath.Join(filepath.Dir(thisFile), "testdata", "repo")
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		t.Fatalf("fixtureRepoRoot: %q is not an existing directory: %v", root, err)
	}
	return root
}

// connectedClient opens root as a quarry.Repo, constructs a server over it with NewServer, wires the
// server to a client over mcp.NewInMemoryTransports, connects both sides, and registers a
// t.Cleanup closing both sessions. It returns the connected client session.
//
// connectedClient takes root as a parameter rather than always calling fixtureRepoRoot itself,
// because card 16's broken-symlink case runs this same helper against a scratch tree the committed
// fixture repository cannot portably hold.
func connectedClient(t *testing.T, root string) *mcp.ClientSession {
	t.Helper()

	repo, err := quarry.Open(root)
	if err != nil {
		t.Fatalf("connectedClient: quarry.Open(%q): %v", root, err)
	}
	server := NewServer(repo, root)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connectedClient: server.Connect: %v", err)
	}
	t.Cleanup(func() {
		if err := serverSession.Close(); err != nil {
			t.Errorf("connectedClient: closing server session: %v", err)
		}
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "mcpserver-test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connectedClient: client.Connect: %v", err)
	}
	t.Cleanup(func() {
		if err := clientSession.Close(); err != nil {
			t.Errorf("connectedClient: closing client session: %v", err)
		}
	})

	return clientSession
}
