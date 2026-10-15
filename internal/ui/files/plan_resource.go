package files

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// OpenPlanResource uses the existing right-pane tab owner, independently from
// editable workspace paths. It cannot save, approve, execute, or open a URL.
func (t *Tabs) OpenPlanResource(sessionID, expectedBinding string, doc planpreview.Document) string {
	if expectedBinding == "" || t.binding != expectedBinding {
		return "The plan resource belongs to an obsolete Host/session binding"
	}
	target, ok := planpreview.ParseAddress(doc.Address)
	if !ok || target.SessionID != sessionID {
		return "Invalid plan resource for the current session"
	}
	t.planSession = sessionID
	// A durable address resolves from history, never from a live question's text.
	if target.Temporary {
		t.Plans.Observe(doc)
	}
	t.Pause()
	return t.Plans.Navigate(sessionID, doc.Address)
}

func (t *Tabs) SyncPlanSession(sessionID string) { t.planSession = sessionID }

func observePlanHistory(m Model) {
	if m.Tabs == nil || !m.PlanSnapshot.ClientReady || !planpreview.HistoryMatchesBinding(m.PlanSnapshot.PlanHistoryBinding, m.BindingID) {
		return
	}
	for _, message := range m.PlanSnapshot.Messages {
		if plan, ok := session.SubmittedPlanFromEvent(message.Raw); ok {
			m.Tabs.Plans.Observe(planpreview.Document{Address: planpreview.PlanAddress(m.PlanSnapshot.SelectedID, plan.CallID), Title: plan.Title, Markdown: plan.Markdown})
		}
	}
}

func paintPlanTabs(c *ui.Context, m Model) {
	if m.Tabs == nil {
		return
	}
	r := &m.Tabs.Plans
	sid := m.PlanSnapshot.SelectedID
	if len(r.SessionTabs(sid)) == 0 {
		return
	}
	ui.Row(c).FillWidth().Wrap().Gap(4).Padding(6, 8).Children(func() {
		if ui.Button(c, label("文件", "Files")).FontSize(11).Clicked() {
			r.ShowFiles(sid)
		}
		for _, address := range r.SessionTabs(sid) {
			name := label("计划", "Plan")
			if doc, ok := r.Documents[address]; ok && doc.Title != "" {
				name = doc.Title
			}
			if address == r.ActiveAddress(sid) {
				name = "▸ " + name
			}
			if ui.Button(c, name).Tooltip(address).FontSize(11).Clicked() {
				r.Navigate(sid, address)
			}
			if ui.Button(c, "×").Tooltip(label("关闭计划", "Close plan")).FontSize(11).Clicked() {
				r.Close(sid, address)
			}
		}
	})
}

func paintPlanResource(c *ui.Context, m Model) bool {
	if m.Tabs == nil {
		return false
	}
	r := &m.Tabs.Plans
	sid := m.PlanSnapshot.SelectedID
	address := r.ActiveAddress(sid)
	if address == "" {
		return false
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, accent := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Accent)
	root := ui.Column(c.Key("plan-resource:" + sid)).Fill().MinHeight(0).Gap(6)
	type navigation struct{ Current, Draft, Error string }
	nav := ui.Local(root, "address", func() navigation { return navigation{Current: address, Draft: address} })
	if nav.Current != address {
		nav.Current, nav.Draft, nav.Error = address, address, ""
	}
	root.Children(func() {
		paintPlanTabs(c, m)
		ui.Row(c).FillWidth().Padding(0, 8).Gap(4).Children(func() {
			ui.TextInput(c, &nav.Draft).Grow(1).Label("Plan resource address")
			if ui.Button(c, label("前往", "Go")).Clicked() {
				nav.Error = r.Navigate(sid, nav.Draft)
			}
			if ui.Button(c, label("复制地址", "Copy address")).Clicked() {
				mygo.Clipboard.WriteText(address)
			}
		})
		if nav.Error != "" {
			ui.Text(c, nav.Error).TextColor(muted).FontSize(12)
		}
		doc, ok := r.Current(sid)
		if !ok {
			target, _ := planpreview.ParseAddress(address)
			message := label("未找到此计划。", "Plan not found.")
			switch {
			case target.Temporary:
				message = label("临时审批预览已过期，请从当前审批卡重新打开。", "This temporary review preview expired; reopen it from the current review card.")
			case !m.PlanSnapshot.ClientReady || !planpreview.HistoryMatchesBinding(m.PlanSnapshot.PlanHistoryBinding, m.BindingID):
				message = label("等待当前会话历史…", "Waiting for current session history…")
			case m.PlanSnapshot.HistoryPageErr != "":
				message = m.PlanSnapshot.HistoryPageErr
				if m.OnLoadPlanHistory != nil && ui.Button(c, label("重试", "Retry")).Clicked() {
					m.OnLoadPlanHistory()
				}
			case m.PlanSnapshot.HistoryPageBusy:
				message = label("正在读取计划历史…", "Reading plan history…")
			case m.PlanSnapshot.HistoryHasMore:
				message = label("正在定位历史计划…", "Locating the historical plan…")
				if m.OnLoadPlanHistory != nil {
					m.OnLoadPlanHistory()
				}
			}
			ui.Text(c, message).TextColor(muted).Padding(8)
			return
		}
		ui.Row(c).FillWidth().Gap(8).Padding(8).Children(func() {
			ui.Text(c, doc.Title).Bold().TextColor(fg).Grow(1)
			if ui.Button(c, label("复制文档", "Copy document")).Clicked() {
				mygo.Clipboard.WriteText(doc.Markdown)
			}
		})
		ui.Text(c, label("只读计划文档", "Read-only plan document")).FontSize(11).TextColor(muted).Padding(0, 8)
		ui.Scroll(c.Key("plan-body:" + address)).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(12).Children(func() {
				markdown.View(c, doc.Markdown, markdown.Style{FontSize: 14, FG: fg, Muted: muted, Accent: accent, CodeBG: chrome.Color(colors.Field)})
			})
		})
	})
	return true
}
