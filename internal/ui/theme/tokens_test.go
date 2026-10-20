package theme

import (
	"strconv"
	"strings"
	"testing"
)

// TestSampledTokensMatchTheHarness samples the --dsw-alias-* / --dsw-specific-*
// tokens the shell draws with and compares them with the harness's own values
// (runtime/vendor/deepseek-harness/packages/client/ui-theme/src/... and
// reference/electron/src/shared/dsh-webui-tokens.css). The notation may differ
// (#f9fafb and rgb(249, 250, 251) are the same color); the colors may not.
//
// Known gaps this wave does not close, because they need a design decision or
// wiring outside the theme package:
//   - there is no --dsw-alias-label-secondary: secondary text uses label-tertiary;
//   - no --dsw-alias-scrollbar-bg-l2 / -hover-l2 are carried;
//   - Field reads --dsw-alias-bg-module-platform in the light half and
//     --dsw-alias-bg-layer-1 in the dark one (module-platform is lighter there).
func TestSampledTokensMatchTheHarness(t *testing.T) {
	cases := []struct {
		token     string
		light     string
		dark      string
		wantLight string
		wantDark  string
	}{
		{"--dsw-alias-bg-base", deepseekLight.BG, deepseekDark.BG, "rgb(255, 255, 255)", "rgb(21, 21, 23)"},
		{"--dsw-alias-label-primary", deepseekLight.FG, deepseekDark.FG, "rgb(15, 17, 21)", "rgb(249, 250, 251)"},
		{"--dsw-alias-label-tertiary", deepseekLight.Muted, deepseekDark.Muted, "rgb(129, 133, 140)", "rgb(173, 178, 184)"},
		{"--dsw-alias-label-caption", deepseekLight.Caption, deepseekDark.Caption, "rgb(173, 178, 184)", "rgb(129, 133, 140)"},
		{"--dsw-alias-label-primary-foreground", deepseekLight.ButtonFG, deepseekDark.ButtonFG, "rgb(255, 255, 255)", "rgb(15, 17, 21)"},
		{"--dsw-alias-border-l2", deepseekLight.Border, deepseekDark.Border, "rgba(0, 0, 0, 0.1)", "rgba(255, 255, 255, 0.12)"},
		{"--dsw-alias-interactive-bg-hover", deepseekLight.Hover, deepseekDark.Hover, "rgba(38, 49, 72, 0.06)", "rgba(255, 255, 255, 0.08)"},
		{"--dsw-alias-interactive-bg-active", deepseekLight.Pressed, deepseekDark.Pressed, "rgba(38, 49, 72, 0.1)", "rgba(255, 255, 255, 0.14)"},
		// The dark accent is design-platform.css's cascaded deepseek-400; the
		// app-level token file's rgb(103, 158, 254) loses the cascade (see the
		// comment on deepseekDark.Accent).
		{"--dsw-alias-button-info-fill", deepseekLight.ButtonInfoFill, deepseekDark.ButtonInfoFill, "rgb(65, 118, 230)", "rgb(122, 170, 255)"},
		{"--dsw-alias-state-business-primary", deepseekLight.Accent, deepseekDark.Accent, "rgb(65, 118, 230)", "rgb(122, 170, 255)"},
		{"--dsw-specific-sidebar-fill", deepseekLight.Sidebar, deepseekDark.Sidebar, "rgb(249, 250, 251)", "rgb(27, 27, 28)"},
		{"--dsw-specific-bubble", deepseekLight.Bubble, deepseekDark.Bubble, "rgb(237, 243, 254)", "rgb(44, 44, 46)"},
		{"--dsw-specific-selector", deepseekLight.SelectorFill, deepseekDark.SelectorFill, "rgb(245, 246, 247)", "rgb(53, 54, 56)"},
	}
	for _, c := range cases {
		if !sameTokenColor(c.light, c.wantLight) {
			t.Errorf("%s light = %q, want %q", c.token, c.light, c.wantLight)
		}
		if !sameTokenColor(c.dark, c.wantDark) {
			t.Errorf("%s dark = %q, want %q", c.token, c.dark, c.wantDark)
		}
	}
	// --dsw-specific-input-major is color-mix(<neutral> 80%, transparent): the
	// color is the neutral, the alpha is the glass slider's 80%.
	if got := tokenAlpha(deepseekLight.InputMajor); got != 204 {
		t.Errorf("light input-major alpha = %d, want 204 (80%%)", got)
	}
	if got := tokenAlpha(deepseekDark.InputMajor); got != 204 {
		t.Errorf("dark input-major alpha = %d, want 204 (80%%)", got)
	}
}

// tokenAlpha returns a CSS color's alpha as 0..255, or -1 when it has none.
func tokenAlpha(s string) int {
	c, ok := parseTokenColor(s)
	if !ok {
		return -1
	}
	return c[3]
}

// sameTokenColor compares two CSS colors by what they paint, not how the sheet
// spells them.
func sameTokenColor(a, b string) bool {
	ca, oka := parseTokenColor(a)
	cb, okb := parseTokenColor(b)
	if !oka || !okb {
		return strings.EqualFold(normalizeToken(a), normalizeToken(b))
	}
	return ca == cb
}

func normalizeToken(s string) string {
	return strings.Join(strings.Fields(s), "")
}

// parseTokenColor reads #rgb, #rrggbb, rgb() and rgba() into R, G, B, A bytes.
func parseTokenColor(s string) ([4]int, bool) {
	s = strings.ToLower(normalizeToken(s))
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		v, err := strconv.ParseUint(s[1:], 16, 32)
		if err != nil {
			return [4]int{}, false
		}
		return [4]int{int(v>>16) & 0xff, int(v>>8) & 0xff, int(v) & 0xff, 255}, true
	}
	body, ok := strings.CutPrefix(s, "rgb(")
	if !ok {
		body, ok = strings.CutPrefix(s, "rgba(")
	}
	if !ok || !strings.HasSuffix(body, ")") {
		return [4]int{}, false
	}
	parts := strings.Split(strings.TrimSuffix(body, ")"), ",")
	if len(parts) != 3 && len(parts) != 4 {
		return [4]int{}, false
	}
	out := [4]int{0, 0, 0, 255}
	for i := 0; i < 3; i++ {
		v, err := strconv.Atoi(parts[i])
		if err != nil {
			return [4]int{}, false
		}
		out[i] = v
	}
	if len(parts) == 4 {
		f, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			return [4]int{}, false
		}
		out[3] = int(f*255 + .5)
	}
	return out, true
}
