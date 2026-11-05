package pluginui

import (
	"context"
	"errors"
	"strconv"
)

// RightPaneKind is a live guide entry from sidebarRightTabs.guide(). ID names
// the provider/entry pair; Kind names the type passed to sidebarRight.openTab.
// Neither is an occurrence handle. Icon is the source icon's declared name or
// its stable provider/entry identity; an absent source icon uses the guide cube.
type RightPaneKind struct {
	ID          string `json:"id"`
	ProviderID  string `json:"providerId"`
	EntryID     string `json:"entryId"`
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Order       int    `json:"order"`
	Available   bool   `json:"available"`
}

type rightPaneOpenResult struct {
	tab RightPaneTab
	err error
}

type rightPaneOpenRequest struct {
	page    *bridgePage
	subject Subject
	kind    string
	result  chan rightPaneOpenResult
}

type rightPaneState struct {
	next    uint64
	pending map[string]*rightPaneOpenRequest
}

func validRightPaneKinds(rows []RightPaneKind) bool {
	if len(rows) > 512 {
		return false
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.ID == "" || row.ProviderID == "" || row.EntryID == "" || row.Kind == "" || seen[row.ID] || len(row.ID) > 32768 || len(row.ProviderID) > 8192 || len(row.EntryID) > 8192 || len(row.Kind) > 8192 || len(row.Label) > 8192 || len(row.Description) > 8192 || len(row.Icon) > 32768 {
			return false
		}
		seen[row.ID] = true
	}
	return true
}

// OpenRightPaneKind delegates one current guide gesture to the public source
// controller. Call outside the UI render goroutine with a bounded context. It
// returns only after the original renderer publishes the actual mounted outlet;
// registration, an inventory record, and a successful send are not a mount.
func (o *Owner) OpenRightPaneKind(ctx context.Context, snapshot Snapshot, kind string) (RightPaneTab, error) {
	if o == nil {
		return RightPaneTab{}, errSessionUnavailable
	}
	if err := ctx.Err(); err != nil {
		return RightPaneTab{}, err
	}
	o.mu.Lock()
	page := o.page
	valid := !o.closed && !o.incompatible && o.state.Connected && o.state.Unavailable == "" && page != nil && page.id == snapshot.Page && o.state.Page == snapshot.Page && o.state.Revision == snapshot.Revision && snapshot.Subject != nil && snapshot.Subject.Kind == "session" && o.state.Subject != nil && o.state.Subject.equal(*snapshot.Subject) && o.selected.equal(*snapshot.Subject)
	registered := false
	for _, row := range o.state.RightPaneKinds {
		if row.Kind == kind && row.Available {
			registered = true
			break
		}
	}
	if !valid || !registered {
		o.mu.Unlock()
		return RightPaneTab{}, errors.New("plugin UI: right pane kind is no longer available")
	}
	o.rightPane.next++
	requestID := strconv.FormatUint(o.rightPane.next, 10)
	request := &rightPaneOpenRequest{page: page, subject: *snapshot.Subject, kind: kind, result: make(chan rightPaneOpenResult, 1)}
	if o.rightPane.pending == nil {
		o.rightPane.pending = make(map[string]*rightPaneOpenRequest)
	}
	o.rightPane.pending[requestID] = request
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		if o.rightPane.pending[requestID] == request {
			delete(o.rightPane.pending, requestID)
		}
		o.mu.Unlock()
	}()
	if err := o.sendToPage(page, map[string]any{"type": "openRightPaneKind", "request": requestID, "revision": snapshot.Revision, "subject": request.subject, "kind": kind}); err != nil {
		return RightPaneTab{}, err
	}
	select {
	case result := <-request.result:
		return result.tab, result.err
	case <-ctx.Done():
		// Cancel the observation, never replay or close an open whose delivery
		// may already have committed in the original source.
		_ = o.sendToPage(page, map[string]any{"type": "cancelRightPaneOpen", "request": requestID, "subject": request.subject})
		return RightPaneTab{}, ctx.Err()
	}
}

func (o *Owner) finishRightPaneOpen(page *bridgePage, requestID, handle, message string, subject *Subject) {
	o.mu.Lock()
	defer o.mu.Unlock()
	request := o.rightPane.pending[requestID]
	if request == nil || request.page != page || o.page != page {
		return
	}
	result := rightPaneOpenResult{err: errSessionUnavailable}
	if subject != nil && subject.equal(request.subject) && o.selected.equal(request.subject) && !o.closed && !o.incompatible {
		if message != "" {
			result.err = errors.New("plugin UI: " + message)
		} else if o.state.Connected && o.state.Unavailable == "" && o.state.Subject != nil && o.state.Subject.equal(request.subject) {
			for _, tab := range o.state.RightPaneTabs {
				if tab.Handle == handle && tab.Kind == request.kind {
					for _, outlet := range o.state.Outlets {
						if outlet.Handle == handle && outlet.Slot == "sidebar.right.pane.tab" && outlet.Available {
							result = rightPaneOpenResult{tab: tab}
							break
						}
					}
					break
				}
			}
		}
	}
	delete(o.rightPane.pending, requestID)
	request.result <- result
}

func (o *Owner) retireRightPaneOpensLocked() {
	for id, request := range o.rightPane.pending {
		delete(o.rightPane.pending, id)
		request.result <- rightPaneOpenResult{err: errSessionUnavailable}
	}
}
