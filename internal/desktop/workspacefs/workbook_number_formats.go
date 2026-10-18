package workspacefs

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

var workbookConditionPattern = regexp.MustCompile("^(<=|>=|<>|<|>|=)([+-]?(?:[0-9]+(?:\\.[0-9]*)?|\\.[0-9]+)(?:[Ee][+-]?[0-9]+)?)$")

type workbookFormatCondition struct {
	operator string
	value    float64
}

func (condition workbookFormatCondition) matches(value float64) bool {
	switch condition.operator {
	case "<":
		return value < condition.value
	case ">":
		return value > condition.value
	case "<=":
		return value <= condition.value
	case ">=":
		return value >= condition.value
	case "=":
		return value == condition.value
	case "<>":
		return value != condition.value
	}
	return false
}

// Conditions select a saved-value display section; they are not worksheet
// formulas and do not invoke the workbook calculation engine.
func workbookSectionCondition(section string) (string, workbookFormatCondition, bool) {
	var clean strings.Builder
	var condition workbookFormatCondition
	quoted, escaped := false, false
	for i := 0; i < len(section); i++ {
		ch := section[i]
		if escaped {
			clean.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			clean.WriteByte(ch)
			escaped = true
			continue
		}
		if ch == '"' {
			quoted = !quoted
		}
		if ch != '[' || quoted {
			clean.WriteByte(ch)
			continue
		}
		end := strings.IndexByte(section[i:], ']')
		if end < 0 {
			return "", condition, false
		}
		end += i
		block := section[i+1 : end]
		if block != "" && strings.ContainsRune("<>=", rune(block[0])) {
			parts := workbookConditionPattern.FindStringSubmatch(block)
			if parts == nil || condition.operator != "" {
				return "", condition, false
			}
			limit, err := strconv.ParseFloat(parts[2], 64)
			if err != nil {
				return "", condition, false
			}
			condition = workbookFormatCondition{parts[1], limit}
		} else {
			clean.WriteString(section[i : end+1])
		}
		i = end
	}
	return clean.String(), condition, true
}

func workbookSelectSection(value float64, sections []string) (string, float64, bool) {
	if len(sections) > 4 {
		return "", value, false
	}
	count := min(3, len(sections))
	// An unquoted text placeholder belongs to the text section, not to zero.
	if count < 4 && len(sections) <= 3 && workbookHasTextPlaceholder(sections[len(sections)-1]) {
		count--
	}
	if count == 0 {
		return "General", value, true
	}
	chosen := 0
	if value < 0 && count > 1 {
		chosen = 1
	} else if value == 0 && count > 2 {
		chosen = 2
	}
	var clean [3]string
	var conditions [3]workbookFormatCondition
	for i := 0; i < count; i++ {
		var ok bool
		clean[i], conditions[i], ok = workbookSectionCondition(sections[i])
		if !ok || i == 2 && conditions[i].operator != "" {
			return "", value, false
		}
	}
	if count == 1 {
		clean[1], clean[2] = clean[0], clean[0]
		conditions[1] = conditions[0]
	} else if count == 2 {
		clean[2] = clean[0]
	}
	if conditions[0].operator != "" || conditions[1].operator != "" {
		switch {
		case conditions[0].matches(value):
			chosen = 0
		case conditions[1].matches(value):
			chosen = 1
		case conditions[0].operator != "" && conditions[1].operator != "":
			chosen = 2
		default:
			chosen = 1
		}
	}
	if count > 1 && value < 0 {
		value = -value
	}
	return clean[chosen], value, true
}

func workbookHasTextPlaceholder(section string) bool {
	quoted, escaped := false, false
	for _, ch := range section {
		if escaped {
			escaped = false
		} else if ch == '\\' {
			escaped = true
		} else if ch == '"' {
			quoted = !quoted
		} else if ch == '@' && !quoted {
			return true
		}
	}
	return false
}

