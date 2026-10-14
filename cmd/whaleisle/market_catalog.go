package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
)

type marketAttempt struct {
	client          *dsh.Client
	store           *marketstore.Store
	generation      uint64
	ctx             context.Context
	cancel          context.CancelFunc
	requestID, spec string
	method          string
	target          *marketstore.InstallTarget
	cancelPending   bool
}

func marketMessage(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}
func (s *mainState) marketCurrentLocked(op *marketAttempt) bool {
	return op != nil && s.marketActive == op && s.dshClient == op.client && s.market == op.store && (op.store == nil || op.store.RuntimeToken() == op.generation)
}
func (s *mainState) invalidateMarketLocked() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.marketActive != nil {
		s.marketActive.cancel()
	}
	s.marketActive = nil
	s.marketInstallBusy = false
	s.marketInstallRequestID = ""
	s.marketApproval = nil
	s.marketCatalogTarget = nil
	s.marketNeedsAllowBuilds = false
	s.marketAllowBuilds = nil
	s.marketIncompatible = nil
}
func (s *mainState) runMarketCatalogInstall(target marketstore.InstallTarget) {
	s.dshMu.Lock()
	s.mu.Lock()
	if s.market == nil || s.marketWorker != nil {
		s.mu.Unlock()
		s.dshMu.Unlock()
		return
	}
	if err := s.market.ValidateTarget(target); err != nil {
		s.marketInstallMsg = err.Error()
		s.mu.Unlock()
		s.dshMu.Unlock()
		return
	}
	s.marketCatalogTarget = &target
	s.marketInstallSpec = target.Spec
	s.mu.Unlock()
	s.dshMu.Unlock()
	s.runMarketInstall(target.Spec, nil)
}
func (s *mainState) beginMarketAttempt(spec string, allow []string, install bool, approval ...*marketAttempt) (*marketAttempt, string) {
	s.dshMu.Lock()
	defer s.dshMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.marketWorker != nil || s.marketInstallBusy {
		return nil, ""
	}
	if len(approval) > 0 {
		a := approval[0]
		if a == nil || s.marketApproval != a || a.client != s.dshClient || a.store != s.market || (a.store != nil && a.store.RuntimeToken() != a.generation) {
			return nil, ""
		}
		if a.target != nil && (a.store == nil || a.store.ValidateTarget(*a.target) != nil) {
			return nil, ""
		}
	}
	if !install && s.dshClient == nil {
		return nil, ""
	}
	if install && len(allow) > 0 {
		a := s.marketApproval
		if a == nil || a.spec != spec || a.client != s.dshClient || a.store != s.market || (a.store != nil && a.store.RuntimeToken() != a.generation) || !slices.Equal(allow, s.marketAllowBuilds) {
			s.marketInstallMsg = marketMessage("构建授权已过期，请重新安装以确认。", "Build approval expired; start installation again.")
			return nil, ""
		}
	}
	var target *marketstore.InstallTarget
	if install && s.marketCatalogTarget != nil && s.marketCatalogTarget.Spec == spec {
		t := *s.marketCatalogTarget
		if s.market == nil {
			return nil, ""
		}
		if e := s.market.ValidateTarget(t); e != nil {
			s.marketInstallMsg = e.Error()
			return nil, ""
		}
		target = &t
	} else if install {
		s.marketCatalogTarget = nil
	}
	op := &marketAttempt{client: s.dshClient, store: s.market, spec: spec, target: target}
	if op.store != nil {
		op.generation = op.store.RuntimeToken()
	}
	op.ctx, op.cancel = context.WithCancel(context.Background())
	if install && op.client != nil {
		op.requestID = marketInstallRequestID()
	}
	s.marketWorker = op
	s.marketActive = op
	s.marketInstallBusy = true
	s.marketInstallRequestID = op.requestID
	s.marketInstallMsg = ""
	s.marketInstallLog = ""
	if install {
		s.marketInstallSpec = spec
		s.marketNeedsAllowBuilds = false
		s.marketAllowBuilds = nil
		s.marketApproval = nil
		s.marketIncompatible = nil
	}
	node := ""
	if s.cfgStore != nil {
		node = s.cfgStore.Get().NodeBin
	}
	return op, node
}

