package workspacefs

import (
	"archive/zip"
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Values remain the stored, copyable text. DisplayValue is derived only from
// the saved value/type/number format; this path never evaluates a formula.
func workbookCell(address, value, kind, format string, date1904 bool) XLSXCell {
	cell := XLSXCell{Address: address, Value: value, Type: kind, NumberFormat: format, DisplayValue: value, FormatSupported: true}
	switch kind {
	case "boolean":
		if value == "1" || strings.EqualFold(value, "true") {
			cell.DisplayValue = "true"
		} else if value == "0" || strings.EqualFold(value, "false") {
			cell.DisplayValue = "false"
		}
	case "date":
		date, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			date, err = time.Parse("2006-01-02", value)
		}
		if err != nil {
			cell.FormatSupported = false
			break
		}
		serial := float64(date.Unix())/86400 + float64(date.Nanosecond())/86400e9 + 25569
		if serial < 61 {
			serial--
		}
		display, _, ok := workbookNumber(serial, format, false)
		cell.FormatSupported = ok
		if ok {
			cell.DisplayValue = display
		}
	case "number":
		if value == "" {
			break
		}
		number, err := strconv.ParseFloat(value, 64)
		if err != nil {
			break
		}
		display, date, ok := workbookNumber(number, format, date1904)
		cell.FormatSupported = ok
		if ok {
			cell.DisplayValue = display
			if date {
				cell.Type = "date"
			}
		}
	}
	return cell
}

// Locale-dependent built-ins without an explicit code are deliberately left
// as raw values. A fallback is disclosed rather than guessing a workbook locale.
func workbookBuiltinFormat(id int) string {
	formats := map[int]string{
		0: "General", 1: "0", 2: "0.00", 3: "#,##0", 4: "#,##0.00",
		5: "$#,##0;($#,##0)", 6: "$#,##0;[Red]($#,##0)",
		7: "$#,##0.00;($#,##0.00)", 8: "$#,##0.00;[Red]($#,##0.00)",
		9: "0%", 10: "0.00%", 11: "0.00E+00", 12: "# ?/?", 13: "# ??/??",
		14: "m/d/yy", 15: "d-mmm-yy", 16: "d-mmm", 17: "mmm-yy",
		18: "h:mm AM/PM", 19: "h:mm:ss AM/PM", 20: "h:mm", 21: "h:mm:ss",
		22: "m/d/yy h:mm",
		37: "#,##0;(#,##0)", 38: "#,##0;[Red](#,##0)",
		39: "#,##0.00;(#,##0.00)", 40: "#,##0.00;[Red](#,##0.00)",
		45: "mm:ss", 46: "[h]:mm:ss", 47: "mmss.0", 48: "##0.0E+0", 49: "@",
	}
	if format, ok := formats[id]; ok {
		return format
	}
	return fmt.Sprintf("builtin:%d", id)
}

type workbookFormats struct {
	presentationStyles []XLSXCellStyle
	codes              map[int]string
	styles             []int
	date1904           bool
}

func (f workbookFormats) code(style int) string {
	if style < 0 || style >= len(f.styles) {
		return "General"
	}
	id := f.styles[style]
	if code, ok := f.codes[id]; ok {
		return code
	}
	return workbookBuiltinFormat(id)
}

func readWorkbookFormats(ctx context.Context, zr *zip.Reader, seen map[string]bool) (workbookFormats, error) {
	out := workbookFormats{codes: map[int]string{}}
	if seen["xl/workbook.xml"] {
		rc, dec, err := xlsxPart(ctx, zr, "xl/workbook.xml")
		if err != nil {
			return out, err
		}
		var book struct {
			Properties struct {
				Date1904 string `xml:"date1904,attr"`
			} `xml:"workbookPr"`
		}
		err = dec.Decode(&book)
		rc.Close()
		if err != nil {
			return out, err
		}
		out.date1904 = book.Properties.Date1904 == "1" || strings.EqualFold(book.Properties.Date1904, "true")
	}
	if !seen["xl/styles.xml"] {
		return out, nil
	}
	rc, dec, err := xlsxPart(ctx, zr, "xl/styles.xml")
	if err != nil {
		return out, err
	}
	defer rc.Close()
	var styles struct {
		Formats []struct {
			ID   int    `xml:"numFmtId,attr"`
			Code string `xml:"formatCode,attr"`
		} `xml:"numFmts>numFmt"`
		Bases []xlsxSourceXF   `xml:"cellStyleXfs>xf"`
		Fonts []xlsxSourceFont `xml:"fonts>font"`
		XFs   []xlsxSourceXF   `xml:"cellXfs>xf"`
	}
	if err := dec.Decode(&styles); err != nil {
		return out, err
	}
	for _, format := range styles.Formats {
		out.codes[format.ID] = format.Code
	}
	for _, xf := range styles.XFs {
		xf = inheritedXLSXXF(xf, styles.Bases)
		format := 0
		if xf.Format != nil {
			format = *xf.Format
		}
		out.styles = append(out.styles, format)
		out.presentationStyles = append(out.presentationStyles, savedXLSXStyle(styles.Fonts, xf))
	}
	return out, nil
}

