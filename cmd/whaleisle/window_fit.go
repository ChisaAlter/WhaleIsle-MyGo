package main

import "github.com/egoist/mygo"

// fitWindowSize fits the preferred window inside a work area, keeping at least
// the preferred minimum when the display can hold it. A 1440x920 default does
// not fit a 1138x640 DIP work area (a 1707x960 screen at 150%), and the old
// fixed size put the account row and lower settings rows off-screen.
func fitWindowSize(width, height, minWidth, minHeight, workWidth, workHeight int) (int, int, int, int) {
	maxWidth := workWidth - 48
	maxHeight := workHeight - 48
	if maxWidth <= 0 || maxHeight <= 0 {
		return width, height, minWidth, minHeight
	}
	if minWidth > maxWidth {
		minWidth = maxWidth
	}
	if minHeight > maxHeight {
		minHeight = maxHeight
	}
	if width > maxWidth {
		width = maxWidth
	}
	if height > maxHeight {
		height = maxHeight
	}
	return width, height, minWidth, minHeight
}

func mainWindowSize() (width, height, minWidth, minHeight int) {
	width, height, minWidth, minHeight = 1440, 920, 960, 640
	displays := mygo.Screen.Displays()
	if len(displays) == 0 {
		return
	}
	work := displays[0].WorkArea
	return fitWindowSize(width, height, minWidth, minHeight, work.Width, work.Height)
}
