package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/ui/i18n"

	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/desktop/workspacefs"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/files"
)

const localDeliveryAddress = "local-delivery"

// Keep one comparable sentinel; resolve its visible text when Error is read so
// a locale switch never freezes this process-wide error in the startup locale.
var errLocalDeliveryPermission = &localDeliveryPermissionError{}

type localDeliveryPermissionError struct{}

func (*localDeliveryPermissionError) Error() string {
	return i18n.T(i18n.KeyCmdLocalPreviewPermissionEnded)
}

// Identity belongs to the explicit gesture, not to a later restored address.
// os.Root confines every open, including symlink/junction resolution.
type deliverableDocumentIdentity struct {
	root, path, rel    string
	rootInfo, fileInfo os.FileInfo
}

func captureDeliverableDocumentIdentity(root, path string, valid func() bool) (*deliverableDocumentIdentity, error) {
	if valid == nil || !valid() {
		return nil, errLocalDeliveryPermission
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(path) || strings.Contains(path, "://") || strings.ContainsAny(path, "\x00") || strings.HasPrefix(root, `\\`) || strings.HasPrefix(path, `\\`) {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewExactFile))
	}
	root, path = filepath.Clean(root), filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewOutsideWorkspace))
	}
	if _, err = workspacefs.JoinRoot(root, filepath.ToSlash(rel)); err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	ri, err := r.Stat(".")
	if err != nil {
		return nil, err
	}
	before, err := r.Stat(rel)
	if err != nil || !before.Mode().IsRegular() {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewRegularFile))
	}
	f, err := r.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !sameDeliverableFile(before, opened) {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewSourceChanged))
	}
	id := &deliverableDocumentIdentity{root: root, path: path, rel: rel, rootInfo: ri, fileInfo: opened}
	if !valid() || !id.current() {
		return nil, errLocalDeliveryPermission
	}
	return id, nil
}

