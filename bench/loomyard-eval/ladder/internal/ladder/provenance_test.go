package ladder

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestReadProvenance_MissingIsNilNoError(t *testing.T) {
	root := t.TempDir()
	p, err := ReadProvenance(root)
	if err != nil {
		t.Fatalf("ReadProvenance() = %v; want no error", err)
	}
	if p != nil {
		t.Errorf("ReadProvenance() = %+v; want nil", p)
	}
}

func TestReadProvenance_MalformedErrorsNamingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ProvenanceFile)
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := ReadProvenance(root)
	if err == nil {
		t.Fatal("ReadProvenance() = nil error; want an error naming the file")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("ReadProvenance() error = %q; want it to name %q", err, path)
	}
}

func TestWriteReadProvenance_RoundTrips(t *testing.T) {
	root := t.TempDir()
	want := &Provenance{
		LadderFile:         "ladders/ladder.yaml",
		LoomyardRepoSHA256: "deadbeef",
		ServerName:         "quarry",
		RepsEffective:      3,
		SelectedCells:      []string{"a0-none"},
	}
	if err := WriteProvenance(root, want); err != nil {
		t.Fatalf("WriteProvenance() = %v; want no error", err)
	}
	got, err := ReadProvenance(root)
	if err != nil {
		t.Fatalf("ReadProvenance() = %v; want no error", err)
	}
	if got.LadderFile != want.LadderFile || got.LoomyardRepoSHA256 != want.LoomyardRepoSHA256 ||
		got.ServerName != want.ServerName || got.RepsEffective != want.RepsEffective {
		t.Errorf("ReadProvenance() = %+v; want %+v", got, want)
	}
}

// canningRunner is a recordingRunner with canned stdout for the exact commands CollectInvocation
// issues.
func canningRunner() *recordingRunner {
	return &recordingRunner{outputs: map[string]string{
		"git rev-parse HEAD":     "quarrysha1234\n",
		"git status --porcelain": "",
		"claude --version":       "2.5.0 (Claude Code)\n",
	}}
}

func TestCollectInvocation_FillsFieldsFromRunner(t *testing.T) {
	quarryRoot := t.TempDir()
	targetRoot := filepath.Join(t.TempDir(), "target-repo")

	r := &recordingRunner{outputs: map[string]string{
		"git rev-parse HEAD":     "quarrysha1234\n",
		"git status --porcelain": " M dirty.txt\n",
		"claude --version":       "2.5.0 (Claude Code)\n",
	}}

	ladderFile := filepath.Join(quarryRoot, "bench", "ladder.yaml")

	in := CollectInput{
		QuarryRepoRoot: quarryRoot,
		LadderFilePath: ladderFile,
		TargetRepoPath: targetRoot,
		ServerName:     "quarry",
		SelectedCells:  []string{"a0-none"},
		RepsEffective:  3,
		ClaudeBinPath:  "claude",
	}

	inv, err := CollectInvocation(context.Background(), r, in)
	if err != nil {
		t.Fatalf("CollectInvocation() = %v; want no error", err)
	}

	if inv.QuarryCommit != "quarrysha1234" {
		t.Errorf("QuarryCommit = %q; want %q", inv.QuarryCommit, "quarrysha1234")
	}
	if !inv.QuarryDirty {
		t.Error("QuarryDirty = false; want true")
	}
	if len(inv.QuarryDirtyFiles) != 1 || inv.QuarryDirtyFiles[0] != "dirty.txt" {
		t.Errorf("QuarryDirtyFiles = %v; want [dirty.txt]", inv.QuarryDirtyFiles)
	}
	if inv.Hostname == "" {
		t.Error("Hostname is empty; want the host name")
	}
	if inv.GoVersion == "" {
		t.Error("GoVersion is empty; want the runtime Go version")
	}
	if inv.ClaudeVersion != "2.5.0 (Claude Code)" {
		t.Errorf("ClaudeVersion = %q; want %q", inv.ClaudeVersion, "2.5.0 (Claude Code)")
	}
	if inv.LoomyardRepoSHA256 == "" || strings.Contains(inv.LoomyardRepoSHA256, targetRoot) {
		t.Errorf("LoomyardRepoSHA256 = %q; want a hash, not the path %q", inv.LoomyardRepoSHA256, targetRoot)
	}
	wantLadderFile := filepath.ToSlash(filepath.Join("bench", "ladder.yaml"))
	if inv.LadderFile != wantLadderFile {
		t.Errorf("LadderFile = %q; want %q", inv.LadderFile, wantLadderFile)
	}
}

