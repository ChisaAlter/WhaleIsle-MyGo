package im

import (
	"context"
	"github.com/egoist/mygo/ui"
	"strings"
	"sync"
	"time"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/imonboard"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type botDirectoryPicker struct {
	mu                    sync.Mutex
	open, loading, hidden bool
	submitted             bool
	request               uint64
	listing               *api.DirectoryListing
	err, confirm          string
}

func botWorkspacePicker(c *ui.Context, m Model, v imonboard.View, b imonboard.Bot, disabled bool, fg, muted ui.Color) {
	controls := ui.Row(c).Gap(8)
	state := ui.Local(controls, "workspace-picker", func() botDirectoryPicker { return botDirectoryPicker{} })
	state.mu.Lock()
	if state.submitted && !v.Busy {
		state.submitted = false
		if state.confirm == b.Workspace {
			state.confirm, state.err = "", ""
		} else if v.Error != "" {
			state.err = v.Error
		}
	}
	state.mu.Unlock()
	load := func(path string) {
		state.mu.Lock()
		state.open = true
		state.loading = true
		state.err = ""
		state.request++
		request := state.request
		state.mu.Unlock()
		services := c.Services()
		services.Invalidate()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			listing, err := m.Store.ListDirectories(ctx, path)
			picked := ""
			nativeFallback := false
			if err != nil && strings.Contains(err.Error(), "directory-picker-unavailable") {
				nativeFallback = true
				picked, err = m.Store.PickDirectory(ctx)
			}
			if err != nil && path != "" && strings.Contains(err.Error(), "directory-unreadable") {
				listing, err = m.Store.ListDirectories(ctx, "")
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.request != request || m.Store.Onboarding.Snapshot().Scope != v.Scope {
				return
			}
			state.loading = false
			if err != nil {
				state.err = err.Error()
			} else if picked != "" {
				state.open = false
				state.confirm = picked
				m.Store.Onboarding.EditWorkspace(v.Scope, b.BotID, picked)
			} else if nativeFallback {
				state.open = false
			} else {
				state.listing = &listing
			}
			services.Invalidate()
		}()
	}
	state.mu.Lock()
	open, loading, hidden, listing, errText, confirm := state.open, state.loading, state.hidden, state.listing, state.err, state.confirm
	state.mu.Unlock()
	controls.Children(func() {
		if ui.Button(c, "选择目录…").Disabled(disabled).Clicked() {
			path := b.Workspace
			if draft, ok := v.WorkspaceDrafts[b.BotID]; ok {
				path = draft
			}
			load(path)
		}
		workspace := b.Workspace
		if draft, ok := v.WorkspaceDrafts[b.BotID]; ok {
			workspace = draft
		}
		if ui.Button(c, "保存工作区…").Disabled(disabled || strings.TrimSpace(workspace) == "" || strings.TrimSpace(workspace) == b.Workspace).Clicked() {
			state.mu.Lock()
			state.confirm = strings.TrimSpace(workspace)
			state.mu.Unlock()
		}
	})
	if open {
		visible := true
		ui.DialogBase(c.Key("bot-workspace-browser"), &visible, func(backdrop, panel ui.Element) {
			_, height := c.Size()
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 560, 16).Height(min(float32(620), height-24)).Gap(8).Children(func() {
				ui.Text(c, "选择机器人工作目录").Bold().TextColor(fg)
				if loading {
					ui.Text(c, "正在读取目录…").TextColor(muted)
				}
				if errText != "" {
					ui.Text(c, errText).TextColor(fg).Selectable()
					if ui.Button(c, "重试").Disabled(loading).Clicked() {
						path := ""
						if listing != nil {
							path = listing.Path
						}
						load(path)
					}
				}
				if listing != nil {
					ui.Text(c, listing.Path).TextColor(fg).Selectable()
					ui.Row(c).Wrap().Gap(6).Children(func() {
						for _, crumb := range listing.Crumbs {
							crumb := crumb
							if ui.Button(c, crumb.Name).Disabled(loading).Clicked() {
								load(crumb.Path)
							}
						}
					})
					if ui.Button(c, "显示 / 隐藏隐藏目录").Disabled(loading).Clicked() {
						state.mu.Lock()
						state.hidden = !state.hidden
						state.mu.Unlock()
					}
					ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
						ui.Column(c).FillWidth().Gap(4).Children(func() {
							for _, entry := range listing.Entries {
								entry := entry
								if entry.Hidden && !hidden {
									continue
								}
								if ui.Button(c, entry.Name).FillWidth().Disabled(loading).Clicked() {
									load(entry.Path)
								}
							}
						})
					})
					if listing.Truncated {
						ui.Text(c, "目录条目较多，当前列表已截断。").FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
				}
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, "使用此目录").Disabled(loading || listing == nil || listing.Path == "" || disabled).Clicked() {
						state.mu.Lock()
						state.confirm = listing.Path
						state.open = false
						state.mu.Unlock()
						m.Store.Onboarding.EditWorkspace(v.Scope, b.BotID, listing.Path)
					}
					if ui.Button(c, "取消").Clicked() {
						visible = false
					}
				})
			})
		})
		if !visible {
			state.mu.Lock()
			state.open = false
			state.request++
			state.mu.Unlock()
		}
	}
	if confirm != "" {
		visible := true
		ui.DialogBaseWithOptions(c.Key("bot-workspace-confirm"), &visible, ui.DialogOptions{DismissDisabled: v.Busy}, func(backdrop, panel ui.Element) {
			_, height := c.Size()
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 500, 16).MaxHeight(height - 24).Gap(10).Children(func() {
				ui.Text(c, "切换机器人工作目录？").Bold().TextColor(fg)
				ui.Text(c, confirm).TextColor(fg).Selectable()
				ui.Text(c, "此机器人的旧聊天绑定会清除；原会话内容保留。下一条消息将使用新工作目录。").TextColor(muted)
				if errText != "" {
					ui.Text(c, errText).FillWidth().TextColor(c.Theme().Danger).Selectable()
				}
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, "确认切换").Disabled(disabled).Clicked() {
						m.Store.Onboarding.EditWorkspace(v.Scope, b.BotID, confirm)
						state.mu.Lock()
						state.submitted, state.err = true, ""
						state.mu.Unlock()
						m.Store.Onboarding.SaveWorkspace(v.Scope, b.BotID, confirm)
					}
					if ui.Button(c, "取消").Disabled(v.Busy).Clicked() {
						visible = false
					}
				})
			})
		})
		if !visible {
			state.mu.Lock()
			state.confirm = ""
			state.err = ""
			state.mu.Unlock()
		}
	}
}
