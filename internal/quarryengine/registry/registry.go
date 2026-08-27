// Package registry implements the language-server registry: a pinned built-in set with an optional
// file overlay.
//
//   - Built-ins (builtins()): a pinned, default-free Registry (a map[string]Entry) for the five
//     supported languages, each entry naming its detection Markers, whether Match requires "all" or
//     "any" of them, the launch Command argv, an operator-facing InstallHint, and — Go's entry only
//     in V1 — PinnedVersion and HasNativeDaemon (see daemon/doc.go's own package doc comment).
//     BuiltinRegistry() exposes this to any caller (the CLI uses it when no servers.yaml overlay is
//     resolvable).
//   - Optional servers.yaml overlay (LoadRegistry(path), load.go): loaded from a resolved absolute
//     path the caller supplies — internal/cli resolves that path via the
//     --config/$QUARRY_CONFIG/os.UserConfigDir() precedence documented in README.md; this package
//     joins nothing of its own. An absent file is not an error (built-ins suffice); a present file
//     decodes with yaml.Decoder.KnownFields(true) (an unknown field anywhere is a loud error naming
//     the offending entry and file path) and every present entry whole-replaces its built-in
//     counterpart — no field-level merge, so an override can never silently mix a stale built-in
//     default with a new one. See docs/servers.yaml.example for every built-in entry at its default
//     values and for how to add a language.
//   - Detection precedence (detect.go): a fixed order (go, rust, csharp, typescript, python, pinned
//     as a slice — map iteration is unordered) so a project matching more than one language's
//     markers (e.g. a Go module vendoring a TypeScript frontend) resolves deterministically to the
//     earlier language. --lang bypasses precedence entirely, looking the override up directly in
//     the registry (an unknown override names every valid language in its error).
//
// registry.go itself defines the registry shape (Entry, Registry), the pinned built-in fallback set,
// the fixed detection precedence order, and entry validation. builtins() is the offline default
// every consumer gets with zero servers.yaml present, and BuiltinRegistry() is the one-line exported
// accessor the CLI layer uses when no servers.yaml overlay is resolvable.
package registry

import "fmt"

// Entry describes how to detect and launch the language server for one language.
// Markers lists the marker files/dirs that identify a project as this language (checked relative to
// the target directory);
// Match selects whether all or any of them must be present.
// Command is the launch argv (the first element is the binary looked up on $PATH);
// InstallHint is the operator-facing command to install that binary when it's missing.
// PinnedVersion is the exact toolchain version the toolchain manager installs when HasNativeDaemon
// is true (empty means "no pinned version, no managed install" — the legacy cold-spawn-per-call
// path).
// HasNativeDaemon reports whether this language has a persistent, `EnsureServer`-managed daemon
// strategy (native or supervised) at all;
// the zero value (false) means the language never goes through `EnsureServer` and keeps today's
// cold-spawn-per-call behavior unchanged.
// Both fields are optional and Go-only in V1: every entry except "go" leaves them at their zero
// values.
//
// The yaml struct tags let LoadRegistry (load.go) decode a servers.yaml entry directly into this
// type: without them, yaml.v3's default field matching would require a YAML key of "installhint"
// (the lowercased Go field name with no separator) rather than the template's snake_case
// "install_hint".
type Entry struct {
	Markers         []string `yaml:"markers"`
	Match           string   `yaml:"match"`
	Command         []string `yaml:"command"`
	InstallHint     string   `yaml:"install_hint"`
	PinnedVersion   string   `yaml:"pinned_version"`
	HasNativeDaemon bool     `yaml:"has_native_daemon"`
}

// Registry maps a canonical language name ("go", "python", "csharp", "typescript", "rust") to its
// Entry.
// The zero value (a nil map) behaves like an empty registry: lookups fail cleanly rather than
// panicking.
type Registry map[string]Entry

// precedence is the fixed language-detection order used by DetectLanguage
// when no --lang override is given. It is pinned here (not derived from map
// iteration, which is unordered in Go) per the batch-local decision: earlier
// languages win when a target directory satisfies more than one language's
// markers (e.g. a Go module vendoring a TypeScript frontend still resolves
// to "go").
var precedence = []string{"go", "rust", "csharp", "typescript", "python"}

// builtins returns the pinned, default-free fallback registry for the five
// supported languages. This is what every consumer gets with zero
// servers.yaml present — operator overrides live only in the seeded
// servers.yaml (see docs/servers.yaml.example), never baked into Go, so
// changing a default never needs a recompile.
func builtins() Registry {
	return Registry{
		"go": {
			Markers:         []string{"go.mod"},
			Match:           "any",
			Command:         []string{"gopls"},
			InstallHint:     "go install golang.org/x/tools/gopls@latest",
			PinnedVersion:   "v0.23.0",
			HasNativeDaemon: true,
		},
		"python": {
			Markers:     []string{"pyproject.toml", "setup.py", "setup.cfg"},
			Match:       "any",
			Command:     []string{"pyright-langserver", "--stdio"},
			InstallHint: "npm install -g pyright",
		},
		"csharp": {
			Markers:     []string{".sln", ".csproj"},
			Match:       "any",
			Command:     []string{"csharp-ls"},
			InstallHint: "dotnet tool install --global csharp-ls",
		},
		"typescript": {
			Markers:     []string{"package.json", "tsconfig.json"},
			Match:       "all",
			Command:     []string{"typescript-language-server", "--stdio"},
			InstallHint: "npm install -g typescript-language-server typescript",
		},
		"rust": {
			Markers:     []string{"Cargo.toml"},
			Match:       "any",
			Command:     []string{"rust-analyzer"},
			InstallHint: "install via rustup component add rust-analyzer",
		},
	}
}

// BuiltinRegistry returns the pinned built-in registry (builtins()).
// It is the registry the CLI layer uses when no servers.yaml overlay is resolvable — e.g.
// no config file at any precedence tier — so lookup still works with zero configuration.
func BuiltinRegistry() Registry {
	return builtins()
}

// validateEntry enforces the closed shape every Entry (built-in or
// file-supplied) must satisfy: non-empty Markers, Match in {"all", "any"},
// non-empty Command, and non-empty InstallHint. Every failure names the
// offending entry's language/alias name so the operator can find it in
// servers.yaml.
func validateEntry(name string, e Entry) error {
	if len(e.Markers) == 0 {
		return fmt.Errorf("quarry: entry %q has no markers", name)
	}
	if e.Match != "all" && e.Match != "any" {
		return fmt.Errorf("quarry: entry %q has invalid match %q; want \"all\" or \"any\"", name, e.Match)
	}
	if len(e.Command) == 0 {
		return fmt.Errorf("quarry: entry %q has no command", name)
	}
	if e.InstallHint == "" {
		return fmt.Errorf("quarry: entry %q has no install hint", name)
	}
	return nil
}
