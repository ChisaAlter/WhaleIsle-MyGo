package terminal

import (
	"fmt"

	"github.com/egoist/mygo"
	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func HostTitle() string { return terminalLabel("Host 终端", "Host terminal") }

// HostState owns only the native renderer. Host process and hold ownership live
// in the Store, independently of this view or the local PTY drawer.
// All HostState methods run on the UI thread.
type HostState struct {
	occurrences     map[string]*HostState
	terminal        *myterm.Terminal
	attachment      *hostterminal.Attachment
	closeTarget     string
	confirmClose    bool
	renderError     string
	selectionNotice string
}

func (h *HostState) drop() {
	if h.terminal != nil {
		_ = h.terminal.Close()
	}
	h.terminal = nil
	h.attachment = nil
	h.renderError = ""
	h.selectionNotice = ""
}
func (h *HostState) Observe(store *hostterminal.Store, session string, visible bool) {
	if store == nil {
		return
	}
	store.Show(session, visible)
	if !visible {
		h.hideOccurrences(store, nil)
		h.drop()
		h.confirmClose = false
		h.closeTarget = ""
	}
}

// HostView offers list/create/connect/close actions. Only StartFresh, carried
// from an explicit new-tab action, creates on mount; ordinary recovery does not.
// Options.Conn prevents a local PTY fallback.
func HostView(c *ui.Context, h *HostState, store *hostterminal.Store, session, themeID string, dark bool, actionOptions ...HostActions) {
	var actions HostActions
	if len(actionOptions) > 0 {
		actions = actionOptions[0]
	}
	if h == nil || store == nil {
		return
	}
	h.Observe(store, session, true)
	v := store.Snapshot(session)
	if actions.StartFresh != nil && *actions.StartFresh && store.StartFresh(v.Owner, session) {
		*actions.StartFresh = false
		v = store.Snapshot(session)
	}
	colors := theme.Resolve(themeID).Active(dark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	target := fmt.Sprintf("%d:%s:%s", v.Owner, session, v.ActiveID)
	if target != h.closeTarget {
		h.closeTarget = target
		h.confirmClose = false
	}
	if h.attachment != v.Attachment {
		h.drop()
		h.attachment = v.Attachment
	}
	if v.Render != nil && v.Attachment != nil {
		r := v.Render
		if r.Snapshot && h.terminal == nil {
			if err := prepareNativeRuntime(); err != nil {
				h.renderError = err.Error()
				store.RenderFailed(v.Attachment, err)
			} else {
				scrollback := -1
				if v.Environment.Scrollback > 0 {
					scrollback = min(20<<20, v.Environment.Scrollback*max(80, v.Environment.MaxCols)*4)
				}
				light, darkTheme := myterm.LightTheme(), myterm.DarkTheme()
				copyOwner, copyID, copyAttachment := v.Owner, v.ActiveID, v.Attachment
				opts := myterm.Options{Conn: v.Attachment, ReadOnly: true, CanCopy: func() bool { return store.CanCopySelection(copyOwner, session, copyID, copyAttachment) }, Theme: light, DarkTheme: darkTheme, Scrollback: scrollback}
				if dark {
					opts.Theme = darkTheme
				}
				var err error
				h.terminal, err = myterm.New(opts)
				if err != nil {
					h.renderError = err.Error()
					store.RenderFailed(v.Attachment, err)
				}
			}
		}
		if h.terminal != nil {
			h.terminal.SetReadOnly(!v.Writable)
			if r.Snapshot {
				h.terminal.Resize(r.Cols, r.Rows)
			}
			h.terminal.Feed([]byte(r.Data))
			store.Acknowledge(v.Attachment, r.Revision)
		}
		v = store.Snapshot(session)
	}
	ui.Column(c).Fill().MinHeight(0).Gap(5).Padding(6, 8, 6, 8).Children(func() {
		ui.Text(c, terminalLabel("Host 保留终端", "Retained Host terminal")).Bold().TextColor(fg)
		ui.Text(c, terminalLabel("终端归当前 Host 和会话所有，隐藏后保持。新建会以 Host 系统用户权限启动 shell。", "Terminals belong to this Host and session and survive hiding. New starts a shell with the Host system user's permissions.")).FontSize(11).TextColor(muted)
		ui.Row(c).Gap(4).Children(func() {
			if ui.Button(c, terminalLabel("刷新列表", "Refresh list")).FontSize(11).Disabled(!v.ClientReady || session == "" || v.Busy).Clicked() {
				store.Refresh(v.Owner, session)
			}
			if ui.Button(c, terminalLabel("新建 Host 终端", "New Host terminal")).FontSize(11).Disabled(!v.Ready || session == "" || v.Busy).Clicked() {
				store.Create(v.Owner, session)
			}
		})
		hostShellView(c, store, v, session, muted)
		if !v.ClientReady || session == "" {
			ui.Text(c, terminalLabel("请选择已连接 Host 的会话。", "Select a session on a connected Host.")).FontSize(11).TextColor(muted)
			return
		}
		if len(v.Items) > 0 {
			ui.Scroll(c).Height(72).FillWidth().Children(func() {
				ui.Column(c).Gap(3).Children(func() {
					for _, item := range v.Items {
						id := item.ID
						title := item.Title
						if title == "" {
							title = id
						}
						if id == v.ActiveID {
							title = "▸ " + title
						}
						if ui.Button(c, terminalLabel("连接：", "Connect: ")+title).FontSize(11).Disabled(!v.Ready || v.Busy).Clicked() {
							store.Connect(v.Owner, session, id)
						}
					}
				})
			})
		}
		if v.ActiveID == "" {
			ui.Text(c, terminalLabel("刷新可查找现有终端；连接只恢复所选身份，不会替换丢失的进程。", "Refresh finds existing terminals. Connect restores the selected identity and never replaces a missing process.")).FontSize(11).TextColor(muted)
		} else {
			status := v.Phase
			if v.Info.State == "exited" {
				status = terminalLabel("进程已退出", "Process exited")
				if v.Info.ExitCode != nil {
					status += fmt.Sprintf(" (%d)", *v.Info.ExitCode)
				}
			}
			if v.Writable {
				status += " · " + terminalLabel("可输入", "Writable")
			} else {
				status += " · " + terminalLabel("只读", "Read-only")
			}
			ui.Text(c, status).FontSize(11).TextColor(muted)
			ui.Text(c, fmt.Sprintf("%s · %s", v.ActiveID, v.Info.Cwd)).FontSize(10).TextColor(muted)
			hostRenameView(c, store, v, session, muted)
			ui.Row(c).Gap(4).Children(func() {
				if ui.Button(c, terminalLabel("重连 / 接管输入", "Reconnect / take control")).FontSize(11).Disabled(!v.Ready || v.Busy || v.Phase == "closing" || v.Phase == "closed").Clicked() {
					store.Connect(v.Owner, session, v.ActiveID)
				}
				if ui.Button(c, terminalLabel("关闭 Host 进程", "Close Host process")).FontSize(11).Disabled(!v.Ready || v.Phase == "closing" || v.Phase == "closed" || v.Busy).Clicked() {
					h.confirmClose = true
				}
			})
		}
		if h.confirmClose {
			ui.Text(c, terminalLabel("将终止此 Host 会话中选中的终端进程。", "This terminates the selected process in this Host session.")).FontSize(11).TextColor(fg)
			ui.Row(c).Gap(4).Children(func() {
				if ui.Button(c, terminalLabel("确认关闭", "Confirm close")).FontSize(11).Clicked() {
					h.confirmClose = false
					store.Close(v.Owner, session, v.ActiveID)
				}
				if ui.Button(c, terminalLabel("取消", "Cancel")).FontSize(11).Clicked() {
					h.confirmClose = false
				}
			})
		}
		if v.Error != "" {
			ui.Text(c, v.Error).FontSize(11).TextColor(muted)
		}
		if h.renderError != "" && h.renderError != v.Error {
			ui.Text(c, h.renderError).FontSize(11).TextColor(muted)
		}
		if h.terminal != nil {
			h.terminal.SetReadOnly(!v.Writable)
			selection, selected := h.terminal.SelectionText()
			if ui.Button(c, terminalLabel("复制选区", "Copy selection")).FontSize(10).Disabled(!selected || selection == "").Clicked() {
				if store.CanCopySelection(v.Owner, session, v.ActiveID, h.attachment) {
					// Read again at the action, never substitute the full screen.
					if text, ok := h.terminal.SelectionText(); ok && text != "" {
						mygo.Clipboard.WriteText(text)
					}
				}
			}
			h.selectionActions(c, store, v, session, actions)
			myterm.View(c, h.terminal).Fill().Grow(1).MinHeight(0)
		}
	})
}
