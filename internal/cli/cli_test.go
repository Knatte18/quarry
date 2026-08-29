// cli_test.go drives RunCLI through its seam: the bare/--help subcommand listing, every command's
// Short, and the ErrNoLanguage error-envelope path.
// It is deliberately untagged, offline, and spawn-free: it never shells out to a subprocess, never
// touches git, and never copies a fixture tree, so it never launches a language server.
// Isolation from the operator's real machine-global config/cache directories goes through the
// userConfigDir/userCacheDir seams (withIsolatedPathSeams below), never t.Chdir — both stdlib
// functions os.UserConfigDir/os.UserCacheDir ignore the process working directory entirely.
// A real "refs" query against a live language server belongs to the //go:build lsp tier
// (quarry's own live tier), not here.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/quarry"
)

// withIsolatedPathSeams redirects both userConfigDir and userCacheDir (paths.go's machine-global
// seams) at the same fresh t.TempDir() for the duration of the test, restoring both originals in
// a t.Cleanup, and returns the shared temp root.
// This is the one isolation mechanism every test below shares, so that a --config/--state-dir-less
// resolveContext call resolves against a throwaway directory instead of the operator's real
// os.UserConfigDir()/os.UserCacheDir() answer.
func withIsolatedPathSeams(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	origConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return root, nil }
	t.Cleanup(func() { userConfigDir = origConfigDir })

	origCacheDir := userCacheDir
	userCacheDir = func() (string, error) { return root, nil }
	t.Cleanup(func() { userCacheDir = origCacheDir })

	return root
}

// TestRunCLI_NoArgsListsRefsSubcommand verifies bare "quarry" lists subcommands and exits 0.
func TestRunCLI_NoArgsListsRefsSubcommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{})

	if exitCode != 0 {
		t.Errorf("RunCLI() = %d; want 0 for no-arg listing", exitCode)
	}
	if got := out.String(); !strings.Contains(got, "refs") {
		t.Errorf("RunCLI() no-arg output missing subcommand %q; got: %q", "refs", got)
	}
}

// TestRunCLI_Help verifies "quarry --help" lists subcommands and exits 0.
func TestRunCLI_Help(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"--help"})

	if exitCode != 0 {
		t.Errorf("RunCLI(--help) = %d; want 0", exitCode)
	}
	if got := out.String(); !strings.Contains(got, "refs") {
		t.Errorf("RunCLI(--help) output missing subcommand %q; got: %q", "refs", got)
	}
}

// TestCommand_EveryCommandHasShort asserts every command node has a non-empty Short.
func TestCommand_EveryCommandHasShort(t *testing.T) {
	t.Parallel()

	violations := collectMissingShorts(Command())
	for _, v := range violations {
		t.Errorf("command %q has no Short description", v)
	}
}

// collectMissingShorts returns command paths for every node whose Short is empty.
func collectMissingShorts(cmd *cobra.Command) []string {
	var violations []string
	if cmd.Short == "" {
		violations = append(violations, cmd.CommandPath())
	}
	for _, child := range cmd.Commands() {
		violations = append(violations, collectMissingShorts(child)...)
	}
	return violations
}

// TestRunCLI_Refs_NoLanguageError verifies "refs" fails with ErrNoLanguage in an empty directory.
func TestRunCLI_Refs_NoLanguageError(t *testing.T) {
	// Redirect the userConfigDir/userCacheDir seams at a fresh temp root so
	// resolveContext degrades to quarry.BuiltinRegistry() deterministically,
	// independent of whatever real servers.yaml the operator's machine has.
	withIsolatedPathSeams(t)

	emptyTargetDir := t.TempDir()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"refs", "MySymbol", "--target-dir", emptyTargetDir})

	if exitCode == 0 {
		t.Fatalf("RunCLI(refs MySymbol --target-dir <empty>) = 0; want non-zero exit for ErrNoLanguage")
	}

	// Assert the JSON envelope shape: exactly one object on one line, ok=false,
	// and a populated, non-empty error field.
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("RunCLI output has %d lines; want exactly 1. output:\n%s", len(lines), out.String())
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &env); err != nil {
		t.Fatalf("RunCLI output is not valid JSON: %v; got: %q", err, lines[0])
	}

	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("RunCLI(refs MySymbol --target-dir <empty>) ok = true; want false")
	}

	errMsg, _ := env["error"].(string)
	if errMsg == "" {
		t.Errorf("RunCLI(refs MySymbol --target-dir <empty>) error field empty or missing; got envelope: %v", env)
	}
	if !strings.Contains(errMsg, "no language detected") {
		t.Errorf("RunCLI(refs MySymbol --target-dir <empty>) error = %q; want it to mention ErrNoLanguage's \"no language detected\"", errMsg)
	}
}

