package shortcuts

// NativeCommandScope is the desktop arbitration contract, independent of a
// command's default or custom physical binding. The original Windows/macOS
// native-priority policy accepts editable and terminal input; command resolvers
// retain their own modal and resource restrictions.
func NativeCommandScope(id string) string {
	switch id {
	case "page.close":
		return "page-close"
	case "page.refresh", "sidebar.right.toggle", "pane.split", "pane.splitDown", "pane.fullscreen.toggle", "session.rename", "harness.restart", "launcher.open":
		return "native-no-modal"
	case "settings.open", "settings.mcp", "settings.skills", "settings.about":
		return "settings-open"
	case "shortcuts.open":
		return "shortcuts-open"
	default:
		return "native-priority"
	}
}
