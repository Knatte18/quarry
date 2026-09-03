// main.go stands in for the measured claude binary the harness's run loop and scorer dispatch both
// invoke. It stands in for two different invocations -- the measured cell and the scorer -- and
// distinguishes them from the arguments alone, since both reach it through the same binary path
// within one process run of the harness and no per-invocation environment is available: an
// invocation carrying "--tools" "" together with "--max-turns" "1" is the scorer, anything else is a
// measured cell. It asserts a different flag set on each branch, failing with a non-zero exit and a
// message on standard error when an expected flag is absent or carries an unexpected value, then
// writes a canned stream-json transcript to standard output selected by an environment variable the
// driving test sets. It depends on nothing outside the standard library and is excluded from the
// module's own build because it lives under a "testdata" directory.
package main

import (
	"bufio"
	"fmt"
	"os"
)

// Environment variables the driving test sets to configure this fake's expectations and behaviour.
// The fake never spells the harness's own built-in tool names -- it is a separate package main and
// cannot reference the harness package's slice -- so the expected value arrives through
// fakeClaudeToolsEnv instead.
const (
	fakeClaudeModelEnv        = "FAKE_CLAUDE_MODEL"
	fakeClaudeEffortEnv       = "FAKE_CLAUDE_EFFORT"
	fakeClaudeMaxTurnsEnv     = "FAKE_CLAUDE_MAX_TURNS"
	fakeClaudeToolsEnv        = "FAKE_CLAUDE_TOOLS"
	fakeClaudeAllowedEnv      = "FAKE_CLAUDE_ALLOWED"
	fakeClaudeControlEnv      = "FAKE_CLAUDE_CONTROL"
	fakeClaudeScorerModelEnv  = "FAKE_CLAUDE_SCORER_MODEL"
	fakeClaudeScorerEffortEnv = "FAKE_CLAUDE_SCORER_EFFORT"
	fakeClaudeStreamEnv       = "FAKE_CLAUDE_STREAM"
	fakeClaudeLeakPrefixEnv   = "FAKE_CLAUDE_LEAK_PREFIX"
)

// flagsWithValue names every flag this fake consumes a following argument for.
var flagsWithValue = map[string]bool{
	"-p": true, "--model": true, "--effort": true, "--max-turns": true,
	"--tools": true, "--allowedTools": true, "--mcp-config": true,
	"--output-format": true, "--setting-sources": true,
}

// booleanFlags names every bare flag this fake records as present without a following argument.
var booleanFlags = map[string]bool{
	"--strict-mcp-config": true, "--verbose": true, "--no-session-persistence": true,
}

func main() {
	args := os.Args[1:]

	// CollectInvocation probes the binary's own version with a bare "--version" invocation, outside
	// either the measured-cell or the scorer argument vector; answer it directly rather than routing
	// it through the flag assertions below.
	if len(args) == 1 && args[0] == "--version" {
		fmt.Println("0.0.0-fake")
		return
	}

	values := map[string]string{}
	present := map[string]bool{}

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case flagsWithValue[a]:
			i++
			if i >= len(args) {
				fail(fmt.Sprintf("flag %s carries no following value", a))
			}
			values[a] = args[i]
		case booleanFlags[a]:
			present[a] = true
		}
	}

	tools, toolsOK := values["--tools"]
	maxTurns, maxTurnsOK := values["--max-turns"]
	isScorer := toolsOK && tools == "" && maxTurnsOK && maxTurns == "1"

	if isScorer {
		assertScorerInvocation(values, present)
		writeScorerReply()
		return
	}

	assertMeasuredCellInvocation(values, present)
	writeCellStream(os.Getenv(fakeClaudeStreamEnv))
}

