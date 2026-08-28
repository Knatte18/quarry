// cli.go exposes quarry's own cobra command tree.
// It is the sole consumer of internal/output: the quarry engine package returns typed Go errors
// and results with no io.Writer/exit-code machinery (per the plan's engine/CLI layering Shared
// Decision), so this file is where every engine result and typed error gets mapped to the
// internal/output JSON envelope.

// Package cli builds quarry's own command tree, exposing "refs" (every reference to a symbol or
// position), "definition" (a symbol or position's definition), "symbol" (a workspace/symbol name
// search), "assert-no-callers" (a CI-shaped gate: fail if a symbol has any caller outside its
// declaration and an allowed list), and the "toc" group — "toc file" and "toc dir" (a source
// file's or a directory's table of contents, tree-sitter-backed rather than LSP-backed) — across
// the languages the quarry engine package supports.
//
// # The exit-code contract
//
// Every verb's single-argument call exits 0 (found), 1 (not found, or any other
// engine error), or 2 (ambiguous — the response body still carries "ok":true with a
// "candidates" field, since multiple valid answers is not a process error, just a
// result the caller must disambiguate). symbol never produces "ambiguous"/exit 2 in
// either shape: returning several workspace/symbol candidates is its ordinary
// successful answer, not an error state needing disambiguation, so its single-arg
// call only ever exits 0 or 1. toc's single-argument call has no "ambiguous" outcome
// either: it exits 0 or 1 only.
//
// A call with 2 or more positional arguments switches to batch mode instead of the
// single-result shape above: it returns one JSON entry per argument under a top-level
// "results" array, each entry carrying a per-entry status. The results array, the
// status vocabulary, and the worst-status exit-code ranking are shared by every verb;
// only the entry's identity key differs by shape. refs/definition/symbol/
// assert-no-callers key each entry on "symbol", the resolved name or position; toc
// file/toc dir key each entry on "path" instead, since a toc lookup addresses a
// filesystem path, not a symbol. The status vocabulary is "found", "not_found",
// "ambiguous" (refs/definition only; toc never produces it, matching its
// single-argument shape above), or "error" (a genuine infrastructure failure,
// distinct from a confirmed-absent "not_found") — and the process exit code is set to
// the worst status present across the whole batch, ranked found(0) < not_found(1) <
// ambiguous(2) < error(3).
package cli

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/internal/output"
	"github.com/Knatte18/quarry/quarry"
)

// Command returns quarry's own cobra command tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quarry",
		Short: "code intelligence: LSP-backed symbol lookups plus tree-sitter-backed file/directory tables of contents",
		RunE:  GroupRunE,
	}

	cmd.PersistentFlags().String("config", "", "explicit path to a servers.yaml overlay, overriding $QUARRY_CONFIG and the user config directory default")
	cmd.PersistentFlags().String("state-dir", "", "explicit daemon state directory, overriding $QUARRY_STATE_DIR and the user cache directory default")

	cmd.AddCommand(refsCommand())
	cmd.AddCommand(definitionCommand())
	cmd.AddCommand(symbolCommand())
	cmd.AddCommand(assertNoCallersCommand())
	cmd.AddCommand(tocCommand())
	return cmd
}

