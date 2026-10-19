package schedule

import schedstore "whaleisle/internal/store/schedule"

// OpenTask shares the manager editor with transcript cards and session entries.
func (p *Panel) OpenTask(row schedstore.Entry, epoch uint64) {
	p.initialized = true
	p.connectionEpoch = epoch
	p.query, p.statusFilter = "", "all"
	p.selectCatalogTask(row)
}

// Selecting a visible catalog row leaves its search and status context intact.
// External entries use OpenTask to establish a fresh catalog context instead.
func (p *Panel) selectCatalogTask(row schedstore.Entry) {
	if p.drafts == nil {
		p.drafts = map[string]*taskDraft{}
	}
	key := taskKey(row)
	if p.drafts[key] == nil {
		p.drafts[key] = newTaskDraft(row)
	}
	p.selected = key
	p.detailTab = "rule"
}
