package settings

import (
	"fmt"
	"unicode/utf16"
)

const CustomInstructionsNamespace = "ui-conversation"
const CustomInstructionsField = "customInstructions"
const CustomInstructionsMaxLength = 1500

// CustomInstructionsLength matches JavaScript string.length and the Host's
// z.string().max(1500), including two UTF-16 units for astral characters.
func CustomInstructionsLength(text string) int { return len(utf16.Encode([]rune(text))) }

func validateCustomInstructions(ns string, drafts map[string]string) error {
	if ns == CustomInstructionsNamespace && CustomInstructionsLength(drafts[CustomInstructionsField]) > CustomInstructionsMaxLength {
		return fmt.Errorf("自定义指令不能超过 1500 字符 / Custom instructions must not exceed 1500 UTF-16 units")
	}
	return nil
}
