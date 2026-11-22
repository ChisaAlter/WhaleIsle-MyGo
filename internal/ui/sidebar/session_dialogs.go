package sidebar

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/workspacepicker"
)

// ArchiveConfirmOtherWork describes custom active work for a running session.
type ArchiveConfirmOtherWork struct {
	Kind  string
	Count int
}

// ArchiveConfirmState holds the state for the running-session archive confirmation modal.
type ArchiveConfirmState struct {
	SessionID   string
	Title       string
	TurnRunning bool
	Subagents   []string
	Jobs        []string
	Schedules   []string
	Others      []ArchiveConfirmOtherWork
	Busy        bool
	Error       string
	OnConfirm   func(sessionID string)
	OnCancel    func()
}

// DeleteConfirmState holds the state for the single-session permanent deletion modal.
type DeleteConfirmState struct {
	SessionID string
	Title     string
	Busy      bool
	ErrorCode string
	Error     string
	OnConfirm func(sessionID string)
	OnCancel  func()
}

// FormatSessionDeleteError maps a Host session deletion error code to its localized message.
func FormatSessionDeleteError(codeOrErr string) string {
	return workspacepicker.FormatSessionDeleteError(codeOrErr)
}

// FormatArchiveConfirmActivityLines returns the localized list of active work items that will be stopped.
func FormatArchiveConfirmActivityLines(s ArchiveConfirmState) []string {
	var lines []string
	sep := i18n.T(i18n.KeyWSArchiveConfirmListSeparator)
	if s.TurnRunning {
		lines = append(lines, i18n.T(i18n.KeyWSArchiveConfirmTurn))
	}
	if len(s.Subagents) > 0 {
		n := strconv.Itoa(len(s.Subagents))
		lines = append(lines, i18n.WSPlural(i18n.KeyWSArchiveConfirmSubagentsBase, len(s.Subagents), "n", n, "names", strings.Join(s.Subagents, sep)))
	}
	if len(s.Jobs) > 0 {
		n := strconv.Itoa(len(s.Jobs))
		lines = append(lines, i18n.WSPlural(i18n.KeyWSArchiveConfirmJobsBase, len(s.Jobs), "n", n, "names", strings.Join(s.Jobs, sep)))
	}
	if len(s.Schedules) > 0 {
		n := strconv.Itoa(len(s.Schedules))
		lines = append(lines, i18n.WSPlural(i18n.KeyWSArchiveConfirmSchedulesBase, len(s.Schedules), "n", n, "names", strings.Join(s.Schedules, sep)))
	}
	for _, other := range s.Others {
		count := other.Count
		if count < 1 {
			count = 1
		}
		lines = append(lines, i18n.WSPlural(i18n.KeyWSArchiveConfirmOtherBase, count, "n", strconv.Itoa(count), "kind", other.Kind))
	}
	return lines
}

