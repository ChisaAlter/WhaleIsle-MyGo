package theme

// SkinLayout is the installed 0.3.10 skins.css geometry, independent of viewport
// state and durable preferences. Zero ContentWidth means the default fluid axis.
type SkinLayout struct {
	Skin                                                                       string
	ContentWidth, ComposerRadius, ComposerBorder, SidebarBorder, HeaderBorder  float32
	SidebarInline, SidebarFont, BrandHeight, BrandBottom, NewHeight, NavHeight float32
	BubbleRadii                                                                [4]float32
	ChatBackground, ComposerBackground, BorderColor, SidebarBorderColor        string
	ComposerShadow                                                             bool
}

func LayoutForSkin(skin string) SkinLayout {
	l := SkinLayout{Skin: SkinDefault, ComposerRadius: 28, ComposerBorder: .5, SidebarInline: 12, SidebarFont: 14, BrandHeight: 60, NewHeight: 38, NavHeight: 36, BubbleRadii: [4]float32{20, 20, 20, 20}}
	switch skin {
	case SkinCodex:
		l.Skin, l.ContentWidth, l.ComposerRadius = skin, 720, 18
		l.SidebarBorder, l.SidebarInline, l.SidebarFont = .5, 10, 13
		l.BrandHeight, l.BrandBottom, l.NewHeight, l.NavHeight = 44, 4, 32, 32
		l.BubbleRadii = [4]float32{18, 18, 4, 18}
		l.ComposerShadow = true
	case SkinClaude:
		l.Skin, l.ContentWidth, l.ComposerRadius = skin, 720, 22
		l.ComposerBorder, l.SidebarBorder, l.HeaderBorder, l.SidebarFont = 1, 1, 1, 13
		l.BubbleRadii = [4]float32{12, 12, 12, 12}
	}
	return l
}
func (l SkinLayout) Active() bool { return l.Skin != SkinDefault }

func layoutForMode(p Preferences, dark bool, c Colors) SkinLayout {
	l := LayoutForSkin(SkinForMode(p, dark))
	l.ChatBackground, l.ComposerBackground, l.BorderColor, l.SidebarBorderColor = c.BG, c.InputMajor, c.Border, c.BorderL1
	if !l.Active() {
		l.SidebarFont = 14 * p.Interface.Size / 16
	}
	id := p.Light
	if dark {
		id = p.Dark
	}
	family := p.Family(id)
	seed := family.Light
	if dark {
		seed = family.Dark
	}
	token := func(name, fallback string) string {
		if s := seed.Overrides[name]; s != "" {
			return s
		}
		return fallback
	}
	if l.Skin == SkinCodex {
		l.ChatBackground, l.ComposerBackground = c.Field, c.Field
	} else if l.Skin == SkinClaude {
		l.ChatBackground = token("--dsh-claude-paper", c.BG)
		l.ComposerBackground = token("--dsh-claude-panel", c.InputMajor)
		l.BorderColor = token("--dsh-claude-border", c.Border)
		l.SidebarBorderColor = l.BorderColor
	}
	return l
}

// HeroBadgeColors resolves the installed business-tertiary / primary-bluish
// aliases. Skin and imported-family token overrides retain precedence.
func HeroBadgeColors(id string, systemDark bool) (background, foreground string) {
	t := ResolveWithSystem(id, systemDark)
	dark := t.Scheme == SchemeDark
	p := CurrentPreferences()
	family := p.Family(p.Light)
	if dark {
		family = p.Family(p.Dark)
	}
	background, foreground = "#e4edfd", "#0e3074"
	if dark {
		background, foreground = "#34415b", "#f9fafb"
	}
	if family.ID != "deepseek" {
		seed := family.Light
		if dark {
			seed = family.Dark
		}
		f := seed.ContrastValue() / 100
		if f < 0 {
			f = 0
		}
		if f > 1 {
			f = 1
		}
		wash := .1 + f*.08
		if dark {
			wash = .14 + f*.10
		}
		background = mix(seed.Background, seed.Accent, wash)
	}
	background = SkinSemanticToken(id, systemDark, "--dsw-alias-state-business-tertiary", background)
	foreground = SkinSemanticToken(id, systemDark, "--dsw-alias-label-primary-bluish", foreground)
	return
}

func ResolvedSkinLayout(id string, systemDark bool) SkinLayout {
	t := ResolveWithSystem(id, systemDark)
	dark := t.Scheme == SchemeDark
	return layoutForMode(CurrentPreferences(), dark, t.Active(dark))
}

// SkinSemanticToken reads the selected half's unexposed source tokens. The
// fallback is already a resolved semantic color, not a baked light/dark value.
func SkinSemanticToken(id string, systemDark bool, name, fallback string) string {
	p := CurrentPreferences()
	family := p.Family(p.Light)
	dark := ResolveWithSystem(id, systemDark).Scheme == SchemeDark
	if dark {
		family = p.Family(p.Dark)
	}
	seed := family.Light
	if dark {
		seed = family.Dark
	}
	if value := seed.Overrides[name]; value != "" {
		return value
	}
	return fallback
}
