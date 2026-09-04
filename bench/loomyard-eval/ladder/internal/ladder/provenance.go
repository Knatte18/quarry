// provenance.go declares the committed provenance record one results root carries, the collector
// that fills an invocation's own facts at startup, the merge policy that lets a results root be
// resumed across more than one invocation, and the memory-path scan, server-hash-drift warning and
// fingerprint-drift comparison that feed the run's non-fatal findings.
//
// Three fields exist in hashed or relative form precisely because this file is committed and no
// tracked file may carry a machine path -- see the overview's no-machine-paths-in-tracked-output
// decision.

package ladder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ProvenanceFile is the file name a results root's provenance record is written under: the single
// spelling of that name. Card 28's summary file and card 29's table file each declare their own
// constant the same way, in the file that writes them.
const ProvenanceFile = "provenance.json"

// Provenance is the committed record of every invocation a results root has ever run, and the
// top-level facts MergeProvenance derives from that invocation list. The resolved auto-memory
// directories are deliberately not carried here as paths -- only their hashes -- because this file
// is committed and no tracked file may carry a machine path; the paths themselves live in
// raw/memory-paths.json inside the results root's untracked raw tree.
type Provenance struct {
	// WrittenAt is the latest invocation's write time.
	WrittenAt string `json:"written_at"`
	// LadderFile is the ladder file's path, relative to the quarry repository root when the
	// operator's value resolves under it, and its base name otherwise -- never an absolute path.
	LadderFile string `json:"ladder_file"`
	// QuarryCommit is the latest invocation's quarry repository commit.
	QuarryCommit string `json:"quarry_commit"`
	// QuarryDirty is the latest invocation's quarry repository dirty flag.
	QuarryDirty bool `json:"quarry_dirty"`
	// QuarryDirtyFiles is the latest invocation's list of dirty file paths.
	QuarryDirtyFiles []string `json:"quarry_dirty_files"`
	// LoomyardCommit is the latest invocation's target repository commit.
	LoomyardCommit string `json:"loomyard_commit"`
	// LoomyardRepoSHA256 is the hex sha256 of the resolved target repository path -- never the path
	// itself, since this file is committed.
	LoomyardRepoSHA256 string `json:"loomyard_repo_sha256"`
	// Hostname is the latest invocation's host name.
	Hostname string `json:"hostname"`
	// GoVersion is the latest invocation's Go toolchain version.
	GoVersion string `json:"go_version"`
	// ClaudeVersion is the latest invocation's claude CLI version.
	ClaudeVersion string `json:"claude_version"`
	// ServerHashes is the hex sha256 of the built MCP server binary, keyed by the cell-and-rep pair
	// that used it, merged across every invocation.
	ServerHashes map[string]string `json:"server_hashes"`
	// SelectedCells is the union of every cell id any invocation has ever selected.
	SelectedCells []string `json:"selected_cells"`
	// RepsEffective is the per-cell repetition count, identical across every invocation.
	RepsEffective int `json:"reps_effective"`
	// MemoryPathHashes is the union of the hex sha256 of every resolved auto-memory directory any
	// invocation has ever observed.
	MemoryPathHashes []string `json:"memory_path_hashes"`
	// ServerName is the MCP server's own name, identical across every invocation.
	ServerName string `json:"server_name"`
	// SessionFingerprints is one SessionFingerprint per completed repetition, keyed by the same
	// cell-and-rep pair as ServerHashes.
	SessionFingerprints map[string]SessionFingerprint `json:"session_fingerprints"`
	// Invocations is every invocation this root has ever run, in the order they completed.
	Invocations []Invocation `json:"invocations"`
}