func sameDeliverableFile(a, b os.FileInfo) bool {
	return a != nil && b != nil && a.Mode().IsRegular() && b.Mode().IsRegular() && os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func (id *deliverableDocumentIdentity) current() bool {
	if id == nil {
		return false
	}
	if _, err := workspacefs.JoinRoot(id.root, filepath.ToSlash(id.rel)); err != nil {
		return false
	}
	r, err := os.OpenRoot(id.root)
	if err != nil {
		return false
	}
	defer r.Close()
	ri, err := r.Stat(".")
	if err != nil || !os.SameFile(id.rootInfo, ri) {
		return false
	}
	fi, err := r.Stat(id.rel)
	return err == nil && sameDeliverableFile(id.fileInfo, fi)
}

func (s *mainState) localDeliverableDocumentSource(owner deliverablePreviewOwner, path string, kind files.DeliveryPreviewKind) (browserpreviewui.DocumentSource, error) {
	var source browserpreviewui.DocumentSource
	if !s.localDeliverablePreviewScope(owner) {
		return source, errLocalDeliveryPermission
	}
	switch kind {
	case files.DeliveryPreviewPDF:
		if !strings.EqualFold(filepath.Ext(path), ".pdf") {
			return source, errors.New(i18n.T(i18n.KeyCmdLocalPreviewPDFSource))
		}
		source.Kind = browserpreviewui.DocumentPDF
	case files.DeliveryPreviewHTML:
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".htm" {
			return source, errors.New(i18n.T(i18n.KeyCmdLocalPreviewHTMLSource))
		}
		source.Kind = browserpreviewui.DocumentHTML
	case files.DeliveryPreviewMedia:
		if !mediapreview.Supports(path) {
			return source, errors.New(i18n.T(i18n.KeyCmdLocalPreviewMediaSource))
		}
		source.Kind = browserpreviewui.DocumentMedia
	default:
		return source, errors.New(i18n.T(i18n.KeyCmdLocalPreviewUnsupported))
	}
	ownerValid := func() bool { return s.localDeliverablePreviewScope(owner) }
	identity, err := captureDeliverableDocumentIdentity(owner.cwd, path, ownerValid)
	if err != nil {
		return source, err
	}
	source.Authority = fmt.Sprintf("local-delivery:%p:%d", owner.supervisor, owner.generation)
	source.SessionID, source.ViewSessionID = owner.scope.SessionID, owner.scope.SessionID
	source.AddressKind, source.Workspace, source.Path = localDeliveryAddress, identity.root, identity.path
	return browserpreviewui.BindLocalDeliveryDocument(source, func() bool { return ownerValid() && identity.current() }), nil
}

// Nil only accepts the route. The selected browser document owns its loading,
// read failure and native-viewer failure; no Host or shell opener is consulted.
func (s *mainState) openLocalDeliverableDocument(owner deliverablePreviewOwner, path string, kind files.DeliveryPreviewKind) error {
	source, err := s.localDeliverableDocumentSource(owner, path, kind)
	if err != nil {
		return err
	}
	s.mu.Lock()
	available := s.win != nil && !s.quitting
	s.mu.Unlock()
	if !available || !source.LocalDeliveryValid() {
		return errLocalDeliveryPermission
	}
	if !s.openWorkspaceDocumentSource(owner.store, source) {
		return errLocalDeliveryPermission
	}
	return nil
}

// The local marker is handled before the ordinary session/Host reader. A saved
// descriptor never manufactures a new owner token on restoration.
func (s *mainState) readLocalDeliverableDocument(ctx context.Context, source browserpreviewui.DocumentSource) browserpreviewui.DocumentResult {
	result := browserpreviewui.DocumentResult{Valid: source.LocalDeliveryValid}
	if !source.LocalDeliveryValid() {
		result.Err = errLocalDeliveryPermission
		return result
	}
	if err := ctx.Err(); err != nil {
		result.Err = err
		return result
	}
	switch source.Kind {
	case browserpreviewui.DocumentPDF:
		result.PDF, result.Err = pdfpreview.OpenWorkspace(source.Workspace, source.Path, source.LocalDeliveryValid)
		if result.Err == nil {
			// Verify the identity captured by OpenWorkspace as well as the
			// gesture's identity, closing a swap-and-restore during the open.
			_, result.Err = result.PDF.ExternalPath()
		}
	case browserpreviewui.DocumentHTML:
		result.HTML, result.Err = htmldocument.OpenWorkspace(source.Workspace, source.Path, source.LocalDeliveryValid)
		if result.Err == nil {
			_, result.Err = result.HTML.ExternalPath()
		}
	case browserpreviewui.DocumentMedia:
		result.Media, result.Err = readLocalDeliverableMedia(ctx, source)
	default:
		result.Err = errors.New(i18n.T(i18n.KeyCmdLocalPreviewUnsupported))
	}
	if result.Err == nil {
		if err := ctx.Err(); err != nil {
			result.Err = err
		} else if !source.LocalDeliveryValid() {
			result.Err = errLocalDeliveryPermission
		}
	}
	if result.Err != nil {
		if result.PDF != nil {
			result.PDF.Close()
			result.PDF = nil
		}
		if result.HTML != nil {
			result.HTML.Close()
			result.HTML = nil
		}
		if result.Media != nil {
			result.Media.Close()
			result.Media = nil
		}
	}
	return result
}

func readLocalDeliverableMedia(ctx context.Context, source browserpreviewui.DocumentSource) (*mediapreview.Document, error) {
	if !source.LocalDeliveryValid() {
		return nil, errLocalDeliveryPermission
	}
	rel, err := filepath.Rel(source.Workspace, source.Path)
	if err != nil {
		return nil, err
	}
	if err = workspacefs.ValidateRelativePath(filepath.ToSlash(rel)); err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(source.Workspace)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	before, err := r.Stat(rel)
	if err != nil || !before.Mode().IsRegular() {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewRegularMedia))
	}
	if before.Size() < 8 || before.Size() > mediapreview.MaxBytes {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewMediaLimit))
	}
	f, err := r.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !sameDeliverableFile(before, opened) || !source.LocalDeliveryValid() {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewMediaBeforeRead))
	}
	data, err := io.ReadAll(io.LimitReader(f, mediapreview.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil || !sameDeliverableFile(opened, after) || int64(len(data)) != opened.Size() || len(data) > mediapreview.MaxBytes {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewMediaDuringRead))
	}
	currentRoot, err := os.OpenRoot(source.Workspace)
	if err != nil {
		return nil, err
	}
	defer currentRoot.Close()
	readRootInfo, err := r.Stat(".")
	if err != nil {
		return nil, err
	}
	currentRootInfo, err := currentRoot.Stat(".")
	if err != nil || !os.SameFile(readRootInfo, currentRootInfo) {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewWorkspaceDuringRead))
	}
	current, err := currentRoot.Stat(rel)
	if err != nil || !sameDeliverableFile(opened, current) {
		return nil, errors.New(i18n.T(i18n.KeyCmdLocalPreviewMediaReplaced))
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if !source.LocalDeliveryValid() {
		return nil, errLocalDeliveryPermission
	}
	if _, _, err = mediapreview.Validate(source.Path, data); err != nil {
		return nil, err
	}
	return mediapreview.Open(filepath.Base(source.Path), data, source.LocalDeliveryValid)
}
