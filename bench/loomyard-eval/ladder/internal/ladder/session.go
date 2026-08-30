// session.go materialises the three session types the matrix launches into their scratch directories,
// installs the tracked orchestration skill every session type shares, and prints the launch command the
// operator runs. This is the batch's replacement for scripts/run_ladder.py's write_run_inputs and
// build_argv/launch_run: the "which files live in which session's scratch directory" question below is
// the whole containment argument the blinding depends on, and the claude subprocess dispatch itself has
// no counterpart here (see the plan's Shared Decision that the harness never dispatches; the session
// does).

package ladder

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Session-input filenames, fixed for every session type this file materialises: the server declaration
// always lives at .mcp.json, the settings document always at .claude/settings.json, and a session's own
// agent definition under .claude/agents/. These are the literals Claude Code's own project-scope
// discovery depends on; nothing in this file constructs a different one.
const (
	serverDeclarationFilename = ".mcp.json"
	settingsRelativePath      = ".claude/settings.json"
	agentsRelativeDir         = ".claude/agents"
	scorerDefinitionFilename  = "scorer.md"
)

// installedSkillRelativePath is the fixed path, under a Claude Code skills root, InstallSkill writes the
// tracked orchestration skill to: ~/.claude/skills/ladder-run/SKILL.md when destRoot is ~/.claude/skills.
const installedSkillRelativePath = "ladder-run/SKILL.md"

// launchSettingSources is the fixed --setting-sources flag value every session launches with, so the
// scratch directory's settings document and agent definitions load as project scope while the installed
// skill loads as user scope. This flag combination ships unverified -- see LaunchCommand's doc comment
// for the documented fallback should project-scope agent discovery turn out to be suppressed by it.
const launchSettingSources = "user,project"

// SessionInputs records what one PrepareRunSession/PrepareScoringSession/PrepareProbeSession call wrote,
// so no later caller re-derives a filename: the scratch directory it wrote into, the name of the agent
// definition it wrote (matching that definition's own frontmatter name: field), and whether it wrote a
// server declaration at .mcp.json.
type SessionInputs struct {
	// ScratchDir is the session's scratch working directory.
	ScratchDir string
	// DefinitionName is the name of the agent definition written into ScratchDir's .claude/agents/ tree.
	DefinitionName string
	// HasServerDeclaration is true when ScratchDir carries (or, for a probe session, is understood by
	// LaunchCommand to carry once dispatched -- see PrepareProbeSession's doc comment) a server
	// declaration at .mcp.json.
	HasServerDeclaration bool
}

// writeJSONDocument marshals doc as indented JSON with a trailing newline to path, creating path's
// parent directory first.
func writeJSONDocument(path string, doc any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ladder: write json document: create %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("ladder: write json document: marshal for %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("ladder: write json document: write %s: %w", path, err)
	}
	return nil
}

// writeAgentDefinition writes body to <scratchDir>/.claude/agents/<filename>, creating the agents
// directory first.
func writeAgentDefinition(scratchDir, filename, body string) error {
	path := filepath.Join(scratchDir, agentsRelativeDir, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ladder: write agent definition: create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("ladder: write agent definition: write %s: %w", path, err)
	}
	return nil
}

