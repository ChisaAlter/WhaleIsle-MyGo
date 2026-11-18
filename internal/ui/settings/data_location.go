package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// DataLocationAccess is the 数据位置 section of 通用: the directory MyGo keeps
// its data in, where to change it, and why a chosen directory was refused.
//
// The access is rebuilt from application state on every frame; Err and
// LastAction therefore carry the outcome of the last attempt, never a draft.
type DataLocationAccess struct {
	// Current is the directory this process is using.
	Current string
	// Default is where 恢复默认 points.
	Default string
	// Recorded is a recorded location that is not usable right now, or the
	// record file itself when the record cannot be read. It is rendered so a
	// fallback to the default is visible instead of silent.
	Recorded string
	// RecordBroken is true when Recorded names an unreadable record rather
	// than a directory that is gone.
	RecordBroken bool
	// ReadOnly marks a run whose data directory comes from the environment or
	// from isolated mode, where recording a location would be ignored.
	ReadOnly       bool
	ReadOnlyReason string
	// Err is the last refusal or failure, in the active locale.
	Err string
	// LastAction reports an applied change or an import outcome.
	LastAction string
	// ImportAvailable offers the legacy directory again after it was
	// skipped or when the migrated copy is gone. It stays available in a
	// read-only run: importing writes into the fixed directory and never
	// touches the record.
	ImportAvailable bool
	// PreviousPath is the location this one replaced. It is offered as an
	// import source after a change moved the profile.
	PreviousPath     string
	OnChange         func()
	OnOpen           func()
	OnReset          func()
	OnImport         func()
	OnImportPrevious func()
}

// dataLocationRow paints the whole data-location section: the current
// directory, the reason a recorded one is not in use, the three actions and
// the last outcome. It uses the shared block rhythm so the page keeps the
// same hairline spacing as its neighbouring rows.
func dataLocationRow(c *ui.Context, d *DataLocationAccess, fg, muted ui.Color) {
	if d == nil {
		return
	}
	// The hint names only what is on screen: the import sentence appears when
	// an import button does, in a read-only run as well.
	hint := i18n.T(i18n.KeyDataLocationHint)
	if d.ImportAvailable || d.PreviousPath != "" {
		hint += " " + i18n.T(i18n.KeyDataLocationHintImport)
	}
	settingsFieldBlock(c, "general-data-location", i18n.T(i18n.KeyDataLocationTitle), hint, fg, muted, func() {
		current := strings.TrimSpace(d.Current)
		if current == "" {
			current = d.Default
		}
		if current != "" {
			// Long paths keep both ends readable; the tooltip carries the
			// exact string the change and reset actions compare.
			ui.Text(c, i18n.T(i18n.KeyDataLocationCurrent)+abbreviatePathMiddle(current, 60)).
				FillWidth().MinWidth(0).SingleLine().Ellipsis("…").
				FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).
				Tooltip(current).TextColor(fg)
		}
		if recorded := strings.TrimSpace(d.Recorded); recorded != "" {
			// Two different sentences: a recorded directory that is gone is
			// still a directory, while a broken record names the record file
			// and must not read as if the default were that file.
			label := fmt.Sprintf(i18n.T(i18n.KeyDataLocationUnavailable), recorded)
			if d.RecordBroken {
				label = fmt.Sprintf(i18n.T(i18n.KeyDataLocationRecordBroken), recorded)
			}
			ui.Text(c, label).
				FillWidth().MinWidth(0).SingleLine().Ellipsis("…").
				FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
				Tooltip(recorded).TextColor(c.Theme().Warning)
		}
		if d.ReadOnly {
			if reason := strings.TrimSpace(d.ReadOnlyReason); reason != "" {
				ui.Text(c, reason).FillWidth().MinWidth(0).
					FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
					TextColor(muted)
			}
		}
		// A read-only run keeps the folder and import actions: neither records
		// a location. Change and reset would write a record the next launch
		// ignores, so they are refused rather than shown disabled.
		editable := !d.ReadOnly
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if editable {
				if button(c, i18n.T(i18n.KeyDataLocationChange), d.OnChange != nil) {
					d.OnChange()
				}
			}
			if button(c, i18n.T(i18n.KeyDataLocationOpen), d.OnOpen != nil) {
				d.OnOpen()
			}
			if editable && (d.Recorded != "" || !strings.EqualFold(strings.TrimSpace(d.Current), strings.TrimSpace(d.Default))) {
				if button(c, i18n.T(i18n.KeyDataLocationReset), d.OnReset != nil) {
					d.OnReset()
				}
			}
			if d.ImportAvailable {
				if button(c, i18n.T(i18n.KeyDataLocationImport), d.OnImport != nil) {
					d.OnImport()
				}
			}
			if d.PreviousPath != "" {
				if button(c, i18n.T(i18n.KeyDataLocationImportPrevious), d.OnImportPrevious != nil) {
					d.OnImportPrevious()
				}
			}
		})
		if err := strings.TrimSpace(d.Err); err != "" {
			ui.Text(c, err).FillWidth().MinWidth(0).
				FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
				TextColor(c.Theme().Warning)
		}
		if action := strings.TrimSpace(d.LastAction); action != "" {
			settingsStatusLine(c, action, muted)
		}
	})
}

// button is the secondary action shape the settings pages use for row
// actions: a hairline border, no fill, 30 high.
func button(c *ui.Context, label string, enabled bool) bool {
	control := ui.Button(c, label).Height(30).Padding(0, 12).Radius(8).
		Border(0.5, c.Theme().Border.Alpha(0.45)).
		FontSize(theme.InterfaceSize(12.5))
	if !enabled {
		control = control.Disabled(true)
	}
	return enabled && control.Clicked()
}

// abbreviatePathMiddle shortens a path in the middle, keeping its root and
// its leaf so both the drive and the last directory stay readable.
func abbreviatePathMiddle(path string, limit int) string {
	runes := []rune(path)
	if limit <= 0 || len(runes) <= limit {
		return path
	}
	const ellipsis = "…"
	keep := limit - len([]rune(ellipsis))
	if keep <= 1 {
		return path
	}
	head := keep / 3
	tail := keep - head
	return string(runes[:head]) + ellipsis + string(runes[len(runes)-tail:])
}
