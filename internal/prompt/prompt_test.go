package prompt

import (
	"testing"
)

func TestParseAgents_Deduplication(t *testing.T) {
	got := parseAgents("claude,claude,cursor,claude")
	want := []string{"claude", "cursor"}

	if len(got) != len(want) {
		t.Fatalf("parseAgents() returned %d agents, want %d: %v", len(got), len(want), got)
	}

	for i, agent := range got {
		if agent != want[i] {
			t.Errorf("parseAgents()[%d] = %q, want %q", i, agent, want[i])
		}
	}
}

func TestParseAgents_TrimsAndLowercases(t *testing.T) {
	got := parseAgents(" Claude , CURSOR , gemini ")
	want := []string{"claude", "cursor", "gemini"}

	if len(got) != len(want) {
		t.Fatalf("parseAgents() returned %d agents, want %d: %v", len(got), len(want), got)
	}

	for i, agent := range got {
		if agent != want[i] {
			t.Errorf("parseAgents()[%d] = %q, want %q", i, agent, want[i])
		}
	}
}

func TestParseAgents_SkipsEmpty(t *testing.T) {
	got := parseAgents("claude,,cursor,")
	want := []string{"claude", "cursor"}

	if len(got) != len(want) {
		t.Fatalf("parseAgents() returned %d agents, want %d: %v", len(got), len(want), got)
	}

	for i, agent := range got {
		if agent != want[i] {
			t.Errorf("parseAgents()[%d] = %q, want %q", i, agent, want[i])
		}
	}
}

func TestParseAgents_Single(t *testing.T) {
	got := parseAgents("claude")
	if len(got) != 1 || got[0] != "claude" {
		t.Errorf("parseAgents(\"claude\") = %v, want [\"claude\"]", got)
	}
}
