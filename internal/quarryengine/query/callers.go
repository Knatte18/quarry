// callers.go implements Callers, the one-connection verified caller lookup: it resolves a
// position, builds a declaration match set from textDocument/definition and
// textDocument/implementation, issues textDocument/references, and verifies each reference by
// issuing textDocument/definition at it — all on the single connection runOnConnection (refs.go)
// acquires, sequentially, under one phase deadline per phase.

package query

import (
	"context"
	"errors"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
)

// Callers resolves opts.Query against the language server for opts.TargetDir and returns two
// reference sets: references, the verified caller list (or every raw reference, unfiltered, when
// opts.SkipVerification is set or verification cannot run — see callersFromClient), and
// declaration, the definition-only result at the query position — never the widened match set
// verification builds internally, since internal/cli's filterUnexpectedCallers removes every
// returned declaration from the violation list, and returning the widened union would silently
// exclude every implementer's own declaration site from the gate.
func Callers(ctx context.Context, opts Options) (references []Reference, declaration []Reference, err error) {
	err = runOnConnection(ctx, opts, func(ctx context.Context, client *lsp.Client, fileURI string, pos lsp.Position, timedOut *bool) error {
		refs, decl, cErr := callersFromClient(ctx, client, fileURI, pos, opts.Timeout, timedOut, opts.SkipVerification)
		if cErr != nil {
			return cErr
		}
		references = refs
		declaration = decl
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return references, declaration, nil
}

// callersFromClient runs the Callers pipeline against an already-built client, fileURI, and pos —
// the seam symbolFromClient (symbol.go) already establishes for testing a multi-call pipeline
// against a hand-built client with no spawn. timeout brackets each phase with its own
// context.WithTimeout(ctx, timeout); timedOut is set (via the pointer runOnConnection passed
// through) whenever a phase's error satisfies errors.Is(err, quarryengine.ErrServerTimeoutSentinel),
// or whenever the per-reference verification loop's own deadline expires with work still
// outstanding — both cases where the connection might be stalled and teardownConnection needs to
// know to dispose of it rather than close it gracefully.
//
// The call order on the connection is fixed: textDocument/definition at pos, then
// textDocument/implementation at pos, then textDocument/references at pos, then one
// textDocument/definition per returned reference, strictly sequentially — lsp.Client is
// single-flight (Call increments an unsynchronized nextID, writeMessage holds no write lock, and
// the response loop reads one shared channel with no pending-request registry), so two concurrent
// calls on it would consume and drop each other's responses.
//
// Verification skips entirely — every reference kept, declaration returned as whatever the
// definition call produced — in exactly these cases: skipVerification is set; the declaration-side
// definition call errored or returned an empty location set; or client.SupportsImplementation()
// reports false or the implementation call errored. In every error case the timed-out flag is set
// first if the error was a server timeout, then the pipeline continues rather than returning the
// error.
func callersFromClient(ctx context.Context, client *lsp.Client, fileURI string, pos lsp.Position, timeout time.Duration, timedOut *bool, skipVerification bool) ([]Reference, []Reference, error) {
	defCtx, defCancel := context.WithTimeout(ctx, timeout)
	defLocs, defErr := client.Definition(defCtx, fileURI, pos)
	defCancel()

	// The definition call always runs, skipVerification or not: its result
	// is the returned declaration value the CLI's declaration exclusion
	// depends on.
	declaration := toSortedReferences(defLocs)

	verify := !skipVerification
	if defErr != nil {
		if errors.Is(defErr, quarryengine.ErrServerTimeoutSentinel) {
			*timedOut = true
		}
		verify = false
	} else if len(defLocs) == 0 {
		verify = false
	}

	var implLocs []lsp.Location
	if verify {
		if !client.SupportsImplementation() {
			verify = false
		} else {
			implCtx, implCancel := context.WithTimeout(ctx, timeout)
			locs, implErr := client.Implementation(implCtx, fileURI, pos)
			implCancel()
			if implErr != nil {
				if errors.Is(implErr, quarryengine.ErrServerTimeoutSentinel) {
					*timedOut = true
				}
				verify = false
			} else {
				implLocs = locs
			}
		}
	}

	var matchSet map[locationKey]bool
	if verify {
		matchSet = declarationMatchSet(defLocs, implLocs)
	}

	referencesCtx, referencesCancel := context.WithTimeout(ctx, timeout)
	refLocs, refErr := client.References(referencesCtx, fileURI, pos)
	referencesCancel()
	if refErr != nil {
		// textDocument/references is the primary answer: unlike the
		// declaration- and implementation-side calls above, its error is
		// returned unchanged rather than swallowed into a verification-skip
		// decision.
		if errors.Is(refErr, quarryengine.ErrServerTimeoutSentinel) {
			*timedOut = true
		}
		return nil, declaration, refErr
	}

	if !verify {
		return toSortedReferences(refLocs), declaration, nil
	}

	outcomes := make([]verificationOutcome, len(refLocs))
	loopCtx, loopCancel := context.WithTimeout(ctx, timeout)
	defer loopCancel()
	for i, refLoc := range refLocs {
		select {
		case <-loopCtx.Done():
			// A verification-phase deadline sets the timed-out flag and
			// still returns a successful result: the flag answers "might
			// this server be stalled?" and governs disposal, while the
			// return value answers "is this answer usable?" and is
			// governed by the fail-closed rule below. This is deliberately
			// inert for daemon.ConnKindSupervised, whose teardown returns
			// without killing or closing regardless of timedOut.
			*timedOut = true
			markRemainingUnattempted(outcomes, i)
			return toSortedReferences(filterVerifiedReferences(refLocs, matchSet, outcomes)), declaration, nil
		default:
		}

		locs, refDefErr := client.Definition(loopCtx, refLoc.URI, refLoc.Range.Start)
		outcomes[i] = verificationOutcome{Locations: locs, Err: refDefErr, Attempted: true}
		if refDefErr != nil && errors.Is(refDefErr, quarryengine.ErrServerTimeoutSentinel) {
			*timedOut = true
			markRemainingUnattempted(outcomes, i+1)
			break
		}
	}

	return toSortedReferences(filterVerifiedReferences(refLocs, matchSet, outcomes)), declaration, nil
}

// markRemainingUnattempted sets every outcome from index from onward to Attempted: false, the
// outcome filterVerifiedReferences always keeps — used when the per-reference verification loop
// stops early (deadline or a per-call timeout) and the remaining references were never queried.
func markRemainingUnattempted(outcomes []verificationOutcome, from int) {
	for j := from; j < len(outcomes); j++ {
		outcomes[j] = verificationOutcome{Attempted: false}
	}
}
