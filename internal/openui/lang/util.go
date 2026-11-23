package lang

import (
	"sync"
	"unicode/utf16"
)

var collatorMu sync.Mutex

func utf16Units(s string) []uint16 { return utf16.Encode([]rune(s)) }

func utf16String(u []uint16) string { return string(utf16.Decode(u)) }

// utf16Len is the JavaScript string length.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}
