package theme

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Color parses a CSS color token from a theme palette (#hex or rgb/rgba).
// Invalid input yields opaque black so a bad token cannot panic the shell.
func Color(s string) ui.Color {
	s = strings.TrimSpace(s)
	if s == "" {
		return ui.RGB(0, 0, 0)
	}
	if strings.HasPrefix(s, "#") {
		c, err := parseHexSafe(s)
		if err != nil {
			return ui.RGB(0, 0, 0)
		}
		return c
	}
	if c, ok := parseRGBFunc(s); ok {
		return c
	}
	return ui.RGB(0, 0, 0)
}

func parseHexSafe(s string) (ui.Color, error) {
	// ui.Hex panics; recover so theme tokens never crash the process.
	var c ui.Color
	var panicked error
	func() {
		defer func() {
			if v := recover(); v != nil {
				panicked = fmt.Errorf("%v", v)
			}
		}()
		c = ui.Hex(s)
	}()
	return c, panicked
}

func parseRGBFunc(s string) (ui.Color, bool) {
	lower := strings.ToLower(s)
	var body string
	switch {
	case strings.HasPrefix(lower, "rgba(") && strings.HasSuffix(lower, ")"):
		body = s[5 : len(s)-1]
	case strings.HasPrefix(lower, "rgb(") && strings.HasSuffix(lower, ")"):
		body = s[4 : len(s)-1]
	default:
		return ui.Color{}, false
	}
	parts := strings.Split(body, ",")
	if len(parts) != 3 && len(parts) != 4 {
		return ui.Color{}, false
	}
	r, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	g, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	b, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err1 != nil || err2 != nil || err3 != nil {
		return ui.Color{}, false
	}
	a := float32(1)
	if len(parts) == 4 {
		f, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 32)
		if err != nil {
			return ui.Color{}, false
		}
		a = float32(f)
	}
	return ui.RGBA(clampByte(r), clampByte(g), clampByte(b), a), true
}

func clampByte(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
