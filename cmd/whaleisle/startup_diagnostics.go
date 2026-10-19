package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/importer"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/ui/launcher"
)

func offlinePluginProfile(cfg *config.Store) startuprec.OfflinePlugins {
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

func (s *mainState) startupDiagnostics() (startuprec.Diagnostics, bool) {
	s.mu.Lock()
	store, cfg, sup, st := s.startupRecovery.store, s.cfgStore, s.sup, s.bootStatus
	busy := s.marketInstallBusy || s.launcherImportBusy || s.taskRestartPending || s.launcherStartBusy || s.quitting
	s.mu.Unlock()
	var logs []string
	reason := ""
	if st.State == harness.Failed && sup != nil {
		logs = sup.CurrentStartLogs()
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
	inventory, err := offlinePluginProfile(cfg).Inventory()
	report := startuprec.Diagnose(logs, reason, inventory, cfg != nil && err == nil)
	generation := 0
	if sup != nil {
		generation = sup.Status().Generation
	}
	identity := fmt.Sprintf("%p\x00%d\x00%s", sup, generation, report.Receipt)
	sum := sha256.Sum256([]byte(identity))
	report.Receipt = hex.EncodeToString(sum[:])
	return report, busy
}

func (s *mainState) withStartupDiagnostics(m launcher.Model) launcher.Model {
	report, busy := s.startupDiagnostics()
	m.StartupDiagnostics = report
	m.StartupRepairBusy = busy
	m.OnRefreshStartupDiagnostics = func() { s.startupRepairNote("") }
	m.OnRepairStartupSuspects = func() { s.repairStartupSuspects(report.Receipt, slices.Clone(report.RepairNames)) }
	m.OnSetStartupPluginEnabled = func(name string, enabled bool) {
		s.changeStartupPlugins(report.Receipt, []string{name}, enabled, false, false)
	}
	m.OnRemoveStartupPlugin = func(name string) { s.changeStartupPlugins(report.Receipt, []string{name}, false, true, false) }
	return m
}
func (s *mainState) startupRepairNote(message string) {
	s.mu.Lock()
	s.launcherNote = message
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}
func (s *mainState) repairStartupSuspects(receipt string, names []string) {
	s.changeStartupPlugins(receipt, names, false, false, true)
}

// Use the existing shared import/market admission flags and Host task gate.
// Offline repair owns the same slot, including the profile write and restart.
func (s *mainState) changeStartupPlugins(receipt string, names []string, enabled, remove, retry bool) {
	report, busy := s.startupDiagnostics()
	if busy || !report.InventoryReady || report.Receipt != receipt || (retry && (!report.CanRepair || !slices.Equal(names, report.RepairNames))) {
		s.startupRepairNote(resumeText("诊断或插件库存已变化，请刷新后重试。", "Diagnostics or plugin inventory changed; refresh and retry."))
		return
	}
	if err := importer.CheckImportReady(paths.DshHome()); err != nil {
		s.startupRepairNote(startuprec.RedactLine(err.Error()))
		return
	}
	s.mu.Lock()
	if s.marketWorker != nil || s.marketInstallBusy || s.launcherImportBusy || s.taskRestartPending || s.launcherStartBusy || s.quitting {
		s.mu.Unlock()
		return
	}
	sup, win, cfg := s.sup, s.win, s.cfgStore
	if cfg == nil || (retry && sup == nil) {
		s.mu.Unlock()
		s.startupRepairNote("Startup configuration is unavailable.")
		return
	}
	s.marketInstallBusy = true
	s.taskRestartPending = true
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.marketInstallBusy = false
			s.taskRestartPending = false
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		var applied []string
		var actionErr error
		did := s.withTaskProtection(win, sup, "restart", func() error {
			current, _ := s.startupDiagnostics()
			if current.Receipt != receipt || (retry && (!current.CanRepair || !slices.Equal(names, current.RepairNames))) {
				return errors.New("suspects or plugin inventory changed; refresh diagnostics")
			}
			if err := importer.CheckImportReady(paths.DshHome()); err != nil {
				return err
			}
			s.mu.Lock()
			valid := s.sup == sup && !s.launcherImportBusy && !s.quitting
			s.mu.Unlock()
			if !valid {
				return errors.New("runtime or import changed; retry from fresh diagnostics")
			}
			profile := offlinePluginProfile(cfg)
			// A remove must stop the kernel before the bundled CLI changes its files.
			// Toggle alignment also stops a live/failed generation before profile edits.
			restart := retry || (sup != nil && sup.Status().State != harness.Stopped && !remove)
			if err := s.closeTaskProtectedResources(); err != nil {
				return err
			}
			if sup != nil {
				sup.Stop()
			}
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
				s.syncHarnessFromConfig()
				s.mu.Lock()
				s.launcherStartBusy = true
				s.startupRecovery.updateHold = false
				s.mu.Unlock()
				sup.Start()
			}
			return nil
		})
		if !did {
			message := resumeText("插件操作未完成。", "Plugin operation did not complete.")
			if len(applied) > 0 {
				message += " " + resumeText("已应用：", "Applied: ") + strings.Join(applied, ", ")
			}
			if actionErr != nil {
				message += " " + startuprec.RedactLine(actionErr.Error())
			}
			s.startupRepairNote(message)
			return
		}
		message := resumeText("已更新插件：", "Updated plugins: ") + strings.Join(applied, ", ")
		if retry {
			message += resumeText("。正在完整重试；Host 健康就绪后才算恢复完成。", ". Retrying full startup; recovery completes only after the Host is healthy.")
		}
		s.startupRepairNote(message)
	}()
}
