// impact.go implements the "impact" subcommand: every caller of a symbol, each paired with its
// own enclosing declaration's identity and line range, via quarry.Impact.
// It also holds the three impact-typed helpers refsCommand's emitLookupResult/classifyLookupError
// have no counterpart for, because impact's result is its own struct rather than
// []quarry.Reference: emitImpactResult, classifyImpactError, and filterImpactWithin.

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/internal/output"
	"github.com/Knatte18/quarry/quarry"
)

// impactCommand builds the "impact" subcommand.
func impactCommand() *cobra.Command {
	var targetDir string
	var lang string
	var timeout time.Duration
	var inFile string
	var within string
	var buildTags string

	impact := &cobra.Command{
		Use:   "impact <symbol|file:line:col>",
		Short: "show every caller of a symbol, each paired with its enclosing declaration",
		Long: `impact shows every caller of a symbol name or an explicit source position, using the LSP
"textDocument/references" request against the language server detected for --target-dir (or
--lang, to override detection), and pairs each caller with its own enclosing declaration — kind,
name, owner, package, signature, and line range — resolved via quarry's tree-sitter-backed table of
contents.

The single positional argument is either:
  - a symbol name, resolved via the language server's workspace/symbol search:
      quarry impact MyFunction
  - an explicit "file:line:col" position (1-based line and column), bypassing
    name resolution entirely:
      quarry impact internal/foo/bar.go:42:8

--in-file <path> resolves each positional argument as a bare symbol name
within exactly that one file, via an exhaustive textDocument/documentSymbol
search rather than a project-wide workspace/symbol search — the positional
is always treated as a bare name, never position-parsed, even if it happens
to look like "file:line:col":
    quarry impact --in-file internal/foo/bar.go MyFunc

Passing 2 or more positional arguments switches to batch mode: each argument
is looked up independently and the results are reported as one array, rather
than the single-symbol envelope above:
    {"ok":true,"results":[{"symbol":...,"status":"found"|"not_found"|"ambiguous"|"error",...}, ...]}
The process exit code is set to the worst status present across the batch
(0 < 1 < 2 < 3). Example:
    quarry impact Foo Bar Baz
--in-file composes with batch mode too, resolving every positional against
the same file:
    quarry impact --in-file internal/foo/bar.go Open Close

A successful lookup's envelope carries "target" (the resolved symbol's own identity), "definition"
(its declaration site, plus its own enclosing declaration's range when one was found), and
"callers" (every verified call site). "target" and "definition" are present together only when the
query resolved to a real declaration. Both "definition" and each caller's "enclosing_range" include
the docstring immediately preceding the declaration in their line range, mirroring "toc file"'s
"start". A caller entry with no "enclosing_range" and no "error" is a file-scope reference — its
line has no listable declaration covering it, a correct answer rather than a failed lookup; a
caller entry carrying an "error" is one whose own file could not be parsed at all. "callers" is
always present as a non-nil array — "[]", never "null", when the symbol has no callers — and a
symbol with no callers still exits 0.

"resolution":"complete" means the language server returned every reference for the query as given —
the same meaning it already carries on "refs" — and asserts nothing about per-caller verification
having run, nor about any caller's enclosing range having resolved. --timeout covers the LSP
request phases only (initialize, resolve, references/definition/implementation); it never bounds
the tree-sitter parse phase that resolves each caller's and the target's own enclosing declaration.

Known limitation, and what --within is for: gopls' references for an
interface method conservatively include every method matching that
name+signature across every structurally-compatible interface anywhere in
the workspace — not just calls through the specific interface value you
queried. This is documented gopls behavior, not a bug in this wrapper, and it
means two unrelated, identically-shaped interfaces in different packages
(e.g. two local "type clock interface { Now() time.Time }" declarations)
conflate their results by default — an unverified caller set is precisely
the noisy case --within exists for. --within <dir> restricts the reported
"callers" — never "target" or "definition", which stay unfiltered — to
those whose file lies within <dir> (relative to --target-dir, or absolute):
    quarry impact --within internal/websterengine SomeMethod`,
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
			buildTagsResolved := ResolveBuildTags(buildTags)
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
					return inFileQuery(dir, inFile, arg)
				}
				return parseQuery(dir, arg)
			}

			if len(args) == 1 {
				query, err := buildQuery(args[0])
				if err != nil {
					SetExit(ctx, output.Err(out, err.Error()))
					return nil
				}

				opts := buildOptions(registry, dir, stateDir, lang, query, timeout, buildTagsResolved)

				result, err := quarry.Impact(ctx, opts)
				if err == nil && within != "" {
					result = filterImpactWithin(result, within, dir)
				}
				emitImpactResult(ctx, out, result, err)
				return nil
			}

			runBatch(ctx, out, args, func(symbol string) (batchStatus, map[string]any) {
				query, err := buildQuery(symbol)
				if err != nil {
					return statusError, map[string]any{"error": err.Error()}
				}
				result, err := quarry.Impact(ctx, buildOptions(registry, dir, stateDir, lang, query, timeout, buildTagsResolved))
				if err == nil && within != "" {
					result = filterImpactWithin(result, within, dir)
				}
				return classifyImpactError(err, result)
			})
			return nil
		},
	}

	impact.Flags().StringVar(&targetDir, "target-dir", "", "project directory to detect the language in and root the server at (default: cwd)")
	impact.Flags().StringVar(&lang, "lang", "", "override language detection with this registry key")
	impact.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "deadline for each LSP request phase (initialize, resolve, references/definition)")
	impact.Flags().StringVar(&inFile, "in-file", "", "resolve each positional argument as a bare symbol name within this one file, instead of a project-wide workspace/symbol search")
	impact.Flags().StringVar(&within, "within", "", "restrict the reported callers to those whose file lies within this directory (relative to --target-dir, or absolute) — target and definition are unaffected — see the interface-method conflation note above")
	impact.Flags().StringVar(&buildTags, "build-tags", "", "comma-separated Go build tags to scope the query to (default: $QUARRY_BUILD_TAGS, or none); an error, not a silent no-op, for a language whose registry entry carries no build-tag template")

	return impact
}

