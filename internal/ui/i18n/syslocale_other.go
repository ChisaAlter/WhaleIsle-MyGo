//go:build !windows

package i18n

import (
	"os"
	"strings"
)

// SystemLocale reports the language the session is set to, so the dialogs shown
// before config.json (and therefore the stored locale) can be read still speak
// one language. LC_ALL and LC_MESSAGES outrank LANG, and a POSIX ("C") setting
// reads English.
func SystemLocale() string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "zh") {
			return LocaleZH
		}
		return LocaleEN
	}
	return LocaleEN
}
