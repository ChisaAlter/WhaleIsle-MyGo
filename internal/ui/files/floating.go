package files

import (
	"errors"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
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
}

// openFloatingPreview opens one native, read-only, always-on-top viewer. Only
// already-loaded values cross into it; it owns no reader, writer or Host client.
func openFloatingPreview(m Model) error {
	if m.Tabs == nil || m.SelectedFile == nil {
		return errors.New("no file preview owner")
	}
	preview, draft, problem := "", "", ""
	if m.Preview != nil {
		preview = *m.Preview
	}
	draft = preview
	if m.EditDraft != nil {
		draft = *m.EditDraft
	}
	if m.PreviewErr != nil {
		problem = *m.PreviewErr
	}
	snapshot, err := makePreviewSnapshot(m.WorkspaceRoot, *m.SelectedFile, preview, draft, problem, imageMode(m, *m.SelectedFile))
	if err != nil {
		return err
	}
	if previewTruncated(m) {
		if snapshot.Problem != "" {
			snapshot.Problem += "\n"
		}
		snapshot.Problem += previewTruncationNotice(m)
	}
	var image *imagePreview
	if snapshot.Image {
		tab := selectedTab(m)
		if tab == nil || tab.Remote || tab.image == nil || tab.image.err != nil || (tab.image.bitmap == nil && tab.image.vector == nil) {
			return errors.New(label("请先等待图片预览成功，再打开浮窗", "Wait for the image preview to load before opening its floating window"))
		}
		image = tab.image
	}
	f := m.Tabs.floating
	if f == nil || f.win == nil || f.win.IsDestroyed() {
		f = &floatingPreview{}
		m.Tabs.floating = f
	}
	f.snapshot, f.image, f.themeID, f.notify = snapshot, image, m.ThemeID, m.notify
	f.tab = FileTab{Path: snapshot.Path, Preview: snapshot.Text, Draft: snapshot.Text}
	f.tabs = Tabs{Preferences: previewPreferences(m), Workspaces: map[string]*WorkspaceTabs{filepath.Clean(snapshot.Root): {Open: []*FileTab{&f.tab}, Active: snapshot.Path}}}
	title := label("浮动预览 · ", "Floating preview · ") + filepath.Base(snapshot.Path)
	if f.win == nil || f.win.IsDestroyed() {
		f.win = mygo.NewWindow(mygo.WindowOptions{Title: title, Width: 720, Height: 520, MinWidth: 320, MinHeight: 220, AlwaysOnTop: true, SkipTaskbar: true, DisableMinimize: true, DisableMaximize: true, DisableFullScreen: true, Content: ui.View(f.view)})
		if f.win == nil || f.win.IsDestroyed() {
			f.image = nil
			return errors.New(label("无法打开浮动预览", "Could not open floating preview"))
		}
		f.win.OnClosed(func() { f.win = nil; f.image = nil; f.snapshot = previewSnapshot{}; f.tab = FileTab{}; f.tabs = Tabs{} })
	} else {
		f.win.SetTitle(title)
		f.win.Invalidate()
	}
	f.win.ShowInactive()
	return nil
}
func (t *Tabs) closeFloatingPreview() {
	if t == nil || t.floating == nil {
		return
	}
	f := t.floating
	t.floating = nil
	if f.win != nil && !f.win.IsDestroyed() {
		f.win.Destroy()
	}
}
func (f *floatingPreview) view(c *ui.Context) {
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
				if ui.Button(c, label("复制文本", "Copy text")).Clicked() {
					mygo.Clipboard.WriteText(f.snapshot.Text)
				}
				if isMarkdown(f.snapshot.Path) && ui.Button(c, label("源码 / 渲染", "Source / Rendered")).Clicked() {
					f.tab.RenderedMarkdown = !f.tab.RenderedMarkdown
				}
			}
			if ui.Button(c, label("关闭", "Close")).Clicked() && f.win != nil {
				f.win.Close()
			}
		})
		ui.Text(c, label("只读快照；再次点击文件工具栏的浮窗按钮可更新。", "Read-only snapshot. Reopen from the file toolbar to update it.")).FontSize(theme.InterfaceSize(10)).TextColor(muted).Padding(0, 12, 4, 12)
		if f.snapshot.Dirty {
			ui.Text(c, label("包含尚未保存的草稿；不会自动保存。", "Includes an unsaved draft; opening this window does not save it.")).FontSize(theme.InterfaceSize(10)).TextColor(muted).Padding(0, 12, 4, 12)
		}
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