// refsCommand builds the "refs" subcommand.
func refsCommand() *cobra.Command {
	var targetDir string
	var lang string
	var timeout time.Duration
	var inFile string
	var within string
	var buildTags string

	refs := &cobra.Command{
		Use:   "refs <symbol|file:line:col>",
		Short: "list every reference to a symbol or source position",
		Long: `refs finds every reference to a symbol name or an explicit source position,
using the LSP "textDocument/references" request against the language server
detected for --target-dir (or --lang, to override detection).

The single positional argument is either:
  - a symbol name, resolved via the language server's workspace/symbol search:
      quarry refs MyFunction
  - an explicit "file:line:col" position (1-based line and column), bypassing
    name resolution entirely:
      quarry refs internal/foo/bar.go:42:8

--in-file <path> resolves each positional argument as a bare symbol name
within exactly that one file, via an exhaustive textDocument/documentSymbol
search rather than a project-wide workspace/symbol search — the positional
is always treated as a bare name, never position-parsed, even if it happens
to look like "file:line:col":
    quarry refs --in-file internal/foo/bar.go MyFunc

Passing 2 or more positional arguments switches to batch mode: each argument
is looked up independently and the results are reported as one array, rather
than the single-symbol envelope above:
    {"ok":true,"results":[{"symbol":...,"status":"found"|"not_found"|"ambiguous"|"error",...}, ...]}
The process exit code is set to the worst status present across the batch
(0 < 1 < 2 < 3). Example:
    quarry refs Foo Bar Baz
--in-file composes with batch mode too, resolving every positional against
the same file:
    quarry refs --in-file internal/foo/bar.go Open Close

The result set is complete and semantically resolved by the language server
(including calls reached only through an interface, which no amount of
grepping can prove) — a caller does not need to cross-check it with grep or
re-verify individual candidates. A successful single-arg lookup carries a
machine-readable "resolution":"complete" field as this trust marker; batch
mode carries the same field on each per-entry "found" result.

Known limitation, and what --within is for: gopls' references for an
interface method conservatively include every method matching that
name+signature across every structurally-compatible interface anywhere in
the workspace — not just calls through the specific interface value you
queried. This is documented gopls behavior, not a bug in this wrapper, but
it means two unrelated, identically-shaped interfaces in different packages
(e.g. two local "type clock interface { Now() time.Time }" declarations)
conflate their results by default. "resolution":"complete" still means
"every result for the query as given" — it is not false — but for an
unscoped interface-method query, "complete" can include out-of-scope noise
a caller must still filter by hand. --within <dir> restricts the result set
to references whose file lies within <dir> (relative to --target-dir, or
absolute), discarding everything else, so a query already known to be
scoped to one package comes back both complete and precise:
    quarry refs --within internal/websterengine SomeMethod`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			// CwdFrom(ctx) resolves the seam cwd — the cwd RunCLIIn
			// injected into ctx, or the process cwd when none was — anchoring
			// both the default target directory and the overlay-base
			// resolution below.
			cwd, err := CwdFrom(ctx)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			dir := targetDir
			if dir == "" {
				dir = cwd
			} else if filepath.IsAbs(dir) {
				dir = filepath.Clean(dir)
			} else {
				dir = filepath.Join(cwd, dir)
			}

			configFlag, _ := cmd.Flags().GetString("config")
			stateDirFlag, _ := cmd.Flags().GetString("state-dir")
			buildTagsResolved := resolveBuildTags(buildTags)
			registry, _, stateDir, err := resolveContext(dir, configFlag, stateDirFlag, buildTagsResolved)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			// buildQuery is the one seam both the single-arg and batch-mode
			// paths below call to turn a positional argument into a Query:
			// --in-file routes every argument through inFileQuery instead of
			// parseQuery, so a positional is always a bare name against that
			// one file, never position-parsed — even when --in-file is
			// combined with batch mode.
			buildQuery := func(arg string) (quarry.Query, error) {
				if inFile != "" {
					return inFileQuery(cwd, inFile, arg)
				}
				return parseQuery(cwd, arg)
			}

			if len(args) == 1 {
				query, err := buildQuery(args[0])
				if err != nil {
					SetExit(ctx, output.Err(out, err.Error()))
					return nil
				}

				opts := buildOptions(registry, dir, stateDir, lang, query, timeout, buildTagsResolved)

				results, err := quarry.References(ctx, opts)
				if err == nil && within != "" {
					results = filterWithin(results, within, dir)
				}
				emitLookupResult(ctx, out, "references", results, err)
				return nil
			}

			runBatch(ctx, out, args, func(symbol string) (batchStatus, map[string]any) {
				query, err := buildQuery(symbol)
				if err != nil {
					return statusError, map[string]any{"error": err.Error()}
				}
				results, err := quarry.References(ctx, buildOptions(registry, dir, stateDir, lang, query, timeout, buildTagsResolved))
				if err == nil && within != "" {
					results = filterWithin(results, within, dir)
				}
				return classifyLookupError(err, "references", results)
			})
			return nil
		},
	}

	refs.Flags().StringVar(&targetDir, "target-dir", "", "project directory to detect the language in and root the server at (default: cwd)")
	refs.Flags().StringVar(&lang, "lang", "", "override language detection with this registry key")
	refs.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "deadline for each LSP request phase (initialize, resolve, references)")
	refs.Flags().StringVar(&inFile, "in-file", "", "resolve each positional argument as a bare symbol name within this one file, instead of a project-wide workspace/symbol search")
	refs.Flags().StringVar(&within, "within", "", "restrict results to references whose file lies within this directory (relative to --target-dir, or absolute) — see the interface-method conflation note above")
	refs.Flags().StringVar(&buildTags, "build-tags", "", "comma-separated Go build tags to scope the query to (default: $QUARRY_BUILD_TAGS, or none); an error, not a silent no-op, for a language whose registry entry carries no build-tag template")

	return refs
}