// SessionFingerprint is what one repetition's session-init record reveals about how that repetition
// was actually configured: the CLI version, the model, the permission mode, the advertised tool and
// server lists, whether the memory-path map was non-empty, and the skill and slash-command counts.
type SessionFingerprint struct {
	// ClaudeCodeVersion is the session-init record's reported CLI version.
	ClaudeCodeVersion string `json:"claude_code_version"`
	// Model is the session-init record's reported model id.
	Model string `json:"model"`
	// PermissionMode is the session-init record's reported permission mode.
	PermissionMode string `json:"permission_mode"`
	// Tools is the session-init record's advertised built-in tool list.
	Tools []string `json:"tools"`
	// MCPServers is the session-init record's advertised MCP server name list.
	MCPServers []string `json:"mcp_servers"`
	// HasMemoryPaths reports whether the session-init record's memory-path map was non-empty.
	HasMemoryPaths bool `json:"has_memory_paths"`
	// SkillCount is the session-init record's advertised skill count.
	SkillCount int `json:"skill_count"`
	// SlashCommandCount is the session-init record's advertised slash-command count.
	SlashCommandCount int `json:"slash_command_count"`
}

// NewSessionFingerprint lifts a SessionFingerprint from one repetition's decoded session-init
// record.
func NewSessionFingerprint(init *SessionInit) SessionFingerprint {
	return SessionFingerprint{
		ClaudeCodeVersion: init.ClaudeCodeVersion,
		Model:             init.Model,
		PermissionMode:    init.PermissionMode,
		Tools:             append([]string(nil), init.Tools...),
		MCPServers:        append([]string(nil), init.MCPServers...),
		HasMemoryPaths:    len(init.MemoryPaths) > 0,
		SkillCount:        len(init.Skills),
		SlashCommandCount: len(init.SlashCommands),
	}
}

// Invocation is one run's own facts, collected once at startup by CollectInvocation and appended to
// a Provenance's Invocations list by MergeProvenance. Every top-level Provenance field MergeProvenance
// derives has a carrier here, so no top-level field is left without a producer.
type Invocation struct {
	// WrittenAt is this invocation's write time.
	WrittenAt string `json:"written_at"`
	// LadderFile is this invocation's ladder file, in the same relative-or-base-name form as
	// Provenance.LadderFile.
	LadderFile string `json:"ladder_file"`
	// SelectedCells is the cell ids this invocation selected.
	SelectedCells []string `json:"selected_cells"`
	// RepsEffective is this invocation's per-cell repetition count.
	RepsEffective int `json:"reps_effective"`
	// QuarryCommit is this invocation's quarry repository commit.
	QuarryCommit string `json:"quarry_commit"`
	// QuarryDirty reports whether the quarry repository carried an uncommitted change.
	QuarryDirty bool `json:"quarry_dirty"`
	// QuarryDirtyFiles is the quarry repository's dirty file paths.
	QuarryDirtyFiles []string `json:"quarry_dirty_files"`
	// LoomyardCommit is this invocation's target repository commit.
	LoomyardCommit string `json:"loomyard_commit"`
	// LoomyardRepoSHA256 is the hex sha256 of the resolved target repository path.
	LoomyardRepoSHA256 string `json:"loomyard_repo_sha256"`
	// ClaudeVersion is this invocation's claude CLI version, probed rather than assumed.
	ClaudeVersion string `json:"claude_version"`
	// GoVersion is this invocation's Go toolchain version.
	GoVersion string `json:"go_version"`
	// Hostname is this invocation's host name.
	Hostname string `json:"hostname"`
	// ServerName is the MCP server's own name, copied from the loaded ladder file.
	ServerName string `json:"server_name"`
	// MemoryPathHashes is the hex sha256 of every resolved auto-memory directory this invocation
	// observed. Unknown at CollectInvocation time; the run loop fills it once the first completed
	// repetition reveals the paths.
	MemoryPathHashes []string `json:"memory_path_hashes"`
	// ServerHashes is this invocation's own server-hash observations, keyed by cell-and-rep pair.
	// Unknown at CollectInvocation time; the run loop fills it as it builds and assigns the binary.
	ServerHashes map[string]string `json:"server_hashes"`
}

