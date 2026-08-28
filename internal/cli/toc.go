// toc.go implements the toc command group ("toc file" and "toc dir") and everything the two
// subcommands share: language validation, path resolution, result marshalling, error
// classification, and the path-keyed batch driver.
//
// toc deliberately bypasses resolveContext and buildOptions. Those helpers load a servers.yaml-
// backed registry.Registry and resolve a daemon state directory — both exist for the language-
// server-backed verbs (refs/definition/symbol/assert-no-callers), which toc has no need of: it
// needs no language server and no daemon state directory. Forcing toc through that machinery
// would make it fail on any machine where no servers.yaml resolves, for a dependency toc never
// actually uses.

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/internal/output"
	"github.com/Knatte18/quarry/quarry"
)

// tocCommand builds the "toc" parent command group.
func tocCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toc",
		Short: "table-of-contents extraction for a single file or a directory",
		RunE:  GroupRunE,
	}

	cmd.AddCommand(tocFileCommand())
	cmd.AddCommand(tocDirCommand())
	return cmd
}

// tocFileCommand builds the "toc file" subcommand.
func tocFileCommand() *cobra.Command {
	var lang string
	var docSentences string

	cmd := &cobra.Command{
		Use:   "file <path>",
		Short: "extract a table of contents for a single file",
		Long: `toc file extracts a table of contents for a single source file: its package or
namespace, its first-paragraph header comment, and every top-level function, method, and type
declaration it contains.

Each symbol entry carries three 1-based, inclusive line numbers: "start", "sigend", and "end".
"start" is the first line of the symbol's docstring when it has one (or its declaration otherwise);
"end" is the last line of the whole declaration, body included. "sigend" is the last line of the
signature alone — the line the body begins on. Read "start" through "sigend" to see a symbol's
docstring plus signature, enough to judge relevance without reading its implementation; read
"start" through "end" to read the whole symbol, docstring and body together. "sigend" is omitted
for a symbol with no body at all, such as a Go type alias.

--lang overrides language detection with an explicit language name, validated against toc's own
supported set — never the servers.yaml-backed registry key --lang means on refs/definition/symbol.

Passing 2 or more positional arguments switches to batch mode: each path is looked up
independently and the results are reported as one array under "results", each entry keyed by
"path" rather than by symbol name, with a per-entry "status" of "found", "not_found", or "error".
The process exit code is set to the worst status present across the batch.

Recommended two-phase discovery flow for an unfamiliar file, with its measured cost on a real
1186-line file holding 40 functions and methods:

  1. quarry toc file --doc-sentences 0 <path> — the map: every symbol's name, signature, and line
     ranges, with no docstring text at all. 8.4 KB for that file.
  2. Read "start" through "sigend" for the few candidates that look relevant, directly from the
     source file — roughly 6 lines each.
  3. Read "start" through "end" for the one that matched, directly from the source file — roughly
     16 lines.

Against reading the whole 1186-line file.

The decisive point is not the byte count: the prose in steps 2 and 3 is read from the source file
itself, never from quarry's rendering of it. The agent never has to trust quarry's "//" and "///"
stripping, its C# XML tag removal, or its sentence-splitting rule — it sees the actual bytes.
--doc-sentences 0 is therefore the recommended discovery mode, not a frugality mode; treating it as
merely a way to save bytes makes it easy to skip.

"start" never moves with --doc-sentences, so the ranges from step 1 stay valid for the read in
steps 2 and 3 regardless of how much docstring text was requested there.

Known imprecision: in a single-line function or method, the signature and the body share one line,
so "start" through "sigend" includes the body — no line-based range can separate them there.

--doc-sentences <N|all> controls how much of each symbol's docstring reaches "docstring" in the
output: 0 omits the key entirely (the discovery mode above); N keeps the first N sentences; "all"
keeps the docstring unchanged. The effective value is resolved highest-precedence-first: the
--doc-sentences flag on this command; then $QUARRY_TOC_CONFIG, an absolute path to a config file;
then a ".quarry.yaml" in the target file's own directory, holding a "toc" mapping with a
"doc_sentences" key — looked up in that directory alone, with no upward search; then the built-in
default of 1.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			cwd, err := CwdFrom(ctx)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			if err := validateTOCLang(lang); err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			// The flag tier is hoisted here, parsed once before the
			// single-argument branch and the batch branch diverge: a
			// non-empty flag value short-circuits the per-argument config
			// file work entirely, which is what keeps the common case from
			// re-reading a config file once per argument. An invalid flag
			// value therefore fails once, up front, before any argument is
			// processed.
			if docSentences != "" {
				if _, err := parseDocSentences(docSentences); err != nil {
					SetExit(ctx, output.Err(out, err.Error()))
					return nil
				}
			}

			if len(args) > 1 {
				runPathBatch(ctx, out, args, func(arg string) (batchStatus, map[string]any) {
					return tocFileOne(cwd, arg, lang, docSentences)
				})
				return nil
			}

			status, fields := tocFileOne(cwd, args[0], lang, docSentences)
			if status != statusFound {
				msg, _ := fields["error"].(string)
				SetExit(ctx, output.Err(out, msg))
				return nil
			}
			SetExit(ctx, output.Ok(out, fields))
			return nil
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "override language detection with this language name (validated against toc's own supported set)")
	// --doc-sentences is registered on "toc file" only: "toc dir" emits
	// headers and never docstrings, so the setting has nothing to affect
	// there.
	cmd.Flags().StringVar(&docSentences, "doc-sentences", "", "number of leading docstring sentences to emit, or \"all\" (default: 1, or the resolved config value)")
	return cmd
}

// tocDirCommand builds the "toc dir" subcommand.
func tocDirCommand() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "dir <path>",
		Short: "extract a table of contents for every supported file in a directory",
		Long: `toc dir lists every file directly inside the given directory whose extension maps to a
supported language — never recursing into subdirectories — and, for each, its package or
namespace, its first-paragraph header comment, and whether it is a test file or a generated file
when the language has a reliable rule for either. A file this cannot parse at all (unreadable,
invalid UTF-8, or a designed-but-unimplemented language) is still listed, with an "error" field in
place of the usual fields; the directory listing itself never fails because of one bad file.

Each entry's "path" is the directory argument as the caller wrote it, joined with the file's base
name — never the absolutised form — so an entry can be pasted straight into "quarry toc file" from
the same working directory.

--lang restricts the listing to that language's own extensions, validated against toc's own
supported set. Naming a designed-but-unimplemented language (e.g. --lang rust) is not an error
here: every matching file is still listed, each carrying the "language not yet supported" error in
its own entry — the flag selects which files to list, and an unimplemented language is a reported
limitation, not a failure of the listing.

Passing 2 or more positional arguments switches to batch mode: each directory is looked up
independently and the results are reported as one array under "results", each entry keyed by
"path" rather than by symbol name, with a per-entry "status" of "found", "not_found", or "error".
The process exit code is set to the worst status present across the batch.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			cwd, err := CwdFrom(ctx)
			if err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			if err := validateTOCLang(lang); err != nil {
				SetExit(ctx, output.Err(out, err.Error()))
				return nil
			}

			if len(args) > 1 {
				runPathBatch(ctx, out, args, func(arg string) (batchStatus, map[string]any) {
					return tocDirOne(cwd, arg, lang)
				})
				return nil
			}

			status, fields := tocDirOne(cwd, args[0], lang)
			if status != statusFound {
				msg, _ := fields["error"].(string)
				SetExit(ctx, output.Err(out, msg))
				return nil
			}
			SetExit(ctx, output.Ok(out, fields))
			return nil
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "restrict the listing to this language's extensions (validated against toc's own supported set)")
	return cmd
}

