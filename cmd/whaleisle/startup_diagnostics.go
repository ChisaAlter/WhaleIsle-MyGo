package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/ui/launcher"
)

func (s *mainState) startupDiagnostics() (startuprec.Diagnostics, bool) {
	s.dshMu.Lock()
	s.mu.Lock()
	store, cfg, sup, st := s.startupRecovery.store, s.cfgStore, s.sup, s.bootStatus
	market, client, busy := s.market, s.dshClient, s.marketInstallBusy || s.launcherImportBusy || s.taskRestartPending || s.quitting
	s.mu.Unlock()
	s.dshMu.Unlock()
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
	var inventory []startuprec.Plugin
	ready := false
	var runtimeToken uint64
	hostGeneration := 0
	if sup != nil {
		hostGeneration = sup.Status().Generation
	}
	if market != nil {
		runtimeToken = market.RuntimeToken()
		snap := market.Snapshot()
		ready = client != nil && snap.ClientReady && !snap.Loading && snap.Err == "" && sup != nil && sup.Status().State == harness.Ready && sup.Status().Generation == st.Generation
		for _, row := range snap.Rows {
			on := row.Enabled != nil && *row.Enabled
			restriction := row.ReadOnlyReason
			if restriction == "" && !row.IdentityKnown {
				restriction = resumeText("尚未确认插件身份", "Plugin identity is unconfirmed")
			}
			inventory = append(inventory, startuprec.Plugin{Name: row.Label, Enabled: on, Mutable: row.CanToggle(), Restriction: startuprec.RedactLine(restriction)})
		}
	}
	// Evidence equality is not runtime ownership. Bind every UI receipt to the
	// exact supervised Host, authenticated client and market attachment epoch.
	// Recheck after collecting evidence so a replacement mid-snapshot is inert.
	s.dshMu.Lock()
	s.mu.Lock()
	same := s.sup == sup && s.dshClient == client && s.market == market && s.bootStatus.Generation == st.Generation
	if market != nil {
		same = same && market.RuntimeToken() == runtimeToken
	}
	if sup != nil {
		actual := sup.Status()
		same = same && actual.Generation == hostGeneration && actual.State == harness.Ready
	}
	s.mu.Unlock()
	s.dshMu.Unlock()
	report := startuprec.Diagnose(logs, reason, inventory, ready && same)
	identity := fmt.Sprintf("%p\x00%d\x00%p\x00%p\x00%d\x00%s", sup, hostGeneration, client, market, runtimeToken, report.Receipt)
	sum := sha256.Sum256([]byte(identity))
	report.Receipt = hex.EncodeToString(sum[:])
	return report, busy
}

func (s *mainState) withStartupDiagnostics(m launcher.Model) launcher.Model {
	report, busy := s.startupDiagnostics()
	m.StartupDiagnostics = report
	m.StartupRepairBusy = busy
	m.OnRefreshStartupDiagnostics = func() {
		s.mu.Lock()
		store := s.market
		s.mu.Unlock()
		if store != nil {
			store.Refresh(context.Background())
		}
	}
	m.OnRepairStartupSuspects = func() { s.repairStartupSuspects(report.Receipt, append([]string(nil), report.RepairNames...)) }
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

// This is the real pluginManager bundle controller used by Settings, operating
// only on freshly verified suspects and inside the existing Host task gate.
// With no healthy controller/inventory it reports a blocker; it does not guess
// at profile files, invent a local disable API or reinstall unrelated plugins.
func (s *mainState) repairStartupSuspects(receipt string, names []string) {
	report, busy := s.startupDiagnostics()
	if busy || !report.CanRepair || report.Receipt != receipt || !slices.Equal(names, report.RepairNames) {
		s.startupRepairNote(resumeText("诊断或插件库存已变化，请刷新后重试。", "Diagnostics or plugin inventory changed; refresh and retry."))
		return
	}
	op, _ := s.beginMarketAttempt("", nil, false)
	if op == nil {
		return
	}
	op.method = "pluginManager/setBundleEnabled"
	s.mu.Lock()
	sup, win, cfg := s.sup, s.win, s.cfgStore
	s.mu.Unlock()
	if sup == nil || cfg == nil || op.store == nil {
		s.finishMarket(op, marketResult{message: resumeText("恢复模式 Host 的插件控制器尚未就绪。", "The recovery-mode Host plugin controller is not ready.")}, false)
		s.releaseMarket(op)
		return
	}
	go func() {
		defer s.releaseMarket(op)
		var applied []string
		var repairErr error
		protected := s.withTaskProtection(win, sup, "restart", func() error {
			// Bound mutations after task confirmation, not the user's reading time.
			ctx, cancel := context.WithTimeout(op.ctx, 30*time.Second)
			defer cancel()
			current, _ := s.startupDiagnostics()
			if current.Receipt != receipt || !current.CanRepair || !slices.Equal(names, current.RepairNames) {
				return errors.New("suspects changed; refresh diagnostics")
			}
			check := func() error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := importer.CheckImportReady(paths.DshHome()); err != nil {
					return err
				}
				s.mu.Lock()
				valid := s.marketCurrentLocked(op) && s.sup == sup && !s.launcherImportBusy && !s.quitting
				s.mu.Unlock()
				if !valid {
					return errors.New("runtime or import changed; retry from fresh diagnostics")
				}
				return nil
			}
			applied, repairErr = startuprec.Repair(names, startuprec.RepairActions{
				Check: check,
				Disable: func(name string) error {
					if err := op.store.ValidateBundleChange(name, false); err != nil {
						return err
					}
					var out *api.ChangeResult
					err := op.client.Call(ctx, op.method, map[string]any{"name": name, "enabled": false}, &out)
					result := marketResultFor(out, err)
					if !result.accepted {
						return errors.New(startuprec.RedactLine(result.message))
					}
					if result.outcome == "restart-required" {
						s.mu.Lock()
						s.marketActivation.Require(sup.Status().Generation)
						s.mu.Unlock()
					}
					return nil
				},
				ClearRecovery: cfg.ClearPluginRecovery,
				Restart: func() error {
					if err := check(); err != nil {
						return err
					}
					s.syncHarnessFromConfig()
					s.mu.Lock()
					s.launcherStartBusy = true
					s.startupRecovery.updateHold = false
					s.mu.Unlock()
					s.stopPhoneRuntime()
					sup.Restart()
					return nil
				},
			})
			return repairErr
		})
		if !protected {
			message := resumeText("针对性修复未完成；恢复状态仍需确认。", "Targeted repair did not complete; recovery still needs review.")
			if len(applied) > 0 {
				message += fmt.Sprintf(resumeText(" 已确认禁用：%s。其余插件未自动处理。", " Confirmed disabled: %s. Remaining plugins were not changed automatically."), strings.Join(applied, ", "))
			}
			if repairErr != nil {
				message += " " + startuprec.RedactLine(repairErr.Error())
			}
			s.finishMarket(op, marketResult{message: message}, false)
			s.startupRepairNote(message)
			return
		}
		s.startupRepairNote(fmt.Sprintf(resumeText("已确认禁用：%s。正在完整重试；只有 Host 健康就绪后才算恢复完成。", "Confirmed disabled: %s. Retrying the full startup; recovery completes only after the Host is healthy."), strings.Join(applied, ", ")))
	}()
}
