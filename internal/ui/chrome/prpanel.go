package chrome

import (
	"fmt"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// PRListItem is one row in the title-bar PR panel.
type PRListItem struct {
	Label         string
	CheckoutLabel string
	OnCheckout    func()
}

// PRPanel is a compact pull-request strip under the title bar.
type PRPanel struct {
	CurrentLabel  string
	MissingGH     bool
	MissingGHText string
	EmptyText     string
	RefreshLabel  string
	ViewLabel     string
	CreateLabel   string
	CloseLabel    string
	Recent        []PRListItem
	OnRefresh     func()
	OnView        func()
	OnCreate      func()
	OnClose       func()
}

// DrawPRPanel draws the PR overview strip when open is true.
func DrawPRPanel(c *ui.Context, colors TitleBarColors, open bool, m PRPanel) {
	if !open {
		return
	}
	muted := colors.Muted
	if muted.A == 0 && muted.R == 0 && muted.G == 0 && muted.B == 0 {
		muted = colors.Foreground
	}
	leftPad, rightPad := TitleBarSafeHorizontalPadding(c.TitleBar(), design.Sp(3))
	ui.Column(c.Key("git-pr-panel")).FillWidth().MinWidth(0).Shrink(0).Background(colors.Background).
		Padding(design.Sp(1.5), rightPad, design.Sp(1.5), leftPad).Gap(design.Sp(1)).
		Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Shrink(0).AlignItems(ui.Center).Gap(design.Sp(2)).Children(func() {
				summary := m.CurrentLabel
				if m.MissingGH && m.MissingGHText != "" {
					summary = m.MissingGHText
				} else if summary == "" && m.EmptyText != "" {
					summary = m.EmptyText
				}
				if summary != "" {
					ui.Text(c, summary).FontSize(theme.InterfaceSize(12)).TextColor(colors.Foreground).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
				} else {
					ui.Box(c).Grow(1).MinWidth(0)
				}
				if !m.MissingGH && m.OnView != nil && m.ViewLabel != "" {
					if coss.Button(c, m.ViewLabel, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Key: "pr-view"}).Shrink(0).Clicked() {
						m.OnView()
					}
				}
				if !m.MissingGH && m.OnCreate != nil && m.CreateLabel != "" {
					if coss.Button(c, m.CreateLabel, coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonXS, Key: "pr-create"}).Shrink(0).Clicked() {
						m.OnCreate()
					}
				}
				if m.OnRefresh != nil && m.RefreshLabel != "" {
					if coss.Button(c, m.RefreshLabel, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Key: "pr-refresh"}).Shrink(0).Clicked() {
						m.OnRefresh()
					}
				}
				if m.OnClose != nil && m.CloseLabel != "" {
					if coss.Button(c, m.CloseLabel, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Key: "pr-close"}).Shrink(0).Clicked() {
						m.OnClose()
					}
				}
			})
			if !m.MissingGH {
				for i, item := range m.Recent {
					it := item
					ui.Row(c.Key(i)).FillWidth().MinWidth(0).Shrink(0).AlignItems(ui.Center).Gap(design.Sp(2)).Children(func() {
						label := it.Label
						if label == "" {
							label = fmt.Sprintf("#%d", i+1)
						}
						ui.Text(c, label).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
						if it.OnCheckout != nil && it.CheckoutLabel != "" {
							if coss.Button(c, it.CheckoutLabel, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Key: "pr-checkout"}).Shrink(0).Clicked() {
								it.OnCheckout()
							}
						}
					})
				}
			}
		})
	border := colors.Border
	ui.Box(c).FillWidth().Height(1).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		design.DrawHairline(p, r, false, border)
	})
}
