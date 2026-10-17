package chat

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TrajectoryModel drives the ConversationRoot「轨迹」tab (Electron ui-trajectory).
type TrajectoryModel struct {
	Snapshot      session.ViewState
	ThemeID       string
	SystemDark    bool
	List          *ui.ListState
	OnLoadEarlier func()
	// SearchQuery filters ledger rows (toolbar search); nil → package fallback.
	SearchQuery *string
	// SelectedID is the focused ledger row for the event-details pane.
	SelectedID *string
	// ActualDuration toggles timeline equal-width vs proportional blocks.
	ActualDuration *bool
	// TurnsCollapsed folds assistant/tool detail under each user turn.
	TurnsCollapsed *bool
	// CallsCollapsed folds tool-call bodies.
	CallsCollapsed *bool
	// DetailTab is overview|args|result|timing for the inspector.
	DetailTab *string
	// OnCopyDetail copies the active inspector text (Electron event-details copy).
	OnCopyDetail func(text string)
}

var (
	trajSearchFB   string
	trajSelectedFB string
	trajActualDur  = true
	trajTurnsFold  bool
	trajCallsFold  bool
	trajDetailTab  = "overview"
)

// TrajectoryView paints toolbar + timeline + ledger + event details.
func TrajectoryView(c *ui.Context, m TrajectoryModel) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)
	hover := chrome.Color(colors.Hover)
	if hover.A == 0 {
		hover = chrome.Color("rgba(0, 0, 0, 0.04)")
	}

	search := &trajSearchFB
	if m.SearchQuery != nil {
		search = m.SearchQuery
	}
	selected := &trajSelectedFB
	if m.SelectedID != nil {
		selected = m.SelectedID
	}
	actualDur := &trajActualDur
	if m.ActualDuration != nil {
		actualDur = m.ActualDuration
	}
	turnsFold := &trajTurnsFold
	if m.TurnsCollapsed != nil {
		turnsFold = m.TurnsCollapsed
	}
	callsFold := &trajCallsFold
	if m.CallsCollapsed != nil {
		callsFold = m.CallsCollapsed
	}
	detailTab := &trajDetailTab
	if m.DetailTab != nil {
		detailTab = m.DetailTab
	}

	rows := buildTrajectoryRows(m.Snapshot.Messages, *search, *turnsFold, *callsFold)

	ui.Column(c).Fill().MinHeight(0).Children(func() {
		// TrajectoryToolbar — sticky 32px strip.
		ui.Row(c).FillWidth().Height(32).Padding(0, 6, 0, 6).Gap(8).AlignItems(ui.Center).Children(func() {
			paintTrajToggle(c, i18n.T(i18n.KeyTrajDuration), *actualDur, muted, hover, fg, func() {
				*actualDur = !*actualDur
			})
			paintTrajAction(c, i18n.T(i18n.KeyTrajTurns), *turnsFold, muted, hover, fg, func() {
				*turnsFold = !*turnsFold
			})
			paintTrajAction(c, i18n.T(i18n.KeyTrajCalls), *callsFold, muted, hover, fg, func() {
				*callsFold = !*callsFold
			})
			ui.Box(c).Grow(1).MinWidth(0)
			ui.Row(c).Width(160).Height(22).Background(field).Border(1, border).Radius(6).
				Padding(0, 8, 0, 8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, search).FillWidth().Placeholder(i18n.T(i18n.KeyTrajSearch)).FontSize(theme.InterfaceSize(12))
			})
		})
		ui.Box(c).FillWidth().Height(1).Background(border)

		// TrajectoryTimeline — overview strip from row kinds.
		ui.Column(c).FillWidth().Padding(6, 8, 4, 8).Gap(4).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyTrajTimeline)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			if *actualDur {
				paintRecordedTrajectoryTimeline(c, rows, selected, accent, muted, border)
			} else {
				ui.Row(c).FillWidth().Height(18).Gap(2).AlignItems(ui.Center).Children(func() {
					for _, row := range rows {
						if ui.Button(c, "").Grow(1).Height(10).Background(trajKindColor(row.Kind, accent, muted, border)).BorderWidth(0).Radius(2).Clicked() {
							*selected = row.ID
						}
					}
				})
			}
		})
		ui.Box(c).FillWidth().Height(1).Background(border)

		// Ledger + event details split.
		ui.Row(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
				// Column headers (Electron: 输入 / 输出 / 思考 / 时间 / 模型 / 工具).
				ui.Row(c).FillWidth().Padding(4, 8, 4, 8).Gap(8).Children(func() {
					for _, h := range []struct {
						label string
						grow  float32
					}{
						{i18n.T(i18n.KeyTrajColInput), 2},
						{i18n.T(i18n.KeyTrajColOutput), 2},
						{i18n.T(i18n.KeyTrajColThink), 1},
						{i18n.T(i18n.KeyTrajColTime), 1},
						{i18n.T(i18n.KeyTrajColModel), 1},
						{i18n.T(i18n.KeyTrajColTools), 1},
					} {
						ui.Text(c, h.label).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(h.grow)
					}
				})
				ui.Box(c).FillWidth().Height(1).Background(border)

				if m.Snapshot.HistoryHasMore && m.OnLoadEarlier != nil {
					label := i18n.T(i18n.KeyTrajLoadEarlier)
					if m.Snapshot.HistoryPageBusy {
						label = i18n.T(i18n.KeyTrajLoading)
					}
					ui.Row(c).FillWidth().Padding(4, 8, 4, 8).Children(func() {
						if ui.Button(c, label).Disabled(m.Snapshot.HistoryPageBusy).
							FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(field).FillWidth().Clicked() {
							m.OnLoadEarlier()
						}
					})
				}

				list := m.List
				if list == nil {
					list = &ui.ListState{}
				}
				if len(rows) == 0 {
					ui.Column(c).Fill().Padding(24, 16, 16, 16).Gap(6).Children(func() {
						if m.Snapshot.HistoryPageBusy {
							ui.Text(c, i18n.T(i18n.KeyTrajLoading)).FontSize(theme.InterfaceSize(13)).TextColor(muted)
						} else {
							ui.Text(c, i18n.T(i18n.KeyTrajEmpty)).FontSize(theme.InterfaceSize(13)).TextColor(muted)
						}
						ui.Box(c).Grow(1)
					})
					return
				}
				ui.List(c, list, len(rows)).Fill().MinHeight(0).Rows(func(lr ui.ListRow) {
					row := rows[lr.Index]
					bg := ui.Transparent
					if *selected == row.ID {
						bg = hover
					}
					btn := ui.Button(lr.Context, "").FillWidth().Background(bg).BorderWidth(0).
						Padding(6, 8, 6, 8).Radius(0)
					btn.Children(func() {
						ui.Row(lr.Context).FillWidth().Gap(8).AlignItems(ui.Start).Children(func() {
							ui.Text(lr.Context, row.Input).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(2)
							ui.Text(lr.Context, row.Output).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(2)
							ui.Text(lr.Context, row.Think).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1)
							ui.Text(lr.Context, row.Time).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1)
							ui.Text(lr.Context, row.Model).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1)
							ui.Text(lr.Context, row.Tools).FontSize(theme.InterfaceSize(11)).TextColor(accent).Grow(1)
						})
					})
					if btn.Clicked() {
						*selected = row.ID
					}
				})
			})

			// Event details inspector (Electron complementary pane).
			ui.Box(c).Width(1).FillHeight().Background(border)
			ui.Column(c).Width(240).FillHeight().Padding(8, 10, 8, 10).Gap(6).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyTrajDetailTitle)).FontSize(theme.InterfaceSize(12)).FontWeight(600).TextColor(fg).Grow(1)
				})
				detail := findTrajectoryRow(rows, *selected)
				if detail == nil {
					ui.Text(c, i18n.T(i18n.KeyTrajDetailHint)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					return
				}
				fullDetail := trajectoryDetailRow(*detail)
				detail = &fullDetail
				ui.Row(c).Gap(6).Children(func() {
					for _, tab := range []struct{ id, label string }{
						{"overview", i18n.T(i18n.KeyTrajTabOverview)},
						{"args", i18n.T(i18n.KeyTrajTabArgs)},
						{"result", i18n.T(i18n.KeyTrajTabResult)},
						{"timing", i18n.T(i18n.KeyTrajTabTiming)},
					} {
						t := tab
						ink := muted
						weight := 400
						if *detailTab == t.id {
							ink = accent
							weight = 600
						}
						if ui.Button(c, t.label).FontSize(theme.InterfaceSize(11)).FontWeight(weight).BorderWidth(0).
							Background(ui.Transparent).TextColor(ink).Clicked() {
							*detailTab = t.id
						}
					}
				})
				ui.Box(c).FillWidth().Height(1).Background(border)
				copyBody := ""
				ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
					switch *detailTab {
					case "args":
						body := detail.Input
						if body == "" {
							body = i18n.T(i18n.KeyTrajNoArgs)
						}
						if detail.RawHint != "" {
							body = detail.RawHint
						}
						copyBody = body
						ui.Text(c, body).FontSize(theme.InterfaceSize(12)).TextColor(fg)
					case "result":
						body := detail.Output
						if body == "" {
							body = detail.Tools
						}
						if body == "" {
							body = i18n.T(i18n.KeyTrajNoResult)
						}
						copyBody = body
						ui.Text(c, body).FontSize(theme.InterfaceSize(12)).TextColor(fg)
					case "timing":
						copyBody = trajectoryTimingDetail(*detail)
						ui.Text(c, copyBody).FontSize(theme.InterfaceSize(12)).TextColor(fg)
					default: // overview
						ui.Text(c, kindLabelZH(detail.Kind)).FontSize(theme.InterfaceSize(12)).FontWeight(600).TextColor(accent)
						var parts []string
						if detail.Input != "" {
							ui.Text(c, i18n.T(i18n.KeyTrajColInput)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							ui.Text(c, detail.Input).FontSize(theme.InterfaceSize(12)).TextColor(fg)
							parts = append(parts, detail.Input)
						}
						if detail.Output != "" {
							ui.Text(c, i18n.T(i18n.KeyTrajColOutput)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							ui.Text(c, detail.Output).FontSize(theme.InterfaceSize(12)).TextColor(fg)
							parts = append(parts, detail.Output)
						}
						if detail.Think != "" {
							ui.Text(c, i18n.T(i18n.KeyTrajColThink)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							ui.Text(c, detail.Think).FontSize(theme.InterfaceSize(12)).TextColor(fg)
							parts = append(parts, detail.Think)
						}
						if detail.Tools != "" {
							ui.Text(c, i18n.T(i18n.KeyTrajColTools)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							ui.Text(c, detail.Tools).FontSize(theme.InterfaceSize(12)).TextColor(fg)
							parts = append(parts, detail.Tools)
						}
						copyBody = strings.Join(parts, "\n")
					}
				})
				if m.OnCopyDetail != nil && strings.TrimSpace(copyBody) != "" {
					if ui.Button(c, i18n.T(i18n.KeyTrajCopy)).FontSize(theme.InterfaceSize(11)).Clicked() {
						m.OnCopyDetail(copyBody)
					}
				}
			})
		})
	})
}