// TestRunCLIIn_TargetDirResolvesAgainstInjectedSeamCwd proves the --target-dir defaulting rebase
// reaches a consumer: a relative --target-dir resolves against the seam cwd RunCLIIn injects,
// never the process cwd, and an absolute --target-dir is honoured unchanged.
// DetectLanguage's ErrNoLanguage message names the resolved targetDir verbatim ("searched markers
// ... under %s"), so it doubles as the observation point without needing any marker file on disk.
func TestRunCLIIn_TargetDirResolvesAgainstInjectedSeamCwd(t *testing.T) {
	// Deliberately not t.Parallel(): withIsolatedPathSeams overrides the package-level
	// userConfigDir/userCacheDir seams, which every other test in this package reads
	// (directly or via resolveContext). Running it concurrently with another parallel
	// test races on those shared vars.
	withIsolatedPathSeams(t)

	seamCwd := t.TempDir()

	var out bytes.Buffer
	exitCode := RunCLIIn(seamCwd, &out, []string{"refs", "MySymbol", "--target-dir", "sub"})
	if exitCode == 0 {
		t.Fatalf("RunCLIIn(seamCwd, refs MySymbol --target-dir sub) = 0; want non-zero exit for ErrNoLanguage")
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("RunCLIIn output is not valid JSON: %v; got: %q", err, out.String())
	}
	errMsg, _ := env["error"].(string)
	wantRelResolved := filepath.Join(seamCwd, "sub")
	if !strings.Contains(errMsg, wantRelResolved) {
		t.Errorf("RunCLIIn(seamCwd, refs --target-dir \"sub\") error = %q; want it to reference the seam-cwd-resolved dir %q", errMsg, wantRelResolved)
	}

	out.Reset()
	absDir := t.TempDir()
	exitCode = RunCLIIn(seamCwd, &out, []string{"refs", "MySymbol", "--target-dir", absDir})
	if exitCode == 0 {
		t.Fatalf("RunCLIIn(seamCwd, refs MySymbol --target-dir %s) = 0; want non-zero exit for ErrNoLanguage", absDir)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("RunCLIIn output is not valid JSON: %v; got: %q", err, out.String())
	}
	errMsg, _ = env["error"].(string)
	if !strings.Contains(errMsg, absDir) {
		t.Errorf("RunCLIIn(seamCwd, refs --target-dir %s) error = %q; want it to reference the absolute dir unchanged", absDir, errMsg)
	}
}

// TestRunCLI_Definition_NoLanguageError verifies "definition" fails with ErrNoLanguage in an empty
// directory.
func TestRunCLI_Definition_NoLanguageError(t *testing.T) {
	// Redirect the userConfigDir/userCacheDir seams at a fresh temp root so
	// resolveContext degrades to quarry.BuiltinRegistry() deterministically,
	// independent of whatever real servers.yaml the operator's machine has.
	withIsolatedPathSeams(t)

	emptyTargetDir := t.TempDir()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"definition", "MySymbol", "--target-dir", emptyTargetDir})

	if exitCode == 0 {
		t.Fatalf("RunCLI(definition MySymbol --target-dir <empty>) = 0; want non-zero exit for ErrNoLanguage")
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("RunCLI output has %d lines; want exactly 1. output:\n%s", len(lines), out.String())
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &env); err != nil {
		t.Fatalf("RunCLI output is not valid JSON: %v; got: %q", err, lines[0])
	}

	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("RunCLI(definition MySymbol --target-dir <empty>) ok = true; want false")
	}

	errMsg, _ := env["error"].(string)
	if errMsg == "" {
		t.Errorf("RunCLI(definition MySymbol --target-dir <empty>) error field empty or missing; got envelope: %v", env)
	}
	if !strings.Contains(errMsg, "no language detected") {
		t.Errorf("RunCLI(definition MySymbol --target-dir <empty>) error = %q; want it to mention ErrNoLanguage's \"no language detected\"", errMsg)
	}
}

// TestRunCLI_Definition_FileLineCharFlags_ReachesNoLanguageError verifies --file/--line/--char
// given together (no positional argument) are accepted by the Args validator and actually build a
// position query -- reaching the same ErrNoLanguage failure a "file:line:col" positional argument
// would, rather than failing at argument parsing or silently no-op'ing.
func TestRunCLI_Definition_FileLineCharFlags_ReachesNoLanguageError(t *testing.T) {
	withIsolatedPathSeams(t)

	emptyTargetDir := t.TempDir()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{
		"definition", "--target-dir", emptyTargetDir,
		"--file", "foo.go", "--line", "1", "--char", "1",
	})

	if exitCode == 0 {
		t.Fatalf("RunCLI(definition --file foo.go --line 1 --char 1 --target-dir <empty>) = 0; want non-zero exit for ErrNoLanguage")
	}

	var env map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &env); err != nil {
		t.Fatalf("RunCLI output is not valid JSON: %v; got: %q", err, out.String())
	}

	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "no language detected") {
		t.Errorf("RunCLI(definition --file --line --char --target-dir <empty>) error = %q; want it to mention ErrNoLanguage's \"no language detected\" -- proves the flags reached a real position query, not an argument-parsing failure", errMsg)
	}
}

