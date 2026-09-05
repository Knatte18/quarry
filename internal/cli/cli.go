// cli.go declares Run, the request pipeline every invocation of the quarry command executes below
// os.Exit, the exit-code constants that pipeline maps into, and fail, the one helper every failing
// step in the pipeline returns through so the failure envelope is written the same way from every
// call site.

package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/internal/repopath"
	"github.com/Knatte18/quarry/quarry"
)

// The four exit codes Run ever returns.
const (
	// exitOK means the query was answered: a directory or file target was found and rendered.
	exitOK = 0
	// exitNegative means the query itself has a negative answer: the target does not exist, or it
	// names a path outside the repository. This is not a usage error — the invocation was well
	// formed and the CLI ran it to a definite, negative conclusion.
	exitNegative = 1
	// exitUsage means the caller asked wrong: an unparseable flag, a missing or extra target, a
	// --root that does not resolve to a directory, or a toc target carrying the glyph grammar's
	// "#" separator in some path segment. TOC is never called on this path.
	exitUsage = 2
	// exitInternal means an I/O or render failure that has nothing to do with the query's answer:
	// an unexpected stat error, a working-directory read failure, or a write to stdout that itself
	// failed.
	exitInternal = 3
)

// fail is the single path every failing step of Run returns through, so the "stdout gets the
// envelope, stderr gets the same sentence" rule is written once and cannot be half-applied at some
// call sites and not others. It writes quarry.RenderErrorJSON(msg) to stdout — the JSON envelope
// always goes to stdout, including on failure and including under --text, so a pipeline parsing
// stdout finds a parseable object on every path — then writes msg followed by "\n" to stderr, and
// usageText to stderr as well when withUsage is true, and returns code.
//
// The stderr sentence is byte-identical to the envelope's error value so the two channels can
// never disagree about what went wrong. There is no text rendering of a failure: the payload is
// two fields with no prose in it, and a --text caller that must distinguish success from failure
// reads the exit code.
func fail(stdout, stderr io.Writer, code int, msg string, withUsage bool) int {
	_, _ = stdout.Write(quarry.RenderErrorJSON(msg))
	_, _ = io.WriteString(stderr, msg+"\n")
	if withUsage {
		_, _ = io.WriteString(stderr, usageText)
	}
	return code
}

// codeForTOCError is the pure mapping step 8 of Run's pipeline uses to turn the error (*Repo).TOC
// returns into an exit code. It is declared as a named function, rather than inlined into Run,
// precisely so discussion.md's "Exit-code mapping" test — a table from a returned error to the
// code, including the errors.Is-through-wrapping behaviour — can be written directly against it.
//
// It returns exitOK for a nil error, exitNegative when errors.Is(err, quarry.ErrTargetNotFound) or
// errors.Is(err, quarry.ErrTargetOutsideRepo), and exitInternal for anything else.
func codeForTOCError(err error) int {
	if err == nil {
		return exitOK
	}
	if errors.Is(err, quarry.ErrTargetNotFound) || errors.Is(err, quarry.ErrTargetOutsideRepo) {
		return exitNegative
	}
	return exitInternal
}

// codeForResolveResult is the pure mapping runResolve's pipeline uses to turn a resolve result
// into an exit code. It is declared as a named function, rather than inlined at the call site, so
// a table test can be written directly against it, mirroring codeForTOCError's own rationale.
//
// It returns exitOK for quarry.StatusFound and quarry.StatusMultipart, and exitNegative for
// quarry.StatusNotFound and quarry.StatusAmbiguous. The empty status also returns exitNegative,
// because an empty status means a pre-resolution rejection carried by the result's Error field,
// not an engine failure. The default returns exitInternal: the status vocabulary is closed, so
// this branch is unreachable, and it exists only so a value the engine never produces cannot
// silently route to a zero exit code.
func codeForResolveResult(r quarry.ResolveResult) int {
	switch r.Status {
	case quarry.StatusFound, quarry.StatusMultipart:
		return exitOK
	case quarry.StatusNotFound, quarry.StatusAmbiguous:
		return exitNegative
	case "":
		return exitNegative
	default:
		return exitInternal
	}
}