type workbookFormatToken struct{ kind, text string }

// Split semicolon sections without splitting quoted or escaped literals.
func workbookFormatSections(code string) []string {
	var sections []string
	start, quoted, escaped := 0, false, false
	for i, ch := range code {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
		} else if ch == '"' {
			quoted = !quoted
		} else if ch == ';' && !quoted {
			sections = append(sections, code[start:i])
			start = i + 1
		}
	}
	return append(sections, code[start:])
}

func workbookFormatTokens(code string) ([]workbookFormatToken, bool) {
	var out []workbookFormatToken
	runes := []rune(code)
	add := func(kind, text string) { out = append(out, workbookFormatToken{kind, text}) }
	for i := 0; i < len(runes); {
		ch := runes[i]
		switch ch {
		case '"':
			j := i + 1
			for j < len(runes) && runes[j] != '"' {
				j++
			}
			if j == len(runes) {
				return nil, false
			}
			add("literal", string(runes[i+1:j]))
			i = j + 1
			continue
		case '\\', '_', '*':
			if i+1 == len(runes) {
				return nil, false
			}
			if ch == '\\' {
				add("literal", string(runes[i+1]))
			} else if ch == '_' {
				add("literal", " ")
			}
			i += 2
			continue
		case '[':
			j := i + 1
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j == len(runes) {
				return nil, false
			}
			part := strings.ToLower(string(runes[i+1 : j]))
			switch part {
			case "h", "hh", "m", "mm", "s", "ss":
				add("elapsed", part)
			case "black", "blue", "cyan", "green", "magenta", "red", "white", "yellow":
				// Color is not part of the copied display text.
			default:
				if symbol, locale, ok := workbookLocaleToken(string(runes[i+1 : j])); ok {
					if symbol != "" {
						add("literal", symbol)
					}
					if locale != "" {
						add("locale", locale)
					}
				} else {
					return nil, false
				}
			}
			i = j + 1
			continue
		}
		rest := strings.ToLower(string(runes[i:]))
		if strings.HasPrefix(rest, "am/pm") || strings.HasPrefix(rest, "a/p") {
			n := 5
			if strings.HasPrefix(rest, "a/p") {
				n = 3
			}
			add("ampm", string(runes[i:i+n]))
			i += n
			continue
		}
		lower := strings.ToLower(string(ch))
		if strings.Contains("ymdhs", lower) {
			j := i + 1
			for j < len(runes) && strings.EqualFold(string(runes[j]), lower) {
				j++
			}
			add(lower, string(runes[i:j]))
			i = j
			continue
		}
		if strings.ContainsRune("0#?.,%Ee+-/", ch) {
			add("pattern", string(ch))
		} else if ch >= '1' && ch <= '9' {
			add("digit", string(ch))
		} else if ch == 'g' || ch == 'G' {
			return nil, false // Era/calendar conversion is not a Gregorian literal.
		} else {
			add("literal", string(ch))
		}
		i++
	}
	return out, true
}

func workbookNumber(value float64, code string, date1904 bool) (string, bool, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || strings.HasPrefix(code, "builtin:") {
		return "", false, false
	}
	if code == "" || strings.EqualFold(code, "General") || code == "@" {
		return strconv.FormatFloat(value, 'f', -1, 64), false, true
	}
	signed := value
	section, value, ok := workbookSelectSection(value, workbookFormatSections(code))
	if !ok {
		return "", false, false
	}
	if strings.EqualFold(section, "General") || section == "@" {
		return strconv.FormatFloat(signed, 'f', -1, 64), false, true
	}
	tokens, ok := workbookFormatTokens(section)
	if !ok {
		return "", false, false
	}
	isDate, locale := false, ""
	displayTokens := make([]workbookFormatToken, 0, len(tokens))
	for _, token := range tokens {
		if token.kind == "locale" {
			if locale != "" && locale != token.text {
				return "", false, false
			}
			locale = token.text
			continue
		}
		if token.kind != "literal" && token.kind != "pattern" && token.kind != "digit" {
			isDate = true
		}
		displayTokens = append(displayTokens, token)
	}
	tokens = displayTokens
	if isDate {
		// Negative date serials have no portable Excel display; retain the raw value.
		display, ok := workbookDate(signed, tokens, date1904, locale)
		return display, true, ok
	}
	display, ok := workbookDecimal(value, tokens)
	return display, false, ok
}