var workbookFractionPattern = regexp.MustCompile("^(?:([#0?]+)( +))?([#0?]+)( *)/( *)([0-9#?]+)$")

func workbookFraction(value float64, tokens []workbookFormatToken) (string, bool, bool) {
	slash, first := -1, -1
	for i, token := range tokens {
		if token.kind == "pattern" && token.text == "/" {
			if slash >= 0 {
				return "", true, false
			}
			slash = i
		}
		if first < 0 && token.kind == "pattern" && strings.ContainsAny(token.text, "0#?") {
			first = i
		}
	}
	if slash < 0 {
		return "", false, false
	}
	if first < 0 || first >= slash {
		return "", true, false
	}
	end := slash + 1
	for end < len(tokens) && tokens[end].kind == "literal" && tokens[end].text == " " {
		end++
	}
	for end < len(tokens) && (tokens[end].kind == "digit" || tokens[end].kind == "pattern" && strings.ContainsAny(tokens[end].text, "0#?")) {
		end++
	}
	var pattern strings.Builder
	for _, token := range tokens[first:end] {
		if token.kind != "pattern" && token.kind != "digit" && !(token.kind == "literal" && token.text == " ") {
			return "", true, false
		}
		pattern.WriteString(token.text)
	}
	parts := workbookFractionPattern.FindStringSubmatch(pattern.String())
	if parts == nil {
		return "", true, false
	}
	prefix, leftPercent, ok := workbookNumberLiteral(tokens[:first])
	if !ok {
		return "", true, false
	}
	suffix, rightPercent, ok := workbookNumberLiteral(tokens[end:])
	if !ok {
		return "", true, false
	}
	value *= math.Pow(100, float64(leftPercent+rightPercent))
	if math.IsInf(value, 0) {
		return "", true, false
	}
	negative := value < 0
	value = math.Abs(value)
	mixed := parts[1] != ""
	whole, fractional := 0.0, value
	if mixed {
		whole = math.Floor(value)
		fractional -= whole
	}
	fixed := strings.ContainsAny(parts[6], "123456789")
	var numerator, denominator int64
	if fixed {
		if strings.ContainsAny(parts[6], "#?") {
			return "", true, false
		}
		var err error
		denominator, err = strconv.ParseInt(parts[6], 10, 64)
		if err != nil || denominator <= 0 || fractional*float64(denominator) > float64(1<<53-1) {
			return "", true, false
		}
		numerator = int64(math.Round(fractional * float64(denominator)))
	} else {
		width := len(parts[6])
		if mixed {
			width = max(width, len(parts[3]))
		}
		// SSF's fraction display uses at most seven denominator digits.
		maximum := int64(math.Pow10(min(width, 7))) - 1
		if fractional > float64(1<<53-1)/float64(maximum) {
			return "", true, false
		}
		numerator, denominator = workbookFractionRatio(fractional, maximum)
	}
	if mixed && numerator >= denominator {
		whole += float64(numerator / denominator)
		numerator %= denominator
	}
	wholeText := ""
	if mixed {
		wholeText = workbookPlaceholderInteger(strconv.FormatFloat(whole, 'f', 0, 64), parts[1], whole == 0)
		if whole == 0 && numerator == 0 && !fixed && wholeText == "" {
			wholeText = "0"
		}
		wholeText += parts[2]
	}
	fractionText := ""
	if numerator == 0 && mixed {
		fractionText = strings.Repeat(" ", len(parts[3])+len(parts[4])+1+len(parts[5])+len(parts[6]))
	} else {
		numeratorText := workbookPlaceholderInteger(strconv.FormatInt(numerator, 10), parts[3], false)
		denominatorText := parts[6]
		if !fixed {
			denominatorText = workbookPlaceholderInteger(strconv.FormatInt(denominator, 10), parts[6], false)
		}
		fractionText = numeratorText + parts[4] + "/" + parts[5] + denominatorText
	}
	if negative {
		prefix = "-" + prefix
	}
	return prefix + wholeText + fractionText + suffix, true, true
}