// definitionCommand builds the "definition" subcommand.
func definitionCommand() *cobra.Command {
	var targetDir string
	var lang string
	var timeout time.Duration
	var inFile string
	var within string
	var buildTags string

	definition := &cobra.Command{
		Use:   "definition <symbol|file:line:col>",
		Short: "show the definition of a symbol or source position",
		Long: `definition shows the definition of a symbol name or an explicit source
position, using the LSP "textDocument/definition" request against the
language server detected for --target-dir (or --lang, to override
detection).

The single positional argument is either:
  - a symbol name, resolved via the language server's workspace/symbol search:
      quarry definition MyFunction
  - an explicit "file:line:col" position (1-based line and column), bypassing
    name resolution entirely:
      quarry definition internal/foo/bar.go:42:8

--in-file <path> resolves each positional argument as a bare symbol name
within exactly that one file, via an exhaustive textDocument/documentSymbol
search rather than a project-wide workspace/symbol search — the positional
is always treated as a bare name, never position-parsed, even if it happens
to look like "file:line:col":
    quarry definition --in-file internal/foo/bar.go MyFunc

Passing 2 or more positional arguments switches to batch mode: each argument
is looked up independently and the results are reported as one array, rather
than the single-symbol envelope above:
    {"ok":true,"results":[{"symbol":...,"status":"found"|"not_found"|"ambiguous"|"error",...}, ...]}
The process exit code is set to the worst status present across the batch
(0 < 1 < 2 < 3). definition has no other shape difference from refs in batch
mode. Example:
    quarry definition Foo Bar Baz
--in-file composes with batch mode too, resolving every positional against
the same file:
    quarry definition --in-file internal/foo/bar.go Open Close

The result is semantically resolved by the language server, not text-matched
— a caller does not need to cross-check it with grep. A successful single-arg
lookup carries a machine-readable "resolution":"complete" field as this trust
marker; batch mode carries the same field on each per-entry "found" result.

--within <dir> restricts the result set to definitions whose file lies
within <dir> (relative to --target-dir, or absolute) — see "refs --help"
for why this exists (interface-method reference conflation across
structurally-identical interfaces in different packages).`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			// CwdFrom(ctx) resolves the seam cwd — the cwd RunCLIIn
			// injected into ctx, or the process cwd when none was — anchoring
			// both the default target directory and the overlay-base
			// resolution below.
			cwd, err := CwdFrom(ctx)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			dir := targetDir
			if dir == "" {
				dir = cwd
			} else if filepath.IsAbs(dir) {
				dir = filepath.Clean(dir)
			} else {
				dir = filepath.Join(cwd, dir)
			}

			configFlag, _ := cmd.Flags().GetString("config")
			stateDirFlag, _ := cmd.Flags().GetString("state-dir")
			buildTagsResolved := resolveBuildTags(buildTags)
			registry, _, stateDir, err := resolveContext(dir, configFlag, stateDirFlag, buildTagsResolved)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			// buildQuery is the one seam both the single-arg and batch-mode
			// paths below call to turn a positional argument into a Query:
			// --in-file routes every argument through inFileQuery instead of
			// parseQuery, so a positional is always a bare name against that
			// one file, never position-parsed — even when --in-file is
			// combined with batch mode.
			buildQuery := func(arg string) (quarry.Query, error) {
				if inFile != "" {
					return inFileQuery(cwd, inFile, arg)
				}
				return parseQuery(cwd, arg)
			}

			if len(args) == 1 {
				query, err := buildQuery(args[0])
				if err != nil {
					SetExit(ctx, output.Err(out, err.Error()))
					return nil
				}

				opts := buildOptions(registry, dir, stateDir, lang, query, timeout, buildTagsResolved)

				results, err := quarry.Definition(ctx, opts)
				if err == nil && within != "" {
					results = filterWithin(results, within, dir)
				}
				emitLookupResult(ctx, out, "definitions", results, err)
				return nil
			}

			runBatch(ctx, out, args, func(symbol string) (batchStatus, map[string]any) {
				query, err := buildQuery(symbol)
				if err != nil {
					return statusError, map[string]any{"error": err.Error()}
				}
				results, err := quarry.Definition(ctx, buildOptions(registry, dir, stateDir, lang, query, timeout, buildTagsResolved))
				if err == nil && within != "" {
					results = filterWithin(results, within, dir)
				}
				return classifyLookupError(err, "definitions", results)
			})
			return nil
		},
	}

	definition.Flags().StringVar(&targetDir, "target-dir", "", "project directory to detect the language in and root the server at (default: cwd)")
	definition.Flags().StringVar(&lang, "lang", "", "override language detection with this registry key")
	definition.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "deadline for each LSP request phase (initialize, resolve, definition)")
	definition.Flags().StringVar(&inFile, "in-file", "", "resolve each positional argument as a bare symbol name within this one file, instead of a project-wide workspace/symbol search")
	definition.Flags().StringVar(&within, "within", "", "restrict results to definitions whose file lies within this directory (relative to --target-dir, or absolute)")
	definition.Flags().StringVar(&buildTags, "build-tags", "", "comma-separated Go build tags to scope the query to (default: $QUARRY_BUILD_TAGS, or none); an error, not a silent no-op, for a language whose registry entry carries no build-tag template")

	return definition
}