var workbookDecimalPattern = regexp.MustCompile("^([#0]+(?:,[#0]+)*)(?:\\.([#0]+))?(?:[Ee]([+-])([0]+))?$")

func workbookDecimal(value float64, tokens []workbookFormatToken) (string, bool) {
	if display, fraction, ok := workbookFraction(value, tokens); fraction {
		return display, ok
	}
	first, last, percentages := -1, -1, 0
	for i, token := range tokens {
		if token.kind == "pattern" && (token.text == "0" || token.text == "#" || token.text == "?") {
			if first < 0 {
				first = i
			}
			last = i
		}
		if token.kind == "pattern" && token.text == "%" {
			percentages++
		}
	}
	if first < 0 {
		var out strings.Builder
		for _, token := range tokens {
			if token.kind != "literal" && token.kind != "digit" && token.text != "-" && token.text != "+" {
				return "", false
			}
			out.WriteString(token.text)
		}
		return out.String(), true
	}
	var pattern strings.Builder
	for _, token := range tokens[first : last+1] {
		if token.kind != "pattern" {
			return "", false
		}
		pattern.WriteString(token.text)
	}
	match := workbookDecimalPattern.FindStringSubmatch(pattern.String())
	if match == nil {
		return "", false
	}
	scale := 0
	for last+1 < len(tokens) && tokens[last+1].kind == "pattern" && tokens[last+1].text == "," {
		last++
		scale++
	}
	literal := func(items []workbookFormatToken) (string, bool) {
		var out strings.Builder
		for _, token := range items {
			if token.kind != "literal" && token.kind != "digit" && token.text != "%" && token.text != "-" && token.text != "+" {
				return "", false
			}
			out.WriteString(token.text)
		}
		return out.String(), true
	}
	prefix, ok := literal(tokens[:first])
	if !ok {
		return "", false
	}
	suffix, ok := literal(tokens[last+1:])
	if !ok {
		return "", false
	}
	value *= math.Pow(100, float64(percentages)) / math.Pow(1000, float64(scale))
	negative := value < 0
	value = math.Abs(value)
	precision := len(match[2])
	// Keep the existing decimal precision boundary for scientific mantissas too.
	if precision > 15 || math.IsInf(value, 0) || len(match[4]) > 10 {
		return "", false
	}
	number := ""
	if match[3] != "" {
		var ok bool
		number, ok = workbookScientific(value, match[1], match[2], match[3], match[4])
		if !ok {
			return "", false
		}
	} else {
		parts := strings.Split(strconv.FormatFloat(value, 'f', precision, 64), ".")
		integer := parts[0]
		zeros := strings.Count(match[1], "0")
		if len(integer) < zeros {
			integer = strings.Repeat("0", zeros-len(integer)) + integer
		}
		if integer == "0" && zeros == 0 {
			integer = ""
		}
		if strings.Contains(match[1], ",") {
			var groups []string
			for len(integer) > 3 {
				groups = append([]string{integer[len(integer)-3:]}, groups...)
				integer = integer[:len(integer)-3]
			}
			groups = append([]string{integer}, groups...)
			integer = strings.Join(groups, ",")
		}
		number = integer
		if len(parts) == 2 {
			fraction := parts[1]
			for len(fraction) > 0 && match[2][len(fraction)-1] == '#' && fraction[len(fraction)-1] == '0' {
				fraction = fraction[:len(fraction)-1]
			}
			if fraction != "" {
				number += "." + fraction
			}
		}
	}
	if negative {
		prefix = "-" + prefix
	}
	return prefix + number + suffix, true
}

