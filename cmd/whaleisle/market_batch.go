package main

import (
	"fmt"
	marketstore "whaleisle/internal/store/market"
)

// A batch reuses the serial installer and its Host-provided build approvals.
// Inventory is refreshed after the last outcome so captured version receipts
// remain valid while the confirmed list is processed.
func (s *mainState) runMarketBatchUpdates(targets []marketstore.InstallTarget) {
	if len(targets) == 0 {
		return
	}
	s.dshMu.Lock()
	s.mu.Lock()
	if s.market == nil || s.marketWorker != nil || s.marketInstallBusy || s.launcherImportBusy {
		s.mu.Unlock()
		s.dshMu.Unlock()
		return
	}
	for _, target := range targets {
		if target.UpdateVersion == "" {
			s.marketInstallMsg = marketMessage("请选择已确认版本的更新。", "Select updates with verified versions.")
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
	}
	s.clearMarketBatchLocked()
	receipts := make(map[string]string, len(targets))
	for _, target := range targets {
		receipts[target.ID] = marketInstallRequestID()
	}
	if err := s.market.QueueUpdates(targets, receipts); err != nil {
		s.marketInstallMsg = marketMessage("无法保存批量操作记录；更新未开始。", "Cannot save batch receipts; updates were not started.")
		s.mu.Unlock()
		s.dshMu.Unlock()
		return
	}
	s.marketBatchReceipts = receipts
	s.marketBatch = append([]marketstore.InstallTarget(nil), targets...)
	s.mu.Unlock()
	s.dshMu.Unlock()
	s.runMarketCatalogInstall(targets[0])
}

func (s *mainState) stopMarketBatch() {
	s.mu.Lock()
	count := len(s.marketBatch)
	s.marketInstallMsg = fmt.Sprintf(marketMessage("已取消待执行的批量项目（%d）；当前安装可单独取消。", "Cleared queued batch items (%d); cancel the active install separately."), count)
	s.clearMarketBatchLocked()
	s.mu.Unlock()
}

// Caller holds mainState.mu. Mark only never-started receipts; active outcomes
// are completed by the existing installer/cancellation path.
func (s *mainState) clearMarketBatchLocked() {
	if len(s.marketBatchReceipts) > 0 && s.market != nil {
		if err := s.market.CancelQueuedUpdates(s.marketBatchReceipts); err != nil {
			s.marketInstallMsg += " " + marketMessage("待执行记录保存失败，请查看历史。", "Could not save queued outcomes; check history.")
		}
	}
	s.marketBatch = nil
	s.marketBatchReceipts = nil
}
