package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/harness"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
	managerui "whaleisle/internal/ui/pluginmanager"
)

// syncPluginContributions is called before the main render's early returns.
// Leaving Plugins destroys its browser capability, including retained actions.
func (s *mainState) syncPluginContributions(visible bool) {
	if !visible {
		s.suspendPluginContributions()
	}
}

func (s *mainState) suspendPluginContributions() {
	s.mu.Lock()
	owner := s.nativePlugins.contributions
	store := s.settings
	hadForm := s.nativePlugins.formSubject != (manager.Subject{})
	s.nativePlugins.formSubject = manager.Subject{}
	s.nativePlugins.contributions = nil
	s.nativePlugins.contributionError = ""
	s.nativePlugins.contributionGeneration = 0
	s.nativePlugins.contributionOrigin, s.nativePlugins.contributionCookie = "", ""
	s.mu.Unlock()
	if hadForm && store != nil {
		store.LeaveServedForms()
	}
	if owner != nil {
		owner.Close()
	}
}

func (s *mainState) nativePluginContributions() (*pluginui.Owner, string) {
	s.mu.Lock()
	st, win, old := s.bootStatus, s.win, s.nativePlugins
	valid := st.State == harness.Ready && old.client != nil && old.client.Endpoint.Origin == st.Endpoint.Origin && old.client.Endpoint.Cookie == st.Endpoint.Cookie
	if !valid || win == nil || !win.IsVisible() || win.IsMinimized() {
		s.mu.Unlock()
		return nil, ""
	}
	if old.contributions != nil && old.contributions.Alive() && old.contributionGeneration == st.Generation && old.contributionOrigin == st.Endpoint.Origin && old.contributionCookie == st.Endpoint.Cookie {
		s.mu.Unlock()
		return old.contributions, old.contributionError
	}
	if old.contributionError != "" {
		s.mu.Unlock()
		return nil, old.contributionError
	}
	s.mu.Unlock()
	if old.contributions != nil {
		old.contributions.Close()
	}
	owner, err := pluginui.New(pluginui.Options{
		Parent: win, Origin: st.Endpoint.Origin, Cookie: st.Endpoint.Cookie,
		Changed:          func() { win.Update(func() {}) },
		OnFailure:        s.onEmbeddedPageFailure,
		PermissionPolicy: desktopwebview.PermissionPolicy{SessionHost: true, NotificationsAllowed: systemNotificationsSupported}.Decide,
	})
	s.mu.Lock()
	current := s.bootStatus.State == harness.Ready && s.bootStatus.Generation == st.Generation && s.bootStatus.Endpoint == st.Endpoint && s.nativePlugins.client == old.client
	if !current {
		s.mu.Unlock()
		if owner != nil {
			owner.Close()
		}
		return nil, ""
	}
	s.nativePlugins.contributionGeneration = st.Generation
	s.nativePlugins.contributionOrigin, s.nativePlugins.contributionCookie = st.Endpoint.Origin, st.Endpoint.Cookie
	if err != nil {
		s.nativePlugins.contributionError = err.Error()
		s.mu.Unlock()
		return nil, err.Error()
	}
	s.nativePlugins.contributions, s.nativePlugins.contributionError = owner, ""
	s.mu.Unlock()
	return owner, ""
}

func contributionSubject(v manager.Subject, packages []manager.Package) pluginui.Subject {
	subject := pluginui.Subject{Kind: v.Kind, Name: v.Name, RowID: v.RowID, ItemID: v.ItemID}
	if subject.Kind == "package" {
		subject.Kind = "bundle"
	}
	if subject.Kind != "bundle" && subject.Kind != "row" && subject.Kind != "item" {
		return pluginui.Subject{Kind: "list"}
	}
	if subject.Kind == "row" {
		for _, pkg := range packages {
			if pkg.Name == subject.Name {
				for _, row := range pkg.Parts {
					if row.RowID == subject.RowID {
						subject.EntryID = row.EntryID
					}
				}
			}
		}
	}
	return subject
}