// PrepareRunSession materialises one main-matrix run's scratch directory for config's n-th repetition:
// the settings document always, the run agent definition always under <config.ID>.md, and the server
// declaration at .mcp.json only when config.Allowed is non-empty.
//
// A config whose allowed set is empty gets no server declaration file and is launched with no server
// flag whatsoever, because a declared server named "quarry" exposing a prefixed namespace is itself the
// structural leak the blinding forbids -- ported verbatim from write_run_inputs's own rule. The scratch
// directory this writes into never receives the scorer definition, and never receives the installed
// skill (see InstallSkill, which never writes into a scratch directory for any session type).
func PrepareRunSession(l *Ladder, c LadderConfig, n int, serverPath, targetDir string) (SessionInputs, error) {
	scratchDir, err := SessionDir(l, c.ID, n)
	if err != nil {
		return SessionInputs{}, err
	}

	settingsPath := filepath.Join(scratchDir, settingsRelativePath)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return SessionInputs{}, fmt.Errorf("ladder: prepare run session: create %s: %w", filepath.Dir(settingsPath), err)
	}
	if err := WriteSettings(l, c, settingsPath); err != nil {
		return SessionInputs{}, err
	}

	runModel := l.RunModel
	if runModel == nil {
		return SessionInputs{}, fmt.Errorf("ladder: prepare run session: ladder.yaml: run_model is unset")
	}
	name, body, err := RunAgentDefinition(l, c, *runModel)
	if err != nil {
		return SessionInputs{}, err
	}
	if err := writeAgentDefinition(scratchDir, name+".md", body); err != nil {
		return SessionInputs{}, err
	}

	hasServer := len(c.Allowed) > 0
	if hasServer {
		path := filepath.Join(scratchDir, serverDeclarationFilename)
		if err := writeJSONDocument(path, MCPConfigDocument(serverPath, targetDir)); err != nil {
			return SessionInputs{}, err
		}
	}

	return SessionInputs{ScratchDir: scratchDir, DefinitionName: name, HasServerDeclaration: hasServer}, nil
}

// scoringSessionConfigID is the fixed configID SessionDir derives the scoring session's scratch
// directory from -- "scoring" with repetition 1, since exactly one scoring session is shared across the
// whole matrix.
const scoringSessionConfigID = "scoring"

// PrepareScoringSession materialises the single shared scoring session's scratch directory: the scorer
// agent definition at the fixed scorer.md filename, and a settings document denying nothing -- no server
// declaration, no run agent definition, no worktree setup, and no server build, since scoring never
// touches the target codebase at all. Task is deliberately not denied here -- see settings.go's
// SettingsDocumentFor doc comment for why a session-wide Task deny leaves the operator's own live
// session unable to dispatch the scorer agent at all.
func PrepareScoringSession(l *Ladder) (SessionInputs, error) {
	scratchDir, err := SessionDir(l, scoringSessionConfigID, 1)
	if err != nil {
		return SessionInputs{}, err
	}

	settingsDoc := SettingsDocument{Permissions: Permissions{
		Allow: []string{"Read", "Grep", "Glob", "Bash"},
		Deny:  []string{},
	}}
	if err := writeJSONDocument(filepath.Join(scratchDir, settingsRelativePath), settingsDoc); err != nil {
		return SessionInputs{}, err
	}

	name, body, err := ScorerAgentDefinition(l)
	if err != nil {
		return SessionInputs{}, err
	}
	if err := writeAgentDefinition(scratchDir, scorerDefinitionFilename, body); err != nil {
		return SessionInputs{}, err
	}

	return SessionInputs{ScratchDir: scratchDir, DefinitionName: name, HasServerDeclaration: false}, nil
}

// probeSessionConfigID returns the configID SessionDir derives a probe session's scratch directory from
// for kind -- "probe-allowlist" or "probe-denylist", matching plan.go's own documented convention for
// SessionDir's non-main-matrix callers.
func probeSessionConfigID(kind string) (string, error) {
	switch kind {
	case ProbeKindAllowlist:
		return "probe-allowlist", nil
	case ProbeKindDenylist:
		return "probe-denylist", nil
	default:
		return "", fmt.Errorf("ladder: prepare probe session: unknown probe kind %q", kind)
	}
}