// codeForExpandAnswer is codeForResolveResult's counterpart for the expand verb's narrower status
// vocabulary, which admits only found, not_found, and ambiguous.
//
// It returns exitOK for quarry.StatusFound and exitNegative for quarry.StatusNotFound and
// quarry.StatusAmbiguous. The default returns exitInternal, unreachable for the same reason
// codeForResolveResult's default is.
func codeForExpandAnswer(a quarry.ExpandAnswer) int {
	switch a.Status {
	case quarry.StatusFound:
		return exitOK
	case quarry.StatusNotFound, quarry.StatusAmbiguous:
		return exitNegative
	default:
		return exitInternal
	}
}

// codeForExpandError maps the error the facade's Expand method returns into an exit code, so
// runExpand's error-branch classification and its exit code stay one table-tested source rather
// than two things that can drift apart.
//
// It returns exitOK for a nil error. When errors.As reaches a *quarry.NotATypeError, a
// *glyph.ParseError, or a *quarry.SelfGlyphError, it returns exitNegative — checked by type, never
// by parsing the error's own message. Anything else returns exitInternal, which is what routes the
// missing-head-span invariant failure, returned by the engine as a plain formatted error naming an
// invariant violation in the walk, to exit 3 with no message parsing anywhere.
func codeForExpandError(err error) int {
	if err == nil {
		return exitOK
	}
	var notType *quarry.NotATypeError
	if errors.As(err, &notType) {
		return exitNegative
	}
	var parseErr *glyph.ParseError
	if errors.As(err, &parseErr) {
		return exitNegative
	}
	var selfErr *quarry.SelfGlyphError
	if errors.As(err, &selfErr) {
		return exitNegative
	}
	return exitInternal
}

// codeForDeltaError is the pure mapping runDelta's pipeline uses to turn the error the facade's
// DeltaGit method returns into an exit code. It is declared as a named function, rather than
// inlined, following the convention the other three mapping functions in this file set, so a
// table test can be written against it, even though this one is nearly constant: a computed delta
// is always the success code, so codeForDeltaError only ever sees a non-nil error, and there is no
// exitNegative branch at all — this query has no negative answer, because nothing changed is a
// true answer to what changed rather than a negative one.
//
// It returns exitUsage when errors.Is(err, quarry.ErrUnknownRevision), errors.Is(err,
// quarry.ErrNotARepository), or errors.Is(err, quarry.ErrRootNotTopLevel). Anything else returns
// exitInternal, which is where a git command failing for any other reason lands.
func codeForDeltaError(err error) int {
	if err == nil {
		return exitOK
	}
	if errors.Is(err, quarry.ErrUnknownRevision) ||
		errors.Is(err, quarry.ErrNotARepository) ||
		errors.Is(err, quarry.ErrRootNotTopLevel) {
		return exitUsage
	}
	return exitInternal
}

// rootUsageMessage formats the CLI's own user-facing sentence for a repopath.ResolveRoot error,
// rather than propagating err.Error(), since repopath's sentinel text is namespaced to that
// package and would leak an internal package name into the CLI's contract. It returns the message
// and true when err wraps repopath.ErrNoRepositoryRoot or repopath.ErrRootNotDirectory, and
// ("", false) for any other error, including nil.
//
// It is a named function rather than an inline switch precisely so a table test can assert both
// sentences directly, without a fixture that cannot exist — the no-root case is unreachable from
// inside this repository without changing the process working directory, which these tests never
// do.
func rootUsageMessage(err error, flagRoot, cwd string) (string, bool) {
	if errors.Is(err, repopath.ErrNoRepositoryRoot) {
		return "no repository root found above " + cwd + "; pass --root", true
	}
	if errors.Is(err, repopath.ErrRootNotDirectory) {
		return "--root is not a directory: " + flagRoot, true
	}
	return "", false
}