func paintTrajToggle(c *ui.Context, label string, on bool, muted, hover, fg ui.Color, click func()) {
	bg := ui.Transparent
	ink := muted
	if on {
		bg = hover
		ink = fg
	}
	if ui.Button(c, label).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(bg).
		TextColor(ink).Padding(2, 7, 2, 7).Radius(4).Clicked() {
		click()
	}
}

func paintTrajAction(c *ui.Context, label string, pressed bool, muted, hover, fg ui.Color, click func()) {
	icon := "⊟"
	if pressed {
		icon = "⊞"
	}
	bg := ui.Transparent
	ink := muted
	if pressed {
		bg = hover
		ink = fg
	}
	if ui.Button(c, icon+" "+label).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(bg).
		TextColor(ink).Padding(2, 7, 2, 7).Radius(4).Clicked() {
		click()
	}
}

type trajectoryContent struct {
	Input, Output, Think, Tools string
	Raw                         string
}

type trajRow struct {
	Full      trajectoryContent
	Timing    *session.TrajectoryTiming
	EventTime float64
	ID        string
	Kind      session.MessageKind
	Role      string
	Input     string
	Output    string
	Think     string
	Time      string
	Model     string
	Tools     string
	RawHint   string
}

func buildTrajectoryRows(msgs []session.Message, query string, turnsFold, callsFold bool) []trajRow {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]trajRow, 0, len(msgs))
	turn := 0
	for _, msg := range msgs {
		if turnsFold && msg.Role == "assistant" && msg.Kind != session.MessageKindText {
			continue
		}
		if callsFold && msg.Kind == session.MessageKindTool {
			continue
		}
		row := messageToTrajRow(msg)
		// Electron ledger groups USER turns; annotate turn index on user rows.
		if strings.EqualFold(msg.Role, "user") && msg.Kind == session.MessageKindText {
			turn++
			row.Model = fmt.Sprintf(i18n.T(i18n.KeyTrajTurnN), turn)
		}
		if q != "" {
			hay := strings.ToLower(row.Full.Input + " " + row.Full.Output + " " + row.Full.Think + " " + row.Full.Tools + " " + row.Model + " " + row.RawHint + " " + row.Full.Raw)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, row)
	}
	return out
}