// TestRunCLI_Definition_PartialPositionFlags_ArgsError verifies giving only some of
// --file/--line/--char, with no positional argument, fails at argument validation with a message
// naming both accepted forms -- rather than silently treating it as a bare 0-argument call or
// panicking on an incomplete position.
func TestRunCLI_Definition_PartialPositionFlags_ArgsError(t *testing.T) {
	withIsolatedPathSeams(t)

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"definition", "--file", "foo.go", "--line", "1"})

	if exitCode == 0 {
		t.Fatalf("RunCLI(definition --file foo.go --line 1) = 0; want non-zero exit for a missing --char")
	}

	got := out.String()
	if !strings.Contains(got, "file:line:col") || !strings.Contains(got, "--file/--line/--char") {
		t.Errorf("RunCLI(definition --file foo.go --line 1) output = %q; want it to name both the positional and --file/--line/--char forms", got)
	}
}

// TestRunCLI_Symbol_NoLanguageError verifies "symbol" fails with ErrNoLanguage in an empty
// directory.
func TestRunCLI_Symbol_NoLanguageError(t *testing.T) {
	// Redirect the userConfigDir/userCacheDir seams at a fresh temp root so
	// resolveContext degrades to quarry.BuiltinRegistry() deterministically,
	// independent of whatever real servers.yaml the operator's machine has.
	withIsolatedPathSeams(t)

	emptyTargetDir := t.TempDir()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"symbol", "MySymbol", "--target-dir", emptyTargetDir})

	if exitCode == 0 {
		t.Fatalf("RunCLI(symbol MySymbol --target-dir <empty>) = 0; want non-zero exit for ErrNoLanguage")
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("RunCLI output has %d lines; want exactly 1. output:\n%s", len(lines), out.String())
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &env); err != nil {
		t.Fatalf("RunCLI output is not valid JSON: %v; got: %q", err, lines[0])
	}

	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("RunCLI(symbol MySymbol --target-dir <empty>) ok = true; want false")
	}

	errMsg, _ := env["error"].(string)
	if errMsg == "" {
		t.Errorf("RunCLI(symbol MySymbol --target-dir <empty>) error field empty or missing; got envelope: %v", env)
	}
	if !strings.Contains(errMsg, "no language detected") {
		t.Errorf("RunCLI(symbol MySymbol --target-dir <empty>) error = %q; want it to mention ErrNoLanguage's \"no language detected\"", errMsg)
	}
}

// TestRunCLI_Symbol_TreatsFileLineColArgumentAsLiteralSearchString proves symbolCommand never
// position-parses "file:line:col" arguments, treating them as literal search strings.
func TestRunCLI_Symbol_TreatsFileLineColArgumentAsLiteralSearchString(t *testing.T) {
	const arg = "foo.go:1:1"

	query := symbolQuery(arg)
	if query.Symbol != arg {
		t.Errorf("symbolQuery(%q).Symbol = %q; want %q", arg, query.Symbol, arg)
	}
	if query.Pos != nil {
		t.Errorf("symbolQuery(%q).Pos = %+v; want nil — the argument must never be position-parsed", arg, query.Pos)
	}

	// The same string, driven through parseQuery (the converter
	// refs/definition use), DOES parse as a position — proving symbolQuery's
	// literal-search-string behavior is a deliberate divergence, not an
	// accident of parseQuery(arg) happening to leave Pos unset for this
	// particular string.
	base := t.TempDir()
	parsed, err := parseQuery(base, arg)
	if err != nil {
		t.Fatalf("parseQuery(%q, %q) error = %v; want nil", base, arg, err)
	}
	if parsed.Pos == nil {
		t.Fatalf("parseQuery(%q).Pos = nil; want a parsed position, to prove symbolQuery's divergence from parseQuery is meaningful", arg)
	}

	withIsolatedPathSeams(t)
	emptyTargetDir := t.TempDir()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"symbol", arg, "--target-dir", emptyTargetDir})

	if exitCode == 0 {
		t.Fatalf("RunCLI(symbol %s --target-dir <empty>) = 0; want non-zero exit for ErrNoLanguage", arg)
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("RunCLI output is not valid JSON: %v; got: %q", err, out.String())
	}

	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "no language detected") {
		t.Errorf("RunCLI(symbol %s --target-dir <empty>) error = %q; want it to mention ErrNoLanguage's \"no language detected\"", arg, errMsg)
	}
}

