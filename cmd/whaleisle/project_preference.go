package main

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
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/desktop/projectpreference"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/settings"
)

type projectPreferenceState struct{ controller projectpreference.Controller }

// projectMountEnabled is the installed desktop policy, including its explicit
// development opt-in and disabled-plugin/recovery overrides.
func projectDevelopmentOverride() string { return os.Getenv("WHALE_PROJECT") }

func projectMountEnabled(cfg config.Config, environment string) bool {
	return (cfg.ProjectEnabled || environment == "1") && !cfg.SkipUserPlugins && !slices.Contains(cfg.DisabledPlugins, "dsh-project")
}

// Keep the prepared adapter private to the owned install server. Its overlay
// selection is recomputed from committed config on every actual start/restart.
// An omitted patch never changes the user patch, package or Project data.
func (s *mainState) projectLaunchAccess(initial config.Config) (func(context.Context, map[string]any) map[string]any, func(context.Context) ([]string, error)) {
	var bridgeMu sync.Mutex
	bridge := harness.NewProjectBridge(initial.NodeBin, paths.DshHome(), initial.Workspace, desktopcomponents.OpenInFileManager, s.selectProjectDirectory)
	currentConfig := func() config.Config {
		cfg := initial
		s.mu.Lock()
		store := s.cfgStore
		s.mu.Unlock()
		if store != nil {
			cfg = store.Get()
		}
		return cfg
	}
	read := func(ctx context.Context) ([]string, error) {
		cfg := currentConfig()
		if !projectMountEnabled(cfg, projectDevelopmentOverride()) {
			return nil, nil
		}
		if !harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{}).ArtifactsReady {
			return nil, nil
		}
		candidate := harness.NewProjectBridge(cfg.NodeBin, paths.DshHome(), cfg.Workspace, desktopcomponents.OpenInFileManager, s.selectProjectDirectory)
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
		if !projectMountEnabled(currentConfig(), projectDevelopmentOverride()) {
			return map[string]any{"ok": false, "available": false, "error": "Project is disabled"}
		}
		bridgeMu.Lock()
		current := bridge
		bridgeMu.Unlock()
		return current.Dispatch(ctx, payload)
	}
	return dispatch, read
}

func (s *mainState) projectPreferenceAccess() *settings.ProjectPreferenceAccess {
	s.mu.Lock()
	store := s.cfgStore
	sup := s.sup
	s.mu.Unlock()
	if store == nil {
		return nil
	}
	view := s.projectPreference.controller.Snapshot(store.Get().ProjectEnabled)
	a := &settings.ProjectPreferenceAccess{State: view}
	if sup != nil {
		a.OnSet = s.setProjectPreference
	}
	return a
}

func (s *mainState) setProjectPreference(next bool) {
	s.mu.Lock()
	store, sup, win, sessions := s.cfgStore, s.sup, s.win, s.sessions
	s.mu.Unlock()
	if store == nil || sup == nil {
		return
	}
	notify := func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	// A second event is rejected by the controller before touching config or work.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
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
			Acquire: func() (func(), error) { return importer.AcquireExternalMaintenance(paths.DshHome()) },
			LeaveLead: func(ctx context.Context) error {
				if sessions == nil {
					return nil
				}
				s.mu.Lock()
				var excluded []string
				for id, draft := range s.composerDrafts {
					if workspaceSessionDraftOccupied(draft) {
						excluded = append(excluded, id)
					}
				}
				s.mu.Unlock()
				return sessions.LeaveProjectLead(ctx, excluded...)
			},
			Protect: func(ctx context.Context, commit func() error) (bool, error) {
				s.mu.Lock()
				same := s.cfgStore == store && s.sup == sup && s.sessions == sessions && !s.launcherImportBusy && !s.quitting
				s.mu.Unlock()
				if !same {
					return false, errors.New("desktop runtime changed; retry Project setting")
				}
				// Acquire above already owns and checked the import lock. Re-acquiring
				// CheckImportReady here would reject this transaction's own lease.
				did, err := s.protectTaskAction(ctx, win, sup, "restart", func() error {
					s.mu.Lock()
					current := s.cfgStore == store && s.sup == sup && s.sessions == sessions && !s.launcherImportBusy && !s.quitting
					s.mu.Unlock()
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
				if err := s.closeTaskProtectedResources(); err != nil {
					return err
				}
				s.syncHarnessFromConfig()
				if err := sup.RestartAndWait(ctx); err != nil {
					return fmt.Errorf("Project restart failed: %w", err)
				}
				return nil
			}, Notify: notify,
		}
		_ = s.projectPreference.controller.Set(ctx, next, transition)
	}()
}
