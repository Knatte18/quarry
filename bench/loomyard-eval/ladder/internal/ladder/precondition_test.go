package ladder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckEnvironmentPrecondition_RejectsEachVariableSetNonEmpty(t *testing.T) {
	tests := []struct {
		name string
		env  []string
	}{
		{"state_dir", []string{"QUARRY_STATE_DIR=/tmp/somewhere"}},
		{"build_tags", []string{"QUARRY_BUILD_TAGS=cgo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckEnvironmentPrecondition(tt.env); err == nil {
				t.Errorf("CheckEnvironmentPrecondition(%v) = nil; want an error", tt.env)
			}
		})
	}
}

func TestCheckEnvironmentPrecondition_AcceptsEmptyAndAbsent(t *testing.T) {
	tests := []struct {
		name string
		env  []string
	}{
		{"both_empty", []string{"QUARRY_STATE_DIR=", "QUARRY_BUILD_TAGS="}},
		{"both_absent", []string{"UNRELATED=1"}},
		{"one_empty_one_absent", []string{"QUARRY_STATE_DIR="}},
		{"nil_env", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckEnvironmentPrecondition(tt.env); err != nil {
				t.Errorf("CheckEnvironmentPrecondition(%v) = %v; want nil error", tt.env, err)
			}
		})
	}
}

// writeSkill writes a minimal SKILL.md at <root>/<skillName>/SKILL.md, with frontmatter body.
func writeSkill(t *testing.T, root, skillName, body string) {
	t.Helper()
	dir := filepath.Join(root, skillName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write SKILL.md under %s: %v", dir, err)
	}
}

func TestScanSkillsForLeak_QuarryMentioningFrontmatterIsAnOffender(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "clean-skill", "---\nname: clean-skill\ndescription: does nothing special\n---\n\nbody\n")
	writeSkill(t, root, "leaky-skill", "---\nname: leaky-skill\ndescription: talks to Quarry's MCP server\n---\n\nbody\n")

	pattern := filepath.Join(root, "*", "SKILL.md")
	report, offenders, err := ScanSkillsForLeak([]string{pattern})
	if err != nil {
		t.Fatalf("ScanSkillsForLeak(%v) = _, _, %v; want nil error", []string{pattern}, err)
	}

	wantOffender := filepath.Join(root, "leaky-skill", "SKILL.md")
	if len(offenders) != 1 || offenders[0] != wantOffender {
		t.Errorf("ScanSkillsForLeak offenders = %v; want [%q]", offenders, wantOffender)
	}
	if len(report.RootsScanned) != 1 || report.RootsScanned[0].FileCount != 2 || report.RootsScanned[0].Skipped {
		t.Errorf("ScanSkillsForLeak report = %+v; want one scanned root with FileCount 2", report)
	}
}

func TestScanSkillsForLeak_AbsentRootIsSkippedNotErrored(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "does-not-exist", "*", "SKILL.md")
	report, offenders, err := ScanSkillsForLeak([]string{absent})
	if err != nil {
		t.Fatalf("ScanSkillsForLeak(%v) = _, _, %v; want nil error", []string{absent}, err)
	}
	if len(offenders) != 0 {
		t.Errorf("ScanSkillsForLeak offenders = %v; want none", offenders)
	}
	if len(report.RootsScanned) != 1 || !report.RootsScanned[0].Skipped {
		t.Errorf("ScanSkillsForLeak report = %+v; want the one root recorded skipped", report)
	}
}

func TestScanSkillsForLeak_CleanTreeYieldsNoOffenders(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "clean-skill", "---\nname: clean-skill\ndescription: nothing to see\n---\n\nbody\n")

	pattern := filepath.Join(root, "*", "SKILL.md")
	_, offenders, err := ScanSkillsForLeak([]string{pattern})
	if err != nil {
		t.Fatalf("ScanSkillsForLeak(%v) = _, _, %v; want nil error", []string{pattern}, err)
	}
	if len(offenders) != 0 {
		t.Errorf("ScanSkillsForLeak offenders = %v; want none", offenders)
	}
}

func TestScanSkillsForLeak_ReportNamesEveryRootWithItsCount(t *testing.T) {
	rootA := t.TempDir()
	writeSkill(t, rootA, "skill-a", "---\nname: skill-a\ndescription: fine\n---\n\nbody\n")
	rootBParent := t.TempDir()
	absentB := filepath.Join(rootBParent, "missing", "*", "SKILL.md")

	patternA := filepath.Join(rootA, "*", "SKILL.md")
	report, _, err := ScanSkillsForLeak([]string{patternA, absentB})
	if err != nil {
		t.Fatalf("ScanSkillsForLeak = _, _, %v; want nil error", err)
	}
	if len(report.RootsScanned) != 2 {
		t.Fatalf("len(report.RootsScanned) = %d; want 2", len(report.RootsScanned))
	}
	if report.RootsScanned[0].Root != patternA || report.RootsScanned[0].FileCount != 1 || report.RootsScanned[0].Skipped {
		t.Errorf("report.RootsScanned[0] = %+v; want Root %q, FileCount 1, not skipped", report.RootsScanned[0], patternA)
	}
	if report.RootsScanned[1].Root != absentB || !report.RootsScanned[1].Skipped {
		t.Errorf("report.RootsScanned[1] = %+v; want Root %q, skipped", report.RootsScanned[1], absentB)
	}
}

func TestDefaultSkillRoots_ReturnsTheTwoFixedPatterns(t *testing.T) {
	roots := DefaultSkillRoots()
	if len(roots) != 2 {
		t.Fatalf("len(DefaultSkillRoots()) = %d; want 2", len(roots))
	}
	for _, root := range roots {
		if filepath.Base(root) != "SKILL.md" {
			t.Errorf("DefaultSkillRoots() root %q does not end in SKILL.md", root)
		}
	}
}