// TestEmitLookupResult_AmbiguousSymbolExitsTwo tests emitLookupResult's handling of ambiguous and
// not-found cases.
func TestEmitLookupResult_AmbiguousSymbolExitsTwo(t *testing.T) {
	tests := []struct {
		name         string
		resultsField string
		err          error
		wantCode     int
		wantOk       bool
		checkBody    func(t *testing.T, env map[string]any)
	}{
		{
			name:         "ambiguous",
			resultsField: "references",
			err: &quarry.ErrAmbiguousSymbol{
				Symbol:     "Foo",
				Candidates: []string{"a.go:1:1", "b.go:2:2"},
			},
			wantCode: 2,
			wantOk:   true,
			checkBody: func(t *testing.T, env map[string]any) {
				t.Helper()
				candidates, ok := env["candidates"].([]any)
				if !ok {
					t.Fatalf("envelope %v missing []any \"candidates\" field", env)
				}
				want := []string{"a.go:1:1", "b.go:2:2"}
				if len(candidates) != len(want) {
					t.Fatalf("candidates = %v; want %v", candidates, want)
				}
				for i, c := range candidates {
					if c != want[i] {
						t.Errorf("candidates[%d] = %v; want %v", i, c, want[i])
					}
				}
			},
		},
		{
			name:         "not_found",
			resultsField: "definitions",
			err:          &quarry.ErrSymbolNotFound{Symbol: "Bar", TargetDir: "/tmp"},
			wantCode:     1,
			wantOk:       false,
			checkBody: func(t *testing.T, env map[string]any) {
				t.Helper()
				errMsg, _ := env["error"].(string)
				if !strings.Contains(errMsg, "Bar") {
					t.Errorf("error = %q; want it to mention %q", errMsg, "Bar")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			ctx, es := NewExitContext(context.Background())

			emitLookupResult(ctx, &out, tt.resultsField, nil, tt.err)

			if es.Code() != tt.wantCode {
				t.Errorf("es.Code() = %d; want %d", es.Code(), tt.wantCode)
			}

			var env map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
				t.Fatalf("emitLookupResult output is not valid JSON: %v; got: %q", err, out.String())
			}

			if ok, _ := env["ok"].(bool); ok != tt.wantOk {
				t.Errorf("envelope ok = %v; want %v", ok, tt.wantOk)
			}

			tt.checkBody(t, env)
		})
	}
}

// TestRunCLI_Refs_RequiresAtLeastOneArg verifies "refs" requires at least one argument.
func TestRunCLI_Refs_RequiresAtLeastOneArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"bare", []string{"refs"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			exitCode := RunCLI(&out, tt.args)

			if exitCode == 0 {
				t.Fatalf("RunCLI(%v) = 0; want non-zero exit for arg-count violation", tt.args)
			}

			var env map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
				t.Fatalf("RunCLI(%v) output is not valid JSON: %v; got: %q", tt.args, err, out.String())
			}
			if ok, _ := env["ok"].(bool); ok {
				t.Errorf("RunCLI(%v) ok = true; want false", tt.args)
			}
		})
	}
}

// TestRunCLI_Refs_TwoArgsIsBatchMode verifies two or more arguments enable batch mode.
func TestRunCLI_Refs_TwoArgsIsBatchMode(t *testing.T) {
	withIsolatedPathSeams(t)

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"refs", "one", "two", "--target-dir", t.TempDir()})

	if exitCode != 3 {
		t.Fatalf("RunCLI(refs one two --target-dir <empty>) = %d; want 3 (worst-outcome rank for an all-error batch)", exitCode)
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("RunCLI output is not valid JSON: %v; got: %q", err, out.String())
	}

	results, ok := env["results"].([]any)
	if !ok {
		t.Fatalf("envelope %v missing []any \"results\" field", env)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d; want 2", len(results))
	}

	wantSymbols := []string{"one", "two"}
	for i, r := range results {
		entry, ok := r.(map[string]any)
		if !ok {
			t.Fatalf("results[%d] = %v; want a JSON object", i, r)
		}
		if status, _ := entry["status"].(string); status != "error" {
			t.Errorf("results[%d][\"status\"] = %q; want \"error\"", i, status)
		}
		if symbol, _ := entry["symbol"].(string); symbol != wantSymbols[i] {
			t.Errorf("results[%d][\"symbol\"] = %q; want %q", i, symbol, wantSymbols[i])
		}
	}
}