// symbolCommand builds the "symbol" subcommand.
func symbolCommand() *cobra.Command {
	var targetDir string
	var lang string
	var timeout time.Duration
	var buildTags string

	symbol := &cobra.Command{
		Use:   "symbol <query>",
		Short: "search workspace symbols by name",
		Long: `symbol searches workspace symbols by name, using the LSP
"workspace/symbol" request against the language server detected for
--target-dir (or --lang, to override detection).

Unlike refs/definition, the positional argument is always treated as a
literal search string — even one that happens to look like "file:line:col" —
never position-parsed:
    quarry symbol MyFunction

Passing 2 or more positional arguments switches to batch mode: each argument
is looked up independently and the results are reported as one array, rather
than the single-symbol envelope above:
    {"ok":true,"results":[{"symbol":...,"status":"found"|"not_found"|"error",...}, ...]}
Unlike refs/definition, symbol's status set is only three-way — there is no
"ambiguous" status and no exit code 2, since symbol never collapses multiple
matches into an ambiguity failure. Example:
    quarry symbol Foo Bar Baz`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			// CwdFrom(ctx) resolves the seam cwd — the cwd RunCLIIn
			// injected into ctx, or the process cwd when none was — anchoring
			// both the default target directory and the overlay-base
			// resolution below.
			cwd, err := CwdFrom(ctx)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			dir := targetDir
			if dir == "" {
				dir = cwd
			} else if filepath.IsAbs(dir) {
				dir = filepath.Clean(dir)
			} else {
				dir = filepath.Join(cwd, dir)
			}

			configFlag, _ := cmd.Flags().GetString("config")
			stateDirFlag, _ := cmd.Flags().GetString("state-dir")
			buildTagsResolved := resolveBuildTags(buildTags)
			registry, _, stateDir, err := resolveContext(dir, configFlag, stateDirFlag, buildTagsResolved)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			if len(args) == 1 {
				opts := buildOptions(registry, dir, stateDir, lang, symbolQuery(args[0]), timeout, buildTagsResolved)

				results, err := quarry.Symbol(ctx, opts)
				if err != nil {
					// Symbol never returns *ErrAmbiguousSymbol (per symbol-semantics,
					// it has no ambiguous state), so emitLookupResult's ambiguity
					// branch does not apply here — this is the simple, uniform
					// error-mapping shape refsCommand used before card 33's retrofit.
					SetExit(ctx, output.Err(out, err.Error()))
					return nil
				}

				SetExit(ctx, output.Ok(out, map[string]any{"symbols": symbolMatchFields(results)}))
				return nil
			}

			// Every batch entry is built directly from the raw arg string as
			// Query.Symbol, exactly like the single-arg path above — symbol's
			// batch mode never calls parseQuery/position-parsing either, so
			// "quarry symbol foo.go:1:1 bar.go:2:2" treats both
			// arguments as literal search strings, not positions, consistent
			// across both arg-count shapes.
			runBatch(ctx, out, args, func(symbol string) (batchStatus, map[string]any) {
				results, err := quarry.Symbol(ctx, buildOptions(registry, dir, stateDir, lang, quarry.Query{Symbol: symbol}, timeout, buildTagsResolved))
				return classifySymbolError(err, results)
			})
			return nil
		},
	}

	symbol.Flags().StringVar(&targetDir, "target-dir", "", "project directory to detect the language in and root the server at (default: cwd)")
	symbol.Flags().StringVar(&lang, "lang", "", "override language detection with this registry key")
	symbol.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "deadline for the workspace/symbol request phase")
	symbol.Flags().StringVar(&buildTags, "build-tags", "", "comma-separated Go build tags to scope the query to (default: $QUARRY_BUILD_TAGS, or none); an error, not a silent no-op, for a language whose registry entry carries no build-tag template")

	return symbol
}