func messageToTrajRow(msg session.Message) trajRow {
	id := msg.ID
	if id == "" {
		id = fmt.Sprintf("seq-%.0f", msg.Seq)
	}
	body := msg.Body
	row := trajRow{
		ID:        id,
		Kind:      msg.Kind,
		Role:      msg.Role,
		Model:     msg.Model,
		Timing:    msg.TrajectoryTime,
		EventTime: msg.EventTime,
	}
	switch msg.Kind {
	case session.MessageKindTool:
		row.Tools = body
		if body == "" {
			row.Tools = msg.EventType
		}
		switch msg.EventType {
		case "tool/result":
			row.Output = body
			if msg.Args != "" {
				row.RawHint = msg.Args
				row.Output = msg.Args
			}
			row.Input = i18n.T(i18n.KeyTrajToolResult)
		default:
			row.Input = i18n.T(i18n.KeyTrajToolCallOnly)
			if msg.Args != "" {
				row.RawHint = msg.Args
				row.Input = msg.Args
			}
		}
	case session.MessageKindThinking:
		row.Think = body
		row.Input = kindLabelZH(msg.Kind)
	case session.MessageKindCompaction:
		row.Input = i18n.T(i18n.KeyTrajCompacted)
		row.Output = body
	case session.MessageKindDeliverables:
		row.Output = body
		if len(msg.Files) > 0 {
			row.Output = strings.Join(msg.Files, ", ")
		}
		row.Input = i18n.T(i18n.KeyTrajDeliverables)
	default:
		switch strings.ToLower(msg.Role) {
		case "user":
			row.Input = body
			row.Tools = i18n.T(i18n.KeyTrajRoleUser)
		case "assistant":
			row.Output = body
			row.Tools = i18n.T(i18n.KeyTrajRoleAssistant)
		case "system":
			row.Input = i18n.T(i18n.KeyTrajRoleSystem)
			row.Output = body
			row.Tools = i18n.T(i18n.KeyTrajRoleSystem)
		default:
			row.Input = body
		}
	}
	row.Time = trajectoryDurationLabel(msg.TrajectoryTime, msg.Streaming)
	if row.RawHint == "" && msg.Args != "" {
		row.RawHint = msg.Args
	}
	if row.RawHint == "" && len(msg.Raw) > 0 {
		row.RawHint = string(msg.Raw)
	}
	row.Full = trajectoryContent{Input: row.Input, Output: row.Output, Think: row.Think, Tools: row.Tools, Raw: string(msg.Raw)}
	row.Input = trajectoryPreview(row.Input)
	row.Output = trajectoryPreview(row.Output)
	row.Think = trajectoryPreview(row.Think)
	row.Tools = trajectoryPreview(row.Tools)
	return row
}

