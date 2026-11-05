package session

// WorkspaceDisplayTitle localizes the Host's automatic title without changing
// its stored identity. An explicit rename to that label pins the chosen name.
func WorkspaceDisplayTitle(title, localizedDefault string) string {
	if title == "default-workspace" {
		return localizedDefault
	}
	return title
}
