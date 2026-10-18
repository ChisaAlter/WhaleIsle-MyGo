package session

import (
	"strings"
	"testing"
)

func TestSkillTargetWireRoots(t *testing.T) {
	for _, tt := range []struct {
		input, cwd, want string
		invalid          bool
	}{
		{"", "", SkillRootUser, false},
		{"user", "/authorized/project", SkillRootUser, false},
		{" user-dsh ", "", SkillRootUser, false},
		{"project", "/authorized/project", SkillRootProject, false},
		{"project-dsh", " /authorized/project ", SkillRootProject, false},
		{"project-dsh", "", "", true},
		{"project", " \t", "", true},
		{"unknown", "/authorized/project", "", true},
		{"user-agents", "/authorized/project", "", true},
	} {
		t.Run(tt.input+tt.cwd, func(t *testing.T) {
			target, err := resolveSkillTarget(tt.input, tt.cwd, "session-1")
			if (err != nil) != tt.invalid {
				t.Fatalf("target=%+v err=%v", target, err)
			}
			if err != nil {
				return
			}
			if target.Root != tt.want || target.Cwd != strings.TrimSpace(tt.cwd) || target.SessionID != "session-1" {
				t.Fatalf("target=%+v", target)
			}
		})
	}
}

func TestSkillHubInstallActualStatus(t *testing.T) {
	for _, tt := range []struct {
		status, want string
		invalid      bool
	}{
		{"installed", "skill installed cool", false},
		{"already-installed", "skill already installed cool", false},
		{"ok", "", true},
		{"conflict", "", true},
		{"", "", true},
	} {
		t.Run(tt.status, func(t *testing.T) {
			message, err := SkillHubInstallMessage(tt.status, "cool", "/authorized/skills/cool")
			if (err != nil) != tt.invalid {
				t.Fatalf("message=%q err=%v", message, err)
			}
			if err == nil && message != tt.want+" (/authorized/skills/cool)" {
				t.Fatalf("message=%q", message)
			}
		})
	}
	if _, err := SkillHubInstallMessage("installed", "", ""); err == nil {
		t.Fatal("missing identity was treated as success")
	}
}