// validateTOCLang reports an error naming toc's own valid --lang values when lang is non-empty and
// not one of them; a nil error means either lang is empty (no override requested) or lang is
// valid.
//
// This validates against quarry.TOCLanguages() — toc's own vocabulary — rather than against the
// servers.yaml-backed registry the existing verbs' --lang validates against inside resolveContext.
// toc never calls resolveContext (see this file's header comment) and so never loads that
// registry; validating against it here would require loading it just for this one check, defeating
// the reason toc bypasses resolveContext in the first place.
func validateTOCLang(lang string) error {
	if lang == "" {
		return nil
	}
	for _, l := range quarry.TOCLanguages() {
		if l == lang {
			return nil
		}
	}
	return fmt.Errorf("toc: unrecognised --lang %q; valid languages: %s", lang, strings.Join(quarry.TOCLanguages(), ", "))
}

// resolveTOCPath joins arg onto cwd unless arg is already absolute, mirroring absOrJoin's
// resolution rule for a toc positional argument.
func resolveTOCPath(cwd, arg string) string {
	if filepath.IsAbs(arg) {
		return filepath.Clean(arg)
	}
	return filepath.Join(cwd, arg)
}

// tocFileOne resolves arg (a "toc file" positional argument) against cwd, validates it names an
// existing, non-directory path, resolves the effective DocSentences value against arg's own
// parent directory, calls quarry.TOCFile, and returns the batch outcome.
//
// Both the single-argument RunE above and runPathBatch's per-argument closure call this function,
// so the two call paths cannot drift apart.
//
// Resolution is per argument, not per invocation: the setting is per-directory and a batch may
// span directories, so each argument resolves the config-file tier against its own parent
// directory. When docSentences is non-empty, resolveDocSentences short-circuits before touching
// any config file — the caller has already validated the flag once, up front, before any
// argument was processed.
func tocFileOne(cwd, arg, lang, docSentences string) (batchStatus, map[string]any) {
	abs := resolveTOCPath(cwd, arg)

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return statusNotFound, map[string]any{"error": err.Error()}
		}
		return statusError, map[string]any{"error": err.Error()}
	}
	if info.IsDir() {
		return statusError, map[string]any{"error": fmt.Sprintf("toc: %s is a directory; use %q for a directory listing", arg, "quarry toc dir")}
	}

	targetDir := filepath.Dir(abs)
	resolvedDocSentences, err := resolveDocSentences(docSentences, targetDir)
	if err != nil {
		return statusError, map[string]any{"error": err.Error()}
	}

	result, err := quarry.TOCFile(abs, lang, quarry.TOCOptions{DocSentences: resolvedDocSentences})
	if err != nil {
		status, msg := classifyTOCError(err)
		return status, map[string]any{"error": msg}
	}

	fields, err := structToFields(result)
	if err != nil {
		return statusError, map[string]any{"error": err.Error()}
	}
	return statusFound, fields
}

