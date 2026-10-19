package platform

import "strconv"

// ProductPhysicalCode is the inverse of the shared native product VK mapping.
func ProductPhysicalCode(vk uint32) string {
	if vk >= 'A' && vk <= 'Z' {
		return "Key" + string(rune(vk))
	}
	if vk >= '0' && vk <= '9' {
		return "Digit" + string(rune(vk))
	}
	if vk >= 112 && vk <= 135 {
		return "F" + strconv.Itoa(int(vk-111))
	}
	return map[uint32]string{13: "Enter", 27: "Escape", 32: "Space", 9: "Tab", 8: "Backspace", 46: "Delete", 37: "ArrowLeft", 38: "ArrowUp", 39: "ArrowRight", 40: "ArrowDown", 188: "Comma", 190: "Period", 191: "Slash", 186: "Semicolon", 222: "Quote", 219: "BracketLeft", 221: "BracketRight", 220: "Backslash", 192: "Backquote", 189: "Minus", 187: "Equal"}[vk]
}

// ProductMacVirtualKey translates NSEvent's physical keyCode to the existing
// product registry's VK-shaped code. Unknown/IME/keypad keys deliberately pass
// through. The result describes position, not the text produced by a layout.
func ProductMacVirtualKey(code uint16) uint32 {
	return macProductKeys[code]
}

var macProductKeys = map[uint16]uint32{
	0: 'A', 1: 'S', 2: 'D', 3: 'F', 4: 'H', 5: 'G', 6: 'Z', 7: 'X', 8: 'C', 9: 'V', 11: 'B', 12: 'Q', 13: 'W', 14: 'E', 15: 'R', 16: 'Y', 17: 'T',
	18: '1', 19: '2', 20: '3', 21: '4', 22: '6', 23: '5', 24: 0xbb, 25: '9', 26: '7', 27: 0xbd, 28: '8', 29: '0', 30: 0xdd, 31: 'O', 32: 'U', 33: 0xdb, 34: 'I', 35: 'P',
	36: 0x0d, 37: 'L', 38: 'J', 39: 0xde, 40: 'K', 41: 0xba, 42: 0xdc, 43: 0xbc, 44: 0xbf, 45: 'N', 46: 'M', 47: 0xbe, 48: 9, 49: 0x20, 50: 0xc0, 51: 8, 53: 0x1b,
	122: 0x70, 120: 0x71, 99: 0x72, 118: 0x73, 96: 0x74, 97: 0x75, 98: 0x76, 100: 0x77, 101: 0x78, 109: 0x79, 103: 0x7a, 111: 0x7b,
	105: 0x7c, 107: 0x7d, 113: 0x7e, 106: 0x7f, 64: 0x80, 79: 0x81, 80: 0x82, 90: 0x83,
	115: 0x24, 116: 0x21, 117: 0x2e, 119: 0x23, 121: 0x22, 123: 0x25, 124: 0x27, 125: 0x28, 126: 0x26,
}

// ProductGDKVirtualKey follows the same known hardware-keycode positions as
// WebKitGTK's DOM code mapping, rather than interpreting locale-dependent
// GdkEventKey.keyval as KeyboardEvent.code. Unknown positions pass through.
func ProductGDKVirtualKey(code uint16) uint32 {
	if code >= 10 && code <= 18 {
		return uint32('1' + code - 10)
	}
	if code >= 24 && code <= 33 {
		return uint32("QWERTYUIOP"[code-24])
	}
	if code >= 38 && code <= 46 {
		return uint32("ASDFGHJKL"[code-38])
	}
	if code >= 52 && code <= 58 {
		return uint32("ZXCVBNM"[code-52])
	}
	if code >= 191 && code <= 202 {
		return uint32(0x7c + code - 191)
	}
	if code >= 67 && code <= 76 {
		return uint32(0x70 + code - 67)
	}
	return gdkProductKeys[code]
}

var gdkProductKeys = map[uint16]uint32{
	9: 0x1b, 19: '0', 20: 0xbd, 21: 0xbb, 22: 8, 23: 9, 34: 0xdb, 35: 0xdd, 36: 0x0d, 47: 0xba, 48: 0xde, 49: 0xc0, 51: 0xdc, 59: 0xbc, 60: 0xbe, 61: 0xbf, 65: 0x20,
	95: 0x7a, 96: 0x7b, 110: 0x24, 111: 0x26, 112: 0x21, 113: 0x25, 114: 0x27, 115: 0x23, 116: 0x28, 117: 0x22, 118: 0x2d, 119: 0x2e,
}
