package ladder

import "testing"

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
