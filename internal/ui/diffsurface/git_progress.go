package diffsurface

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
)

// DrawGitProgress shows bounded, operation-owned progress; dismissal never stops
// a mutating Git command. Only message preparation has a cancellation action.
func DrawGitProgress(c *ui.Context, p *git.Progress, fg, muted ui.Color, onDismiss func()) {
	if p == nil {
		return
	}
	s := p.Snapshot()
	ui.Column(c).FillWidth().Padding(4, 12, 4, 12).Gap(2).Children(func() {
		ui.Row(c).FillWidth().Gap(6).Children(func() {
			ui.Text(c, s.Phase).FontSize(11).TextColor(fg).Grow(1).MaxLines(2)
			if s.Done && onDismiss != nil {
				if ui.Button(c, word("关闭进度", "Dismiss progress")).FontSize(10).Clicked() {
					onDismiss()
				}
			}
		})
		ui.Text(c, s.Root).FontSize(10).TextColor(muted).MaxLines(1)
		lines := s.Lines
		if len(lines) > 5 {
			lines = lines[len(lines)-5:]
		}
		for _, line := range lines {
			ui.Text(c, line).FontSize(10).TextColor(muted).MaxLines(2)
		}
	})
}
