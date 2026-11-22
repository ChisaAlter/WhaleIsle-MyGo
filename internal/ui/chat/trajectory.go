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
	OnOpenImage func(string)
	Snapshot    session.ViewState
	// PaneWidth is the host layout width before the first frame. Standalone
	// callers may omit it when the view fills Context.Size.
	PaneWidth     float32
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
	// ActualTime retains idle gaps; nil matches the original hidden switch default (false).
	ActualTime *bool
	// TurnsCollapsed folds assistant/tool detail under each user turn.
	TurnsCollapsed *bool
	// CallsCollapsed folds tool-call bodies.
	CallsCollapsed *bool
	// DetailTab is overview|args|result|timing|options|usage for the inspector.
	DetailTab *string
	// OnCopyDetail copies the active inspector text (Electron event-details copy).
	OnCopyDetail func(text string)
}

var (
	trajSearchFB   string
	trajSelectedFB string
	trajActualDur  = false // original dsh.trajectory.duration store defaults false (sequence view)
	trajTurnsFold  bool
	trajCallsFold  bool
	trajDetailTab  = "overview"
)

// TrajectoryView paints toolbar + timeline + ledger + event details.
func TrajectoryView(c *ui.Context, m TrajectoryModel) {
	renderAttachmentSnapshot(m.Snapshot, func() { trajectoryView(c, m) })
}