func mergePluginContributions(native []managerui.Contribution, owner *pluginui.Owner, snapshot pluginui.Snapshot) []managerui.Contribution {
	out := append([]managerui.Contribution(nil), native...)
	for _, entry := range snapshot.Entries {
		kind, key := "", entry.Key
		switch entry.Slot {
		case "plugins.item":
			kind, key = "item", entry.ID
		case "plugins.bundle.config":
			kind = "bundle"
		case "plugins.row.config":
			kind = "row"
		default:
			continue
		}
		if key == "" {
			continue
		}
		at := -1
		for i, v := range out {
			if v.Kind == kind && v.Key == key {
				at = i
				break
			}
		}
		if at >= 0 {
			// Mature native editors retain the original served namespace writes;
			// their labels and summaries still follow the live slot declaration.
			if entry.Label != "" {
				out[at].Title = entry.Label
			}
			if entry.Summary != "" {
				out[at].Summary = entry.Summary
			}
			out[at].Order = entry.Order
			continue
		}
		label := entry.Label
		if label == "" {
			label = key
		}
		slot := entry.Slot
		out = append(out, managerui.Contribution{Kind: kind, Key: key, Title: label, Summary: entry.Summary, Order: entry.Order, Render: func(c *ui.Context, _ manager.Subject) { renderPluginContributionSurface(c, owner, slot) }})
	}
	return out
}

func renderPluginContributionSurface(c *ui.Context, owner *pluginui.Owner, slot string) {
	if owner == nil {
		return
	}
	snapshot := owner.Snapshot()
	open := ui.Column(c.Key("host-contribution-" + slot)).FillWidth().MinWidth(0).Gap(8)
	open.Children(func() {
		if owner.RenderNativeSlot(c, slot, "", -1) {
			return
		}
		if snapshot.Surface != slot {
			if ui.Button(c, i18n.T(i18n.KeyCmdPluginContentOpen)).Clicked() {
				owner.Show(slot)
			}
			return
		}
		if ui.Button(c, i18n.T(i18n.KeyCmdPluginContentCollapse)).Clicked() {
			owner.Show("")
		}
		height := max(100, min(snapshot.Height, 560))
		ui.Box(c).FillWidth().Height(float32(height)).MinWidth(0).Role(ui.RoleGroup).Label(pluginCopy("插件内容", "Plugin content")).DrawOver(func(p *ui.Painter, r ui.Rect) {
			clip := p.NativeSurfaceClip()
			rect := func(v ui.Rect) mygo.Rectangle {
				return mygo.Rectangle{X: int(math.Ceil(float64(v.X))), Y: int(math.Ceil(float64(v.Y))), Width: max(0, int(math.Floor(float64(v.X+v.W))-math.Ceil(float64(v.X)))), Height: max(0, int(math.Floor(float64(v.Y+v.H))-math.Ceil(float64(v.Y))))}
			}
			owner.Present(rect(r), rect(clip), nil)
		})
	})
}

func pluginCopy(zh, en string) string {
	// Use the same locale owner as the rest of the native Plugins surface.
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}

func pluginContributionNotice(c *ui.Context, owner *pluginui.Owner, message string) {
	pluginModuleFailures(c, owner)
	if message != "" {
		ui.Text(c, fmt.Sprintf(pluginCopy("插件内容未连接：%s", "Plugin content unavailable: %s"), message)).FillWidth().TextColor(c.Theme().Warning)
	}
	if owner != nil {
		snapshot := owner.Snapshot()
		if snapshot.Error != "" {
			ui.Text(c, snapshot.Error).FillWidth().TextColor(c.Theme().Warning)
		}
		if !snapshot.Connected {
			ui.Text(c, pluginCopy("正在连接插件内容…", "Connecting plugin content…")).FillWidth().TextColor(c.Theme().TextMuted)
		}
	}
}
