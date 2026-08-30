// plan.go ports the matrix planning half of scripts/run_ladder.py: enumerating the ordered list of
// (config, repetition) pairs the whole suite plans against, and deriving each pair's scratch session
// directory from the ladder's session_dir_template.

package ladder

import (
	"fmt"
	"strconv"
	"strings"
)

// PlanRuns returns the ordered list of all 45 RunPair values in l's matrix: every non-cold config's
// Reps repetitions first, then the cold config's. It is the reporting view of the whole matrix; MainRuns
// and ColdRuns each filter its own partition out of this same ordered list, so neither driver iterates
// PlanRuns's output directly.
func PlanRuns(l *Ladder) []RunPair {
	var pairs []RunPair
	for _, config := range l.Configs {
		if config.Cold {
			continue
		}
		for n := 1; n <= l.Reps; n++ {
			pairs = append(pairs, RunPair{Config: config, N: n})
		}
	}
	for _, config := range l.Configs {
		if !config.Cold {
			continue
		}
		for n := 1; n <= l.Reps; n++ {
			pairs = append(pairs, RunPair{Config: config, N: n})
		}
	}
	return pairs
}

// MainRuns is the main-matrix partition of PlanRuns -- every non-cold pair.
func MainRuns(l *Ladder) []RunPair {
	var pairs []RunPair
	for _, pair := range PlanRuns(l) {
		if !pair.Config.Cold {
			pairs = append(pairs, pair)
		}
	}
	return pairs
}

// ColdRuns is the cold-cell partition of PlanRuns -- every cold pair.
func ColdRuns(l *Ladder) []RunPair {
	var pairs []RunPair
	for _, pair := range PlanRuns(l) {
		if pair.Config.Cold {
			pairs = append(pairs, pair)
		}
	}
	return pairs
}

// SessionDir derives the scratch working directory for one session by substituting "{config_id}" with
// configID and "{n}" with n in l.SessionDirTemplate -- the single derivation site every scratch
// directory path in the suite goes through, so no caller ever formats the template itself.
//
// n is the repetition index uniformly, not just a main-matrix rep: the scoring session uses a configID
// of "scoring" with n of 1, and the two probe sessions use "probe-allowlist" and "probe-denylist" with n
// of 1, alongside the 45 run sessions' own (config.ID, n) pairs.
//
// Returns an error naming l.SessionDirTemplate as unset when it is empty.
func SessionDir(l *Ladder, configID string, n int) (string, error) {
	if l.SessionDirTemplate == "" {
		return "", fmt.Errorf("session dir: ladder.yaml: session_dir_template is unset")
	}
	replacer := strings.NewReplacer("{config_id}", configID, "{n}", strconv.Itoa(n))
	return replacer.Replace(l.SessionDirTemplate), nil
}