func TestCollectInvocation_LadderFileOutsideRepoUsesBaseName(t *testing.T) {
	quarryRoot := t.TempDir()
	outsideLadderFile := filepath.Join(t.TempDir(), "some-ladder.yaml")

	r := canningRunner()
	in := CollectInput{
		QuarryRepoRoot: quarryRoot,
		LadderFilePath: outsideLadderFile,
		TargetRepoPath: filepath.Join(t.TempDir(), "target"),
		ServerName:     "quarry",
		ClaudeBinPath:  "claude",
	}
	inv, err := CollectInvocation(context.Background(), r, in)
	if err != nil {
		t.Fatalf("CollectInvocation() = %v; want no error", err)
	}
	if inv.LadderFile != "some-ladder.yaml" {
		t.Errorf("LadderFile = %q; want base name %q", inv.LadderFile, "some-ladder.yaml")
	}
}

func TestMergeProvenance(t *testing.T) {
	first := Invocation{
		WrittenAt:          "t1",
		LadderFile:         "ladder.yaml",
		SelectedCells:      []string{"a0-none"},
		RepsEffective:      3,
		QuarryCommit:       "sha1",
		LoomyardCommit:     "loom1",
		LoomyardRepoSHA256: "hash1",
		ClaudeVersion:      "1.0",
		GoVersion:          "go1.26",
		Hostname:           "host1",
		ServerName:         "quarry",
		MemoryPathHashes:   []string{"mh1"},
		ServerHashes:       map[string]string{"a0-none/1": "sh1"},
	}

	merged, err := MergeProvenance(nil, first)
	if err != nil {
		t.Fatalf("MergeProvenance(nil, first) = %v; want no error", err)
	}
	if len(merged.Invocations) != 1 {
		t.Fatalf("MergeProvenance(nil, first).Invocations has %d entries; want 1", len(merged.Invocations))
	}

	second := Invocation{
		WrittenAt:          "t2",
		LadderFile:         "ladder.yaml",
		SelectedCells:      []string{"a3-full"},
		RepsEffective:      3,
		QuarryCommit:       "sha2",
		LoomyardCommit:     "loom2",
		LoomyardRepoSHA256: "hash1",
		ClaudeVersion:      "1.1",
		GoVersion:          "go1.26",
		Hostname:           "host2",
		ServerName:         "quarry",
		MemoryPathHashes:   []string{"mh2"},
		ServerHashes:       map[string]string{"a3-full/1": "sh2"},
	}

	merged, err = MergeProvenance(merged, second)
	if err != nil {
		t.Fatalf("MergeProvenance() = %v; want no error", err)
	}

	if len(merged.Invocations) != 2 {
		t.Fatalf("merged.Invocations has %d entries; want 2 (both invocations stay readable)", len(merged.Invocations))
	}

	wantCells := []string{"a0-none", "a3-full"}
	gotCells := append([]string(nil), merged.SelectedCells...)
	sort.Strings(gotCells)
	if !stringSlicesEqual(gotCells, wantCells) {
		t.Errorf("merged.SelectedCells = %v; want the union %v", gotCells, wantCells)
	}

	wantHashes := []string{"mh1", "mh2"}
	gotHashes := append([]string(nil), merged.MemoryPathHashes...)
	sort.Strings(gotHashes)
	if !stringSlicesEqual(gotHashes, wantHashes) {
		t.Errorf("merged.MemoryPathHashes = %v; want the union %v", gotHashes, wantHashes)
	}

	if merged.ServerHashes["a0-none/1"] != "sh1" || merged.ServerHashes["a3-full/1"] != "sh2" {
		t.Errorf("merged.ServerHashes = %v; want both keys merged", merged.ServerHashes)
	}

	if merged.QuarryCommit != "sha2" || merged.ClaudeVersion != "1.1" || merged.Hostname != "host2" {
		t.Errorf("merged top-level latest-wins fields = commit=%q claude=%q host=%q; want the second invocation's values",
			merged.QuarryCommit, merged.ClaudeVersion, merged.Hostname)
	}
	if merged.Invocations[0].QuarryCommit != "sha1" {
		t.Errorf("merged.Invocations[0].QuarryCommit = %q; want %q (first invocation's own value stays readable)", merged.Invocations[0].QuarryCommit, "sha1")
	}

	t.Run("DifferingRepsEffective_Refused", func(t *testing.T) {
		bad := second
		bad.RepsEffective = 5
		_, err := MergeProvenance(merged, bad)
		if err == nil {
			t.Fatal("MergeProvenance() with differing reps_effective = nil error; want an error naming both values")
		}
		if !strings.Contains(err.Error(), "3") || !strings.Contains(err.Error(), "5") {
			t.Errorf("error = %q; want it to name both reps_effective values", err)
		}
	})

	t.Run("DifferingLadderFile_Refused", func(t *testing.T) {
		bad := second
		bad.LadderFile = "other.yaml"
		_, err := MergeProvenance(merged, bad)
		if err == nil {
			t.Fatal("MergeProvenance() with differing ladder_file = nil error; want an error naming both values")
		}
		if !strings.Contains(err.Error(), "ladder.yaml") || !strings.Contains(err.Error(), "other.yaml") {
			t.Errorf("error = %q; want it to name both ladder_file values", err)
		}
	})

	t.Run("DifferingLoomyardRepoSHA256_Refused", func(t *testing.T) {
		bad := second
		bad.LoomyardRepoSHA256 = "hash2"
		_, err := MergeProvenance(merged, bad)
		if err == nil {
			t.Fatal("MergeProvenance() with differing loomyard_repo_sha256 = nil error; want an error naming both values")
		}
	})

	t.Run("DifferingServerName_Refused", func(t *testing.T) {
		bad := second
		bad.ServerName = "other-server"
		_, err := MergeProvenance(merged, bad)
		if err == nil {
			t.Fatal("MergeProvenance() with differing server_name = nil error; want an error naming both values")
		}
	})
}