// emitImpactResult maps an Impact outcome to the output envelope, reproducing emitLookupResult's
// error routing and order for the incoming err argument: an *quarry.ErrAmbiguousSymbol matched via
// errors.As emits "candidates" through output.Ok and forces exit 2; every other error (including
// quarry.ErrSymbolNotFoundSentinel, which is deliberately not special-cased here either) falls
// through to output.Err's hardcoded exit 1.
// On success, result is marshalled through structToFields and "resolution":"complete" is added to
// the returned map before it is emitted through output.Ok. A structToFields failure is itself an
// output.Err exit 1, but reworded off structToFields' own "toc: " prefix — see
// rewordImpactMarshalFailure — so this verb's error envelope never names a verb the caller never
// invoked.
func emitImpactResult(ctx context.Context, out io.Writer, result quarry.ImpactResult, err error) {
	if err != nil {
		var ambiguous *quarry.ErrAmbiguousSymbol
		if errors.As(err, &ambiguous) {
			// output.Ok always returns 0, which SetExit would treat as a no-op
			// anyway; the exit code must be forced to 2 via a separate
			// SetExit call, exactly as emitLookupResult does.
			output.Ok(out, map[string]any{"candidates": ambiguous.Candidates})
			SetExit(ctx, 2)
			return
		}

		SetExit(ctx, output.Err(out, err.Error()))
		return
	}

	fields, marshalErr := structToFields(result)
	if marshalErr != nil {
		SetExit(ctx, output.Err(out, rewordImpactMarshalFailure(marshalErr)))
		return
	}
	fields["resolution"] = "complete"

	SetExit(ctx, output.Ok(out, fields))
}

