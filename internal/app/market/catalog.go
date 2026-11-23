package marketctl

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/dsh/api"
	marketstore "whaleisle/internal/store/market"
)

// InstallReadyFor queries whether Node/marketplace installation prerequisites
// are available, caching the result for 1s off the UI thread.
func (c *Coordinator) InstallReadyFor(nodeBin string) bool {
	return c.installReady.get(nodeBin, time.Second, func() bool {
		return desktopmarket.InstallReady(nodeBin)
	}, c.invalidateUI)
}

// InspectInstalledPlugin validates/inspects a manually entered plugin spec on the Host.
func (c *Coordinator) InspectInstalledPlugin(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	op, _ := c.BeginAttempt("", nil, false)
	if op == nil {
		return
	}
	op.Method = "pluginManager/inspect"
	go func() {
		defer c.releaseAttempt(op)
		var out api.PluginSpecInspection
		err := op.Client.Call(op.Ctx, op.Method, map[string]any{"spec": name}, &out)
		message := name
		if err != nil {
			message = err.Error()
		} else if len(out) > 0 {
			message = string(out)
		}
		c.finishAttempt(op, marketResult{message: message}, false)
	}()
}

// Invalidate cancels active marketplace operations and marks in-flight receipts interrupted.
func (c *Coordinator) Invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.unlockPublish()
	if c.active != nil {
		c.active.Cancel()
		if c.active.Store != nil && c.active.ReceiptID != "" {
			if err := c.active.Store.FinishInstall(c.active.ReceiptID, "interrupted", "result-unconfirmed"); err != nil {
				c.installMsg = marketMessage("中断记录保存失败，请查看历史。", "Could not save interrupted result; check history.")
			}
		}
	}
	c.clearBatchLocked()
	c.active = nil
	c.installBusy = false
	c.installRequestID = ""
	c.approval = nil
	c.catalogTarget = nil
	c.needsAllowBuilds = false
	c.allowBuilds = nil
	c.incompatible = nil
}

// RunCatalogInstall begins installing or updating an approved catalog target.
func (c *Coordinator) RunCatalogInstall(target marketstore.InstallTarget) {
	env := c.hostEnv()
	c.mu.Lock()
	if c.store == nil || c.worker != nil || c.installBusy || env.LauncherImportBusy {
		switch {
		case c.store == nil:
			c.installMsg = marketMessage("运行时未连接，无法安装。", "The runtime is not connected; installation is unavailable.")
		case env.LauncherImportBusy:
			c.installMsg = marketMessage("安装器正忙，请稍后重试。", "The installer is busy; try again in a moment.")
		default:
			c.installMsg = marketMessage("正在刷新插件目录，请稍后重试。", "The catalog is refreshing; try again in a moment.")
		}
		c.unlockPublish()
		return
	}
	if err := c.store.ValidateTarget(target); err != nil {
		c.installMsg = err.Error()
		c.clearBatchLocked()
		c.unlockPublish()
		return
	}
	c.catalogTarget = &target
	c.installSpec = target.Spec
	owner := Owner{Client: c.client, Store: c.store, Generation: c.store.RuntimeToken()}
	c.unlockPublish()
	c.RunInstallForOwner(target.Spec, nil, &owner)
}

// BeginAttempt validates prerequisites and reserves the worker slot without owner pinning.
func (c *Coordinator) BeginAttempt(spec string, allow []string, install bool, approval ...*Attempt) (*Attempt, string) {
	return c.BeginAttemptForOwner(spec, allow, install, nil, approval...)
}

