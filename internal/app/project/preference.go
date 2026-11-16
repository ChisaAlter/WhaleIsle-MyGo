package projectctl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/projectpreference"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/settings"
)

// DevelopmentOverride returns the explicit WHALE_PROJECT environment opt-in value.
func DevelopmentOverride() string { return os.Getenv("WHALE_PROJECT") }

// MountEnabled reports whether Project should be mounted for cfg and the environment override.
func MountEnabled(cfg config.Config, environment string) bool {
	return (cfg.ProjectEnabled || environment == "1") && !cfg.SkipUserPlugins && !slices.Contains(cfg.DisabledPlugins, "dsh-project")
}

// LaunchAccess prepares the private Host project bridge dispatch and launch-patch reader callbacks.
func (m *Manager) LaunchAccess(initial config.Config) (func(context.Context, map[string]any) map[string]any, func(context.Context) ([]string, error)) {
	var bridgeMu sync.Mutex
	openPath := m.deps.OpenWorkspaceDirectory
	bridge := harness.NewProjectBridge(initial.NodeBin, paths.DshHome(), initial.Workspace, openPath, m.SelectDirectory)
	currentConfig := func() config.Config {
		cfg := initial
		if store := m.configStore(); store != nil {
			cfg = store.Get()
		}
		return cfg
	}
	read := func(ctx context.Context) ([]string, error) {
		cfg := currentConfig()
		if !MountEnabled(cfg, DevelopmentOverride()) {
			return nil, nil
		}
		if !harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{}).ArtifactsReady {
			return nil, nil
		}
		candidate := harness.NewProjectBridge(cfg.NodeBin, paths.DshHome(), cfg.Workspace, openPath, m.SelectDirectory)
		available := candidate.Ready(ctx)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !available {
			return nil, nil
		}
		overlay, err := candidate.PrepareOverlay(ctx, filepath.Join(paths.VendorDir(), "dsh-project"), paths.WebProfileDir(), paths.HarnessRoot())
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			log.Println("Project unavailable: desktop overlay preparation failed")
			return nil, nil
		}
		bridgeMu.Lock()
		bridge = candidate
		bridgeMu.Unlock()
		return []string{overlay}, nil
	}
	dispatch := func(ctx context.Context, payload map[string]any) map[string]any {
		if !MountEnabled(currentConfig(), DevelopmentOverride()) {
			return map[string]any{"ok": false, "available": false, "error": "Project is disabled"}
		}
		bridgeMu.Lock()
		current := bridge
		bridgeMu.Unlock()
		return current.Dispatch(ctx, payload)
	}
	return dispatch, read
}

// PreferenceAccess returns the settings UI projection and toggle handler for Project enablement.
func (m *Manager) PreferenceAccess() *settings.ProjectPreferenceAccess {
	store := m.configStore()
	snap := m.hostSnapshot()
	if store == nil {
		return nil
	}
	view := m.preference.Snapshot(store.Get().ProjectEnabled)
	a := &settings.ProjectPreferenceAccess{State: view}
	if snap.HasSupervisor {
		a.OnSet = m.SetPreference
	}
	return a
}

// SetPreference toggles Project enablement asynchronously through task protection and Host restart.
func (m *Manager) SetPreference(next bool) {
	store := m.configStore()
	sessions := m.sessionsStore()
	snap := m.hostSnapshot()
	var win any
	if m.deps.MainWindow != nil {
		win = m.deps.MainWindow()
	}
	if store == nil || !snap.HasSupervisor {
		return
	}
	supID := snap.SupervisorID
	lc := m.lifecycleContext()
	notify := func() {
		if win != nil {
			m.updateWindow(win, func() {})
		}
	}
	// A second event is rejected by the controller before touching config or work.
	go func() {
		ctx, cancel := context.WithTimeout(lc, 4*time.Minute)
		defer cancel()
		transition := projectpreference.Transition{
			Read: func() bool { return store.Get().ProjectEnabled },
			Write: func(expected, value bool) error {
				return store.UpdateChecked(func(c *config.Config) error {
					if c.ProjectEnabled != expected {
						return errors.New("Project setting changed; retry")
					}
					c.ProjectEnabled = value
					return nil
				})
			},
			Acquire: m.deps.AcquireExternalMaintenance,
			LeaveLead: func(ctx context.Context) error {
				if sessions == nil {
					return nil
				}
				var excluded []string
				if m.deps.OccupiedDraftSessionIDs != nil {
					excluded = m.deps.OccupiedDraftSessionIDs()
				}
				return sessions.LeaveProjectLead(ctx, excluded...)
			},
			Protect: func(ctx context.Context, commit func() error) (bool, error) {
				now := m.hostSnapshot()
				same := m.configStore() == store && now.HasSupervisor && now.SupervisorID == supID && m.sessionsStore() == sessions && !now.ImportBusy && !now.Quitting
				if !same {
					return false, errors.New("desktop runtime changed; retry Project setting")
				}
				if m.deps.ProtectTaskAction == nil {
					return false, errors.New("task protection is unavailable")
				}
				// Acquire above already owns and checked the import lock. Re-acquiring
				// CheckImportReady here would reject this transaction's own lease.
				did, err := m.deps.ProtectTaskAction(ctx, win, "restart", func() error {
					cur := m.hostSnapshot()
					current := m.configStore() == store && cur.HasSupervisor && cur.SupervisorID == supID && m.sessionsStore() == sessions && !cur.ImportBusy && !cur.Quitting
					if !current {
						return errors.New("desktop runtime changed during Project confirmation")
					}
					if !next && sessions != nil && sessions.ProjectLeadSelected() {
						return errors.New("leave the current Project Lead before disabling Project")
					}
					return commit()
				})
				if errors.Is(err, taskcontrol.ErrCancelled) {
					return false, nil
				}
				return did, err
			},
			Restart: func(ctx context.Context) error {
				if m.deps.RestartHost == nil {
					return errors.New("Host restart is unavailable")
				}
				if err := m.deps.RestartHost(ctx); err != nil {
					return fmt.Errorf("Project restart failed: %w", err)
				}
				return nil
			},
			Notify: notify,
		}
		_ = m.preference.Set(ctx, next, transition)
	}()
}
