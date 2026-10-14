package chat

import "strings"

// QuoteIntoDraft prepends a markdown block-quote of body into draft (Electron 引用).
func QuoteIntoDraft(draft *string, body string) {
	if draft == nil {
		return
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	var b strings.Builder
	for i, line := range strings.Split(body, "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("> ")
		b.WriteString(line)
	}
	quoted := b.String() + "\n\n"
	cur := strings.TrimLeft(*draft, " \t")
	if cur == "" {
		*draft = quoted
		return
	}
	*draft = quoted + cur
}