func TestMergeProvenance_NoAbsolutePathAnywhereInOutput(t *testing.T) {
	tmp := t.TempDir()
	inv := Invocation{
		WrittenAt:          "t1",
		LadderFile:         "ladder.yaml",
		SelectedCells:      []string{"a0-none"},
		RepsEffective:      1,
		QuarryCommit:       "sha1",
		LoomyardCommit:     "loom1",
		LoomyardRepoSHA256: sha256Hex(filepath.Join(tmp, "loomyard")),
		ClaudeVersion:      "1.0",
		GoVersion:          "go1.26",
		Hostname:           "host1",
		ServerName:         "quarry",
		MemoryPathHashes:   []string{sha256Hex(filepath.Join(tmp, "memory"))},
		ServerHashes:       map[string]string{"a0-none/1": "sh1"},
	}
	p, err := MergeProvenance(nil, inv)
	if err != nil {
		t.Fatalf("MergeProvenance() = %v; want no error", err)
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.Contains(data, []byte(tmp)) {
		t.Errorf("marshalled provenance contains the temporary directory's own prefix %q:\n%s", tmp, data)
	}
}

func TestScanMemoryPaths(t *testing.T) {
	t.Run("CleanDirectory_NoFinding", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("nothing interesting"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		f, err := ScanMemoryPaths([]string{dir})
		if err != nil {
			t.Fatalf("ScanMemoryPaths() = %v; want no error", err)
		}
		if f != nil {
			t.Errorf("ScanMemoryPaths() = %+v; want nil", f)
		}
	})

	t.Run("TokenInFile_FatalFindingNamesFile", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "leaked.md")
		if err := os.WriteFile(path, []byte("this mentions quarry by name"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		f, err := ScanMemoryPaths([]string{dir})
		if err != nil {
			t.Fatalf("ScanMemoryPaths() = %v; want no error", err)
		}
		if f == nil || !f.Fatal {
			t.Fatalf("ScanMemoryPaths() = %+v; want a fatal finding", f)
		}
		if !strings.Contains(f.Message, path) {
			t.Errorf("finding message = %q; want it to name %q", f.Message, path)
		}
	})

	t.Run("NonExistentDirectory_FatalFinding_NotError", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "does-not-exist")
		f, err := ScanMemoryPaths([]string{missing})
		if err != nil {
			t.Fatalf("ScanMemoryPaths() = %v; want no error, a fatal finding instead", err)
		}
		if f == nil || !f.Fatal {
			t.Fatalf("ScanMemoryPaths() = %+v; want a fatal finding naming the missing directory", f)
		}
		if !strings.Contains(f.Message, missing) {
			t.Errorf("finding message = %q; want it to name %q", f.Message, missing)
		}
	})

	t.Run("EmptyPathList_NeitherFindingNorError", func(t *testing.T) {
		f, err := ScanMemoryPaths(nil)
		if err != nil {
			t.Fatalf("ScanMemoryPaths(nil) = %v; want no error", err)
		}
		if f != nil {
			t.Errorf("ScanMemoryPaths(nil) = %+v; want nil", f)
		}
	})
}

