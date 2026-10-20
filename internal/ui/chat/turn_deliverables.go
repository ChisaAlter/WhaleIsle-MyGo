package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets/diff"
	"whaleisle/internal/ui/i18n"
)

// TurnDeliveryView observes the recorded summary and desktop metadata once per
// mounted owner. File handoffs and review navigation remain explicit gestures,
// revalidated against the original declaration and captured Host scope.
func TurnDeliveryView(c *ui.Context, d session.TurnDelivery, m Model, st ToolCardStyle) {
	if m.DisableDeveloperTools {
		d.Changes = nil
		if len(d.Files) == 0 {
			return
		}
	}
	snapshot := m.Snapshot.TurnFiles
	scope := snapshot.Scope
	enabled := m.HistoryStore != nil && scope.Binding != "" && scope.SessionID == m.Snapshot.SelectedID
	preview := func(target session.TurnFileTarget) {
		if path, ok := m.HistoryStore.TurnFilePreviewPath(scope, target); ok && m.OnOpenDeliverable != nil {
			m.OnOpenDeliverable(path)
		}
	}
	run := func(target session.TurnFileTarget, action string) {
		if enabled {
			if m.OnOpenChangesReview != nil && target.Kind == "changes" && (action == "summary" || action == "diff") {
				m.OnOpenChangesReview(scope, target)
				return
			}
			go m.HistoryStore.RunTurnFileAction(context.Background(), scope, session.TurnFileRequest{Target: target, Action: action})
		}
	}
	if m.OnOpenDeliverable == nil {
		preview = nil
	}
	AddressedTurnDeliveryView(c, d, snapshot, enabled, run, preview, st, func(target session.TurnFileTarget, action string) {
		go m.HistoryStore.RunTurnFileAction(context.Background(), scope, session.TurnFileRequest{Target: target, Action: action})
	})
}

