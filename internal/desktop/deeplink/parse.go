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
	if raw == "" || strings.ContainsRune(raw, '\\') {
		return Target{}, false
	}
	for _, r := range raw {
		if r < 0x20 || r == 0x7f {
			return Target{}, false
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return Target{}, false
	}
	if !strings.EqualFold(u.Scheme, Scheme) {
		return Target{}, false
	}
	t := Target{Raw: raw}
	host := strings.ToLower(strings.TrimSpace(u.Host))
	path := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
	q := u.Query()

	var ok bool
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
			t, ok = parseAction(a, q, t)
		} else {
			t, ok = parsePath(path, q, t)
		}
	case "launcher":
		t.Action, ok = ActionLauncher, true
	case "settings":
		t.Action = ActionSettings
		t.Section = strings.TrimSpace(q.Get("section"))
		if path != "" && t.Section == "" {
			t.Section = path
		}
		ok = true
	case "session":
		t.Action = ActionSession
		t.SessionID = firstNonEmpty(path, q.Get("id"), q.Get("sessionId"))
		ok = true
	case "usage":
		t.Action, ok = ActionUsage, true
	case "market", "plugins":
		t.Action, ok = ActionMarket, true
	case "schedule":
		t.Action, ok = ActionSchedule, true
	case "agents":
		t.Action, ok = ActionAgents, true
	case "auth", "callback", "oauth":
		t.Action, ok = ActionAuth, true
	default:
		t, ok = parsePath(host+"/"+path, q, t)
	}
	if !ok || !validateTarget(&t) {
		return Target{}, false
	}
	return t, true
}

func validateTarget(t *Target) bool {
	if !validDeeplinkParam(t.Section, 64, false) {
		return false
	}
	if !validDeeplinkParam(t.SessionID, 128, true) {
		return false
	}
	return true
}

func validDeeplinkParam(v string, maxLen int, allowDot bool) bool {
	if v == "" {
		return true
	}
	if len(v) > maxLen || v == "." || v == ".." || strings.Contains(v, "..") {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		isAlphaNum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if isAlphaNum || c == '-' || c == '_' || (allowDot && c == '.') {
			continue
		}
		return false
	}
	return true
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