// assertMeasuredCellInvocation checks every flag run.go's invokeMeasuredProcess sends for a measured
// cell, failing with a readable message on the first mismatch.
func assertMeasuredCellInvocation(values map[string]string, present map[string]bool) {
	requireEqual(values, "--model", os.Getenv(fakeClaudeModelEnv))
	requireEqual(values, "--effort", os.Getenv(fakeClaudeEffortEnv))
	requireEqual(values, "--max-turns", os.Getenv(fakeClaudeMaxTurnsEnv))
	requireEqual(values, "--tools", os.Getenv(fakeClaudeToolsEnv))
	requireNonEmpty(values, "--mcp-config")
	requirePresent(present, "--strict-mcp-config")
	requireEqual(values, "--output-format", "stream-json")
	requirePresent(present, "--verbose")
	requirePresent(present, "--no-session-persistence")
	requireEqualPresent(values, "--setting-sources", "")

	if os.Getenv(fakeClaudeControlEnv) == "1" {
		if _, ok := values["--allowedTools"]; ok {
			fail("measured cell invocation carries --allowedTools for a control cell, which must never grant it")
		}
		return
	}
	requireEqual(values, "--allowedTools", os.Getenv(fakeClaudeAllowedEnv))
}

// assertScorerInvocation checks every flag run.go's RunScorer sends, including the two differ-from-
// the-cell assertions that catch a model/effort mix-up between the two invocations rather than
// passing it quietly.
func assertScorerInvocation(values map[string]string, present map[string]bool) {
	wantModel := os.Getenv(fakeClaudeScorerModelEnv)
	requireEqual(values, "--model", wantModel)
	if wantModel == os.Getenv(fakeClaudeModelEnv) {
		fail("scorer invocation's expected model equals the cell's expected model -- the fixture cannot distinguish a mix-up")
	}

	wantEffort := os.Getenv(fakeClaudeScorerEffortEnv)
	requireEqual(values, "--effort", wantEffort)
	if wantEffort == os.Getenv(fakeClaudeEffortEnv) {
		fail("scorer invocation's expected effort equals the cell's expected effort -- the fixture cannot distinguish a mix-up")
	}

	requireEqual(values, "--tools", "")
	requireEqual(values, "--max-turns", "1")
	requireNonEmpty(values, "--mcp-config")
	requirePresent(present, "--strict-mcp-config")
	requireEqual(values, "--output-format", "stream-json")
	requirePresent(present, "--verbose")
	requirePresent(present, "--no-session-persistence")
	requireEqualPresent(values, "--setting-sources", "")

	if _, ok := values["--allowedTools"]; ok {
		fail("scorer invocation carries --allowedTools, which the scorer must never receive")
	}
}

func requireEqual(values map[string]string, flag, want string) {
	got, ok := values[flag]
	if !ok {
		fail(fmt.Sprintf("flag %s is absent; want %q", flag, want))
	}
	if got != want {
		fail(fmt.Sprintf("flag %s = %q; want %q", flag, got, want))
	}
}

// requireEqualPresent is requireEqual for a flag whose expected value may legitimately be the empty
// string, so the presence check and the value check are both explicit.
func requireEqualPresent(values map[string]string, flag, want string) {
	got, ok := values[flag]
	if !ok {
		fail(fmt.Sprintf("flag %s is absent; want it present with value %q", flag, want))
	}
	if got != want {
		fail(fmt.Sprintf("flag %s = %q; want %q", flag, got, want))
	}
}

func requireNonEmpty(values map[string]string, flag string) {
	got, ok := values[flag]
	if !ok || got == "" {
		fail(fmt.Sprintf("flag %s is absent or empty; want a non-empty value", flag))
	}
}

func requirePresent(present map[string]bool, flag string) {
	if !present[flag] {
		fail(fmt.Sprintf("flag %s is absent; want it present", flag))
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "fakeclaude:", msg)
	os.Exit(1)
}