// TestBatchRunner_WorstOutcomeWinsExitCode verifies runBatch sets exit code to the worst status
// present.
func TestBatchRunner_WorstOutcomeWinsExitCode(t *testing.T) {
	t.Parallel()

	outcomes := map[string]struct {
		status batchStatus
		fields map[string]any
	}{
		"a": {statusFound, map[string]any{"references": []any{}}},
		"b": {statusNotFound, nil},
		"c": {statusAmbiguous, map[string]any{"candidates": []string{"x"}}},
		"d": {statusError, map[string]any{"error": "boom"}},
	}
	lookupOne := func(symbol string) (batchStatus, map[string]any) {
		o := outcomes[symbol]
		return o.status, o.fields
	}

	tests := []struct {
		name       string
		symbols    []string
		wantCode   int
		wantStatus []string
	}{
		{"all_found", []string{"a"}, 0, []string{"found"}},
		{"found_and_not_found", []string{"a", "b"}, 1, []string{"found", "not_found"}},
		{"found_not_found_ambiguous", []string{"a", "b", "c"}, 2, []string{"found", "not_found", "ambiguous"}},
		{"found_not_found_ambiguous_error", []string{"a", "b", "c", "d"}, 3, []string{"found", "not_found", "ambiguous", "error"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			ctx, es := NewExitContext(context.Background())

			runBatch(ctx, &out, tt.symbols, lookupOne)

			if es.Code() != tt.wantCode {
				t.Errorf("es.Code() = %d; want %d", es.Code(), tt.wantCode)
			}

			var env map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
				t.Fatalf("runBatch output is not valid JSON: %v; got: %q", err, out.String())
			}

			results, ok := env["results"].([]any)
			if !ok {
				t.Fatalf("envelope %v missing []any \"results\" field", env)
			}
			if len(results) != len(tt.wantStatus) {
				t.Fatalf("len(results) = %d; want %d", len(results), len(tt.wantStatus))
			}
			for i, r := range results {
				entry, ok := r.(map[string]any)
				if !ok {
					t.Fatalf("results[%d] = %v; want a JSON object", i, r)
				}
				if status, _ := entry["status"].(string); status != tt.wantStatus[i] {
					t.Errorf("results[%d][\"status\"] = %q; want %q", i, status, tt.wantStatus[i])
				}
			}
		})
	}
}

// TestEmitLookupResult_SuccessCarriesResolutionCompleteMarker verifies success includes
// "resolution":"complete".
func TestEmitLookupResult_SuccessCarriesResolutionCompleteMarker(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	ctx, es := NewExitContext(context.Background())

	emitLookupResult(ctx, &out, "references", []quarry.Reference{{File: "a.go", Line: 1, Character: 2}}, nil)

	if es.Code() != 0 {
		t.Errorf("es.Code() = %d; want 0", es.Code())
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("emitLookupResult output is not valid JSON: %v; got: %q", err, out.String())
	}

	if resolution, _ := env["resolution"].(string); resolution != "complete" {
		t.Errorf("envelope %v missing \"resolution\":\"complete\"; got %q", env, resolution)
	}
}

// TestClassifyLookupError_FoundCarriesResolutionCompleteMarker verifies statusFound includes
// "resolution":"complete".
func TestClassifyLookupError_FoundCarriesResolutionCompleteMarker(t *testing.T) {
	t.Parallel()

	status, fields := classifyLookupError(nil, "references", []quarry.Reference{{File: "a.go", Line: 1, Character: 2}})

	if status != statusFound {
		t.Errorf("classifyLookupError(nil, ...) status = %q; want %q", status, statusFound)
	}
	if resolution, _ := fields["resolution"].(string); resolution != "complete" {
		t.Errorf("classifyLookupError(nil, ...) fields = %v; missing \"resolution\":\"complete\"", fields)
	}

	ambiguousStatus, ambiguousFields := classifyLookupError(&quarry.ErrAmbiguousSymbol{Symbol: "Foo", Candidates: []string{"a.go:1:1"}}, "references", nil)
	if ambiguousStatus != statusAmbiguous {
		t.Fatalf("classifyLookupError(ambiguous, ...) status = %q; want %q", ambiguousStatus, statusAmbiguous)
	}
	if _, ok := ambiguousFields["resolution"]; ok {
		t.Errorf("classifyLookupError(ambiguous, ...) fields = %v; want no \"resolution\" field", ambiguousFields)
	}
}