func AddressedTurnDeliveryView(c *ui.Context, d session.TurnDelivery, snapshot session.TurnFilesSnapshot, enabled bool, run func(session.TurnFileTarget, string), preview func(session.TurnFileTarget), st ToolCardStyle, readers ...func(session.TurnFileTarget, string)) {
	scope := snapshot.Scope
	key := fmt.Sprintf("turn-files:%s:%d:%d", scope.Binding, scope.Generation, d.Turn)
	seat := ui.Box(c.Key(key + ":reads")).Width(0).Height(0)
	requested := ui.Local(seat, "requested", func() map[session.TurnFileRequest]bool { return make(map[session.TurnFileRequest]bool) })
	read := run
	if len(readers) > 0 {
		read = readers[0]
	}
	ensure := func(target session.TurnFileTarget, action string) {
		request := session.TurnFileRequest{Target: target, Action: action}
		if _, exists := snapshot.Results[request]; enabled && read != nil && !exists && !(*requested)[request] {
			(*requested)[request] = true
			read(target, action)
		}
	}
	if d.Changes != nil {
		ensure(*d.Changes, "summary")
	}
	hostTarget := session.TurnFileTarget{}
	if d.Changes != nil {
		hostTarget = *d.Changes
	} else if len(d.Files) > 0 {
		hostTarget = d.Files[0].Target
	}
	if d.Changes != nil || len(d.Files) > 0 {
		ensure(hostTarget, "host")
	}
	// Like the original turn tail, an absent/expired/empty summary contributes
	// no changed-files card. Explicit presented files remain independently usable.
	if len(d.Files) == 0 {
		if d.Changes == nil {
			return
		}
		summary := snapshot.Results[session.TurnFileRequest{Target: *d.Changes, Action: "summary"}].Summary
		if summary == nil || len(summary.Files) == 0 {
			return
		}
	}
	box := ui.Column(c.Key(key)).FillWidth().Gap(6).Padding(10).Border(1, st.Border).Radius(6).Background(st.Field)
	expanded := ui.Local(box, "expanded", func() bool { return false })
	box.Children(func() {
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatDeliveryTitle), d.Turn)).FontSize(12).Bold().TextColor(st.FG)
		ui.Text(c, fmt.Sprintf("Session %s · Host %s", scope.SessionID, snapshot.HostLabel)).FontSize(10).TextColor(st.Muted).Selectable()
		hostRequest := session.TurnFileRequest{Target: hostTarget, Action: "host"}
		host := snapshot.Results[hostRequest]
		nativeAvailable := host.Host != nil && host.Host.Available
		ui.Row(c).Gap(6).Children(func() {
			label := i18n.T(i18n.KeyChatDeliveryCheckHost)
			if host.Error != "" {
				label = i18n.T(i18n.KeyChatDeliveryRetryHost)
			}
			if ui.Button(c, label).FontSize(10).Disabled(!enabled || host.Busy).Clicked() {
				run(hostTarget, "host")
			}
			if host.Busy {
				ui.Text(c, i18n.T(i18n.KeyChatResourceReading)).FontSize(10).TextColor(st.Muted)
			}
			if host.Host != nil {
				status := host.Host.Name
				if !host.Host.Available {
					status += " · " + i18n.T(i18n.KeyChatDeliveryNoDesktop)
				}
				ui.Text(c, status).FontSize(10).TextColor(st.Muted)
			}
		})
		if host.Error != "" {
			ui.Text(c, host.Error).FontSize(10).TextColor(st.Muted)
		}
		if d.Changes != nil {
			target := *d.Changes
			request := session.TurnFileRequest{Target: target, Action: "summary"}
			state := snapshot.Results[request]
			label := fmt.Sprintf(i18n.T(i18n.KeyChatDeliveryReview), target.Turn, target.Seq)
			if state.Error != "" {
				label = i18n.T(i18n.KeyChatDeliveryRetryReview)
			}
			if ui.Button(c, label).FontSize(11).Disabled(!enabled || state.Busy).Clicked() {
				run(target, "summary")
			}
			if state.Busy {
				ui.Text(c, i18n.T(i18n.KeyChatDeliveryReadSummary)).FontSize(10).TextColor(st.Muted)
			}
			if state.Error != "" {
				ui.Text(c, state.Error).FontSize(10).TextColor(st.Muted)
			}
			if summary := state.Summary; summary != nil {
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatDeliveryFiles), summary.Total, summary.Added, summary.Deleted)).FontSize(11).TextColor(st.Muted)
				rows := summary.Files
				if !*expanded && len(rows) > 4 {
					rows = rows[:4]
				}
				for index, file := range rows {
					selected := target
					selected.Index = index
					comparison := snapshot.Results[session.TurnFileRequest{Target: selected, Action: "diff"}]
					ui.Column(c.Key(fmt.Sprintf("changed:%d", index))).FillWidth().Gap(4).Children(func() {
						label := fmt.Sprintf("Review %s · +%d −%d", file.Display, file.Added, file.Deleted)
						if file.Binary {
							label += " · binary"
						}
						if file.Oversized {
							label += " · oversized"
						}
						if comparison.Error != "" {
							label = i18n.T(i18n.KeyChatDeliveryRetryDiff) + ": " + file.Display
						}
						if ui.Button(c, label).FontSize(11).Disabled(!enabled || comparison.Busy).Clicked() {
							run(selected, "diff")
						}
						ui.Text(c, fmt.Sprintf("%s · event %d / index %d", file.Path, selected.Seq, selected.Index)).FontSize(10).TextColor(st.Muted).Selectable()
						if comparison.Busy {
							ui.Text(c, i18n.T(i18n.KeyChatDeliveryReadDiff)).FontSize(10).TextColor(st.Muted)
						}
						if comparison.Error != "" {
							ui.Text(c, comparison.Error).FontSize(10).TextColor(st.Muted)
						}
						if content := comparison.Diff; content != nil {
							switch content.Kind {
							case "binary":
								ui.Text(c, i18n.T(i18n.KeyChatDeliveryBinary)).FontSize(11).TextColor(st.Muted)
							case "oversized":
								ui.Text(c, i18n.T(i18n.KeyChatDeliveryTooLarge)).FontSize(11).TextColor(st.Muted)
							case "text":
								patch := content.Patch()
								if len(patch) > 64<<10 {
									patch = patch[:64<<10]
									if at := strings.LastIndexByte(patch, '\n'); at >= 0 {
										patch = patch[:at]
									}
									ui.Text(c, i18n.T(i18n.KeyChatDeliveryDiffClip)).FontSize(10).TextColor(st.Muted)
								}
								if patch == "" {
									ui.Text(c, i18n.T(i18n.KeyChatToolNoDiff)).FontSize(11).TextColor(st.Muted)
								} else {
									diff.View(c, patch, diff.Style{FontSize: 11, FG: st.FG, Muted: st.Muted, Add: ui.RGB(40, 160, 90), Del: ui.RGB(210, 70, 70), Hunk: st.Accent})
								}
							}
						}
						TurnFileNativeActions(c, selected, snapshot, nativeAvailable && enabled, run, st)
					})
				}
			}
		}
		rows := d.Files
		if !*expanded && len(rows) > 4 {
			rows = rows[:4]
		}
		for _, file := range rows {
			f := file
			ui.Column(c.Key(fmt.Sprintf("presented:%d:%d", f.Target.Seq, f.Target.Index))).FillWidth().Gap(4).Children(func() {
				ui.Text(c, f.Path).FontSize(12).TextColor(st.FG).Selectable()
				if f.Description != "" {
					ui.Text(c, f.Description).FontSize(11).TextColor(st.Muted).Selectable()
				}
				ui.Text(c, fmt.Sprintf("event %d / index %d", f.Target.Seq, f.Target.Index)).FontSize(10).TextColor(st.Muted)
				if ui.Button(c, i18n.T(i18n.KeyChatResourcePreview)).FontSize(10).Disabled(!enabled || preview == nil).Clicked() {
					preview(f.Target)
				}
				TurnFileNativeActions(c, f.Target, snapshot, nativeAvailable && enabled, run, st)
			})
		}
		hasMore := len(d.Files) > 4
		if d.Changes != nil {
			if sum := snapshot.Results[session.TurnFileRequest{Target: *d.Changes, Action: "summary"}].Summary; sum != nil && len(sum.Files) > 4 {
				hasMore = true
			}
		}
		if hasMore {
			label := i18n.T(i18n.KeyChatDeliveryShowAll)
			if *expanded {
				label = i18n.T(i18n.KeyChatRowCollapse)
			}
			if ui.Button(c, label).FontSize(10).Clicked() {
				*expanded = !*expanded
			}
		}
	})
}

