package chat

import (
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

// VisualReplyView sits outside ToolCardView's disclosure. HTML is always plain
// native text until interaction mounts a lifetime-bound AppContainer child.
// The chat process never loads the HTML or executes its scripts.
func VisualReplyView(c *ui.Context, p session.VisualReplyPublication, m Model, fg, muted, border, field ui.Color) {
	id := p.File.AttachmentID
	state := m.VisualReply
	active := state.SessionID == m.Snapshot.SelectedID && state.AttachmentID == id
	ui.Column(c.Key("published-html-"+id)).FillWidth().Gap(6).Padding(10).Background(field).Border(1, border).Radius(6).Children(func() {
		ui.Text(c, p.Title).FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "已发布 HTML / Published HTML").FontSize(11).TextColor(muted)
		ui.Row(c).Gap(6).Children(func() {
			if m.OnInteractVisualReply != nil && m.OnInlineVisualReply == nil {
				if ui.Button(c, "在卡片中交互 / Interact inline").Disabled(active && (state.Loading || state.InteractiveLoading || state.InteractiveActive)).
					Tooltip("在隔离进程中的卡片表面运行控件和脚本；不开放网络、本机文件、应用权限或持久登录。 / Runs this page in an isolated process without network, local files or application access.").Clicked() {
					m.OnInteractVisualReply(id)
				}
			}
			if active && (state.InteractiveLoading || state.InteractiveActive) && m.OnStopInteractiveVisualReply != nil {
				if ui.Button(c, "停止交互 / Stop interaction").Clicked() {
					m.OnStopInteractiveVisualReply()
				}
			}
			if ui.Button(c, "安全图片预览 / Safe image preview").Disabled(m.OnPreviewVisualReply == nil || (active && (state.Loading || state.PreviewLoading))).Clicked() {
				m.OnPreviewVisualReply(id)
			}
			if ui.Button(c, "查看完整源码 / View source").Disabled(m.OnReadVisualReply == nil || (active && state.Loading)).Clicked() {
				m.OnReadVisualReply(id)
			}
			if active && state.Ready {
				if ui.Button(c, "另存 HTML… / Save as…").Disabled(m.OnSaveVisualReply == nil).Clicked() {
					m.OnSaveVisualReply(id)
				}
			}
			if active && m.OnCloseVisualReply != nil {
				if ui.Button(c, "关闭 / Close").Clicked() {
					m.OnCloseVisualReply()
				}
			}
		})
		if m.OnInlineVisualReply != nil {
			inlineVisualReplyView(c, p, m, m.OnInlineVisualReply(id), fg, muted, field)
		}
		if !active {
			return
		}
		if state.Loading {
			ui.Text(c, "正在授权读取… / Reading published page…").TextColor(muted)
		}
		if state.Error != "" {
			ui.Text(c, state.Error).TextColor(muted)
		}
		if state.InteractiveLoading {
			ui.Text(c, "正在准备隔离交互表面… / Preparing isolated interactive surface…").TextColor(muted)
		}
		if state.InteractiveActive {
			if state.Surface != nil {
				c.After(100 * time.Millisecond)
				expanded := state.InteractiveExpanded
				if ui.Button(c, "展开 / Expand").Disabled(m.HistoryStore == nil).Clicked() {
					m.HistoryStore.ToggleVisualReplyExpanded()
				}
				if !expanded {
					visualReplySurfaceSeat(c, state.Surface, m.ThemeID, m.SystemDark, 360)
				}
				if expanded {
					open := true
					ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
						panel.Width(960).MaxWidthPercent(94).Padding(12).Background(field).Children(func() {
							ui.Row(c).FillWidth().Children(func() {
								ui.Text(c, p.Title).Grow(1).TextColor(fg)
								if ui.Button(c, "收起 / Collapse").Clicked() {
									open = false
								}
							})
							visualReplySurfaceSeat(c, state.Surface, m.ThemeID, m.SystemDark, 600)
						})
					})
					if !open && m.HistoryStore != nil {
						m.HistoryStore.ToggleVisualReplyExpanded()
					}
				}
			}
		}
		if state.InteractiveError != "" {
			ui.Text(c, "交互页未启动或已停止，源码仍可查看或另存 / Interactive page unavailable: "+state.InteractiveError).TextColor(muted)
		}
		if state.PreviewLoading {
			ui.Text(c, "正在检查隔离环境并生成静态预览… / Checking isolation and rendering a static preview…").TextColor(muted)
		}
		if state.PreviewError != "" {
			ui.Text(c, "安全预览不可用，仍可查看源码或另存 / Safe preview unavailable: "+state.PreviewError).TextColor(muted)
		}
		if m.VisualReplyBitmap != nil && state.PreviewID != 0 {
			ui.Image(c, m.VisualReplyBitmap).FillWidth().AspectRatio(float32(state.PreviewWidth) / float32(state.PreviewCapturedHeight)).Fit(ui.Contain)
			ui.Text(c, "此图为静态 PNG；请在卡片交互表面操作 / This image is static; use the interactive surface for controls").FontSize(11).TextColor(muted)
			if state.PreviewCapturedHeight < state.PreviewContentHeight {
				ui.Text(c, "页面超过4000像素，预览未覆盖底部 / Preview is capped at4000 pixels; the bottom is not shown.").FontSize(11).TextColor(muted)
			}
		}
		if state.Ready {
			source := state.HTML
			ui.TextArea(c.Key("html-source"), &source).ReadOnly(true).FillWidth().Lines(8, 20).FontSize(12).TextColor(fg)
			ui.Text(c, "源码视图为纯文本；安全图片预览为 PNG，交互页在隔离进程运行。另存后在外部浏览器打开可能运行脚本或访问网络。 / Source is plain text; image preview is PNG, and interaction uses a separate isolated process. Opening a saved page externally may run scripts or access the network.").FontSize(11).TextColor(muted)
		}
	})
}

