// tocconfig_test.go exercises resolveTOCConfigPath's precedence, loadTOCConfig's file handling,
// parseDocSentences's value grammar, and resolveDocSentences's end-to-end chain, all over files
// written into a t.TempDir().
//
// None of the subtests here are marked t.Parallel(): several use t.Setenv to redirect
// $QUARRY_TOC_CONFIG, and t.Setenv is incompatible with parallel subtests.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

func TestResolveTOCConfigPath_Precedence(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     func(targetDir string) string
	}{
		{
			name:     "env set",
			envValue: "/env/toc.yaml",
			want:     func(string) string { return "/env/toc.yaml" },
		},
		{
			name:     "default tier",
			envValue: "",
			want:     func(targetDir string) string { return filepath.Join(targetDir, ".quarry.yaml") },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("QUARRY_TOC_CONFIG", tt.envValue)
			targetDir := t.TempDir()
			got := resolveTOCConfigPath(targetDir)
			want := tt.want(targetDir)
			if got != want {
				t.Errorf("resolveTOCConfigPath(%q) = %q; want %q", targetDir, got, want)
			}
		})
	}
}

func TestLoadTOCConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	raw, err := loadTOCConfig(filepath.Join(dir, "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("loadTOCConfig(missing) error = %v; want nil", err)
	}
	if raw != nil {
		t.Errorf("loadTOCConfig(missing) = %v; want nil", raw)
	}
}

func TestLoadTOCConfig_EmptyAndCommentsOnly(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty file", ""},
		{"comments-only file", "# nothing here\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".quarry.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("os.WriteFile(%q) failed: %v", path, err)
			}
			raw, err := loadTOCConfig(path)
			if err != nil {
				t.Fatalf("loadTOCConfig(%q) error = %v; want nil", tt.name, err)
			}
			if raw != nil {
				t.Errorf("loadTOCConfig(%q) = %v; want nil", tt.name, raw)
			}
		})
	}
}

func TestLoadTOCConfig_ValuePresent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"integer value", "toc:\n  doc_sentences: 3\n", "3"},
		{"all value", "toc:\n  doc_sentences: all\n", "all"},
		{"zero value", "toc:\n  doc_sentences: 0\n", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".quarry.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("os.WriteFile(%q) failed: %v", path, err)
			}
			raw, err := loadTOCConfig(path)
			if err != nil {
				t.Fatalf("loadTOCConfig(%q) error = %v; want nil", tt.name, err)
			}
			if raw == nil {
				t.Fatalf("loadTOCConfig(%q) = nil; want %q", tt.name, tt.want)
			}
			if *raw != tt.want {
				t.Errorf("loadTOCConfig(%q) = %q; want %q", tt.name, *raw, tt.want)
			}
		})
	}
}

func TestLoadTOCConfig_UnknownKey(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"unknown top-level key", "toc:\n  doc_sentences: 1\nbogus: true\n"},
		{"unknown key inside toc section", "toc:\n  doc_sentences: 1\n  bogus: true\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".quarry.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("os.WriteFile(%q) failed: %v", path, err)
			}
			_, err := loadTOCConfig(path)
			if err == nil {
				t.Fatalf("loadTOCConfig(%q) error = nil; want a loud error naming the unknown key", tt.name)
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("loadTOCConfig(%q) error = %q; want it to name the file path %q", tt.name, err.Error(), path)
			}
		})
	}
}

func TestLoadTOCConfig_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".quarry.yaml")
	if err := os.WriteFile(path, []byte("toc: [this is not a mapping\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) failed: %v", path, err)
	}
	_, err := loadTOCConfig(path)
	if err == nil {
		t.Fatal("loadTOCConfig(malformed) error = nil; want a decode error")
	}
}