// writeCellStream writes one of the canned measured-cell streams named by stream to standard output.
// An unrecognised or empty stream name is itself a fixture-configuration error, since a driving test
// that forgot to set FAKE_CLAUDE_STREAM should fail loudly rather than silently defaulting.
func writeCellStream(stream string) {
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	writeInit(w, os.Getenv(fakeClaudeToolsEnv))

	switch stream {
	case "normal":
		writeAssistant(w, "call-1", "let me look around")
		writeAssistant(w, "call-2", "Here is my answer.\n\n```json\n{\"relevant_files\":[\"a.go\"],\"key_symbols\":[],\"summary\":\"stub\",\"confidence\":\"high\",\"open_questions\":[]}\n```\n")
		writeResult(w, "completed", "end_turn", false)
	case "max_turns":
		writeAssistant(w, "call-1", "still working, no final answer yet")
		writeResult(w, "max_turns", "end_turn", false)
	case "no_fence":
		writeAssistant(w, "call-1", "I am done but I forgot the fenced block entirely.")
		writeResult(w, "completed", "end_turn", false)
	case "leak_prefix":
		leak := os.Getenv(fakeClaudeLeakPrefixEnv)
		writeAssistant(w, "call-1", fmt.Sprintf("I would have called %stoc here if I could.", leak))
		writeAssistant(w, "call-2", "Here is my answer.\n\n```json\n{\"relevant_files\":[],\"key_symbols\":[],\"summary\":\"stub\",\"confidence\":\"high\",\"open_questions\":[]}\n```\n")
		writeResult(w, "completed", "end_turn", false)
	case "partial_fail":
		writeAssistant(w, "call-1", "partial output before a simulated infrastructure failure")
		w.Flush()
		os.Exit(1)
	default:
		fail(fmt.Sprintf("unrecognised %s value %q", fakeClaudeStreamEnv, stream))
	}
}

// writeScorerReply writes the fixed canned scorer reply: a session-init record, one assistant record
// carrying a fenced json block satisfying the exploration schema's required fields, and a result
// record.
func writeScorerReply() {
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	writeInit(w, "")
	writeAssistant(w, "scorer-call-1", "```json\n{\"recall\": 1.0, \"precision\": 1.0, \"summary_matches\": true}\n```\n")
	writeResult(w, "completed", "end_turn", false)
}

func writeInit(w *bufio.Writer, tools string) {
	fmt.Fprintf(w,
		`{"type":"system","subtype":"init","uuid":"fake-init","timestamp":"2026-01-01T00:00:00Z","session_id":"fake-session","model":"fake-model","tools":%s,"mcp_servers":[],"permissionMode":"default","claude_code_version":"0.0.0-fake","memory_paths":{},"skills":[],"slash_commands":[]}`+"\n",
		toJSONStringArray(tools),
	)
}

func writeAssistant(w *bufio.Writer, id, text string) {
	fmt.Fprintf(w,
		`{"type":"assistant","uuid":"fake-%s","timestamp":"2026-01-01T00:00:01Z","message":{"id":%q,"model":"fake-model","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":0,"cache_creation_input_tokens":0},"content":[{"type":"text","text":%q}]}}`+"\n",
		id, id, text,
	)
}

func writeResult(w *bufio.Writer, terminalReason, stopReason string, isError bool) {
	fmt.Fprintf(w,
		`{"type":"result","subtype":"success","uuid":"fake-result","timestamp":"2026-01-01T00:00:02Z","num_turns":1,"duration_ms":100,"duration_api_ms":80,"total_cost_usd":0.001,"terminal_reason":%q,"stop_reason":%q,"is_error":%v,"permission_denials":[]}`+"\n",
		terminalReason, stopReason, isError,
	)
}

// toJSONStringArray turns a comma-separated tool list into a JSON string array literal, e.g.
// "Bash,Glob" becomes `["Bash","Glob"]`. An empty input becomes `[]`.
func toJSONStringArray(commaSeparated string) string {
	if commaSeparated == "" {
		return "[]"
	}
	out := "["
	start := 0
	first := true
	for i := 0; i <= len(commaSeparated); i++ {
		if i == len(commaSeparated) || commaSeparated[i] == ',' {
			if !first {
				out += ","
			}
			out += fmt.Sprintf("%q", commaSeparated[start:i])
			first = false
			start = i + 1
		}
	}
	return out + "]"
}