func trajectoryView(c *ui.Context, m TrajectoryModel) {
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

	root := ui.Column(c.Key("trajectory:" + m.Snapshot.PlanHistoryBinding + ":" + m.Snapshot.SelectedID)).Fill().MinHeight(0)
	folds := ui.Local(root, "folds", func() trajectoryFoldState { return trajectoryFoldState{} })
	allRows := trajectoryAllRows(trajectoryMessages(m.Snapshot), "")
	ledgerRows := func() []trajRow { return trajectoryLedgerRows(folds.rows(allRows, *search, *turnsFold, *callsFold)) }
	rows := ledgerRows()
	resize := ui.Local(root, "detail-width", func() trajectoryResize { return trajectoryResize{} })
	revealed := ui.Local(root, "revealed-row", func() string { return "" })
	timeRange := ui.Local(root, "time-range", func() trajectoryRangeState { return trajectoryRangeState{} })
	actualTime := m.ActualTime != nil && *m.ActualTime
	timeRange.setProjection(!*actualDur && !actualTime, *actualDur && !actualTime, !*actualDur && actualTime)
	rangeMatches := timeRange.matches(allRows)
	if timeRange.active && *selected != *revealed && !rangeMatches[*selected] {
		timeRange.clear()
		rangeMatches = nil
	}
	if *selected != "" && *revealed != *selected {
		folds.reveal(allRows, *selected)
		rows = ledgerRows()
	}
	if m.List != nil && *selected != "" && *revealed != *selected {
		for index, row := range rows {
			if row.ID == *selected {
				m.List.FollowEnd = false
				m.List.ScrollTo(index, ui.Center)
				*revealed = *selected
				break
			}
		}
	}
	root.Children(func() {
		// TrajectoryToolbar .inner — wrap flex, padding 4px 6px, col-gap 8,
		// row-gap 4 (TrajectoryToolbar.module.css). Controls are ~20 DIP
		// transparent buttons; the search field is a single compact
		// 0 1 164px slot with a leading 11px search icon.
		palette := newTrajectoryPalette(m.ThemeID, m.SystemDark, colors)
		ui.Row(c).Label("Trajectory toolbar").FillWidth().Wrap().Padding(4, 6, 4, 6).Gap(8).AlignItems(ui.Center).Children(func() {
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
			ui.Row(c).MaxWidth(164).MinWidth(84).Height(22).Grow(0).Shrink(1).Background(chrome.FrostInset(colors, m.SystemDark, palette.layer2)).Border(1, chrome.Color(colors.BorderL4)).Radius(6).
				Padding(0, 6, 0, 6).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, trajIconSearch).Width(11).Height(11).TextColor(muted)
				ui.TextInput(c, search).FillWidth().MinWidth(0).BorderWidth(0).Background(ui.Transparent).Placeholder(i18n.T(i18n.KeyTrajSearch)).FontSize(theme.InterfaceSize(12))
			})
		})
		ui.Box(c).FillWidth().Height(1).Background(border)

		// TrajectoryTimeline — three-lane compact chart with the original's
		// drag-range focus overlay (TrajectoryTimeline.tsx renders drag
		// ranges, click-sized focus, and Escape reset on the plot itself).
		ui.Column(c).Label("Trajectory timeline section").FillWidth().Padding(0, 6, 2, 6).Children(func() {
			paintTrajectoryLanes(c, allRows, selected, timeRange, m.List, rows, palette)
		})
		ui.Box(c).FillWidth().Height(1).Background(border)

		// Ledger + event details split.
		ui.Row(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).Label("Trajectory ledger").Grow(1).FillHeight().MinWidth(0).MinHeight(0).Children(func() {
				if m.Snapshot.HistoryHasMore && m.OnLoadEarlier != nil {
					label := i18n.T(i18n.KeyTrajLoadEarlier)
					if m.Snapshot.HistoryPageBusy {
						label = i18n.T(i18n.KeyTrajLoading)
					}
					ui.Row(c).FillWidth().Padding(4, 8, 4, 8).Children(func() {
						if ui.Button(c, label).Cursor(ui.CursorPointer).Disabled(m.Snapshot.HistoryPageBusy).
							FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(chrome.FrostInset(colors, m.SystemDark, field)).FillWidth().Clicked() {
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
					turnStart := lr.Index == 0 || rows[lr.Index-1].TurnKey != row.TurnKey
					bg := ui.Transparent
					if rangeMatches != nil && rangeMatches[row.ID] {
						bg = chrome.FrostInset(colors, m.SystemDark, field)
					}
					if *selected == row.ID {
						bg = hover
					}
					ui.Column(lr.Context).FillWidth().MinWidth(0).Children(func() {
						btn := ui.Button(lr.Context, "").Label("Trajectory row " + row.ID).FillWidth().Height(30).Background(bg).BorderWidth(0).
							Padding(0).Radius(0).Cursor(ui.CursorPointer)
						btn.Children(func() {
							ui.Row(lr.Context).FillWidth().Gap(0).Children(func() {
								// Fixed-width disclosure slot on every row so
								// badge and title keep one x across foldable
								// and plain rows.
								ui.Box(lr.Context).Width(20).FillHeight().Children(func() {
									if row.FoldKey != "" {
										glyph := "▾"
										if row.Folded {
											glyph = "▸"
										}
										if ui.Button(lr.Context, glyph).Label("Fold trajectory " + row.ID).Width(20).Height(28).Padding(0).BorderWidth(0).Background(ui.Transparent).Cursor(ui.CursorPointer).Clicked() {
											folds.toggle(row)
										}
									}
								})
								ui.Column(lr.Context).Grow(1).MinWidth(0).Children(func() { paintTrajectoryRecord(lr.Context, row, palette, turnStart) })
							})
						})
						if btn.Clicked() {
							if timeRange.active && !rangeMatches[row.ID] {
								timeRange.clear()
							}
							*selected = row.ID
						}
						if lr.Index < len(rows)-1 {
							ui.Box(lr.Context).FillWidth().Height(1).Background(border)
						}
					})
				})
			})

			// The original local inspector exists only for a selected record.
			detail := findTrajectoryRow(rows, *selected)
			if detail == nil {
				detail = findTrajectoryRow(allRows, *selected)
			}
			if detail == nil {
				return
			}
			paneWidth := m.PaneWidth
			if paneWidth <= 0 {
				paneWidth, _ = c.Size()
			}
			width := trajectoryInspectorWidth(paneWidth)
			if resize.width > 0 {
				low, high := trajectoryResizeBounds(paneWidth)
				width = min(max(low, resize.width), high)
			}
			viewportWidth, _ := c.Size()
			if viewportWidth > 760 {
				paintTrajectorySplitter(c, resize, paneWidth, width, border)
			}
			inspector := chrome.FrostSurface(ui.Column(c).Key("trajectory-inspector").Label("Trajectory event inspector").Width(width).FillHeight().MinWidth(0), colors, m.SystemDark, chrome.Color(colors.BG), 0).Padding(8, 10, 8, 10).Gap(6)
			if viewportWidth <= 760 {
				inspector.Absolute().Right(0).Top(0).Bottom(0).Width(min(paneWidth*.92, 420))
			}
			inspector.Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyTrajDetailTitle)).FontSize(theme.InterfaceSize(12)).FontWeight(600).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
					if ui.Button(c, "×").Label("Close trajectory details").Width(24).Height(24).Padding(0).BorderWidth(0).Background(ui.Transparent).Cursor(ui.CursorPointer).Clicked() {
						*selected = ""
					}
				})
				fullDetail := trajectoryDetailRow(*detail)
				detail = &fullDetail
				schemaPayload, headerPayload := trajectoryRequestPayloads(m.Snapshot, detail.ID)
				requestPayload, requestEvents := trajectoryRequestBodies(m.Snapshot, detail.ID)
				programAvailable := trajectoryProgramAvailable(detail.Source)
				historyAvailable := detail.Source.Kind == session.MessageKindSystemPrompt || headerPayload != ""
				if *detailTab == "program" && !programAvailable || *detailTab == "history-diff" && !historyAvailable || *detailTab == "schema" && schemaPayload == "" || *detailTab == "request-header" && headerPayload == "" || (*detailTab == "request" || *detailTab == "request-events" || *detailTab == "usage") && requestPayload == "" {
					*detailTab = "overview"
				}
				ui.ScrollHorizontal(c).FillWidth().Height(28).Children(func() {
					ui.Row(c).Gap(6).Children(func() {
						for _, tab := range []struct{ id, label string }{
							{"overview", i18n.T(i18n.KeyTrajTabOverview)},
							{"args", i18n.T(i18n.KeyTrajTabArgs)},
							{"result", i18n.T(i18n.KeyTrajTabResult)},
							{"timing", i18n.T(i18n.KeyTrajTabTiming)},
							{"options", TrajectoryLocale("tab.options")},
							{"raw", trajectoryText("原文", "Raw")},
							{"source", trajectoryText("来源", "Source")},
							{"program", trajectoryText("程序", "Program")},
							{"history-diff", trajectoryText("历史差异", "History diff")},
							{"usage", TrajectoryLocale("tab.usage")},
						} {
							if tab.id == "program" {
								if !programAvailable {
									continue
								}
							}
							if tab.id == "history-diff" && !historyAvailable {
								continue
							}
							if tab.id == "usage" && requestPayload == "" {
								continue
							}
							t := tab
							ink := muted
							weight := 400
							if *detailTab == t.id {
								ink = accent
								weight = 600
							}
							if ui.Button(c, t.label).FontSize(theme.InterfaceSize(11)).FontWeight(weight).BorderWidth(0).
								Background(ui.Transparent).TextColor(ink).Cursor(ui.CursorPointer).Clicked() {
								*detailTab = t.id
							}
						}
					})
				})
				if schemaPayload != "" || headerPayload != "" {
					ui.ScrollHorizontal(c).Label("Trajectory request tabs").FillWidth().Height(28).Children(func() {
						paintTrajectoryRequestTabs(c, detailTab, schemaPayload, headerPayload, accent, muted)
					})
				}
				if requestPayload != "" {
					ui.ScrollHorizontal(c).Label("Trajectory lifecycle tabs").FillWidth().Height(28).Children(func() {
						paintTrajectoryLifecycleTabs(c, detailTab, true, accent, muted)
					})
				}
				ui.Box(c).FillWidth().Height(1).Background(border)
				copyBody := ""
				ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
					if handled, body := trajectoryNativeDetail(c, m, *detail, allRows, *detailTab, selected, fg, muted, field, accent); handled {
						copyBody = body
						return
					}
					switch *detailTab {
					case "request", "request-events":
						copyBody = requestPayload
						if *detailTab == "request-events" {
							copyBody = requestEvents
						}
						trajectoryPayloadView(c, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID+":"+detail.ID+":"+*detailTab, copyBody, fg, muted)
					case "schema", "request-header":
						copyBody = schemaPayload
						if *detailTab == "request-header" {
							copyBody = headerPayload
						}
						if copyBody == "" {
							ui.Text(c, trajectoryText("历史记录未提供此数据", "Not available in loaded history")).TextColor(muted)
						} else {
							trajectoryPayloadView(c, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID+":"+detail.ID+":"+*detailTab, copyBody, fg, muted)
						}
					case "args":
						body := detail.Input
						if body == "" {
							body = i18n.T(i18n.KeyTrajNoArgs)
						}
						if detail.RawHint != "" {
							body = detail.RawHint
						}
						copyBody = body
						trajectoryPayloadView(c, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID+":"+detail.ID+":"+*detailTab, body, fg, muted)
					case "result":
						body := detail.Output
						if body == "" {
							body = detail.Tools
						}
						if body == "" {
							body = i18n.T(i18n.KeyTrajNoResult)
						}
						copyBody = body
						trajectoryPayloadView(c, m.Snapshot.PlanHistoryBinding+":"+m.Snapshot.SelectedID+":"+detail.ID+":"+*detailTab, body, fg, muted)
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
					if ui.Button(c, i18n.T(i18n.KeyTrajCopy)).FontSize(theme.InterfaceSize(11)).Cursor(ui.CursorPointer).Clicked() {
						m.OnCopyDetail(copyBody)
					}
				}
			})
		})
	})
}

