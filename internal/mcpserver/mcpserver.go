// mcpserver.go declares NewServer, which constructs the MCP server this package's cmd/quarry-mcp
// binary runs over a stdio transport.

package mcpserver

import (
	"github.com/Knatte18/quarry/quarry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverVersion is this server implementation's own version string, reported in its MCP
// Implementation. It is deliberately not a contract: nothing in the ladder harness reads it, and it
// tracks this package's own development rather than the module version.
const serverVersion = "0.1.0"

// NewServer constructs the MCP server for repo, rooted at root, and registers its one tool. repo
// and root are both already resolved and validated by the caller — NewServer performs no
// validation of its own and returns no error.
//
// The server's implementation name "quarry" and its one tool's name "toc" are external contracts,
// not choices made here: the ladder config declares server: {name: quarry} and
// quarry_tools: [toc], and the harness composes the tool identifier mcp__quarry__toc from them, so
// spelling either name differently means the granted ladder cell's tool is never allowed.
func NewServer(repo *quarry.Repo, root string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "quarry", Version: serverVersion}, nil)
	registerTOC(s, repo, root)
	return s
}
