package files

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/workspacefs"
)

// DeliveryPreviewKind identifies a product-owned, read-only saved-file viewer.
// These routes have no relationship to Open on Host or the system file opener.
type DeliveryPreviewKind string

const (
	DeliveryPreviewPDF   DeliveryPreviewKind = "pdf"
	DeliveryPreviewHTML  DeliveryPreviewKind = "html"
	DeliveryPreviewMedia DeliveryPreviewKind = "media"
)

// PreviewDeliveryPath accepts exactly the declared local file for a read-only
// preview. Nil means an asynchronous reader/viewer was accepted; its loading and
// terminal error state belong to the selected Files tab or document route. It
// does not certify successful rendering. Unlike OpenPresentedPath it never
// interprets line suffixes or falls back to a Host/default application. The caller
// has already revalidated the declaration's current session/delivery coordinates.
func PreviewDeliveryPath(m Model, path string) error {
	unavailable := func() error {
		return errors.New(label("无法预览：文件路径不可达或不在当前工作区", "Cannot preview: the file is unavailable or outside the current workspace"))
	}
	unsupported := func() error {
		return errors.New(label("无法预览此文件类型，可选择在 Host 打开", "This file type cannot be previewed; choose Open on Host to open it"))
	}
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) {
		return unavailable()
	}
	if u, err := url.Parse(path); strings.Contains(path, "://") || (err == nil && u.Scheme != "" && len(u.Scheme) != 1) {
		return unavailable()
	}
	root := m.WorkspaceRoot
	if !filepath.IsAbs(root) || (!filepath.IsAbs(path) && (strings.HasPrefix(path, `\`) || (len(path) > 1 && path[1] == ':'))) {
		return unavailable()
	}
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, full)
	}
	rel, err := filepath.Rel(root, full)
	if err != nil || outsideRoot(rel) || rel == "." {
		return unavailable()
	}
	// Apply the Files surface's full canonical path policy, including its .git
	// exclusion, before selecting a tab or opening a document viewer.
	if _, err := workspacefs.JoinRoot(root, filepath.ToSlash(rel)); err != nil {
		return unavailable()
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return unavailable()
	}
	realFile, err := filepath.EvalSymlinks(full)
	if err != nil {
		return unavailable()
	}
	realRel, err := filepath.Rel(realRoot, realFile)
	if err != nil || outsideRoot(realRel) {
		return unavailable()
	}
	info, err := os.Stat(full)
	if err != nil || !info.Mode().IsRegular() {
		return unavailable()
	}
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() {
		return unavailable()
	}
	if !deliveryPreviewSupported(path) {
		return unsupported()
	}
	if m.SelectedFile == nil || (m.Tabs != nil && (m.Tabs.binding != m.BindingID || m.Tabs.boundRoot != m.WorkspaceRoot)) {
		return unavailable()
	}
	// Flush the committed native text into its owning tab before checking dirty
	// state, without starting a default/Host read as prepareTabs would do.
	capture(m)
	selected := filepath.ToSlash(rel)
	if w := tabsFor(m); w != nil {
		if t := findTab(w, selected); t != nil && t.Dirty() {
			return errors.New(label("文件有未保存的草稿；请先保存后再预览", "The file has an unsaved draft; save it before previewing"))
		}
	}
	if kind := deliveryDocumentKind(path); kind != "" {
		if m.OpenDeliveryDocument == nil {
			return errors.New(label("本地只读预览器不可用", "Local read-only viewer is unavailable"))
		}
		// The dedicated callback owns a real loading/error document route. The
		// general document/Host opener cannot be a fallback for this gesture.
		return m.OpenDeliveryDocument(full, kind)
	}
	source := &deliveryPreviewSource{binding: m.BindingID, session: m.SessionID, root: root, path: selected, canonical: realFile, file: info, rootFile: rootInfo}
	if m.Tabs == nil {
		// Legacy text-only callers have no persistent owner for asynchronous
		// viewers. Their immediate text read still uses the same rooted handle.
		if isPreviewImage(path) || isTablePreviewPath(path) {
			return errors.New(label("此预览需要文件标签页", "This preview requires a file tab"))
		}
		if m.EditDraft != nil && m.Preview != nil && *m.EditDraft != *m.Preview {
			return errors.New(label("无法切换预览：请先保存当前草稿", "Cannot switch previews; save the current draft first"))
		}
		page, err := source.readTextPage(context.Background(), 0)
		if err != nil {
			return err
		}
		*m.SelectedFile = selected
		if m.Preview != nil {
			*m.Preview = page.Text
		}
		if m.EditDraft != nil {
			*m.EditDraft = page.Text
		}
		if m.PreviewErr != nil {
			*m.PreviewErr = ""
		}
	} else {
		w := tabsFor(m)
		pauseSave(selectedTab(m))
		m.Tabs.stopTextRead(true, false)
		m.Tabs.stopImageRead(false)
		m.Tabs.stopResourceObserver()
		tab := findTab(w, selected)
		if tab == nil {
			tab = &FileTab{Path: selected}
			w.Open = append(w.Open, tab)
		}
		tab.closeDocumentSurface()
		disposeSave(tab)
		tab.deliverySource = source
		tab.Remote, tab.Editing = false, false
		tab.Preview, tab.Draft, tab.Error, tab.Status = "", "", "", ""
		tab.tablePage, tab.image = nil, nil
		tab.tableRequest = workspacefs.XLSXPageRequest{}
		w.Active, m.Tabs.Root = selected, filepath.Clean(root)
		restore(m, tab)
		reloadSelectedFile(m)
	}
	if m.Editing != nil {
		*m.Editing = false
	}
	if tab := selectedTab(m); tab != nil {
		tab.Editing = false
	}
	return nil
}

func deliveryDocumentKind(path string) DeliveryPreviewKind {
	if mediapreview.Supports(path) {
		return DeliveryPreviewMedia
	}
	switch workspacefs.ClassifyDoc(path) {
	case workspacefs.DocPDF:
		return DeliveryPreviewPDF
	case workspacefs.DocHTML:
		return DeliveryPreviewHTML
	}
	return ""
}

// The file pane treats unrecognized extensions as text. Only known inert
// document, image, media and text formats are admitted for this delivery entry
// point; other file types can be opened by a separate explicit Host action.
func deliveryPreviewSupported(path string) bool {
	if isPreviewImage(path) {
		return true
	}
	if isTablePreviewPath(path) || deliveryDocumentKind(path) != "" {
		return true
	}
	if workspacefs.ClassifyDoc(path) == workspacefs.DocOther {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case "", ".txt", ".md", ".markdown", ".log", ".csv", ".tsv", ".json", ".jsonl", ".xml", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf", ".properties", ".env", ".sql", ".go", ".c", ".h", ".cc", ".cpp", ".hpp", ".cs", ".java", ".kt", ".rs", ".py", ".rb", ".php", ".js", ".jsx", ".ts", ".tsx", ".css", ".scss", ".less", ".sh", ".bash", ".zsh", ".ps1", ".diff", ".patch", ".rst", ".tex":
		return true
	}
	return false
}
