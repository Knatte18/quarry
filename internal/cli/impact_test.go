// impact_test.go drives impact.go's pure helpers directly, following cli_test.go's offline,
// spawn-free pattern: no subprocess, no language server, no fake LSP seam. internal/cli has no
// fake language server, so this file follows the same split every existing test in this package
// uses — a path that fails before any server spawn, or a pure helper called directly — rather than
// inventing one; the end-to-end claim belongs to the live-tier test, not here.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// TestEmitImpactResult covers emitImpactResult's error routing and its success shape.
func TestEmitImpactResult(t *testing.T) {
	tests := []struct {
		name      string
		result    quarry.ImpactResult
		err       error
		wantCode  int
		wantOk    bool
		checkBody func(t *testing.T, env map[string]any)
	}{
		{
			name: "ambiguous",
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
			name:     "not_found",
			err:      &quarry.ErrSymbolNotFound{Symbol: "Bar", TargetDir: "/tmp"},
			wantCode: 1,
			wantOk:   false,
			checkBody: func(t *testing.T, env map[string]any) {
				t.Helper()
				errMsg, _ := env["error"].(string)
				if !strings.Contains(errMsg, "Bar") {
					t.Errorf("error = %q; want it to mention %q", errMsg, "Bar")
				}
			},
		},
		{
			name:     "other_error",
			err:      errors.New("boom"),
			wantCode: 1,
			wantOk:   false,
			checkBody: func(t *testing.T, env map[string]any) {
				t.Helper()
				errMsg, _ := env["error"].(string)
				if errMsg != "boom" {
					t.Errorf("error = %q; want %q", errMsg, "boom")
				}
			},
		},
		{
			name: "success",
			result: quarry.ImpactResult{
				Target:     &quarry.ImpactTarget{Name: "Foo", Kind: quarry.TOCKindFunction},
				Definition: &quarry.ImpactDefinition{File: "/repo/foo.go", Line: 10},
				Callers: []quarry.ImpactCaller{
					{File: "/repo/bar.go", CallSiteLine: 5},
				},
			},
			wantCode: 0,
			wantOk:   true,
			checkBody: func(t *testing.T, env map[string]any) {
				t.Helper()
				if resolution, _ := env["resolution"].(string); resolution != "complete" {
					t.Errorf("envelope %v missing \"resolution\":\"complete\"; got %q", env, resolution)
				}
				if _, ok := env["target"].(map[string]any); !ok {
					t.Errorf("envelope %v missing \"target\" object", env)
				}
				if _, ok := env["definition"].(map[string]any); !ok {
					t.Errorf("envelope %v missing \"definition\" object", env)
				}
				callers, ok := env["callers"].([]any)
				if !ok {
					t.Fatalf("envelope %v missing []any \"callers\" field", env)
				}
				if len(callers) != 1 {
					t.Errorf("len(callers) = %d; want 1", len(callers))
				}
			},
		},
		{
			name: "success_no_callers_emits_empty_array_never_null",
			result: quarry.ImpactResult{
				Target:     &quarry.ImpactTarget{Name: "Foo"},
				Definition: &quarry.ImpactDefinition{File: "/repo/foo.go", Line: 10},
				Callers:    []quarry.ImpactCaller{},
			},
			wantCode: 0,
			wantOk:   true,
			checkBody: func(t *testing.T, env map[string]any) {
				t.Helper()
				callers, ok := env["callers"].([]any)
				if !ok {
					t.Fatalf("envelope %v \"callers\" is not a JSON array (likely emitted null instead of [])", env)
				}
				if len(callers) != 0 {
					t.Errorf("len(callers) = %d; want 0", len(callers))
				}
			},
		},
		{
			name: "success_no_target_or_definition_omits_both_keys",
			result: quarry.ImpactResult{
				Callers: []quarry.ImpactCaller{},
			},
			wantCode: 0,
			wantOk:   true,
			checkBody: func(t *testing.T, env map[string]any) {
				t.Helper()
				if _, ok := env["target"]; ok {
					t.Errorf("envelope %v carries a \"target\" key; want it entirely omitted", env)
				}
				if _, ok := env["definition"]; ok {
					t.Errorf("envelope %v carries a \"definition\" key; want it entirely omitted", env)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			ctx, es := NewExitContext(context.Background())

			emitImpactResult(ctx, &out, tt.result, tt.err)

			if es.Code() != tt.wantCode {
				t.Errorf("es.Code() = %d; want %d", es.Code(), tt.wantCode)
			}

			var env map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
				t.Fatalf("emitImpactResult output is not valid JSON: %v; got: %q", err, out.String())
			}

			if ok, _ := env["ok"].(bool); ok != tt.wantOk {
				t.Errorf("envelope ok = %v; want %v", ok, tt.wantOk)
			}

			tt.checkBody(t, env)
		})
	}
}

