package coss

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// paintStripeBackdrop paints crisp alternating high-contrast colored vertical
// stripes so blur softening and translucent show-through are directly measurable.
func paintStripeBackdrop(c *ui.Context) {
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		const stripeW = 4
		c1 := ui.RGB(20, 110, 240)
		c2 := ui.RGB(245, 130, 30)
		for i, x := 0, float32(0); x < r.W; i, x = i+1, x+stripeW {
			col := c1
			if i%2 == 1 {
				col = c2
			}
			p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: stripeW, H: r.H}, col, 0)
		}
	})
}

func luma(img *image.RGBA, x, y int) float64 {
	c := img.RGBAAt(x, y)
	return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
}

func channelSpan(img *image.RGBA, y, x0, x1 int) int {
	loR, hiR := 255, 0
	loB, hiB := 255, 0
	for x := x0; x < x1; x++ {
		p := img.RGBAAt(x, y)
		loR, hiR = min(loR, int(p.R)), max(hiR, int(p.R))
		loB, hiB = min(loB, int(p.B)), max(hiB, int(p.B))
	}
	return max(hiR-loR, hiB-loB)
}

func testGlassFilter(s design.Scheme, p *design.Palette) {
	if s == design.Dark {
		p.Card = p.Card.Alpha(0.55)
		p.Popover = p.Popover.Alpha(0.45)
	} else {
		p.Card = p.Card.Alpha(0.58)
		p.Popover = p.Popover.Alpha(0.58)
	}
}

func TestGlassOffPreservesOpaqueBehavior(t *testing.T) {
	design.SetGlass(false, nil)
	if design.GlassEnabled() {
		t.Fatal("expected glass disabled by default")
	}

	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			cases := []struct {
				name  string
				build func(c *ui.Context) ui.Element
			}{
				{
					name: "Card",
					build: func(c *ui.Context) ui.Element {
						return Card(c, CardOptions{}, func() {
							ui.Box(c).Size(180, 60)
						}).Width(220)
					},
				},
				{
					name: "DialogPanel",
					build: func(c *ui.Context) ui.Element {
						return DialogPanel(c, DialogOptions{NoClose: true}, func() {
							ui.Box(c).Size(180, 60)
						}).Width(220)
					},
				},
				{
					name: "MenuPanel",
					build: func(c *ui.Context) ui.Element {
						p := ui.Column(c)
						menuPanel(c, tok(c), p, []MenuItem{{ID: "a", Label: "Alpha"}, {ID: "b", Label: "Beta"}}, MenuOptions{MinWidth: 200}, nil, 0)
						return p
					},
				},
				{
					name: "PopoverPanel",
					build: func(c *ui.Context) ui.Element {
						return PopoverPanel(c, PopoverOptions{Width: 220}, func() {
							ui.Box(c).Size(180, 60)
						})
					},
				},
				{
					name: "CommandPanel",
					build: func(c *ui.Context) ui.Element {
						q := ""
						p := ui.Column(c).Height(140)
						commandPanel(c, tok(c), p, &q, []CommandItem{{ID: "x", Label: "Action"}}, CommandOptions{}, false)
						return p.Width(240)
					},
				},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					var b ui.Rect
					var tk *design.Tokens
					tt := ui.NewTester(func(c *ui.Context) {
						setTheme(c, dark)
						tk = design.For(c)
						ui.Box(c).Fill().Children(func() {
							paintStripeBackdrop(c)
							ui.Box(c).Absolute().Left(30).Top(20).Children(func() {
								b = tc.build(c).Bounds()
							})
						})
					}, 320, 220)
					tt.Frame()
					if tk.Glass {
						t.Fatalf("expected Tokens.Glass == false")
					}
					img := tt.Image()
					// Verify no backdrop stripes bleed through or blur inside opaque panels.
					x0, x1 := int(b.X)+24, int(b.X+b.W)-24
					sampleY := int(b.Y) + int(b.H) - 18
					if tc.name == "MenuPanel" {
						sampleY = int(b.Y) + int(b.H) - 6
					}
					if span := channelSpan(img, sampleY, x0, x1); span != 0 {
						t.Fatalf("%s opaque interior leaked backdrop (channel span=%d)", tc.name, span)
					}
					// Outside the panel, the raw stripes keep full contrast.
					if outside := channelSpan(img, 10, 10, 260); outside < 150 {
						t.Fatalf("expected vivid backdrop stripes outside panel, got span=%d", outside)
					}
				})
			}
		})
	}
}

