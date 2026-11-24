package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func browseArchiveFilter(m Model) string {
	return sharedArchiveFilter(m)
}

func workspaceRowsStatus(c *ui.Context, m Model, fg ui.Color) {
	layout := m.Snapshot.WorkspaceLayout
	if layout.PreferenceError != "" {
		ui.Text(c, layout.PreferenceError).FontSize(fs(11)).TextColor(fg)
		if coss.Button(c, workspaceText("重试视图偏好", "Retry view preferences"), coss.ButtonOptions{
			Variant: coss.ButtonLink,
			Size:    coss.ButtonXS,
			Key:     "workspace-retry-prefs",
		}).FontSize(fs(11)).Clicked() && m.OnRetryWorkspacePreferences != nil {
			m.OnRetryWorkspacePreferences()
		}
	}
	if layout.Error != "" {
		ui.Text(c, layout.Error).FontSize(fs(11)).TextColor(fg)
	}
}

func workspaceListFeedback(c *ui.Context, m Model, items []session.Item, fg, muted ui.Color) {
	if m.Snapshot.ListErr != nil {
		// The classified sentence, never the transport's own words: those stay
		// in the session store's log line.
		line := workspaceText("会话列表没有加载：", "Sessions did not load: ") + m.Snapshot.ListErr.SentenceText()
		ui.Text(c, line).FontSize(fs(12)).TextColor(fg).Padding(design.Sp(2))
		if m.OnRefresh != nil && coss.Button(c, workspaceText("重试加载会话", "Retry loading sessions"), coss.ButtonOptions{
			Variant: coss.ButtonOutline,
			Size:    coss.ButtonSM,
			Key:     "workspace-retry-list",
		}).Clicked() {
			m.OnRefresh()
		}
		return
	}
	filter := browseArchiveFilter(m)
	if m.Snapshot.ListLoading || m.Snapshot.ClientReady && !m.Snapshot.WorkspaceReady {
		ui.Text(c, workspaceText("正在加载会话…", "Loading sessions…")).FontSize(fs(12)).TextColor(muted).Padding(design.Sp(3))
		return
	}
	if !m.Snapshot.WorkspaceReady {
		// An empty list names the state that is actually holding it: a service
		// that has not started says so, and the workspace copy only speaks for
		// a running service that has no workspace selected yet.
		line := m.HostNote
		if line == "" {
			line = workspaceText("连接后显示会话", "Connect to load sessions")
		}
		ui.Text(c, line).FontSize(fs(12)).TextColor(muted).Padding(design.Sp(3))
		return
	}
	if len(visibleWorkspaceGroups(m, items)) > 0 || len(archivedSectionItems(m)) > 0 {
		return
	}
	t := tok(c)
	topMargin := design.Sp(20)
	if filter == "only" {
		topMargin = design.Sp(4)
	}
	ui.Column(c.Key("workspace-empty-state")).Label("Empty sessions").FillWidth().AlignItems(ui.Center).
		Gap(design.Sp(2)).Margin(topMargin, 0, 0, 0).Padding(0, design.Sp(3)).Children(func() {
		ui.Box(c).Width(24).Height(24).Margin(0, 0, design.Sp(1), 0).Draw(func(p *ui.Painter, r ui.Rect) {
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
		ui.Text(c, text).FontSize(fs(13)).FixedLineHeight(20).TextColor(muted)
		if filter == "only" && canSetArchiveFilter(m) && coss.Button(c, i18n.T(i18n.KeyWSEmptyViewOthers), coss.ButtonOptions{
			Variant: coss.ButtonLink,
			Size:    coss.ButtonXS,
			Key:     "workspace-empty-view-others",
		}).Padding(0).FontSize(fs(13)).FixedLineHeight(20).TextColor(t.Primary).Clicked() {
			setArchiveFilter(m, "default")
		}
	})
}
