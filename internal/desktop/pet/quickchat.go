package pet

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// QuickChatOptions bind one compact card to one authenticated Host generation.
// Current and OpenMain run only on the UI thread. Neither is page IPC.
type QuickChatOptions struct {
	Origin      string
	Cookie      string
	Unavailable string
	Current     func() bool
	OpenMain    func(string)
}

// QuickChat is UI-thread owned, including every request slot and draft. Worker
// goroutines capture immutable clients/arguments and return via Window.Update.
type QuickChat struct {
	owner       *Companion
	parent      *mygo.Window
	win         *mygo.Window
	options     QuickChatOptions
	client      *quickChatClient
	life        context.Context
	cancel      context.CancelFunc
	closed      bool
	stale       bool
	state       quickChatSnapshot
	draft       string
	status      string
	readNote    string
	catalogNote string
	catalogAt   time.Time
	list        ui.ListState
	history     quickChatRequest
	catalog     quickChatRequest
	sending     quickChatRequest
	picking     quickChatRequest
	stopping    quickChatRequest
	jumping     quickChatRequest
}

// OpenQuickChat preserves the draft and focuses an already open live card.
// Reopening an invalidated card starts a fresh connection, retaining its draft.
func (p *Companion) OpenQuickChat(options QuickChatOptions) {
	if p == nil || p.win == nil || p.win.IsDestroyed() || !p.ChatAllowed() {
		return
	}
	draft := ""
	if old := p.QuickChat; old != nil {
		if old.check() && old.win != nil {
			old.win.Show()
			old.win.Focus()
			return
		}
		draft = old.draft
		p.CloseQuickChat()
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &QuickChat{owner: p, parent: p.win, options: options, life: ctx, cancel: cancel, draft: draft, status: "正在读取共同会话… / Loading shared conversation…"}
	q.list.FollowEnd = true
	q.list.Key = func(index int) any { return quickChatHistoryKey(q.state, index) }
	if options.Unavailable != "" {
		q.stale, q.status = true, options.Unavailable
	} else {
		client, err := newQuickChatClient(options.Origin, options.Cookie)
		if err != nil {
			q.stale, q.status = true, "Host 地址不可用 / Host unavailable"
		} else {
			q.client = client
		}
	}
	p.QuickChat = q
	p.setQuickChatOpen(true)
	w := mygo.NewWindow(mygo.WindowOptions{
		Title: "鲸鱼娘 · 聊聊 / Whale quick chat", Width: 460, Height: 620,
		MinWidth: 420, MinHeight: 580, Parent: p.win, AlwaysOnTop: true,
		SkipTaskbar: true, DisableMaximize: true, DisableFullScreen: true,
		Content: ui.View(q.view),
	})
	q.win = w
	if w == nil || w.IsDestroyed() {
		p.Error = "聊天卡片创建失败 / Quick chat window could not open"
		p.CloseQuickChat()
		return
	}
	w.OnClosed(func() {
		q.shutdown()
		if p.QuickChat == q {
			p.QuickChat = nil
			p.setQuickChatOpen(false)
		}
	})
	w.Focus()
	if q.stale {
		return
	}
	q.refreshHistory()
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.Update(func() {
					if !q.check() {
						return
					}
					q.refreshHistory()
					if time.Since(q.catalogAt) >= time.Minute {
						q.refreshCatalog()
					}
				})
			}
		}
	}()
}

func (p *Companion) CloseQuickChat() {
	if p == nil || p.QuickChat == nil {
		return
	}
	q := p.QuickChat
	p.QuickChat = nil
	p.setQuickChatOpen(false)
	q.shutdown()
	if q.win != nil && !q.win.IsDestroyed() {
		q.win.Destroy()
	}
}

// setQuickChatOpen publishes only card visibility to non-UI readers. The
// QuickChat pointer itself remains UI-thread owned.
func (p *Companion) setQuickChatOpen(open bool) {
	p.assetMu.Lock()
	p.quickChatOpen = open
	p.assetMu.Unlock()
}

