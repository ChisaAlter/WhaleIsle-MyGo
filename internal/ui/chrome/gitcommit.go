package chrome

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// GitCommitStrip is a one-line commit message row shown under the title bar.
type GitCommitStrip struct {
	Message     *string
	Dirty       bool
	Placeholder string
	CommitLabel string
	CancelLabel string
	// GenerateLabel paints the message generation action; empty hides it. The
	// port of git-generate.js produces copy only: committing stays explicit.
	GenerateLabel string
	// Generating disables the generation action while that step runs.
	Generating bool
	// GenerateError is the bounded failure copy; manual entry keeps working.
	GenerateError string
	OnCommit      func()
	OnCancel      func()
	OnGenerate    func()
}

// GitCommitStrip draws the commit message input when Open is true.
func DrawGitCommitStrip(c *ui.Context, colors TitleBarColors, open bool, m GitCommitStrip) {
	if !open || m.Message == nil {
		return
	}
	muted := colors.Muted
	if muted.A == 0 && muted.R == 0 && muted.G == 0 && muted.B == 0 {
		muted = colors.Foreground
	}
	ui.Row(c).Fill().Height(36).Background(colors.Background).
		Padding(0, 12, 0, 12).AlignItems(ui.Center).Gap(8).
		Children(func() {
			inp := ui.TextInput(c, m.Message).Grow(1).Placeholder(m.Placeholder)
			if m.GenerateError != "" {
				ui.Text(c, m.GenerateError).FontSize(theme.InterfaceSize(11)).TextColor(muted).MaxLines(1)
			}
			if m.OnGenerate != nil && m.GenerateLabel != "" {
				label := m.GenerateLabel
				if m.Generating {
					label = m.GenerateLabel + "…"
				}
				if ui.Button(c, label).Disabled(m.Generating).Clicked() {
					m.OnGenerate()
				}
			}
			commitEnabled := m.Dirty && m.OnCommit != nil && m.CommitLabel != ""
			if commitEnabled {
				if inp.Submitted() {
					m.OnCommit()
				}
				if ui.Button(c, m.CommitLabel).Clicked() {
					m.OnCommit()
				}
			} else if m.CommitLabel != "" {
				ui.Button(c, m.CommitLabel).Disabled(true)
			}
			if m.OnCancel != nil && m.CancelLabel != "" {
				if ui.Button(c, m.CancelLabel).Clicked() {
					m.OnCancel()
				}
			}
		})
	ui.Box(c).Fill().Height(1).Background(colors.Border)
}
