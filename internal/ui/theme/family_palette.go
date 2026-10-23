package theme

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func familyLuminance(color string) float64 {
	n, _ := strconv.ParseUint(strings.TrimPrefix(color, "#"), 16, 24)
	channel := func(shift uint) float64 {
		v := float64((n>>shift)&255) / 255
		if v <= .03928 {
			return v / 12.92
		}
		return math.Pow((v+.055)/1.055, 2.4)
	}
	return .2126*channel(16) + .7152*channel(8) + .0722*channel(0)
}
func familyContrast(a, b string) float64 {
	left, right := familyLuminance(a), familyLuminance(b)
	return (math.Max(left, right) + .05) / (math.Min(left, right) + .05)
}
func familyAlpha(color string, alpha float64) string {
	n, _ := strconv.ParseUint(strings.TrimPrefix(color, "#"), 16, 24)
	return fmt.Sprintf("rgba(%d, %d, %d, %.3f)", (n>>16)&255, (n>>8)&255, n&255, alpha)
}

// Native semantic tokens follow ui-theme/derive.ts. Unexposed CSS variables
// remain in the document so importing/exporting a family does not erase them.
func derivePalette(c Colors, s Seed) Colors {
	f := math.Max(0, math.Min(1, s.ContrastValue()/100))
	dark := familyLuminance(s.Background) < familyLuminance(s.Foreground)
	wash, strong := .10+f*.08, .18+f*.10
	cardFG, cardAccent, overlayFG, overlayAccent, layerFG, layerAccent := .006+f*.016, .05+f*.04, .012+f*.02, .06+f*.04, .02+f*.02, .07+f*.04
	if dark {
		wash, strong = .14+f*.10, .26+f*.12
		cardFG, cardAccent, overlayFG, overlayAccent, layerFG, layerAccent = .02+f*.04, .08+f*.05, .035+f*.05, .10+f*.05, .05+f*.05, .12+f*.06
	}
	c.Field = mix(mix(s.Background, s.Foreground, cardFG), s.Accent, cardAccent)
	c.Titlebar = mix(mix(s.Background, s.Foreground, overlayFG), s.Accent, overlayAccent)
	c.SelectorFill = mix(mix(s.Background, s.Foreground, layerFG), s.Accent, layerAccent)
	c.Muted, c.Caption = deepseekLight.Muted, deepseekLight.Caption
	if dark {
		c.Muted, c.Caption = deepseekDark.Muted, deepseekDark.Caption
	}
	c.Secondary = mix(s.Foreground, s.Background, .38-f*.14)
	c.Border = familyAlpha(s.Foreground, .1+f*.2)
	c.BorderL1 = familyAlpha(s.Foreground, .08+f*.18)
	c.BorderL4 = familyAlpha(s.Foreground, .16+f*.08)
	c.Sidebar = mix(s.Background, s.Accent, strong)
	c.Bubble = mix(s.Background, s.Accent, wash)
	c.BubbleHighlight = mix(s.Background, s.Accent, strong)
	c.InputMajor = c.Titlebar
	c.ButtonFG = s.Foreground
	for _, candidate := range []string{s.Background, "#ffffff", "#0f1115"} {
		if familyContrast(s.Accent, candidate) > familyContrast(s.Accent, c.ButtonFG) {
			c.ButtonFG = candidate
		}
	}
	for name, value := range s.Overrides {
		if value == "" {
			continue
		}
		switch name {
		case "--dsw-alias-bg-base":
			c.BG = value
		case "--dsw-alias-bg-layer-1":
			c.Field = value
		case "--dsw-alias-bg-layer-2":
			c.SelectorFill = value
		case "--dsw-alias-bg-overlay":
			c.Titlebar = value
			c.InputMajor = value
		case "--dsw-alias-label-primary":
			c.FG = value
		case "--dsw-alias-label-secondary":
			c.Secondary = value
		case "--dsw-alias-label-primary-foreground":
			c.ButtonFG = value
		case "--dsw-alias-brand-primary", "--dsw-alias-state-business-primary":
			c.Accent = value
		case "--dsw-alias-button-info-fill":
			c.ButtonInfoFill = value
		case "--dsw-alias-button-info-hover":
			c.ButtonInfoHover = value
		case "--dsw-alias-border-l1":
			c.BorderL1 = value
		case "--dsw-alias-border-l2":
			c.Border = value
		case "--dsw-alias-label-tertiary":
			c.Muted = value
		case "--dsw-alias-label-caption":
			c.Caption = value
		case "--dsw-alias-markdown-code-block":
			c.CodeBlock = value
		case "--dsw-alias-interactive-bg-hover":
			c.Hover = value
		case "--dsw-alias-interactive-bg-active":
			c.Pressed = value
		case "--dsw-specific-sidebar-fill":
			c.Sidebar = value
		case "--dsw-specific-bubble":
			c.Bubble = value
		case "--dsw-specific-bubble-highlight":
			c.BubbleHighlight = value
		}
	}
	// The trigger aliases follow the platform sheet's light/dark selection;
	// custom code-block overrides flow through the light alias before an
	// explicit trigger override is applied.
	hover, active := deepseekLight.Hover, deepseekLight.Pressed
	if dark {
		hover, active = deepseekDark.Hover, deepseekDark.Pressed
	}
	if value := s.Overrides["--dsw-alias-interactive-bg-hover"]; value != "" {
		hover = value
	}
	if value := s.Overrides["--dsw-alias-interactive-bg-active"]; value != "" {
		active = value
	}
	c.TriggerBG, c.TriggerHover = c.CodeBlock, hover
	if dark {
		c.TriggerBG, c.TriggerHover = hover, active
	}
	if value := s.Overrides["--dsw-alias-turn-trigger-bg"]; value != "" {
		c.TriggerBG = value
	}
	if value := s.Overrides["--dsw-alias-turn-trigger-bg-hover"]; value != "" {
		c.TriggerHover = value
	}
	return c
}