// CheckQuickChat is called on Host/task or behavior-setting changes, and on
// every paint/action/result. A disabled pet-chat setting dismisses the card.
func (p *Companion) CheckQuickChat() {
	if p == nil || p.QuickChat == nil {
		return
	}
	if !p.ChatAllowed() {
		p.CloseQuickChat()
		return
	}
	p.QuickChat.check()
}
func (q *QuickChat) shutdown() {
	if q.closed {
		return
	}
	q.closed = true
	q.cancel()
	q.abortRequests()
}
func (q *QuickChat) abortRequests() {
	q.history.abort()
	q.catalog.abort()
	q.sending.abort()
	q.picking.abort()
	q.stopping.abort()
	q.jumping.abort()
}
func (q *QuickChat) check() bool {
	if q.closed || q.stale {
		return false
	}
	current := q.owner.QuickChat == q && q.owner.win == q.parent && q.parent != nil && !q.parent.IsDestroyed() && q.win != nil && !q.win.IsDestroyed() && q.owner.ChatAllowed() && q.options.Current != nil && q.options.Current()
	if current {
		return true
	}
	q.stale = true
	q.cancel()
	q.abortRequests()
	q.status = "Host 或设置已变化，草稿保留。重新点聊聊连接。 / Connection changed; reopen quick chat."
	if q.win != nil {
		q.win.Invalidate()
	}
	return false
}
func (q *QuickChat) refreshHistory() {
	if !q.check() || q.picking.busy() {
		return
	}
	ctx, serial := q.history.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		next, err := client.state(ctx, false)
		win.Update(func() {
			if !q.check() || !q.history.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err != nil {
				q.readNote = "共同会话暂不可读，请检查 Host / Shared history unavailable"
				return
			}
			q.readNote = ""
			changed := q.state.replaceHistory(next)
			if changed {
				q.sending.abort()
				q.picking.abort()
				q.stopping.abort()
				q.jumping.abort()
				q.catalog.abort()
				q.catalogAt = time.Time{}
				q.list.ScrollToEnd()
				if expected == "" {
					q.status = "与主窗口使用同一持久会话 / Shared with main conversation"
				} else {
					q.status = "宠物会话已更新；旧请求已停止等待，草稿保留 / Pet session changed"
				}
			}
			if changed || q.catalogAt.IsZero() {
				q.refreshCatalog()
			}
		})
	}()
}
func (q *QuickChat) refreshCatalog() {
	if !q.check() || q.state.SessionID == "" || q.picking.busy() {
		return
	}
	ctx, serial := q.catalog.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		next, err := client.state(ctx, true)
		win.Update(func() {
			if !q.check() || !q.catalog.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err != nil || next.SessionID != expected {
				q.catalogNote = "模型目录暂不可用 / Model catalog unavailable"
				if err == nil {
					q.refreshHistory()
				}
				return
			}
			// A slow catalog never replaces fresher history or model selection.
			q.state.Groups = next.Groups
			q.catalogAt = time.Now()
			q.catalogNote = ""
			if next.Groups == nil {
				q.catalogNote = "Host 未提供模型目录 / Host model catalog unavailable"
			}
		})
	}()
}
func (q *QuickChat) send() {
	if !q.check() || q.state.SessionID == "" || q.picking.busy() || q.stopping.busy() {
		return
	}
	draft := q.draft
	text, err := quickChatText(draft)
	if err != nil {
		q.status = "请输入 1–2000 字；表情可能占两字 / Enter up to 2000 UTF-16 units"
		return
	}
	ctx, serial := q.sending.begin(q.life, 140*time.Second)
	if ctx == nil {
		return
	}
	q.status = "正在等待共同会话回复… / Waiting for shared session…"
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		err := client.send(ctx, expected, text)
		win.Update(func() {
			if !q.check() || !q.sending.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err == nil {
				if q.draft == draft {
					q.draft = ""
				}
				q.status = "本轮已完成，正在同步记录 / Turn completed; syncing history"
			} else if errors.Is(err, errQuickChatSessionChanged) {
				q.status = "会话已改变，消息未发送；请重新查看 / Pet session changed before send"
			} else {
				q.status = "未确认本轮完成；请查看共同记录，勿重复发送 / Completion unconfirmed; check history"
			}
			q.history.abort()
			q.refreshHistory()
		})
	}()
}
func (q *QuickChat) cancelWait() {
	if !q.check() || !q.sending.busy() {
		return
	}
	q.sending.abort()
	q.status = "已取消本地等待。Host 可能仍在运行或排队。 / Wait cancelled; Host may continue."
	q.refreshHistory()
}
func (q *QuickChat) choose(choice quickChatModel) {
	if !q.check() || q.state.SessionID == "" || q.sending.busy() || q.stopping.busy() || !q.state.allows(choice) {
		return
	}
	if q.state.Model != nil && *q.state.Model == choice {
		return
	}
	ctx, serial := q.picking.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	q.history.abort()
	q.catalog.abort()
	q.status = "正在切换共同会话模型… / Changing shared session model…"
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		selected, err := client.selectModel(ctx, expected, choice)
		win.Update(func() {
			if !q.check() || !q.picking.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err == nil {
				q.state.Model = selected
				q.status = "模型已更新 / Model updated"
			} else {
				q.status = "模型未确认切换，请刷新后重试 / Model change unconfirmed"
			}
			q.history.abort()
			q.refreshHistory()
			q.catalogAt = time.Time{}
		})
	}()
}
func (q *QuickChat) stopTurn() {
	if !q.check() || q.state.SessionID == "" || q.picking.busy() {
		return
	}
	ctx, serial := q.stopping.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	q.status = "请求停止宠物会话当前轮… / Stopping the current pet turn…"
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		err := client.stop(ctx, expected)
		win.Update(func() {
			if !q.check() || !q.stopping.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err == nil {
				q.sending.abort()
				q.status = "Host 已接受停止当前轮；排队消息请到主会话管理 / Stop accepted; queued messages remain"
			} else {
				q.status = "未确认停止，请到主会话查看 / Stop unconfirmed; check main conversation"
			}
			q.history.abort()
			q.refreshHistory()
		})
	}()
}
func (q *QuickChat) openMain() {
	if !q.check() || q.state.SessionID == "" || q.options.OpenMain == nil {
		return
	}
	ctx, serial := q.jumping.begin(q.life, 30*time.Second)
	if ctx == nil {
		return
	}
	client, win, expected := q.client, q.win, q.state.SessionID
	go func() {
		err := client.sameSession(ctx, expected)
		win.Update(func() {
			if !q.check() || !q.jumping.finish(serial) || q.state.SessionID != expected {
				return
			}
			if err != nil {
				q.status = "会话身份已变化或不可用，请刷新 / Session changed or unavailable"
				q.refreshHistory()
				return
			}
			q.options.OpenMain(expected)
		})
	}()
}
