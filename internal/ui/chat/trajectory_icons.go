package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Artwork from the original ui-trajectory table/toolbar (artwork.ts and the
// bespoke wrench/information/compacted glyphs in TrajectoryTable.tsx).
var (
	trajIconClock     = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="5.25"/><path d="M8 4.75V8l2.25 1.5"/></svg>`))
	trajIconUser      = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25"><path d="M8 8.25C9.51878 8.25 10.75 7.01878 10.75 5.5C10.75 3.98122 9.51878 2.75 8 2.75C6.48122 2.75 5.25 3.98122 5.25 5.5C5.25 7.01878 6.48122 8.25 8 8.25Z"/><path d="M2.5 14.5C2.5 11.5 5.25 10.25 8 10.25C10.75 10.25 13.5 11.5 13.5 14.5"/></svg>`))
	trajIconGear      = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25"><path d="M8 9.75012C8.9665 9.75012 9.75 8.96662 9.75 8.00012C9.75 7.03362 8.9665 6.25012 8 6.25012C7.0335 6.25012 6.25 7.03362 6.25 8.00012C6.25 8.96662 7.0335 9.75012 8 9.75012Z"/><path d="M13.5 8.00012C13.5 8.16679 13.4934 8.33078 13.4809 8.49212L14.6 9.31012C14.7117 9.39145 14.7617 9.53612 14.725 9.67078L13.825 11.9508C13.79 12.0808 13.6517 12.1461 13.5234 12.1045L12.205 11.6808C11.9534 11.9461 11.6667 12.1711 11.35 12.3461L11.175 13.7078C11.16 13.8395 11.0484 13.9378 10.915 13.9378H8.585C8.45337 13.9378 8.34171 13.8411 8.32504 13.7111L8.15004 12.3518C7.83171 12.1768 7.54337 11.9495 7.29171 11.6828L5.97671 12.1061C5.84837 12.1478 5.71004 12.0828 5.67504 11.9528L4.77504 9.67278C4.73837 9.53812 4.78837 9.39345 4.90004 9.31212L6.01904 8.49412C6.00654 8.33278 5.99987 8.16879 5.99987 8.00212C5.99987 7.83545 6.00654 7.67145 6.01904 7.51012L4.90004 6.69212C4.78837 6.61078 4.73837 6.46612 4.77504 6.33145L5.67504 4.05145C5.71004 3.92145 5.84837 3.85612 5.97671 3.89778L7.29504 4.32145C7.54671 4.05612 7.83337 3.83112 8.15004 3.65612L8.32504 2.29445C8.34004 2.16278 8.45171 2.06445 8.58504 2.06445H10.915C11.0484 2.06445 11.16 2.16112 11.175 2.29112L11.35 3.65278C11.6684 3.82778 11.9567 4.05478 12.2084 4.32145L13.5267 3.89778C13.655 3.85645 13.7934 3.92145 13.8284 4.05145L14.7284 6.33145C14.765 6.46612 14.715 6.61078 14.6034 6.69212L13.484 7.51012C13.4965 7.67145 13.5032 7.83345 13.5032 8.00012"/></svg>`))
	trajIconSparkle   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25"><path d="M5.875 3C5.875 6.33333 7.54167 8 10.875 8C7.54167 8 5.875 9.66667 5.875 13C5.875 9.66667 4.20833 8 0.875 8C4.20833 8 5.875 6.33333 5.875 3Z"/><path d="M12.375 1.55823C12.375 3.39156 13.2917 4.30823 15.125 4.30823C13.2917 4.30823 12.375 5.22489 12.375 7.05823C12.375 5.22489 11.4583 4.30823 9.625 4.30823C11.4583 4.30823 12.375 3.39156 12.375 1.55823Z"/><path d="M12.375 10.4418C12.375 11.7751 13.0417 12.4418 14.375 12.4418C13.0417 12.4418 12.375 13.1084 12.375 14.4418C12.375 13.1084 11.7083 12.4418 10.375 12.4418C11.7083 12.4418 12.375 11.7751 12.375 10.4418Z"/></svg>`))
	trajIconWrench    = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 3.3a3.8 3.8 0 0 1-4.8 4.8l-5.1 5.1a1.6 1.6 0 1 1-2.3-2.3l5.1-5.1A3.8 3.8 0 0 1 11.7 1l-2.3 2.3 2.3 2.3L14 3.3Z"/></svg>`))
	trajIconInfo      = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"><circle cx="8" cy="8" r="6.7"/><circle cx="8" cy="5.5" r=".85" fill="currentColor" stroke="none"/><path d="M8 7.75v3.4" stroke-width="1.8"/></svg>`))
	trajIconCompacted = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m2.5 2.5 3.75 3.75M3 6.25h3.25V3"/><path d="m13.5 2.5-3.75 3.75M13 6.25H9.75V3"/><path d="m2.5 13.5 3.75-3.75M3 9.75h3.25V13"/><path d="m13.5 13.5-3.75-3.75M13 9.75H9.75V13"/></svg>`))
	trajIconSearch    = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25"><path d="M6.58727 11.8586C9.55061 11.8586 11.9529 9.45637 11.9529 6.49304C11.9529 3.5297 9.55061 1.12744 6.58727 1.12744C3.62394 1.12744 1.22168 3.5297 1.22168 6.49304C1.22168 9.45637 3.62394 11.8586 6.58727 11.8586Z"/><path d="M10.2991 10.3933L14.7783 14.8725"/></svg>`))
)