// TestClassifyImpactError covers classifyImpactError's three error branches and its nil-error
// "found" branch.
func TestClassifyImpactError(t *testing.T) {
	t.Parallel()

	ambiguousStatus, ambiguousFields := classifyImpactError(&quarry.ErrAmbiguousSymbol{Symbol: "Foo", Candidates: []string{"a.go:1:1"}}, quarry.ImpactResult{})
	if ambiguousStatus != statusAmbiguous {
		t.Errorf("classifyImpactError(ambiguous, ...) status = %q; want %q", ambiguousStatus, statusAmbiguous)
	}
	if _, ok := ambiguousFields["candidates"]; !ok {
		t.Errorf("classifyImpactError(ambiguous, ...) fields = %v; want a \"candidates\" field", ambiguousFields)
	}

	notFoundStatus, notFoundFields := classifyImpactError(quarry.ErrSymbolNotFoundSentinel, quarry.ImpactResult{})
	if notFoundStatus != statusNotFound {
		t.Errorf("classifyImpactError(ErrSymbolNotFoundSentinel, ...) status = %q; want %q", notFoundStatus, statusNotFound)
	}
	if len(notFoundFields) != 0 {
		t.Errorf("classifyImpactError(ErrSymbolNotFoundSentinel, ...) fields = %v; want no extra fields", notFoundFields)
	}

	errorStatus, errorFields := classifyImpactError(errors.New("boom"), quarry.ImpactResult{})
	if errorStatus != statusError {
		t.Errorf("classifyImpactError(boom, ...) status = %q; want %q", errorStatus, statusError)
	}
	if errMsg, _ := errorFields["error"].(string); errMsg != "boom" {
		t.Errorf("classifyImpactError(boom, ...) fields[\"error\"] = %q; want %q", errMsg, "boom")
	}

	result := quarry.ImpactResult{
		Target:  &quarry.ImpactTarget{Name: "Foo"},
		Callers: []quarry.ImpactCaller{},
	}
	foundStatus, foundFields := classifyImpactError(nil, result)
	if foundStatus != statusFound {
		t.Errorf("classifyImpactError(nil, ...) status = %q; want %q", foundStatus, statusFound)
	}
	if resolution, _ := foundFields["resolution"].(string); resolution != "complete" {
		t.Errorf("classifyImpactError(nil, ...) fields = %v; missing \"resolution\":\"complete\"", foundFields)
	}
	if _, ok := foundFields["target"]; !ok {
		t.Errorf("classifyImpactError(nil, ...) fields = %v; missing marshalled \"target\"", foundFields)
	}
}

// TestRunBatch_ImpactIdentityObjectKeyedTargetNotSymbol proves the merge-order collision the
// identity-object-is-keyed-target-not-symbol Shared Decision avoids: each entry's "symbol" key
// still holds the query string the argument supplied, even though the per-entry fields also carry
// an identity object keyed "target".
func TestRunBatch_ImpactIdentityObjectKeyedTargetNotSymbol(t *testing.T) {
	t.Parallel()

	result := quarry.ImpactResult{
		Target:     &quarry.ImpactTarget{Name: "Foo"},
		Definition: &quarry.ImpactDefinition{File: "/repo/foo.go", Line: 10},
		Callers:    []quarry.ImpactCaller{},
	}

	var out bytes.Buffer
	ctx, es := NewExitContext(context.Background())

	args := []string{"Foo", "Bar"}
	runBatch(ctx, &out, args, func(symbol string) (batchStatus, map[string]any) {
		return classifyImpactError(nil, result)
	})

	if es.Code() != 0 {
		t.Errorf("es.Code() = %d; want 0", es.Code())
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("runBatch output is not valid JSON: %v; got: %q", err, out.String())
	}

	results, ok := env["results"].([]any)
	if !ok {
		t.Fatalf("envelope %v missing []any \"results\" field", env)
	}
	if len(results) != len(args) {
		t.Fatalf("len(results) = %d; want %d", len(results), len(args))
	}

	for i, r := range results {
		entry, ok := r.(map[string]any)
		if !ok {
			t.Fatalf("results[%d] = %v; want a JSON object", i, r)
		}
		if symbol, _ := entry["symbol"].(string); symbol != args[i] {
			t.Errorf("results[%d][\"symbol\"] = %q; want %q", i, symbol, args[i])
		}
		if _, ok := entry["target"].(map[string]any); !ok {
			t.Errorf("results[%d] = %v; missing \"target\" identity object", i, entry)
		}
	}
}

