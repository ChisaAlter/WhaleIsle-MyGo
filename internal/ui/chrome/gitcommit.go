package chrome

import (
	"github.com/egoist/mygo/ui"
)

// GitCommitStrip is a one-line commit message row shown under the title bar.
type GitCommitStrip struct {
	Message     *string
	Dirty       bool
	Placeholder string
	CommitLabel string
	CancelLabel string
	OnCommit    func()
	OnCancel    func()
}

// GitCommitStrip draws the commit message input when Open is true.
func DrawGitCommitStrip(c *ui.Context, colors TitleBarColors, open bool, m GitCommitStrip) {
	if !open || m.Message == nil {
		return
	}
	ui.Row(c).Fill().Height(36).Background(colors.Background).
		Padding(0, 12, 0, 12).AlignItems(ui.Center).Gap(8).
		Children(func() {
			inp := ui.TextInput(c, m.Message).Grow(1).Placeholder(m.Placeholder)
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
