package main

import (
	"context"
	"fmt"
	"time"

	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/store/session"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
)

// Explicit opens and restored tabs share the same saved-file reading pipeline.
// The descriptor identifies the original source, not its temporary preview URL.
func (s *mainState) openWorkspaceDocument(sessions *session.Store, sid, root, path, kind string, floating ...bool) {
	if sessions == nil || sid == "" {
		return
	}
	s.mu.Lock()
	win, current := s.win, s.sessions
	s.mu.Unlock()
	if win == nil || current != sessions {
		return
	}
	source := browserpreviewui.DocumentSource{
		Authority: s.browserWorkspaceOwner().Authority,
		SessionID: sid, ViewSessionID: sid, AddressKind: "session", Workspace: root, Path: path, Kind: kind,
	}
	s.openWorkspaceDocumentSource(sessions, source, floating...)
}

func (s *mainState) openWorkspaceDocumentSource(sessions *session.Store, source browserpreviewui.DocumentSource, floating ...bool) bool {
	if sessions == nil || !s.workspaceDocumentReady(source) {
		return false
	}
	s.mu.Lock()
	win := s.win
	if s.sessions != sessions.AgentNavigationRootStore() || win == nil || s.quitting {
		s.mu.Unlock()
		return false
	}
	if s.layout.Workspace == nil {
		s.layout.Workspace = &frame.WorkspacePanes{}
	}
	panes := s.layout.Workspace
	s.filesPaneOpen = true
	s.rightPaneTab = "browser"
	s.browserPreviewOpen, s.webviewOpen = false, false
	s.mu.Unlock()
	s.closeWebViewHost()
	panes.OpenDocument(source, func() *browserpreviewui.Controller { return s.newBrowserController(win) }, floating...)
	win.Update(func() {})
	return true
}

func (s *mainState) workspaceDocumentReady(source browserpreviewui.DocumentSource) bool {
	if source.LocalDelivery || source.AddressKind == localDeliveryAddress {
		return source.LocalDeliveryValid()
	}
	s.mu.Lock()
	sessions, win, quitting := s.sessions, s.win, s.quitting
	s.mu.Unlock()
	if quitting || sessions == nil || win == nil {
		return false
	}
	snap := sessions.Snapshot()
	viewID := source.ViewSessionID
	if viewID == "" {
		viewID = source.SessionID
	}
	if source.Authority == "" || source.Authority != s.browserWorkspaceOwner().Authority ||
		viewID == "" || viewID != snap.SelectedID || source.Workspace != snap.SelectedCwd {
		return false
	}
	if source.AddressKind == "subagent" {
		sessions = s.workspaceDocumentStore(source)
		if sessions == nil {
			return false
		}
		view := sessions.AgentNavigationSnapshot()
		_, _, err := sessions.AgentMediaPreviewSource(view.Token, workspaceDocumentAddress(source))
		return err == nil
	}
	return source.SessionID == viewID
}

func workspaceDocumentAddress(source browserpreviewui.DocumentSource) session.AgentSessionAddress {
	return session.AgentSessionAddress{Kind: source.AddressKind, ParentSessionID: source.ParentSessionID,
		ChildSessionID: source.SessionID, Mode: source.Mode}
}