// TestRunBatch_ImpactMixedStatusWorstWins proves the worst status wins the exit code under
// statusRank, and that every entry still carries its own "symbol" and "status" keys.
func TestRunBatch_ImpactMixedStatusWorstWins(t *testing.T) {
	t.Parallel()

	foundResult := quarry.ImpactResult{Target: &quarry.ImpactTarget{Name: "Found"}, Callers: []quarry.ImpactCaller{}}

	lookupOne := func(symbol string) (batchStatus, map[string]any) {
		switch symbol {
		case "found":
			return classifyImpactError(nil, foundResult)
		case "not_found":
			return classifyImpactError(quarry.ErrSymbolNotFoundSentinel, quarry.ImpactResult{})
		case "ambiguous":
			return classifyImpactError(&quarry.ErrAmbiguousSymbol{Symbol: symbol, Candidates: []string{"a.go:1:1"}}, quarry.ImpactResult{})
		default:
			return classifyImpactError(errors.New("boom"), quarry.ImpactResult{})
		}
	}

	args := []string{"found", "not_found", "ambiguous", "error"}
	wantStatuses := []string{"found", "not_found", "ambiguous", "error"}

	var out bytes.Buffer
	ctx, es := NewExitContext(context.Background())

	runBatch(ctx, &out, args, lookupOne)

	if es.Code() != 3 {
		t.Fatalf("es.Code() = %d; want 3 (worst-outcome rank for a mixed batch ending in error)", es.Code())
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("runBatch output is not valid JSON: %v; got: %q", err, out.String())
	}

	results, ok := env["results"].([]any)
	if !ok {
		t.Fatalf("envelope %v missing []any \"results\" field", env)
	}
	if len(results) != len(args) {
		t.Fatalf("len(results) = %d; want %d", len(results), len(args))
	}
	for i, r := range results {
		entry, ok := r.(map[string]any)
		if !ok {
			t.Fatalf("results[%d] = %v; want a JSON object", i, r)
		}
		if symbol, _ := entry["symbol"].(string); symbol != args[i] {
			t.Errorf("results[%d][\"symbol\"] = %q; want %q", i, symbol, args[i])
		}
		if status, _ := entry["status"].(string); status != wantStatuses[i] {
			t.Errorf("results[%d][\"status\"] = %q; want %q", i, status, wantStatuses[i])
		}
	}
}

// TestFilterImpactWithin covers FilterImpactWithin's normalization and its own-invariant
// preservation: target/definition untouched, filtered callers dropped, a non-nil result even when
// everything is dropped.
func TestFilterImpactWithin(t *testing.T) {
	t.Parallel()

	inScope := quarry.ImpactCaller{File: "/repo/internal/websterengine/poll.go", CallSiteLine: 5}
	outOfScope := quarry.ImpactCaller{File: "/repo/internal/perchengine/identity.go", CallSiteLine: 10}
	target := &quarry.ImpactTarget{Name: "Foo"}
	definition := &quarry.ImpactDefinition{File: "/repo/internal/websterengine/poll.go", Line: 5}

	tests := []struct {
		name        string
		within      string
		baseDir     string
		callers     []quarry.ImpactCaller
		wantCallers []quarry.ImpactCaller
	}{
		{
			name:        "absolute_within_keeps_only_in_scope",
			within:      "/repo/internal/websterengine",
			baseDir:     "/anything",
			callers:     []quarry.ImpactCaller{inScope, outOfScope},
			wantCallers: []quarry.ImpactCaller{inScope},
		},
		{
			// A relative --within value is exactly what an un-normalized filter
			// silently turns into an empty result set: filepath.Rel errors when
			// compared against an absolute Caller.File, so normalization here is
			// load-bearing, not ceremony.
			name:        "relative_within_resolves_against_baseDir",
			within:      "internal/websterengine",
			baseDir:     "/repo",
			callers:     []quarry.ImpactCaller{inScope, outOfScope},
			wantCallers: []quarry.ImpactCaller{inScope},
		},
		{
			name:        "everything_dropped_stays_non_nil_empty",
			within:      "/repo/internal/websterengine",
			baseDir:     "/anything",
			callers:     []quarry.ImpactCaller{outOfScope},
			wantCallers: []quarry.ImpactCaller{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := quarry.ImpactResult{Target: target, Definition: definition, Callers: tt.callers}
			got := FilterImpactWithin(result, tt.within, tt.baseDir)

			if got.Target != target {
				t.Errorf("FilterImpactWithin(...).Target = %v; want it untouched (%v)", got.Target, target)
			}
			if got.Definition != definition {
				t.Errorf("FilterImpactWithin(...).Definition = %v; want it untouched (%v)", got.Definition, definition)
			}

			if got.Callers == nil {
				t.Fatalf("FilterImpactWithin(...).Callers = nil; want a non-nil slice even when empty")
			}
			if len(got.Callers) != len(tt.wantCallers) {
				t.Fatalf("FilterImpactWithin(...).Callers = %v; want %v", got.Callers, tt.wantCallers)
			}
			for i := range got.Callers {
				if got.Callers[i] != tt.wantCallers[i] {
					t.Errorf("FilterImpactWithin(...).Callers[%d] = %v; want %v", i, got.Callers[i], tt.wantCallers[i])
				}
			}
		})
	}
}