func trajectoryPreview(text string) string {
	return clipRunes(strings.ReplaceAll(strings.TrimSpace(text), "\n", " "), 80)
}

// Only ledger cells are abbreviated. The inspector and clipboard share the
// complete projection, while RawHint retains the complete args/event fallback.
func trajectoryDetailRow(row trajRow) trajRow {
	row.Input, row.Output = row.Full.Input, row.Full.Output
	row.Think, row.Tools = row.Full.Think, row.Full.Tools
	return row
}

func formatEventTime(ms float64) string {
	// Harness event time is unix milliseconds.
	sec := int64(ms / 1000)
	if sec <= 0 {
		return ""
	}
	return time.UnixMilli(int64(ms)).Local().Format("15:04:05.000")
}

func findTrajectoryRow(rows []trajRow, id string) *trajRow {
	if id == "" {
		return nil
	}
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

func kindLabelZH(k session.MessageKind) string {
	switch k {
	case session.MessageKindTool:
		return i18n.T(i18n.KeyTrajKindTool)
	case session.MessageKindThinking:
		return i18n.T(i18n.KeyTrajKindThink)
	case session.MessageKindCompaction:
		return i18n.T(i18n.KeyTrajCompacted)
	case session.MessageKindDeliverables:
		return i18n.T(i18n.KeyTrajDeliverables)
	default:
		return i18n.T(i18n.KeyTrajKindMessage)
	}
}

func trajKindColor(k session.MessageKind, accent, muted, border ui.Color) ui.Color {
	switch k {
	case session.MessageKindTool:
		return accent
	case session.MessageKindThinking:
		return muted
	case session.MessageKindCompaction:
		return border
	default:
		return chrome.Color("rgb(148, 163, 184)")
	}
}

func clipRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
