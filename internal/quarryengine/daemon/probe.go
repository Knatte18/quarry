// probe.go implements the one readiness gate every EnsureServer strategy runs regardless of how it
// got its connection — a fresh spawn for native,
// or a reused dial or a fresh spawn for supervised — before handing that connection back to the
// caller.
// Per the design's caution, even a -remote=auto/dialed connection can silently hand back a
// reference to a hung shared instance, so a successful initialize handshake alone is not sufficient
// proof of health;
// probe is what actually exercises the connection end-to-end.

package daemon

import (
	"context"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
)

// probe queries the language server to verify the connection is alive and responsive.
func probe(ctx context.Context, client *lsp.Client, timeout time.Duration) error {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := client.WorkspaceSymbol(probeCtx, "")
	return err
}