func TestWarnOnServerHashDrift(t *testing.T) {
	t.Run("NoDrift_Nil", func(t *testing.T) {
		p := &Provenance{ServerHashes: map[string]string{"a0-none/1": "sh1", "a0-none/2": "sh1"}}
		if f := WarnOnServerHashDrift(p); f != nil {
			t.Errorf("WarnOnServerHashDrift() = %+v; want nil", f)
		}
	})

	t.Run("Drift_NonFatalFinding", func(t *testing.T) {
		p := &Provenance{ServerHashes: map[string]string{"a0-none/1": "sh1", "a0-none/2": "sh2"}}
		f := WarnOnServerHashDrift(p)
		if f == nil {
			t.Fatal("WarnOnServerHashDrift() = nil; want a finding")
		}
		if f.Fatal {
			t.Error("WarnOnServerHashDrift() finding is fatal; want non-fatal")
		}
	})
}

func TestNewSessionFingerprint_PopulatesMCPServerStatuses(t *testing.T) {
	init := &SessionInit{
		ClaudeCodeVersion: "2.1.236",
		Model:             "claude-opus",
		MCPServers: []MCPServerStatus{
			{Name: "quarry", Status: "connected"},
			{Name: "other", Status: "failed"},
		},
	}
	fp := NewSessionFingerprint(init)

	wantNames := []string{"quarry", "other"}
	if !stringSlicesEqual(fp.MCPServers, wantNames) {
		t.Errorf("NewSessionFingerprint().MCPServers = %v; want %v in record order", fp.MCPServers, wantNames)
	}
	wantStatuses := map[string]string{"quarry": "connected", "other": "failed"}
	if !stringMapsEqual(fp.MCPServerStatuses, wantStatuses) {
		t.Errorf("NewSessionFingerprint().MCPServerStatuses = %v; want %v", fp.MCPServerStatuses, wantStatuses)
	}
}

func TestCompareFingerprints(t *testing.T) {
	base := SessionFingerprint{ClaudeCodeVersion: "1.0", Model: "m1", Tools: []string{"Bash"}}
	drifted := SessionFingerprint{ClaudeCodeVersion: "1.1", Model: "m1", Tools: []string{"Bash"}}

	p := &Provenance{SessionFingerprints: map[string]SessionFingerprint{
		"a0-none/1": base,
		"a0-none/2": drifted,
	}}

	findings := CompareFingerprints(p)
	if len(findings) != 1 {
		t.Fatalf("CompareFingerprints() = %d findings; want 1", len(findings))
	}
	if findings[0].Fatal {
		t.Error("CompareFingerprints() finding is fatal; want non-fatal")
	}
	if !strings.Contains(findings[0].Message, "claude_code_version") {
		t.Errorf("finding message = %q; want it to name the differing field", findings[0].Message)
	}
}

func TestDiffSessionFingerprint_ServerStatusOnlyChangeIsReported(t *testing.T) {
	a := SessionFingerprint{
		ClaudeCodeVersion: "2.1.236",
		Model:             "m1",
		Tools:             []string{"Bash"},
		MCPServers:        []string{"quarry"},
		MCPServerStatuses: map[string]string{"quarry": "connected"},
	}
	b := SessionFingerprint{
		ClaudeCodeVersion: "2.1.236",
		Model:             "m1",
		Tools:             []string{"Bash"},
		MCPServers:        []string{"quarry"},
		MCPServerStatuses: map[string]string{"quarry": "failed"},
	}

	diff := diffSessionFingerprint(a, b)
	if diff == "" {
		t.Fatal("diffSessionFingerprint() = \"\"; want a difference naming the server-status change")
	}
	if !strings.Contains(diff, "mcp_server_statuses") {
		t.Errorf("diff = %q; want it to name mcp_server_statuses", diff)
	}
}