type marketResult struct {
	message, log string
	accepted     bool
	pending      []string
	incompatible []api.ChangeResultErrorIncompatibleItem
}

func marketResultFor(out *api.ChangeResult, err error) marketResult {
	r := marketResult{}
	if err != nil {
		r.message = marketMessage("结果未知：", "Result unknown: ") + err.Error()
		return r
	}
	if out == nil {
		r.message = marketMessage("结果未知，请刷新库存。", "Result unknown; refresh inventory.")
		return r
	}
	r.pending = append([]string(nil), out.PendingBuilds...)
	if out.PackageResult != nil {
		r.log = out.PackageResult.Output
	}
	if len(out.Warnings) > 0 {
		r.log = strings.TrimSpace(r.log + "\n" + strings.Join(out.Warnings, "\n"))
	}
	if out.Error != nil {
		r.message = out.Error.Code
		if out.Error.Diagnostic != nil {
			r.message += " " + *out.Error.Diagnostic
		}
		r.incompatible = append([]api.ChangeResultErrorIncompatibleItem(nil), out.Error.Incompatible...)
		if strings.TrimSpace(r.message) == "" {
			r.message = marketMessage("操作失败。", "Operation failed.")
		}
	} else {
		switch out.Application {
		case "applied":
			r.accepted = true
			r.message = marketMessage("已应用；正在核对库存。", "Applied; checking inventory.")
		case "restart-required":
			r.accepted = true
			r.message = marketMessage("已保存，需重启运行时后生效。", "Saved; a runtime restart is required to take effect.")
		case "cancelled":
			r.message = marketMessage("安装已取消。", "Installation cancelled.")
		case "failed":
			r.message = marketMessage("操作失败。", "Operation failed.")
		default:
			r.message = marketMessage("结果未确认，请刷新库存。", "Result unconfirmed; refresh inventory.")
		}
	}
	if len(r.pending) > 0 {
		r.accepted = false
		r.message += " " + marketMessage("需要明确授权以下构建脚本。", "Explicit build-script approval is required.")
	}
	return r
}
func (s *mainState) finishMarket(op *marketAttempt, r marketResult, install bool) {
	s.dshMu.Lock()
	defer s.dshMu.Unlock()
	s.mu.Lock()
	if !s.marketCurrentLocked(op) {
		s.mu.Unlock()
		return
	}
	s.marketActive = nil
	s.marketInstallBusy = false
	s.marketInstallRequestID = ""
	s.marketInstallMsg = r.message
	s.marketInstallLog = r.log
	if install {
		s.marketAllowBuilds = r.pending
		s.marketNeedsAllowBuilds = len(r.pending) > 0
		s.marketIncompatible = r.incompatible
		if len(r.pending) > 0 || len(r.incompatible) > 0 {
			s.marketApproval = op
		} else {
			s.marketApproval = nil
		}
	}
	if op.method == "pluginManager/setVersionExemption" && r.accepted {
		s.marketIncompatible = nil
	}
	win := s.win
	s.mu.Unlock()
	if op.store != nil {
		op.store.Refresh(context.Background())
	}
	if win != nil {
		win.Update(func() {})
	}
}
func (s *mainState) releaseMarket(op *marketAttempt) {
	op.cancel()
	s.mu.Lock()
	if s.marketWorker == op {
		s.marketWorker = nil
	}
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}
func (s *mainState) runMarketInstall(spec string, allowBuilds []string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return
	}
	allowBuilds = append([]string(nil), allowBuilds...)
	op, node := s.beginMarketAttempt(spec, allowBuilds, true)
	if op == nil {
		return
	}
	go func() {
		defer s.releaseMarket(op)
		var r marketResult
		if op.client != nil {
			opts := map[string]any{"requestId": op.requestID}
			if len(allowBuilds) > 0 {
				opts["approvedBuilds"] = allowBuilds
			}
			var out *api.ChangeResult
			err := op.client.Call(op.ctx, "pluginManager/installBundle", map[string]any{"spec": spec, "options": opts}, &out)
			pending := false
			if out != nil {
				switch strings.ToLower(strings.TrimSpace(out.Stage)) {
				case "pending", "running", "installing", "queued":
					pending = true
				}
			}
			if (err != nil || pending) && op.ctx.Err() == nil {
				var waited *api.ChangeResult
				if e := op.client.Call(op.ctx, "pluginManager/waitForInstall", map[string]any{"requestId": op.requestID}, &waited); e == nil && waited != nil {
					out, err = waited, nil
				} else if e != nil {
					err = e
				}
			}
			r = marketResultFor(out, err)
		} else {
			res := desktopmarket.InstallGithubSpec(spec, allowBuilds, desktopmarket.InstallOptions{NodeBin: node})
			r = marketResult{message: res.Error, log: res.Log}
			if res.NeedsAllowBuilds {
				r.pending = append([]string(nil), res.AllowBuilds...)
			}
			if res.OK && !res.NeedsAllowBuilds {
				r.accepted = true
				r.message = marketMessage("已保存，需重启运行时后生效。", "Saved; a runtime restart is required.")
			}
			if r.message == "" {
				r.message = marketMessage("结果未确认。", "Result unconfirmed.")
			}
		}
		s.finishMarket(op, r, true)
	}()
}
func (s *mainState) runMarketChange(method string, args map[string]any) {
	s.runMarketChangeApproved(method, args)
}
func (s *mainState) runMarketChangeApproved(method string, args map[string]any, approval ...*marketAttempt) {
	op, _ := s.beginMarketAttempt("", nil, false, approval...)
	if op == nil {
		return
	}
	op.method = method
	go func() {
		defer s.releaseMarket(op)
		var out *api.ChangeResult
		e := op.client.Call(op.ctx, method, args, &out)
		s.finishMarket(op, marketResultFor(out, e), false)
	}()
}
func (s *mainState) acceptMarketExemption(packageVersion, runtimeVersion string) {
	s.dshMu.Lock()
	s.mu.Lock()
	a := s.marketApproval
	valid := a != nil && a.client == s.dshClient && a.store == s.market && (a.store == nil || a.store.RuntimeToken() == a.generation)
	match := false
	for _, it := range s.marketIncompatible {
		if it.Name+"@"+it.Version == packageVersion && it.RuntimeVersion == runtimeVersion {
			match = true
		}
	}
	if valid && a.target != nil {
		valid = s.market != nil && s.market.ValidateTarget(*a.target) == nil
	}
	s.mu.Unlock()
	s.dshMu.Unlock()
	if !valid || !match {
		return
	}
	s.runMarketChangeApproved("pluginManager/setVersionExemption", map[string]any{"packageVersion": packageVersion, "runtimeVersion": runtimeVersion, "enabled": true, "acceptRisk": true}, a)
}
func (s *mainState) cancelMarketInstall() {
	s.dshMu.Lock()
	s.mu.Lock()
	op := s.marketActive
	if !s.marketCurrentLocked(op) || op.client == nil || op.requestID == "" || op.cancelPending {
		s.mu.Unlock()
		s.dshMu.Unlock()
		return
	}
	op.cancelPending = true
	s.mu.Unlock()
	s.dshMu.Unlock()
	go func() {
		var out api.PluginInstallCancellation
		e := op.client.Call(op.ctx, "pluginManager/cancelInstall", map[string]any{"requestId": op.requestID}, &out)
		if e == nil && out.Status == "cancelled" {
			s.finishMarket(op, marketResult{message: marketMessage("安装已取消。", "Installation cancelled.")}, true)
			op.cancel()
			return
		}
		s.dshMu.Lock()
		defer s.dshMu.Unlock()
		s.mu.Lock()
		if !s.marketCurrentLocked(op) {
			s.mu.Unlock()
			return
		}
		op.cancelPending = false
		if e != nil {
			s.marketInstallMsg = e.Error()
		} else {
			switch out.Status {
			case "cancelled":
				s.marketInstallMsg = marketMessage("已取消，等待安装请求结束并核对库存。", "Cancelled; waiting for installation to settle and checking inventory.")
			case "too-late":
				s.marketInstallMsg = marketMessage("已进入应用阶段，无法取消。", "Already applying; cancellation is too late.")
			case "not-running":
				s.marketInstallMsg = marketMessage("无活动安装请求；结果尚未确认。", "No active installation; result remains unconfirmed.")
			default:
				s.marketInstallMsg = fmt.Sprintf("Unknown cancellation status: %s", out.Status)
			}
		}
		win := s.win
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}()
}
