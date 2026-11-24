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

// TrimSpace trims ECMAScript white space and line terminators (String.prototype.trim).
func TrimSpace(s string) string { return trimJSSpace(s) }

// UTF16Len returns the JavaScript length of s.
func UTF16Len(s string) int { return utf16Len(s) }

// UTF16Slice returns s.slice(start, end) in UTF-16 units (clamped indices).
func UTF16Slice(s string, start, end int) string {
	u := utf16Units(s)
	if start < 0 {
		start = 0
	}
	if end > len(u) {
		end = len(u)
	}
	if start >= end {
		return ""
	}
	return utf16String(u[start:end])
}

// Member reads an own property of a value like the language's member
// access (object keys, array index/length, string index/length).
func Member(v Value, key string) Value { return memberOf(v, key) }

// JSRound implements Math.round.
func JSRound(x float64) float64 { return jsRound(x) }