// resolveContext performs the three pre-flight derivations every lookup command needs — the
// absolute target directory, the servers.yaml overlay load, and the state directory — each told
// its resolved inputs rather than deriving them from a lyx hub: quarry has no hub, so there is no
// in-hub/out-of-hub branch here, only the resolution this seam now owns outright.
//
// dir is the already-defaulted directory (the caller has already resolved the seam cwd into it
// when --target-dir was omitted), never the raw --target-dir flag value: passing the raw flag
// would resolve filepath.Abs("") (the process working directory) rather than the seam cwd.
//
// configFlag and stateDirFlag are the --config and --state-dir flag values, threaded straight
// through to resolveConfigPath and resolveStateDir so their own $QUARRY_CONFIG/$QUARRY_STATE_DIR
// and user-directory-default precedence tiers apply unchanged.
//
// buildTags is the already-resolved (resolveBuildTags-normalized) build-tag set, threaded
// straight through to resolveStateDir so the returned state directory carries its "tags-<hex>"
// segment whenever buildTags is non-empty.
//
// The returned error carries a resolveConfigPath, quarry.LoadRegistry, or resolveStateDir failure
// unchanged — a malformed servers.yaml, or a userConfigDir/userCacheDir failure, still fails the
// lookup rather than degrading silently.
func resolveContext(dir, configFlag, stateDirFlag string, buildTags []string) (quarry.Registry, string, string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		// Preserve the pre-refactor fallback exactly: when filepath.Abs itself
		// fails, the previous derivation's AnchorPath() was
		// filepath.Join(filepath.Dir(dir), filepath.Base(dir)), which
		// filepath.Clean(dir) reproduces byte for byte, so the failure mode
		// does not silently change.
		abs = filepath.Clean(dir)
	}

	configPath, err := resolveConfigPath(configFlag)
	if err != nil {
		return nil, "", "", err
	}
	registry, err := quarry.LoadRegistry(configPath)
	if err != nil {
		return nil, "", "", err
	}

	stateDir, err := resolveStateDir(stateDirFlag, abs, buildTags)
	if err != nil {
		return nil, "", "", err
	}

	return registry, abs, stateDir, nil
}

// buildOptions constructs a quarry.Options value, ensuring all construction
// sites thread StateDir and BuildTags consistently.
func buildOptions(registry quarry.Registry, targetDir string, stateDir string, lang string, query quarry.Query, timeout time.Duration, buildTags []string) quarry.Options {
	return quarry.Options{
		Registry:  registry,
		TargetDir: targetDir,
		StateDir:  stateDir,
		Lang:      lang,
		Query:     query,
		Timeout:   timeout,
		BuildTags: buildTags,
	}
}

// symbolQuery builds a quarry.Query for a bare symbol name, never position-parsed.
func symbolQuery(arg string) quarry.Query {
	return quarry.Query{Symbol: arg}
}

