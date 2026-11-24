package coss

import (
	"sync"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// Icon names a Lucide icon (coss's icon set, ISC) drawn in currentColor.
type Icon uint8

const (
	IconNone Icon = iota
	IconChevronDown
	IconChevronUp
	IconChevronRight
	IconChevronLeft
	IconChevronsUpDown
	IconChevronsLeft
	IconChevronsRight
	IconCheck
	IconX
	IconMinus
	IconPlus
	IconSearch
	IconLoader
	IconEllipsis
	IconInfo
	IconCircleAlert
	IconTriangleAlert
	IconCircleCheck
	IconArrowDown
	IconArrowUp
	IconPaperclip
	IconAtSign
	IconReply
	IconCopy
	IconPencil
	IconTrash
	IconWifiOff
	IconLock
	IconMessage
	IconSlash
	IconCircleDot
	IconSmile
	IconFile
	// IconCheckboxCheck and IconCheckboxMinus are coss's own checkbox
	// indicators, drawn at stroke width 3.
	IconCheckboxCheck
	IconCheckboxMinus
	// Page icons (Lucide, ISC) for the tool pages: plugins, market,
	// scheduled tasks and usage.
	IconRefresh
	IconPlay
	IconPause
	IconCalendar
	IconClock
	IconDownload
	IconPuzzle
	IconExternalLink
	IconHistory
	IconChartBar
	IconArrowLeft
	iconCount
)

var iconPaths = [iconCount]string{
	IconChevronDown:    `<path d="m6 9 6 6 6-6"/>`,
	IconChevronUp:      `<path d="m18 15-6-6-6 6"/>`,
	IconChevronRight:   `<path d="m9 18 6-6-6-6"/>`,
	IconChevronLeft:    `<path d="m15 18-6-6 6-6"/>`,
	IconChevronsUpDown: `<path d="m7 15 5 5 5-5"/><path d="m7 9 5-5 5 5"/>`,
	IconChevronsLeft:   `<path d="m11 17-5-5 5-5"/><path d="m18 17-5-5 5-5"/>`,
	IconChevronsRight:  `<path d="m6 17 5-5-5-5"/><path d="m13 17 5-5-5-5"/>`,
	IconCheck:          `<path d="M20 6 9 17l-5-5"/>`,
	IconX:              `<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`,
	IconMinus:          `<path d="M5 12h14"/>`,
	IconPlus:           `<path d="M5 12h14"/><path d="M12 5v14"/>`,
	IconSearch:         `<path d="m21 21-4.34-4.34"/><circle cx="11" cy="11" r="8"/>`,
	IconLoader:         `<path d="M21 12a9 9 0 1 1-6.219-8.56"/>`,
	IconEllipsis:       `<circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/>`,
	IconInfo:           `<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>`,
	IconCircleAlert:    `<circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/>`,
	IconTriangleAlert:  `<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>`,
	IconCircleCheck:    `<circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/>`,
	IconArrowDown:      `<path d="M12 5v14"/><path d="m19 12-7 7-7-7"/>`,
	IconArrowUp:        `<path d="m5 12 7-7 7 7"/><path d="M12 19V5"/>`,
	IconPaperclip:      `<path d="m16 6-8.414 8.586a2 2 0 0 0 2.829 2.829l8.414-8.586a4 4 0 1 0-5.657-5.657l-8.379 8.551a6 6 0 1 0 8.485 8.485l8.379-8.551"/>`,
	IconAtSign:         `<circle cx="12" cy="12" r="4"/><path d="M16 8v5a3 3 0 0 0 6 0v-1a10 10 0 1 0-4 8"/>`,
	IconReply:          `<path d="M20 18v-2a4 4 0 0 0-4-4H4"/><path d="m9 17-5-5 5-5"/>`,
	IconCopy:           `<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>`,
	IconPencil:         `<path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"/><path d="m15 5 4 4"/>`,
	IconTrash:          `<path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2"/>`,
	IconWifiOff:        `<path d="M12 20h.01"/><path d="M8.5 16.429a5 5 0 0 1 7 0"/><path d="M5 12.859a10 10 0 0 1 5.17-2.69"/><path d="M19 12.859a10 10 0 0 0-2.007-1.523"/><path d="M2 8.82a15 15 0 0 1 4.177-2.643"/><path d="M22 8.82a15 15 0 0 0-11.288-3.764"/><path d="m2 2 20 20"/>`,
	IconLock:           `<rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>`,
	IconMessage:        `<path d="M22 17a2 2 0 0 1-2 2H6.828a2 2 0 0 0-1.414.586l-2.202 2.202A.71.71 0 0 1 2 21.286V5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2z"/>`,
	IconSlash:          `<path d="M22 2 2 22"/>`,
	IconCircleDot:      `<circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="1"/>`,
	IconSmile:          `<circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><path d="M9 9h.01"/><path d="M15 9h.01"/>`,
	IconCheckboxCheck:  `<path d="M5.252 12.7 10.2 18.63 18.748 5.37"/>`,
	IconCheckboxMinus:  `<path d="M5.252 12h13.496"/>`,
	IconFile:           `<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/>`,
	IconRefresh:        `<path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/>`,
	IconPlay:           `<path d="M5 5a2 2 0 0 1 3.008-1.728l11.997 6.998a2 2 0 0 1 .003 3.458l-12 7A2 2 0 0 1 5 19z"/>`,
	IconPause:          `<rect x="14" y="3" width="5" height="18" rx="1"/><rect x="5" y="3" width="5" height="18" rx="1"/>`,
	IconCalendar:       `<path d="M8 2v4"/><path d="M16 2v4"/><rect width="18" height="18" x="3" y="4" rx="2"/><path d="M3 10h18"/>`,
	IconClock:          `<circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>`,
	IconDownload:       `<path d="M12 15V3"/><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/>`,
	IconPuzzle:         `<path d="M15.39 4.39a1 1 0 0 0 1.68-.474 2.5 2.5 0 1 1 3.014 3.015 1 1 0 0 0-.474 1.68l1.683 1.682a2.414 2.414 0 0 1 0 3.414L19.61 15.39a1 1 0 0 1-1.68-.474 2.5 2.5 0 1 0-3.014 3.015 1 1 0 0 1 .474 1.68l-1.683 1.682a2.414 2.414 0 0 1-3.414 0L8.61 19.61a1 1 0 0 0-1.68.474 2.5 2.5 0 1 1-3.014-3.015 1 1 0 0 0 .474-1.68l-1.683-1.682a2.414 2.414 0 0 1 0-3.414L4.39 8.61a1 1 0 0 1 1.68.474 2.5 2.5 0 1 0 3.014-3.015 1 1 0 0 1-.474-1.68l1.683-1.682a2.414 2.414 0 0 1 3.414 0z"/>`,
	IconExternalLink:   `<path d="M15 3h6v6"/><path d="M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>`,
	IconHistory:        `<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/>`,
	IconChartBar:       `<path d="M3 3v16a2 2 0 0 0 2 2h16"/><path d="M18 17V9"/><path d="M13 17V5"/><path d="M8 17v-3"/>`,
	IconArrowLeft:      `<path d="m12 19-7-7 7-7"/><path d="M19 12H5"/>`,
}

var (
	iconMu  sync.Mutex
	iconSVG [iconCount]*ui.SVG
)

// SVG returns the parsed icon, parsed once.
func (i Icon) SVG() *ui.SVG {
	if i == IconNone || i >= iconCount {
		return nil
	}
	iconMu.Lock()
	defer iconMu.Unlock()
	if s := iconSVG[i]; s != nil {
		return s
	}
	src := `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="` + i.stroke() + `" stroke-linecap="round" stroke-linejoin="round">` + iconPaths[i] + `</svg>`
	s := ui.MustParseSVG([]byte(src))
	iconSVG[i] = s
	return s
}

func (i Icon) stroke() string {
	if i == IconCheckboxCheck || i == IconCheckboxMinus {
		return "3"
	}
	return "2"
}

// icon builds i at size in col, at the 80% opacity coss gives icons
// without an opacity class unless full is set.
func icon(c *ui.Context, i Icon, size float32, col ui.Color, full bool) ui.Element {
	e := ui.Icon(c, i.SVG()).Size(size, size).Shrink(0).TextColor(col)
	if !full {
		e.Opacity(design.OpacityIcon)
	}
	return e
}