// paintTrajToggle renders the original .toggle control: a 20 DIP transparent
// button with an 11px clock icon and xxs-12 tertiary label; the pressed state
// fills with the interactive hover wash (TrajectoryToolbar.module.css
// .toggle/.toggle[data-pressed]).
func paintTrajToggle(c *ui.Context, label string, on bool, muted, hover, fg ui.Color, click func()) {
	bg := ui.Transparent
	ink := muted
	if on {
		bg = hover
		ink = fg
	}
	btn := ui.Button(c, "").Label(label).Height(20).BorderWidth(0).Background(bg).
		Padding(0, 4, 0, 4).Radius(4).Cursor(ui.CursorPointer)
	btn.Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, trajIconClock).Width(11).Height(11).TextColor(ink)
			ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(ink).SingleLine()
		})
	})
	if btn.Clicked() {
		click()
	}
}

// paintTrajAction renders the original .action control — the same 20 DIP
// recipe with the ⊟/⊞ fold glyph the source paints as a plain text span.
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
	btn := ui.Button(c, "").Label(label).Height(20).BorderWidth(0).Background(bg).
		Padding(0, 4, 0, 4).Radius(4).Cursor(ui.CursorPointer)
	btn.Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Text(c, icon).FontSize(theme.InterfaceSize(12)).TextColor(ink).SingleLine()
			ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(ink).SingleLine()
		})
	})
	if btn.Clicked() {
		click()
	}
}