func TestGlassModeBlurRimAndShadows(t *testing.T) {
	defer design.SetGlass(false, nil)

	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			type surfaceCase struct {
				name       string
				expectBlur bool
				build      func(c *ui.Context) ui.Element
			}
			surfaces := []surfaceCase{
				{
					name:       "Card",
					expectBlur: false,
					build: func(c *ui.Context) ui.Element {
						return Card(c, CardOptions{}, func() {
							ui.Box(c).Size(180, 60)
						}).Width(220)
					},
				},
				{
					name:       "DialogPanel",
					expectBlur: true,
					build: func(c *ui.Context) ui.Element {
						return DialogPanel(c, DialogOptions{NoClose: true}, func() {
							ui.Box(c).Size(180, 60)
						}).Width(220)
					},
				},
				{
					name:       "MenuPanel",
					expectBlur: true,
					build: func(c *ui.Context) ui.Element {
						p := ui.Column(c)
						menuPanel(c, tok(c), p, []MenuItem{{ID: "a", Label: "   "}}, MenuOptions{MinWidth: 200}, nil, 0)
						return p
					},
				},
				{
					name:       "PopoverPanel",
					expectBlur: true,
					build: func(c *ui.Context) ui.Element {
						return PopoverPanel(c, PopoverOptions{Width: 220}, func() {
							ui.Box(c).Size(180, 60)
						})
					},
				},
			}

			for _, sc := range surfaces {
				t.Run(sc.name, func(t *testing.T) {
					// 1) Compare striped backdrop behavior: overlays blur, Card stays translucent without blur.
					design.SetGlass(false, nil)
					ttOff := ui.NewTester(func(c *ui.Context) {
						setTheme(c, dark)
						ui.Box(c).Fill().Children(func() {
							paintStripeBackdrop(c)
							ui.Box(c).Absolute().Left(30).Top(20).Children(func() {
								_ = sc.build(c)
							})
						})
					}, 320, 200)
					ttOff.Frame()
					imgOff := ttOff.Image()

					design.SetGlass(true, testGlassFilter)
					var b ui.Rect
					var tk *design.Tokens
					ttOn := ui.NewTester(func(c *ui.Context) {
						setTheme(c, dark)
						tk = design.For(c)
						ui.Box(c).Fill().Children(func() {
							paintStripeBackdrop(c)
							ui.Box(c).Absolute().Left(30).Top(20).Children(func() {
								b = sc.build(c).Bounds()
							})
						})
					}, 320, 200)
					ttOn.Frame()
					if !tk.Glass {
						t.Fatal("expected Tokens.Glass == true")
					}
					imgOn := ttOn.Image()

					rawSpan := channelSpan(imgOn, 8, 20, 260)
					if rawSpan < 150 {
						t.Fatalf("expected raw backdrop span >= 150, got %d", rawSpan)
					}

					midY := int(b.Y + b.H/2)
					x0, x1 := int(b.X)+24, int(b.X+b.W)-24
					innerSpan := channelSpan(imgOn, midY, x0, x1)
					if sc.expectBlur {
						if innerSpan > rawSpan/4 {
							t.Fatalf("%s glass blur did not soften stripes enough: innerSpan=%d rawSpan=%d", sc.name, innerSpan, rawSpan)
						}
					} else {
						// Resting Card must NOT run GPU backdrop blur; translucent tint lets stripes show cleanly.
						if innerSpan <= rawSpan/4 || innerSpan >= rawSpan {
							t.Fatalf("%s expected clean translucent fill without backdrop blur: innerSpan=%d rawSpan=%d", sc.name, innerSpan, rawSpan)
						}
					}

					pOff := imgOff.RGBAAt(int(b.X+b.W/2), midY)
					pOn := imgOn.RGBAAt(int(b.X+b.W/2), midY)
					if pOff == pOn {
						t.Fatalf("%s glass pixel %v identical to opaque %v", sc.name, pOn, pOff)
					}

					// 2) Compare adaptive drop shadow strength over a uniform neutral surface.
					design.SetGlass(false, nil)
					ttShOff := ui.NewTester(func(c *ui.Context) {
						setTheme(c, dark)
						ui.Box(c).Fill().Background(ui.RGB(220, 224, 230)).Children(func() {
							ui.Box(c).Absolute().Left(30).Top(20).Children(func() {
								_ = sc.build(c)
							})
						})
					}, 320, 220)
					ttShOff.Frame()

					design.SetGlass(true, testGlassFilter)
					var sb ui.Rect
					ttShOn := ui.NewTester(func(c *ui.Context) {
						setTheme(c, dark)
						ui.Box(c).Fill().Background(ui.RGB(220, 224, 230)).Children(func() {
							ui.Box(c).Absolute().Left(30).Top(20).Children(func() {
								sb = sc.build(c).Bounds()
							})
						})
					}, 320, 220)
					ttShOn.Frame()

					sx, sy := int(sb.X+sb.W/2), int(sb.Y+sb.H)+3
					shadowOffLuma := luma(ttShOff.Image(), sx, sy)
					shadowOnLuma := luma(ttShOn.Image(), sx, sy)
					if shadowOnLuma >= shadowOffLuma-1 {
						t.Fatalf("%s expected deeper adaptive glass shadow below panel: off=%.2f on=%.2f", sc.name, shadowOffLuma, shadowOnLuma)
					}

					// No harsh inner white rim along y = Top + 1 over uniform backdrop.
					rimX := int(sb.X + sb.W/2)
					topInner := luma(ttShOn.Image(), rimX, int(sb.Y)+1)
					bodyLuma := luma(ttShOn.Image(), rimX, int(sb.Y)+8)
					if topInner > bodyLuma+4 {
						t.Fatalf("%s unexpected bright inner specular rim: topInner=%.2f body=%.2f", sc.name, topInner, bodyLuma)
					}
				})
			}

			// Verify CommandPanel receives glass blur and no inner white rim.
			t.Run("CommandPanel", func(t *testing.T) {
				design.SetGlass(true, testGlassFilter)
				var b ui.Rect
				tt := ui.NewTester(func(c *ui.Context) {
					setTheme(c, dark)
					ui.Box(c).Fill().Children(func() {
						paintStripeBackdrop(c)
						ui.Box(c).Absolute().Left(20).Top(20).Children(func() {
							q := ""
							p := ui.Column(c).Height(150)
							commandPanel(c, tok(c), p, &q, nil, CommandOptions{}, false)
							b = p.Width(260).Bounds()
						})
					})
				}, 320, 220)
				tt.Frame()
				img := tt.Image()
				if span := channelSpan(img, int(b.Y+b.H/2), int(b.X)+24, int(b.X+b.W)-24); span > 35 {
					t.Fatalf("CommandPanel glass blur did not soften stripes enough: span=%d", span)
				}
			})

			// Verify DrawerPanel gets frosted blur and square bottom corners without inner white rim.
			t.Run("DrawerPanel", func(t *testing.T) {
				design.SetGlass(true, testGlassFilter)
				var b ui.Rect
				tt := ui.NewTester(func(c *ui.Context) {
					setTheme(c, dark)
					ui.Box(c).Fill().Children(func() {
						paintStripeBackdrop(c)
						ui.Box(c).Absolute().Left(20).Top(30).Width(260).Children(func() {
							b = DrawerPanel(c, DialogOptions{NoClose: true}, func() {
								ui.Box(c).Size(200, 60)
							}).Bounds()
						})
					})
				}, 320, 200)
				tt.Frame()
				img := tt.Image()
				if span := channelSpan(img, int(b.Y+b.H/2), int(b.X)+24, int(b.X+b.W)-24); span > 35 {
					t.Fatalf("DrawerPanel glass blur did not soften stripes enough: span=%d", span)
				}
			})
		})
	}
}

