// flags.go declares request, the parsed shape of one invocation, parseArgs, the hand-rolled
// parser that produces it, and glyphsPreset, the frozen token slice the glyphs verb rewrites its
// invocation to. The parser is hand-rolled rather than built on the standard library's flag
// package because flag cannot express --depth all alongside --depth 3, nor --no-symbols as a
// third state distinct from an absent --symbols.

package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Knatte18/quarry/quarry"
)

// usageError is the one error class that maps to exit 2. Its string is the CLI's own single-line
// complaint with no prefix and no wrapped chain. It is a distinct type rather than a sentinel
// precisely because each occurrence carries its own message.
type usageError string

// Error implements the error interface, returning the message verbatim.
func (e usageError) Error() string { return string(e) }

// request is the parsed shape of one invocation of parseArgs. root holds the --root value exactly
// as given, empty when the flag was absent. symbols is nil when neither --symbols nor
// --no-symbols was given, which is the engine's per-target default. unit holds the --unit value
// exactly as given, empty when the flag was absent. view holds the --view value exactly as given,
// empty when the flag was absent, which means the same thing as "full".
type request struct {
	verb    string
	target  string
	depth   int
	symbols *bool
	text    bool
	root    string
	unit    string
	view    string
	help    bool
}

// glyphsPreset is the frozen expansion `quarry glyphs <target>` rewrites to: --view glyphs,
// --depth all, --symbols. It is a var only because Go has no constant slice expression for a
// slice literal; no code may append into it — the rewrite in parseArgs below builds a fresh slice
// from this one and the caller's own tokens, and appending into glyphsPreset's backing array
// would let a second invocation in the same process see the first invocation's target. Its exact
// tokens are spelled in three other places that must change with it: usageText, docs/rewrite-plan.md
// section 5, and the byte-identity test in internal/cli/glyphs_test.go.
var glyphsPreset = []string{"--view", "glyphs", "--depth", "all", "--symbols"}