// BeginAttemptForOwner validates Host/store ownership and reserves the single
// active market operation slot.
func (c *Coordinator) BeginAttemptForOwner(spec string, allow []string, install bool, expected *Owner, approval ...*Attempt) (*Attempt, string) {
	env := c.hostEnv()
	if env.LauncherImportBusy {
		return nil, ""
	}
	c.mu.Lock()
	defer c.unlockPublish()

	if expected != nil && (c.client != expected.Client || c.store != expected.Store || (expected.Store != nil && expected.Store.RuntimeToken() != expected.Generation)) {
		return nil, ""
	}
	if c.worker != nil || c.installBusy {
		return nil, ""
	}
	if len(approval) > 0 {
		a := approval[0]
		if a == nil || c.approval != a || a.Client != c.client || a.Store != c.store || (a.Store != nil && a.Store.RuntimeToken() != a.Generation) {
			return nil, ""
		}
		if a.Target != nil && (a.Store == nil || a.Store.ValidateTarget(*a.Target) != nil) {
			return nil, ""
		}
	}
	if !install && c.client == nil {
		return nil, ""
	}
	if install && len(allow) > 0 {
		a := c.approval
		if a == nil || a.Spec != spec || a.Client != c.client || a.Store != c.store || (a.Store != nil && a.Store.RuntimeToken() != a.Generation) || !slices.Equal(allow, c.allowBuilds) {
			c.installMsg = marketMessage("构建授权已过期，请重新安装以确认。", "Build approval expired; start installation again.")
			return nil, ""
		}
	}
	var target *marketstore.InstallTarget
	if !install && len(approval) > 0 && approval[0] != nil {
		target = approval[0].Target
	}
	if install && c.catalogTarget != nil && c.catalogTarget.Spec == spec {
		t := *c.catalogTarget
		if c.store == nil {
			return nil, ""
		}
		if e := c.store.ValidateTarget(t); e != nil {
			c.installMsg = e.Error()
			return nil, ""
		}
		target = &t
	} else if install {
		c.catalogTarget = nil
	}
	op := &Attempt{Client: c.client, Store: c.store, Spec: spec, Target: target}
	if op.Store != nil {
		op.Generation = op.Store.RuntimeToken()
	}
	op.Ctx, op.Cancel = c.childContext()
	if install && op.Client != nil {
		op.RequestID = marketInstallRequestID()
	}
	if install {
		id, action := "manual-install", "install"
		if target != nil {
			id = target.ID
			if target.UpdateVersion != "" {
				action = "update"
			}
		}
		op.ReceiptID = marketInstallRequestID()
		if target != nil && len(c.batch) > 0 && c.batch[0] == *target {
			if receipt := c.batchReceipts[target.ID]; receipt != "" {
				op.ReceiptID = receipt
			}
		}
		if op.Store == nil {
			op.Cancel()
			c.installMsg = marketMessage("无法保存操作记录；安装未开始。", "Cannot save operation receipt; installation was not started.")
			return nil, ""
		}
		if err := op.Store.BeginInstall(op.ReceiptID, id, action); err != nil {
			op.Cancel()
			c.installMsg = marketMessage("无法保存操作记录；安装未开始。", "Cannot save operation receipt; installation was not started.")
			return nil, ""
		}
		if len(c.batch) > 0 && (target == nil || c.batch[0] != *target) {
			c.clearBatchLocked()
		}
	}
	c.worker = op
	c.active = op
	c.installBusy = true
	c.installRequestID = op.RequestID
	c.installMsg = ""
	c.installLog = ""
	if install {
		c.installSpec = spec
		c.needsAllowBuilds = false
		c.allowBuilds = nil
		c.approval = nil
		c.incompatible = nil
	}
	return op, env.NodeBin
}

func (c *Coordinator) childContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(c.lifecycleContext())
}

type marketResult struct {
	message, log          string
	outcome, historyError string
	accepted              bool
	pending               []string
	incompatible          []api.ChangeResultErrorIncompatibleItem
}

