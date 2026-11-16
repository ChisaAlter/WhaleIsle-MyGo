package launcherctl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/importer"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/ui/i18n"
)

type startupInventoryResult struct {
	plugins []startuprec.Plugin
	err     error
}

// OfflinePluginProfile constructs offline profile read/toggle/remove operations for cfg.
func OfflinePluginProfile(cfg *config.Store) startuprec.OfflinePlugins {
	p := startuprec.OfflinePlugins{ProfileDir: paths.WebProfileDir()}
	if cfg != nil {
		p.LoadDisabled = func() []string { return cfg.Get().DisabledPlugins }
		p.SaveDisabled = func(names []string) error { return cfg.Update(func(c *config.Config) { c.DisabledPlugins = names }) }
		p.Uninstall = func(name string) error {
			result := desktopmarket.RemovePluginContext(context.Background(), name, desktopmarket.InstallOptions{NodeBin: cfg.Get().NodeBin, DshHome: paths.DshHome()})
			if !result.OK {
				return errors.New(result.Error)
			}
			return nil
		}
	}
	return p
}

// StartupInventory reads or serves the 1s cached offline plugin inventory.
func (c *Controller) StartupInventory(cfg *config.Store, fresh bool, notify func()) ([]startuprec.Plugin, error) {
	profile := OfflinePluginProfile(cfg)
	key := profile.ProfileDir
	if cfg != nil {
		key += "\x00" + strings.Join(cfg.Get().DisabledPlugins, "\x00")
	}
	read := func() startupInventoryResult {
		plugins, err := profile.Inventory()
		return startupInventoryResult{plugins, err}
	}
	if fresh {
		r := read()
		c.inventoryProbe.put(key, r)
		return r.plugins, r.err
	}
	r := c.inventoryProbe.get(key, time.Second, read, notify)
	return r.plugins, r.err
}

// StartupDiagnostics computes the offline plugin repair receipt and shared busy state.
func (c *Controller) StartupDiagnostics(fresh bool, supID string, notify func()) (startuprec.Diagnostics, bool) {
	env := c.hostEnv()
	cfg := env.ConfigStore
	st := env.BootStatus

	c.mu.Lock()
	store := c.recovery.Store
	startBusy := c.startBusy
	c.mu.Unlock()

	marketBusy := false
	if c.deps != nil {
		marketBusy = c.deps.MarketBusy()
	}
	busy := marketBusy || env.LauncherImportBusy || env.TaskRestartPending || startBusy || env.Quitting

	var logs []string
	reason := ""
	if st.State == harness.Failed && env.HasSupervisor {
		logs = env.StartLogs
		if st.Err != nil {
			reason = st.Err.Error()
		}
	} else if cfg != nil && cfg.PluginRecoveryEvidence().SkipUserPlugins {
		rec := cfg.PluginRecoveryEvidence()
		logs = rec.LogTail
		reason = rec.Reason
	} else if store != nil && store.Last().Failed() {
		last := store.Last()
		logs = last.LogTail
		reason = last.Error
	}
	inventory, err := c.StartupInventory(cfg, fresh, notify)
	report := startuprec.Diagnose(logs, reason, inventory, cfg != nil && err == nil)
	generation := 0
	if env.HasSupervisor {
		generation = st.Generation
	}
	identity := fmt.Sprintf("%s\x00%d\x00%s", supID, generation, report.Receipt)
	sum := sha256.Sum256([]byte(identity))
	report.Receipt = hex.EncodeToString(sum[:])
	return report, busy
}

// StartupRepairNote updates the launcher banner and triggers a main window update.
func (c *Controller) StartupRepairNote(message string) {
	c.mu.Lock()
	c.note = message
	c.mu.Unlock()
	env := c.hostEnv()
	if c.deps != nil && env.MainWindow != nil {
		c.deps.UpdateWindow(env.MainWindow, func() {})
	}
}

// RepairSuspects triggers offline repair of suspect startup plugins matching receipt.
func (c *Controller) RepairSuspects(receipt string, names []string, supID string) {
	c.ChangeStartupPlugins(receipt, names, false, false, true, supID)
}

