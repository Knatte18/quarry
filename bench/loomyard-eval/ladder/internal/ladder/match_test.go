package ladder

import "testing"

func TestMatchesBareToken(t *testing.T) {
	const token = "toc"

	tests := []struct {
		name string
		text string
		want bool
	}{
		{"UnrelatedWord_Protocol", "protocol", false},
		{"UnrelatedWord_Stochastic", "stochastic", false},
		{"UnrelatedWord_October", "October", false},
		{"ExactMatch", "toc", true},
		{"UppercaseMatch", "TOC", true},
		{"WordInSentence", "the toc tool", true},
		{"BacktickWrapped", "`toc`", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchesBareToken(tt.text, token); got != tt.want {
				t.Errorf("MatchesBareToken(%q, %q) = %v; want %v", tt.text, token, got, tt.want)
			}
		})
	}
}

func TestMatchesBareToken_MetacharacterTokenIsLiteral(t *testing.T) {
	const token = "a.b*c"

	if MatchesBareToken("aXbYYYc, matched as a regex", token) {
		t.Errorf("MatchesBareToken interpreted %q as a regular expression instead of matching it literally", token)
	}
	if !MatchesBareToken("call a.b*c now", token) {
		t.Errorf("MatchesBareToken(%q, %q) = false; want true for the literal token", "call a.b*c now", token)
	}
}

func TestMatchesComposedString(t *testing.T) {
	const prefix = "mcp__quarry__"
	const repoPath = "/home/user/work/quarry"

	tests := []struct {
		name string
		text string
		s    string
		want bool
	}{
		{"MCPPrefixSubstring", "the tool mcp__quarry__toc was called", prefix, true},
		{"RepoRootPathSubstring", "reading from /home/user/work/quarry/README.md", repoPath, true},
		{"DifferentlyCasedCopyDoesNotMatch", "MCP__QUARRY__TOC", prefix, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchesComposedString(tt.text, tt.s); got != tt.want {
				t.Errorf("MatchesComposedString(%q, %q) = %v; want %v", tt.text, tt.s, got, tt.want)
			}
		})
	}
}

func TestBareTokenAlternation_EmptySlice(t *testing.T) {
	if got := BareTokenAlternation(nil); got != nil {
		t.Errorf("BareTokenAlternation(nil) = %v; want nil", got)
	}
	if got := BareTokenAlternation([]string{}); got != nil {
		t.Errorf("BareTokenAlternation([]string{}) = %v; want nil", got)
	}
	if got := BareTokenAlternation([]string{"", ""}); got != nil {
		t.Errorf(`BareTokenAlternation([]string{"", ""}) = %v; want nil`, got)
	}
}

func TestBareTokenAlternation_Matches(t *testing.T) {
	re := BareTokenAlternation([]string{"toc", "impact"})
	if re == nil {
		t.Fatalf("BareTokenAlternation([toc, impact]) = nil; want a compiled pattern")
	}
	if !re.MatchString("the toc tool ran") {
		t.Errorf("alternation did not match %q", "the toc tool ran")
	}
	if !re.MatchString("IMPACT analysis") {
		t.Errorf("alternation did not match %q", "IMPACT analysis")
	}
	if re.MatchString("protocol") {
		t.Errorf("alternation unexpectedly matched %q", "protocol")
	}
}
