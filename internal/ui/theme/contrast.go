package theme

// SecondaryInk returns the shipped palette's secondary label for a scheme
// (--dsw-alias-label-tertiary, the harness's label-tertiary alias), as a CSS
// color. The glass materials keep at least 3:1 for it on every surface they
// paint: it is the lightest light-half label and the dimmest dark-half label the
// shell ships, so text in that color stays readable on glass over any wallpaper.
//
// A custom family derives its own tertiary label from its seeds (derive.ts), so
// a family with an extreme seed can come out dimmer than this reference; the
// theme resolver owns that derivation.
func SecondaryInk(dark bool) string {
	if dark {
		return deepseekDark.Muted
	}
	return deepseekLight.Muted
}

// PrimaryInk returns the shipped palette's body label (--dsw-alias-label-primary)
// for a scheme, as a CSS color.
func PrimaryInk(dark bool) string {
	if dark {
		return deepseekDark.FG
	}
	return deepseekLight.FG
}
