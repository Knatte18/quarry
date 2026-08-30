// callcontext.go resolves the per-call context every language-server-backed tool needs before it
// can call the facade: the registry, the absolute target directory, the daemon state directory,
// the build-tag set, and the per-entry timeout. It follows resolveContext's sequence in
// internal/cli/cli.go exactly, through the exported helpers, so the derived state directory stays
// bit-for-bit identical to the CLI's — a divergence would silently spawn a second gopls daemon and
// forfeit warm-daemon reuse.
//
// The toc tools call effectiveTargetDir directly and never call resolveCall: tocFileCommand and
// tocDirCommand (internal/cli/toc.go) never call resolveContext, quarry.LoadRegistry, or
// ResolveStateDir either, so a malformed servers.yaml must not fail a toc call.

package mcpserver

import (
	"time"

	"github.com/Knatte18/quarry/internal/cli"
	"github.com/Knatte18/quarry/quarry"
)

// callContext holds the per-call state a language-server-backed tool's handler derives once, then
// reuses across every entry in that call.
type callContext struct {
	// Registry is the servers.yaml-backed language-server registry loaded for this call.
	Registry quarry.Registry
	// TargetDir is the absolute project directory this call resolved, either the launch default or
	// an absolutised per-call override.
	TargetDir string
	// StateDir is the absolute daemon state directory this call resolved, bit-for-bit identical to
	// what internal/cli's own resolution produces for the same inputs.
	StateDir string
	// BuildTags is the normalized build-tag set this call resolved.
	BuildTags []string
	// Timeout is the launch-only --timeout value, carried straight from Config.Timeout: a
	// per-entry deadline for each entry's facade call, not a whole-call budget, exactly as the CLI
	// applies Options.Timeout per invocation.
	Timeout time.Duration
}

// resolveCall resolves the full per-call context for a language-server-backed tool: the absolute
// target directory (cfg.TargetDir), the normalized build-tag set (cli.ResolveBuildTags), the
// servers.yaml config path (cli.ResolveConfigPath) and the registry loaded from it
// (quarry.LoadRegistry), and the daemon state directory (cli.ResolveStateDir) — in that order,
// exactly mirroring resolveContext's sequence in internal/cli/cli.go.
//
// Using these exported helpers rather than a local copy is what keeps the state-directory key
// bit-for-bit identical to the CLI's; a divergence would silently spawn a second gopls daemon and
// forfeit warm-daemon reuse.
func resolveCall(cfg Config, buildTags string) (callContext, error) {
	absTargetDir := cfg.TargetDir

	tags := cli.ResolveBuildTags(buildTags)

	configPath, err := cli.ResolveConfigPath(cfg.ConfigPath)
	if err != nil {
		return callContext{}, err
	}

	registry, err := quarry.LoadRegistry(configPath)
	if err != nil {
		return callContext{}, err
	}

	stateDir, err := cli.ResolveStateDir(cfg.StateDir, absTargetDir, tags)
	if err != nil {
		return callContext{}, err
	}

	return callContext{
		Registry:  registry,
		TargetDir: absTargetDir,
		StateDir:  stateDir,
		BuildTags: tags,
		Timeout:   cfg.Timeout,
	}, nil
}

// options builds a quarry.Options value for one entry's facade call, populated exactly as
// buildOptions populates it in internal/cli/cli.go — Registry, TargetDir, StateDir, Lang, Query,
// Timeout, BuildTags — and leaving SkipVerification at its zero value so the default is "verify".
func (c callContext) options(lang string, query quarry.Query) quarry.Options {
	return quarry.Options{
		Registry:  c.Registry,
		TargetDir: c.TargetDir,
		StateDir:  c.StateDir,
		Lang:      lang,
		Query:     query,
		Timeout:   c.Timeout,
		BuildTags: c.BuildTags,
	}
}
