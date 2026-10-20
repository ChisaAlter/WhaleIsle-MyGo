package settings

import (
	"errors"
	"unicode/utf16"
	"whaleisle/internal/store/storelocale"
)

const CustomInstructionsNamespace = "ui-conversation"
const CustomInstructionsField = "customInstructions"
const CustomInstructionsMaxLength = 1500

// CustomInstructionsLength matches JavaScript string.length and the Host's
// z.string().max(1500), including two UTF-16 units for astral characters.
func CustomInstructionsLength(text string) int { return len(utf16.Encode([]rune(text))) }

func LimitCustomInstructions(text string) string {
	units := 0
	for offset, r := range text {
		size := 1
		if r > 0xffff {
			size = 2
		}
		if units+size > CustomInstructionsMaxLength {
			return text[:offset]
		}
		units += size
	}
	return text
}

func validateCustomInstructions(ns string, drafts map[string]string) error {
	if ns == CustomInstructionsNamespace && CustomInstructionsLength(drafts[CustomInstructionsField]) > CustomInstructionsMaxLength {
		return errors.New(storelocale.T(storelocale.KeyStoreSettingsCustomInstructionsMustNotExceed1500))
	}
	return nil
}