// assertNoCallersCommand builds the "assert-no-callers" subcommand, a CI gate that fails if
// a symbol has any caller outside its declaration and --except paths.
func assertNoCallersCommand() *cobra.Command {
	var targetDir string
	var lang string
	var timeout time.Duration
	var except []string
	var within string
	var buildTags string
	var noVerify bool

	cmd := &cobra.Command{
		Use:   "assert-no-callers <symbol|file:line:col>",
		Short: "fail if a symbol has any caller outside its declaration and --except paths",
		Long: `assert-no-callers finds every reference to a symbol name or an explicit
source position (the same resolution refs/definition use), then fails if any
reference remains once its own declaration site and every --except path are
excluded.

Intended for a mill batch's "verify:" step: mechanically confirm a Deletes: or
Moves: batch's target symbol truly has no remaining external callers before
the batch is approved, turning a review-discipline judgment call into a
deterministic CI gate.

The single positional argument is either a symbol name (resolved via
workspace/symbol) or an explicit "file:line:col" position, exactly as for
refs/definition. --except may be repeated; each is a file path (relative to
--target-dir, or absolute) allowed to keep referencing the symbol without
failing the check — typically the symbol's own sanctioned wrapper.

Exit 0: no unexpected callers remain; the envelope carries an empty "callers"
list. Exit 1: either one or more unexpected callers were found (the envelope
carries "violation":true and the "callers" list), or the lookup itself failed
(not found, server error) — the "violation" field is the only way to tell
these two exit-1 cases apart. Exit 2: the symbol name was ambiguous (more than
one workspace/symbol candidate) — the envelope carries "candidates", exactly
as refs/definition already report ambiguity.

--within <dir> is an ordinary optional scoping filter: it restricts the
caller search to references whose file lies within <dir> (relative to
--target-dir, or absolute), before --except is applied. Useful when you
already know the callers you care about live under one directory and want
to narrow the search, but not required for a correct interface-method
check — see below.

gopls' references for an interface method conservatively include every
method matching that name+signature across every structurally-compatible
interface anywhere in the workspace — not just calls through the interface
value you're actually checking (e.g. two unrelated local "type clock
interface { Now() time.Time }" declarations in different packages). By
default, assert-no-callers filters that conservative set itself: it
resolves each candidate reference's own definition and keeps only those
that resolve back to the queried symbol's declaration, so the reported
callers are precise without needing --within to scope away the noise.

--no-verify reinstates the older, noisier behaviour that reports every
gopls reference unfiltered by declaration, including the interface-method
conflation above. --no-verify is assert-no-callers-only: refs, definition,
and symbol are unchanged by this task and gain no verification flag.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			// CwdFrom(ctx) resolves the seam cwd — the cwd RunCLIIn
			// injected into ctx, or the process cwd when none was — anchoring
			// both the default target directory and the overlay-base
			// resolution below.
			cwd, err := CwdFrom(ctx)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			dir := targetDir
			if dir == "" {
				dir = cwd
			} else if filepath.IsAbs(dir) {
				dir = filepath.Clean(dir)
			} else {
				dir = filepath.Join(cwd, dir)
			}

			configFlag, _ := cmd.Flags().GetString("config")
			stateDirFlag, _ := cmd.Flags().GetString("state-dir")
			buildTagsResolved := resolveBuildTags(buildTags)
			registry, _, stateDir, err := resolveContext(dir, configFlag, stateDirFlag, buildTagsResolved)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			query, err := parseQuery(cwd, args[0])
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			opts := buildOptions(registry, dir, stateDir, lang, query, timeout, buildTagsResolved)
			opts.SkipVerification = noVerify

			// The declaration set must come from a real textDocument/definition
			// rather than from the caller-supplied position: quarryengine.Position.Character
			// is a 1-based byte column, while quarry.Reference.Character is a
			// 1-based UTF-16 column, so the two coincide only on a pure-ASCII
			// line. quarry.Callers returns the definition-only declaration set
			// precisely so this exclusion keeps working on any line.
			refs, declRefs, err := quarry.Callers(ctx, opts)
			if err != nil {
				emitAmbiguousOrError(ctx, out, err)
				return nil
			}
			if within != "" {
				// Scope the candidate set before --except runs. --within is
				// an ordinary optional filter here — see the Long help for
				// what it does now that verification runs by default inside
				// quarry.Callers, ahead of all three filters below.
				refs = filterWithin(refs, within, dir)
			}

			exceptAbs := make(map[string]bool, len(except))
			for _, e := range except {
				p := e
				if !filepath.IsAbs(p) {
					p = filepath.Join(dir, p)
				}
				exceptAbs[filepath.Clean(p)] = true
			}

			violations := filterUnexpectedCallers(refs, declRefs, exceptAbs)
			if len(violations) == 0 {
				SetExit(ctx, output.Ok(out, map[string]any{"callers": []map[string]any{}}))
				return nil
			}

			output.Ok(out, map[string]any{"violation": true, "callers": referenceFields(violations)})
			SetExit(ctx, 1)
			return nil
		},
	}

	cmd.Flags().StringVar(&targetDir, "target-dir", "", "project directory to detect the language in and root the server at (default: cwd)")
	cmd.Flags().StringVar(&lang, "lang", "", "override language detection with this registry key")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "deadline for each LSP request phase (initialize, resolve, references/definition)")
	cmd.Flags().StringArrayVar(&except, "except", nil, "file path allowed to reference the symbol without failing the check (repeatable)")
	cmd.Flags().StringVar(&within, "within", "", "optional: restrict the caller search to references whose file lies within this directory (relative to --target-dir, or absolute), applied before --except")
	cmd.Flags().StringVar(&buildTags, "build-tags", "", "comma-separated Go build tags to scope the query to (default: $QUARRY_BUILD_TAGS, or none); an error, not a silent no-op, for a language whose registry entry carries no build-tag template")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "skip per-caller definition verification (fail-closed by default: an unverifiable reference is kept as a violation rather than dropped); assert-no-callers only")

	return cmd
}

// emitAmbiguousOrError maps References/Definition/Callers errors to the output envelope.
func emitAmbiguousOrError(ctx context.Context, out io.Writer, err error) bool {
	var ambiguous *quarry.ErrAmbiguousSymbol
	if errors.As(err, &ambiguous) {
		output.Ok(out, map[string]any{"candidates": ambiguous.Candidates})
		SetExit(ctx, 2)
		return false
	}
	SetExit(ctx, output.Err(out, err.Error()))
	return false
}

// filterUnexpectedCallers returns entries in refs that are neither in declRefs nor in exceptAbs.
func filterUnexpectedCallers(refs []quarry.Reference, declRefs []quarry.Reference, exceptAbs map[string]bool) []quarry.Reference {
	declSet := make(map[quarry.Reference]bool, len(declRefs))
	for _, d := range declRefs {
		declSet[d] = true
	}

	var violations []quarry.Reference
	for _, r := range refs {
		if declSet[r] {
			continue
		}
		if exceptAbs[filepath.Clean(r.File)] {
			continue
		}
		violations = append(violations, r)
	}
	return violations
}

// filterWithin returns entries in refs whose file lies within the specified directory.
func filterWithin(refs []quarry.Reference, within, baseDir string) []quarry.Reference {
	w := within
	if !filepath.IsAbs(w) {
		w = filepath.Join(baseDir, w)
	}
	// baseDir itself may still be relative here (e.g. --target-dir "."
	// passed through verbatim, never resolved to absolute elsewhere in this
	// file) — filepath.Abs resolves whatever remains against the process's
	// actual working directory, the same convention parseQuery's own
	// "file:line:col" path resolution already uses. Reference.File (what
	// every entry in refs is compared against) is always absolute, so w
	// must be too, or every comparison below silently fails.
	if abs, err := filepath.Abs(w); err == nil {
		w = abs
	}
	w = filepath.Clean(w)

	var filtered []quarry.Reference
	for _, r := range refs {
		if isWithinDir(w, filepath.Clean(r.File)) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// isWithinDir reports whether target lies within or is exactly dir.
func isWithinDir(dir, target string) bool {
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// symbolMatchFields converts SymbolMatch results to JSON-encodable maps.
func symbolMatchFields(matches []quarry.SymbolMatch) []map[string]any {
	fields := make([]map[string]any, len(matches))
	for i, m := range matches {
		fields[i] = map[string]any{
			"name":      m.Name,
			"kind":      m.Kind,
			"file":      m.File,
			"line":      m.Line,
			"character": m.Character,
		}
	}
	return fields
}

// emitLookupResult maps References/Definition results to the output envelope.
func emitLookupResult(ctx context.Context, out io.Writer, resultsField string, results []quarry.Reference, err error) {
	if err != nil {
		var ambiguous *quarry.ErrAmbiguousSymbol
		if errors.As(err, &ambiguous) {
			// output.Ok always returns 0, which SetExit would treat as a no-op
			// anyway; the exit code must be forced to 2 via a separate
			// SetExit call, exactly as the plan's exit-code-contract
			// decision specifies.
			output.Ok(out, map[string]any{"candidates": ambiguous.Candidates})
			SetExit(ctx, 2)
			return
		}

		// No other engine error type gets special-cased: ErrSymbolNotFound and
		// everything else fall through to output.Err's hardcoded exit 1, which
		// is already the design's "not found" contract value.
		SetExit(ctx, output.Err(out, err.Error()))
		return
	}

	// "resolution":"complete" is the machine-readable trust marker a caller
	// can key on to skip a redundant grep/re-verify pass: the language server
	// already resolved the query exhaustively, unlike a text-matched result.
	SetExit(ctx, output.Ok(out, map[string]any{resultsField: referenceFields(results), "resolution": "complete"}))
}

// referenceFields converts Reference results to JSON-encodable maps.
func referenceFields(refs []quarry.Reference) []map[string]any {
	fields := make([]map[string]any, len(refs))
	for i, r := range refs {
		fields[i] = map[string]any{
			"file":      r.File,
			"line":      r.Line,
			"character": r.Character,
		}
	}
	return fields
}

// parseQuery converts a string argument to a Query, parsing "file:line:col" positions or treating it as a symbol name.
// base must be an absolute path — the seam cwd, never the process cwd — against which a relative
// "file:line:col" path is resolved.
func parseQuery(base, arg string) (quarry.Query, error) {
	pos, ok := parsePosition(arg)
	if !ok {
		return quarry.Query{Symbol: arg}, nil
	}

	// quarry.Query.Pos.File must be an absolute path — References turns
	// it into a file:// URI directly, with no further resolution — so a relative
	// "file:line:col" argument is resolved against base here, the one point
	// where the CLI, not the engine, owns path interpretation. base is never
	// the process cwd, so this never falls back to filepath.Abs.
	pos.File = absOrJoin(base, pos.File)

	return quarry.Query{Pos: &pos}, nil
}

// inFileQuery converts a bare symbol name to an InFile Query, never position-parsed.
// base must be an absolute path — the seam cwd, never the process cwd — against which a relative
// --in-file path is resolved.
func inFileQuery(base, inFilePath, name string) (quarry.Query, error) {
	// quarry.InFileQuery.File must be an absolute path — References
	// turns it into a file:// URI directly, with no further resolution — so a
	// relative --in-file path is resolved against base here, exactly like
	// parseQuery resolves Pos.File: the CLI layer, not the engine, owns path
	// interpretation. base is never the process cwd, so this never falls back
	// to filepath.Abs.
	absFile := absOrJoin(base, inFilePath)

	return quarry.Query{InFile: &quarry.InFileQuery{File: absFile, Name: name}}, nil
}

// absOrJoin returns path unchanged if it is already absolute (cleaned), or joined onto base
// otherwise.
// base must itself be absolute; this is the one path-resolution rule the seam cwd governs for a
// caller-supplied argument, shared by parseQuery and inFileQuery.
func absOrJoin(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, path)
}

// parsePosition reports whether arg has the "file:line:col" shape and parses it.
func parsePosition(arg string) (quarry.Position, bool) {
	lastColon := strings.LastIndex(arg, ":")
	if lastColon < 0 {
		return quarry.Position{}, false
	}
	col, err := strconv.Atoi(arg[lastColon+1:])
	if err != nil {
		return quarry.Position{}, false
	}

	rest := arg[:lastColon]
	secondColon := strings.LastIndex(rest, ":")
	if secondColon < 0 {
		return quarry.Position{}, false
	}
	line, err := strconv.Atoi(rest[secondColon+1:])
	if err != nil {
		return quarry.Position{}, false
	}

	file := rest[:secondColon]
	if file == "" {
		return quarry.Position{}, false
	}

	return quarry.Position{File: file, Line: line, Character: col}, true
}

// batchStatus is the per-entry outcome in batch mode, shared by every verb's batch driver —
// runBatch's symbol-keyed entries and runPathBatch's (internal/cli/toc.go) path-keyed entries
// alike.
type batchStatus string

const (
	statusFound     batchStatus = "found"
	statusNotFound  batchStatus = "not_found"
	statusAmbiguous batchStatus = "ambiguous"
	statusError     batchStatus = "error"
)

// statusRank maps batchStatus values to exit-code ranks.
var statusRank = map[batchStatus]int{
	statusFound:     0,
	statusNotFound:  1,
	statusAmbiguous: 2,
	statusError:     3,
}

// classifyLookupError maps a References/Definition outcome to a batchStatus and JSON fields.
func classifyLookupError(err error, resultsField string, results []quarry.Reference) (batchStatus, map[string]any) {
	if err == nil {
		// Mirror emitLookupResult's single-arg "resolution":"complete" marker
		// per batch entry, so a batch-mode caller gets the same trust signal
		// on each "found" result the single-arg envelope carries.
		return statusFound, map[string]any{resultsField: referenceFields(results), "resolution": "complete"}
	}

	var ambiguous *quarry.ErrAmbiguousSymbol
	if errors.As(err, &ambiguous) {
		return statusAmbiguous, map[string]any{"candidates": ambiguous.Candidates}
	}

	if errors.Is(err, quarry.ErrSymbolNotFoundSentinel) {
		return statusNotFound, nil
	}

	return statusError, map[string]any{"error": err.Error()}
}

// classifySymbolError maps a Symbol outcome to a batchStatus and JSON fields.
func classifySymbolError(err error, results []quarry.SymbolMatch) (batchStatus, map[string]any) {
	if err == nil {
		return statusFound, map[string]any{"symbols": symbolMatchFields(results)}
	}

	if errors.Is(err, quarry.ErrSymbolNotFoundSentinel) {
		return statusNotFound, nil
	}

	return statusError, map[string]any{"error": err.Error()}
}

// runBatch drives batch mode, calling lookupOne per entry and reporting results with exit code matching the worst status.
func runBatch(ctx context.Context, out io.Writer, args []string, lookupOne func(symbol string) (batchStatus, map[string]any)) {
	entries := make([]map[string]any, len(args))
	worst := statusFound
	for i, arg := range args {
		status, fields := lookupOne(arg)
		if statusRank[status] > statusRank[worst] {
			worst = status
		}

		entry := map[string]any{"symbol": arg, "status": string(status)}
		for k, v := range fields {
			entry[k] = v
		}
		entries[i] = entry
	}

	output.Ok(out, map[string]any{"results": entries})
	if statusRank[worst] != 0 {
		SetExit(ctx, statusRank[worst])
	}
}

// RunCLI is the public seam for quarry's own CLI.
func RunCLI(out io.Writer, args []string) int {
	return RunCLIIn("", out, args)
}

// RunCLIIn is RunCLI's seam-cwd-carrying sibling: an empty cwd means "read the process cwd" and
// delegates to Execute exactly as RunCLI always has, while any other value seeds cwd into
// the execution context via ExecuteIn.
// The branch exists because WithCwd panics on an empty directory, so a uniform delegation to
// ExecuteIn would panic on every existing RunCLI call.
func RunCLIIn(cwd string, out io.Writer, args []string) int {
	if cwd == "" {
		return Execute(Command(), out, args)
	}
	return ExecuteIn(Command(), cwd, out, args)
}
