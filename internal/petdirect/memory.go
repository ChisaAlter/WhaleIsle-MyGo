package petdirect

// Memory is owned by the native window/UI thread. Tickets isolate a cancelled,
// closed or replaced request from later drafts and replies. Nothing persists.
type Memory struct {
	serial   uint64
	active   bool
	messages []Message
}
type Ticket struct {
	ID      uint64
	Text    string
	History []Message
}

func (m *Memory) Begin(text string) (Ticket, bool) {
	text, err := Text(text)
	if err != nil || m.active {
		return Ticket{}, false
	}
	m.serial++
	m.active = true
	return Ticket{m.serial, text, append([]Message(nil), m.messages...)}, true
}
func (m *Memory) Finish(ticket Ticket, reply string) bool {
	if !m.active || ticket.ID != m.serial {
		return false
	}
	m.active = false
	m.messages = append(m.messages, Message{"user", ticket.Text}, Message{"assistant", truncate(reply, ReplyLimit)})
	if len(m.messages) > HistoryTurns*2 {
		m.messages = append([]Message(nil), m.messages[len(m.messages)-HistoryTurns*2:]...)
	}
	return true
}
func (m *Memory) Cancel() { m.serial++; m.active = false }
func (m *Memory) Reset()  { m.Cancel(); m.messages = nil }

// A draft from the shared session must never become a new direct API send (or
// vice versa), particularly after a Host outcome is unknown.
func RetainDraft(oldDirect, nextDirect bool, draft string) string {
	if oldDirect || nextDirect {
		return ""
	}
	return draft
}
