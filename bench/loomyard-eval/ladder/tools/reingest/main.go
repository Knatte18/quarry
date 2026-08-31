// Command reingest recomputes usage.json for every already-ingested run directory under a results
// root, using the current internal/ladder.ExtractUsage — the per-API-call accounting fix landed in
// commit 7fdb4f2 replaced the original per-transcript-record summing, which multiply-counted every
// multi-block API call (up to 2.15x on a real matrix run). It never re-dispatches anything: the
// transcript.jsonl and .claude/agents/<config-id>.md files a prior ingest already copied into each run
// directory are read as-is, and only usage.json is rewritten. The previous usage.json is preserved
// alongside as usage.json.orig so the recompute is trivially reversible.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func main() {
	resultsRoot := flag.String("results-root", "", "the results directory to walk (required)")
	flag.Parse()

	if *resultsRoot == "" {
		fmt.Fprintln(os.Stderr, "reingest: --results-root is required")
		os.Exit(1)
	}

	rawRoot := filepath.Join(*resultsRoot, "raw")
	configDirs, err := os.ReadDir(rawRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reingest: read %s: %v\n", rawRoot, err)
		os.Exit(1)
	}

	total, changed, failed := 0, 0, 0
	for _, configDir := range configDirs {
		if !configDir.IsDir() {
			continue
		}
		configID := configDir.Name()
		repDirs, err := os.ReadDir(filepath.Join(rawRoot, configID))
		if err != nil {
			fmt.Fprintf(os.Stderr, "reingest: read %s: %v\n", filepath.Join(rawRoot, configID), err)
			os.Exit(1)
		}
		for _, repDir := range repDirs {
			if !repDir.IsDir() {
				continue
			}
			runDir := filepath.Join(rawRoot, configID, repDir.Name())
			usagePath := filepath.Join(runDir, "usage.json")
			transcriptPath := filepath.Join(runDir, "transcript.jsonl")
			if _, err := os.Stat(usagePath); err != nil {
				continue // not an ingested run directory
			}
			total++

			did, err := reingestOne(runDir, usagePath, transcriptPath, configID)
			if err != nil {
				fmt.Printf("FAIL  %s/%s: %v\n", configID, repDir.Name(), err)
				failed++
				continue
			}
			if did {
				changed++
			}
		}
	}

	fmt.Printf("\n%d run directories scanned, %d usage.json rewritten, %d unchanged, %d failed\n",
		total, changed, total-changed-failed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

// reingestOne recomputes runDir's usage.json. It reports whether the recomputed document differs from
// the one on disk (by turns or any token count), and always writes the recomputed document plus a
// usage.json.orig backup of what was there before, even when the numbers are unchanged, so a re-run is
// idempotent and every file's backup reflects the immediately-prior state.
func reingestOne(runDir, usagePath, transcriptPath, configID string) (bool, error) {
	oldData, err := os.ReadFile(usagePath)
	if err != nil {
		return false, fmt.Errorf("read old usage.json: %w", err)
	}
	var old ladder.Usage
	if err := json.Unmarshal(oldData, &old); err != nil {
		return false, fmt.Errorf("parse old usage.json: %w", err)
	}

	records, err := ladder.ReadTranscript(transcriptPath)
	if err != nil {
		return false, fmt.Errorf("read transcript: %w", err)
	}

	definitionPath := filepath.Join(runDir, ".claude", "agents", configID+".md")
	grantedTools, err := ladder.GrantedToolsFromDefinition(definitionPath)
	if err != nil {
		return false, fmt.Errorf("granted tools: %w", err)
	}

	fresh, err := ladder.ExtractUsage(records, old.Transcript, old.TranscriptSource, grantedTools)
	if err != nil {
		return false, fmt.Errorf("extract usage: %w", err)
	}

	changed := fresh.NumTurns != old.NumTurns ||
		fresh.Tokens.InputTokens != old.Tokens.InputTokens ||
		fresh.Tokens.OutputTokens != old.Tokens.OutputTokens ||
		fresh.Tokens.CacheReadInputTokens != old.Tokens.CacheReadInputTokens ||
		fresh.Tokens.CacheCreationInputTokens != old.Tokens.CacheCreationInputTokens

	if changed {
		fmt.Printf("%-24s turns %2d -> %2d   cacheRead %7d -> %7d   cacheWrite %7d -> %7d\n",
			filepath.Base(filepath.Dir(runDir))+"/"+filepath.Base(runDir),
			old.NumTurns, fresh.NumTurns,
			old.Tokens.CacheReadInputTokens, fresh.Tokens.CacheReadInputTokens,
			old.Tokens.CacheCreationInputTokens, fresh.Tokens.CacheCreationInputTokens)
	}

	if err := os.WriteFile(usagePath+".orig", oldData, 0o644); err != nil {
		return false, fmt.Errorf("write usage.json.orig backup: %w", err)
	}

	freshData, err := json.MarshalIndent(fresh, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal fresh usage.json: %w", err)
	}
	freshData = append(freshData, '\n')
	if err := os.WriteFile(usagePath, freshData, 0o644); err != nil {
		return false, fmt.Errorf("write fresh usage.json: %w", err)
	}

	return changed, nil
}