func sessionArchiveConfirmDialog(c *ui.Context, m Model) {
	if m.ArchiveConfirm == nil || m.ArchiveConfirm.SessionID == "" {
		return
	}
	s := *m.ArchiveConfirm
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	danger := chrome.Color(colors.Danger)
	if danger.A == 0 {
		danger = ui.RGB(242, 90, 90)
	}
	title := strings.TrimSpace(s.Title)
	if title == "" {
		title = i18n.T(i18n.KeyWSSessionUntitled)
	}
	open := true
	ui.DialogBase(c.Key("session-archive-confirm-modal"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogReveal(c, backdrop, panel)
		chrome.FrostSurface(panel.Width(420).MaxWidthPercent(94).Padding(24).Gap(16), colors, m.SystemDark, chrome.Color(colors.Field), 24).Border(0.5, border).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyWSArchiveConfirmTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg)
			ui.Text(c, i18n.WSf(i18n.KeyWSArchiveConfirmDesc, "title", title)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
			lines := FormatArchiveConfirmActivityLines(s)
			if len(lines) > 0 {
				ui.Column(c).FillWidth().Gap(6).Padding(10, 12).Radius(12).Background(chrome.FrostInset(colors, m.SystemDark, chrome.Color(colors.BG))).Border(0.5, border).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyWSArchiveConfirmActivity)).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(fg)
					for _, line := range lines {
						ui.Text(c, line).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
					}
				})
			}
			if s.Error != "" {
				ui.Text(c, s.Error).FontSize(theme.InterfaceSize(12)).TextColor(danger)
			}
			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				cancelLabel := workspaceText("取消", "Cancel")
				cancelBtn := ui.Button(c, cancelLabel).Label(cancelLabel).Height(34).Padding(0, 14).Radius(10).Disabled(s.Busy)
				if !s.Busy {
					cancelBtn.Cursor(ui.CursorPointer)
				}
				if cancelBtn.Clicked() && s.OnCancel != nil {
					s.OnCancel()
				}
				actionLabel := i18n.T(i18n.KeyWSArchiveConfirmAction)
				if s.Busy {
					actionLabel = i18n.T(i18n.KeyWSArchiveConfirmPending)
				}
				confirmBtn := ui.Button(c, actionLabel).Label(actionLabel).Height(34).Padding(0, 14).Radius(10).TextColor(danger).Border(0.5, border).Background(ui.Transparent).Disabled(s.Busy)
				if !s.Busy {
					confirmBtn.Cursor(ui.CursorPointer)
				}
				if confirmBtn.Clicked() && s.OnConfirm != nil {
					s.OnConfirm(s.SessionID)
				}
			})
		})
	})
	if !open && !s.Busy && s.OnCancel != nil {
		s.OnCancel()
	}
}

func sessionDeleteConfirmDialog(c *ui.Context, m Model) {
	if m.DeleteConfirm == nil || m.DeleteConfirm.SessionID == "" {
		return
	}
	s := *m.DeleteConfirm
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	danger := chrome.Color(colors.Danger)
	if danger.A == 0 {
		danger = ui.RGB(242, 90, 90)
	}
	title := strings.TrimSpace(s.Title)
	if title == "" {
		title = i18n.T(i18n.KeyWSSessionUntitled)
	}
	open := true
	ui.DialogBase(c.Key("session-delete-confirm-modal"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogReveal(c, backdrop, panel)
		chrome.FrostSurface(panel.Width(400).MaxWidthPercent(94).Padding(24).Gap(16), colors, m.SystemDark, chrome.Color(colors.Field), 24).Border(0.5, border).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyWSDeleteSessionTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg)
			ui.Text(c, i18n.WSf(i18n.KeyWSDeleteSessionDesc, "name", title)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
			errMsg := strings.TrimSpace(s.Error)
			if s.ErrorCode != "" {
				errMsg = FormatSessionDeleteError(s.ErrorCode)
			}
			if errMsg != "" {
				ui.Text(c, errMsg).FontSize(theme.InterfaceSize(12)).TextColor(danger)
			}
			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				cancelLabel := workspaceText("取消", "Cancel")
				cancelBtn := ui.Button(c, cancelLabel).Label(cancelLabel).Height(34).Padding(0, 14).Radius(10).Disabled(s.Busy)
				if !s.Busy {
					cancelBtn.Cursor(ui.CursorPointer)
				}
				if cancelBtn.Clicked() && s.OnCancel != nil {
					s.OnCancel()
				}
				actionLabel := i18n.T(i18n.KeyWSMenuDeleteSession)
				if s.Busy {
					actionLabel = i18n.T(i18n.KeyWSDeleteSessionPending)
				}
				confirmLabel := workspaceText("确认删除会话", "Confirm delete session")
				deleteBtn := ui.Button(c, actionLabel).Label(confirmLabel).Height(34).Padding(0, 14).Radius(10).TextColor(danger).Border(0.5, border).Background(ui.Transparent).Disabled(s.Busy)
				if !s.Busy {
					deleteBtn.Cursor(ui.CursorPointer)
				}
				if deleteBtn.Clicked() && s.OnConfirm != nil {
					s.OnConfirm(s.SessionID)
				}
			})
		})
	})
	if !open && !s.Busy && s.OnCancel != nil {
		s.OnCancel()
	}
}