// TurnFileNativeActions shares the Files preview's default-open/application-menu
// interaction, while retaining recorded session/sequence/index route authority.
func TurnFileNativeActions(c *ui.Context, target session.TurnFileTarget, snapshot session.TurnFilesSnapshot, enabled bool, run func(session.TurnFileTarget, string), st ToolCardStyle) {
	state := snapshot.Results[(session.TurnFileRequest{Target: target, Action: "open"}).StateKey()]
	apps, exists := snapshot.Results[session.TurnFileRequest{Target: target, Action: "applications"}]
	seat := ui.Box(c.Key(fmt.Sprintf("native-apps:%s:%d:%v", snapshot.Scope.Binding, snapshot.Scope.Generation, target))).Width(0).Height(0)
	requested := ui.Local(seat, "requested", func() bool { return false })
	if enabled && run != nil && !exists && !*requested {
		*requested = true
		run(target, "applications")
	}
	ui.Row(c).Gap(6).Children(func() {
		for _, action := range []string{"open", "reveal"} {
			label := i18n.T(i18n.KeyChatDeliveryOpenHost)
			if action == "reveal" {
				label = i18n.T(i18n.KeyChatDeliveryRevealHost)
			}
			if state.Error != "" && state.Action == action {
				label = i18n.T(i18n.KeyChatDeliveryRetryAction) + " " + action
			}
			if ui.Button(c, label).FontSize(10).Disabled(!enabled || state.Busy).Clicked() {
				run(target, action)
			}
		}
		ui.Button(c, "Open with…").FontSize(10).Disabled(!enabled || state.Busy || apps.Busy).Menu(func(menu *ui.Menu) {
			for _, app := range apps.Applications {
				caption := strings.TrimSpace(app.Name)
				if caption == "" {
					caption = app.ID
				}
				if app.Default {
					caption += " (default)"
				}
				if menu.Item(caption).Chosen() {
					run(target, session.TurnFileApplicationAction(app.ID))
				}
			}
			if menu.Item("Refresh applications").Chosen() {
				run(target, "applications")
			}
		})
	})
	if apps.Busy {
		ui.Text(c, "Loading applications…").FontSize(10).TextColor(st.Muted)
	}
	if apps.Error != "" {
		ui.Text(c, apps.Error).FontSize(10).TextColor(st.Muted)
	}
	if state.Busy {
		ui.Text(c, state.Action+"…").FontSize(10).TextColor(st.Muted)
	}
	if state.Error != "" {
		ui.Text(c, state.Error).FontSize(10).TextColor(st.Muted)
	}
	if state.Ready {
		ui.Text(c, i18n.T(i18n.KeyChatDeliveryAccepted) + " "+state.Action).FontSize(10).TextColor(st.Muted)
	}
}