// TestRunCLI_Impact_RequiresAtLeastOneArg verifies "impact" requires at least one argument.
func TestRunCLI_Impact_RequiresAtLeastOneArg(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"impact"})

	if exitCode == 0 {
		t.Fatalf("RunCLI(impact) = 0; want non-zero exit for arg-count violation")
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("RunCLI(impact) output is not valid JSON: %v; got: %q", err, out.String())
	}
	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("RunCLI(impact) ok = true; want false")
	}
}

// TestRunCLI_Impact_NoLanguageError verifies "impact" fails with ErrNoLanguage in an empty
// directory.
func TestRunCLI_Impact_NoLanguageError(t *testing.T) {
	// Redirect the userConfigDir/userCacheDir seams at a fresh temp root so
	// resolveContext degrades to quarry.BuiltinRegistry() deterministically,
	// independent of whatever real servers.yaml the operator's machine has.
	withIsolatedPathSeams(t)

	emptyTargetDir := t.TempDir()

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"impact", "MySymbol", "--target-dir", emptyTargetDir})

	if exitCode == 0 {
		t.Fatalf("RunCLI(impact MySymbol --target-dir <empty>) = 0; want non-zero exit for ErrNoLanguage")
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
		t.Errorf("RunCLI(impact MySymbol --target-dir <empty>) ok = true; want false")
	}

	errMsg, _ := env["error"].(string)
	if errMsg == "" {
		t.Errorf("RunCLI(impact MySymbol --target-dir <empty>) error field empty or missing; got envelope: %v", env)
	}
	if !strings.Contains(errMsg, "no language detected") {
		t.Errorf("RunCLI(impact MySymbol --target-dir <empty>) error = %q; want it to mention ErrNoLanguage's \"no language detected\"", errMsg)
	}
}

// TestRunCLI_Impact_TwoArgsIsBatchMode verifies two or more arguments produce one "results" entry
// per argument, keyed on "symbol" with a "status", and the worst-status exit code.
func TestRunCLI_Impact_TwoArgsIsBatchMode(t *testing.T) {
	withIsolatedPathSeams(t)

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"impact", "one", "two", "--target-dir", t.TempDir()})

	if exitCode != 3 {
		t.Fatalf("RunCLI(impact one two --target-dir <empty>) = %d; want 3 (worst-outcome rank for an all-error batch)", exitCode)
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

// TestRunCLI_Impact_BuildTagsFailsForLanguageWithNoTemplate proves --build-tags fails rather than
// silently succeeding for a language whose registry entry carries no build-tag template:
// language detection resolves before any spawn, and python's registry entry has no build-tag
// template.
func TestRunCLI_Impact_BuildTagsFailsForLanguageWithNoTemplate(t *testing.T) {
	withIsolatedPathSeams(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"fixture\"\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(pyproject.toml) failed: %v", err)
	}

	var out bytes.Buffer
	exitCode := RunCLI(&out, []string{"impact", "MySymbol", "--target-dir", dir, "--build-tags", "foo"})

	if exitCode == 0 {
		t.Fatalf("RunCLI(impact MySymbol --target-dir <python dir> --build-tags foo) = 0; want non-zero exit")
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &env); err != nil {
		t.Fatalf("RunCLI output is not valid JSON: %v; got: %q", err, out.String())
	}
	if ok, _ := env["ok"].(bool); ok {
		t.Errorf("RunCLI(impact ... --build-tags foo) ok = true; want false")
	}
	errMsg, _ := env["error"].(string)
	if errMsg == "" {
		t.Errorf("RunCLI(impact ... --build-tags foo) error field empty or missing; got envelope: %v", env)
	}
}
