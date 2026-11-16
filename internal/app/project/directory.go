package projectctl

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/workspacepicker"
)

type directoryIntent struct {
	client *dsh.Client
	path   string
}

type directoryCheck struct {
	reason string
}

func directoryKey(path string) string { return strings.ToLower(filepath.Clean(path)) }

// WorkspaceDirectoryActions builds the directory action entries for registered workspace rows.
func WorkspaceDirectoryActions(available bool, state func(path string) (active, refused bool), run func(string)) []workspacepicker.DirectoryAction {
	if !available {
		return nil
	}
	picker := false
	return []workspacepicker.DirectoryAction{{
		ID:     "project.create",
		Order:  20,
		Picker: &picker,
		Run:    run,
		Visible: func(path string) bool {
			_, refused := state(path)
			return !refused
		},
		LabelForPath: func(path string) string {
			active, _ := state(path)
			if i18n.Locale() == i18n.LocaleZH {
				if active {
					return "打开 Project"
				}
				return "在此目录新建 Project"
			}
			if active {
				return "Open Project"
			}
			return "Create Project in this directory"
		},
	}}
}

// DirectoryActions builds workspace picker directory actions from cached Project listings and directory checks.
func (m *Manager) DirectoryActions(win any) []workspacepicker.DirectoryAction {
	snap := m.hostSnapshot()
	client := snap.Client
	cfgStore := m.configStore()
	sessions := m.sessionsStore()

	m.mu.Lock()
	f := &m.primary.flow
	available := client != nil && f.client == client && f.listing.Available
	active := map[string]bool{}
	for _, project := range f.listing.Projects {
		if project.Lifecycle != "archived" {
			active[directoryKey(project.CanonicalWorkingDirectory)] = true
		}
	}
	m.mu.Unlock()
	if !available || cfgStore != nil && !MountEnabled(cfgStore.Get(), DevelopmentOverride()) {
		return nil
	}
	if sessions != nil {
		rows, _ := sessions.CachedWorkspaceChoices(client)
		for _, row := range rows {
			m.primary.directoryCheckReason(win, client, row.Path)
		}
	}
	m.mu.Lock()
	current := f.client == client && f.listing.Available
	refused := map[string]bool{}
	for key, check := range f.directoryChecks {
		refused[key] = check.reason != ""
	}
	m.mu.Unlock()
	return WorkspaceDirectoryActions(current, func(path string) (bool, bool) {
		key := directoryKey(path)
		return active[key], refused[key]
	}, func(path string) { m.CreateFromDirectory(win, path) })
}

// SelectDirectory resolves an explicitly authorized registered workspace or opens the native directory picker.
func (m *Manager) SelectDirectory(ctx context.Context, suggested string) (string, error) {
	var win any
	if m.deps.MainWindow != nil {
		win = m.deps.MainWindow()
	}
	sessions := m.sessionsStore()
	cfgStore := m.configStore()
	snap := m.hostSnapshot()
	if win == nil || !snap.HasSupervisor {
		return "", errors.New(i18n.T(i18n.KeyCmdProjectDirectoryUnavailable))
	}
	if snap.RemoteConnected {
		return "", errors.New(i18n.T(i18n.KeyCmdProjectLocalDirectoryRemote))
	}
	st := snap.SupervisorStatus
	if st.State != harness.Ready || st.Endpoint.Origin == "" {
		return "", errors.New(i18n.T(i18n.KeyCmdWorkspaceHostUnavailable))
	}
	client := snap.Client

	m.mu.Lock()
	intent := m.primary.flow.directoryIntent
	if intent != nil && intent.client == client && m.primary.flow.client == client && strings.EqualFold(filepath.Clean(intent.path), filepath.Clean(suggested)) {
		m.primary.flow.directoryIntent = nil
	} else {
		intent = nil
	}
	m.mu.Unlock()

	if intent != nil && sessions != nil && client != nil && client.Endpoint.Origin == st.Endpoint.Origin && client.Endpoint.Cookie == st.Endpoint.Cookie {
		if choices, ready := sessions.CachedWorkspaceChoices(client); ready {
			for _, choice := range choices {
				if strings.EqualFold(filepath.Clean(choice.Path), filepath.Clean(intent.path)) {
					// The Project bridge still rechecks the Host registry and logical
					// authority before accepting the selected working directory.
					return choice.Path, nil
				}
			}
		}
	}
	if intent != nil {
		return "", errors.New("工作目录登记状态已变化，请刷新目录后重试。")
	}
	initial := ""
	if cfgStore != nil {
		initial = cfgStore.Get().Workspace
	}
	if directorypicker.SelectablePath(suggested) {
		initial = suggested
	}
	selected, err := directorypicker.SelectAndRegister(ctx, func() ([]string, error) {
		if m.deps.OpenDirectoryDialog == nil {
			return nil, errors.New(i18n.T(i18n.KeyCmdProjectDirectoryUnavailable))
		}
		return m.deps.OpenDirectoryDialog(win, i18n.T(i18n.KeyCmdChooseProjectDirectory), initial)
	}, func(ctx context.Context, chosen string) error {
		// A dialog may remain open while Host restarts; never register into a stale Host.
		currentSnap := m.hostSnapshot()
		current := currentSnap.SupervisorStatus
		if !currentSnap.HasSupervisor || current.State != harness.Ready || current.Endpoint != st.Endpoint {
			return errors.New(i18n.T(i18n.KeyCmdWorkspaceHostSelectionChanged))
		}
		registerCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if m.deps.EnsureWorkspace != nil {
			return m.deps.EnsureWorkspace(registerCtx, current.Endpoint, chosen)
		}
		return harness.EnsureWorkspace(registerCtx, nil, current.Endpoint, chosen)
	})
	if err != nil {
		return "", err
	}
	if cfgStore != nil {
		if err = cfgStore.Update(func(c *config.Config) { c.Workspace = selected }); err != nil {
			return "", errors.New(i18n.T(i18n.KeyCmdWorkspacePreferenceSaveFailed))
		}
	}
	if sessions != nil {
		sessions.SetWorkspaceCwd(selected)
	}
	m.updateWindow(win, func() {})
	return selected, nil
}

