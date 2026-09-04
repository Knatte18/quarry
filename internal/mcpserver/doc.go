// Package mcpserver binds the quarry facade onto MCP tools: it constructs the MCP server, declares
// the toc tool, and translates between the protocol's request/result shapes and the facade's own
// Repo and TOCOptions. The quarry facade package (github.com/Knatte18/quarry/quarry) is this
// package's only route to the engine — nothing here imports internal/engine, and a mechanical test
// enforces that.
//
// Nothing in this package writes to standard output. That stream is reserved for the framed MCP
// transport (cmd/quarry-mcp's stdio connection); a stray write here would corrupt the protocol
// stream in a way indistinguishable, from the client's side, from a malformed frame.
package mcpserver
