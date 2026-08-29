package mcpserver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Knatte18/quarry/internal/cli"
	"github.com/Knatte18/quarry/quarry"
)

func TestEffectiveTargetDir_OverrideAbsolutised(t *testing.T) {
	cfg := Config{TargetDir: "/launch/default"}

	got, err := effectiveTargetDir(cfg, "relative/override")
	if err != nil {
		t.Fatalf("effectiveTargetDir(%v, %q) error = %v", cfg, "relative/override", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("effectiveTargetDir(%v, %q) = %q; want an absolute path", cfg, "relative/override", got)
	}
	want, err := filepath.Abs("relative/override")
	if err != nil {
		t.Fatalf("filepath.Abs error = %v", err)
	}
	if got != want {
		t.Errorf("effectiveTargetDir(%v, %q) = %q; want %q", cfg, "relative/override", got, want)
	}
}

func TestEffectiveTargetDir_EmptyOverrideUsesLaunchDefault(t *testing.T) {
	cfg := Config{TargetDir: "/launch/default"}

	got, err := effectiveTargetDir(cfg, "")
	if err != nil {
		t.Fatalf("effectiveTargetDir(%v, \"\") error = %v", cfg, err)
	}
	if got != cfg.TargetDir {
		t.Errorf("effectiveTargetDir(%v, \"\") = %q; want %q (the launch default)", cfg, got, cfg.TargetDir)
	}
}

func TestResolveCall_StateDirMatchesCLIResolution(t *testing.T) {
	stateDir := t.TempDir()
	cfg := Config{TargetDir: t.TempDir(), StateDir: stateDir, Timeout: 5 * time.Second}

	got, err := resolveCall(cfg, "", "")
	if err != nil {
		t.Fatalf("resolveCall(%v, \"\", \"\") error = %v", cfg, err)
	}

	want, err := cli.ResolveStateDir(cfg.StateDir, cfg.TargetDir, cli.ResolveBuildTags(""))
	if err != nil {
		t.Fatalf("cli.ResolveStateDir error = %v", err)
	}
	if got.StateDir != want {
		t.Errorf("resolveCall(...).StateDir = %q; want %q (cli.ResolveStateDir's own result for the identical inputs)", got.StateDir, want)
	}
}

func TestResolveCall_BuildTagsSegment(t *testing.T) {
	stateDir := t.TempDir()
	cfg := Config{TargetDir: t.TempDir(), StateDir: stateDir}

	withoutTags, err := resolveCall(cfg, "", "")
	if err != nil {
		t.Fatalf("resolveCall(%v, \"\", \"\") error = %v", cfg, err)
	}
	withTags, err := resolveCall(cfg, "", "sometag")
	if err != nil {
		t.Fatalf("resolveCall(%v, \"\", \"sometag\") error = %v", cfg, err)
	}

	if withoutTags.StateDir == withTags.StateDir {
		t.Errorf("resolveCall with a non-empty build-tag set produced the same StateDir as with none: %q", withTags.StateDir)
	}
	wantTagged, err := cli.ResolveStateDir(cfg.StateDir, cfg.TargetDir, cli.ResolveBuildTags("sometag"))
	if err != nil {
		t.Fatalf("cli.ResolveStateDir error = %v", err)
	}
	if withTags.StateDir != wantTagged {
		t.Errorf("resolveCall(..., \"sometag\").StateDir = %q; want %q", withTags.StateDir, wantTagged)
	}
	if len(withTags.BuildTags) == 0 {
		t.Error("resolveCall(..., \"sometag\").BuildTags is empty; want a non-empty normalized set")
	}
}

func TestCallContext_Options_SkipVerificationFalse(t *testing.T) {
	c := callContext{TargetDir: "/target", StateDir: "/state", Timeout: 5 * time.Second}
	opts := c.options("go", quarry.Query{Symbol: "Foo"})
	if opts.SkipVerification {
		t.Error("callContext.options(...).SkipVerification = true; want false (the default is \"verify\")")
	}
}

// TestCallContext_Options_TimeoutUndividedPerEntry asserts callContext.options carries cfg.Timeout
// through to quarry.Options.Timeout in full for every entry of a batch, rather than dividing it —
// batching-execution-model's per-entry-timeout rule.
func TestCallContext_Options_TimeoutUndividedPerEntry(t *testing.T) {
	const timeout = 17 * time.Second
	c := callContext{TargetDir: "/target", StateDir: "/state", Timeout: timeout}

	entries := []string{"Foo", "Bar", "Baz"}
	for _, e := range entries {
		opts := c.options("go", quarry.Query{Symbol: e})
		if opts.Timeout != timeout {
			t.Errorf("callContext.options(%q).Timeout = %v; want the full, undivided %v", e, opts.Timeout, timeout)
		}
	}
}

func TestCallContext_Options_FieldsPopulated(t *testing.T) {
	reg := quarry.BuiltinRegistry()
	c := callContext{Registry: reg, TargetDir: "/target", StateDir: "/state", BuildTags: []string{"a", "b"}, Timeout: 5 * time.Second}
	query := quarry.Query{Symbol: "Foo"}

	opts := c.options("go", query)
	if opts.TargetDir != c.TargetDir {
		t.Errorf("opts.TargetDir = %q; want %q", opts.TargetDir, c.TargetDir)
	}
	if opts.StateDir != c.StateDir {
		t.Errorf("opts.StateDir = %q; want %q", opts.StateDir, c.StateDir)
	}
	if opts.Lang != "go" {
		t.Errorf("opts.Lang = %q; want \"go\"", opts.Lang)
	}
	if opts.Query != query {
		t.Errorf("opts.Query = %v; want %v", opts.Query, query)
	}
	if len(opts.BuildTags) != len(c.BuildTags) {
		t.Errorf("opts.BuildTags = %v; want %v", opts.BuildTags, c.BuildTags)
	}
}