type trajectoryContent struct {
	Input, Output, Think, Tools string
	Raw                         string
}

type trajRow struct {
	Source    session.Message
	TurnKey   string
	FoldKey   string
	Folded    bool
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
	// Result is the inline `→ result` continuation TrajectoryCell paints on
	// a tool call row after its tool/result row is merged in.
	Result string
	// TurnNumber is the 1-based gutter marker (#1, #2 …) on the row that
	// opens a turn; zero means "inside a turn" or "no turn".
	TurnNumber int
}

func trajectoryAllRows(msgs []session.Message, query string) []trajRow {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]trajRow, 0, len(msgs))
	groups, numbers := trajectoryTurnGroups(msgs)
	seenTurns := map[string]bool{}
	for index, msg := range msgs {
		row := messageToTrajRow(msg)
		row.TurnKey = groups[index]
		if msg.Role == "user" && msg.Kind == session.MessageKindText && !seenTurns[row.TurnKey] {
			seenTurns[row.TurnKey] = true
			number, ok := numbers[row.TurnKey]
			if !ok {
				number = float64(len(seenTurns))
			}
			row.Model = fmt.Sprintf(i18n.T(i18n.KeyTrajTurnN), int(number))
			row.TurnNumber = int(number)
		}
		out = append(out, row)
	}
	// The original renders records before the first user turn as a preamble
	// section — no turn rail and no #n marker (TrajectoryTable between-turns).
	for i := range out {
		if out[i].Role == "user" && out[i].Kind == session.MessageKindText {
			break
		}
		out[i].TurnKey = ""
	}
	out = mergeTrajectoryToolResults(out)
	if q != "" {
		filtered := out[:0]
		for _, row := range out {
			hay := strings.ToLower(row.Full.Input + " " + row.Full.Output + " " + row.Full.Think + " " + row.Full.Tools + " " + row.Model + " " + row.RawHint + " " + row.Result + " " + row.Full.Raw)
			if strings.Contains(hay, q) {
				filtered = append(filtered, row)
			}
		}
		out = filtered
	}
	return out
}