// CollectInput carries every fact CollectInvocation needs but cannot compute itself.
type CollectInput struct {
	// QuarryRepoRoot is the quarry repository's root path.
	QuarryRepoRoot string
	// LadderFilePath is the operator's own ladder-file path, absolute or relative.
	LadderFilePath string
	// TargetRepoPath is the resolved target (loomyard) repository path.
	TargetRepoPath string
	// ServerName is the loaded ladder file's MCP server name.
	ServerName string
	// SelectedCells is the cell ids this invocation selected.
	SelectedCells []string
	// RepsEffective is the effective per-cell repetition count.
	RepsEffective int
	// ClaudeBinPath is the claude binary path to probe for its version.
	ClaudeBinPath string
}

// ReadProvenance reads the provenance record at resultsRoot/provenance.json. A results root that has
// never been written to is not a fault: when the file is absent, ReadProvenance returns a nil record
// and no error. A file that is present but unparseable is an error naming the file.
func ReadProvenance(resultsRoot string) (*Provenance, error) {
	path := filepath.Join(resultsRoot, ProvenanceFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read provenance %s: %w", path, err)
	}

	var p Provenance
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("read provenance %s: %w", path, err)
	}
	return &p, nil
}

// WriteProvenance writes p to resultsRoot/provenance.json.
func WriteProvenance(resultsRoot string, p *Provenance) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("write provenance: %w", err)
	}
	path := filepath.Join(resultsRoot, ProvenanceFile)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write provenance %s: %w", path, err)
	}
	return nil
}

// CollectInvocation is the single producer of one invocation's own facts: the quarry repository's
// commit, dirty flag and dirty-file list from its own status and revision output through r; the
// target repository's commit the same way; the hex sha256 of the resolved target-repository path;
// the ladder-file path reduced to its repository-relative or base-name form; the server name copied
// from the loaded file; the host name; the Go version from the runtime rather than a subprocess; and
// the CLI version from a version invocation of the claude binary through r, because the probe host
// and the plan's reference host reported different versions and the value must be recorded rather
// than assumed. It leaves ServerHashes and MemoryPathHashes empty: those two fields are unknown at
// startup and the run loop fills them as the run proceeds.
func CollectInvocation(ctx context.Context, r Runner, in CollectInput) (Invocation, error) {
	quarryCommit, err := gitHead(ctx, r, in.QuarryRepoRoot)
	if err != nil {
		return Invocation{}, fmt.Errorf("collect invocation: quarry commit: %w", err)
	}
	quarryStatus, err := gitStatusPorcelain(ctx, r, in.QuarryRepoRoot)
	if err != nil {
		return Invocation{}, fmt.Errorf("collect invocation: quarry status: %w", err)
	}
	quarryDirtyFiles := parsePorcelainFiles(quarryStatus)

	loomyardCommit, err := gitHead(ctx, r, in.TargetRepoPath)
	if err != nil {
		return Invocation{}, fmt.Errorf("collect invocation: loomyard commit: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return Invocation{}, fmt.Errorf("collect invocation: hostname: %w", err)
	}

	ladderFileRel, err := ladderFileRelativeOrBase(in.QuarryRepoRoot, in.LadderFilePath)
	if err != nil {
		return Invocation{}, fmt.Errorf("collect invocation: ladder file path: %w", err)
	}

	claudeVersion, err := probeClaudeVersion(ctx, r, in.ClaudeBinPath)
	if err != nil {
		return Invocation{}, fmt.Errorf("collect invocation: claude version: %w", err)
	}

	return Invocation{
		WrittenAt:          time.Now().UTC().Format(time.RFC3339),
		LadderFile:         ladderFileRel,
		SelectedCells:      append([]string(nil), in.SelectedCells...),
		RepsEffective:      in.RepsEffective,
		QuarryCommit:       quarryCommit,
		QuarryDirty:        len(quarryDirtyFiles) > 0,
		QuarryDirtyFiles:   quarryDirtyFiles,
		LoomyardCommit:     loomyardCommit,
		LoomyardRepoSHA256: sha256Hex(in.TargetRepoPath),
		ClaudeVersion:      claudeVersion,
		GoVersion:          runtime.Version(),
		Hostname:           hostname,
		ServerName:         in.ServerName,
	}, nil
}

// gitHead returns the current commit of the git repository at dir, run through r.
func gitHead(ctx context.Context, r Runner, dir string) (string, error) {
	var out bytes.Buffer
	if err := r.Run(ctx, Cmd{Dir: dir, Name: "git", Args: []string{"rev-parse", "HEAD"}, Stdout: &out}); err != nil {
		return "", fmt.Errorf("git rev-parse HEAD in %s: %w", dir, err)
	}
	return strings.TrimSpace(out.String()), nil
}

// gitStatusPorcelain returns the porcelain status output of the git repository at dir, run through
// r.
func gitStatusPorcelain(ctx context.Context, r Runner, dir string) (string, error) {
	var out bytes.Buffer
	if err := r.Run(ctx, Cmd{Dir: dir, Name: "git", Args: []string{"status", "--porcelain"}, Stdout: &out}); err != nil {
		return "", fmt.Errorf("git status --porcelain in %s: %w", dir, err)
	}
	return out.String(), nil
}

// parsePorcelainFiles extracts the file path from every non-blank line of `git status --porcelain`
// output, stripping the two-character status code and the separating space.
func parsePorcelainFiles(porcelain string) []string {
	var files []string
	for _, line := range strings.Split(porcelain, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) > 3 {
			files = append(files, line[3:])
		} else {
			files = append(files, strings.TrimSpace(line))
		}
	}
	return files
}

