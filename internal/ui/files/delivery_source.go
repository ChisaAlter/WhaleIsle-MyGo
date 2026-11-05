package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"whaleisle/internal/desktop/workspacefs"
)

// Runtime-only authority for one declared saved local file. A tab can retain its
// draft across Host changes, but neither an old read nor a new frame may silently
// reinterpret this declaration using another Host, session, or replacement file.
type deliveryPreviewSource struct {
	binding, session, root, path, canonical string
	file, rootFile                          os.FileInfo
}

func (s *deliveryPreviewSource) current(m Model) bool {
	return s != nil && s.binding == m.BindingID && s.session == m.SessionID && s.root == m.WorkspaceRoot &&
		(m.Tabs == nil || m.Tabs.binding == m.BindingID && m.Tabs.boundRoot == m.WorkspaceRoot)
}

func deliverySourceChanged() error {
	return errors.New(label("本地预览来源已更改，请从当前会话的文件卡重新打开。", "Local preview source changed; reopen it from the current session's file card."))
}

func (s *deliveryPreviewSource) resolve() (string, error) {
	full, err := workspacefs.JoinRoot(s.root, s.path)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(full)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || canonical != s.canonical || !os.SameFile(s.file, info) {
		return "", deliverySourceChanged()
	}
	return full, nil
}

// The pathname check is for policy/error presentation only. Delivered bytes are
// read exclusively through this rooted, identity-checked handle, never by opening
// the resolved pathname again. This also rejects junction/symlink replacement.
func (s *deliveryPreviewSource) open() (*os.Root, *os.File, os.FileInfo, error) {
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return nil, nil, nil, err
	}
	reject := func(err error) (*os.Root, *os.File, os.FileInfo, error) {
		_ = root.Close()
		return nil, nil, nil, err
	}
	info, err := root.Stat(".")
	if err != nil {
		return reject(err)
	}
	if !os.SameFile(s.rootFile, info) {
		return reject(deliverySourceChanged())
	}
	file, err := root.Open(filepath.FromSlash(s.path))
	if err != nil {
		return reject(err)
	}
	info, err = file.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(s.file, info) {
		_ = file.Close()
		if err == nil {
			err = deliverySourceChanged()
		}
		return reject(err)
	}
	return root, file, info, nil
}

func (s *deliveryPreviewSource) readRange(ctx context.Context, offset, limit int64) ([]byte, os.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	root, file, before, err := s.open()
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	defer file.Close()
	if offset < 0 || offset > before.Size() {
		return nil, nil, errors.New("local preview offset is outside the file")
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, nil, err
	}
	after, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	current, err := root.Stat(filepath.FromSlash(s.path))
	if err != nil {
		return nil, nil, err
	}
	if !os.SameFile(before, after) || !os.SameFile(before, current) || before.Size() != after.Size() || before.Size() != current.Size() || !before.ModTime().Equal(after.ModTime()) || !before.ModTime().Equal(current.ModTime()) || before.Mode() != current.Mode() {
		return nil, nil, deliverySourceChanged()
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return data, before, nil
}

func (s *deliveryPreviewSource) readBytes(ctx context.Context, limit int64) ([]byte, error) {
	data, info, err := s.readRange(ctx, 0, limit)
	if err == nil && (int64(len(data)) > limit || info.Size() > limit) {
		err = errors.New(label("文件超过只读预览大小限制", "File exceeds the read-only preview size limit"))
	}
	return data, err
}

func (s *deliveryPreviewSource) readTextPage(ctx context.Context, offset int64) (workspacefs.TextPage, error) {
	data, info, err := s.readRange(ctx, offset, workspacefs.TextPreviewLimit)
	if err != nil {
		return workspacefs.TextPage{}, err
	}
	overflow := len(data) > workspacefs.TextPreviewLimit
	if overflow {
		data = data[:workspacefs.TextPreviewLimit]
	}
	for _, b := range data {
		if b == 0 {
			return workspacefs.TextPage{}, workspacefs.ErrBinary
		}
	}
	if !utf8.Valid(data) && overflow {
		for n := 1; n < utf8.UTFMax && n <= len(data); n++ {
			prefix, tail := data[:len(data)-n], data[len(data)-n:]
			if !utf8.FullRune(tail) && utf8.Valid(prefix) {
				data = prefix
				break
			}
		}
	}
	if !utf8.Valid(data) {
		return workspacefs.TextPage{}, workspacefs.ErrBinary
	}
	next := offset + int64(len(data))
	return workspacefs.TextPage{Text: string(data), Next: next, EOF: !overflow && next >= info.Size(), Version: fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())}, nil
}