// PrepareProbeSession materialises one permission probe's scratch directory: its bespoke agent
// definition (see ProbeAgentDefinition) and a settings document, into its own scratch directory keyed by
// probeSessionConfigID.
//
// The allowlist probe's settings do not deny the probed tool -- only its agent definition withholds it
// from its own tools: allowlist -- while the deny-list probe's settings do deny it, on top of a
// definition that grants it. The two layers are only independently established if each is probed with
// the other neutralised, which is exactly this pairing.
//
// This function's own signature carries neither a server path nor a target directory, so unlike
// PrepareRunSession it writes no .mcp.json content itself: HasServerDeclaration on the returned
// SessionInputs is true for both kinds as the signal that a caller holding those two values (the
// cli-session-commands batch's dispatch command, out of scope here) still owes this scratch directory a
// server declaration before launch. Generating the probe inputs is in scope here; dispatching them,
// including that remaining write, is not.
func PrepareProbeSession(l *Ladder, kind string) (SessionInputs, error) {
	configID, err := probeSessionConfigID(kind)
	if err != nil {
		return SessionInputs{}, err
	}
	scratchDir, err := SessionDir(l, configID, 1)
	if err != nil {
		return SessionInputs{}, err
	}

	name, body, err := ProbeAgentDefinition(l, kind)
	if err != nil {
		return SessionInputs{}, err
	}
	if err := writeAgentDefinition(scratchDir, name+".md", body); err != nil {
		return SessionInputs{}, err
	}

	// Task is deliberately not part of either probe's deny-list -- see settings.go's SettingsDocumentFor
	// doc comment for why a session-wide Task deny leaves the operator's own live session unable to
	// dispatch the probe agent at all.
	deniedName := MCPName(probeDeniedTool)
	var deny []string
	if kind == ProbeKindDenylist {
		deny = []string{deniedName}
	} else {
		deny = []string{}
	}
	settingsDoc := SettingsDocument{Permissions: Permissions{
		Allow: []string{"Read", "Grep", "Glob", "Bash"},
		Deny:  deny,
	}}
	if err := writeJSONDocument(filepath.Join(scratchDir, settingsRelativePath), settingsDoc); err != nil {
		return SessionInputs{}, err
	}

	return SessionInputs{ScratchDir: scratchDir, DefinitionName: name, HasServerDeclaration: true}, nil
}

// InstallSkill copies the tracked orchestration skill at sourcePath to
// <destRoot>/ladder-run/SKILL.md, overwriting any existing copy, and returns the installed path.
//
// It takes both paths as parameters, rather than hardcoding either, so this function is testable before
// the tracked skill file itself exists and so its destination is not baked in. It is called for every
// session type uniformly and must never write into a scratch directory: the skill body names quarry
// throughout, and a blinded session's scratch directory is that agent's own working directory -- writing
// the skill there would be exactly the leak the settings/agent-definition layers exist to prevent.
func InstallSkill(sourcePath, destRoot string) (string, error) {
	destPath := filepath.Join(destRoot, installedSkillRelativePath)

	source, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("ladder: install skill: open %s: %w", sourcePath, err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", fmt.Errorf("ladder: install skill: create %s: %w", filepath.Dir(destPath), err)
	}
	dest, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("ladder: install skill: create %s: %w", destPath, err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return "", fmt.Errorf("ladder: install skill: copy %s to %s: %w", sourcePath, destPath, err)
	}
	return destPath, nil
}

// LaunchCommand returns the exact command line the operator runs to launch the session inputs describes:
// the --setting-sources flag set fixed at launchSettingSources, plus a --mcp-config flag pointing at
// inputs.ScratchDir's server declaration only when inputs.HasServerDeclaration.
//
// This flag combination -- isolating settings while still loading project-local agent definitions and
// user-scope skills -- ships unverified; see the plan's Shared Decision on why no smoke launch is
// performed. Should project-scope agent discovery turn out to be suppressed by this flag combination,
// the documented fallback is to write the definitions into ~/.claude/agents/ under a "ladder-<config-id>"
// name instead, with prepare-session responsible for removing them again; the installed skill is never
// relocated into a scratch directory to work around anything.
func LaunchCommand(inputs SessionInputs) string {
	parts := []string{"claude", "--setting-sources", launchSettingSources}
	if inputs.HasServerDeclaration {
		parts = append(parts, "--mcp-config", filepath.Join(inputs.ScratchDir, serverDeclarationFilename))
	}
	return strings.Join(parts, " ")
}
