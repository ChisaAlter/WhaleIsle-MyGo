package pet

import (
	"errors"

	"whaleisle/internal/petdirect"
)

func (q *QuickChat) sendDirect() {
	if !q.check() || q.options.Direct == nil {
		return
	}
	draft := q.draft
	ticket, ok := q.direct.Begin(draft)
	if !ok {
		q.status = "请输入 1–2000 字；表情可能占两字 / Enter up to 2000 UTF-16 units"
		return
	}
	ctx, serial := q.sending.begin(q.life, petdirect.Timeout)
	if ctx == nil {
		q.direct.Cancel()
		return
	}
	q.status = "正在请求已配置 API；临时聊天不会写入共同会话 / Sending to configured API…"
	client, win := q.options.Direct, q.win
	go func() {
		reply, err := client.Send(ctx, ticket.Text, ticket.History)
		win.Update(func() {
			if !q.check() || !q.sending.finish(serial) {
				return
			}
			if err != nil {
				q.direct.Cancel()
				switch {
				case errors.Is(err, petdirect.ErrCredentials):
					q.status = "桌面 API 凭据不可用，请在设置中检查 / Desktop API credentials unavailable"
				case errors.Is(err, petdirect.ErrEndpoint):
					q.status = "API 地址无效或使用非本机明文 HTTP / API endpoint is invalid or insecure"
				case errors.Is(err, petdirect.ErrChanged):
					q.status = "API 或人设设置已变化；关闭后重新点聊聊 / Settings changed; close and reopen"
				default:
					q.status = "未确认 API 回复，请勿直接重复发送；服务端可能已收到 / API result unconfirmed; avoid duplicate sends"
				}
				return
			}
			if !q.direct.Finish(ticket, reply) {
				return
			}
			q.state.History = append(q.state.History, quickChatRow{Role: "user", Text: ticket.Text, Seq: float64(ticket.ID * 2)}, quickChatRow{Role: "her", Text: reply, Seq: float64(ticket.ID*2 + 1)})
			if len(q.state.History) > petdirect.HistoryTurns*2 {
				q.state.History = append([]quickChatRow(nil), q.state.History[len(q.state.History)-petdirect.HistoryTurns*2:]...)
			}
			if q.draft == draft {
				q.draft = ""
			}
			q.list.ScrollToEnd()
			q.status = "临时回复已收到；关闭窗口后清除 / Temporary reply received"
		})
	}()
}