// parseArgs parses args, which is os.Args[1:], into a request. It is pure over its argument
// slice: it resolves no path, stats nothing, and reads no working directory, which is what lets
// its table test run with no fixtures.
//
// --help and -h are scanned for at any position, before anything else, and before any unknown
// flag, missing verb, or unrecognised verb is rejected: when found, parseArgs returns a request
// with help set and a nil error, so help wins over every other complaint.
//
// The verb gate accepts exactly "toc", "glyphs", "resolve", "expand" and "name". --depth,
// --symbols, --no-symbols and --view are valid for "toc" only; every other verb rejects each with
// a usage error naming the flag and the verb, checked at the point the flag is recognised so that
// rejection takes precedence over the flag's own value validation. --view's own vocabulary is
// closed at exactly two values, "full" and "glyphs"; an absent --view means "full", and any other
// value is a usage error rather than a silent fallback to the complete answer, which is the whole
// point of the closed set. --text is valid for every verb, while --root is valid for the four
// repository verbs (toc, glyphs, resolve and expand) only, since "name" reads nothing from the
// filesystem. --unit is valid for "name" only, and is required there: a "name" invocation with no
// --unit is rejected with a usage error naming the missing flag rather than the verb. Every verb
// requires exactly one target; parseArgs classifies none of them further — whether "expand"'s
// target contains a "#" is the grammar's question, not this parser's, so parseArgs stays pure over
// the argument slice — no root discovery, no engine call — with nothing left in its own table test
// that depended on rejecting a bare path here.
//
// "glyphs" is a frozen preset over "toc": once its own pre-rewrite validation (see the branch
// immediately below the verb gate) passes, parseArgs rewrites the invocation to "toc" plus
// glyphsPreset's tokens plus the caller's own target and re-parses it, so nothing downstream — not
// the dispatch switch, not runTOC, not the renderers — can tell a "glyphs" invocation from its
// expansion. req.verb is "toc" after the rewrite, which is why the dispatch switch in cli.go needs
// no new case for "glyphs".
//
// Immediately after the flag loop ends, two view-dependent rules run before the target-count
// check: when req.view is "glyphs", an explicit --no-symbols is rejected, and an absent --symbols
// defaults to true. Both checks live here, after the loop, rather than inside the
// --symbols/--no-symbols/--view cases themselves, because --view and --symbols/--no-symbols can be
// given in either order on the command line, and a check made at the point either flag is read
// would accept one order and reject the other. The rule itself exists because the engine's own
// per-target default for TOCOptions.Symbols is nil, meaning false for a directory target: a
// "toc --view glyphs <dir>" with no symbols flag would otherwise answer with an empty symbol
// list — a view whose entire content is symbols returning none. This is a default, not a filter:
// "--view full --no-symbols" and a viewless "--no-symbols" are untouched, so the complete answer
// stays exactly one flag away; "--no-symbols" is rejected under "--view glyphs" rather than
// honoured, because honouring it would ask for a view of nothing.
func parseArgs(args []string) (request, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return request{help: true}, nil
		}
	}

	if len(args) == 0 {
		return request{}, usageError("no verb given; expected: toc, glyphs, resolve, expand, or name")
	}

	verb := args[0]
	if strings.HasPrefix(verb, "-") {
		return request{}, usageError("no verb given; expected: toc, glyphs, resolve, expand, or name")
	}
	if verb != "toc" && verb != "glyphs" && verb != "resolve" && verb != "expand" && verb != "name" {
		return request{}, usageError(fmt.Sprintf("unknown verb: %s", verb))
	}

	if verb == "glyphs" {
		return parseGlyphsArgs(args[1:])
	}

	req := request{verb: verb}
	var targets []string

	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if !strings.HasPrefix(tok, "-") {
			targets = append(targets, tok)
			continue
		}

		// Split the equals form (--flag=value) from the space-separated form (--flag value) on the
		// first "=" only, so a value that itself contains "=" is preserved verbatim.
		name, value, hasValue := strings.Cut(tok, "=")

		nextValue := func() (string, bool) {
			if hasValue {
				return value, true
			}
			if i+1 < len(rest) {
				i++
				return rest[i], true
			}
			return "", false
		}

		switch name {
		case "--depth":
			if verb != "toc" {
				return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, verb))
			}
			v, ok := nextValue()
			if !ok {
				return request{}, usageError(fmt.Sprintf("%s requires a value", name))
			}
			if v == "all" {
				req.depth = quarry.DepthAll
				continue
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return request{}, usageError(fmt.Sprintf(
					"--depth must be a non-negative integer or \"all\", got %q", v))
			}
			req.depth = n
		case "--view":
			if verb != "toc" {
				return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, verb))
			}
			v, ok := nextValue()
			if !ok {
				return request{}, usageError(fmt.Sprintf("%s requires a value", name))
			}
			if v != "full" && v != "glyphs" {
				return request{}, usageError(fmt.Sprintf("--view must be \"full\" or \"glyphs\", got %q", v))
			}
			req.view = v
		case "--symbols":
			if verb != "toc" {
				return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, verb))
			}
			if req.symbols != nil && !*req.symbols {
				return request{}, usageError("--symbols and --no-symbols cannot both be given")
			}
			t := true
			req.symbols = &t
		case "--no-symbols":
			if verb != "toc" {
				return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, verb))
			}
			if req.symbols != nil && *req.symbols {
				return request{}, usageError("--symbols and --no-symbols cannot both be given")
			}
			f := false
			req.symbols = &f
		case "--text":
			req.text = true
		case "--root":
			if verb == "name" {
				return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, verb))
			}
			v, ok := nextValue()
			if !ok {
				return request{}, usageError(fmt.Sprintf("%s requires a value", name))
			}
			req.root = v
		case "--unit":
			if verb != "name" {
				return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, verb))
			}
			v, ok := nextValue()
			if !ok {
				return request{}, usageError(fmt.Sprintf("%s requires a value", name))
			}
			req.unit = v
		default:
			return request{}, usageError(fmt.Sprintf("unknown flag: %s", tok))
		}
	}

	// Both view-dependent symbols rules run here, after the flag loop, rather than inside the
	// --symbols/--no-symbols/--view cases above: --view and --symbols/--no-symbols can be given in
	// either order, so a check at the point either one is read would accept one order and reject
	// the other. See parseArgs's own doc comment for the rest of the reasoning.
	if req.view == "glyphs" {
		if req.symbols != nil && !*req.symbols {
			return request{}, usageError("--no-symbols is not valid with --view glyphs")
		}
		if req.symbols == nil {
			t := true
			req.symbols = &t
		}
	}

	if len(targets) != 1 {
		return request{}, usageError(fmt.Sprintf("%s takes exactly one target, got %d", verb, len(targets)))
	}
	req.target = targets[0]

	if verb == "name" && req.unit == "" {
		return request{}, usageError("--unit is required for name")
	}

	return req, nil
}