func marketResultFor(out *api.ChangeResult, err error) marketResult {
	r := marketResult{outcome: "unknown", historyError: "result-unconfirmed"}
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
		r.outcome, r.historyError = "failed", "operation-failed"
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
			r.outcome, r.historyError = "applied", ""
			r.accepted = true
			r.message = marketMessage("已应用；正在核对库存。", "Applied; checking inventory.")
		case "restart-required":
			r.outcome, r.historyError = "restart-required", ""
			r.accepted = true
			r.message = marketMessage("已保存，需重启运行时后生效。", "Saved; a runtime restart is required to take effect.")
		case "overridden":
			r.outcome, r.historyError = "overridden", ""
			r.message = marketMessage("此变更被更高优先级配置覆盖；请检查插件配置。", "This change is overridden by higher-priority configuration; check plugin settings.")
		case "cancelled":
			r.outcome, r.historyError = "cancelled", ""
			r.message = marketMessage("安装已取消。", "Installation cancelled.")
		case "failed":
			r.outcome, r.historyError = "failed", "operation-failed"
			r.message = marketMessage("操作失败。", "Operation failed.")
		default:
			r.message = marketMessage("结果未确认，请刷新库存。", "Result unconfirmed; refresh inventory.")
		}
	}
	if len(r.pending) > 0 {
		r.outcome = ""
		r.accepted = false
		r.message += " " + marketMessage("需要明确授权以下构建脚本。", "Explicit build-script approval is required.")
	}
	return r
}

func (c *Coordinator) finishAttempt(op *Attempt, r marketResult, install bool) {
	env := c.hostEnv()
	c.mu.Lock()
	if !c.currentAttemptLocked(op) {
		c.unlockPublish()
		return
	}
	c.active = nil
	c.installBusy = false
	c.installRequestID = ""
	c.installMsg = r.message
	c.installLog = r.log
	if op.Store != nil && op.ReceiptID != "" {
		outcome := r.outcome
		if len(r.pending) > 0 || len(r.incompatible) > 0 {
			outcome = "awaiting-approval"
		}
		if err := op.Store.FinishInstall(op.ReceiptID, outcome, r.historyError, r.log, r.message); err != nil {
			c.installMsg += " " + marketMessage("结果保存失败；历史仅在本次会话可见。", "Result could not be saved; history is available only in this session.")
		}
	}
	if install && len(c.batch) > 0 && op.Target != nil && c.batch[0] == *op.Target {
		if r.accepted {
			c.batch = c.batch[1:]
			op.ContinueBatch = len(c.batch) > 0
			if !op.ContinueBatch {
				c.batchReceipts = nil
			}
		} else if len(r.pending) == 0 && len(r.incompatible) == 0 {
			c.installMsg += fmt.Sprintf(marketMessage(" 批量更新已停止，剩余 %d 项未执行。", " Batch stopped; %d remaining items were not run."), len(c.batch)-1)
			c.clearBatchLocked()
		} else {
			c.installMsg += marketMessage(" 批量更新暂停，等待此插件的授权。", " Batch paused for this plugin's approval.")
		}
	}
	if r.accepted && r.outcome == "restart-required" {
		c.activation.Require(env.BootGeneration)
	}
	if install {
		c.allowBuilds = r.pending
		c.needsAllowBuilds = len(r.pending) > 0
		c.incompatible = r.incompatible
		if len(r.pending) > 0 || len(r.incompatible) > 0 {
			c.approval = op
		} else {
			c.approval = nil
		}
	}
	if op.Method == "pluginManager/setVersionExemption" && r.accepted {
		c.incompatible = nil
	}
	c.unlockPublish()

	awaitingUpdateApproval := op.Target != nil && op.Target.UpdateVersion != "" &&
		(len(r.pending) > 0 || len(r.incompatible) > 0 || (op.Method == "pluginManager/setVersionExemption" && r.accepted))
	c.refreshChangedInventory()
	if op.Store != nil && !awaitingUpdateApproval && !op.ContinueBatch {
		op.Store.RefreshShared(c.lifecycleContext())
	}
	c.updateUI()
}

func (c *Coordinator) releaseAttempt(op *Attempt) {
	op.Cancel()
	c.mu.Lock()
	var next *marketstore.InstallTarget
	if c.worker == op {
		c.worker = nil
		if op.ContinueBatch && len(c.batch) > 0 {
			t := c.batch[0]
			next = &t
		}
	}
	c.unlockPublish()
	if next != nil {
		c.RunCatalogInstall(*next)
	}
	c.updateUI()
}

