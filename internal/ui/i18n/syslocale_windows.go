//go:build windows

package i18n

import "syscall"

// SystemLocale reports the language Windows itself is set to, so the dialogs
// shown before config.json (and therefore the stored locale) can be read still
// speak one language. Only the two catalog languages exist, so a system set to
// any other language reads English.
func SystemLocale() string {
	langID, _, _ := procGetUserDefaultUILanguage.Call()
	// The primary language id is the low 10 bits of a LANGID; 0x04 is Chinese
	// (any script or region).
	if uint16(langID)&0x3ff == 0x04 {
		return LocaleZH
	}
	return LocaleEN
}

var procGetUserDefaultUILanguage = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")
