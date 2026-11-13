package main

import "testing"

func TestSettingsModelsSessionActionFiltersChatNoise(t *testing.T) {
	for _, noise := range []string{
		"",
		"session created",
		"prompt accepted",
		"create workspace session: session/conflict: details",
		"llm: 2 configurable, 1 listed, 4 models",
		"offline",
	} {
		if got := settingsModelsSessionAction(noise, "offline"); got != "" {
			t.Fatalf("settingsModelsSessionAction(%q) = %q, want empty", noise, got)
		}
	}
	for _, keep := range []string{"current session model updated", "当前会话模型已更新"} {
		if got := settingsModelsSessionAction(keep, ""); got != keep {
			t.Fatalf("settingsModelsSessionAction(%q) = %q, want %q", keep, got, keep)
		}
	}
}

func TestSettingsSkillsSessionActionFiltersChatNoiseAndJoinsCleanly(t *testing.T) {
	for _, noise := range []string{
		"",
		"session created",
		"create workspace session: err",
		"llm: 1 configurable",
	} {
		if got := settingsSkillsSessionAction(noise); got != "" {
			t.Fatalf("settingsSkillsSessionAction(%q) = %q, want empty", noise, got)
		}
	}
	for _, keep := range []string{
		"skill updated: demo",
		"skill created demo",
		"skills/update: rpc failed",
		`skill "demo" conflicts with an existing skill`,
	} {
		if got := settingsSkillsSessionAction(keep); got != keep {
			t.Fatalf("settingsSkillsSessionAction(%q) = %q, want %q", keep, got, keep)
		}
	}
	if got := joinNonEmptyLines("", "  Skill saved: demo  ", "", "Skill invocation: 1 updated, 0 failed"); got != "Skill saved: demo\nSkill invocation: 1 updated, 0 failed" {
		t.Fatalf("joinNonEmptyLines = %q", got)
	}
}
