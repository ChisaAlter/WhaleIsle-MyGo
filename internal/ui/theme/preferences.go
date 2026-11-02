package theme

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// FontPreference uses MyGo's native font-family fallback list and DIP size.
type FontPreference struct {
	Family string  `json:"family"`
	Size   float32 `json:"size"`
}

// DefaultTextFamily mirrors ui-theme/src/styles/base.css. MyGo resolves the
// comma-separated families in order, including CJK fallbacks after Segoe UI.
const DefaultTextFamily = "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif"

// base.css deliberately omits a generic monospace tail from code: Windows
// CJK text should reach the explicit PingFang/YaHei fallback, not SimSun.
const DefaultCodeFamily = "'SF Mono', 'JetBrains Mono', 'Fira Code', Consolas, 'Liberation Mono', Menlo, Courier, 'PingFang SC', 'Microsoft YaHei'"

type Seed struct {
	Background string            `json:"background"`
	Foreground string            `json:"foreground"`
	Accent     string            `json:"accent"`
	Contrast   *float64          `json:"contrast,omitempty"`
	Overrides  map[string]string `json:"overrides,omitempty"`
}
type Family struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Light Seed   `json:"light"`
	Dark  Seed   `json:"dark"`
	Skin  string `json:"skin,omitempty"`
}
type Preferences struct {
	Light     string         `json:"light"`
	Dark      string         `json:"dark"`
	Custom    []Family       `json:"custom"`
	Interface FontPreference `json:"interface"`
	Chat      FontPreference `json:"chat"`
	Code      FontPreference `json:"code"`
	Composer  FontPreference `json:"composer"`
	Terminal  FontPreference `json:"terminal"`
}

func Defaults() Preferences {
	return Preferences{Light: "deepseek", Dark: "deepseek", Interface: FontPreference{Family: DefaultTextFamily, Size: 16}, Chat: FontPreference{Family: DefaultTextFamily, Size: 14}, Code: FontPreference{Family: DefaultCodeFamily, Size: 13}, Composer: FontPreference{Family: DefaultTextFamily, Size: 14}, Terminal: FontPreference{Family: "monospace", Size: 13}}
}
func (p Preferences) Clone() Preferences {
	p.Custom = append([]Family(nil), p.Custom...)
	for i := range p.Custom {
		p.Custom[i].Light = p.Custom[i].Light.Clone()
		p.Custom[i].Dark = p.Custom[i].Dark.Clone()
	}
	return p
}

func (s Seed) Clone() Seed {
	if s.Contrast != nil {
		value := *s.Contrast
		s.Contrast = &value
	}
	if s.Overrides != nil {
		values := map[string]string{}
		for k, v := range s.Overrides {
			values[k] = v
		}
		s.Overrides = values
	}
	return s
}

func (s Seed) ContrastValue() float64 {
	if s.Contrast == nil {
		return 46
	}
	return *s.Contrast
}
func (p Preferences) Normalized() Preferences {
	d := Defaults()
	if p.Light == "" {
		p.Light = d.Light
	}
	if p.Dark == "" {
		p.Dark = d.Dark
	}
	for _, pair := range []struct {
		p *FontPreference
		d FontPreference
	}{{&p.Interface, d.Interface}, {&p.Chat, d.Chat}, {&p.Code, d.Code}, {&p.Composer, d.Composer}, {&p.Terminal, d.Terminal}} {
		if pair.p.Size <= 0 {
			pair.p.Size = pair.d.Size
		}
	}
	return p
}
func (p Preferences) Validate() error {
	for _, f := range []FontPreference{p.Interface, p.Chat, p.Code, p.Composer, p.Terminal} {
		if f.Size < 8 || f.Size > 48 {
			return fmt.Errorf("字号需在 8–48 / Font size must be 8–48")
		}
	}
	seen := map[string]bool{}
	for _, f := range Library() {
		seen[f.ID] = true
	}
	for _, f := range p.Custom {
		if strings.TrimSpace(f.ID) == "" || strings.TrimSpace(f.Name) == "" || seen[f.ID] {
			return fmt.Errorf("自定义主题名称和唯一 ID 必填 / Unique theme ID and name required")
		}
		seen[f.ID] = true
		if f.Skin != "" && f.Skin != SkinDefault && f.Skin != SkinCodex && f.Skin != SkinClaude {
			return fmt.Errorf("Unknown theme skin %q", f.Skin)
		}
		for _, s := range []Seed{f.Light, f.Dark} {
			if s.ContrastValue() < 0 || s.ContrastValue() > 100 {
				return fmt.Errorf("Contrast must be between 0 and 100")
			}
			for _, c := range []string{s.Background, s.Foreground, s.Accent} {
				if len(c) != 7 || c[0] != '#' {
					return fmt.Errorf("颜色格式为 #RRGGBB / Use #RRGGBB")
				}
				if _, err := strconv.ParseUint(c[1:], 16, 24); err != nil {
					return err
				}
			}
		}
	}
	if !seen[p.Light] || !seen[p.Dark] {
		return fmt.Errorf("请选择已有主题 / Select an existing theme")
	}
	return nil
}

