package deeplink

import (
	"net/url"
	"strings"
)

// Action is a high-level route derived from a whaleisle:// URL.
type Action string

const (
	ActionNone     Action = ""
	ActionOpen     Action = "open"
	ActionSettings Action = "settings"
	ActionSession  Action = "session"
	ActionLauncher Action = "launcher"
	ActionUsage    Action = "usage"
	ActionMarket   Action = "market"
	ActionSchedule Action = "schedule"
	ActionAgents   Action = "agents"
	ActionAuth     Action = "auth" // OAuth callback (whaleisle://auth)
)

// Target is a parsed deep link.
type Target struct {
	Action    Action
	SessionID string
	Section   string // settings section hint (desktop, credentials, …)
	Raw       string
}

// Parse interprets whaleisle:// URLs.
//
// Supported forms:
//
//	whaleisle://open
//	whaleisle://launcher
//	whaleisle://settings[?section=desktop|credentials]
//	whaleisle://session/<id>
//	whaleisle://usage
//	whaleisle:///?action=settings&section=desktop
func Parse(raw string) (Target, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Target{}, false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Target{}, false
	}
	if !strings.EqualFold(u.Scheme, Scheme) {
		return Target{}, false
	}
	t := Target{Raw: raw}
	host := strings.ToLower(strings.TrimSpace(u.Host))
	path := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
	q := u.Query()

	switch host {
	case "", "open":
		if path == "" && q.Get("action") == "" {
			t.Action = ActionOpen
			return t, true
		}
		if host == "open" {
			t.Action = ActionOpen
			return t, true
		}
		// path-only or query action on empty host
		if a := strings.ToLower(q.Get("action")); a != "" {
			return parseAction(a, q, t)
		}
		return parsePath(path, q, t)
	case "launcher":
		t.Action = ActionLauncher
		return t, true
	case "settings":
		t.Action = ActionSettings
		t.Section = strings.TrimSpace(q.Get("section"))
		if path != "" && t.Section == "" {
			t.Section = path
		}
		return t, true
	case "session":
		t.Action = ActionSession
		t.SessionID = firstNonEmpty(path, q.Get("id"), q.Get("sessionId"))
		return t, true
	case "usage":
		t.Action = ActionUsage
		return t, true
	case "market", "plugins":
		t.Action = ActionMarket
		return t, true
	case "schedule":
		t.Action = ActionSchedule
		return t, true
	case "agents":
		t.Action = ActionAgents
		return t, true
	case "auth", "callback", "oauth":
		t.Action = ActionAuth
		return t, true
	default:
		return parsePath(host+"/"+path, q, t)
	}
}

func parseAction(a string, q url.Values, t Target) (Target, bool) {
	switch a {
	case "open":
		t.Action = ActionOpen
	case "launcher":
		t.Action = ActionLauncher
	case "settings":
		t.Action = ActionSettings
		t.Section = strings.TrimSpace(q.Get("section"))
	case "session":
		t.Action = ActionSession
		t.SessionID = firstNonEmpty(q.Get("id"), q.Get("sessionId"))
	case "usage":
		t.Action = ActionUsage
	case "market", "plugins":
		t.Action = ActionMarket
	case "schedule":
		t.Action = ActionSchedule
	case "agents":
		t.Action = ActionAgents
	case "auth", "callback", "oauth":
		t.Action = ActionAuth
	default:
		return Target{}, false
	}
	return t, true
}

func parsePath(path string, q url.Values, t Target) (Target, bool) {
	path = strings.Trim(path, "/")
	if path == "" {
		t.Action = ActionOpen
		return t, true
	}
	parts := strings.SplitN(path, "/", 2)
	switch strings.ToLower(parts[0]) {
	case "launcher":
		t.Action = ActionLauncher
	case "settings":
		t.Action = ActionSettings
		if len(parts) > 1 {
			t.Section = parts[1]
		} else {
			t.Section = strings.TrimSpace(q.Get("section"))
		}
	case "session":
		t.Action = ActionSession
		if len(parts) > 1 {
			t.SessionID = parts[1]
		} else {
			t.SessionID = firstNonEmpty(q.Get("id"), q.Get("sessionId"))
		}
	case "usage":
		t.Action = ActionUsage
	case "market", "plugins":
		t.Action = ActionMarket
	case "schedule":
		t.Action = ActionSchedule
	case "agents":
		t.Action = ActionAgents
	case "auth", "callback", "oauth":
		t.Action = ActionAuth
	case "open":
		t.Action = ActionOpen
	default:
		return Target{}, false
	}
	return t, true
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