func workbookDate(value float64, tokens []workbookFormatToken, date1904 bool, locale string) (string, bool) {
	if value < 0 || value > 2958465 {
		return "", false
	}
	precision, ampm, hasSeconds := 0, false, false
	for i, token := range tokens {
		if token.kind == "s" || token.kind == "elapsed" && strings.HasPrefix(token.text, "s") {
			hasSeconds = true
		}
		if token.kind == "ampm" {
			ampm = true
		}
		if token.kind == "s" && i+2 < len(tokens) && tokens[i+1].text == "." {
			for j := i + 2; j < len(tokens) && tokens[j].kind == "pattern" && tokens[j].text == "0"; j++ {
				precision++
			}
		}
	}
	if precision > 9 {
		return "", false
	}
	// Round within a day so large serials do not lose fractional-second precision.
	days := int(math.Floor(value))
	factor := math.Pow10(precision)
	seconds := math.Floor((value - float64(days)) * 86400)
	if hasSeconds {
		seconds = math.Round((value-float64(days))*86400*factor) / factor
	}
	if seconds >= 86400 {
		days++
		seconds = 0
	}
	whole := int(seconds)
	hour, minute, second := whole/3600, whole/60%60, whole%60
	base, offset := time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC), days
	if date1904 {
		base = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	} else if days >= 60 {
		offset--
	}
	date := base.AddDate(0, 0, offset)
	year, month, day := date.Year(), int(date.Month()), date.Day()
	if !date1904 && days == 60 {
		year, month, day = 1900, 2, 29 // Excel's retained leap-year compatibility.
	} else if !date1904 && days == 0 {
		year, month, day = 1900, 1, 0
	}
	if year > 9999 {
		return "", false
	}
	names := workbookDateLocales[locale]
	if locale == "" {
		names = workbookDateLocales["en"]
	}
	number := func(n, width int) string {
		if width > 1 {
			return fmt.Sprintf("%02d", n)
		}
		return strconv.Itoa(n)
	}
	var out strings.Builder
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		width := len(token.text)
		switch token.kind {
		case "literal":
			out.WriteString(token.text)
		case "digit":
			out.WriteString(token.text)
		case "pattern":
			if token.text != "/" && token.text != "-" && token.text != "." {
				return "", false
			}
			out.WriteString(token.text)
		case "y":
			if width <= 2 {
				out.WriteString(fmt.Sprintf("%02d", year%100))
			} else {
				out.WriteString(fmt.Sprintf("%04d", year))
			}
		case "d":
			switch width {
			case 1, 2:
				out.WriteString(number(day, width))
			case 3:
				out.WriteString(names.abbreviatedDays[int(date.Weekday())])
			case 4:
				out.WriteString(names.days[int(date.Weekday())])
			default:
				return "", false
			}
		case "m":
			minutes := false
			for j := i - 1; j >= 0; j-- {
				if tokens[j].kind == "literal" || tokens[j].kind == "pattern" {
					continue
				}
				minutes = tokens[j].kind == "h" || tokens[j].kind == "elapsed" && strings.HasPrefix(tokens[j].text, "h")
				break
			}
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].kind == "literal" || tokens[j].kind == "pattern" {
					continue
				}
				minutes = minutes || tokens[j].kind == "s"
				break
			}
			if minutes {
				out.WriteString(number(minute, width))
			} else {
				switch width {
				case 1, 2:
					out.WriteString(number(month, width))
				case 3:
					out.WriteString(names.abbreviatedMonths[month-1])
				case 4:
					out.WriteString(names.months[month-1])
				case 5:
					out.WriteString(names.narrowMonths[month-1])
				default:
					return "", false
				}
			}
		case "h":
			if width > 2 {
				return "", false
			}
			h := hour
			if ampm {
				h = hour % 12
				if h == 0 {
					h = 12
				}
			}
			out.WriteString(number(h, width))
		case "s":
			if width > 2 {
				return "", false
			}
			out.WriteString(number(second, width))
			if precision > 0 && i+2 < len(tokens) && tokens[i+1].text == "." && tokens[i+2].text == "0" {
				fraction := int(math.Round((seconds - float64(whole)) * factor))
				out.WriteString(fmt.Sprintf(".%0*d", precision, fraction))
				i += precision + 1
			}
		case "ampm":
			parts := strings.Split(token.text, "/")
			if locale != "" && len(token.text) == 5 {
				parts = names.meridiem
			}
			if hour < 12 {
				out.WriteString(parts[0])
			} else {
				out.WriteString(parts[1])
			}
		case "elapsed":
			total := int64(days)*86400 + int64(whole)
			switch token.text[0] {
			case 'h':
				total /= 3600
			case 'm':
				total /= 60
			}
			out.WriteString(number(int(total), width))
		default:
			return "", false
		}
	}
	return out.String(), true
}

func workbookFormatNotice(cells []XLSXCell) string {
	count := 0
	for _, cell := range cells {
		if !cell.FormatSupported {
			count++
		}
	}
	if count == 0 {
		return ""
	}
	return fmt.Sprintf("\n%d cell(s) use unsupported number formats and show stored values. Supported: decimal/grouped/percent/currency, fractions, scientific/engineering, conditional sections and date/time formats. Explicit en/de/fr/zh/ja Gregorian locale markers are supported; unknown system/locale/calendar directives and other unsupported codes retain raw values. Raw values remain copyable.\n", count)
}