// TestBuildOptions_ThreadsEveryFieldFromItsArguments verifies buildOptions threads all fields
// correctly.
func TestBuildOptions_ThreadsEveryFieldFromItsArguments(t *testing.T) {
	t.Parallel()

	registry := quarry.BuiltinRegistry()
	query := quarry.Query{Symbol: "Foo"}
	stateDir := "/state/dir"
	buildTags := []string{"a", "b"}

	got := buildOptions(registry, "/target", stateDir, "go", query, 5*time.Second, buildTags)

	if got.TargetDir != "/target" {
		t.Errorf("buildOptions(...).TargetDir = %q; want %q", got.TargetDir, "/target")
	}
	if got.StateDir != stateDir {
		t.Errorf("buildOptions(...).StateDir = %q; want %q", got.StateDir, stateDir)
	}
	if got.Lang != "go" {
		t.Errorf("buildOptions(...).Lang = %q; want %q", got.Lang, "go")
	}
	if got.Query != query {
		t.Errorf("buildOptions(...).Query = %+v; want %+v", got.Query, query)
	}
	if got.Timeout != 5*time.Second {
		t.Errorf("buildOptions(...).Timeout = %v; want %v", got.Timeout, 5*time.Second)
	}
	if len(got.BuildTags) != len(buildTags) {
		t.Fatalf("buildOptions(...).BuildTags = %v; want %v", got.BuildTags, buildTags)
	}
	for i := range buildTags {
		if got.BuildTags[i] != buildTags[i] {
			t.Errorf("buildOptions(...).BuildTags[%d] = %q; want %q", i, got.BuildTags[i], buildTags[i])
		}
	}
}

// TestInFileQuery_ProducesInFileNeverPosEvenForFileLineColShapedName proves inFileQuery never
// position-parses.
func TestInFileQuery_ProducesInFileNeverPosEvenForFileLineColShapedName(t *testing.T) {
	t.Parallel()

	const name = "foo.go:1:1"

	base := t.TempDir()
	query, err := inFileQuery(base, "internal/foo/bar.go", name)
	if err != nil {
		t.Fatalf("inFileQuery(%q, %q, %q) error = %v; want nil", base, "internal/foo/bar.go", name, err)
	}

	if query.Pos != nil {
		t.Errorf("inFileQuery(...).Pos = %+v; want nil — the name must never be position-parsed", query.Pos)
	}
	if query.InFile == nil {
		t.Fatalf("inFileQuery(...).InFile = nil; want a populated *InFileQuery")
	}
	if query.InFile.Name != name {
		t.Errorf("inFileQuery(...).InFile.Name = %q; want %q", query.InFile.Name, name)
	}
	if !filepath.IsAbs(query.InFile.File) {
		t.Errorf("inFileQuery(...).InFile.File = %q; want an absolute path", query.InFile.File)
	}
}

// TestInFileQuery_ResolvesRelativePathToAbsolute verifies a relative path resolves against the
// explicit base argument rather than the process cwd.
func TestInFileQuery_ResolvesRelativePathToAbsolute(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()

	query, err := inFileQuery(cwd, "relative/bar.go", "MyFunc")
	if err != nil {
		t.Fatalf("inFileQuery(%q, %q, %q) error = %v; want nil", cwd, "relative/bar.go", "MyFunc", err)
	}

	want := filepath.Join(cwd, "relative/bar.go")
	if query.InFile == nil || query.InFile.File != want {
		t.Errorf("inFileQuery(...).InFile.File = %+v; want %q", query.InFile, want)
	}
}

// TestInFileFlag_RegisteredOnRefsAndDefinitionOnlyNotSymbol verifies --in-file is on
// refs/definition but not symbol.
func TestInFileFlag_RegisteredOnRefsAndDefinitionOnlyNotSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     *cobra.Command
		wantHas bool
	}{
		{"refs", refsCommand(), true},
		{"definition", definitionCommand(), true},
		{"symbol", symbolCommand(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasFlag := tt.cmd.Flags().Lookup("in-file") != nil
			if hasFlag != tt.wantHas {
				t.Errorf("%s command has --in-file registered = %v; want %v", tt.name, hasFlag, tt.wantHas)
			}
		})
	}
}