// library is the built-in palette. It is immutable, so it is built once: the
// render path asked for it on every frame (a Family slice and its Seeds),
// which showed up as one of the largest per-frame allocators.
var library = buildLibrary()

func Library() []Family { return library }

func buildLibrary() []Family {
	seed := func(bg, fg, accent string, contrast float64) Seed {
		return Seed{Background: bg, Foreground: fg, Accent: accent, Contrast: &contrast}
	}
	return []Family{
		{ID: "deepseek", Name: "DeepSeek", Light: seed("#ffffff", "#0f1115", "#4176e6", 46), Dark: seed("#151517", "#f5f5f5", "#6ea8ff", 41)},
		codexBuiltinFamily(),
		claudeBuiltinFamily(),
		{ID: "midnight", Name: "午夜 / Midnight", Light: seed("#f3f6fb", "#1a1f2b", "#3b6fd4", 44), Dark: seed("#0b0d12", "#e8eef9", "#6ea8ff", 48)},
		{ID: "celadon", Name: "青瓷 / Celadon", Light: seed("#f3faf7", "#10211c", "#0f766e", 44), Dark: seed("#071411", "#e7f6f1", "#3dd6b5", 50)},
		{ID: "violet", Name: "暮紫 / Violet", Light: seed("#f7f3fc", "#1c1524", "#7c3aed", 46), Dark: seed("#120e18", "#f3eefc", "#c4a1ff", 50)},
		{ID: "amber", Name: "琥珀 / Amber", Light: seed("#fbf6ee", "#1c1915", "#b45309", 48), Dark: seed("#14100b", "#f6efe4", "#e2b15c", 52)},
		{ID: "paper", Name: "宣纸 / Paper", Light: seed("#f3efe6", "#1c1915", "#0f766e", 50), Dark: seed("#1a1712", "#f6efe4", "#5eead4", 48)},
		{ID: "contrast", Name: "对比 / Contrast", Light: seed("#ffffff", "#050505", "#111111", 68), Dark: seed("#050505", "#f5f5f5", "#ffffff", 64)},
	}
}
func (p Preferences) Families() []Family { return append(Library(), p.Custom...) }
func (p Preferences) Family(id string) Family {
	for _, f := range p.Families() {
		if f.ID == id {
			return f
		}
	}
	return Library()[0]
}

var preferenceState = struct {
	sync.RWMutex
	value    Preferences
	rendered Preferences
	selected selectedFamilies
}{value: Defaults(), rendered: renderPreferences(Defaults()), selected: selectFamilies(Defaults())}

// selectedFamilies holds the palettes the chosen families derive. Resolve runs
// for themed elements throughout every frame (each sidebar row among them), so
// they are derived once per SetPreferences rather than on every call.
type selectedFamilies struct {
	light, dark       Colors
	hasLight, hasDark bool
}

func selectFamilies(p Preferences) selectedFamilies {
	var out selectedFamilies
	l, d := p.Family(p.Light), p.Family(p.Dark)
	if l.ID != "deepseek" {
		out.light, out.hasLight = palette(l.Light, false), true
	}
	if d.ID != "deepseek" {
		out.dark, out.hasDark = palette(d.Dark, true), true
	}
	return out
}

