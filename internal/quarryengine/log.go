// log.go implements the engine's shared package-level logger.

package quarryengine

import (
	"log/slog"
	"os"
)

// Logger is quarryengine's own package-level slog handler: it writes to
// stderr at slog.LevelWarn unless the process has configured otherwise. Since
// the threshold defaults to Warn, an slog.Info call anywhere in this package
// is suppressed by default and only becomes visible once a caller lowers the
// handler's level — this matches Loomyard's own internal/logger, which also
// defaults to warn-level stderr output.
var Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
