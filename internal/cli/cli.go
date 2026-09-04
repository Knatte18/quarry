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
	// exitUsage means the caller asked wrong: an unparseable flag, a missing or extra target, or a
	// --root that does not resolve to a directory. TOC is never called on this path.
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

// Run is the whole of the quarry command below os.Exit. args is os.Args[1:]. It executes the
// request pipeline in this fixed order — the step that decides an exit code is the step that
// decides the message, so the two things are never allowed to drift apart:
//
//  1. Parse flags and the verb. A usage error is exit 2; anything else is exit 3.
//  2. When help was requested, write usageText to stdout and return exit 0 — help is a successful
//     query about the CLI, not a usage error, so it never touches stderr or exit 2.
//  3. Read the working directory. This is the one place in the package that does.
//  4. Resolve the repository root by calling internal/repopath, --root or discovery.
//  5. Convert the target to a clean, repository-relative path. Escaping the root is exit 1, named
//     with the target exactly as given, since a target that escaped the root has no meaningful
//     repository-relative form.
//  6. Lstat the resolved target: not found is exit 1, named with the repository-relative path; any
//     other stat error is exit 3. Lstat, never Stat, so a symlink named as the target is treated as
//     a file and not followed, the same rule engine.resolveTarget already follows.
//  7. Open the repository.
//  8. Run the query, and map its error through codeForTOCError rather than an inline errors.Is
//     chain at this call site, so the mapping stays table-testable. Steps 5 and 6 have already
//     excluded both sentinel errors in the common case; these branches exist because the target can
//     be removed between the stat and the walk, and Run must not report that race as success.
//  9. Render: quarry.RenderText under --text, quarry.RenderJSON otherwise. A render error, or a
//     failed write of its bytes to stdout, is exit 3 — that is an I/O failure, which is what exit 3
//     already means.
//
// The error value returned to a caller never carries the engine's wrapped chain for exit 1 or
// exit 2: those name conditions quarry itself defines, so quarry spells them, and passing
// something like `engine: resolve target "x": engine: target not found` through would leak an
// internal package name into a public contract. Exit 3 is the opposite case and carries
// err.Error() whole, behind the one "internal error: " prefix. Every message fail is called with
// is single-line; none embeds a newline.
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
	rel, err := repopath.RepoRelTarget(root, base, req.target)
	if err != nil {
		if errors.Is(err, quarry.ErrTargetOutsideRepo) {
			return fail(stdout, stderr, exitNegative, "target outside repository: "+req.target, false)
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
	// The two sentinel branches below are race-only in the common case: steps 5 and 6 have already
	// excluded both errors, so they fire only when the target is removed between step 6's stat and
	// the engine's own walk. Reporting that race as exitOK would be a false positive.
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
