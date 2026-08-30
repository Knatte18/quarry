// precondition.go ports the two launch preconditions the session-provisioning batch checks before a
// session is materialised: the environment scrub's second application point, and the skill-listing leak
// scan. Neither has a direct counterpart in scripts/run_ladder.py, which dispatched a subprocess
// directly and so never needed to reason about what a live session's own skill discovery would expose.

package ladder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckEnvironmentPrecondition hard-fails, naming the offending variable, when either QUARRY_STATE_DIR
// or QUARRY_BUILD_TAGS is set non-empty in env.
//
// This is the second of the three points the environment scrub applies at -- the first is
// MCPConfigDocument's own env block (server.go), the third is the warm-up client's spawn environment
// (Warm, warm.go). It exists partly as cover for an unestablished question this port does not resolve:
// whether an MCP server declaration's environment block replaces or augments the environment the
// launching CLI would otherwise inherit into the spawned server. Because that is unresolved,
// CheckEnvironmentPrecondition checks the operator's own shell environment at launch time -- the env a
// caller is expected to pass as os.Environ() -- rather than the environment the spawned server process
// actually receives, which this function has no way to observe.
//
// An empty or entirely absent value passes.
func CheckEnvironmentPrecondition(env []string) error {
	for _, key := range scrubbedEnvKeys {
		if value, ok := envLookup(env, key); ok && value != "" {
			return fmt.Errorf(
				"ladder: check environment precondition: %s is set non-empty in the operator's shell -- clear it before launching a session",
				key,
			)
		}
	}
	return nil
}

// SkillRootScan records one root ScanSkillsForLeak scanned: whether it was skipped because it does not
// exist, and, when scanned, how many SKILL.md files matched under it.
type SkillRootScan struct {
	// Root is the glob pattern this scan entry covers, exactly as passed to ScanSkillsForLeak.
	Root string
	// Skipped is true when the root's base directory did not exist, so it was scanned as zero files
	// rather than as an error.
	Skipped bool
	// FileCount is the number of SKILL.md files the glob matched under Root; always zero when Skipped.
	FileCount int
}

// ScanReport is the full result of one ScanSkillsForLeak call: every root scanned, in the order given,
// so a report with zero offenders is auditable rather than silently vacuous -- a caller can see that a
// clean result came from actually finding files and reading them, not from every root being skipped.
type ScanReport struct {
	// RootsScanned holds one SkillRootScan per root ScanSkillsForLeak was given, in the given order.
	RootsScanned []SkillRootScan
}

// DefaultSkillRoots returns the two glob patterns Claude Code's own skill discovery reads: the
// user-scope skills root and the plugin-cache root, rooted at the current user's home directory.
// Installed skills actually live under the plugin cache on this machine, so scanning the user-scope root
// alone would pass ScanSkillsForLeak vacuously.
func DefaultSkillRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		// An unresolvable home directory yields a root whose base directory can never exist, so
		// ScanSkillsForLeak reports it skipped rather than panicking -- consistent with this function's
		// own "skip rather than error" contract for a root that is not present.
		home = ""
	}
	return []string{
		filepath.Join(home, ".claude", "skills", "*", "SKILL.md"),
		filepath.Join(home, ".claude", "plugins", "cache", "*", "*", "*", "skills", "*", "SKILL.md"),
	}
}

// globRootBaseDir returns the directory portion of a glob pattern that precedes its first wildcard
// character, so ScanSkillsForLeak can test that directory's existence before globbing it.
func globRootBaseDir(pattern string) string {
	idx := strings.IndexAny(pattern, "*?[")
	if idx == -1 {
		return pattern
	}
	return filepath.Dir(pattern[:idx])
}

// ScanSkillsForLeak scans each glob pattern in roots for SKILL.md files whose contents mention "quarry"
// case-insensitively, and returns a ScanReport of every root scanned alongside the offending file paths.
//
// Every subagent transcript carries a record enumerating the session's available skills by name and
// description verbatim, with no tool call involved, so a skill's frontmatter is a leak channel into a
// blinded run agent's transcript that no working-directory hygiene can close.
//
// A root whose base directory (the portion of its glob pattern before the first wildcard) does not exist
// is skipped rather than erroring, and recorded in the returned report as such, so a vacuous pass is
// visible rather than silent.
//
// This scan's real limit: a session's skill listing also enumerates built-in and managed skills that
// live under neither scanned root, so a clean report bounds only the skills this harness or the operator
// installed under roots, never the whole channel. The blinding gate over the transcript remains the
// detector for the rest.
//
// The scan is advisory for a rung and hard-failing for a config whose allowed set is empty; that
// distinction is the caller's to apply, since this function has no config to consult.
func ScanSkillsForLeak(roots []string) (report ScanReport, offenders []string, err error) {
	for _, root := range roots {
		base := globRootBaseDir(root)
		if _, statErr := os.Stat(base); os.IsNotExist(statErr) {
			report.RootsScanned = append(report.RootsScanned, SkillRootScan{Root: root, Skipped: true})
			continue
		} else if statErr != nil {
			return ScanReport{}, nil, fmt.Errorf("ladder: scan skills for leak: stat %s: %w", base, statErr)
		}

		matches, globErr := filepath.Glob(root)
		if globErr != nil {
			return ScanReport{}, nil, fmt.Errorf("ladder: scan skills for leak: glob %s: %w", root, globErr)
		}
		report.RootsScanned = append(report.RootsScanned, SkillRootScan{Root: root, FileCount: len(matches)})

		for _, match := range matches {
			data, readErr := os.ReadFile(match)
			if readErr != nil {
				return ScanReport{}, nil, fmt.Errorf("ladder: scan skills for leak: read %s: %w", match, readErr)
			}
			if strings.Contains(strings.ToLower(string(data)), "quarry") {
				offenders = append(offenders, match)
			}
		}
	}
	return report, offenders, nil
}