// classifyImpactError maps an Impact outcome to a batchStatus and JSON fields, the batch-mode
// counterpart to emitImpactResult. It has three incoming-error branches — one more than
// emitImpactResult's two — because batch mode must distinguish "not_found" from "error" in the
// status vocabulary where the single-argument shape collapses both onto exit 1:
// *quarry.ErrAmbiguousSymbol via errors.As routes to statusAmbiguous with "candidates";
// quarry.ErrSymbolNotFoundSentinel via errors.Is routes to statusNotFound with no extra fields;
// anything else routes to statusError with an "error" field. No fourth branch is added to that
// incoming-error routing and the three are never reordered.
// The nil-error branch yields statusFound carrying the marshalled result and the same
// "resolution":"complete" marker every found entry gets — unless structToFields itself fails, in
// which case that same nil-error branch returns statusError with the reworded message under an
// "error" key. This is not a fourth error branch: it is the same disposition tocFileOne already
// uses for exactly this failure, inside its own nil-error branch, so batch mode's behaviour on a
// marshal failure matches the existing verbs rather than being invented here.
func classifyImpactError(err error, result quarry.ImpactResult) (batchStatus, map[string]any) {
	if err != nil {
		var ambiguous *quarry.ErrAmbiguousSymbol
		if errors.As(err, &ambiguous) {
			return statusAmbiguous, map[string]any{"candidates": ambiguous.Candidates}
		}

		if errors.Is(err, quarry.ErrSymbolNotFoundSentinel) {
			return statusNotFound, nil
		}

		return statusError, map[string]any{"error": err.Error()}
	}

	fields, marshalErr := structToFields(result)
	if marshalErr != nil {
		return statusError, map[string]any{"error": rewordImpactMarshalFailure(marshalErr)}
	}
	fields["resolution"] = "complete"
	return statusFound, fields
}

// rewordImpactMarshalFailure reworks a structToFields failure into an impact-specific marshalling
// message. structToFields wraps both of its failure modes with a literal "toc: " prefix, because it
// was written for the toc verbs; that prefix is stripped and replaced with "impact: " here so the
// single-argument (emitImpactResult) and batch (classifyImpactError) shapes carry the identical,
// correctly-attributed message rather than naming a verb the caller never invoked.
func rewordImpactMarshalFailure(err error) string {
	return fmt.Sprintf("impact: %s", strings.TrimPrefix(err.Error(), "toc: "))
}

// filterImpactWithin filters result's caller list to entries whose file lies within within
// (relative to baseDir, or absolute), leaving Target and Definition untouched — --within is a CLI
// flag with no engine option behind it, so quarry.Impact itself is unfiltered.
// within is normalized exactly as FilterWithin normalizes its own within argument — joined onto
// baseDir when relative, then filepath.Abs, then filepath.Clean — before isWithinDir is called per
// entry: every compared path is absolute, so skipping this normalization would make filepath.Rel
// error inside isWithinDir and silently filter every caller out, producing an empty-but-successful
// answer.
// The returned Callers slice is always non-nil, even when nothing survives, so it still marshals as
// "[]".
func filterImpactWithin(result quarry.ImpactResult, within, baseDir string) quarry.ImpactResult {
	w := within
	if !filepath.IsAbs(w) {
		w = filepath.Join(baseDir, w)
	}
	// baseDir itself may still be relative here (e.g. --target-dir "."
	// passed through verbatim) — filepath.Abs resolves whatever remains
	// against the process's actual working directory, mirroring
	// FilterWithin's identical fallback.
	if abs, err := filepath.Abs(w); err == nil {
		w = abs
	}
	w = filepath.Clean(w)

	filtered := make([]quarry.ImpactCaller, 0, len(result.Callers))
	for _, c := range result.Callers {
		if isWithinDir(w, filepath.Clean(c.File)) {
			filtered = append(filtered, c)
		}
	}

	result.Callers = filtered
	return result
}
