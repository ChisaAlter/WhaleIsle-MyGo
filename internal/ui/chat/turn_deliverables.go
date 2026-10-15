package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets/diff"
)

// TurnDeliveryView is an explicit-gesture consumer of recorded coordinates.
// Rendering only projects memory. The existing file preview callback is used
// only after revalidating its original presented declaration and Host scope.
func TurnDeliveryView(c *ui.Context, d session.TurnDelivery, m Model, st ToolCardStyle) {
	snapshot := m.Snapshot.TurnFiles
	scope := snapshot.Scope
	enabled := m.HistoryStore != nil && scope.Binding != "" && scope.SessionID == m.Snapshot.SelectedID
	key := fmt.Sprintf("turn-files:%s:%d:%d", scope.Binding, scope.Generation, d.Turn)
	box := ui.Column(c.Key(key)).FillWidth().Gap(6).Padding(10).Border(1, st.Border).Radius(6).Background(st.Field)
	expanded := ui.Local(box, "expanded", func() bool { return false })
	run := func(target session.TurnFileTarget, action string) {
		if enabled {
			go m.HistoryStore.RunTurnFileAction(context.Background(), scope, session.TurnFileRequest{Target: target, Action: action})
		}
	}
	box.Children(func() {
		ui.Text(c, fmt.Sprintf("本轮交付与变更 / Turn %d deliveries and review", d.Turn)).FontSize(12).Bold().TextColor(st.FG)
		ui.Text(c, fmt.Sprintf("Session %s · Host %s", scope.SessionID, snapshot.HostLabel)).FontSize(10).TextColor(st.Muted).Selectable()
		hostTarget := session.TurnFileTarget{}
		if d.Changes != nil {
			hostTarget = *d.Changes
		} else if len(d.Files) > 0 {
			hostTarget = d.Files[0].Target
		}
		hostRequest := session.TurnFileRequest{Target: hostTarget, Action: "host"}
		host := snapshot.Results[hostRequest]
		nativeAvailable := host.Host != nil && host.Host.Available
		ui.Row(c).Gap(6).Children(func() {
			label := "检查 Host 文件打开能力 / Check Host desktop"
			if host.Error != "" {
				label = "重试 Host 信息 / Retry Host info"
			}
			if ui.Button(c, label).FontSize(10).Disabled(!enabled || host.Busy).Clicked() {
				run(hostTarget, "host")
			}
			if host.Busy {
				ui.Text(c, "读取中 / Reading…").FontSize(10).TextColor(st.Muted)
			}
			if host.Host != nil {
				status := host.Host.Name
				if !host.Host.Available {
					status += " · 桌面不可用 / desktop unavailable"
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
			label := fmt.Sprintf("查看本轮变更 / Review turn %d · event %d", target.Turn, target.Seq)
			if state.Error != "" {
				label = "重试读取本轮变更 / Retry reading this turn's changes"
			}
			if ui.Button(c, label).FontSize(11).Disabled(!enabled || state.Busy).Clicked() {
				run(target, "summary")
			}
			if state.Busy {
				ui.Text(c, "正在读取该轮摘要 / Reading this turn's summary…").FontSize(10).TextColor(st.Muted)
			}
			if state.Error != "" {
				ui.Text(c, state.Error).FontSize(10).TextColor(st.Muted)
			}
			if summary := state.Summary; summary != nil {
				ui.Text(c, fmt.Sprintf("%d 个文件 / files · +%d −%d", summary.Total, summary.Added, summary.Deleted)).FontSize(11).TextColor(st.Muted)
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
							label = "重试读取对比 / Retry comparison: " + file.Display
						}
						if ui.Button(c, label).FontSize(11).Disabled(!enabled || comparison.Busy).Clicked() {
							run(selected, "diff")
						}
						ui.Text(c, fmt.Sprintf("%s · event %d / index %d", file.Path, selected.Seq, selected.Index)).FontSize(10).TextColor(st.Muted).Selectable()
						if comparison.Busy {
							ui.Text(c, "读取对比中 / Reading comparison…").FontSize(10).TextColor(st.Muted)
						}
						if comparison.Error != "" {
							ui.Text(c, comparison.Error).FontSize(10).TextColor(st.Muted)
						}
						if content := comparison.Diff; content != nil {
							switch content.Kind {
							case "binary":
								ui.Text(c, "二进制文件，无文本对比 / Binary file; no text comparison").FontSize(11).TextColor(st.Muted)
							case "oversized":
								ui.Text(c, "文件过大，Host 未提供文本对比 / Too large for a text comparison").FontSize(11).TextColor(st.Muted)
							case "text":
								patch := content.Patch()
								if len(patch) > 64<<10 {
									patch = patch[:64<<10]
									if at := strings.LastIndexByte(patch, '\n'); at >= 0 {
										patch = patch[:at]
									}
									ui.Text(c, "对比展示截断为64 KiB / Comparison display limited to 64 KiB").FontSize(10).TextColor(st.Muted)
								}
								if patch == "" {
									ui.Text(c, "无文本差异 / No text changes").FontSize(11).TextColor(st.Muted)
								} else {
									diff.View(c, patch, diff.Style{FontSize: 11, FG: st.FG, Muted: st.Muted, Add: ui.RGB(40, 160, 90), Del: ui.RGB(210, 70, 70), Hunk: st.Accent})
								}
							}
						}
						turnFileNativeActions(c, selected, snapshot, nativeAvailable && enabled, run, st)
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
				if ui.Button(c, "预览 / Preview").FontSize(10).Disabled(!enabled || m.OnOpenDeliverable == nil).Clicked() {
					if path, ok := m.HistoryStore.TurnFilePreviewPath(scope, f.Target); ok {
						m.OnOpenDeliverable(path)
					}
				}
				turnFileNativeActions(c, f.Target, snapshot, nativeAvailable && enabled, run, st)
			})
		}
		hasMore := len(d.Files) > 4
		if d.Changes != nil {
			if sum := snapshot.Results[session.TurnFileRequest{Target: *d.Changes, Action: "summary"}].Summary; sum != nil && len(sum.Files) > 4 {
				hasMore = true
			}
		}
		if hasMore {
			label := "展开全部文件 / Show all files"
			if *expanded {
				label = "收起 / Collapse"
			}
			if ui.Button(c, label).FontSize(10).Clicked() {
				*expanded = !*expanded
			}
		}
	})
}
func turnFileNativeActions(c *ui.Context, target session.TurnFileTarget, snapshot session.TurnFilesSnapshot, enabled bool, run func(session.TurnFileTarget, string), st ToolCardStyle) {
	state := snapshot.Results[(session.TurnFileRequest{Target: target, Action: "open"}).StateKey()]
	ui.Row(c).Gap(6).Children(func() {
		for _, action := range []string{"open", "reveal"} {
			label := "在 Host 打开 / Open on Host"
			if action == "reveal" {
				label = "在 Host 定位 / Reveal on Host"
			}
			if state.Error != "" && state.Action == action {
				label = "重试文件动作 / Retry file " + action
			}
			if ui.Button(c, label).FontSize(10).Disabled(!enabled || state.Busy).Clicked() {
				run(target, action)
			}
		}
	})
	if state.Busy {
		ui.Text(c, state.Action+"…").FontSize(10).TextColor(st.Muted)
	}
	if state.Error != "" {
		ui.Text(c, state.Error).FontSize(10).TextColor(st.Muted)
	}
	if state.Ready {
		ui.Text(c, "Host 已接受文件动作 / Host accepted file "+state.Action).FontSize(10).TextColor(st.Muted)
	}
}