func TestParseDocSentences(t *testing.T) {
	tests := []struct {
		raw     string
		want    int
		wantErr bool
	}{
		{"all", quarry.TOCAllSentences, false},
		{"0", 0, false},
		{"1", 1, false},
		{"7", 7, false},
		{"-1", 0, true},
		{"All", 0, true},
		{"sentence", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := parseDocSentences(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseDocSentences(%q) error = nil; want an error", tt.raw)
				}
				if !strings.Contains(err.Error(), "all") || !strings.Contains(err.Error(), "non-negative") {
					t.Errorf("parseDocSentences(%q) error = %q; want it to name both valid forms", tt.raw, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDocSentences(%q) error = %v; want nil", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("parseDocSentences(%q) = %d; want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestResolveDocSentences_Precedence(t *testing.T) {
	t.Run("no file, no env, no flag: default", func(t *testing.T) {
		t.Setenv("QUARRY_TOC_CONFIG", "")
		targetDir := t.TempDir()
		got, err := resolveDocSentences("", targetDir)
		if err != nil {
			t.Fatalf("resolveDocSentences(\"\", %q) error = %v; want nil", targetDir, err)
		}
		if got != 1 {
			t.Errorf("resolveDocSentences(\"\", %q) = %d; want 1", targetDir, got)
		}
	})

	t.Run("quarry.yaml in target directory: 3", func(t *testing.T) {
		t.Setenv("QUARRY_TOC_CONFIG", "")
		targetDir := t.TempDir()
		writeQuarryYAML(t, targetDir, "toc:\n  doc_sentences: 3\n")
		got, err := resolveDocSentences("", targetDir)
		if err != nil {
			t.Fatalf("resolveDocSentences error = %v; want nil", err)
		}
		if got != 3 {
			t.Errorf("resolveDocSentences = %d; want 3", got)
		}
	})

	t.Run("quarry.yaml doc_sentences all", func(t *testing.T) {
		t.Setenv("QUARRY_TOC_CONFIG", "")
		targetDir := t.TempDir()
		writeQuarryYAML(t, targetDir, "toc:\n  doc_sentences: all\n")
		got, err := resolveDocSentences("", targetDir)
		if err != nil {
			t.Fatalf("resolveDocSentences error = %v; want nil", err)
		}
		if got != quarry.TOCAllSentences {
			t.Errorf("resolveDocSentences = %d; want %d (TOCAllSentences)", got, quarry.TOCAllSentences)
		}
	})

	t.Run("quarry.yaml doc_sentences 0 distinguishable from nothing", func(t *testing.T) {
		t.Setenv("QUARRY_TOC_CONFIG", "")
		targetDir := t.TempDir()
		writeQuarryYAML(t, targetDir, "toc:\n  doc_sentences: 0\n")
		got, err := resolveDocSentences("", targetDir)
		if err != nil {
			t.Fatalf("resolveDocSentences error = %v; want nil", err)
		}
		if got != 0 {
			t.Errorf("resolveDocSentences = %d; want 0", got)
		}
	})

	t.Run("QUARRY_TOC_CONFIG wins over quarry.yaml in target directory", func(t *testing.T) {
		targetDir := t.TempDir()
		writeQuarryYAML(t, targetDir, "toc:\n  doc_sentences: 3\n")

		envDir := t.TempDir()
		envPath := filepath.Join(envDir, "other.yaml")
		if err := os.WriteFile(envPath, []byte("toc:\n  doc_sentences: 5\n"), 0o644); err != nil {
			t.Fatalf("os.WriteFile(%q) failed: %v", envPath, err)
		}
		t.Setenv("QUARRY_TOC_CONFIG", envPath)

		got, err := resolveDocSentences("", targetDir)
		if err != nil {
			t.Fatalf("resolveDocSentences error = %v; want nil", err)
		}
		if got != 5 {
			t.Errorf("resolveDocSentences = %d; want 5 (the $QUARRY_TOC_CONFIG file, not the target directory's own)", got)
		}
	})

	t.Run("flag wins over env and file", func(t *testing.T) {
		targetDir := t.TempDir()
		writeQuarryYAML(t, targetDir, "toc:\n  doc_sentences: 3\n")

		envDir := t.TempDir()
		envPath := filepath.Join(envDir, "other.yaml")
		if err := os.WriteFile(envPath, []byte("toc:\n  doc_sentences: 5\n"), 0o644); err != nil {
			t.Fatalf("os.WriteFile(%q) failed: %v", envPath, err)
		}
		t.Setenv("QUARRY_TOC_CONFIG", envPath)

		got, err := resolveDocSentences("0", targetDir)
		if err != nil {
			t.Fatalf("resolveDocSentences error = %v; want nil", err)
		}
		if got != 0 {
			t.Errorf("resolveDocSentences(\"0\", ...) = %d; want 0 (the flag)", got)
		}
	})

	// This is the single most important case in this file: without it, a later
	// "helpful" change that walks upward would pass every other precedence case
	// here while silently introducing the repository-root concept toc
	// deliberately does not have.
	t.Run("quarry.yaml in the parent directory is ignored", func(t *testing.T) {
		t.Setenv("QUARRY_TOC_CONFIG", "")
		parent := t.TempDir()
		writeQuarryYAML(t, parent, "toc:\n  doc_sentences: 9\n")
		targetDir := filepath.Join(parent, "child")
		if err := os.Mkdir(targetDir, 0o755); err != nil {
			t.Fatalf("os.Mkdir(%q) failed: %v", targetDir, err)
		}

		got, err := resolveDocSentences("", targetDir)
		if err != nil {
			t.Fatalf("resolveDocSentences error = %v; want nil", err)
		}
		if got != 1 {
			t.Errorf("resolveDocSentences = %d; want the default 1 — the parent directory's .quarry.yaml must be ignored", got)
		}
	})
}

// writeQuarryYAML writes content to ".quarry.yaml" inside dir.
func writeQuarryYAML(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, ".quarry.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) failed: %v", path, err)
	}
}