// Run is the whole of the quarry command below os.Exit. args is os.Args[1:]. It executes the four
// steps every verb shares, then dispatches to that verb's own pipeline — the step that decides an
// exit code is the step that decides the message, so the two things are never allowed to drift
// apart within any one pipeline:
//
//  1. Parse flags and the verb. A usage error is exit 2; anything else is exit 3.
//  2. When help was requested, write usageText to stdout and return exit 0 — help is a successful
//     query about the CLI, not a usage error, so it never touches stderr or exit 2.
//  3. Read the working directory. This is the one place in the package that does.
//  4. Resolve the repository root by calling internal/repopath.ResolveRoot, --root or discovery,
//     then compute the base directory a relative target is interpreted against: the root when
//     --root was given, the working directory otherwise. A repopath error is translated to the
//     CLI's own usage sentence by rootUsageMessage, which keeps repopath's own namespaced sentinel
//     text out of the CLI's contract.
//
// Run then switches on req.verb and calls one of runTOC, runResolve, runExpand, or runDelta. The
// default case returns exitInternal with an internal-error message rather than falling through to
// a zero exit code; it is unreachable for every word other than the four verbs, because parseArgs
// already rejects any other verb as a usage error before Run ever sees it.
//
// runTOC's own pipeline, continuing from step 4 above:
//
//  1. Convert the target to a clean, repository-relative path with internal/repopath.RepoRelTarget.
//     Escaping the root is exit 1, named with the target exactly as given, since a target that
//     escaped the root has no meaningful repository-relative form. A target carrying the glyph
//     grammar's "#" separator in some path segment is exit 2, named "target contains the glyph
//     separator" followed by the target as given, with usageText following on stderr: that target
//     is malformed, not missing or out of scope.
//  2. Lstat the resolved target: not found is exit 1, named with the repository-relative path; any
//     other stat error is exit 3. Lstat, never Stat, so a symlink named as the target is treated as
//     a file and not followed, the same rule engine.resolveTarget already follows.
//  3. Open the repository.
//  4. Run the query, and map its error through codeForTOCError rather than an inline errors.Is
//     chain at this call site, so the mapping stays table-testable. Steps 1 and 2 have already
//     excluded both sentinel errors in the common case; these branches exist because the target can
//     be removed between the stat and the walk, and runTOC must not report that race as success.
//  5. Render: quarry.RenderText under --text, quarry.RenderJSON otherwise. A render error, or a
//     failed write of its bytes to stdout, is exit 3 — that is an I/O failure, which is what exit 3
//     already means.
//
// runResolve's own pipeline, continuing from step 4 above, performs no stat at all: the target not
// existing is the engine's own answer with a payload, and pre-empting it with the failure path
// would destroy exactly that answer.
//
//  1. Pass req.target to the facade verbatim: no classification, no path conversion, no rebasing,
//     no stat. The grammar is the only classifier, so a bare path with no "#" reaches
//     repo.Resolve exactly as any other malformed input does, and is answered by
//     resolveGlyphTarget's own pre-resolution rejection rather than by anything in this pipeline.
//  2. Open the repository. A failure is exit 3.
//  3. Call the facade's Resolve method with a one-element slice holding req.target. A non-nil
//     error is exit 3: an engine read failure is not an answer about a glyph.
//  4. A returned slice whose length is not exactly one is exit 3, named with the count — the facade
//     contracts a positional one-to-one mapping, so this is unreachable and is stated so a contract
//     change cannot silently produce a zero exit code.
//  5. Render the single result: quarry.RenderResolveText under --text, quarry.RenderResolveJSON
//     otherwise. A render error, or a failed write of its bytes to stdout, is exit 3. The payload is
//     written before the code is computed, in the next step, so a negative answer is rendered
//     rather than replaced by the failure envelope.
//  6. Return codeForResolveResult of that result.
//
// runExpand's own pipeline, continuing from step 4 above, takes no base directory and performs
// neither path conversion nor a stat: this verb accepts a glyph only, and the grammar itself,
// through glyph.Parse inside the facade, is what rejects a target that is not one.
//
//  1. Open the repository. A failure is exit 3.
//  2. Call the facade's Expand method with the target verbatim.
//  3. On a non-nil error, branch by type through errors.As, never by parsing the error's message:
//     a *quarry.NotATypeError is exit 1 with usage suppressed, named "expand " followed by the
//     value's identifier field, ": not a type, kind " and the value's kind field — quarry's own
//     sentence, spelled from the value's fields rather than the error's own text, so the engine's
//     package-name prefix never leaks through; a *glyph.ParseError is exit 1 with usage suppressed,
//     named "expand: " followed by the error's own Error() text, which already quotes the target
//     and names its rejection reason in full, rather than the reason word alone; a
//     *quarry.SelfGlyphError is exit 1 with usage suppressed, named "expand " followed by the
//     value's ID field and ": not a type, self" — spelled from the value's own field for the same
//     reason the NotATypeError branch is; anything else is exit 3 with the internal-error prefix,
//     which is where the missing-head-span invariant failure lands. All four route through
//     codeForExpandError for the code, so the mapping stays the single table-tested source.
//  4. On a nil error, render the answer: quarry.RenderExpandText under --text,
//     quarry.RenderExpandJSON otherwise. A render error, or a failed write of its bytes to stdout,
//     is exit 3.
//  5. Return codeForExpandAnswer of the answer.
//
// runDelta's own pipeline, continuing from step 4 above, converts the target through the shared
// repository-relative target helper first, exactly as runTOC's own pipeline does, so one argument
// cannot mean two things: quarry resolves a relative target against the caller's working
// directory while git would resolve a raw pathspec against the root. The consequence is that a
// lone dot means the current directory rather than the repository root when run from a
// subdirectory, identically to the table-of-contents verb, and that helper's two existing
// rejections carry over unchanged — a target escaping the root is the negative code, and a target
// carrying the glyph separator is a usage error with the usage text. This verb performs no stat on
// the target at any point, unlike the table-of-contents verb: a path that does not exist now may
// well have existed at the from revision, and a deleted directory is exactly the change this
// query exists to report; a pathspec matching nothing is a true, empty answer.
//
//  1. Convert the target the same way runTOC's own step 1 does.
//  2. Open the repository. A failure is exit 3.
//  3. Call the facade's DeltaGit method with req.from, req.to and the converted target. Map its
//     error through codeForDeltaError rather than an inline errors.Is chain at this call site, so
//     the mapping stays table-testable, mirroring codeForTOCError's own rationale. An
//     unresolvable revision, a root that is not a repository, and a root that is not that
//     repository's top level are each a usage error carrying quarry's own sentence, spelled from
//     the aliased typed error's own fields through type extraction rather than by parsing any
//     message, exactly as runExpand's own two sentences already are: the revision sentence names
//     the revision exactly as given, and the top-level sentence names both the root and the
//     top-level git reported. Any other failure is the internal code carrying the wrapped message
//     whole behind the existing internal-error prefix.
//  4. A computed delta is always the success code — including an empty delta and including a
//     batch in which some entries carry an error disposition, since either of those returning a
//     failure code would make a complete answer look like a failure to a shell gate.
//  5. Render: quarry.RenderDeltaText under --text, quarry.RenderDeltaJSON otherwise. A render
//     error, or a failed write of its bytes to stdout, is exit 3.
//
// The error value returned to a caller never carries the engine's wrapped chain for exit 1 or
// exit 2: those name conditions quarry itself defines, so quarry spells them, and passing
// something like `engine: resolve target "x": engine: target not found` through would leak an
// internal package name into a public contract. Exit 3 is the opposite case and carries
// err.Error() whole, behind the one "internal error: " prefix. Every message fail is called with
// is single-line; none embeds a newline. runExpand's *glyph.ParseError branch, above, is not a
// violation of this rule despite carrying the error's own Error() text whole: that text is
// composed by github.com/Knatte18/quarry/glyph, a public package the contract already names, not
// by internal/engine or another internal package, so quarry is still spelling the condition
// itself rather than leaking one of the internal names this rule guards against.
func Run(args []string, stdout, stderr io.Writer) int {
	req, err := parseArgs(args)
	if err != nil {
		var uerr usageError
		if errors.As(err, &uerr) {
			return fail(stdout, stderr, exitUsage, uerr.Error(), true)
		}
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	if req.help {
		_, _ = io.WriteString(stdout, usageText)
		return exitOK
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	root, err := repopath.ResolveRoot(req.root, cwd)
	if err != nil {
		if msg, ok := rootUsageMessage(err, req.root, cwd); ok {
			return fail(stdout, stderr, exitUsage, msg, true)
		}
		// Guards the case where repopath.ResolveRoot returns an error that is neither sentinel —
		// unreachable today, since DiscoverRoot and ResolveRoot only ever wrap
		// repopath.ErrNoRepositoryRoot or repopath.ErrRootNotDirectory. Stated anyway, so every
		// step of this pipeline spells both dispositions and a later change to that contract
		// cannot silently fall through to a zero exit code.
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	base := cwd
	if req.root != "" {
		base = root
	}

	switch req.verb {
	case "toc":
		return runTOC(req, root, base, stdout, stderr)
	case "resolve":
		return runResolve(req, root, base, stdout, stderr)
	case "expand":
		return runExpand(req, root, stdout, stderr)
	case "delta":
		return runDelta(req, root, base, stdout, stderr)
	default:
		// Unreachable for every word other than the four verbs parseArgs accepts: the parser
		// already rejects any other verb as a usage error before Run ever sees it.
		return fail(stdout, stderr, exitInternal, "internal error: unknown verb: "+req.verb, false)
	}
}

// runTOC is the table-of-contents verb's own pipeline, continuing from Run's shared four steps.
// Its behaviour is unchanged from Run's own body before this pipeline was extracted from it; see
// Run's doc comment for the numbered steps.
func runTOC(req request, root, base string, stdout, stderr io.Writer) int {
	rel, err := repopath.RepoRelTarget(root, base, req.target)
	if err != nil {
		if errors.Is(err, quarry.ErrTargetOutsideRepo) {
			return fail(stdout, stderr, exitNegative, "target outside repository: "+req.target, false)
		}
		if errors.Is(err, quarry.ErrTargetHasSeparator) {
			return fail(stdout, stderr, exitUsage, "target contains the glyph separator \"#\": "+req.target, true)
		}
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	if os.IsNotExist(err) {
		return fail(stdout, stderr, exitNegative, "target not found: "+rel, false)
	}
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	targetIsFile := !info.IsDir()

	repo, err := quarry.Open(root)
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	answer, err := repo.TOC(rel, quarry.TOCOptions{Depth: req.depth, Symbols: req.symbols})
	// The two sentinel branches below are race-only in the common case: the stat above has already
	// excluded both errors, so they fire only when the target is removed between that stat and the
	// engine's own walk. Reporting that race as exitOK would be a false positive.
	if code := codeForTOCError(err); code != exitOK {
		switch code {
		case exitNegative:
			if errors.Is(err, quarry.ErrTargetOutsideRepo) {
				return fail(stdout, stderr, exitNegative, "target outside repository: "+req.target, false)
			}
			return fail(stdout, stderr, exitNegative, "target not found: "+rel, false)
		default:
			return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
		}
	}

	if req.text {
		if _, err := io.WriteString(stdout, quarry.RenderText(answer, targetIsFile)); err != nil {
			return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
		}
		return exitOK
	}

	out, err := quarry.RenderJSON(answer)
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	if _, err := stdout.Write(out); err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	return exitOK
}

// runResolve is the resolve verb's own pipeline, continuing from Run's shared four steps. See
// Run's doc comment for the numbered steps this function executes in fixed order.
func runResolve(req request, root, base string, stdout, stderr io.Writer) int {
	repo, err := quarry.Open(root)
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	results, err := repo.Resolve([]string{req.target})
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	if len(results) != 1 {
		return fail(stdout, stderr, exitInternal, "internal error: resolve returned "+strconv.Itoa(len(results))+" results for one target", false)
	}
	result := results[0]

	if req.text {
		if _, err := io.WriteString(stdout, quarry.RenderResolveText(result)); err != nil {
			return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
		}
		return codeForResolveResult(result)
	}

	out, err := quarry.RenderResolveJSON(result)
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	if _, err := stdout.Write(out); err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	return codeForResolveResult(result)
}

// runExpand is the expand verb's own pipeline, continuing from Run's shared four steps. It takes
// no base directory, because this verb accepts a glyph only and performs no path work at all. See
// Run's doc comment for the numbered steps this function executes in fixed order.
func runExpand(req request, root string, stdout, stderr io.Writer) int {
	repo, err := quarry.Open(root)
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	answer, err := repo.Expand(req.target)
	if err != nil {
		var notType *quarry.NotATypeError
		if errors.As(err, &notType) {
			msg := "expand " + notType.ID + ": not a type, kind " + string(notType.Kind)
			return fail(stdout, stderr, codeForExpandError(err), msg, false)
		}
		var parseErr *glyph.ParseError
		if errors.As(err, &parseErr) {
			msg := "expand: " + parseErr.Error()
			return fail(stdout, stderr, codeForExpandError(err), msg, false)
		}
		var selfErr *quarry.SelfGlyphError
		if errors.As(err, &selfErr) {
			msg := "expand " + selfErr.ID + ": not a type, self"
			return fail(stdout, stderr, codeForExpandError(err), msg, false)
		}
		return fail(stdout, stderr, codeForExpandError(err), "internal error: "+err.Error(), false)
	}

	if req.text {
		if _, err := io.WriteString(stdout, quarry.RenderExpandText(answer)); err != nil {
			return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
		}
		return codeForExpandAnswer(answer)
	}

	out, err := quarry.RenderExpandJSON(answer)
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	if _, err := stdout.Write(out); err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	return codeForExpandAnswer(answer)
}

// runDelta is the delta verb's own pipeline, continuing from Run's shared four steps. See Run's
// doc comment for the numbered steps this function executes in fixed order.
func runDelta(req request, root, base string, stdout, stderr io.Writer) int {
	rel, err := repopath.RepoRelTarget(root, base, req.target)
	if err != nil {
		if errors.Is(err, quarry.ErrTargetOutsideRepo) {
			return fail(stdout, stderr, exitNegative, "target outside repository: "+req.target, false)
		}
		if errors.Is(err, quarry.ErrTargetHasSeparator) {
			return fail(stdout, stderr, exitUsage, "target contains the glyph separator \"#\": "+req.target, true)
		}
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	repo, err := quarry.Open(root)
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}

	answer, err := repo.DeltaGit(req.from, req.to, rel)
	if err != nil {
		var revErr *quarry.UnknownRevisionError
		if errors.As(err, &revErr) {
			msg := "delta: unknown revision " + revErr.Rev
			return fail(stdout, stderr, codeForDeltaError(err), msg, false)
		}
		var topErr *quarry.RootNotTopLevelError
		if errors.As(err, &topErr) {
			msg := "delta: root " + topErr.Root + " is not the repository top level (top level is " + topErr.TopLevel + ")"
			return fail(stdout, stderr, codeForDeltaError(err), msg, false)
		}
		if errors.Is(err, quarry.ErrNotARepository) {
			msg := "delta: root is not a git repository: " + root
			return fail(stdout, stderr, codeForDeltaError(err), msg, false)
		}
		return fail(stdout, stderr, codeForDeltaError(err), "internal error: "+err.Error(), false)
	}

	if req.text {
		if _, err := io.WriteString(stdout, quarry.RenderDeltaText(answer)); err != nil {
			return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
		}
		return exitOK
	}

	out, err := quarry.RenderDeltaJSON(answer)
	if err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	if _, err := stdout.Write(out); err != nil {
		return fail(stdout, stderr, exitInternal, "internal error: "+err.Error(), false)
	}
	return exitOK
}
