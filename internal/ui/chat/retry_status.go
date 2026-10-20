package chat

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
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
		message = i18n.T(i18n.KeyChatRetryAuthFailed)
	case "ACCOUNT_SIGNED_OUT":
		message = i18n.T(i18n.KeyChatRetrySignedOut)
	case "ACCOUNT_SIGN_IN_REQUIRED":
		message = i18n.T(i18n.KeyChatRetrySignIn)
	case "QUOTA", "ACCOUNT_QUOTA":
		message = i18n.T(i18n.KeyChatRetryQuota)
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
		label := i18n.T(i18n.KeyChatRetryWaiting)
		switch current.State {
		case "started":
			label = i18n.T(i18n.KeyChatRetryStarted)
		case "cancelled":
			label = i18n.T(i18n.KeyChatRetryCancelled)
		}
		text := fmt.Sprintf("%s · %d/%s", label, current.Retry, maximum)
		if current.State == "scheduled" {
			seconds := retrySeconds(*deadline, time.Now())
			if seconds > 0 {
				text += fmt.Sprintf(" · 预计 %d 秒 / ~%ds", seconds, seconds)
				c.Services().Invalidate()
			} else {
				text += " · " + i18n.T(i18n.KeyChatRetryAwaitingHost)
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
		return i18n.T(i18n.KeyChatTurnMaxTokens)
	}
	if r.Kind == "aborted" && r.Reason.Kind == "hook" && r.Reason.Reason == "deepseek-account/signed-out" {
		return i18n.T(i18n.KeyChatTurnSignedOut)
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
