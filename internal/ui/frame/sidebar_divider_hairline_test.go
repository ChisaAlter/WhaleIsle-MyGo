package frame

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/sidebar"
	"whaleisle/internal/ui/theme"
)

// TestSidebarDividerSinglePhysicalPixelAt150DPI verifies that in frosted glass
// mode at 150% DPI the boundary between sidebar.View and the main column renders
// exactly one single 1-physical-pixel hairline owned by sidebarColumns, with no
// duplicate sidebar-glass-hairline on the left and no raw-wallpaper gap on the right.
func TestSidebarDividerSinglePhysicalPixelAt150DPI(t *testing.T) {
	t.Cleanup(func() {
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})

	pixelL1 := func(a, b color.RGBA) int {
		abs := func(v int) int {
			if v < 0 {
				return -v
			}
			return v
		}
		return abs(int(a.R)-int(b.R)) + abs(int(a.G)-int(b.G)) + abs(int(a.B)-int(b.B))
	}

	for _, dark := range []bool{false, true} {
		for _, hideMask := range []bool{false, true} {
			name := fmt.Sprintf("dark=%v/hideMask=%v", dark, hideMask)
			t.Run(name, func(t *testing.T) {
				prefs := wallpapers.Defaults()
				prefs.Glass = 80
				prefs.HideSidebarMask = hideMask
				chrome.ConfigureBackdrop("wall.png", prefs)
				chrome.SetBackdropDark(dark)
				design.SetGlass(true, func(s design.Scheme, p *design.Palette) {
					if s == design.Dark {
						p.Background = ui.RGBA(21, 21, 23, 0.30)
						p.Sidebar = ui.RGBA(27, 27, 28, 0.55)
						p.Border = ui.RGBA(255, 255, 255, 0.14)
						p.SidebarBorder = ui.RGBA(255, 255, 255, 0.10)
					} else {
						p.Background = ui.RGBA(255, 255, 255, 0.24)
						p.Sidebar = ui.RGBA(249, 250, 251, 0.40)
						p.Border = ui.RGBA(0, 0, 0, 0.14)
						p.SidebarBorder = ui.RGBA(0, 0, 0, 0.10)
					}
				})

				themeID := theme.IDDeepseekLight
				backdrop := ui.RGB(80, 195, 210)
				if dark {
					themeID = theme.IDDeepseekDark
					backdrop = ui.RGB(10, 95, 72)
				}

				sizes := LayoutSizes{Sidebar: 280}
				tt := ui.NewTester(func(c *ui.Context) {
					if dark {
						c.SetTheme(ui.DarkTheme())
						design.Bind(c, design.Dark, ui.Hex("#7aaaff"), ui.Hex("#0f1115"))
					} else {
						c.SetTheme(ui.LightTheme())
						design.Bind(c, design.Light, ui.Hex("#4176e6"), ui.Hex("#ffffff"))
					}
					ui.Column(c).Fill().Background(backdrop).Children(func() {
						sidebarColumns(c, &sizes, func() {
							sidebar.View(c, sidebar.Model{
								ThemeID:    themeID,
								SystemDark: dark,
							})
						}, func() {
							ui.Box(c).Fill().Background(frameCanvasFill(tk(c)))
						})
					})
				}, 1200, 120)
				tt.SetScale(1.5)
				tt.SetPreferences(ui.Preferences{ReduceMotion: true})
				tt.Frame()

				img := tt.Image()
				// Sidebar is 280 DIP -> [0, 420) px; divider is [280, 281] DIP -> [420, 422) px;
				// main starts at 422 px. Sample a clean horizontal line mid-height (y = 90 px).
				y := 90
				sbInner := img.RGBAAt(416, y)
				sbEdge := img.RGBAAt(419, y)
				if diff := pixelL1(sbEdge, sbInner); diff > 2 {
					t.Fatalf("%s duplicate sidebar-glass-hairline detected at x=419: edge=%v inner=%v (L1=%d)", name, sbEdge, sbInner, diff)
				}

				mainRef := img.RGBAAt(424, y)
				hairlineCols := 0
				for x := 420; x <= 423; x++ {
					px := img.RGBAAt(x, y)
					if pixelL1(px, mainRef) > 2 {
						hairlineCols++
					}
				}
				if hairlineCols != 1 {
					t.Fatalf("%s expected exactly 1 physical hairline pixel in [420..423], got %d (419=%v 420=%v 421=%v 422=%v 424=%v)",
						name, hairlineCols,
						img.RGBAAt(419, y), img.RGBAAt(420, y), img.RGBAAt(421, y), img.RGBAAt(422, y), mainRef)
				}
			})
		}
	}
}