func visualReplySurfaceSeat(c *ui.Context, surface session.VisualReplySurface, themeID string, dark bool, height float32) {
	resolved := theme.Resolve(themeID)
	colors := resolved.Active(dark)
	appearance := "light"
	if resolved.Scheme == theme.SchemeDark || (resolved.Scheme == theme.SchemeSystem && dark) {
		appearance = "dark"
	}
	if pixels, ok := surface.(interface {
		RenderPixels(*ui.Context, float32, string, map[string]string) bool
	}); ok {
		if pixels.RenderPixels(c, height, appearance, map[string]string{"--bg": colors.BG, "--background": colors.BG, "--surface": colors.Field, "--text": colors.FG, "--text-secondary": colors.Muted, "--border": colors.Border, "--accent": colors.Accent}) {
			return
		}
	}
	ui.Box(c).FillWidth().Height(height).DrawOver(func(p *ui.Painter, r ui.Rect) {
		clip := p.NativeSurfaceClip()
		_ = surface.Present(int(r.X), int(r.Y), int(r.W), int(r.H), int(clip.X), int(clip.Y), int(clip.W), int(clip.H), appearance, map[string]string{"--bg": colors.BG, "--background": colors.BG, "--surface": colors.Field, "--text": colors.FG, "--text-secondary": colors.Muted, "--border": colors.Border, "--accent": colors.Accent})
	})
}

func inlineVisualReplyView(c *ui.Context, p session.VisualReplyPublication, m Model, page session.InlineVisualReply, fg, muted, field ui.Color) {
	if page.Loading {
		ui.Text(c, "正在加载交互页面… / Loading interactive page…").TextColor(muted)
		return
	}
	if page.Error != "" {
		ui.Text(c, page.Error).TextColor(muted)
		if page.Retry != nil && ui.Button(c, "重试 / Retry").Clicked() {
			page.Retry()
		}
		return
	}
	if page.Surface == nil {
		return
	}
	c.After(100 * time.Millisecond)
	height := float32(max(80, min(2000, page.Height)))
	if !page.Expanded {
		if ui.Button(c, "展开 / Expand").Clicked() {
			page.ToggleExpanded()
		}
		visualReplySurfaceSeat(c, page.Surface, m.ThemeID, m.SystemDark, height)
		return
	}
	// Preserve the inline layout while moving the same child to the dialog seat.
	ui.Box(c).FillWidth().Height(height)
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		panel.Width(960).MaxWidthPercent(94).Padding(12).Background(field).Children(func() {
			ui.Row(c).FillWidth().Children(func() {
				ui.Text(c, p.Title).Grow(1).TextColor(fg)
				if ui.Button(c, "收起 / Collapse").Clicked() {
					open = false
				}
			})
			visualReplySurfaceSeat(c, page.Surface, m.ThemeID, m.SystemDark, 600)
		})
	})
	if !open {
		page.ToggleExpanded()
	}
}
