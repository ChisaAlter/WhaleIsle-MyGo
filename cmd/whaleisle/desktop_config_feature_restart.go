package main

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/taskcontrol"
)

// Only the explicitly patched feature fields belong to this restart. Desktop
// appearance, login and recovery fields retain their independent save effects.
type desktopConfigFeaturePatch struct {
	dshbot    *desktopConfigFeatureValue
	assistant *desktopConfigFeatureValue
	remote    *desktopConfigFeatureValue
}

type desktopConfigFeatureValue struct {
	previous bool
	expected bool
}

var errDesktopConfigFeaturesUnchanged = errors.New("no feature values remain owned by this patch")

func (s *mainState) rollbackDesktopConfigFeatures(store *config.Store, patch desktopConfigFeaturePatch) error {
	err := store.UpdateChecked(func(c *config.Config) error {
		changed := false
		restore := func(value *bool, owned *desktopConfigFeatureValue) {
			// A newer value is another writer's decision, not ours to undo.
			if owned != nil && owned.previous != owned.expected && *value == owned.expected {
				*value = owned.previous
				changed = true
			}
		}
		restore(&c.DshbotEnabled, patch.dshbot)
		restore(&c.WhaleAssistantEnabled, patch.assistant)
		restore(&c.RemoteWorkspaceEnabled, patch.remote)
		if !changed {
			return errDesktopConfigFeaturesUnchanged
		}
		return nil
	})
	if errors.Is(err, errDesktopConfigFeaturesUnchanged) {
		err = nil
	}
	if err == nil {
		// Also replace the supervisor's rejected next-start overlay. This reads
		// the current accepted config, including unrelated intervening writes.
		s.syncHarnessFromConfig()
	}
	return err
}

func (s *mainState) completeDesktopConfigFeatures(store *config.Store, patch desktopConfigFeaturePatch, win *mygo.Window, did bool, restartErr error) {
	if did && restartErr == nil {
		return
	}
	if restartErr == nil || errors.Is(restartErr, taskcontrol.ErrCancelled) {
		restartErr = errors.New(lifecycleText("功能重启已取消，未接受此次功能更改。", "Feature restart cancelled; these feature changes were not accepted."))
	} else {
		restartErr = fmt.Errorf("%s: %w", lifecycleText("功能更改未能完成", "Feature changes could not be completed"), restartErr)
	}
	if err := s.rollbackDesktopConfigFeatures(store, patch); err != nil {
		restartErr = errors.Join(restartErr, fmt.Errorf("%s: %w", lifecycleText("恢复先前功能设置失败", "Could not restore the previous feature settings"), err))
	}
	// The bridge already returned its queued response. Use the existing visible
	// launcher error surface for refusal/cancellation as well as start failure.
	s.taskProtectionError(win, restartErr)
}
