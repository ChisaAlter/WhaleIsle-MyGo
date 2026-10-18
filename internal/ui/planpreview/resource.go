package planpreview

import (
	"net/url"
	"strings"
)

// Document is a read-only plan resource. Its address names the source Session
// and exact tool invocation (or a temporary question identity), never a file.
type Document struct{ Address, Title, Markdown string }
type Address struct {
	SessionID, ID string
	Temporary     bool
	// SessionID remains the parent/tab owner for an addressed child.
	ChildSessionID, Mode string
}

func PlanAddress(sessionID, callID string) string { return resourceAddress("plan", sessionID, callID) }
func (a Address) IsSubagent() bool                { return a.ChildSessionID != "" }
func SubagentPlanAddress(parentID, childID, mode, callID string) string {
	return "dsh-resource://plan/subagent/" + escapePart(parentID) + "/" + escapePart(childID) + "/" + escapePart(mode) + "/" + escapePart(callID)
}
func ReviewAddress(sessionID, requestID string) string {
	return resourceAddress("plan-review", sessionID, requestID)
}
func resourceAddress(kind, sessionID, id string) string {
	return "dsh-resource://" + kind + "/" + escapePart(sessionID) + "/" + escapePart(id)
}

// Match encodeURIComponent used by the original resource provider.
func escapePart(value string) string {
	return strings.NewReplacer("+", "%20", "%21", "!", "%27", "'", "%28", "(", "%29", ")", "%2A", "*").Replace(url.QueryEscape(value))
}
func canonicalAddress(address string) string {
	target, ok := ParseAddress(address)
	if !ok {
		return address
	}
	if target.IsSubagent() {
		return SubagentPlanAddress(target.SessionID, target.ChildSessionID, target.Mode, target.ID)
	}
	if target.Temporary {
		return ReviewAddress(target.SessionID, target.ID)
	}
	return PlanAddress(target.SessionID, target.ID)
}
func ParseAddress(raw string) (Address, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "dsh-resource" || (u.Host != "plan" && u.Host != "plan-review") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsAny(raw, "?#") || u.User != nil {
		return Address{}, false
	}
	parts := strings.Split(strings.TrimPrefix(u.EscapedPath(), "/"), "/")
	for i, part := range parts {
		decoded, err := url.PathUnescape(part)
		if err != nil || decoded == "" {
			return Address{}, false
		}
		parts[i] = decoded
	}
	if len(parts) == 2 {
		return Address{SessionID: parts[0], ID: parts[1], Temporary: u.Host == "plan-review"}, true
	}
	if len(parts) == 5 && u.Host == "plan" && parts[0] == "subagent" && (parts[3] == "one-shot" || parts[3] == "continuable" || parts[3] == "unknown") {
		return Address{SessionID: parts[1], ChildSessionID: parts[2], Mode: parts[3], ID: parts[4]}, true
	}
	return Address{}, false
}

// Resources reuses the window's right-pane tab owner. Only resource identities
// enter layout recovery; body text and temporary review contents stay in memory.
type Resources struct {
	owner     string
	Tabs      []string
	Active    map[string]string
	Documents map[string]Document `json:"-"`
}

// HostOwner reads the existing Files binding: actual client identity, NUL,
// selected Session. Session navigation does not change the resource owner.
func HostOwner(binding string) string { return strings.SplitN(binding, "\x00", 2)[0] }

func (r *Resources) BindOwner(owner string) {
	if r.owner == owner {
		return
	}
	if r.owner != "" {
		r.Tabs, r.Active = nil, nil
	} else {
		// Layout recovery carries only durable addresses. A temporary review is
		// confined to its original Host/window and cannot be recovered from layout.
		keep := make([]string, 0, len(r.Tabs))
		for _, address := range r.Tabs {
			if target, ok := ParseAddress(address); ok && !target.Temporary {
				keep = append(keep, address)
			}
		}
		r.Tabs = keep
		for sid, address := range r.Active {
			if target, ok := ParseAddress(address); !ok || target.Temporary || target.SessionID != sid {
				delete(r.Active, sid)
			}
		}
	}
	r.Documents = nil
	r.owner = owner
}

func (r *Resources) Observe(doc Document) {
	if _, ok := ParseAddress(doc.Address); !ok {
		return
	}
	doc.Address = canonicalAddress(doc.Address)
	if r.Documents == nil {
		r.Documents = map[string]Document{}
	}
	if _, exists := r.Documents[doc.Address]; !exists {
		r.Documents[doc.Address] = doc
	}
}
func (r *Resources) Navigate(sessionID, address string) string {
	target, ok := ParseAddress(strings.TrimSpace(address))
	if !ok {
		return "Invalid plan resource address"
	}
	if target.SessionID != sessionID || sessionID == "" {
		return "This plan belongs to another session; switch to that session first"
	}
	address = canonicalAddress(strings.TrimSpace(address))
	if r.Active == nil {
		r.Active = map[string]string{}
	}
	r.Active[sessionID] = address
	for _, value := range r.Tabs {
		if value == address {
			return ""
		}
	}
	r.Tabs = append(r.Tabs, address)
	return ""
}
func (r *Resources) ActiveAddress(sessionID string) string {
	address := r.Active[sessionID]
	target, ok := ParseAddress(address)
	if !ok || target.SessionID != sessionID {
		return ""
	}
	return address
}
func (r *Resources) Current(sessionID string) (Document, bool) {
	address := r.ActiveAddress(sessionID)
	doc, ok := r.Documents[address]
	return doc, ok && address != ""
}
func (r *Resources) SessionTabs(sessionID string) []string {
	var out []string
	for _, address := range r.Tabs {
		if target, ok := ParseAddress(address); ok && target.SessionID == sessionID {
			out = append(out, address)
		}
	}
	return out
}
func (r *Resources) Close(sessionID, address string) {
	for i, v := range r.Tabs {
		if v == address {
			r.Tabs = append(r.Tabs[:i], r.Tabs[i+1:]...)
			break
		}
	}
	if r.ActiveAddress(sessionID) == address {
		delete(r.Active, sessionID)
	}
}
func (r *Resources) ShowFiles(sessionID string) { delete(r.Active, sessionID) }

// HistoryMatchesBinding prevents a replaced Host's retained transcript from
// repopulating the new resource cache before its first authoritative snapshot.
func HistoryMatchesBinding(historyBinding, currentBinding string) bool {
	return historyBinding != "" && historyBinding == currentBinding
}
