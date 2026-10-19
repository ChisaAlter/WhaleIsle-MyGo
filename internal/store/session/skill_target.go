package session

import (
	"fmt"
	"strings"

	"whaleisle/internal/dsh"
)

// These are the Host skill-inventory wire values, not presentation labels.
const (
	SkillRootUser    = "user-dsh"
	SkillRootProject = "project-dsh"
)

// SkillTarget captures the selected, already-authorized workspace and session.
// Resolving a target never chooses or registers a new filesystem directory.
type SkillTarget struct {
	client    *dsh.Client
	Root      string
	Cwd       string
	SessionID string
}

// MatchesClient binds a Settings result or action to its captured Host.
func (t SkillTarget) MatchesClient(client *dsh.Client) bool {
	return client != nil && t.client == client
}

// SameScope compares the Host and session/workspace owning an editor. Changing
// its user/project destination does not change the editor's captured scope.
func (t SkillTarget) SameScope(other SkillTarget) bool {
	return t.client != nil && t.client == other.client && t.Cwd == other.Cwd && t.SessionID == other.SessionID
}

func resolveSkillTarget(root, cwd, sessionID string) (SkillTarget, error) {
	root = strings.TrimSpace(root)
	switch root {
	case "", "user", SkillRootUser:
		root = SkillRootUser
	case "project", SkillRootProject:
		root = SkillRootProject
	default:
		return SkillTarget{}, fmt.Errorf("unsupported skill root %q", root)
	}
	cwd = strings.TrimSpace(cwd)
	if root == SkillRootProject && cwd == "" {
		return SkillTarget{}, fmt.Errorf("project skills require a selected workspace directory")
	}
	return SkillTarget{Root: root, Cwd: cwd, SessionID: sessionID}, nil
}

// SkillHubInstallMessage recognizes only the Host's two successful outcomes.
// Unknown responses must not be presented as a completed installation.
func SkillHubInstallMessage(status, name, directory string) (string, error) {
	var message string
	switch status {
	case "installed":
		message = "skill installed "
	case "already-installed":
		message = "skill already installed "
	default:
		return "", fmt.Errorf("skill installation was not confirmed (status %q)", status)
	}
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("skill installation returned no skill name")
	}
	message += name
	if directory != "" {
		message += " (" + directory + ")"
	}
	return message, nil
}
