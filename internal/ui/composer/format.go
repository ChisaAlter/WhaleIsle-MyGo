package composer

import (
	"strconv"
	"strings"
)

// compactCount renders a token count the way the status line does: 812, 12.3K,
// 77.7M. Values at or above 100 of a unit drop the decimal, matching the
// reference's compact formatter.
func compactCount(value float64) string {
	if value < 0 {
		value = 0
	}
	if value < 1000 {
		return strconv.FormatFloat(value, 'f', 0, 64)
	}
	scaled := func(f float64) string {
		if f >= 100 {
			return strconv.FormatFloat(f, 'f', 0, 64)
		}
		return strings.TrimSuffix(strconv.FormatFloat(f, 'f', 1, 64), ".0")
	}
	switch {
	case value < 1e6:
		return scaled(value/1e3) + "K"
	case value < 1e9:
		return scaled(value/1e6) + "M"
	default:
		return scaled(value/1e9) + "B"
	}
}

// compactPercent renders a percentage with one decimal, dropping a trailing .0.
func compactPercent(value float64) string {
	text := strconv.FormatFloat(value, 'f', 1, 64)
	return strings.TrimSuffix(text, ".0") + "%"
}