// parseGlyphsArgs is parseArgs's own pre-scan and rewrite for the "glyphs" verb, run on rest, the
// tokens after the "glyphs" word itself. Its only job is to make every rejection name "glyphs"
// rather than "toc" — the verb the rewritten invocation below actually carries — and only once
// that validation passes does it build the rewritten argument slice and re-parse it.
//
// The walk uses the same index-based loop shape and the same strings.Cut(tok, "=")-on-the-first-"="
// splitting parseArgs's own main loop uses. A token not beginning with "-" is counted as a target.
// --view, --depth, --symbols, --no-symbols and --unit are each rejected with the existing
// "%s is not valid for %s" format, naming "glyphs" rather than "toc", so
// "quarry glyphs --depth 1 x" says "--depth is not valid for glyphs", never "toc". --text is
// accepted and takes no value. --root is accepted and consumes its value the same way the main
// loop's nextValue closure does — the part after "=" when the token carried one, otherwise the
// following token, which the loop then skips — so "glyphs --root <path> <target>" counts as one
// target rather than two; when there is no following token to consume, it is rejected with the
// existing "%s requires a value" message, exactly as the main loop's own --root case does, rather
// than falling through to a target-count message that would name the wrong problem. Any other flag
// token is rejected with the existing verb-free "unknown flag: %s" message, formatted with the
// whole token as given, matching the main loop's own use of tok rather than name. After the walk, a
// target count other than one is rejected with "glyphs takes exactly one target, got %d".
//
// Only when the pre-scan passes does parseGlyphsArgs build the rewritten argument slice — the
// literal "toc", then glyphsPreset's tokens, then the original rest tokens — into a freshly
// allocated slice sharing no backing array with glyphsPreset, and return parseArgs of it directly.
// Recursion depth is one and bounded by construction, because the rewritten slice's verb is "toc"
// and "toc" has no preset of its own to rewrite through. The re-parse cannot itself produce a usage
// error, because this pre-scan has already rejected everything it could reject; if it ever did,
// its message would name "toc", not "glyphs".
func parseGlyphsArgs(rest []string) (request, error) {
	targetCount := 0
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if !strings.HasPrefix(tok, "-") {
			targetCount++
			continue
		}

		name, value, hasValue := strings.Cut(tok, "=")

		nextValue := func() (string, bool) {
			if hasValue {
				return value, true
			}
			if i+1 < len(rest) {
				i++
				return rest[i], true
			}
			return "", false
		}

		switch name {
		case "--view", "--depth", "--symbols", "--no-symbols", "--unit":
			return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, "glyphs"))
		case "--text":
			// Accepted, and takes no value.
		case "--root":
			if _, ok := nextValue(); !ok {
				return request{}, usageError(fmt.Sprintf("%s requires a value", name))
			}
		default:
			return request{}, usageError(fmt.Sprintf("unknown flag: %s", tok))
		}
	}

	if targetCount != 1 {
		return request{}, usageError(fmt.Sprintf("glyphs takes exactly one target, got %d", targetCount))
	}

	rewritten := make([]string, 0, 1+len(glyphsPreset)+len(rest))
	rewritten = append(rewritten, "toc")
	rewritten = append(rewritten, glyphsPreset...)
	rewritten = append(rewritten, rest...)
	return parseArgs(rewritten)
}