// tocDirOne resolves arg (a "toc dir" positional argument) against cwd, validates it names an
// existing directory, calls quarry.TOCDir, composes each listed file's caller-relative "path" via
// tocDirEntries, and returns the batch outcome.
//
// Both the single-argument RunE above and runPathBatch's per-argument closure call this function,
// so the two call paths cannot drift apart.
func tocDirOne(cwd, arg, lang string) (batchStatus, map[string]any) {
	abs := resolveTOCPath(cwd, arg)

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return statusNotFound, map[string]any{"error": err.Error()}
		}
		return statusError, map[string]any{"error": err.Error()}
	}
	if !info.IsDir() {
		return statusError, map[string]any{"error": fmt.Sprintf("toc: %s is a file; use %q for a single file", arg, "quarry toc file")}
	}

	result, err := quarry.TOCDir(abs, lang)
	if err != nil {
		status, msg := classifyTOCError(err)
		return status, map[string]any{"error": msg}
	}

	files, err := tocDirEntries(arg, result)
	if err != nil {
		return statusError, map[string]any{"error": err.Error()}
	}
	return statusFound, map[string]any{"files": files}
}

// tocDirEntries builds the []any "files" entries "toc dir" emits for one directory argument as the
// caller wrote it (never the absolutised form), so each entry's composed "path" round-trips
// straight into a follow-up "toc file" call from the same working directory.
//
// quarry.TOCDirResult.Files carries each file's base name, but DirEntry.Name is tagged json:"-"
// specifically because internal/cli, not the engine, composes the caller-relative path — so after
// structToFields marshals result through encoding/json, the decoded files no longer carry it. This
// function zips the decoded array back to result.Files by index (the marshal preserves slice
// order, so element i of the decoded array is result.Files[i]) and injects "path" from the typed
// entry while walking the pair, asserting the two lengths match before the loop rather than
// indexing into a shorter slice.
//
// Both "toc dir"'s single-argument RunE and runPathBatch's per-argument closure (via tocDirOne)
// call this one helper, so a multi-argument "toc dir a b" cannot silently produce a "files" array
// with no per-file "path" — the one thing the batch path cannot inherit from runPathBatch itself,
// which only ever writes the entry-level "path" key (the argument), never the per-file path inside
// "files".
func tocDirEntries(arg string, result quarry.TOCDirResult) ([]any, error) {
	fields, err := structToFields(result)
	if err != nil {
		return nil, err
	}

	rawFiles, _ := fields["files"].([]any)
	if len(rawFiles) != len(result.Files) {
		return nil, fmt.Errorf("toc: internal error: marshalled %d files but the typed result has %d", len(rawFiles), len(result.Files))
	}

	for i, raw := range rawFiles {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("toc: internal error: files[%d] did not marshal to a JSON object", i)
		}
		entry["path"] = filepath.Join(arg, result.Files[i].Name)
	}
	return rawFiles, nil
}

