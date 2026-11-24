package sidebar

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
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
	t := tok(c)
	title := strings.TrimSpace(s.Title)
	if title == "" {
		title = i18n.T(i18n.KeyWSSessionUntitled)
	}
	open := true
	lines := FormatArchiveConfirmActivityLines(s)
	var body func()
	if len(lines) > 0 || s.Error != "" {
		body = func() {
			if len(lines) > 0 {
				ui.Column(c).FillWidth().Gap(design.Sp(1.5)).Padding(design.Sp(2.5), design.Sp(3)).Radius(design.RadiusXL).
					Background(t.Muted.Alpha(0.5)).Border(design.BorderWidth, t.Border).Children(func() {
					sbText(c, i18n.T(i18n.KeyWSArchiveConfirmActivity), 12, design.WeightMedium, t.Foreground)
					for _, line := range lines {
						sbText(c, line, 12, design.WeightNormal, t.MutedForeground).FixedLineHeight(fs(18))
					}
				})
			}
			if s.Error != "" {
				sbText(c, s.Error, 12, design.WeightNormal, t.Destructive)
			}
		}
	}
	coss.AlertDialog(c, &open, coss.DialogOptions{
		Title:           i18n.T(i18n.KeyWSArchiveConfirmTitle),
		Description:     i18n.WSf(i18n.KeyWSArchiveConfirmDesc, "title", title),
		MaxWidth:        420,
		DismissDisabled: s.Busy,
		Key:             "session-archive-confirm-modal",
		Footer: func() {
			cancelLabel := workspaceText("取消", "Cancel")
			if coss.Button(c, cancelLabel, coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonMD,
				Disabled: s.Busy,
				Key:      "archive-confirm-cancel",
			}).Label(cancelLabel).Clicked() && s.OnCancel != nil {
				s.OnCancel()
			}
			actionLabel := i18n.T(i18n.KeyWSArchiveConfirmAction)
			if s.Busy {
				actionLabel = i18n.T(i18n.KeyWSArchiveConfirmPending)
			}
			if coss.Button(c, actionLabel, coss.ButtonOptions{
				Variant:  coss.ButtonDestructiveOutline,
				Size:     coss.ButtonMD,
				Disabled: s.Busy,
				Loading:  s.Busy,
				Key:      "archive-confirm-submit",
			}).Label(actionLabel).Clicked() && s.OnConfirm != nil {
				s.OnConfirm(s.SessionID)
			}
		},
	}, body)
	if !open && !s.Busy && s.OnCancel != nil {
		s.OnCancel()
	}
}

func sessionDeleteConfirmDialog(c *ui.Context, m Model) {
	if m.DeleteConfirm == nil || m.DeleteConfirm.SessionID == "" {
		return
	}
	s := *m.DeleteConfirm
	t := tok(c)
	title := strings.TrimSpace(s.Title)
	if title == "" {
		title = i18n.T(i18n.KeyWSSessionUntitled)
	}
	errMsg := strings.TrimSpace(s.Error)
	if s.ErrorCode != "" {
		errMsg = FormatSessionDeleteError(s.ErrorCode)
	}
	var body func()
	if errMsg != "" {
		body = func() {
			sbText(c, errMsg, 12, design.WeightNormal, t.Destructive)
		}
	}
	open := true
	coss.AlertDialog(c, &open, coss.DialogOptions{
		Title:           i18n.T(i18n.KeyWSDeleteSessionTitle),
		Description:     i18n.WSf(i18n.KeyWSDeleteSessionDesc, "name", title),
		MaxWidth:        400,
		DismissDisabled: s.Busy,
		Key:             "session-delete-confirm-modal",
		Footer: func() {
			cancelLabel := workspaceText("取消", "Cancel")
			if coss.Button(c, cancelLabel, coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonMD,
				Disabled: s.Busy,
				Key:      "delete-confirm-cancel",
			}).Label(cancelLabel).Clicked() && s.OnCancel != nil {
				s.OnCancel()
			}
			actionLabel := i18n.T(i18n.KeyWSMenuDeleteSession)
			if s.Busy {
				actionLabel = i18n.T(i18n.KeyWSDeleteSessionPending)
			}
			confirmLabel := workspaceText("确认删除会话", "Confirm delete session")
			if coss.Button(c, actionLabel, coss.ButtonOptions{
				Variant:  coss.ButtonDestructive,
				Size:     coss.ButtonMD,
				Disabled: s.Busy,
				Loading:  s.Busy,
				Key:      "delete-confirm-submit",
			}).Label(confirmLabel).Clicked() && s.OnConfirm != nil {
				s.OnConfirm(s.SessionID)
			}
		},
	}, body)
	if !open && !s.Busy && s.OnCancel != nil {
		s.OnCancel()
	}
}
