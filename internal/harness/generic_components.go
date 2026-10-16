package harness

// ResolveComponentNode uses the same Node selection as the Host without
// importing the component manager (which itself uses Host install helpers).
func ResolveComponentNode(configured string) (string, error) {
	return resolveNode(configured)
}
