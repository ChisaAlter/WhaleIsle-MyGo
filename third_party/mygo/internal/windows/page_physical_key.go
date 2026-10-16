package windows

// WebView2 exposes Win32 scan codes in lParam. VK letters/OEM punctuation are
// layout dependent, whereas the accepted registry uses physical DOM codes.
// Unknown/IME/AltGraph inputs never fall back to a guessed physical key.
func productPhysicalVK(lparam uint32) uint32 {
	scan := (lparam >> 16) & 0xff
	if lparam&(1<<24) != 0 {
		switch scan {
		case 0x1c:
			return 13
		case 0x1d:
			return 17
		case 0x38:
			return 18
		case 0x4b:
			return 37
		case 0x48:
			return 38
		case 0x4d:
			return 39
		case 0x50:
			return 40
		case 0x53:
			return 46
		case 0x5b:
			return 91
		case 0x5c:
			return 92
		}
		return 0
	}
	if scan >= 0x02 && scan <= 0x0a {
		return uint32('1') + scan - 0x02
	}
	if scan == 0x0b {
		return '0'
	}
	rows := []struct {
		start uint32
		keys  string
	}{{0x10, "QWERTYUIOP"}, {0x1e, "ASDFGHJKL"}, {0x2c, "ZXCVBNM"}}
	for _, row := range rows {
		if scan >= row.start && scan < row.start+uint32(len(row.keys)) {
			return uint32(row.keys[scan-row.start])
		}
	}
	if scan >= 0x3b && scan <= 0x44 {
		return 112 + scan - 0x3b
	}
	if scan >= 0x64 && scan <= 0x6e {
		return 124 + scan - 0x64
	}
	return map[uint32]uint32{0x01: 27, 0x0c: 189, 0x0d: 187, 0x0e: 8, 0x0f: 9, 0x1a: 219, 0x1b: 221, 0x1c: 13, 0x1d: 17, 0x27: 186, 0x28: 222, 0x29: 192, 0x2a: 16, 0x2b: 220, 0x33: 188, 0x34: 190, 0x35: 191, 0x36: 16, 0x38: 18, 0x39: 32, 0x57: 122, 0x58: 123, 0x76: 135}[scan]
}
