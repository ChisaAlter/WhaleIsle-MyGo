package workspacefs

import (
	"strconv"
	"strings"
)

// Number separators remain the explicit format's separators. Currency tokens
// insert only their supplied symbol; no currency, rate, or system locale is inferred.
func workbookLocaleToken(block string) (string, string, bool) {
	if !strings.HasPrefix(block, "$") {
		return "", "", false
	}
	value := strings.TrimPrefix(block, "$")
	for i, ch := range value {
		if ch != '-' {
			continue
		}
		if locale, ok := workbookKnownLocale(value[i+1:]); ok {
			return value[:i], locale, true
		}
	}
	if value == "" || strings.Contains(value, "-") {
		// F800/F400 and other system-selected or unknown locale/calendar codes
		// cannot be resolved from the document alone.
		return "", "", false
	}
	return value, "", true
}

func workbookKnownLocale(value string) (string, bool) {
	switch strings.ToLower(value) {
	case "en", "en-us", "en-gb":
		return "en", true
	case "de", "de-de":
		return "de", true
	case "fr", "fr-fr":
		return "fr", true
	case "zh", "zh-cn", "zh-hans", "zh-hans-cn":
		return "zh", true
	case "zh-tw", "zh-hant", "zh-hant-tw":
		return "zh-Hant", true
	case "ja", "ja-jp":
		return "ja", true
	}
	id, err := strconv.ParseUint(value, 16, 16)
	if err != nil {
		return "", false
	}
	switch id {
	case 0x409, 0x809:
		return "en", true
	case 0x407:
		return "de", true
	case 0x40c:
		return "fr", true
	case 0x804:
		return "zh", true
	case 0x404:
		return "zh-Hant", true
	case 0x411:
		return "ja", true
	}
	return "", false
}

type workbookDateLocale struct {
	months, abbreviatedMonths, narrowMonths []string
	days, abbreviatedDays                  []string
	meridiem                               []string
}

// Gregorian names for explicit locale markers; the non-English names follow
// Unicode CLDR's Gregorian format-name data. System and alternate-calendar
// directives are left unsupported rather than silently changing the date.
var workbookDateLocales = map[string]workbookDateLocale{
	"en": {
		months: strings.Split("January|February|March|April|May|June|July|August|September|October|November|December", "|"),
		abbreviatedMonths: strings.Split("Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec", "|"),
		narrowMonths: strings.Split("J|F|M|A|M|J|J|A|S|O|N|D", "|"),
		days: strings.Split("Sunday|Monday|Tuesday|Wednesday|Thursday|Friday|Saturday", "|"),
		abbreviatedDays: strings.Split("Sun|Mon|Tue|Wed|Thu|Fri|Sat", "|"),
		meridiem: []string{"AM", "PM"},
	},
	"de": {
		months: strings.Split("Januar|Februar|März|April|Mai|Juni|Juli|August|September|Oktober|November|Dezember", "|"),
		abbreviatedMonths: strings.Split("Jan.|Feb.|März|Apr.|Mai|Juni|Juli|Aug.|Sept.|Okt.|Nov.|Dez.", "|"),
		narrowMonths: strings.Split("J|F|M|A|M|J|J|A|S|O|N|D", "|"),
		days: strings.Split("Sonntag|Montag|Dienstag|Mittwoch|Donnerstag|Freitag|Samstag", "|"),
		abbreviatedDays: strings.Split("So.|Mo.|Di.|Mi.|Do.|Fr.|Sa.", "|"),
		meridiem: []string{"AM", "PM"},
	},
	"fr": {
		months: strings.Split("janvier|février|mars|avril|mai|juin|juillet|août|septembre|octobre|novembre|décembre", "|"),
		abbreviatedMonths: strings.Split("janv.|févr.|mars|avr.|mai|juin|juil.|août|sept.|oct.|nov.|déc.", "|"),
		narrowMonths: strings.Split("J|F|M|A|M|J|J|A|S|O|N|D", "|"),
		days: strings.Split("dimanche|lundi|mardi|mercredi|jeudi|vendredi|samedi", "|"),
		abbreviatedDays: strings.Split("dim.|lun.|mar.|mer.|jeu.|ven.|sam.", "|"),
		meridiem: []string{"AM", "PM"},
	},
	"zh": {
		months: strings.Split("一月|二月|三月|四月|五月|六月|七月|八月|九月|十月|十一月|十二月", "|"),
		abbreviatedMonths: strings.Split("1月|2月|3月|4月|5月|6月|7月|8月|9月|10月|11月|12月", "|"),
		narrowMonths: strings.Split("1|2|3|4|5|6|7|8|9|10|11|12", "|"),
		days: strings.Split("星期日|星期一|星期二|星期三|星期四|星期五|星期六", "|"),
		abbreviatedDays: strings.Split("周日|周一|周二|周三|周四|周五|周六", "|"),
		meridiem: []string{"上午", "下午"},
	},
	"zh-Hant": {
		months: strings.Split("1月|2月|3月|4月|5月|6月|7月|8月|9月|10月|11月|12月", "|"),
		abbreviatedMonths: strings.Split("1月|2月|3月|4月|5月|6月|7月|8月|9月|10月|11月|12月", "|"),
		narrowMonths: strings.Split("1|2|3|4|5|6|7|8|9|10|11|12", "|"),
		days: strings.Split("星期日|星期一|星期二|星期三|星期四|星期五|星期六", "|"),
		abbreviatedDays: strings.Split("週日|週一|週二|週三|週四|週五|週六", "|"),
		meridiem: []string{"上午", "下午"},
	},
	"ja": {
		months: strings.Split("1月|2月|3月|4月|5月|6月|7月|8月|9月|10月|11月|12月", "|"),
		abbreviatedMonths: strings.Split("1月|2月|3月|4月|5月|6月|7月|8月|9月|10月|11月|12月", "|"),
		narrowMonths: strings.Split("1|2|3|4|5|6|7|8|9|10|11|12", "|"),
		days: strings.Split("日曜日|月曜日|火曜日|水曜日|木曜日|金曜日|土曜日", "|"),
		abbreviatedDays: strings.Split("日|月|火|水|木|金|土", "|"),
		meridiem: []string{"午前", "午後"},
	},
}