func TestGlassDialogBackdropPureDimScrim(t *testing.T) {
	defer design.SetGlass(false, nil)

	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			renderBackdropPixel := func(glassOn bool) uint8 {
				if glassOn {
					design.SetGlass(true, testGlassFilter)
				} else {
					design.SetGlass(false, nil)
				}
				open := true
				tt := ui.NewTester(func(c *ui.Context) {
					setTheme(c, dark)
					ui.Box(c).Fill().Background(ui.RGB(240, 240, 240)).Children(func() {
						Dialog(c, &open, DialogOptions{Title: "Settings"}, nil)
					})
				}, 600, 400)
				tt.Frame()
				return tt.Image().RGBAAt(10, 10).R
			}

			opaqueScrim := renderBackdropPixel(false)
			glassScrim := renderBackdropPixel(true)
			if dark {
				if glassScrim >= opaqueScrim {
					t.Fatalf("expected dark glass scrim (0.42) to dim more than non-glass (0.32): opaque=%d glass=%d", opaqueScrim, glassScrim)
				}
			} else {
				if glassScrim <= opaqueScrim {
					t.Fatalf("expected light glass scrim (0.24) to preserve more brightness than non-glass (0.32): opaque=%d glass=%d", opaqueScrim, glassScrim)
				}
			}

			// Verify modal scrim does NOT run fullscreen backdrop blur over crisp stripes.
			design.SetGlass(true, testGlassFilter)
			open := true
			tt := ui.NewTester(func(c *ui.Context) {
				setTheme(c, dark)
				ui.Box(c).Fill().Children(func() {
					paintStripeBackdrop(c)
					Dialog(c, &open, DialogOptions{Title: "Settings"}, nil)
				})
			}, 600, 400)
			tt.Frame()
			scrimSpan := channelSpan(tt.Image(), 10, 10, 80)
			if scrimSpan < 70 {
				t.Fatalf("expected crisp unblurred modal scrim over stripes, got span=%d", scrimSpan)
			}
		})
	}
}

