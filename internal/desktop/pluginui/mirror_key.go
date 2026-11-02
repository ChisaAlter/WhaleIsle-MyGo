package pluginui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

func mirrorKey(event ui.InputEvent) (string, string, int) {
	code := event.Code
	named := map[ui.Key]struct {
		key, code string
		vk        int
	}{
		ui.KeyBackspace: {"Backspace", "Backspace", 8}, ui.KeyTab: {"Tab", "Tab", 9}, ui.KeyEnter: {"Enter", "Enter", 13}, ui.KeyEscape: {"Escape", "Escape", 27}, ui.KeySpace: {" ", "Space", 32}, ui.KeyDelete: {"Delete", "Delete", 46}, ui.KeyInsert: {"Insert", "Insert", 45}, ui.KeyLeft: {"ArrowLeft", "ArrowLeft", 37}, ui.KeyUp: {"ArrowUp", "ArrowUp", 38}, ui.KeyRight: {"ArrowRight", "ArrowRight", 39}, ui.KeyDown: {"ArrowDown", "ArrowDown", 40}, ui.KeyHome: {"Home", "Home", 36}, ui.KeyEnd: {"End", "End", 35}, ui.KeyPageUp: {"PageUp", "PageUp", 33}, ui.KeyPageDown: {"PageDown", "PageDown", 34},
	}
	if value, ok := named[event.Key]; ok {
		return value.key, value.code, value.vk
	}
	if event.Key >= ui.KeyA && event.Key <= ui.KeyZ {
		letter := byte('a' + event.Key - ui.KeyA)
		key := string(letter)
		if event.Mods&ui.Shift != 0 {
			key = strings.ToUpper(key)
		}
		return key, "Key" + strings.ToUpper(string(letter)), int('A' + event.Key - ui.KeyA)
	}
	if event.Key >= ui.Key0 && event.Key <= ui.Key9 {
		digit := byte('0' + event.Key - ui.Key0)
		return string(digit), "Digit" + string(digit), int(digit)
	}
	if code != "" {
		return "Unidentified", code, 0
	}
	return "", "", 0
}
