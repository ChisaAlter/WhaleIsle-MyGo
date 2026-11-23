package marketctl

import (
	"context"
	"fmt"

	"whaleisle/internal/harness"
	marketstore "whaleisle/internal/store/market"
)

// RestartPlugins restarts the DSH runtime when a plugin change requires restart.
func (c *Coordinator) RestartPlugins() {
	env := c.hostEnv()
	c.mu.Lock()
	allowed := c.activation.Pending && c.worker == nil && !c.installBusy && !env.TaskRestartPending && !env.Quitting
	c.unlockPublish()
	if !allowed || !env.HasSupervisor || c.deps == nil {
		return
	}
	c.deps.SyncHarnessAndRestart()
	c.updateUI()
}

// ObserveActivation updates pending-restart status when a Host status arrives.
// actualStatus passes supervisor.Status() captured cleanly without nesting locks.
func (c *Coordinator) ObserveActivation(st harness.Status, actualStatus *harness.Status) {
	c.mu.Lock()
	if actualStatus != nil {
		if actualStatus.Generation != st.Generation || actualStatus.State != st.State {
			c.unlockPublish()
			return
		}
	}
	completed := c.activation.Observe(st.Generation, st.State.String())
	store := c.store
	c.unlockPublish()
	if completed && store != nil {
		c.refreshChangedInventory()
		store.RefreshShared(context.Background())
	}
}

// RunBatchUpdates reserves and starts a serial update batch for confirmed targets.
func (c *Coordinator) RunBatchUpdates(targets []marketstore.InstallTarget) {
	if len(targets) == 0 {
		return
	}
	env := c.hostEnv()
	if env.LauncherImportBusy {
		return
	}
	c.mu.Lock()
	if c.store == nil || c.worker != nil || c.installBusy {
		c.unlockPublish()
		return
	}
	for _, target := range targets {
		if target.UpdateVersion == "" {
			c.installMsg = marketMessage("请选择已确认版本的更新。", "Select updates with verified versions.")
			c.unlockPublish()
			return
		}
		if err := c.store.ValidateTarget(target); err != nil {
			c.installMsg = err.Error()
			c.unlockPublish()
			return
		}
	}
	c.clearBatchLocked()
	receipts := make(map[string]string, len(targets))
	for _, target := range targets {
		receipts[target.ID] = marketInstallRequestID()
	}
	if err := c.store.QueueUpdates(targets, receipts); err != nil {
		c.installMsg = marketMessage("无法保存批量操作记录；更新未开始。", "Cannot save batch receipts; updates were not started.")
		c.unlockPublish()
		return
	}
	c.batchReceipts = receipts
	c.batch = append([]marketstore.InstallTarget(nil), targets...)
	c.unlockPublish()
	c.RunCatalogInstall(targets[0])
}

// StopBatch clears remaining queued batch updates while letting any running item settle.
func (c *Coordinator) StopBatch() {
	c.mu.Lock()
	count := len(c.batch)
	c.installMsg = fmt.Sprintf(marketMessage("已取消待执行的批量项目（%d）；当前安装可单独取消。", "Cleared queued batch items (%d); cancel the active install separately."), count)
	c.clearBatchLocked()
	c.unlockPublish()
}

// Caller holds c.mu.
func (c *Coordinator) clearBatchLocked() {
	if len(c.batchReceipts) > 0 && c.store != nil {
		if err := c.store.CancelQueuedUpdates(c.batchReceipts); err != nil {
			c.installMsg += " " + marketMessage("待执行记录保存失败，请查看历史。", "Could not save queued outcomes; check history.")
		}
	}
	c.batch = nil
	c.batchReceipts = nil
}
