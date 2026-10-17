package browserpreview

import "whaleisle/internal/ui/i18n"

func browserChromeLabel(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}