// TestFilterWithin tests the --within filtering logic, which mitigates interface-method reference
// conflation.
func TestFilterWithin(t *testing.T) {
	t.Parallel()

	inScope1 := quarry.Reference{File: "/repo/internal/websterengine/poll.go", Line: 203}
	inScope2 := quarry.Reference{File: "/repo/internal/websterengine/state.go", Line: 10}
	crossPackage := quarry.Reference{File: "/repo/internal/perchengine/identity.go", Line: 44}
	// A sibling directory whose name merely starts with the same prefix —
	// proves filterWithin does not fall back to a naive string-prefix
	// check, which would wrongly treat "internal/webster" as containing
	// anything under "internal/websterengine" (they share no path
	// component boundary in common beyond the literal substring).
	prefixCollision := quarry.Reference{File: "/repo/internal/webstercli/cli.go", Line: 5}

	tests := []struct {
		name     string
		within   string
		baseDir  string
		refs     []quarry.Reference
		wantRefs []quarry.Reference
	}{
		{
			name:     "absolute_within_keeps_only_in_scope",
			within:   "/repo/internal/websterengine",
			baseDir:  "/anything", // unused: within is already absolute
			refs:     []quarry.Reference{inScope1, inScope2, crossPackage, prefixCollision},
			wantRefs: []quarry.Reference{inScope1, inScope2},
		},
		{
			name:     "relative_within_resolves_against_baseDir",
			within:   "internal/websterengine",
			baseDir:  "/repo",
			refs:     []quarry.Reference{inScope1, crossPackage},
			wantRefs: []quarry.Reference{inScope1},
		},
		{
			name:     "prefix_collision_directory_excluded",
			within:   "/repo/internal/webster",
			baseDir:  "/anything",
			refs:     []quarry.Reference{prefixCollision},
			wantRefs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := filterWithin(tt.refs, tt.within, tt.baseDir)
			if len(got) != len(tt.wantRefs) {
				t.Fatalf("filterWithin() = %v; want %v", got, tt.wantRefs)
			}
			for i := range got {
				if got[i] != tt.wantRefs[i] {
					t.Errorf("filterWithin()[%d] = %v; want %v", i, got[i], tt.wantRefs[i])
				}
			}
		})
	}
}

// TestClassifySymbolError_MultipleMatchesIsFoundNotAmbiguous verifies symbol never produces
// ambiguous status.
func TestClassifySymbolError_MultipleMatchesIsFoundNotAmbiguous(t *testing.T) {
	t.Parallel()

	status, fields := classifySymbolError(nil, []quarry.SymbolMatch{{Name: "Foo"}, {Name: "FooBar"}})

	if status != statusFound {
		t.Errorf("classifySymbolError(nil, <2 matches>) status = %q; want %q", status, statusFound)
	}

	symbols, ok := fields["symbols"].([]map[string]any)
	if !ok {
		t.Fatalf("fields %v missing []map[string]any \"symbols\" field", fields)
	}
	if len(symbols) != 2 {
		t.Fatalf("len(symbols) = %d; want 2", len(symbols))
	}
	wantNames := []string{"Foo", "FooBar"}
	for i, s := range symbols {
		if name, _ := s["name"].(string); name != wantNames[i] {
			t.Errorf("symbols[%d][\"name\"] = %q; want %q", i, name, wantNames[i])
		}
	}
}

// TestRunCLI_AssertNoCallers_NoLanguageError verifies "assert-no-callers" fails with ErrNoLanguage
// in an empty directory.
func TestRunCLI_AssertNoCallers_NoLanguageError(t *testing.T) {
	// Redirect the userConfigDir/userCacheDir seams at a fresh temp root so
	// resolveContext degrades to quarry.BuiltinRegistry() deterministically,
	// independent of whatever real servers.yaml the operator's machine has.
	withIsolatedPathSeams(t)

	emptyTargetDir := t.TempDir()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"assert-no-callers", "MySymbol", "--target-dir", emptyTargetDir})

	if exitCode == 0 {
		t.Fatalf("RunCLI(assert-no-callers MySymbol --target-dir <empty>) = 0; want non-zero exit for ErrNoLanguage")
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("RunCLI output has %d lines; want exactly 1. output:\n%s", len(lines), out.String())
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &env); err != nil {
		t.Fatalf("RunCLI output is not valid JSON: %v; got: %q", err, lines[0])
	}

	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("RunCLI(assert-no-callers MySymbol --target-dir <empty>) ok = true; want false")
	}
	if env["violation"] != nil {
		t.Errorf(`RunCLI(assert-no-callers MySymbol --target-dir <empty>) envelope carries "violation"; want absent for a lookup failure, not a real violation`)
	}

	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "no language detected") {
		t.Errorf("RunCLI(assert-no-callers MySymbol --target-dir <empty>) error = %q; want it to mention ErrNoLanguage's \"no language detected\"", errMsg)
	}
}

// TestRunCLI_AssertNoCallers_RequiresExactlyOneArg verifies "assert-no-callers" requires exactly
// one argument.
func TestRunCLI_AssertNoCallers_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"bare", []string{"assert-no-callers"}},
		{"two_args", []string{"assert-no-callers", "Foo", "Bar"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			exitCode := RunCLI(&out, tt.args)

			if exitCode == 0 {
				t.Errorf("RunCLI(%v) = 0; want non-zero exit for wrong arg count", tt.args)
			}
		})
	}
}

