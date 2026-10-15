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
type Seed struct {
	Background string `json:"background"`
	Foreground string `json:"foreground"`
	Accent     string `json:"accent"`
}
type Family struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Light Seed   `json:"light"`
	Dark  Seed   `json:"dark"`
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
	return Preferences{Light: "deepseek", Dark: "deepseek", Interface: FontPreference{Size: 14}, Chat: FontPreference{Size: 14}, Code: FontPreference{Family: "monospace", Size: 13}, Composer: FontPreference{Size: 14}, Terminal: FontPreference{Family: "monospace", Size: 13}}
}
func (p Preferences) Clone() Preferences { p.Custom = append([]Family(nil), p.Custom...); return p }
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
		if pair.p.Family == "" {
			pair.p.Family = pair.d.Family
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
		for _, s := range []Seed{f.Light, f.Dark} {
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
func Library() []Family {
	return []Family{
		{"deepseek", "DeepSeek", Seed{"#ffffff", "#0f1115", "#4176e6"}, Seed{"#151517", "#f5f5f5", "#679efe"}},
		{"midnight", "午夜 / Midnight", Seed{"#f3f6fb", "#1a1f2b", "#3b6fd4"}, Seed{"#0b0d12", "#e8eef9", "#6ea8ff"}},
		{"celadon", "青瓷 / Celadon", Seed{"#f3faf7", "#10211c", "#0f766e"}, Seed{"#071411", "#e7f6f1", "#3dd6b5"}},
		{"violet", "暮紫 / Violet", Seed{"#f7f3fc", "#1c1524", "#7c3aed"}, Seed{"#120e18", "#f3eefc", "#c4a1ff"}},
		{"amber", "琥珀 / Amber", Seed{"#fbf6ee", "#1c1915", "#b45309"}, Seed{"#14100b", "#f6efe4", "#e2b15c"}},
		{"paper", "宣纸 / Paper", Seed{"#f3efe6", "#1c1915", "#0f766e"}, Seed{"#1a1712", "#f6efe4", "#5eead4"}},
		{"contrast", "对比 / Contrast", Seed{"#ffffff", "#050505", "#111111"}, Seed{"#050505", "#f5f5f5", "#ffffff"}},
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
	value Preferences
}{value: Defaults()}

func SetPreferences(p Preferences) {
	preferenceState.Lock()
	preferenceState.value = p.Normalized().Clone()
	preferenceState.Unlock()
}
func CurrentPreferences() Preferences {
	preferenceState.RLock()
	defer preferenceState.RUnlock()
	return preferenceState.value.Clone()
}
func InterfaceSize(base float32) float32 { return base * CurrentPreferences().Interface.Size / 14 }
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
	c.Bubble = mix(c.BG, c.Accent, .12)
	c.InputMajor = c.Field
	c.SelectorFill = c.Field
	c.ButtonInfoFill = c.Accent
	c.ButtonFG = c.BG
	return c
}
func selectedPalettes(t Theme) Theme {
	p := CurrentPreferences()
	l, d := p.Family(p.Light), p.Family(p.Dark)
	if l.ID != "deepseek" {
		t.Light = palette(l.Light, false)
	}
	if d.ID != "deepseek" {
		t.Dark = palette(d.Dark, true)
	}
	return t
}
