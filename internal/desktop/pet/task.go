package pet

import (
	"html"
	"net/url"
	"strings"
	"whaleisle/internal/store/session"
)

// TaskStatus identifies the selected session. Global watcher activity may supply
// a running display without pretending that the selected session is active.
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
	if raw == "whale-pet://localhost/motion-contact" {
		return "motion-contact"
	}
	if raw == "whale-pet://localhost/refresh-growth" {
		return "refresh-growth"
	}
	if raw == "whale-pet://localhost/feed-tokens" {
		return "feed-tokens"
	}
	for _, action := range []string{"sleep-pet", "wake-pet", "wake-contact"} {
		if raw == "whale-pet://localhost/"+action {
			return action
		}
	}
	if raw == "whale-pet://localhost/pat-pet" {
		return "pat-pet"
	}
	if raw == "whale-pet://localhost/feed-file" {
		return "feed-file"
	}
	if raw == "whale-pet://localhost/look-screen" {
		return "look-screen"
	}
	if raw == "whale-pet://localhost/settings" {
		return "settings"
	}
	if raw == "whale-pet://localhost/quick-chat" {
		return "quick-chat"
	}
	if raw == "whale-pet://localhost/chat" {
		return "chat"
	}
	if raw == "whale-pet://localhost/hide" {
		return "hide"
	}
	if task.SessionID != "" && raw == "whale-pet://localhost/open?session="+url.QueryEscape(task.SessionID) {
		return "open"
	}
	return ""
}

// taskWithGlobalActivity combines the current selected-session card with the
// active Host watcher. Global work has no single navigation target.
func taskWithGlobalActivity(task TaskStatus, owner, state string) TaskStatus {
	if owner != "" && state == "working" && task.State != "running" {
		return TaskStatus{State: "running", Title: "其他会话正在运行 / Other conversations are working"}
	}
	return task
}

// The caller holds assetMu, matching access to task and notifications.
func (p *Companion) activityTaskLocked() TaskStatus {
	return taskWithGlobalActivity(p.task, p.notifications.owner, p.notifications.snapshot.State)
}
