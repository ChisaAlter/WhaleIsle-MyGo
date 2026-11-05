package main

import (
	"fmt"
	"path/filepath"
	"strings"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	browser "whaleisle/internal/ui/browserpreview"
)

func (s *mainState) declaredDeliverableOwnerCurrent(owner deliverablePreviewOwner, path string, target session.TurnFileTarget) bool {
	if owner.store == nil || owner.client == nil || owner.scope.Binding == "" {
		return false
	}
	s.mu.Lock()
	current := !s.quitting && s.win != nil && s.sessions == owner.store && s.bootStatus.State == harness.Ready && s.bootStatus.Generation == owner.generation && s.bootStatus.Endpoint == owner.endpoint
	s.mu.Unlock()
	s.dshMu.Lock()
	current = current && s.dshClient == owner.client && s.dshEndpoint == owner.endpoint.Origin+"\x00"+owner.endpoint.Cookie
	s.dshMu.Unlock()
	declared, valid := owner.store.TurnFilePreviewPath(owner.scope, target)
	return current && valid && declared == path
}

// previewDeliverableOnHost handles only the routes the confined local reader
// cannot own: remote Hosts, external declared files and original Office viewers.
// The returned bool means this explicit Host-document route consumed the gesture.
func (s *mainState) previewDeliverableOnHost(owner deliverablePreviewOwner, path string) (bool, error) {
	local := s.localDeliverablePreviewScope(owner)
	office := false
	switch strings.ToLower(filepath.Ext(path)) {
	case ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx":
		office = true
	}
	outside := false
	if filepath.IsAbs(path) {
		relative, err := filepath.Rel(owner.cwd, path)
		outside = err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))
	}
	if local && !office && !outside {
		return false, nil
	}
	target, declared := owner.store.DeclaredTurnFileTarget(owner.scope, path)
	valid := func() bool { return declared && s.declaredDeliverableOwnerCurrent(owner, path, target) }
	if !valid() {
		return true, fmt.Errorf("%s", resumeText("无法预览：文件声明或所属 Host/会话已更改", "Cannot preview: the declaration or its Host/session changed"))
	}
	address, err := desktop.DeclaredDocumentURL(owner.endpoint.Origin, owner.scope.SessionID, path)
	if err != nil {
		return true, err
	}
	source := browser.DocumentSource{Authority: fmt.Sprintf("declared-host:%p:%d:%d:%d", owner.client, owner.generation, target.Seq, target.Index), SessionID: owner.scope.SessionID, ViewSessionID: owner.scope.SessionID, Workspace: owner.cwd, Path: path, AddressKind: "declared-delivery", Kind: browser.DocumentHost}
	options := desktop.HostOptions{URL: address, Title: path, Untrusted: true, SessionOrigin: owner.endpoint.Origin, SessionCookie: owner.endpoint.Cookie, PermissionPolicy: desktop.PermissionPolicy{NotificationsAllowed: systemNotificationsSupported}.Decide, ShortcutPreferences: s.pageShortcutPreferences(), OnShortcutCloseWindow: s.queueProductShortcutClose}
	source = browser.BindDeclaredDeliveryDocument(source, valid, options)
	if !s.openWorkspaceDocumentSource(owner.store, source) {
		return true, fmt.Errorf("%s", resumeText("无法预览：交付文件来源已更改", "Cannot preview: the delivery owner changed"))
	}
	return true, nil
}