// structToFields re-marshals v — a struct carrying the json tags that fix toc's emitted key set —
// into a map[string]any via encoding/json, so output.Ok's map[string]any parameter is fed exactly
// the keys the struct tags define and every omitempty rule is honoured in this one place rather
// than restated by hand at each call site.
func structToFields(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("toc: marshal result: %w", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("toc: unmarshal result: %w", err)
	}
	return fields, nil
}

// classifyTOCError maps an engine toc error to a batchStatus and a CLI-facing message. It is the
// one place an engine toc error becomes a CLI outcome, shared by "toc file", "toc dir", and the
// per-argument closures runPathBatch drives for both.
//
// The returned status is statusError in every branch, deliberately: found/not_found/ambiguous/
// error is the closed vocabulary the four pre-existing verbs already share, and an unsupported
// language is an error, not a fifth kind of outcome. The distinction a caller needs — "quarry
// cannot read this language at all" versus "quarry failed to read this particular file" — lives
// entirely in the message this function returns, not in the status.
//
// The sentinel branch is detected with errors.Is rather than a string comparison against
// err.Error(), because every real caller wraps quarry.ErrLanguageUnsupported (toc's own
// resolveLanguage always returns it via fmt.Errorf("...: %w", ...)); a == comparison would pass an
// artificial unwrapped test case while failing to match the wrapped error every real caller
// actually produces.
func classifyTOCError(err error) (batchStatus, string) {
	if errors.Is(err, quarry.ErrLanguageUnsupported) {
		// TOCImplemented(), not TOCLanguages(): this message answers "what can
		// quarry actually read", and naming the full designed set here would
		// list a language like rust as available in the very error saying it
		// is not.
		return statusError, fmt.Sprintf("toc: language not yet supported; quarry can currently read: %s", strings.Join(quarry.TOCImplemented(), ", "))
	}
	return statusError, err.Error()
}

// runPathBatch is toc's own batch driver, a near-copy of runBatch with exactly one line
// different: the per-entry identity key is "path" rather than "symbol", and its value is the
// positional argument echoed verbatim, exactly as runBatch echoes its own "symbol" key. It reuses
// the existing batchStatus constants and the statusRank map unchanged — the shared status
// vocabulary and ranking are the same across every verb; only the identity key differs for toc.
//
// runBatch itself is deliberately not generalized to take a configurable key. Its "symbol" key is
// part of the output shape all four pre-existing verbs already emit, and parameterizing it would
// change them. This duplication is a deliberate choice, not an oversight for a later cleanup to
// collapse the two functions into one.
func runPathBatch(ctx context.Context, out io.Writer, args []string, lookupOne func(path string) (batchStatus, map[string]any)) {
	entries := make([]map[string]any, len(args))
	worst := statusFound
	for i, arg := range args {
		status, fields := lookupOne(arg)
		if statusRank[status] > statusRank[worst] {
			worst = status
		}

		entry := map[string]any{"path": arg, "status": string(status)}
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
