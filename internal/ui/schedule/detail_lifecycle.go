package schedule

import (
	"context"

	schedstore "whaleisle/internal/store/schedule"
)

type deleteReceipt struct {
	key      string
	revision uint64
}

func scheduleEntryAvailable(snap schedstore.ViewState, row schedstore.Entry) bool {
	for _, entry := range snap.Rows {
		if taskKey(entry) == taskKey(row) {
			return true
		}
	}
	return false
}

// Deactivate mirrors unmounting the task detail. Call on an actual visibility
// transition, not on every paint; a current failed draft remains until then.
func (p *Panel) Deactivate() {
	if !p.initialized {
		return
	}
	p.closeDetail()
	p.initialized = false
}

func (p *Panel) closeDetail() {
	p.mu.Lock()
	p.detailOwner++
	p.actionRequest++
	p.actionPending, p.actionFailed = false, false
	p.deleteAck = nil
	p.committed = nil
	p.status, p.statusOwner = "", ""
	p.historyRequest++
	p.historyKey, p.historyLoading = "", false
	p.historyLatest = ""
	p.historyState = deliveryHistoryState{}
	p.mu.Unlock()
	p.selected, p.detailTab = "", "rule"
	p.drafts, p.deleting, p.deliveryExpanded = nil, nil, nil
}

func (p *Panel) actionCurrentLocked(owner, request, epoch uint64) bool {
	return p.detailOwner == owner && p.actionRequest == request && p.connectionEpoch == epoch
}

// Only a ready all-session catalog can retire a missing detail. A failed edit
// remains visible with its draft; an acknowledged delete waits for a newer read.
// The return value closes a task tab only for its own confirmed deletion. An
// externally missing task leaves that tab showing the missing-task state.
func (p *Panel) reconcileDetail(snap schedstore.ViewState) bool {
	if p.selected == "" || !snap.CatalogReady {
		return false
	}
	for _, row := range snap.Rows {
		if taskKey(row) == p.selected {
			p.mu.Lock()
			if p.deleteAck != nil && snap.CatalogRevision > p.deleteAck.revision {
				p.deleteAck = nil
			}
			p.mu.Unlock()
			return false
		}
	}
	p.mu.Lock()
	retain := p.actionPending || p.actionFailed
	confirmed := false
	if p.deleteAck != nil && p.deleteAck.key == p.selected {
		retain = snap.CatalogRevision <= p.deleteAck.revision
		confirmed = !retain
	}
	p.mu.Unlock()
	if retain {
		return false
	}
	p.closeDetail()
	return confirmed
}

func (p *Panel) deleteTask(m Model, row schedstore.Entry) {
	snap := m.Store.Snapshot()
	p.mu.Lock()
	p.actionRequest++
	owner, request, epoch := p.detailOwner, p.actionRequest, p.connectionEpoch
	p.actionPending, p.actionFailed = true, false
	p.status, p.statusOwner = "", taskKey(row)
	p.mu.Unlock()
	go func() {
		err := m.Store.DeleteEntry(context.Background(), row)
		current := m.Store.Snapshot()
		p.mu.Lock()
		if !p.actionCurrentLocked(owner, request, epoch) || current.ConnectionEpoch != epoch {
			p.mu.Unlock()
			return
		}
		p.actionPending, p.actionFailed = false, err != nil
		if err != nil {
			p.status, p.statusOwner = err.Error(), taskKey(row)
		} else {
			p.deleteAck = &deleteReceipt{key: taskKey(row), revision: snap.CatalogRevision}
		}
		p.mu.Unlock()
		if m.OnInvalidate != nil {
			m.OnInvalidate()
		}
	}()
}

func (p *Panel) selectDetailTab(tab string) {
	if p.detailTab == tab {
		return
	}
	if p.detailTab == "records" {
		p.mu.Lock()
		p.historyRequest++
		p.historyKey, p.historyLoading = "", false
		p.historyLatest = ""
		p.historyState = deliveryHistoryState{}
		p.mu.Unlock()
	}
	p.detailTab = tab
}
