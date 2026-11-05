package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/desktop/projectpreference"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/settings"
)

// Each switch keeps its own accepted/pending view; the existing maintenance
// lease serializes the shared Harness restart with Project and profile import.
type featurePreferenceState struct {
	assistant projectpreference.Controller
	remote    projectpreference.Controller
}

type builtInFeature uint8

const (
	whaleAssistantFeature builtInFeature = iota
	remoteWorkspaceFeature
)

func featureEnabled(c config.Config, feature builtInFeature) bool {
	if feature == whaleAssistantFeature {
		return c.WhaleAssistantEnabled
	}
	return c.RemoteWorkspaceEnabled
}

func (s *mainState) featurePreferenceController(feature builtInFeature) *projectpreference.Controller {
	if feature == whaleAssistantFeature {
		return &s.featurePreferences.assistant
	}
	return &s.featurePreferences.remote
}

func (s *mainState) featurePreferenceAccess(feature builtInFeature) *settings.ProjectPreferenceAccess {
	s.mu.Lock()
	store, sup := s.cfgStore, s.sup
	s.mu.Unlock()
	if store == nil {
		return nil
	}
	view := s.featurePreferenceController(feature).Snapshot(featureEnabled(store.Get(), feature))
	label := lifecycleText("鲸鱼娘助理", "Whale assistant")
	if feature == remoteWorkspaceFeature {
		label = lifecycleText("远程工作区", "Remote workspace")
	}
	view.Error = strings.ReplaceAll(view.Error, "Project", label)
	access := &settings.ProjectPreferenceAccess{State: view}
	if sup != nil {
		access.OnSet = func(next bool) { s.setFeaturePreference(feature, next) }
	}
	return access
}

func (s *mainState) setWhaleAssistantPreference(next bool) {
	s.setFeaturePreference(whaleAssistantFeature, next)
}

func (s *mainState) setFeaturePreference(feature builtInFeature, next bool) {
	s.mu.Lock()
	store, sup, win := s.cfgStore, s.sup, s.win
	s.mu.Unlock()
	if store == nil || sup == nil {
		return
	}
	controller := s.featurePreferenceController(feature)
	notify := func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		_ = controller.Set(ctx, next, projectpreference.Transition{
			Read: func() bool { return featureEnabled(store.Get(), feature) },
			Write: func(expected, value bool) error {
				err := store.UpdateChecked(func(c *config.Config) error {
					if featureEnabled(*c, feature) != expected {
						return errors.New(lifecycleText("功能设置已改变，请重试。", "Feature setting changed; retry."))
					}
					if feature == whaleAssistantFeature {
						c.WhaleAssistantEnabled = value
					} else {
						c.RemoteWorkspaceEnabled = value
					}
					return nil
				})
				if err == nil {
					// Rollback uses this same write, so the next start never retains
					// the rejected overlay selection either.
					s.syncHarnessFromConfig()
				}
				return err
			},
			Acquire: func() (func(), error) { return importer.AcquireExternalMaintenance(paths.DshHome()) },
			Protect: func(ctx context.Context, commit func() error) (bool, error) {
				current := func() bool {
					s.mu.Lock()
					defer s.mu.Unlock()
					return s.cfgStore == store && s.sup == sup && !s.launcherImportBusy && !s.quitting
				}
				if !current() {
					return false, errors.New(lifecycleText("桌面运行状态已改变，请重试。", "Desktop runtime changed; retry."))
				}
				did, err := s.protectTaskAction(ctx, win, sup, "restart", func() error {
					if !current() {
						return errors.New(lifecycleText("确认期间运行状态已改变，请重试。", "Runtime changed during confirmation; retry."))
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
				if err := sup.RestartAndWait(ctx); err != nil {
					return fmt.Errorf("%s: %w", lifecycleText("功能重启失败", "Feature restart failed"), err)
				}
				return nil
			},
			Notify: notify,
		})
	}()
}
