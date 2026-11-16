package launcherctl

const (
	WindowWidth     = 1060
	WindowHeight    = 660
	WindowMinWidth  = 860
	WindowMinHeight = 560
	CaptionHeight   = 48
)

// FirstWindow reports whether cold startup hands the initial visible window
// to the dedicated launcher window rather than the main window.
func FirstWindow(isolated, hidden, background, settings bool) bool {
	return !isolated && !hidden && !background && !settings
}

// Stage abstracts the native window operations needed by the launcher stage machine.
type Stage interface {
	LauncherOpen() bool
	ShowLauncher() bool
	CloseLauncher()
	ShowMain()
	MainVisible() bool
	Quitting() bool
}

// StageResult describes what window transitions ApplyStage performed.
type StageResult struct {
	LauncherShown  bool
	LauncherClosed bool
	MainShown      bool
}

// ApplyStage synchronizes the launcher and main windows with the desired open flag.
func ApplyStage(stage Stage, open bool) StageResult {
	var res StageResult
	if IsolatedDesktopHidden() {
		return res
	}
	if open {
		if stage.LauncherOpen() {
			return res
		}
		if stage.ShowLauncher() {
			res.LauncherShown = true
			return res
		}
		if !stage.Quitting() && !stage.MainVisible() {
			stage.ShowMain()
			res.MainShown = true
		}
		return res
	}
	if stage.LauncherOpen() {
		stage.CloseLauncher()
		res.LauncherClosed = true
	}
	if !stage.Quitting() && !stage.MainVisible() {
		stage.ShowMain()
		res.MainShown = true
	}
	return res
}
