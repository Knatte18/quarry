// main.go is the stub MCP server card 32's mcp_test.go drives. It speaks line-delimited JSON-RPC
// over standard input and output, answers an initialize request, advertises exactly two tools --
// one a test cell grants and one it does not -- on a tools-list request, and answers a tools-call
// request with a deterministic JSON text payload. It exits on end of input. It depends on nothing
// outside the standard library and is excluded from the module's own build because it lives under a
// "testdata" directory, which the go tool never treats as a package.
package main

import (
	"bufio"
	"encoding/json"
	"os"
)

// grantedToolName and otherToolName are the two tools this stub advertises. The driving test grants
// grantedToolName to its cell and leaves otherToolName withheld, so the correspondence assertion has
// something to fail on if it were wrong.
const (
	grantedToolName = "toc"
	otherToolName   = "other"
)

// rpcRequest is the minimal JSON-RPC 2.0 request shape this stub reads. A request carrying no ID is
// a notification and gets no response.
type rpcRequest struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
}

// rpcResponse is the JSON-RPC 2.0 response shape this stub writes.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result"`
}

// toolDef is one entry of the tools-list response's tools array.
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		if len(req.ID) == 0 {
			// A notification (e.g. "notifications/initialized") gets no response.
			continue
		}

		var result any
		switch req.Method {
		case "initialize":
			result = map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "stubmcp", "version": "0.0.0"},
			}
		case "tools/list":
			result = map[string]any{
				"tools": []toolDef{
					{Name: grantedToolName, Description: "stub granted tool", InputSchema: map[string]any{"type": "object"}},
					{Name: otherToolName, Description: "stub withheld tool", InputSchema: map[string]any{"type": "object"}},
				},
			}
		case "tools/call":
			result = map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": `{"stub":"deterministic-payload"}`},
				},
			}
		default:
			result = map[string]any{}
		}

		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
		data, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		out.Write(data)
		out.WriteByte('\n')
		out.Flush()
	}
}
