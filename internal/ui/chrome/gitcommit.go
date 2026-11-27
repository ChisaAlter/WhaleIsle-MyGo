package chrome

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
	leftPad, rightPad := TitleBarSafeHorizontalPadding(c.TitleBar(), design.Sp(3))
	ui.Row(c.Key("git-commit-strip")).FillWidth().MinWidth(0).Height(40).Shrink(0).Background(colors.Background).
		Padding(0, rightPad, 0, leftPad).AlignItems(ui.Center).Gap(design.Sp(2)).
		Children(func() {
			var inp ui.Element
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				inp = coss.Input(c, m.Message, coss.InputOptions{
					Size:        coss.InputSM,
					Placeholder: m.Placeholder,
					Key:         "git-commit-message",
				})
			})
			if m.GenerateError != "" {
				ui.Text(c, m.GenerateError).FontSize(theme.InterfaceSize(11)).TextColor(design.For(c).Destructive).MaxLines(1)
			}
			if m.OnGenerate != nil && m.GenerateLabel != "" {
				label := m.GenerateLabel
				if m.Generating {
					label = m.GenerateLabel + "…"
				}
				if coss.Button(c, label, coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonSM,
					Disabled: m.Generating,
					Key:      "git-commit-generate",
				}).Shrink(0).Clicked() {
					m.OnGenerate()
				}
			}
			commitEnabled := m.Dirty && m.OnCommit != nil && m.CommitLabel != ""
			if commitEnabled {
				if inp.Submitted() {
					m.OnCommit()
				}
				if coss.Button(c, m.CommitLabel, coss.ButtonOptions{
					Variant: coss.ButtonDefault,
					Size:    coss.ButtonSM,
					Key:     "git-commit-submit",
				}).Shrink(0).Clicked() {
					m.OnCommit()
				}
			} else if m.CommitLabel != "" {
				coss.Button(c, m.CommitLabel, coss.ButtonOptions{
					Variant:  coss.ButtonDefault,
					Size:     coss.ButtonSM,
					Disabled: true,
					Key:      "git-commit-submit",
				}).Shrink(0)
			}
			if m.OnCancel != nil && m.CancelLabel != "" {
				if coss.Button(c, m.CancelLabel, coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonSM,
					Key:     "git-commit-cancel",
				}).Shrink(0).Clicked() {
					m.OnCancel()
				}
			}
		})
	border := colors.Border
	ui.Box(c).FillWidth().Height(1).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		design.DrawHairline(p, r, false, border)
	})
}