// RunInstall initiates a GitHub or npm package install without pinning a view owner.
func (c *Coordinator) RunInstall(spec string, allowBuilds []string) {
	c.RunInstallForOwner(spec, allowBuilds, nil)
}

// RunInstallForOwner executes a marketplace install bound to the captured Host/store owner.
func (c *Coordinator) RunInstallForOwner(spec string, allowBuilds []string, expected *Owner) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return
	}
	allowBuilds = append([]string(nil), allowBuilds...)
	op, node := c.BeginAttemptForOwner(spec, allowBuilds, true, expected)
	if op == nil {
		return
	}
	go func() {
		defer c.releaseAttempt(op)
		var r marketResult
		if op.Client != nil {
			opts := map[string]any{"requestId": op.RequestID}
			if len(allowBuilds) > 0 {
				opts["approvedBuilds"] = allowBuilds
			}
			var out *api.ChangeResult
			err := op.Client.Call(op.Ctx, "pluginManager/installBundle", map[string]any{"spec": spec, "options": opts}, &out)
			pending := false
			if out != nil {
				switch strings.ToLower(strings.TrimSpace(out.Stage)) {
				case "pending", "running", "installing", "queued":
					pending = true
				}
			}
			if (err != nil || pending) && op.Ctx.Err() == nil {
				var waited *api.ChangeResult
				if e := op.Client.Call(op.Ctx, "pluginManager/waitForInstall", map[string]any{"requestId": op.RequestID}, &waited); e == nil && waited != nil {
					out, err = waited, nil
				} else if e != nil {
					err = e
				}
			}
			r = marketResultFor(out, err)
		} else {
			res := desktopmarket.InstallGithubSpec(spec, allowBuilds, desktopmarket.InstallOptions{NodeBin: node})
			r = marketResult{message: res.Error, log: res.Log, outcome: "failed", historyError: "operation-failed"}
			if res.NeedsAllowBuilds {
				r.pending = append([]string(nil), res.AllowBuilds...)
				r.outcome = ""
			}
			if res.OK && !res.NeedsAllowBuilds {
				r.outcome, r.historyError = "restart-required", ""
				r.accepted = true
				r.message = marketMessage("已保存，需重启运行时后生效。", "Saved; a runtime restart is required.")
			}
			if r.message == "" {
				r.message = marketMessage("结果未确认。", "Result unconfirmed.")
			}
		}
		c.finishAttempt(op, r, true)
	}()
}

// RunChange executes a bundle configuration change RPC without owner pinning.
func (c *Coordinator) RunChange(method string, args map[string]any) {
	c.RunChangeApproved(method, args)
}

// RunChangeApproved runs a bundle change using an optional captured approval attempt.
func (c *Coordinator) RunChangeApproved(method string, args map[string]any, approval ...*Attempt) {
	c.RunChangeForOwner(method, args, nil, approval...)
}

