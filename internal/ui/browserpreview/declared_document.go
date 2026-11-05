package browserpreview

import (
	"errors"
	"sync/atomic"
	desktop "whaleisle/internal/desktop/webview"
)

type declaredDocumentPermit struct {
	source  DocumentSource
	valid   func() bool
	options desktop.HostOptions
	revoked atomic.Bool
}

// BindDeclaredDeliveryDocument is an in-memory grant from the native delivery
// consumer. A restored descriptor cannot create a Host or borrow another grant.
func BindDeclaredDeliveryDocument(source DocumentSource, valid func() bool, options desktop.HostOptions) DocumentSource {
	source.DeclaredDelivery = true
	source.declaredDelivery = nil
	source.declaredDelivery = &declaredDocumentPermit{source: source, valid: valid, options: options}
	return source
}
func (source DocumentSource) DeclaredDeliveryValid() bool {
	p := source.declaredDelivery
	if !source.DeclaredDelivery || p == nil || p.revoked.Load() {
		return false
	}
	descriptor := source
	descriptor.declaredDelivery = nil
	if descriptor != p.source || p.valid == nil || !p.valid() {
		p.revoked.Store(true)
		return false
	}
	return !p.revoked.Load()
}
func (source DocumentSource) DeclaredDeliveryResult() DocumentResult {
	result := DocumentResult{Valid: source.DeclaredDeliveryValid}
	if !source.DeclaredDeliveryValid() {
		result.Err = errors.New(browserChromeLabel("交付文件预览权限已失效；请从原交付卡重新打开。", "Delivery preview permission ended; reopen it from the original card."))
		return result
	}
	options := source.declaredDelivery.options
	result.Viewer = &options
	return result
}
func (b *Controller) denyDeclaredDocument(tab *Tab) {
	b.clearSourceDocument(tab)
	tab.sourcePending = false
	tab.ActionMessage = ""
	tab.Error = browserChromeLabel("交付文件预览权限已失效；请从当前 Host 与会话的原交付卡重新打开。", "Delivery preview permission ended; reopen the original card in its Host and session.")
}
