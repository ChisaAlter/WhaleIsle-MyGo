package markdown

import "testing"

func TestSpansListContent(t *testing.T) {
	tests := []struct {
		name, source, want string
	}{
		{"tight", "- first\n- second", "• first\n• second"},
		{"loose", "- first\n\n- second", "• first\n• second"},
		{"ordered start", "3. third\n4. fourth", "3. third\n4. fourth"},
		{"nested mixed", "3. parent\n   - child\n\n     7. grandchild\n     8. next grandchild\n   - next child\n4. next parent", "3. parent\n  • child\n    7. grandchild\n    8. next grandchild\n  • next child\n4. next parent"},
		{"multiple paragraphs", "- first\n\n  continuation\n\n- second", "• first\ncontinuation\n• second"},
		{"inline styles", "- **bold** and `code`\n- *italic*", "• bold and code\n• italic"},
		{"surrounding paragraphs", "Before\n\n- first\n- second\n\nAfter", "Before\n• first\n• second\nAfter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := concat(Spans(tt.source, Style{FontSize: 14})); got != tt.want {
				t.Fatalf("list text = %q, want %q", got, tt.want)
			}
		})
	}
}
