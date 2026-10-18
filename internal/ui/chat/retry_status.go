package chat

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func displayFailure(raw json.RawMessage) (message, code string) {
	var failure struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if json.Unmarshal(raw, &failure) != nil {
		return string(raw), ""
	}
	message, code = failure.Message, failure.Code
	switch code {
	case "AUTH":
		message = "身份验证失败，请检查登录状态。 / Authentication failed. Check your sign-in."
	case "ACCOUNT_SIGNED_OUT":
		message = "已停止，因为你已退出 DeepSeek。 / Stopped because you signed out of DeepSeek."
	case "ACCOUNT_SIGN_IN_REQUIRED":
		message = "请先登录 DeepSeek。 / Sign in to DeepSeek to continue."
	case "QUOTA", "ACCOUNT_QUOTA":
		message = "额度不足。 / Account quota exhausted."
	}
	if message == "" && len(raw) > 0 {
		message = string(raw)
	}
	return message, code
}

func retrySeconds(deadline, now time.Time) int {
	return max(0, int(math.Ceil(deadline.Sub(now).Seconds())))
}

func RetryStatusView(c *ui.Context, m Model, msg session.Message, fg, muted ui.Color) {
	attempts := msg.Retry.Attempts
	if len(attempts) == 0 {
		return
	}
	current := attempts[len(attempts)-1]
	scope := fmt.Sprintf("retry:%s:%s:%s:%.0f", m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, current.RetryID, current.Seq)
	box := ui.Column(c.Key(scope)).FillWidth().Gap(4)
	// As in the original client, anchor the Host's delay to first presentation:
	// Host time and client wall time need not share a clock. This is an estimate,
	// and only llm/retry-started can transition the scheduled status.
	deadline := ui.Local(box, "deadline", func() time.Time { return time.Now().Add(time.Duration(current.DelayMs * float64(time.Millisecond))) })
	box.Children(func() {
		maximum := fmt.Sprint(current.MaxRetries)
		if current.Mode == "always" {
			maximum = "∞"
		}
		label := "等待重试 / Waiting to retry"
		switch current.State {
		case "started":
			label = "重试已开始 / Retry started"
		case "cancelled":
			label = "重试已取消 / Retry cancelled"
		}
		text := fmt.Sprintf("%s · %d/%s", label, current.Retry, maximum)
		if current.State == "scheduled" {
			seconds := retrySeconds(*deadline, time.Now())
			if seconds > 0 {
				text += fmt.Sprintf(" · 预计 %d 秒 / ~%ds", seconds, seconds)
				c.Services().Invalidate()
			} else {
				text += " · 等待 Host / Awaiting Host"
			}
		}
		open := workflowExpanded(m.ThinkingExpanded, scope, false)
		arrow := "▸ "
		if open {
			arrow = "▾ "
		}
		if ui.Button(c, arrow+text).FontSize(12).TextColor(muted).Clicked() {
			workflowToggle(m.ThinkingExpanded, scope, false)
		}
		if !open {
			return
		}
		for _, a := range attempts {
			ui.Text(c, fmt.Sprintf("#%d · %s · %.0f ms", a.Retry, a.State, a.DelayMs)).FontSize(11).TextColor(muted)
			message, code := displayFailure(a.Failure)
			if code != "" {
				message += " [" + code + "]"
			}
			if message != "" {
				ui.Text(c, message).FontSize(12).TextColor(fg)
			}
		}
	})
}

type turnReason struct {
	Kind   string          `json:"kind"`
	Error  json.RawMessage `json:"error"`
	Reason struct {
		Kind   string `json:"kind"`
		Reason string `json:"reason"`
	} `json:"reason"`
}

func readTurnReason(msg session.Message) turnReason {
	var e struct {
		Data struct {
			Reason turnReason `json:"reason"`
		} `json:"data"`
	}
	_ = json.Unmarshal(msg.Raw, &e)
	return e.Data.Reason
}
func terminalReasonDetail(msg session.Message) string {
	r := readTurnReason(msg)
	if r.Kind == "max-tokens" {
		return "输出达到本次请求的 token 上限，回答可能不完整。 / Output reached the request token limit; the answer may be incomplete."
	}
	if r.Kind == "aborted" && r.Reason.Kind == "hook" && r.Reason.Reason == "deepseek-account/signed-out" {
		return "已停止，因为你已退出 DeepSeek。 / Stopped because you signed out of DeepSeek. [ACCOUNT_SIGNED_OUT]"
	}
	if r.Kind == "error" || r.Kind == "failed" {
		message, code := displayFailure(r.Error)
		if code != "" {
			message += " [" + code + "]"
		}
		return message
	}
	if r.Kind == "aborted" {
		return r.Reason.Reason
	}
	return ""
}