// RunChangeForOwner executes a toggle, remove, or version-exemption change for expected owner.
func (c *Coordinator) RunChangeForOwner(method string, args map[string]any, expected *Owner, approval ...*Attempt) {
	op, _ := c.BeginAttemptForOwner("", nil, false, expected, approval...)
	if op == nil {
		return
	}
	op.Method = method
	if method == "pluginManager/setBundleEnabled" || method == "pluginManager/removeBundle" {
		name, _ := args["name"].(string)
		if op.Store == nil {
			c.finishAttempt(op, marketResult{message: marketMessage("插件库存不可用，请刷新。", "Plugin inventory unavailable; refresh first.")}, false)
			c.releaseAttempt(op)
			return
		}
		if err := op.Store.ValidateBundleChange(name, method == "pluginManager/removeBundle"); err != nil {
			c.finishAttempt(op, marketResult{message: err.Error()}, false)
			c.releaseAttempt(op)
			return
		}
	}
	id, action := "manual-install", "exemption"
	if op.Target != nil {
		id = op.Target.ID
	}
	if name, ok := args["name"].(string); ok && op.Store != nil {
		id = op.Store.BundleHistoryID(name)
	}
	if method == "pluginManager/removeBundle" {
		action = "remove"
	}
	if method == "pluginManager/setBundleEnabled" {
		action = "toggle"
	}
	op.ReceiptID = marketInstallRequestID()
	if op.Store == nil {
		c.finishAttempt(op, marketResult{message: marketMessage("无法保存操作记录；变更未开始。", "Cannot save operation receipt; change was not started.")}, false)
		c.releaseAttempt(op)
		return
	}
	if err := op.Store.BeginInstall(op.ReceiptID, id, action); err != nil {
		op.ReceiptID = ""
		c.finishAttempt(op, marketResult{message: marketMessage("无法保存操作记录；变更未开始。", "Cannot save operation receipt; change was not started.")}, false)
		c.releaseAttempt(op)
		return
	}
	go func() {
		defer c.releaseAttempt(op)
		var out *api.ChangeResult
		e := op.Client.Call(op.Ctx, method, args, &out)
		c.finishAttempt(op, marketResultFor(out, e), false)
	}()
}

// AcceptExemption approves an incompatible version exemption for an awaiting install.
func (c *Coordinator) AcceptExemption(packageVersion, runtimeVersion string) {
	c.mu.Lock()
	a := c.approval
	valid := a != nil && a.Client == c.client && a.Store == c.store && (a.Store == nil || a.Store.RuntimeToken() == a.Generation)
	match := false
	for _, it := range c.incompatible {
		if it.Name+"@"+it.Version == packageVersion && it.RuntimeVersion == runtimeVersion {
			match = true
		}
	}
	if valid && a.Target != nil {
		valid = c.store != nil && c.store.ValidateTarget(*a.Target) == nil
	}
	c.unlockPublish()
	if !valid || !match {
		return
	}
	c.RunChangeApproved("pluginManager/setVersionExemption", map[string]any{
		"packageVersion": packageVersion,
		"runtimeVersion": runtimeVersion,
		"enabled":        true,
		"acceptRisk":     true,
	}, a)
}

// CancelInstall requests cancellation of an active Host install job.
func (c *Coordinator) CancelInstall() {
	c.mu.Lock()
	op := c.active
	if !c.currentAttemptLocked(op) || op.Client == nil || op.RequestID == "" || op.CancelPending {
		c.unlockPublish()
		return
	}
	op.CancelPending = true
	c.unlockPublish()

	go func() {
		var out api.PluginInstallCancellation
		e := op.Client.Call(op.Ctx, "pluginManager/cancelInstall", map[string]any{"requestId": op.RequestID}, &out)
		if e == nil && out.Status == "cancelled" {
			c.finishAttempt(op, marketResult{message: marketMessage("安装已取消。", "Installation cancelled."), outcome: "cancelled"}, true)
			op.Cancel()
			return
		}
		c.mu.Lock()
		if !c.currentAttemptLocked(op) {
			c.unlockPublish()
			return
		}
		op.CancelPending = false
		if e != nil {
			c.installMsg = e.Error()
		} else {
			switch out.Status {
			case "cancelled":
				c.installMsg = marketMessage("已取消，等待安装请求结束并核对库存。", "Cancelled; waiting for installation to settle and checking inventory.")
			case "too-late":
				c.installMsg = marketMessage("已进入应用阶段，无法取消。", "Already applying; cancellation is too late.")
			case "not-running":
				c.installMsg = marketMessage("无活动安装请求；结果尚未确认。", "No active installation; result remains unconfirmed.")
			default:
				c.installMsg = fmt.Sprintf("Unknown cancellation status: %s", out.Status)
			}
		}
		c.unlockPublish()
		c.updateUI()
	}()
}