func (s *deliveryPreviewSource) readTable(ctx context.Context, request workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
	data, err := s.readBytes(ctx, 16<<20)
	if err != nil {
		return nil, err
	}
	return workspacefs.ReadNamedWorkbookBytesPage(ctx, s.path, data, request)
}

func (s *deliveryPreviewSource) probe() (os.FileInfo, error) {
	root, file, info, err := s.open()
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	_ = root.Close()
	return info, nil
}

func previewPathResolver(m Model, tab *FileTab) func() (string, error) {
	if tab != nil && tab.deliverySource != nil {
		source := tab.deliverySource
		if !source.current(m) {
			return func() (string, error) { return "", deliverySourceChanged() }
		}
		return source.resolve
	}
	root, path := m.WorkspaceRoot, tab.Path
	return func() (string, error) { return workspacefs.JoinRoot(root, path) }
}

// Strip every alternate reader on each operation, not just the admission frame.
// The source remains on the tab when it is revisited, paged, observed or reloaded.
func deliveryReadModel(m Model, tab *FileTab) (Model, error) {
	if tab == nil || tab.deliverySource == nil {
		return m, nil
	}
	m.ReadRemote, m.ReadRemoteContext, m.ReadTextPageContext = nil, nil, nil
	m.ReadImageContext = nil
	m.ReadTableContext, m.StatRemote, m.StatRemoteContext = nil, nil, nil
	m.OpenDocumentViewer = nil
	m.WatchRemoteContext = nil
	if !tab.deliverySource.current(m) {
		return m, deliverySourceChanged()
	}
	source := tab.deliverySource
	m.WatchRemoteContext = func(ctx context.Context, _ string, changed func()) error {
		// Poll only this saved local file. Host directory subscriptions must never
		// turn a local delivery into a Host-owned resource.
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		previous, previousErr := source.probe()
		// Re-read after establishing the baseline: the initial observer read may
		// have completed before this watcher goroutine started its first probe.
		if ctx.Err() == nil {
			changed()
		}
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				next, err := source.probe()
				changedState := (previousErr == nil) != (err == nil)
				if err == nil && previous != nil {
					changedState = changedState || !os.SameFile(previous, next) || previous.Size() != next.Size() || !previous.ModTime().Equal(next.ModTime()) || previous.Mode() != next.Mode()
				}
				if changedState && ctx.Err() == nil {
					changed()
				}
				previous, previousErr = next, err
			}
		}
	}
	return m, nil
}

func rejectDeliveryRead(m Model, tab *FileTab, err error) {
	if m.Tabs != nil {
		m.Tabs.stopTextRead(false, false)
		m.Tabs.stopResourceObserver()
		m.Tabs.stopImageRead(false)
	}
	if !tab.Dirty() {
		tab.Preview, tab.Draft = "", ""
	}
	tab.Error = err.Error()
	tab.readLoading, tab.readResume, tab.readBlocked = false, false, true
	tab.Editing = false
	tab.image = nil
	tab.closeDocumentSurface()
	restore(m, tab)
}

func syncDeliveryPreview(m Model) {
	if tab := selectedTab(m); tab != nil && tab.deliverySource != nil && !tab.deliverySource.current(m) {
		rejectDeliveryRead(m, tab, deliverySourceChanged())
	}
}