// TestFilterUnexpectedCallers tests the filtering logic for unexpected callers.
func TestFilterUnexpectedCallers(t *testing.T) {
	t.Parallel()

	decl := quarry.Reference{File: "/repo/pkg/foo.go", Line: 10, Character: 6}
	wrapper := quarry.Reference{File: "/repo/pkg/wrapper.go", Line: 20, Character: 3}
	caller := quarry.Reference{File: "/repo/other/bar.go", Line: 5, Character: 12}

	tests := []struct {
		name      string
		refs      []quarry.Reference
		declRefs  []quarry.Reference
		exceptAbs map[string]bool
		want      []quarry.Reference
	}{
		{
			name:     "declaration_only_is_clean",
			refs:     []quarry.Reference{decl},
			declRefs: []quarry.Reference{decl},
			want:     nil,
		},
		{
			name:      "except_path_is_excluded",
			refs:      []quarry.Reference{decl, wrapper},
			declRefs:  []quarry.Reference{decl},
			exceptAbs: map[string]bool{"/repo/pkg/wrapper.go": true},
			want:      nil,
		},
		{
			name:     "unexpected_caller_survives",
			refs:     []quarry.Reference{decl, wrapper, caller},
			declRefs: []quarry.Reference{decl},
			exceptAbs: map[string]bool{
				"/repo/pkg/wrapper.go": true,
			},
			want: []quarry.Reference{caller},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := filterUnexpectedCallers(tt.refs, tt.declRefs, tt.exceptAbs)
			if len(got) != len(tt.want) {
				t.Fatalf("filterUnexpectedCallers() = %v; want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("filterUnexpectedCallers()[%d] = %v; want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestAssertNoCallersFilterOrdering covers filterWithin composed with filterUnexpectedCallers in
// the exact order assertNoCallersCommand applies them -- --within, then --except, then the
// declaration exclusion -- over a hand-built reference slice.
//
// This is the only part of that ordering internal/cli can observe: quarry.Callers is a direct
// package-level call with no injection seam, so no test here can see the verification step that
// now runs ahead of all three filters. internal/quarryengine/query/callers_test.go covers
// verification itself.
func TestAssertNoCallersFilterOrdering(t *testing.T) {
	t.Parallel()

	const baseDir = "/repo"
	const within = "internal/websterengine"

	decl := quarry.Reference{File: "/repo/internal/websterengine/poll.go", Line: 1, Character: 6}
	exceptRef := quarry.Reference{File: "/repo/internal/websterengine/wrapper.go", Line: 5, Character: 3}
	outOfScope := quarry.Reference{File: "/repo/internal/perchengine/identity.go", Line: 10, Character: 2}
	violation := quarry.Reference{File: "/repo/internal/websterengine/caller.go", Line: 20, Character: 4}

	refs := []quarry.Reference{decl, exceptRef, outOfScope, violation}

	scoped := filterWithin(refs, within, baseDir)
	if len(scoped) != 3 {
		t.Fatalf("filterWithin(...) = %v; want 3 entries (outOfScope excluded)", scoped)
	}
	for _, r := range scoped {
		if r == outOfScope {
			t.Fatalf("filterWithin(...) = %v; want outOfScope %v excluded", scoped, outOfScope)
		}
	}

	exceptAbs := map[string]bool{filepath.Clean(exceptRef.File): true}
	got := filterUnexpectedCallers(scoped, []quarry.Reference{decl}, exceptAbs)

	if len(got) != 1 {
		t.Fatalf("filterUnexpectedCallers(filterWithin(...)) = %v; want exactly [%v]", got, violation)
	}
	if got[0] != violation {
		t.Errorf("filterUnexpectedCallers(filterWithin(...))[0] = %v; want %v", got[0], violation)
	}
	for _, r := range got {
		if r == decl {
			t.Errorf("got %v; want the declaration reference %v excluded", got, decl)
		}
		if r == exceptRef {
			t.Errorf("got %v; want the --except reference %v excluded", got, exceptRef)
		}
		if r == outOfScope {
			t.Errorf("got %v; want the out-of-scope reference %v excluded", got, outOfScope)
		}
	}
}

// TestBuildTagsFlag_RegisteredOnAllFiveVerbs verifies every verb accepts --build-tags, and that
// assert-no-callers additionally accepts --no-verify, by looking the flags up on the built
// command tree rather than by executing a query.
func TestBuildTagsFlag_RegisteredOnAllFiveVerbs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"refs", refsCommand()},
		{"definition", definitionCommand()},
		{"symbol", symbolCommand()},
		{"assert-no-callers", assertNoCallersCommand()},
		{"impact", impactCommand()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Flags().Lookup("build-tags") == nil {
				t.Errorf("%s command has no --build-tags flag registered", tt.name)
			}
		})
	}

	for _, tt := range tests {
		hasNoVerify := tt.cmd.Flags().Lookup("no-verify") != nil
		wantNoVerify := tt.name == "assert-no-callers"
		if hasNoVerify != wantNoVerify {
			t.Errorf("%s command has --no-verify registered = %v; want %v (assert-no-callers-only)", tt.name, hasNoVerify, wantNoVerify)
		}
	}
}
