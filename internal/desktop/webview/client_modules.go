package webview

import (
	"errors"
	"strings"

	"whaleisle/internal/desktop/pluginslots"
)

// ClientModules belongs to this WebView's isolated Host session, never to an
// unrelated browser tab or the native Host inventory controller.
func (h *Host) ClientModules() pluginslots.ClientModulesSnapshot {
	if h == nil || !h.Alive() {
		return pluginslots.ClientModulesSnapshot{}
	}
	return h.gateway.ClientModules()
}

func (h *Host) RetryClientModules(page uint64) error {
	if h == nil || !h.Alive() {
		return errors.New("client modules: page closed")
	}
	return h.gateway.RetryClientModules(page)
}

func (h *Host) disconnectClientModulesForNavigation(raw string) {
	if h.gateway == nil {
		return
	}
	current, next := h.URL(), h.logicalURL(raw)
	// A fragment-only route keeps the same document and module owner alive.
	if current != next && strings.SplitN(current, "#", 2)[0] == strings.SplitN(next, "#", 2)[0] {
		return
	}
	h.gateway.DisconnectClientModules()
}

// DocumentViewerRecommended is true only while this exact live page reports a
// matching external renderer or mounted builtin workbook owner from its registry.
func (h *Host) DocumentViewerRecommended() bool {
	state := h.ClientModules()
	return state.Document != nil && (state.Document.Extension || state.Document.Spreadsheet)
}

// WorkbookViewerAvailable is a presentation capability of this exact resource
// owner. It grants neither native bindings nor access to another Host/session.
func (h *Host) WorkbookViewerAvailable() bool {
	state := h.ClientModules()
	return state.Document != nil && state.Document.Spreadsheet
}
