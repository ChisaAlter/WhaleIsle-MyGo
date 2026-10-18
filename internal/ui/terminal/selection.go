package terminal

import "strings"

// FormatSelection mirrors the original terminal Add to chat action. A fence
// longer than any selected backtick run keeps the selection inside one block.
func FormatSelection(text string) string {
	body := strings.Trim(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if body == "" {
		return ""
	}
	longest, run := 0, 0
	for _, ch := range body {
		if ch == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	length := longest + 1
	if length < 3 {
		length = 3
	}
	fence := strings.Repeat("\x60", length)
	return fence + "terminal\n" + body + "\n" + fence
}