// trajectoryLedgerVisible reports whether a record renders as a ledger row.
// Request boundaries are zero-height separators, reasoning folds into the
// assistant record, and auxiliary status/event-typed records (turn/end,
// session/title, step markers) are not rows.
func trajectoryLedgerVisible(row trajRow) bool {
	return row.Kind != trajectoryRequestKind && row.Kind != session.MessageKindThinking &&
		row.Role != "status" && !strings.Contains(row.Role, "/")
}

func trajectoryLedgerRows(rows []trajRow) []trajRow {
	out := make([]trajRow, 0, len(rows))
	for _, row := range rows {
		if trajectoryLedgerVisible(row) {
			out = append(out, row)
		}
	}
	return out
}

// mergeTrajectoryToolResults folds each tool/result row into its tool/call
// row so the ledger renders the original's single `name {args} → result`
// cell (TrajectoryCell.tsx). Pairing prefers ToolCallView.CallID; a result
// with no resolvable CallID merges into the most recent unmatched call.
func mergeTrajectoryToolResults(rows []trajRow) []trajRow {
	callByID := map[string]int{}
	for i, row := range rows {
		if row.Kind == session.MessageKindTool && row.Source.EventType != "tool/result" && row.Source.Tool != nil && row.Source.Tool.CallID != "" {
			callByID[row.Source.Tool.CallID] = i
		}
	}
	resultOf := map[int]string{}
	consumedCall := map[int]bool{}
	drop := map[int]bool{}
	for i, row := range rows {
		if row.Kind != session.MessageKindTool || row.Source.EventType != "tool/result" {
			continue
		}
		target := -1
		if row.Source.Tool != nil && row.Source.Tool.CallID != "" {
			if j, ok := callByID[row.Source.Tool.CallID]; ok && !consumedCall[j] {
				target = j
			}
		}
		if target < 0 {
			for j := i - 1; j >= 0; j-- {
				if rows[j].Kind != session.MessageKindTool || rows[j].Source.EventType == "tool/result" {
					break
				}
				if !consumedCall[j] {
					target = j
				}
				break
			}
		}
		if target < 0 {
			continue
		}
		preview := row.Full.Output
		if preview == "" {
			preview = row.Output
		}
		resultOf[target] = trajectoryPreview(strings.Join(strings.Fields(preview), " "))
		consumedCall[target] = true
		drop[i] = true
	}
	merged := make([]trajRow, 0, len(rows)-len(drop))
	for i, row := range rows {
		if drop[i] {
			continue
		}
		if result, ok := resultOf[i]; ok {
			row.Result = result
		}
		merged = append(merged, row)
	}
	return merged
}

func messageToTrajRow(msg session.Message) trajRow {
	id := msg.ID
	if id == "" {
		id = fmt.Sprintf("seq-%.0f", msg.Seq)
	}
	body := msg.Body
	row := trajRow{
		Source:    msg,
		ID:        id,
		Kind:      msg.Kind,
		Role:      msg.Role,
		Model:     msg.Model,
		Timing:    msg.TrajectoryTime,
		EventTime: msg.EventTime,
	}
	switch msg.Kind {
	case trajectoryRequestKind:
		row.Input = body
		row.Output = msg.Args
		row.RawHint = msg.Args
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
			if content := session.UserMessageContent(msg); content.Structured {
				parts := []string{content.Text}
				for _, attachment := range content.Attachments {
					parts = append(parts, attachment.Type+": "+attachment.Name+" ("+attachment.AttachmentID+")")
				}
				for _, block := range content.ExtraBlocks {
					parts = append(parts, string(block))
				}
				row.Input = strings.Join(parts, "\n")
			}
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
	return time.UnixMilli(int64(ms)).Local().Format("2006-01-02 15:04:05.000 MST")
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