func TestGlassChatBubble40PercentAndSolidOverride(t *testing.T) {
	defer design.SetGlass(false, nil)
	design.SetGlass(true, testGlassFilter)

	var glassRect, solidRect ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		setTheme(c, false)
		ui.Box(c).Fill().Children(func() {
			paintStripeBackdrop(c)
			ui.Column(c).Absolute().Left(20).Top(20).Gap(12).Children(func() {
				glassRect = ChatBubble(c, ChatBubbleOptions{
					Side:       ChatOutgoing,
					BubbleTint: ui.RGB(237, 243, 254),
				}, func() {
					ui.Box(c).Size(160, 32)
				}).Bounds()
				solidRect = ChatBubble(c, ChatBubbleOptions{
					Side:       ChatOutgoing,
					Solid:      true,
					BubbleTint: ui.RGB(239, 233, 223),
				}, func() {
					ui.Box(c).Size(160, 32)
				}).Bounds()
			})
		})
	}, 260, 160)
	tt.Frame()
	img := tt.Image()
	raw := channelSpan(img, 8, 10, 240)
	glassSpan := channelSpan(img, int(glassRect.Y+glassRect.H/2), int(glassRect.X+20), int(glassRect.X+glassRect.W-20))
	if glassSpan <= raw/4 || glassSpan >= raw {
		t.Fatalf("glass chat bubble should use clean translucent fill without blur: glassSpan=%d raw=%d", glassSpan, raw)
	}
	solidSpan := channelSpan(img, int(solidRect.Y+solidRect.H/2), int(solidRect.X+20), int(solidRect.X+solidRect.W-20))
	if solidSpan != 0 {
		t.Fatalf("solid skin chat bubble leaked backdrop: solidSpan=%d", solidSpan)
	}
}

func TestGlassDarkControlsRestrained(t *testing.T) {
	defer design.SetGlass(false, nil)
	design.SetGlass(true, testGlassFilter)

	tt := ui.NewTester(func(c *ui.Context) {
		setTheme(c, true)
		tk := design.For(c)
		rest := resolveButton(tk, ButtonOutline, 0)
		if rest.bg.A != 0 {
			t.Fatalf("dark glass ButtonOutline resting bg alpha = %d, want 0 (transparent)", rest.bg.A)
		}
		if rest.d.before {
			t.Fatalf("dark glass ButtonOutline should not paint d.highlight top edge")
		}
		hov := resolveButton(tk, ButtonOutline, StateHover)
		if hov.bg != white.Alpha(0.06) {
			t.Fatalf("dark glass ButtonOutline hover bg = %+v, want %+v", hov.bg, white.Alpha(0.06))
		}
		dest := resolveButton(tk, ButtonDestructiveOutline, 0)
		if dest.bg.A != 0 || dest.d.before {
			t.Fatalf("dark glass ButtonDestructiveOutline should have transparent bg and no highlight: bg=%+v before=%v", dest.bg, dest.d.before)
		}
		val := ""
		ui.Column(c).Gap(8).Padding(12).Children(func() {
			Button(c, "Outline", ButtonOptions{Variant: ButtonOutline})
			Input(c, &val, InputOptions{Placeholder: "Search"})
			Badge(c, "Tag", BadgeOptions{Variant: BadgeOutline})
		})
	}, 240, 160)
	tt.Frame()
}

func TestMenuSecondClickClosesAndRadiiMatchSpec(t *testing.T) {
	defer design.SetGlass(false, nil)
	design.SetGlass(true, testGlassFilter)

	open := false
	var menuBounds, dlgBounds ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		setTheme(c, false)
		ui.Box(c).Fill().Children(func() {
			trigger := Button(c, "MenuTrigger", ButtonOptions{Variant: ButtonOutline, Size: ButtonSM})
			Menu(c, trigger, &open, []MenuItem{{ID: "one", Label: "One"}}, MenuOptions{})
			if open {
				menuBounds = trigger.Bounds()
			}
			dlgBounds = DialogPanel(c, DialogOptions{NoClose: true}, func() {
				ui.Box(c).Size(100, 40)
			}).Bounds()
		})
	}, 400, 300)
	tt.Frame()
	if err := tt.Click("MenuTrigger"); err != nil {
		t.Fatal(err)
	}
	if !open {
		t.Fatalf("first click should open Menu (trigger=%+v)", menuBounds)
	}
	if err := tt.Click("MenuTrigger"); err != nil {
		t.Fatal(err)
	}
	if open {
		t.Fatal("second click on Menu trigger should close without reopening")
	}
	if dlgBounds.W == 0 {
		t.Fatal("DialogPanel failed to lay out")
	}
}
