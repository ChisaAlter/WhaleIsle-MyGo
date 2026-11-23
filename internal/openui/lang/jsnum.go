package lang

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

// NumberToString implements ECMAScript Number::toString(x) (radix 10).
func NumberToString(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case x == 0:
		return "0"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	case x < 0:
		return "-" + NumberToString(-x)
	}
	// Shortest round-trip digits.
	e := strconv.FormatFloat(x, 'e', -1, 64) // d.ddddde±XX
	mant, expStr, _ := strings.Cut(e, "e")
	digits := strings.Replace(mant, ".", "", 1)
	exp, _ := strconv.Atoi(expStr)
	k := len(digits)
	n := exp + 1
	switch {
	case k <= n && n <= 21:
		return digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return "0." + strings.Repeat("0", -n) + digits
	}
	ee := n - 1
	sign := "+"
	if ee < 0 {
		sign = "-"
		ee = -ee
	}
	if k == 1 {
		return digits + "e" + sign + strconv.Itoa(ee)
	}
	return digits[:1] + "." + digits[1:] + "e" + sign + strconv.Itoa(ee)
}

func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xA0, 0xFEFF, 0x2028, 0x2029:
		return true
	}
	return unicode.Is(unicode.Zs, r)
}

func trimJSSpace(s string) string { return strings.TrimFunc(s, isJSSpace) }

// StringToNumber implements ECMAScript StringToNumber (Number("...")).
func StringToNumber(s string) float64 {
	s = trimJSSpace(s)
	if s == "" {
		return 0
	}
	if len(s) > 2 && s[0] == '0' {
		base := 0
		switch s[1] {
		case 'x', 'X':
			base = 16
		case 'o', 'O':
			base = 8
		case 'b', 'B':
			base = 2
		}
		if base != 0 {
			var f float64
			for _, c := range s[2:] {
				d := digitVal(c)
				if d < 0 || d >= base {
					return math.NaN()
				}
				f = f*float64(base) + float64(d)
			}
			return f
		}
	}
	body := s
	neg := false
	if body[0] == '+' || body[0] == '-' {
		neg = body[0] == '-'
		body = body[1:]
	}
	if body == "Infinity" {
		if neg {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	if n := decimalPrefix(body); n == 0 || n != len(body) {
		return math.NaN()
	}
	f, err := strconv.ParseFloat(body, 64)
	if err != nil {
		if ne, ok := err.(*strconv.NumError); !ok || ne.Err != strconv.ErrRange {
			return math.NaN()
		}
	}
	if neg {
		f = -f
	}
	return f
}

func digitVal(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// decimalPrefix returns the length of the longest StrUnsignedDecimalLiteral
// prefix of s (without Infinity), or 0.
func decimalPrefix(s string) int {
	i := 0
	intDigits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		intDigits++
	}
	fracDigits := 0
	if i < len(s) && s[i] == '.' {
		j := i + 1
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
			fracDigits++
		}
		if intDigits > 0 || fracDigits > 0 {
			i = j
		}
	}
	if intDigits == 0 && fracDigits == 0 {
		return 0
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		k := j
		for k < len(s) && s[k] >= '0' && s[k] <= '9' {
			k++
		}
		if k > j {
			i = k
		}
	}
	return i
}

// ParseFloat implements ECMAScript parseFloat: the longest numeric prefix
// after leading whitespace, or NaN.
func ParseFloat(s string) float64 {
	s = strings.TrimLeftFunc(s, isJSSpace)
	neg := false
	body := s
	if body != "" && (body[0] == '+' || body[0] == '-') {
		neg = body[0] == '-'
		body = body[1:]
	}
	if strings.HasPrefix(body, "Infinity") {
		if neg {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	n := decimalPrefix(body)
	if n == 0 {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(body[:n], 64)
	if neg {
		f = -f
	}
	return f
}

// jsRound implements Math.round (halves round toward +Infinity).
func jsRound(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	if x == 0 {
		return x
	}
	if x > 0 && x < 0.5 {
		return 0
	}
	if x < 0 && x >= -0.5 {
		return math.Copysign(0, -1)
	}
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	return r
}

// jsMod implements the JavaScript % operator.
func jsMod(a, b float64) float64 { return math.Mod(a, b) }

// jsTrunc implements Math.trunc.
func jsTrunc(x float64) float64 { return math.Trunc(x) }
