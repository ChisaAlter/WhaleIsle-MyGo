package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func browseArchiveFilter(m Model) string {
	return sharedArchiveFilter(m)
}

func workspaceRowsStatus(c *ui.Context, m Model, fg ui.Color) {
	layout := m.Snapshot.WorkspaceLayout
	if layout.PreferenceError != "" {
		ui.Text(c, layout.PreferenceError).FontSize(theme.InterfaceSize(11)).TextColor(fg)
		if ui.Button(c, workspaceText("重试视图偏好", "Retry view preferences")).FontSize(theme.InterfaceSize(11)).Clicked() && m.OnRetryWorkspacePreferences != nil {
			m.OnRetryWorkspacePreferences()
		}
	}
	if layout.Error != "" {
		ui.Text(c, layout.Error).FontSize(theme.InterfaceSize(11)).TextColor(fg)
	}
}

func workspaceListFeedback(c *ui.Context, m Model, items []session.Item, fg, muted ui.Color) {
	if m.Snapshot.ListErr != "" {
		ui.Text(c, m.Snapshot.ListErr).FontSize(theme.InterfaceSize(12)).TextColor(fg).Padding(8)
		if m.OnRefresh != nil && ui.Button(c, workspaceText("重试加载会话", "Retry loading sessions")).Clicked() {
			m.OnRefresh()
		}
		return
	}
	filter := browseArchiveFilter(m)
	if m.Snapshot.ListLoading || m.Snapshot.ClientReady && !m.Snapshot.WorkspaceReady {
		ui.Text(c, workspaceText("正在加载会话…", "Loading sessions…")).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(12)
		return
	}
	if !m.Snapshot.WorkspaceReady {
		ui.Text(c, workspaceText("连接后显示会话", "Connect to load sessions")).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(12)
		return
	}
	if len(visibleWorkspaceGroups(m, items)) > 0 || len(archivedSectionItems(m)) > 0 {
		return
	}
	margin := float32(80)
	if filter == "only" {
		margin = 16
	}
	ui.Column(c.Key("workspace-empty-state")).Label("Empty sessions").FillWidth().AlignItems(ui.Center).Gap(8).Margin(margin, 0, 0, 0).Padding(0, 12).Children(func() {
		ui.Box(c).Width(24).Height(24).Margin(0, 0, 4, 0).Draw(func(p *ui.Painter, r ui.Rect) {
			path := &ui.Path{}
			if filter == "only" {
				path.MoveTo(r.X+4, r.Y+8)
				path.LineTo(r.X+4, r.Y+20)
				path.LineTo(r.X+20, r.Y+20)
				path.LineTo(r.X+20, r.Y+8)
				path.MoveTo(r.X+2, r.Y+4)
				path.LineTo(r.X+22, r.Y+4)
				path.LineTo(r.X+22, r.Y+8)
				path.LineTo(r.X+2, r.Y+8)
				path.Close()
				path.MoveTo(r.X+9, r.Y+12)
				path.LineTo(r.X+15, r.Y+12)
			} else {
				for _, y := range []float32{6, 12, 18} {
					path.MoveTo(r.X+5, r.Y+y)
					path.LineTo(r.X+19, r.Y+y)
				}
			}
			p.StrokePath(path, 1.5, muted)
		})
		text := i18n.T(i18n.KeyWSEmptyNone)
		if filter == "only" {
			text = workspaceText("暂无已归档会话", "No archived sessions")
		}
		ui.Text(c, text).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).TextColor(muted)
		if filter == "only" && canSetArchiveFilter(m) && ui.Button(c, i18n.T(i18n.KeyWSEmptyViewOthers)).Padding(0).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).BorderWidth(0).Background(ui.Transparent).TextColor(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Accent)).Clicked() {
			setArchiveFilter(m, "default")
		}
	})
}