// The selected view owns the occurrence; its source can be an explicitly bound
// child address. Recovery never selects or follows a child, swaps a Host client,
// or treats the persisted descriptor as a live permission token.
func (s *mainState) readWorkspaceDocument(ctx context.Context, source browserpreviewui.DocumentSource) browserpreviewui.DocumentResult {
	if source.LocalDelivery || source.AddressKind == localDeliveryAddress {
		return s.readLocalDeliverableDocument(ctx, source)
	}
	s.mu.Lock()
	win, host, sessions := s.win, s.bootStatus, s.sessions
	s.mu.Unlock()
	result := browserpreviewui.DocumentResult{}
	if sessions == nil || !s.workspaceDocumentReady(source) {
		result.Err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPreviewSwitchSessionWorkspace))
		return result
	}
	sid, root, path := source.SessionID, source.Workspace, source.Path
	var reader mediapreview.Caller
	var sourceValid func() bool
	var sourceErr error
	var childToken uint64
	child := source.AddressKind == "subagent"
	address := workspaceDocumentAddress(source)
	if child {
		sessions = s.workspaceDocumentStore(source)
		if sessions == nil {
			result.Err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPreviewSourceClosed))
			return result
		}
		view := sessions.AgentNavigationSnapshot()
		childToken = view.Token
		reader, sourceValid, sourceErr = sessions.AgentMediaPreviewSource(childToken, address)
		if sourceErr != nil {
			result.Err = sourceErr
			return result
		}
	} else {
		reader, sourceValid, sourceErr = sessions.MediaPreviewSource(sid)
	}
	viewID := source.ViewSessionID
	if viewID == "" {
		viewID = sid
	}
	valid := func() bool {
		s.mu.Lock()
		st := s.bootStatus
		current := !s.quitting && s.sessions == sessions.AgentNavigationRootStore() && s.win == win &&
			st.Generation == host.Generation && st.State == host.State &&
			st.Endpoint.Origin == host.Endpoint.Origin && st.Endpoint.Cookie == host.Endpoint.Cookie
		s.mu.Unlock()
		snap := sessions.Snapshot()
		return current && (sourceErr != nil || sourceValid()) &&
			snap.SelectedID == viewID && snap.SelectedCwd == root &&
			source.Authority == s.browserWorkspaceOwner().Authority
	}
	result.Valid = valid
	if !valid() {
		result.Err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPreviewSessionChanged))
		return result
	}

	if child {
		sessions.SetAgentResourceStatus(childToken, address, path, true, nil)
		result.Canceled = func() {
			if valid() {
				sessions.SetAgentResourceStatus(childToken, address, path, false, context.Canceled)
			}
		}
	}
	var office *session.OfficePDFDocument
	switch source.Kind {
	case browserpreviewui.DocumentOfficePDF:
		if child {
			result.Err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPreviewChildUnsupported))
			break
		}
		s.setOfficePreviewMessage(sessions, sid, path, i18n.T(i18n.KeyCmdOfficePreparing))
		// The Host supplies complete multipart PDF bytes. Its conversion path and
		// the new PDF capability remain private to this live occurrence.
		office, result.Err = sessions.RenderOfficePDFBytesForSession(ctx, sid, path)
		if result.Err == nil && valid() {
			result.PDF, result.Err = pdfpreview.OpenBytes(officePreviewName(path), office.Data, valid)
		} else if result.Err == nil {
			result.Err = fmt.Errorf("Office conversion session changed")
		}
	case browserpreviewui.DocumentPDF:
		sessions.SetLastAction(i18n.T(i18n.KeyCmdPDFPreparing))
		if sourceErr == nil {
			readCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			var data []byte
			data, result.Err = mediapreview.ReadSnapshot(readCtx, reader, sid, path, pdfpreview.MaxBytes, valid)
			cancel()
			if result.Err == nil {
				result.PDF, result.Err = pdfpreview.OpenBytes(path, data, valid)
			}
		} else {
			result.PDF, result.Err = pdfpreview.OpenWorkspace(root, path, valid)
		}
	case browserpreviewui.DocumentImage:
		if sourceErr == nil {
			readCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			data, err := mediapreview.ReadSnapshot(readCtx, reader, sid, path, int(imagepreview.MaxBytes), valid)
			cancel()
			result.Err = err
			if err == nil {
				result.HTML, result.Err = htmldocument.OpenImageBytes(path, data, valid)
			}
		} else {
			result.HTML, result.Err = htmldocument.OpenImageWorkspace(root, path, valid)
		}
	case browserpreviewui.DocumentHTML:
		sessions.SetLastAction(i18n.T(i18n.KeyCmdHTMLPreparing))
		if sourceErr == nil {
			readCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			result.HTML, result.Err = htmldocument.OpenRemote(readCtx, reader, sid, path, valid)
			cancel()
		} else {
			result.HTML, result.Err = htmldocument.OpenWorkspace(root, path, valid)
		}
	default:
		result.Err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdPreviewRestoreUnsupported))
	}
	html := result.HTML
	readFailed := result.Err != nil
	result.Complete = func(err error) {
		if !valid() {
			return
		}
		if child {
			sessions.SetAgentResourceStatus(childToken, address, path, false, err)
			return
		}
		if source.Kind == browserpreviewui.DocumentOfficePDF {
			switch {
			case err != nil && readFailed:
				s.setOfficePreviewMessage(sessions, sid, path, officePreviewFailure(err))
			case err != nil:
				s.setOfficePreviewMessage(sessions, sid, path, fmt.Sprintf(i18n.T(i18n.KeyCmdOfficeReaderFailed), err.Error()))
			case office != nil:
				s.setOfficePreviewMessage(sessions, sid, path, officePreviewReady(office.MissingFonts))
			}
		} else if err != nil {
			sessions.SetLastAction(fmt.Sprintf(i18n.T(i18n.KeyCmdDocumentPreviewFailed), err.Error()))
		} else if html != nil {
			sessions.SetLastAction(fmt.Sprintf(i18n.T(i18n.KeyCmdHTMLSnapshotOpened), html.Resources.Summary()))
		} else {
			sessions.SetLastAction(i18n.T(i18n.KeyCmdPDFSnapshotOpened))
		}
	}
	return result
}
