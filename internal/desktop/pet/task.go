package pet

import (
	"html"
	"net/url"
	"strings"
	"whaleisle/internal/store/session"
)

// TaskStatus is deliberately limited to the selected session on this connection.
// Non-running is not a claim of successful completion. No message bodies leave the store.
type TaskStatus struct{ State, SessionID, Title string }

func TaskFromSession(online bool, s session.ViewState) TaskStatus {
	if !online || !s.ClientReady {
		return TaskStatus{State: "offline"}
	}
	if s.ListLoading || s.ListErr != "" {
		return TaskStatus{State: "unknown"}
	}
	for _, item := range s.Items {
		if item.SessionID != "" && item.SessionID == s.SelectedID {
			state := "idle"
			if item.Running {
				state = "running"
			}
			return TaskStatus{State: state, SessionID: item.SessionID, Title: item.Title}
		}
	}
	return TaskStatus{State: "unknown"}
}

func renderTaskPage(page []byte, task TaskStatus) []byte {
	label := "状态未知 / Unknown"
	switch task.State {
	case "running":
		label = "运行中 / Running"
	case "idle":
		label = "未运行 / Idle"
	case "offline":
		label = "离线 / Offline"
	}
	open := ""
	if task.SessionID != "" {
		open = `<a id="open" href="open?session=` + html.EscapeString(url.QueryEscape(task.SessionID)) + `" title="回到此会话 / Open this conversation">回到会话 / Open chat</a>`
	}
	return []byte(strings.NewReplacer("{{STATUS}}", html.EscapeString(label), "{{TITLE}}", html.EscapeString(task.Title), "{{OPEN}}", open).Replace(string(page)))
}

// Only these exact local navigation actions are accepted. The page never gets
// arbitrary IPC, a file API, a URL opener, or a session selector.
func taskAction(raw string, task TaskStatus) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "whale-pet" || u.Host != "localhost" || u.User != nil || u.Fragment != "" {
		return ""
	}
	if raw == "whale-pet://localhost/chat" {
		return "chat"
	}
	if u.Path == "/hide" && u.RawQuery == "" {
		return "hide"
	}
	if task.SessionID != "" && raw == "whale-pet://localhost/open?session="+url.QueryEscape(task.SessionID) {
		return "open"
	}
	return ""
}