// The convergents match the preview formatter's bounded-denominator fraction
// display. Fixed denominators above retain their exact requested denominator.
func workbookFractionRatio(value float64, maximum int64) (int64, int64) {
	previousN, previousD, currentN, currentD := int64(0), int64(1), int64(1), int64(0)
	remaining := value
	for {
		integer := math.Floor(remaining)
		nextD := integer*float64(currentD) + float64(previousD)
		if nextD > float64(maximum) {
			return currentN, currentD
		}
		nextN := int64(integer)*currentN + previousN
		denominator := int64(nextD)
		remainder := remaining - integer
		if remainder < 0.00000005 || denominator == maximum {
			return nextN, denominator
		}
		previousN, currentN = currentN, nextN
		previousD, currentD = currentD, denominator
		remaining = 1 / remainder
	}
}

func workbookPlaceholderInteger(digits, pattern string, blankZero bool) string {
	if blankZero && digits == "0" && !strings.Contains(pattern, "0") {
		digits = ""
	}
	missing := len(pattern) - len(digits)
	if missing <= 0 {
		return digits
	}
	var prefix strings.Builder
	for _, ch := range pattern[:missing] {
		switch ch {
		case '0':
			prefix.WriteByte('0')
		case '?':
			prefix.WriteByte(' ')
		}
	}
	return prefix.String() + digits
}

func workbookNumberLiteral(tokens []workbookFormatToken) (string, int, bool) {
	var out strings.Builder
	percentages := 0
	for _, token := range tokens {
		if token.kind == "pattern" && token.text == "%" {
			percentages++
		} else if token.kind != "literal" && token.kind != "digit" && token.text != "-" && token.text != "+" {
			return "", 0, false
		}
		out.WriteString(token.text)
	}
	return out.String(), percentages, true
}

func workbookScientific(value float64, integerPattern, fractionPattern, exponentSign, exponentPattern string) (string, bool) {
	if strings.Contains(integerPattern, ",") || len(integerPattern) > 15 {
		return "", false
	}
	exponent, mantissa := 0, value
	if value != 0 {
		scientific := strings.Split(strconv.FormatFloat(value, 'e', -1, 64), "e")
		coefficient, err := strconv.ParseFloat(scientific[0], 64)
		if err != nil {
			return "", false
		}
		power, err := strconv.Atoi(scientific[1])
		if err != nil {
			return "", false
		}
		if strings.Contains(integerPattern, "#") {
			period := len(integerPattern)
			exponent = int(math.Floor(float64(power)/float64(period))) * period
		} else {
			exponent = power - len(integerPattern) + 1
		}
		mantissa = coefficient * math.Pow10(power-exponent)
	}
	rounded, err := strconv.ParseFloat(strconv.FormatFloat(mantissa, 'f', len(fractionPattern), 64), 64)
	if err != nil {
		return "", false
	}
	if rounded >= math.Pow10(len(integerPattern)) {
		period := 1
		if strings.Contains(integerPattern, "#") {
			period = len(integerPattern)
		}
		rounded /= math.Pow10(period)
		exponent += period
	}
	pattern := integerPattern
	if fractionPattern != "" {
		pattern += "." + fractionPattern
	}
	tokens, ok := workbookFormatTokens(pattern)
	if !ok {
		return "", false
	}
	display, ok := workbookDecimal(rounded, tokens)
	if !ok {
		return "", false
	}
	sign := "+"
	if exponent < 0 {
		sign, exponent = "-", -exponent
	} else if exponentSign == "-" {
		sign = ""
	}
	exponentText := strconv.Itoa(exponent)
	if len(exponentText) < len(exponentPattern) {
		exponentText = strings.Repeat("0", len(exponentPattern)-len(exponentText)) + exponentText
	}
	return display + "E" + sign + exponentText, true
}
