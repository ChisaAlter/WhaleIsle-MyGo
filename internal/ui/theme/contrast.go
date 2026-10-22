package theme

// SecondaryInk returns the shipped palette's secondary label for a scheme
// (--dsw-alias-label-secondary), as a CSS color.
func SecondaryInk(dark bool) string {
	if dark {
		return deepseekDark.Secondary
	}
	return deepseekLight.Secondary
}

// PrimaryInk returns the shipped palette's body label (--dsw-alias-label-primary)
// for a scheme, as a CSS color.
func PrimaryInk(dark bool) string {
	if dark {
		return deepseekDark.FG
	}
	return deepseekLight.FG
}
