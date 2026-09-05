// flags.go declares request, the parsed shape of one invocation, and parseArgs, the hand-rolled
// parser that produces it. The parser is hand-rolled rather than built on the standard library's
// flag package because flag cannot express --depth all alongside --depth 3, nor --no-symbols as a
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
// --no-symbols was given, which is the engine's per-target default. from and to hold the delta
// verb's --from and --to values exactly as given; from is empty only when the flag was absent
// entirely, which parseArgs itself rejects as a usage error, and to is empty when the flag was
// absent, which is how the working tree is spelled to the git delta method. Neither field records
// whether its flag was given: the git delta method takes both revisions as plain strings with the
// empty string already meaning the working tree, so a present-but-empty state would have no way to
// reach it and no defined behaviour if it did. unit holds the --unit value exactly as given, empty
// when the flag was absent.
type request struct {
	verb    string
	target  string
	depth   int
	symbols *bool
	text    bool
	root    string
	unit    string
	help    bool
	from    string
	to      string
}

// parseArgs parses args, which is os.Args[1:], into a request. It is pure over its argument
// slice: it resolves no path, stats nothing, and reads no working directory, which is what lets
// its table test run with no fixtures.
//
// --help and -h are scanned for at any position, before anything else, and before any unknown
// flag, missing verb, or unrecognised verb is rejected: when found, parseArgs returns a request
// with help set and a nil error, so help wins over every other complaint.
//
// The verb gate accepts exactly "toc", "resolve", "expand", "delta" and "name". --depth, --symbols
// and --no-symbols are valid for "toc" only; --from and --to are valid for "delta" only; --unit is
// valid for "name" only, and is required there: a "name" invocation with no --unit is rejected
// with a usage error naming the missing flag rather than the verb. Every other verb rejects a flag
// outside its own scope with a usage error naming the flag and the verb, checked at the point the
// flag is recognised so that rejection takes precedence over the flag's own value validation.
// --text is valid for every verb, while --root is valid for the four repository verbs only, since
// "name" reads nothing from the filesystem. Every verb requires
// exactly one target; parseArgs classifies none of them further — whether "expand"'s target
// contains a "#" is the grammar's question, not this parser's, so parseArgs stays pure over the
// argument slice — no root discovery, no engine call — with nothing left in its own table test
// that depended on rejecting a bare path here.
func parseArgs(args []string) (request, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return request{help: true}, nil
		}
	}

	if len(args) == 0 {
		return request{}, usageError("no verb given; expected: toc, resolve, expand, delta, or name")
	}

	verb := args[0]
	if strings.HasPrefix(verb, "-") {
		return request{}, usageError("no verb given; expected: toc, resolve, expand, delta, or name")
	}
	if verb != "toc" && verb != "resolve" && verb != "expand" && verb != "delta" && verb != "name" {
		return request{}, usageError(fmt.Sprintf("unknown verb: %s", verb))
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
		case "--from":
			if verb != "delta" {
				return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, verb))
			}
			v, ok := nextValue()
			if !ok {
				return request{}, usageError(fmt.Sprintf("%s requires a value", name))
			}
			if v == "" {
				return request{}, usageError(fmt.Sprintf("%s value must not be empty", name))
			}
			req.from = v
		case "--to":
			if verb != "delta" {
				return request{}, usageError(fmt.Sprintf("%s is not valid for %s", name, verb))
			}
			v, ok := nextValue()
			if !ok {
				return request{}, usageError(fmt.Sprintf("%s requires a value", name))
			}
			if v == "" {
				return request{}, usageError(fmt.Sprintf("%s value must not be empty", name))
			}
			req.to = v
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

	if len(targets) != 1 {
		return request{}, usageError(fmt.Sprintf("%s takes exactly one target, got %d", verb, len(targets)))
	}
	req.target = targets[0]

	if verb == "delta" && req.from == "" {
		return request{}, usageError("delta requires --from")
	}
	if verb == "name" && req.unit == "" {
		return request{}, usageError("--unit is required for name")
	}

	return req, nil
}