func preferencesEqual(a, b Preferences) bool {
	if a.Light != b.Light || a.Dark != b.Dark ||
		a.Interface != b.Interface || a.Chat != b.Chat ||
		a.Code != b.Code || a.Composer != b.Composer || a.Terminal != b.Terminal ||
		len(a.Custom) != len(b.Custom) {
		return false
	}
	for i := range a.Custom {
		x, y := a.Custom[i], b.Custom[i]
		if x.ID != y.ID || x.Name != y.Name || x.Skin != y.Skin || !seedEqual(x.Light, y.Light) || !seedEqual(x.Dark, y.Dark) {
			return false
		}
	}
	return true
}

func seedEqual(a, b Seed) bool {
	if a.Background != b.Background || a.Foreground != b.Foreground || a.Accent != b.Accent ||
		(a.Contrast == nil) != (b.Contrast == nil) || len(a.Overrides) != len(b.Overrides) {
		return false
	}
	if a.Contrast != nil && *a.Contrast != *b.Contrast {
		return false
	}
	for k, v := range a.Overrides {
		if bV, ok := b.Overrides[k]; !ok || bV != v {
			return false
		}
	}
	return true
}

func SetPreferences(p Preferences) {
	p = p.Normalized()
	preferenceState.RLock()
	same := preferencesEqual(preferenceState.value, p)
	preferenceState.RUnlock()
	if same {
		return
	}
	p = p.Clone()
	rendered := renderPreferences(p)
	selected := selectFamilies(p)
	preferenceState.Lock()
	preferenceState.value = p
	preferenceState.rendered = rendered
	preferenceState.selected = selected
	preferenceState.Unlock()
}
func CurrentPreferences() Preferences {
	preferenceState.RLock()
	defer preferenceState.RUnlock()
	return preferenceState.value.Clone()
}
func InterfaceSize(base float32) float32 { return base * CurrentPreferences().Interface.Size / 16 }
func mix(a, b string, t float64) string {
	x, _ := strconv.ParseUint(strings.TrimPrefix(a, "#"), 16, 24)
	y, _ := strconv.ParseUint(strings.TrimPrefix(b, "#"), 16, 24)
	var out uint64
	for _, shift := range []uint{16, 8, 0} {
		u := float64((x >> shift) & 255)
		v := float64((y >> shift) & 255)
		out |= uint64(u+(v-u)*t+.5) << shift
	}
	return fmt.Sprintf("#%06x", out)
}
func palette(s Seed, dark bool) Colors {
	c := deepseekLight
	if dark {
		c = deepseekDark
	}
	c.BG = s.Background
	c.FG = s.Foreground
	c.Accent = s.Accent
	c.Muted = mix(c.FG, c.BG, .42)
	c.Caption = mix(c.FG, c.BG, .6)
	c.Field = mix(c.BG, c.FG, .06)
	c.Sidebar = mix(c.BG, c.FG, .025)
	c.Titlebar = c.Sidebar
	c.Border = mix(c.BG, c.FG, .12)
	c.Hover = mix(c.BG, c.FG, .08)
	c.Pressed = deepseekLight.Pressed
	if dark {
		c.Pressed = deepseekDark.Pressed
	}
	c.Bubble = mix(c.BG, c.Accent, .12)
	c.InputMajor = c.Field
	c.SelectorFill = c.Field
	c.ButtonInfoFill = c.Accent
	// Custom families use the original derive.ts info-hover tint. The original
	// info button has no separate active rule, so native pressed uses this too.
	tint := "#000000"
	if dark {
		tint = "#ffffff"
	}
	c.ButtonInfoHover = mix(c.Accent, tint, .18)
	c.ButtonFG = c.BG
	return derivePalette(c, s)
}
func selectedPalettes(t Theme) Theme {
	preferenceState.RLock()
	s := preferenceState.selected
	preferenceState.RUnlock()
	if s.hasLight {
		t.Light = s.light
	}
	if s.hasDark {
		t.Dark = s.dark
	}
	return t
}
