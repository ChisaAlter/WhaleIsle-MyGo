package projectctl

import (
	"errors"
	"fmt"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/harness"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/projects"
)

// CapturePRViewOwner captures the active session/host/workspace identity for PR view authorization checks.
func (m *Manager) CapturePRViewOwner(root string) func() bool {
	sessions := m.sessionsStore()
	snap := m.hostSnapshot()
	host, client := snap.Status, snap.Client
	selected, cwd, epoch := "", "", uint64(0)
	if sessions != nil {
		v := sessions.SelectionSummary()
		selected, cwd, epoch = v.SelectedID, v.SelectedCwd, sessions.ComposerSessionGeneration()
	}
	currentRoot := func() string {
		cfg := m.configStore()
		value := ""
		if cfg != nil {
			value = cfg.Get().Workspace
		}
		if sessions != nil {
			if actual := sessions.SelectionSummary().SelectedCwd; actual != "" {
				value = actual
			}
		}
		return value
	}
	return func() bool {
		nowSnap := m.hostSnapshot()
		current := m.sessionsStore() == sessions &&
			nowSnap.Status.Generation == host.Generation &&
			nowSnap.Status.Endpoint == host.Endpoint &&
			nowSnap.Status.State == host.State &&
			!nowSnap.Quitting &&
			nowSnap.Client == client
		if !current || strings.TrimSpace(root) != strings.TrimSpace(currentRoot()) {
			return false
		}
		if sessions != nil {
			now := sessions.SelectionSummary()
			return now.SelectedID == selected && now.SelectedCwd == cwd && sessions.ComposerSessionGeneration() == epoch
		}
		return true
	}
}

// BindNativeConsumers wires settings navigation and authorized file consumers onto model and its observer children.
func (m *Manager) BindNativeConsumers(win any, model *projects.Model) {
	if model == nil {
		return
	}
	model.OnOpenModelSettings = func() {
		if m.deps.OpenModelSettings != nil {
			m.deps.OpenModelSettings()
		}
	}
	sessions := m.sessionsStore()
	snap := m.hostSnapshot()
	host, client := snap.Status, snap.Client
	if sessions == nil {
		return
	}
	selected := sessions.SelectedSessionID()
	navigation, err := sessions.CaptureComposerTarget(selected)
	accessCurrent := func() bool { return m.FileAccessCurrent(sessions, client, host) }
	admissionCurrent := func() bool { return err == nil && navigation.NavigationCurrent() && accessCurrent() }
	model.OnOpenAuthorizedFile = func(target projectflow.FileTarget) {
		if !admissionCurrent() {
			return
		}
		if _, relErr := m.TargetRelative(target); relErr != nil {
			sessions.SetLastAction(errcopy.Sentence(relErr))
			return
		}
		checkPath := func(path string) error {
			if !accessCurrent() {
				return errors.New(localizedText("项目文件授权已失效，请从报告重新打开。", "Project file access expired. Reopen it from the report."))
			}
			copy := target
			copy.CanonicalPath = path
			_, relErr := m.TargetRelative(copy)
			return relErr
		}
		binding := fmt.Sprintf("project:%p:%d:%s:%s", client, host.Generation, target.ProjectID, target.CoordinatorSessionID)
		local := false
		if m.deps.ManagedMarketProfile != nil {
			local = m.deps.ManagedMarketProfile(host) != ""
		}
		if m.deps.OpenAuthorizedFile == nil {
			sessions.SetLastAction(localizedText("项目文件打开能力未配置。", "Project file opener is unavailable."))
			return
		}
		err := m.deps.OpenAuthorizedFile(win, AuthorizedFileRequest{
			Target:         target,
			Binding:        binding,
			Client:         client,
			HostGeneration: host.Generation,
			ThemeID:        model.ThemeID,
			SystemDark:     model.SystemDark,
			Local:          local,
			AccessCurrent:  accessCurrent,
			CheckPath:      checkPath,
		})
		if err != nil {
			sessions.SetLastAction(errcopy.Sentence(err))
			return
		}
		m.invalidateWindow(win)
	}
	if original := model.ObserverModel; original != nil {
		model.ObserverModel = func(o *projects.ObserverState) projects.Model {
			child := original(o)
			m.BindNativeConsumers(win, &child)
			return child
		}
	}
}

// FileAccessCurrent reports whether a retained Project file's owning Host, client, and session store remain valid.
func (m *Manager) FileAccessCurrent(sessions *session.Store, client *dsh.Client, host harness.Status) bool {
	snap := m.hostSnapshot()
	return m.sessionsStore() == sessions &&
		snap.Status.Generation == host.Generation &&
		snap.Status.Endpoint == host.Endpoint &&
		snap.Status.State == harness.Ready &&
		!snap.Quitting &&
		snap.Client == client
}

// TargetRelative validates and returns the relative path of target inside its authorized Project root.
func (m *Manager) TargetRelative(t projectflow.FileTarget) (string, error) {
	return TargetRelative(t, m.deps.ValidateRelativeFile)
}

// TargetRelative validates t using validate(root, canonicalPath) and fails closed if validate is nil.
func TargetRelative(t projectflow.FileTarget, validate func(root, canonicalPath string) (string, error)) (string, error) {
	fail := errors.New(localizedText("项目文件路径不属于报告授权的工作目录。", "The Project file is outside the report's authorized root."))
	if t.ProjectID == "" || t.CoordinatorSessionID == "" || strings.ContainsRune(t.Root, 0) || strings.ContainsRune(t.CanonicalPath, 0) {
		return "", fail
	}
	if validate == nil {
		return "", fail
	}
	rel, err := validate(t.Root, t.CanonicalPath)
	if err != nil {
		return "", err
	}
	if rel == "" || rel == "." {
		return "", fail
	}
	return rel, nil
}

func absoluteProjectPath(root, value string) string {
	value = strings.ReplaceAll(value, `\`, "/")
	if strings.HasPrefix(value, "/") || len(value) > 1 && value[1] == ':' {
		return value
	}
	return strings.TrimRight(strings.ReplaceAll(root, `\`, "/"), "/") + "/" + value
}