// probeClaudeVersion runs claudeBinPath --version through r and returns the trimmed output.
func probeClaudeVersion(ctx context.Context, r Runner, claudeBinPath string) (string, error) {
	var out bytes.Buffer
	if err := r.Run(ctx, Cmd{Name: claudeBinPath, Args: []string{"--version"}, Stdout: &out}); err != nil {
		return "", fmt.Errorf("probe claude version: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// sha256Hex returns the hex sha256 digest of s.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ladderFileRelativeOrBase reduces ladderFilePath to its path relative to quarryRepoRoot when it
// resolves under that root, and to its base name otherwise -- so an operator who passes an absolute
// path never writes their home directory into this committed record.
func ladderFileRelativeOrBase(quarryRepoRoot, ladderFilePath string) (string, error) {
	absRoot, err := filepath.Abs(quarryRepoRoot)
	if err != nil {
		return "", err
	}
	absFile, err := filepath.Abs(ladderFilePath)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absFile)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(rel), nil
	}
	return filepath.Base(ladderFilePath), nil
}

// MergeProvenance appends next to existing's invocation list and derives every top-level field from
// the resulting invocation list, never overwriting an existing record: a narrower second run would
// otherwise erase cells from the incomplete list and make an unfinished root read as finished.
// selected_cells and memory_path_hashes are the union across every invocation; server_hashes merges
// by key; written_at, quarry_commit, quarry_dirty, quarry_dirty_files, loomyard_commit,
// claude_version, go_version and hostname take the latest invocation's value, since each answers
// "what was true the last time this root was written to" while every invocation's own value stays
// readable in the array. ladder_file, loomyard_repo_sha256 and server_name must be identical across
// invocations; reps_effective must be identical across invocations; a differing value in any of these
// four is an error naming both values -- a root assembled from two different ladder files, two
// different checkouts, two different server names or two different per-cell sample sizes is not one
// root. existing may be nil, for a fresh results root's first invocation.
func MergeProvenance(existing *Provenance, next Invocation) (*Provenance, error) {
	if existing == nil {
		return &Provenance{
			WrittenAt:           next.WrittenAt,
			LadderFile:          next.LadderFile,
			QuarryCommit:        next.QuarryCommit,
			QuarryDirty:         next.QuarryDirty,
			QuarryDirtyFiles:    append([]string(nil), next.QuarryDirtyFiles...),
			LoomyardCommit:      next.LoomyardCommit,
			LoomyardRepoSHA256:  next.LoomyardRepoSHA256,
			Hostname:            next.Hostname,
			GoVersion:           next.GoVersion,
			ClaudeVersion:       next.ClaudeVersion,
			ServerHashes:        cloneStringMap(next.ServerHashes),
			SelectedCells:       sortedUnique(next.SelectedCells),
			RepsEffective:       next.RepsEffective,
			MemoryPathHashes:    sortedUnique(next.MemoryPathHashes),
			ServerName:          next.ServerName,
			SessionFingerprints: map[string]SessionFingerprint{},
			Invocations:         []Invocation{next},
		}, nil
	}

	if existing.LadderFile != next.LadderFile {
		return nil, fmt.Errorf("merge provenance: ladder_file differs: %q vs %q", existing.LadderFile, next.LadderFile)
	}
	if existing.LoomyardRepoSHA256 != next.LoomyardRepoSHA256 {
		return nil, fmt.Errorf("merge provenance: loomyard_repo_sha256 differs: %q vs %q", existing.LoomyardRepoSHA256, next.LoomyardRepoSHA256)
	}
	if existing.ServerName != next.ServerName {
		return nil, fmt.Errorf("merge provenance: server_name differs: %q vs %q", existing.ServerName, next.ServerName)
	}
	if existing.RepsEffective != next.RepsEffective {
		return nil, fmt.Errorf("merge provenance: reps_effective differs: %d vs %d", existing.RepsEffective, next.RepsEffective)
	}

	invocations := append(append([]Invocation(nil), existing.Invocations...), next)

	merged := &Provenance{
		WrittenAt:           next.WrittenAt,
		LadderFile:          existing.LadderFile,
		QuarryCommit:        next.QuarryCommit,
		QuarryDirty:         next.QuarryDirty,
		QuarryDirtyFiles:    append([]string(nil), next.QuarryDirtyFiles...),
		LoomyardCommit:      next.LoomyardCommit,
		LoomyardRepoSHA256:  existing.LoomyardRepoSHA256,
		Hostname:            next.Hostname,
		GoVersion:           next.GoVersion,
		ClaudeVersion:       next.ClaudeVersion,
		ServerHashes:        cloneStringMap(existing.ServerHashes),
		SelectedCells:       sortedUnion(existing.SelectedCells, next.SelectedCells),
		RepsEffective:       existing.RepsEffective,
		MemoryPathHashes:    sortedUnion(existing.MemoryPathHashes, next.MemoryPathHashes),
		ServerName:          existing.ServerName,
		SessionFingerprints: cloneFingerprintMap(existing.SessionFingerprints),
		Invocations:         invocations,
	}
	for k, v := range next.ServerHashes {
		merged.ServerHashes[k] = v
	}
	return merged, nil
}

// cloneStringMap returns a copy of m, or an empty non-nil map when m is nil, so a caller can always
// range or assign into the result without a nil check.
func cloneStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// cloneFingerprintMap returns a copy of m, or an empty non-nil map when m is nil.
func cloneFingerprintMap(m map[string]SessionFingerprint) map[string]SessionFingerprint {
	out := make(map[string]SessionFingerprint, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// sortedUnique returns a sorted copy of ss with duplicates removed.
func sortedUnique(ss []string) []string {
	return sortedUnion(ss, nil)
}

// sortedUnion returns the sorted union of a and b, with duplicates removed.
func sortedUnion(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		seen[s] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// errStopScan stops ScanMemoryPaths's directory walk as soon as a matching file is found, without
// treating the early stop as an I/O failure.
var errStopScan = errors.New("stop scan: match found")

// ScanMemoryPaths walks each named directory and returns a fatal finding naming the first file whose
// content matches the bare token "quarry". A named path that does not exist on disk is a fatal
// *Finding, not an error and not silence: the harness cannot tell "this directory is clean" from
// "this directory was never scanned", and reporting the second as the first is precisely the silent
// failure V1's derived path produced. An empty path list is neither a finding nor an error -- the run
// continues and the fact is recorded in the fingerprint, since an absent memory directory is the
// outcome the check wants. The error return is reserved for an I/O failure reading a directory or
// file that does exist.
func ScanMemoryPaths(paths []string) (*Finding, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			if os.IsNotExist(err) {
				return &Finding{
					Gate:    "memory_path_scan",
					Fatal:   true,
					Message: fmt.Sprintf("!! memory path %s does not exist", p),
				}, nil
			}
			return nil, fmt.Errorf("scan memory path %s: %w", p, err)
		}

		var found *Finding
		walkErr := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if MatchesBareToken(string(data), "quarry") {
				found = &Finding{
					Gate:    "memory_path_scan",
					Fatal:   true,
					Message: fmt.Sprintf("!! memory path file %s contains the bare token \"quarry\"", path),
				}
				return errStopScan
			}
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, errStopScan) {
			return nil, fmt.Errorf("scan memory path %s: %w", p, walkErr)
		}
		if found != nil {
			return found, nil
		}
	}

	return nil, nil
}

// WarnOnServerHashDrift returns the always-non-fatal server_hash_drift finding when more than one
// distinct server binary hash appears across p's server_hashes map, and nil when at most one does.
func WarnOnServerHashDrift(p *Provenance) *Finding {
	seen := make(map[string]bool, len(p.ServerHashes))
	for _, h := range p.ServerHashes {
		seen[h] = true
	}
	if len(seen) <= 1 {
		return nil
	}
	return &Finding{
		Gate:  "server_hash_drift",
		Fatal: false,
		Message: fmt.Sprintf(
			"server_hash_drift: %d distinct server binary hashes appear in this root", len(seen),
		),
	}
}

// CompareFingerprints reports each repetition's session fingerprint drift from the root's first
// fingerprint -- ordered by cell-and-rep key -- as always-non-fatal observations. It returns nil
// when the root carries fewer than two fingerprints.
func CompareFingerprints(p *Provenance) []Finding {
	if len(p.SessionFingerprints) < 2 {
		return nil
	}

	keys := make([]string, 0, len(p.SessionFingerprints))
	for k := range p.SessionFingerprints {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	first := p.SessionFingerprints[keys[0]]
	var findings []Finding
	for _, k := range keys[1:] {
		diff := diffSessionFingerprint(first, p.SessionFingerprints[k])
		if diff == "" {
			continue
		}
		findings = append(findings, Finding{
			Gate:    "session_fingerprint_drift",
			Fatal:   false,
			Message: fmt.Sprintf("session_fingerprint_drift: %s differs from %s: %s", k, keys[0], diff),
		})
	}
	return findings
}

// diffSessionFingerprint returns a human-readable description of every field that differs between a
// and b, or the empty string when none do.
func diffSessionFingerprint(a, b SessionFingerprint) string {
	var parts []string
	if a.ClaudeCodeVersion != b.ClaudeCodeVersion {
		parts = append(parts, fmt.Sprintf("claude_code_version %q vs %q", a.ClaudeCodeVersion, b.ClaudeCodeVersion))
	}
	if a.Model != b.Model {
		parts = append(parts, fmt.Sprintf("model %q vs %q", a.Model, b.Model))
	}
	if a.PermissionMode != b.PermissionMode {
		parts = append(parts, fmt.Sprintf("permission_mode %q vs %q", a.PermissionMode, b.PermissionMode))
	}
	if !stringSlicesEqual(a.Tools, b.Tools) {
		parts = append(parts, fmt.Sprintf("tools %v vs %v", a.Tools, b.Tools))
	}
	if !stringSlicesEqual(a.MCPServers, b.MCPServers) {
		parts = append(parts, fmt.Sprintf("mcp_servers %v vs %v", a.MCPServers, b.MCPServers))
	}
	if a.HasMemoryPaths != b.HasMemoryPaths {
		parts = append(parts, fmt.Sprintf("has_memory_paths %v vs %v", a.HasMemoryPaths, b.HasMemoryPaths))
	}
	if a.SkillCount != b.SkillCount {
		parts = append(parts, fmt.Sprintf("skill_count %d vs %d", a.SkillCount, b.SkillCount))
	}
	if a.SlashCommandCount != b.SlashCommandCount {
		parts = append(parts, fmt.Sprintf("slash_command_count %d vs %d", a.SlashCommandCount, b.SlashCommandCount))
	}
	return strings.Join(parts, "; ")
}

// stringSlicesEqual reports whether a and b contain the same strings in the same order.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
