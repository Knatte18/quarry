package ladder

import "testing"

// testSessionRepoRoot is an arbitrary but fixed repo root these tests join a relative
// session_dir_template against -- its value has no bearing on the properties under test
// (distinctness, error-on-unset), only that it is applied uniformly.
const testSessionRepoRoot = "/fixture-repo-root"

func TestSessionDir_All45RunPairsDeriveDistinctDirectories(t *testing.T) {
	l := mustLoadLadder(t)

	seen := make(map[string]bool)
	for _, pair := range PlanRuns(l) {
		dir, err := SessionDir(l, testSessionRepoRoot, pair.Config.ID, pair.N)
		if err != nil {
			t.Fatalf("SessionDir(l, %q, %d) = _, %v; want nil error", pair.Config.ID, pair.N, err)
		}
		if seen[dir] {
			t.Errorf("SessionDir(l, %q, %d) = %q; already produced by another run pair", pair.Config.ID, pair.N, dir)
		}
		seen[dir] = true
	}
	if len(seen) != 45 {
		t.Errorf("SessionDir produced %d distinct directories across PlanRuns(l); want 45", len(seen))
	}
}

func TestSessionDir_ScoringAndProbeSessionsAreDistinctFromRunsAndEachOther(t *testing.T) {
	l := mustLoadLadder(t)

	runDirs := make(map[string]bool)
	for _, pair := range PlanRuns(l) {
		dir, err := SessionDir(l, testSessionRepoRoot, pair.Config.ID, pair.N)
		if err != nil {
			t.Fatalf("SessionDir(l, %q, %d) = _, %v; want nil error", pair.Config.ID, pair.N, err)
		}
		runDirs[dir] = true
	}

	scoringDir, err := SessionDir(l, testSessionRepoRoot, "scoring", 1)
	if err != nil {
		t.Fatalf("SessionDir(l, \"scoring\", 1) = _, %v; want nil error", err)
	}
	allowlistDir, err := SessionDir(l, testSessionRepoRoot, "probe-allowlist", 1)
	if err != nil {
		t.Fatalf("SessionDir(l, \"probe-allowlist\", 1) = _, %v; want nil error", err)
	}
	denylistDir, err := SessionDir(l, testSessionRepoRoot, "probe-denylist", 1)
	if err != nil {
		t.Fatalf("SessionDir(l, \"probe-denylist\", 1) = _, %v; want nil error", err)
	}

	special := map[string]string{
		"scoring":         scoringDir,
		"probe-allowlist": allowlistDir,
		"probe-denylist":  denylistDir,
	}
	for name, dir := range special {
		if runDirs[dir] {
			t.Errorf("%s session directory %q collides with a run session directory", name, dir)
		}
	}
	if scoringDir == allowlistDir || scoringDir == denylistDir || allowlistDir == denylistDir {
		t.Errorf("scoring/probe session directories are not pairwise distinct: scoring=%q allowlist=%q denylist=%q", scoringDir, allowlistDir, denylistDir)
	}
}

func TestSessionDir_ErrorsWhenTemplateUnset(t *testing.T) {
	l := mustLoadLadder(t)
	l.SessionDirTemplate = ""

	if _, err := SessionDir(l, testSessionRepoRoot, "a0-none", 1); err == nil {
		t.Errorf("SessionDir(l, \"a0-none\", 1) = _, nil; want an error naming the unset template")
	}
}

func TestPlanRuns_Yields45PairsWithColdStrictlyLast(t *testing.T) {
	l := mustLoadLadder(t)

	pairs := PlanRuns(l)
	if len(pairs) != 45 {
		t.Fatalf("PlanRuns(l) yielded %d pairs; want 45", len(pairs))
	}

	firstColdIndex := -1
	for i, pair := range pairs {
		if pair.Config.Cold {
			firstColdIndex = i
			break
		}
	}
	if firstColdIndex == -1 {
		t.Fatalf("PlanRuns(l) yielded no cold pair; want exactly one cold config's repetitions")
	}
	for i, pair := range pairs {
		wantCold := i >= firstColdIndex
		if pair.Config.Cold != wantCold {
			t.Errorf("PlanRuns(l)[%d].Config.Cold = %v; want %v (cold pairs must be strictly last)", i, pair.Config.Cold, wantCold)
		}
	}
}

func TestPlanRuns_IsDeterministic(t *testing.T) {
	l := mustLoadLadder(t)

	first := PlanRuns(l)
	second := PlanRuns(l)

	if len(first) != len(second) {
		t.Fatalf("PlanRuns(l) returned %d pairs then %d pairs; want the same length both times", len(first), len(second))
	}
	for i := range first {
		if first[i].Config.ID != second[i].Config.ID || first[i].N != second[i].N {
			t.Errorf("PlanRuns(l)[%d] = (%s, %d) then (%s, %d); want identical ordering across calls",
				i, first[i].Config.ID, first[i].N, second[i].Config.ID, second[i].N)
		}
	}
}

func TestMainRunsAndColdRuns_PartitionPlanRunsDisjointly(t *testing.T) {
	l := mustLoadLadder(t)

	mains := MainRuns(l)
	colds := ColdRuns(l)

	if len(mains) != 42 {
		t.Errorf("MainRuns(l) yielded %d pairs; want 42", len(mains))
	}
	if len(colds) != 3 {
		t.Errorf("ColdRuns(l) yielded %d pairs; want 3", len(colds))
	}

	for _, pair := range mains {
		if pair.Config.Cold {
			t.Errorf("MainRuns(l) included cold config %q", pair.Config.ID)
		}
	}
	for _, pair := range colds {
		if !pair.Config.Cold {
			t.Errorf("ColdRuns(l) included non-cold config %q", pair.Config.ID)
		}
	}

	seen := make(map[string]bool, len(mains)+len(colds))
	for _, pair := range mains {
		seen[pair.Config.ID] = true
	}
	for _, pair := range colds {
		if seen[pair.Config.ID] {
			t.Errorf("ColdRuns(l) config %q also appears in MainRuns(l); want a disjoint partition", pair.Config.ID)
		}
	}

	if len(mains)+len(colds) != len(PlanRuns(l)) {
		t.Errorf("MainRuns(l)+ColdRuns(l) = %d pairs; want the same total as PlanRuns(l) (%d)", len(mains)+len(colds), len(PlanRuns(l)))
	}
}
