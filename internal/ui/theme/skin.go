package theme

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

const (
	SkinDefault = "default"
	SkinCodex   = "codex"
	SkinClaude  = "claude"
)

// ClaudeProseFamily is skins.css's complete conversation serif stack. It does
// not replace code fonts, tool labels, brand text or the saved font preferences.
const ClaudeProseFamily = `"Source Serif 4", "Iowan Old Style", "Charter", Georgia, Cambria, "Times New Roman", "Songti SC", "Noto Serif SC", "Source Han Serif SC", SimSun, serif`

// FamilySkin reads presentation metadata; unknown and legacy documents keep the
// product chrome. Skin is explicit on copies, rather than inferred from names.
func FamilySkin(f Family) string {
	switch f.Skin {
	case SkinCodex, SkinClaude:
		return f.Skin
	default:
		return SkinDefault
	}
}

// SkinForMode selects the same half as the resolved palette. Its mode argument
// is the effective light/dark mode, not necessarily the OS appearance.
func SkinForMode(p Preferences, dark bool) string {
	id := p.Light
	if dark {
		id = p.Dark
	}
	return FamilySkin(p.Family(id))
}

// ResolvedSkin follows pinned light/dark and system changes without writing a
// derived style into the user's durable font preferences.
func ResolvedSkin(id string, systemDark bool) string {
	dark := ResolveWithSystem(id, systemDark).Scheme == SchemeDark
	return SkinForMode(CurrentPreferences(), dark)
}

func ContextSkin(c *ui.Context) string { return SkinForMode(CurrentPreferences(), c.Theme().Dark) }

// SkinFontRole names CSS scopes, not global font preferences. Assistant prose
// and user bubbles share the serif face; the surrounding controls do not.
type SkinFontRole uint8

const (
	SkinInterfaceFont SkinFontRole = iota
	SkinProseFont
	SkinComposerFont
	SkinHeroFont
)

// FontForSkin returns drawing-only role typography. Codex deliberately uses the
// normal font stack. Claude's CSS places Inter before custom interface fonts
// and fixes conversation prose independently of user font selections.
func FontForSkin(p Preferences, skin string, role SkinFontRole) FontPreference {
	var font FontPreference
	switch role {
	case SkinProseFont:
		font = p.Chat
	case SkinComposerFont:
		font = p.Composer
	default:
		font = p.Interface
	}
	if skin != SkinClaude {
		return font
	}
	EnsureSkinFonts()
	if role == SkinInterfaceFont {
		if !strings.HasPrefix(font.Family, `"Inter", `) {
			font.Family = `"Inter", ` + font.Family
		}
	} else {
		font.Family = ClaudeProseFamily
	}
	return font
}

func ContextFont(c *ui.Context, role SkinFontRole) FontPreference {
	return FontForSkin(RenderPreferences(), ContextSkin(c), role)
}
