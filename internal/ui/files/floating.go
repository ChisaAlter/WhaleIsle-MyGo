package files

import (
	"context"
	"errors"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"time"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
	"whaleisle/internal/ui/widgets/markdown"
)

type floatingPreview struct {
	win      *mygo.Window
	snapshot previewSnapshot
	image    *imagePreview
	tabs     Tabs
	tab      FileTab
	themeID  string
	notify   func()
	reader   textReadQueue
	owner    textReadOwner
	read     func(context.Context) textReadResult
	loading  bool
}

// Floating text previews read saved bytes through the captured source binding.
// No editor draft or mutable session lookup crosses into the worker.
func openFloatingPreview(m Model) error {
	if m.Tabs == nil || m.SelectedFile == nil || m.WorkspaceRoot == "" || *m.SelectedFile == "" {
		return errors.New("no file preview owner")
	}
	if imageMode(m, *m.SelectedFile) {
		return errors.New("image floating preview requires the document viewer")
	}
	snapshot := previewSnapshot{Root: m.WorkspaceRoot, Path: *m.SelectedFile}
	f := m.Tabs.floating
	if f == nil || f.win == nil || f.win.IsDestroyed() {
		f = &floatingPreview{}
		m.Tabs.floating = f
	}
	f.snapshot, f.image, f.themeID, f.notify = snapshot, nil, m.ThemeID, m.notify
	f.owner = textReadOwner{binding: m.BindingID, root: snapshot.Root, session: m.SessionID, path: snapshot.Path}
	f.read = savedFloatingReader(m, snapshot.Path)
	f.tab = FileTab{Path: snapshot.Path, Preview: snapshot.Text, Draft: snapshot.Text}
	f.tabs = Tabs{Preferences: previewPreferences(m), Workspaces: map[string]*WorkspaceTabs{filepath.Clean(snapshot.Root): {Open: []*FileTab{&f.tab}, Active: snapshot.Path}}}
	title := label("浮动预览 · ", "Floating preview · ") + filepath.Base(snapshot.Path)
	if f.win == nil || f.win.IsDestroyed() {
		f.win = mygo.NewWindow(mygo.WindowOptions{Title: title, Width: 720, Height: 520, MinWidth: 320, MinHeight: 220, AlwaysOnTop: true, SkipTaskbar: true, DisableMinimize: true, DisableMaximize: true, DisableFullScreen: true, Content: ui.View(f.view)})
		if f.win == nil || f.win.IsDestroyed() {
			f.image = nil
			return errors.New(label("无法打开浮动预览", "Could not open floating preview"))
		}
		f.win.OnClosed(func() {
			f.reader.stop(true)
			f.win = nil
			f.image = nil
			f.snapshot = previewSnapshot{}
			f.tab = FileTab{}
			f.tabs = Tabs{}
		})
	} else {
		f.win.SetTitle(title)
		f.win.Invalidate()
	}
	f.reader.setNotify(f.win.Invalidate)
	f.refresh()
	f.win.ShowInactive()
	return nil
}
func (t *Tabs) closeFloatingPreview() {
	if t == nil || t.floating == nil {
		return
	}
	f := t.floating
	t.floating = nil
	f.reader.stop(true)
	if f.win != nil && !f.win.IsDestroyed() {
		f.win.Destroy()
	}
}
func (f *floatingPreview) refresh() {
	f.loading = true
	f.snapshot.Problem = ""
	f.reader.request(f.owner, f.read)
}
func (f *floatingPreview) view(c *ui.Context) {
	if result, ok := f.reader.take(f.owner); ok {
		f.loading = false
		f.snapshot.Problem = ""
		if result.err != nil {
			f.snapshot.Problem = result.err.Error()
		} else {
			f.snapshot.Text = result.text
			f.tab.Preview, f.tab.Draft = result.text, result.text
			if result.truncated {
				f.snapshot.Problem = label("内容过长，仅显示第一页。", "Large file: showing the first page only.")
			}
		}
	}

	theme.ApplyInterface(c)
	dark := mygo.Theme.IsDark()
	colors := theme.Resolve(f.themeID).Active(dark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	m := Model{Tabs: &f.tabs, WorkspaceRoot: f.snapshot.Root, SelectedFile: &f.snapshot.Path, Preview: &f.snapshot.Text, EditDraft: &f.snapshot.Text, ThemeID: f.themeID, SystemDark: dark, notify: c.Invalidate}
	ui.Column(c).Fill().MinHeight(0).Background(chrome.Color(colors.BG)).Children(func() {
		ui.Row(c).FillWidth().Wrap().Gap(6).Padding(8, 12).Children(func() {
			ui.Text(c, f.snapshot.Path).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1)
			if f.image == nil {
				before := previewPreferences(m).WordWrap
				paintWrapToggle(c, m)
				if before != previewPreferences(m).WordWrap && f.notify != nil {
					f.notify()
				}
				if widgets.Button(c, colors, label("复制文本", "Copy text"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
					mygo.Clipboard.WriteText(f.snapshot.Text)
				}
				if isMarkdown(f.snapshot.Path) && widgets.Button(c, colors, label("源码或渲染", "Source or rendered"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
					f.tab.RenderedMarkdown = !f.tab.RenderedMarkdown
				}
			}
			if widgets.Button(c, colors, label("刷新", "Refresh"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(f.loading).Clicked() {
				f.refresh()
			}
			if widgets.Button(c, colors, label("关闭", "Close"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() && f.win != nil {
				f.win.Close()
			}
		})
		ui.Text(c, label("已保存文件的只读预览；点击刷新读取最新内容。", "Saved-file preview. Refresh to read the latest saved content.")).FontSize(theme.InterfaceSize(10)).TextColor(muted).Padding(0, 12, 4, 12)
		if f.snapshot.Problem != "" {
			ui.Text(c, f.snapshot.Problem).FontSize(theme.InterfaceSize(10)).TextColor(muted)
		}
		if f.image != nil {
			if f.image.note != "" {
				ui.Text(c, f.image.note).FontSize(theme.InterfaceSize(10)).TextColor(muted)
			}
			paintImageView(c, f.image, &f.tab.imageView)
		} else if f.tab.RenderedMarkdown && isMarkdown(f.snapshot.Path) {
			ui.Scroll(c).Fill().MinHeight(0).Children(func() {
				ui.Column(c).FillWidth().Padding(12).Children(func() { markdown.ViewInteractive(c, f.snapshot.Text, sourceStyle(m, fg, muted), nil) })
			})
		} else {
			paintSourceArea(c, m, f.snapshot.Path, &f.snapshot.Text, true, "floating", fg, muted)
		}
	})
}

// The desktop floating action uses the same local saved file as FilePreview.
// Separate Host-owned fallback resources retain their captured readers.
func savedFloatingReader(m Model, path string) func(context.Context) textReadResult {
	root, remote, pageReader, tableReader := m.WorkspaceRoot, m.ReadRemoteContext, m.ReadTextPageContext, m.ReadTableContext
	if desktopFileToolbar(m, path) {
		remote, pageReader = nil, nil
	}
	return func(ctx context.Context) textReadResult {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		out := textReadResult{}
		if ctx.Err() != nil {
			out.err = ctx.Err()
			return out
		}
		if isTablePreviewPath(path) {
			if tableReader != nil {
				out.table, out.err = tableReader(ctx, path, workspacefs.XLSXPageRequest{})
			} else if remote != nil || pageReader != nil {
				out.err = errors.New("Host table preview reader is unavailable")
			} else {
				full, err := workspacefs.JoinRoot(root, path)
				out.err = err
				if err == nil {
					out.table, out.err = readTablePreview(ctx, full, workspacefs.XLSXPageRequest{})
				}
			}
			if out.table != nil {
				out.text = out.table.Text
			}
		} else if pageReader != nil {
			page, err := pageReader(ctx, path, 0)
			out.text, out.err, out.truncated = page.Text, err, !page.EOF
		} else if remote != nil {
			out.text, out.err = remote(ctx, path)
		} else {
			full, err := workspacefs.JoinRoot(root, path)
			out.err = err
			if err == nil {
				page, err := workspacefs.ReadFileTextPage(full, 0)
				out.text, out.err, out.truncated = page.Text, err, !page.EOF
			}
		}
		if ctx.Err() != nil {
			out.err = ctx.Err()
		}
		return out
	}
}