// ChangeStartupPlugins shares the market/import slot and protected task barrier
// to toggle, remove, or repair offline plugins cleanly.
func (c *Controller) ChangeStartupPlugins(receipt string, names []string, enabled, remove, retry bool, supID string) {
	if c.deps == nil {
		return
	}
	report, busy := c.StartupDiagnostics(true, supID, nil)
	if busy || !report.InventoryReady || report.Receipt != receipt || (retry && (!report.CanRepair || !slices.Equal(names, report.RepairNames))) {
		c.StartupRepairNote(i18n.T(i18n.KeyCmdDiagnosticsOrPluginInventoryChangedRef))
		return
	}
	if err := importer.CheckImportReady(paths.DshHome()); err != nil {
		c.StartupRepairNote(startuprec.RedactLine(err.Error()))
		return
	}

	env := c.hostEnv()
	c.mu.Lock()
	startBusy := c.startBusy
	c.mu.Unlock()
	if env.LauncherImportBusy || env.TaskRestartPending || startBusy || env.Quitting {
		return
	}
	if env.ConfigStore == nil || (retry && !env.HasSupervisor) {
		c.StartupRepairNote(i18n.T(i18n.KeyCmdStartupConfigUnavailable))
		return
	}
	releaseMarket, ok := c.deps.TryAcquireRepairSlot()
	if !ok {
		return
	}
	if !c.deps.TrySetTaskRestartPending(true) {
		releaseMarket()
		return
	}

	cfg := env.ConfigStore
	win := env.MainWindow
	go func() {
		defer func() {
			c.deps.TrySetTaskRestartPending(false)
			releaseMarket()
			if win != nil {
				c.deps.UpdateWindow(win, func() {})
			}
		}()
		var applied []string
		var actionErr error
		did := c.deps.WithTaskProtection(win, "restart", func() error {
			current, _ := c.StartupDiagnostics(true, supID, nil)
			if current.Receipt != receipt || (retry && (!current.CanRepair || !slices.Equal(names, current.RepairNames))) {
				return errors.New(i18n.T(i18n.KeyCmdStartupPluginInventoryChanged))
			}
			if err := importer.CheckImportReady(paths.DshHome()); err != nil {
				return err
			}
			curEnv := c.hostEnv()
			if !c.deps.HasCurrentSupervisor() || curEnv.LauncherImportBusy || curEnv.Quitting {
				return errors.New(i18n.T(i18n.KeyCmdStartupRuntimeChanged))
			}
			profile := OfflinePluginProfile(cfg)
			restart := retry || (curEnv.HasSupervisor && curEnv.BootStatus.State != harness.Stopped && !remove)
			if err := c.deps.CloseTaskProtectedResources(); err != nil {
				return err
			}
			c.deps.StopSupervisorIfPresent()
			for _, name := range names {
				if err := importer.CheckImportReady(paths.DshHome()); err != nil {
					actionErr = err
					return err
				}
				if remove {
					actionErr = profile.Remove(name)
				} else {
					actionErr = profile.SetEnabled(name, enabled)
				}
				if actionErr != nil {
					return actionErr
				}
				applied = append(applied, name)
			}
			if retry {
				if actionErr = cfg.ClearPluginRecovery(); actionErr != nil {
					return actionErr
				}
			}
			if restart {
				if actionErr = importer.CheckImportReady(paths.DshHome()); actionErr != nil {
					return actionErr
				}
				c.deps.SyncHarnessFromConfig()
				c.SetStartBusy(true)
				if u := c.updaterCenter(); u != nil {
					u.SetHold(false)
				}
				c.deps.StartHarnessWithConfig(cfg.Get())
			}
			return nil
		})

		if !did {
			message := i18n.T(i18n.KeyCmdPluginOperationDidNotComplete)
			if len(applied) > 0 {
				message += " " + i18n.T(i18n.KeyCmdApplied) + strings.Join(applied, ", ")
			}
			if actionErr != nil {
				message += " " + startuprec.RedactLine(actionErr.Error())
			}
			c.StartupRepairNote(message)
			return
		}
		message := i18n.T(i18n.KeyCmdUpdatedPlugins) + strings.Join(applied, ", ")
		if retry {
			message += i18n.T(i18n.KeyCmdRetryingFullStartupRecoveryCompletesOn)
		}
		c.StartupRepairNote(message)
	}()
}