// trajectoryPalette carries the kind/timeline tokens the original CSS
// resolves through --dsw-alias-* names; SkinSemanticToken keeps each value
// overridable by the active family seed.
type trajectoryPalette struct {
	fg, muted, secondary, caption ui.Color
	accent, danger, hover         ui.Color
	field, modulePlatform         ui.Color
	layer2                        ui.Color
	businessTertiary              ui.Color
	warnLabel, warnTertiary       ui.Color
	successTertiary               ui.Color
	assistantViolet               ui.Color
	assistantVioletBG             ui.Color
	turnAccent                    ui.Color
	borderL1                      ui.Color
	dark                          bool
}

func newTrajectoryPalette(themeID string, systemDark bool, colors theme.Colors) trajectoryPalette {
	token := func(name, fallback string) ui.Color {
		return chrome.Color(theme.SkinSemanticToken(themeID, systemDark, name, fallback))
	}
	violet := token("--dsw-alias-trajectory-assistant", "rgb(136, 120, 224)")
	// --dsw-alias-brand-primary-new 60% + --dsw-alias-state-error-secondary.
	assistant := token("--dsw-alias-trajectory-message", "rgb(144, 122, 205)")
	p := trajectoryPalette{
		fg:                chrome.Color(colors.FG),
		muted:             chrome.Color(colors.Muted),
		secondary:         chrome.Color(colors.Secondary),
		caption:           chrome.Color(colors.Caption),
		accent:            chrome.Color(colors.Accent),
		danger:            chrome.Color(colors.Danger),
		hover:             chrome.Color(colors.Hover),
		field:             chrome.Color(colors.Field),
		layer2:            chrome.Color(colors.SelectorFill),
		businessTertiary:  token("--dsw-alias-state-business-tertiary", "#e8effe"),
		warnLabel:         token("--dsw-alias-state-warn-label", "#b35c00"),
		warnTertiary:      token("--dsw-alias-state-warn-tertiary", "#fdf0e0"),
		successTertiary:   token("--dsw-alias-state-success-tertiary", "#e5f6ec"),
		assistantViolet:   assistant,
		assistantVioletBG: assistant.Alpha(.15),
		turnAccent:        token("--dsw-alias-trajectory-accent", "#9db4d0").Alpha(.22),
		borderL1:          chrome.Color(colors.BorderL1),
		dark:              systemDark,
	}
	_ = violet
	return p
}

// trajKind is the original TrajectoryCellKind the native message maps to.
func trajCellKind(row trajRow) string {
	switch row.Kind {
	case session.MessageKindTool:
		return "tool"
	case session.MessageKindCompaction:
		return "compacted"
	case session.MessageKindThinking:
		return "context"
	case trajectoryRequestKind:
		return "context"
	default:
		switch row.Role {
		case "user":
			return "user"
		case "system":
			return "system"
		}
		return "message"
	}
}

// trajKindBadge resolves the kindTag face (icon, ink, fill) for the badge slot.
func trajKindBadge(p trajectoryPalette, kind string) (*ui.SVG, ui.Color, ui.Color) {
	switch kind {
	case "user":
		return trajIconUser, p.accent, p.businessTertiary
	case "system":
		return trajIconGear, p.secondary, p.modulePlatform
	case "context":
		return trajIconInfo, p.secondary, p.successTertiary
	case "compacted":
		return trajIconCompacted, p.secondary, p.modulePlatform
	case "tool":
		return trajIconWrench, p.warnLabel, p.warnTertiary
	}
	return trajIconSparkle, p.assistantViolet, p.assistantVioletBG
}

// trajKindTagLabel is the 10px kindTag label shown in the wide variant.
func trajKindTagLabel(kind string) string {
	switch kind {
	case "user":
		return trajectoryText("用户", "User")
	case "system":
		return trajectoryText("系统", "System")
	case "context":
		return trajectoryText("请求", "Request")
	case "compacted":
		return trajectoryText("已压缩", "Compacted")
	case "tool":
		return trajectoryText("工具", "Tool")
	}
	return trajectoryText("助手", "Assistant")
}

// trajTimelineColor is the .span fill: user blue, context green, message
// violet, tool warn amber, secondary otherwise, error overrides all.
func trajTimelineColor(p trajectoryPalette, row trajRow, kind string) ui.Color {
	if row.Source.EventType == "tool/result" && row.Source.Tool != nil && row.Source.Tool.State == "error" {
		return p.danger
	}
	switch kind {
	case "user":
		return p.accent
	case "context":
		return p.secondary
	case "tool":
		return p.warnLabel
	case "compacted":
		return p.secondary
	default:
		return p.assistantViolet
	}
}
