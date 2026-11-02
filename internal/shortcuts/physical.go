package shortcuts

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"whaleisle/internal/shortcutprefs"
)

func PhysicalCode(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	upper := strings.ToUpper(key)
	if code := physicalCodes[upper]; code != "" {
		return code
	}
	if len(upper) == 1 && upper[0] >= 'A' && upper[0] <= 'Z' {
		return "Key" + upper
	}
	if len(upper) == 1 && upper[0] >= '0' && upper[0] <= '9' {
		return "Digit" + upper
	}
	if len(upper) > 1 && upper[0] == 'F' {
		if n, err := strconv.Atoi(upper[1:]); err == nil && n >= 1 && n <= 24 {
			return upper
		}
	}
	for _, code := range []string{"Enter", "Escape", "Space", "Tab", "Backspace", "Delete", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"} {
		if strings.EqualFold(key, code) {
			return code
		}
	}
	if strings.HasPrefix(key, "Key") && len(key) == 4 && key[3] >= 'A' && key[3] <= 'Z' {
		return key
	}
	if strings.HasPrefix(key, "Digit") && len(key) == 6 && key[5] >= '0' && key[5] <= '9' {
		return key
	}
	for _, code := range physicalCodes {
		if strings.EqualFold(key, code) {
			return code
		}
	}
	return ""
}
func PhysicalKey(code string) string {
	for key, value := range physicalCodes {
		if value == code {
			return key
		}
	}
	if strings.HasPrefix(code, "Key") {
		return strings.TrimPrefix(code, "Key")
	}
	if strings.HasPrefix(code, "Digit") {
		return strings.TrimPrefix(code, "Digit")
	}
	return code
}
func parseDesktopChord(raw string) (Chord, error) {
	if strings.TrimSpace(raw) == "" {
		return Chord{}, nil
	}
	parts := strings.Split(raw, "+")
	mods := map[string]bool{}
	for _, part := range parts[:len(parts)-1] {
		m := ""
		switch strings.ToUpper(strings.TrimSpace(part)) {
		case "MOD", "PRIMARY", "CTRLORCMD", "COMMANDORCONTROL":
			m = "Mod"
		case "CTRL", "CONTROL":
			m = "Control"
		case "CMD", "COMMAND", "META", "SUPER":
			m = "Meta"
		case "ALT", "OPTION":
			m = "Alt"
		case "SHIFT":
			m = "Shift"
		}
		if m == "" || mods[m] {
			return Chord{}, fmt.Errorf("Invalid or duplicate physical modifier")
		}
		mods[m] = true
	}
	keys := strings.Split(parts[len(parts)-1], "&")
	if len(keys) > 2 {
		return Chord{}, fmt.Errorf("Hold at most two distinct physical keys")
	}
	codes := []string{}
	for _, key := range keys {
		code := PhysicalCode(key)
		if code == "" {
			return Chord{}, fmt.Errorf("Unsupported physical key")
		}
		codes = append(codes, code)
	}
	if len(codes) == 2 && codes[0] == codes[1] {
		return Chord{}, fmt.Errorf("Hold two distinct physical keys")
	}
	sort.Strings(codes)
	c := Chord{Key: PhysicalKey(codes[0]), Modifiers: []string{}}
	c.Shift, c.Alt = mods["Shift"], mods["Alt"]
	if len(codes) == 2 {
		c.SecondKey = PhysicalKey(codes[1])
	}
	for _, m := range []string{"Mod", "Control", "Alt", "Shift", "Meta"} {
		if mods[m] {
			c.Modifiers = append(c.Modifiers, m)
		}
	}
	return c, nil
}
func (c Chord) PhysicalBinding() *shortcutprefs.Binding {
	if c.Key == "" {
		return nil
	}
	b := &shortcutprefs.Binding{Code: PhysicalCode(c.Key), SecondCode: PhysicalCode(c.SecondKey), Modifiers: []string{}}
	mods := c.Modifiers
	if mods == nil {
		mods = []string{"Mod"}
		if c.Shift {
			mods = append(mods, "Shift")
		}
		if c.Alt {
			mods = append(mods, "Alt")
		}
	}
	for _, m := range mods {
		switch m {
		case "Mod":
			b.Modifiers = append(b.Modifiers, "primary")
		case "Control":
			b.Modifiers = append(b.Modifiers, "control")
		case "Alt":
			b.Modifiers = append(b.Modifiers, "alt")
		case "Shift":
			b.Modifiers = append(b.Modifiers, "shift")
		case "Meta":
			b.Modifiers = append(b.Modifiers, "meta")
		}
	}
	return b
}
func overlappingChords(left, right Chord) bool {
	a, b := left.PhysicalBinding(), right.PhysicalBinding()
	if a == nil || b == nil {
		return false
	}
	mods := func(values []string) string {
		set := map[string]bool{}
		for _, m := range values {
			if m == "primary" {
				m = "control"
				if runtime.GOOS == "darwin" {
					m = "meta"
				}
			}
			set[m] = true
		}
		keys := []string{}
		for m := range set {
			keys = append(keys, m)
		}
		sort.Strings(keys)
		return strings.Join(keys, "+")
	}
	if mods(a.Modifiers) != mods(b.Modifiers) {
		return false
	}
	if a.SecondCode != "" && b.SecondCode != "" {
		return a.Code == b.Code && a.SecondCode == b.SecondCode
	}
	return a.Code == b.Code || a.SecondCode != "" && a.SecondCode == b.Code || b.SecondCode != "" && a.Code == b.SecondCode
}