// The Hero option follows the selected Session's cwd, never the application's
// default workspace. The native source action authorizes only this same owner.
func (c *Controller) bindHero(win any, m *projects.Model, sessions *session.Store, client *dsh.Client, snapshot session.ViewState) {
	path := snapshot.SelectedCwd
	if !m.Listing.Available || client == nil || path == "" || directoryKey(path) == directoryKey(snapshot.ScratchCwd) {
		return
	}
	m.HeroWorkspace = path
	for _, project := range m.Listing.Projects {
		if project.Lifecycle != "archived" && directoryKey(project.CanonicalWorkingDirectory) == directoryKey(path) {
			m.HeroExisting = true
			break
		}
	}
	key := directoryKey(path)
	m.HeroDirectoryReason = c.directoryCheckReason(win, client, path)
	c.mgr.mu.Lock()
	check := c.flow.directoryChecks[key]
	m.HeroError = c.flow.createError
	c.mgr.mu.Unlock()
	m.OnCreateFromDraft = func() {
		now := sessions.SelectionSummary()
		if now.SelectedID != snapshot.SelectedID || directoryKey(now.SelectedCwd) != key {
			return
		}
		snap := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		current := check != nil && c.flow.client == client && c.flow.directoryChecks[key] == check && check.reason == ""
		c.mgr.mu.Unlock()
		if snap.Client == client && current {
			c.mgr.CreateWithDraft(win, path)
		}
	}
}

// directoryCheckReason shares the published asynchronous preflight
// between Hero and registered Workspace rows; no render waits for an RPC.
func (c *Controller) directoryCheckReason(win any, client *dsh.Client, path string) string {
	if client == nil || path == "" {
		return ""
	}
	key := directoryKey(path)
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	if f.client != client {
		c.mgr.mu.Unlock()
		return ""
	}
	if f.directoryChecks == nil {
		f.directoryChecks = map[string]*directoryCheck{}
	}
	check := f.directoryChecks[key]
	initial := check == nil
	if initial {
		check = &directoryCheck{}
		f.directoryChecks[key] = check
	}
	reason := check.reason
	c.mgr.mu.Unlock()
	if initial && win != nil {
		go func() {
			ctx, cancel := context.WithTimeout(lc, 15*time.Second)
			defer cancel()
			rpc := projectRPC(client)
			result, err := rpc.CheckDirectory(ctx, path)
			c.mgr.updateWindow(win, func() {
				c.mgr.mu.Lock()
				defer c.mgr.mu.Unlock()
				if c.flow.client != client || c.flow.directoryChecks[key] != check {
					return
				}
				// As in the installed client, a failed preflight leaves creation
				// available; the real creation boundary still enforces the rule.
				if err == nil && !result.Allowed {
					check.reason = result.Reason
					if check.reason == "" {
						check.reason = "unavailable"
					}
				}
			})
		}()
	}
	return reason
}
