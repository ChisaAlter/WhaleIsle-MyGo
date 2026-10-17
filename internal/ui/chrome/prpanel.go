package chrome

import (
	"fmt"

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
	ui.Column(c).Fill().Background(colors.Background).
		Padding(6, 12, 6, 12).Gap(4).
		Children(func() {
			ui.Row(c).Fill().AlignItems(ui.Center).Gap(8).Children(func() {
				summary := m.CurrentLabel
				if m.MissingGH && m.MissingGHText != "" {
					summary = m.MissingGHText
				} else if summary == "" && m.EmptyText != "" {
					summary = m.EmptyText
				}
				if summary != "" {
					ui.Text(c, summary).FontSize(12).TextColor(colors.Foreground).Grow(1)
				} else {
					ui.Box(c).Grow(1)
				}
				if !m.MissingGH && m.OnView != nil && m.ViewLabel != "" {
					if ui.Button(c, m.ViewLabel).Clicked() {
						m.OnView()
					}
				}
				if !m.MissingGH && m.OnCreate != nil && m.CreateLabel != "" {
					if ui.Button(c, m.CreateLabel).Clicked() {
						m.OnCreate()
					}
				}
				if m.OnRefresh != nil && m.RefreshLabel != "" {
					if ui.Button(c, m.RefreshLabel).Clicked() {
						m.OnRefresh()
					}
				}
				if m.OnClose != nil && m.CloseLabel != "" {
					if ui.Button(c, m.CloseLabel).Clicked() {
						m.OnClose()
					}
				}
			})
			if !m.MissingGH {
				for i, item := range m.Recent {
					it := item
					ui.Row(c).Fill().AlignItems(ui.Center).Gap(8).Children(func() {
						label := it.Label
						if label == "" {
							label = fmt.Sprintf("#%d", i+1)
						}
						ui.Text(c, label).FontSize(11).TextColor(muted).Grow(1)
						if it.OnCheckout != nil && it.CheckoutLabel != "" {
							if ui.Button(c, it.CheckoutLabel).Clicked() {
								it.OnCheckout()
							}
						}
					})
				}
			}
		})
	ui.Box(c).Fill().Height(1).Background(colors.Border)
}
