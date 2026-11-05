package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/diff"
)

var (
	deliverableChevronDownIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 4.5L6 7.5L9 4.5"/></svg>`))

	// Exact 28x28 FileTypeIcon SVGs from @deepseek-ai/dsh-client-ui-primitives/src/FileTypeIcon.tsx
	fileTypeCodeSVG    = ui.MustParseSVG([]byte(`<svg viewBox="0 0 28 28" fill="none"><path d="M17.291 2C17.8261 2.00018 18.3389 2.21465 18.7148 2.5957L23.4219 7.36719C23.7924 7.74272 24 8.24854 24 8.77637V23C24 24.6569 22.6569 26 21 26H7C5.34315 26 4 24.6569 4 23V5C4 3.34315 5.34315 2 7 2H17.291Z" fill="#4D93F8"/><path d="M18 2.38281C18.2738 2.43173 18.526 2.56293 18.7227 2.7626L23.2256 7.32617C23.4274 7.53068 23.5621 7.7902 23.6123 8.07227L19 8.07129C18.4477 8.07129 18 7.62357 18 7.07129V2.38281Z" fill="#FFFFFF" fill-opacity="0.4"/><path d="M11.3746 14.4004L9.28641 16.588C9.09923 16.7841 9.09923 17.0928 9.28641 17.2889L11.3746 19.4765M16.6251 14.4004L18.7133 16.588C18.9005 16.7841 18.9005 17.0928 18.7133 17.2889L16.6251 19.4765M14.7003 13.7002L13.3003 20.1768" stroke="#FFFFFF" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	fileTypeDocxSVG    = ui.MustParseSVG([]byte(`<svg viewBox="0 0 28 28" fill="none"><path d="M17.291 2C17.8261 2.00018 18.3389 2.21465 18.7148 2.5957L23.4219 7.36719C23.7924 7.74272 24 8.24854 24 8.77637V23C24 24.6569 22.6569 26 21 26H7C5.34315 26 4 24.6569 4 23V5C4 3.34315 5.34315 2 7 2H17.291Z" fill="#4D93F8"/><path d="M18 2.38281C18.2738 2.43173 18.526 2.56293 18.7227 2.7626L23.2256 7.32617C23.4274 7.53068 23.5621 7.7902 23.6123 8.07227L19 8.07129C18.4477 8.07129 18 7.62357 18 7.07129V2.38281Z" fill="#FFFFFF" fill-opacity="0.4"/><path d="M8.32775 12.6364H10.0366L11.3377 18.3317H11.4158L12.9002 12.6364H14.4321L15.9215 18.3366H15.9996L17.3008 12.6364H19.0096L16.8296 21H15.2282L13.7009 15.495H13.6326L12.1052 21H10.5087L8.32775 12.6364Z" fill="#FFFFFF"/></svg>`))
	fileTypeExcelSVG   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 28 28" fill="none"><path d="M17.291 2C17.8261 2.00018 18.3389 2.21465 18.7148 2.5957L23.4219 7.36719C23.7924 7.74272 24 8.24854 24 8.77637V23C24 24.6569 22.6569 26 21 26H7C5.34315 26 4 24.6569 4 23V5C4 3.34315 5.34315 2 7 2H17.291Z" fill="#15B567"/><path d="M18 2.38281C18.2738 2.43173 18.526 2.56293 18.7227 2.7626L23.2256 7.32617C23.4274 7.53068 23.5621 7.7902 23.6123 8.07227L19 8.07129C18.4477 8.07129 18 7.62357 18 7.07129V2.38281Z" fill="#FFFFFF" fill-opacity="0.4"/><path d="M10.7429 12.6364L12.6519 15.6927H12.7154L14.6341 12.6364H16.5724L13.8481 16.8182L16.6261 21H14.6634L12.7154 17.9925H12.6519L10.7039 21H8.75098L11.5388 16.8182L8.7998 12.6364H10.7429Z" fill="#FFFFFF"/></svg>`))
	fileTypePptSVG     = ui.MustParseSVG([]byte(`<svg viewBox="0 0 28 28" fill="none"><path d="M17.291 2C17.8261 2.00018 18.3389 2.21465 18.7148 2.5957L23.4219 7.36719C23.7924 7.74272 24 8.24854 24 8.77637V23C24 24.6569 22.6569 26 21 26H7C5.34315 26 4 24.6569 4 23V5C4 3.34315 5.34315 2 7 2H17.291Z" fill="#FA8832"/><path d="M18 2.38281C18.2738 2.43173 18.526 2.56293 18.7227 2.7626L23.2256 7.32617C23.4274 7.53068 23.5621 7.7902 23.6123 8.07227L19 8.07129C18.4477 8.07129 18 7.62357 18 7.07129V2.38281Z" fill="#FFFFFF" fill-opacity="0.4"/><path d="M10.0361 21V12.6364H13.3366C13.9713 12.6364 14.5116 12.7584 14.9576 13.0026C15.4035 13.2435 15.7436 13.5788 15.978 14.0085C16.2156 14.4349 16.3344 14.9265 16.3344 15.483C16.3344 16.0397 16.214 16.5312 15.9731 16.9577C15.7323 17.384 15.3856 17.7177 14.9332 17.9585C14.484 18.1994 13.9388 18.3198 13.2975 18.3198H11.1933V16.904H13.0193C13.361 16.904 13.6426 16.8454 13.8639 16.7282C14.0885 16.6078 14.2545 16.4418 14.3619 16.2303C14.4726 16.0155 14.5279 15.7664 14.5279 15.483C14.5279 15.1966 14.4726 14.9493 14.3619 14.741C14.2545 14.5294 14.0885 14.3666 13.8639 14.2527C13.6394 14.1355 13.3545 14.0769 13.0095 14.0769H11.8085V21H10.0361Z" fill="#FFFFFF"/></svg>`))
	fileTypePdfSVG     = ui.MustParseSVG([]byte(`<svg viewBox="0 0 28 28" fill="none"><path d="M17.291 2C17.8261 2.00018 18.3389 2.21465 18.7148 2.5957L23.4219 7.36719C23.7924 7.74272 24 8.24854 24 8.77637V23C24 24.6569 22.6569 26 21 26H7C5.34315 26 4 24.6569 4 23V5C4 3.34315 5.34315 2 7 2H17.291Z" fill="#F34E4E"/><path d="M18 2.38281C18.2738 2.43173 18.526 2.56293 18.7227 2.7626L23.2256 7.32617C23.4274 7.53068 23.5621 7.7902 23.6123 8.07227L19 8.07129C18.4477 8.07129 18 7.62357 18 7.07129V2.38281Z" fill="#FFFFFF" fill-opacity="0.4"/><path d="M14.1225 11.9818C13.2371 11.4914 11.7975 12.1645 13.1223 15.0525C14.3849 17.8051 16.5044 19.3457 18.1886 19.0462C20.068 18.7121 19.0853 16.8669 16.8351 16.8763C13.8352 16.8888 8.95321 18.4264 9.00051 20.119C9.03529 21.3636 10.6742 20.6929 11.7379 18.6137C12.7117 16.7104 14.0356 14.2738 14.4046 12.7077C14.539 12.1372 14.3168 12.0894 14.1225 11.9818Z" stroke="#FFFFFF" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	fileTypeMdSVG      = ui.MustParseSVG([]byte(`<svg viewBox="0 0 28 28" fill="none"><path d="M17.291 2C17.8261 2.00018 18.3389 2.21465 18.7148 2.5957L23.4219 7.36719C23.7924 7.74272 24 8.24854 24 8.77637V23C24 24.6569 22.6569 26 21 26H7C5.34315 26 4 24.6569 4 23V5C4 3.34315 5.34315 2 7 2H17.291Z" fill="#4D93F8"/><path d="M18 2.38281C18.2738 2.43173 18.526 2.56293 18.7227 2.7626L23.2256 7.32617C23.4274 7.53068 23.5621 7.7902 23.6123 8.07227L19 8.07129C18.4477 8.07129 18 7.62357 18 7.07129V2.38281Z" fill="#FFFFFF" fill-opacity="0.4"/><path d="M7.8 20V14.2H9.35L11.05 17.85H11.12L12.82 14.2H14.37V20H13.15V16.15H13.09L11.52 19.45H10.65L9.08 16.15H9.02V20H7.8Z" fill="#FFFFFF"/><path d="M15.6 20V14.2H17.75C18.4 14.2 18.95 14.32 19.4 14.56C19.85 14.8 20.19 15.14 20.42 15.58C20.65 16.02 20.77 16.53 20.77 17.1C20.77 17.67 20.65 18.18 20.42 18.62C20.19 19.06 19.85 19.4 19.4 19.64C18.95 19.88 18.4 20 17.75 20H15.6ZM16.92 18.88H17.68C18.35 18.88 18.84 18.73 19.16 18.43C19.48 18.13 19.64 17.69 19.64 17.1C19.64 16.51 19.48 16.07 19.16 15.77C18.84 15.47 18.35 15.32 17.68 15.32H16.92V18.88Z" fill="#FFFFFF"/></svg>`))
	fileTypeGenericSVG = ui.MustParseSVG([]byte(`<svg viewBox="0 0 28 28" fill="none"><path d="M17.291 2C17.8261 2.00018 18.3389 2.21465 18.7148 2.5957L23.4219 7.36719C23.7924 7.74272 24 8.24854 24 8.77637V23C24 24.6569 22.6569 26 21 26H7C5.34315 26 4 24.6569 4 23V5C4 3.34315 5.34315 2 7 2H17.291Z" fill="#C9CDD4"/><path d="M18 2.38281C18.2738 2.43173 18.526 2.56293 18.7227 2.7626L23.2256 7.32617C23.4274 7.53068 23.5621 7.7902 23.6123 8.07227L19 8.07129C18.4477 8.07129 18 7.62357 18 7.07129V2.38281Z" fill="#FFFFFF" fill-opacity="0.55"/></svg>`))

	// OpenTargetButton application icons (Chrome 4-color ring & Terminal/Code badge)
	openAppChromeSVG = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="7" fill="#FFFFFF"/><path d="M8 1C10.97 1 13.5 2.85 14.52 5.46H8.15C6.54 5.46 5.21 6.53 4.82 8.01L2.29 3.63C3.57 2.03 5.66 1 8 1Z" fill="#EA4335"/><path d="M1.48 5.46C1.17 6.25 1 7.11 1 8C1 11.54 3.63 14.46 7.04 14.93L9.57 10.55C9.13 10.74 8.64 10.85 8.13 10.85C6.39 10.85 4.98 9.44 4.98 7.7C4.98 6.84 5.33 6.06 5.89 5.49L3.25 2.7C2.46 3.46 1.85 4.4 1.48 5.46Z" fill="#34A853"/><path d="M8 15C7.67 15 7.35 14.98 7.04 14.93L9.57 10.55C10.58 10.09 11.28 9.07 11.28 7.88C11.28 6.95 10.85 6.12 10.17 5.58L14.52 5.46C14.83 6.25 15 7.11 15 8C15 11.87 11.87 15 8 15Z" fill="#FBBC05"/><circle cx="8" cy="8" r="2.75" fill="#4285F4"/></svg>`))
	openAppCodeSVG   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><rect x="1" y="1" width="14" height="14" rx="3.5" fill="#20242A" stroke="#4B5263" stroke-width="0.8"/><path d="M4.8 5.5L7.3 8L4.8 10.5" stroke="#61AFEF" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/><path d="M8.5 10.5H11.4" stroke="#ABB2BF" stroke-width="1.4" stroke-linecap="round"/></svg>`))
)

type deliverablePalette struct {
	borderL1, borderL2, borderL4 ui.Color
	fill, fillHover              ui.Color
	cardGlass                    ui.Color
	tileBg                       ui.Color
	rowHover                     ui.Color
	pillHover                    ui.Color
	addColor, delColor           ui.Color
	pathColor, chevronColor      ui.Color
}

func resolveDeliverablePalette(st ToolCardStyle) deliverablePalette {
	dark := st.FG.R > 128
	if dark {
		return deliverablePalette{
			borderL1:     ui.RGBA(255, 255, 255, 0.05),
			borderL2:     ui.RGBA(255, 255, 255, 0.08),
			borderL4:     ui.RGBA(255, 255, 255, 0.16),
			fill:         ui.RGBA(28, 28, 30, 0.72),
			fillHover:    ui.RGBA(38, 38, 40, 0.78),
			cardGlass:    ui.RGB(33, 33, 35),
			tileBg:       ui.RGBA(255, 255, 255, 0.05),
			rowHover:     ui.RGBA(255, 255, 255, 0.08),
			pillHover:    ui.RGBA(255, 255, 255, 0.08),
			addColor:     ui.RGB(34, 197, 94),
			delColor:     ui.RGB(242, 90, 90),
			pathColor:    ui.RGB(173, 178, 184),
			chevronColor: ui.RGB(207, 211, 214),
		}
	}
	return deliverablePalette{
		borderL1:     ui.RGBA(0, 0, 0, 0.04),
		borderL2:     ui.RGBA(0, 0, 0, 0.10),
		borderL4:     ui.RGBA(0, 0, 0, 0.16),
		fill:         ui.RGB(250, 250, 250),
		fillHover:    ui.RGB(245, 245, 245),
		cardGlass:    ui.RGB(255, 255, 255),
		tileBg:       ui.RGBA(255, 255, 255, 0.50),
		rowHover:     ui.RGBA(38, 49, 72, 0.06),
		pillHover:    ui.RGBA(38, 49, 72, 0.06),
		addColor:     ui.RGB(22, 163, 74),
		delColor:     ui.RGB(220, 38, 38),
		pathColor:    st.Muted,
		chevronColor: st.Muted,
	}
}

// hasVisibleTurnDelivery reports whether a TurnDelivery row will paint visible content.
func hasVisibleTurnDelivery(d session.TurnDelivery, m Model) bool {
	if len(d.Files) > 0 {
		return true
	}
	if m.DisableDeveloperTools || d.Changes == nil {
		return false
	}
	state := m.Snapshot.TurnFiles.Results[session.TurnFileRequest{Target: *d.Changes, Action: "summary"}]
	if state.Summary != nil && len(state.Summary.Files) > 0 {
		return true
	}
	return false
}

// TurnDeliveryView observes the recorded summary and desktop metadata once per
// mounted owner. Each gesture is revalidated against the original declaration
// and captured Host scope. Preview has no route to the general Host opener.
func TurnDeliveryView(c *ui.Context, d session.TurnDelivery, m Model, st ToolCardStyle) {
	if m.DisableDeveloperTools {
		d.Changes = nil
		if len(d.Files) == 0 {
			return
		}
	}
	snapshot := m.Snapshot.TurnFiles
	scope := snapshot.Scope
	enabled := m.HistoryStore != nil && scope.Binding != "" && scope.SessionID == m.Snapshot.SelectedID
	if d.Changes != nil {
		req := session.TurnFileRequest{Target: *d.Changes, Action: "summary"}
		_, exists := snapshot.Results[req]
		if enabled && !exists {
			seat := ui.Box(c.Key(fmt.Sprintf("turn-files-summary-init:%s:%d:%d", scope.Binding, scope.Generation, d.Turn))).Width(0).Height(0)
			asked := ui.Local(seat, "asked", func() bool { return false })
			if !*asked {
				*asked = true
				go m.HistoryStore.RunTurnFileAction(context.Background(), scope, req)
			}
		}
		// A tool's proposed patch is not the Host's recorded workspace change.
		// Missing, expired and failed summaries leave no "Edited files" card;
		// declared deliveries and the tool's own intent view remain independent.
	}
	var preview func(session.TurnFileTarget) error
	if enabled && (m.OnPreviewDeliverable != nil || m.OnPreviewDeliverablePresentation != nil) {
		preview = func(target session.TurnFileTarget) error {
			path, ok := m.HistoryStore.TurnFilePreviewPath(scope, target)
			if !ok {
				return errors.New(historyNavigationText("文件已不可用，请重新载入会话", "The file is no longer available; reload the session"))
			}
			if m.OnPreviewDeliverablePresentation != nil {
				return m.OnPreviewDeliverablePresentation(path, "mini")
			}
			return m.OnPreviewDeliverable(path)
		}
	}
	run := func(target session.TurnFileTarget, action string) {
		if enabled {
			if m.OnOpenChangesReview != nil && target.Kind == "changes" && (action == "summary" || action == "diff") {
				if m.HistoryStore.TurnFilesScope() == scope {
					m.OnOpenChangesReview(scope, target)
				}
				return
			}
			go m.HistoryStore.RunTurnFileAction(context.Background(), scope, session.TurnFileRequest{Target: target, Action: action})
		}
	}
	addressedTurnDeliveryView(c, d, snapshot, enabled, run, preview, st, func(target session.TurnFileTarget, action string) {
		go m.HistoryStore.RunTurnFileAction(context.Background(), scope, session.TurnFileRequest{Target: target, Action: action})
	})
}

// AddressedTurnDeliveryView also serves read-only child histories, whose preview
// callback owns its own result presentation. Keep that existing contract intact.
func AddressedTurnDeliveryView(c *ui.Context, d session.TurnDelivery, snapshot session.TurnFilesSnapshot, enabled bool, run func(session.TurnFileTarget, string), preview func(session.TurnFileTarget), st ToolCardStyle, readers ...func(session.TurnFileTarget, string)) {
	var inspect func(session.TurnFileTarget) error
	if preview != nil {
		inspect = func(target session.TurnFileTarget) error { preview(target); return nil }
	}
	addressedTurnDeliveryView(c, d, snapshot, enabled, run, inspect, st, readers...)
}

func addressedTurnDeliveryView(c *ui.Context, d session.TurnDelivery, snapshot session.TurnFilesSnapshot, enabled bool, run func(session.TurnFileTarget, string), preview func(session.TurnFileTarget) error, st ToolCardStyle, readers ...func(session.TurnFileTarget, string)) {
	scope := snapshot.Scope
	key := fmt.Sprintf("turn-files:%s:%d:%d", scope.Binding, scope.Generation, d.Turn)
	seat := ui.Box(c.Key(key + ":reads")).Width(0).Height(0)
	requested := ui.Local(seat, "requested", func() map[session.TurnFileRequest]bool { return make(map[session.TurnFileRequest]bool) })
	read := run
	if len(readers) > 0 {
		read = readers[0]
	}
	ensure := func(target session.TurnFileTarget, action string) {
		request := session.TurnFileRequest{Target: target, Action: action}
		if _, exists := snapshot.Results[request]; enabled && read != nil && !exists && !(*requested)[request] {
			(*requested)[request] = true
			read(target, action)
		}
	}
	if d.Changes != nil {
		ensure(*d.Changes, "summary")
	}
	hostTarget := session.TurnFileTarget{}
	if d.Changes != nil {
		hostTarget = *d.Changes
	} else if len(d.Files) > 0 {
		hostTarget = d.Files[0].Target
	}
	if d.Changes != nil || len(d.Files) > 0 {
		ensure(hostTarget, "host")
	}
	if len(d.Files) == 0 {
		if d.Changes == nil {
			return
		}
		summary := snapshot.Results[session.TurnFileRequest{Target: *d.Changes, Action: "summary"}].Summary
		if summary == nil || len(summary.Files) == 0 {
			return
		}
	}
	host := snapshot.Results[session.TurnFileRequest{Target: hostTarget, Action: "host"}]
	nativeAvailable := enabled && host.Host != nil && host.Host.Available
	pal := resolveDeliverablePalette(st)
	box := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(8)
	expanded := ui.Local(box, "expanded", func() bool { return false })
	box.Children(func() {
		if host.Error != "" && !host.Unavailable {
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				ui.Text(c, host.Error).FontSize(11).TextColor(st.Muted)
				if stdButton(c, st.Colors, i18n.T(i18n.KeyChatDeliveryRetryHost), buttonGhost, !enabled || host.Busy || run == nil).Clicked() {
					run(hostTarget, "host")
				}
			})
		} else if host.Host != nil && !host.Host.Available {
			ui.Text(c, historyNavigationText("Host 桌面不可用", "Host desktop unavailable")).FontSize(11).TextColor(st.Muted)
		}
		if d.Changes != nil {
			target := *d.Changes
			state := snapshot.Results[session.TurnFileRequest{Target: target, Action: "summary"}]
			if state.Summary != nil && len(state.Summary.Files) > 0 {
				single := len(state.Summary.Files) == 1
				cardBorder := pal.borderL2
				if single {
					cardBorder = pal.borderL1
				}
				chrome.GlassSurface(ui.Column(c.Key("changes")).FillWidth().MinWidth(0).Radius(16), pal.cardGlass, chrome.MaterialPanel, 16).Border(0.5, cardBorder).Clip().DrawOver(func(p *ui.Painter, r ui.Rect) {
					if cardBorder.A > 0 && r.W > 0 && r.H > 0 {
						p.Stroke(r, cardBorder, 16, 0.5)
					}
				}).Children(func() {
					headerTitle := fmt.Sprintf(historyNavigationText("已编辑 %d 个文件", "Edited %d files"), state.Summary.Total)
					if state.Summary.Total == 1 && len(state.Summary.Files) == 1 {
						headerTitle = historyNavigationText("已编辑 ", "Edited ") + path.Base(strings.ReplaceAll(state.Summary.Files[0].Display, `\`, "/"))
					}
					hdr := ui.Button(c.Key("changes-header"), "").Label(headerTitle).FillWidth().MinWidth(0).Height(60).
						Padding(8, 10).Gap(10).BorderWidth(0).Background(pal.fill).AlignItems(ui.Center).
						Disabled(!enabled || state.Busy || run == nil)
					if hdr.Hovered() {
						hdr.Background(pal.fillHover)
					}
					hdr.Children(func() {
						ui.Box(c).Size(28, 28).Shrink(0).Center().PassThrough().Children(func() {
							headerIconName := ""
							if single {
								headerIconName = state.Summary.Files[0].Display
							}
							paintDeliverableFileTileBadge(c, headerIconName, st.Accent, 28)
						})
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).PassThrough().Children(func() {
							ui.Text(c, headerTitle).FontSize(13).FixedLineHeight(20).FontWeight(500).TextColor(st.FG).SingleLine().PassThrough()
							if single && state.Summary.Files[0].Binary {
								ui.Text(c, historyNavigationText("二进制", "Binary")).FontSize(10).FixedLineHeight(16).TextColor(pal.pathColor).PassThrough()
							} else {
								ui.Row(c).Gap(6).AlignItems(ui.Center).PassThrough().Children(func() {
									ui.Text(c, fmt.Sprintf("+%d", state.Summary.Added)).FontSize(10).FixedLineHeight(16).TextColor(pal.addColor).PassThrough()
									ui.Text(c, fmt.Sprintf("-%d", state.Summary.Deleted)).FontSize(10).FixedLineHeight(16).TextColor(pal.delColor).PassThrough()
								})
							}
						})
					})
					if hdr.Clicked() && enabled && !state.Busy && run != nil {
						run(target, "summary")
					}
					rows := state.Summary.Files
					if !*expanded && len(rows) > 4 {
						rows = rows[:4]
					}
					if len(rows) > 0 {
						ui.Box(c).FillWidth().Height(1).Background(pal.borderL2)
					}
					for index, file := range rows {
						selected := target
						selected.Index = index
						comparison := snapshot.Results[session.TurnFileRequest{Target: selected, Action: "diff"}]
						ui.Column(c.Key(fmt.Sprintf("changed:%d", index))).FillWidth().MinWidth(0).Children(func() {
							rowWrap := ui.Row(c).FillWidth().MinWidth(0).Height(34).Padding(0).Gap(0).AlignItems(ui.Center)
							showInline := ui.Local(rowWrap, "show-diff", func() bool { return false })
							if rowWrap.Hovered() {
								rowWrap.Background(pal.rowHover)
							}
							rowWrap.Children(func() {
								rowBtn := ui.Button(c.Key("diff-btn"), "").Label(file.Display).Grow(1).MinWidth(0).Height(34).
									Padding(7, 14, 7, 14).Gap(10).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).
									Disabled(!enabled || comparison.Busy || run == nil)
								if rowBtn.Hovered() {
									rowWrap.Background(pal.rowHover)
								}
								rowBtn.Children(func() {
									ui.Text(c, file.Display).FontSize(12).FixedLineHeight(20).TextColor(pal.pathColor).Grow(1).MinWidth(0).SingleLine().PassThrough()
									if file.Binary {
										ui.Text(c, historyNavigationText("二进制", "Binary")).FontSize(11).FixedLineHeight(18).TextColor(pal.pathColor).Shrink(0).PassThrough()
									} else if file.Oversized {
										ui.Text(c, i18n.T(i18n.KeyChatDeliveryTooLarge)).FontSize(11).FixedLineHeight(18).TextColor(pal.pathColor).Shrink(0).PassThrough()
									} else {
										ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).PassThrough().Children(func() {
											ui.Text(c, fmt.Sprintf("+%d", file.Added)).FontSize(11).FixedLineHeight(18).TextColor(pal.addColor).PassThrough()
											ui.Text(c, fmt.Sprintf("-%d", file.Deleted)).FontSize(11).FixedLineHeight(18).TextColor(pal.delColor).PassThrough()
										})
									}
								})
								hoverSince := ui.Local(rowBtn, "diff-hover-since", func() time.Time { return time.Time{} })
								if rowBtn.Hovered() {
									if hoverSince.IsZero() {
										*hoverSince = c.Now()
									}
									remaining := 500*time.Millisecond - c.Now().Sub(*hoverSince)
									if remaining > 0 {
										rowBtn.DrawOver(func(p *ui.Painter, _ ui.Rect) { p.After(remaining) })
									}
								} else {
									*hoverSince = time.Time{}
								}
								showHover := enabled && rowBtn.Hovered() && !hoverSince.IsZero() && c.Now().Sub(*hoverSince) >= 500*time.Millisecond
								if showHover {
									ensure(selected, "diff")
								}
								ui.PopoverBase(c, rowBtn, &showHover, func(panel ui.Element) {
									panel.Width(620).MaxHeight(360).Padding(10).Radius(8).Border(1, st.Border).Background(st.Field).Children(func() {
										ui.Text(c, file.Display).FontSize(12).TextColor(st.FG)
										ui.Scroll(c).MaxHeight(310).Children(func() {
											if comparison.Diff != nil && comparison.Diff.Kind == "text" {
												diff.View(c, comparison.Diff.Patch(), diff.Style{FontSize: 11, FG: st.FG, Muted: st.Muted, Add: pal.addColor, Del: pal.delColor, Hunk: st.Accent})
											} else if comparison.Error != "" {
												ui.Text(c, comparison.Error).TextColor(st.Muted)
											} else if comparison.Diff != nil {
												ui.Text(c, comparison.Diff.Kind).TextColor(st.Muted)
											} else {
												ui.Text(c, historyNavigationText("正在读取差异…", "Reading diff…")).TextColor(st.Muted)
											}
										})
									})
								})
								if rowBtn.Clicked() && enabled && !comparison.Busy && run != nil {
									*showInline = true
									run(selected, "diff")
								}
								deliverableActionPill(c, selected, file.Display, snapshot, nativeAvailable, run, st, 24)
							})
							if *showInline && comparison.Error != "" && !comparison.Unavailable {
								ui.Row(c).Padding(0, 14, 6, 14).Children(func() {
									ui.Text(c, comparison.Error).FontSize(10).TextColor(st.Muted)
								})
							}
							if content := comparison.Diff; *showInline && content != nil {
								ui.Column(c).FillWidth().Padding(4, 14, 8, 14).Children(func() {
									switch content.Kind {
									case "binary":
										ui.Text(c, i18n.T(i18n.KeyChatDeliveryBinary)).FontSize(11).TextColor(st.Muted)
									case "oversized":
										ui.Text(c, i18n.T(i18n.KeyChatDeliveryTooLarge)).FontSize(11).TextColor(st.Muted)
									case "text":
										patch := content.Patch()
										if len(patch) > 64<<10 {
											patch = patch[:64<<10]
											if at := strings.LastIndexByte(patch, '\n'); at >= 0 {
												patch = patch[:at]
											}
											ui.Text(c, i18n.T(i18n.KeyChatDeliveryDiffClip)).FontSize(10).TextColor(st.Muted)
										}
										if patch == "" {
											ui.Text(c, i18n.T(i18n.KeyChatToolNoDiff)).FontSize(11).TextColor(st.Muted)
										} else {
											diff.View(c, patch, diff.Style{FontSize: 11, FG: st.FG, Muted: st.Muted, Add: ui.RGB(40, 160, 90), Del: ui.RGB(210, 70, 70), Hunk: st.Accent})
										}
									}
								})
							}
						})
					}
				})
			} else if state.Error != "" && !state.Unavailable {
				ui.Row(c).Gap(6).Children(func() {
					ui.Text(c, state.Error).FontSize(11).TextColor(st.Muted)
					if stdButton(c, st.Colors, i18n.T(i18n.KeyChatDeliveryRetryReview), buttonGhost, !enabled || state.Busy || run == nil).Clicked() {
						run(target, "summary")
					}
				})
			}
		}
		rows := d.Files
		if !*expanded && len(rows) > 4 {
			rows = rows[:4]
		}
		vw, _ := c.Size()
		if len(rows) >= 2 && vw >= 680 {
			for i := 0; i < len(rows); i += 2 {
				pair := rows[i:]
				if len(pair) > 2 {
					pair = pair[:2]
				}
				ui.Row(c.Key(fmt.Sprintf("presented-row:%d", i))).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Start).Children(func() {
					for _, file := range pair {
						ui.Column(c.Key(fmt.Sprintf("col:%d:%d", file.Target.Seq, file.Target.Index))).Grow(1).Basis(0).MinWidth(0).Children(func() {
							presentedFileCard(c, file, snapshot, enabled, nativeAvailable, run, preview, st)
						})
					}
					if len(pair) == 1 {
						ui.Box(c).Grow(1).Basis(0).MinWidth(0)
					}
				})
			}
		} else {
			for _, file := range rows {
				presentedFileCard(c, file, snapshot, enabled, nativeAvailable, run, preview, st)
			}
		}
		hasMore := len(d.Files) > 4
		if d.Changes != nil {
			if sum := snapshot.Results[session.TurnFileRequest{Target: *d.Changes, Action: "summary"}].Summary; sum != nil && len(sum.Files) > 4 {
				hasMore = true
			}
		}
		if hasMore {
			label := i18n.T(i18n.KeyChatDeliveryShowAll)
			if *expanded {
				label = i18n.T(i18n.KeyChatRowCollapse)
			}
			if stdButton(c, st.Colors, label, buttonGhost, false).Clicked() {
				*expanded = !*expanded
			}
		}
	})
}

func presentedFileCard(c *ui.Context, file session.TurnPresentedFile, snapshot session.TurnFilesSnapshot, enabled, nativeAvailable bool, run func(session.TurnFileTarget, string), preview func(session.TurnFileTarget) error, st ToolCardStyle) {
	target := file.Target
	name := path.Base(strings.ReplaceAll(file.Path, `\`, "/"))
	meta := strings.TrimSpace(file.Description)
	if meta == "" {
		meta = strings.ToUpper(attachmentExtension(name))
		if meta == "" {
			meta = i18n.T(i18n.KeyChatAttachmentFile)
		}
	}
	pal := resolveDeliverablePalette(st)
	owner := ui.Column(c.Key(fmt.Sprintf("presented:%d:%d", target.Seq, target.Index))).FillWidth().MinWidth(0).Gap(3)
	previewError := ui.Local(owner, "preview-error", func() string { return "" })
	previewErrorAt := ui.Local(owner, "preview-error-at", func() time.Time { return time.Time{} })
	state := snapshot.Results[(session.TurnFileRequest{Target: target, Action: "open"}).StateKey()]
	apps, exists := snapshot.Results[session.TurnFileRequest{Target: target, Action: "applications"}]
	nativeAvailable = nativeAvailable && !state.NativeUnavailable
	asked := ui.Local(owner, "applications-requested", func() bool { return false })
	if nativeAvailable && run != nil && !exists && !*asked {
		*asked = true
		run(target, "applications")
	}
	owner.Children(func() {
		cardRow := ui.Row(c).FillWidth().MinWidth(0).Height(60).Radius(16).Background(pal.fill).BorderWidth(0).AlignItems(ui.Center).Clip().DrawOver(func(p *ui.Painter, r ui.Rect) {
			if pal.borderL1.A > 0 && r.W > 0 && r.H > 0 {
				p.Stroke(r, pal.borderL1, 16, 0.5)
			}
		})
		if cardRow.Hovered() {
			cardRow.Background(pal.fillHover)
		}
		cardRow.Children(func() {
			label := historyNavigationText("预览 ", "Preview ") + name
			card := ui.Button(c.Key("preview"), "").Label(label).Description(file.Path).Tooltip(file.Path).Grow(1).MinWidth(0).Height(60).
				Padding(8, 10).Gap(10).Radius(16).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).
				Disabled(!enabled || preview == nil)
			if card.Hovered() {
				cardRow.Background(pal.fillHover)
			}
			card.Children(func() {
				ui.Box(c).Size(38, 38).Radius(8).Background(pal.tileBg).Border(0.5, pal.borderL2).Shrink(0).Center().PassThrough().Children(func() {
					paintDeliverableFileTileBadge(c, name, st.Accent, 24)
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).PassThrough().Children(func() {
					ui.Text(c, name).FontSize(13).FixedLineHeight(20).FontWeight(500).TextColor(st.FG).SingleLine().PassThrough()
					status, failed, opacity, repaint := presentedMetadata(snapshot, state, meta, c.Now())
					if *previewError != "" && !state.Busy && !state.SettledAt.After(*previewErrorAt) {
						status, failed, opacity, repaint = historyNavigationText("无法预览文件，点击重试", "Could not preview file; try again"), true, 1, 0
					}
					if apps.Error != "" && !state.Busy && !state.Ready && state.Error == "" {
						status, failed = historyNavigationText("无法读取可用应用", "Could not read available applications"), true
					}
					ink := pal.pathColor
					if failed {
						ink = st.Err
					}
					presentedStatusSeat(c, status, ink, opacity, repaint)
				})
			})
			if card.Clicked() && enabled && preview != nil {
				*previewError = ""
				if err := preview(target); err != nil {
					*previewError = err.Error()
					*previewErrorAt = c.Now()
				}
			}
			deliverableActionPill(c, target, name, snapshot, nativeAvailable, run, st, 24)
		})
	})
}

func deliverableActionPill(c *ui.Context, target session.TurnFileTarget, displayName string, snapshot session.TurnFilesSnapshot, nativeAvailable bool, run func(session.TurnFileTarget, string), st ToolCardStyle, height float32) {
	name := path.Base(strings.ReplaceAll(displayName, `\`, "/"))
	pal := resolveDeliverablePalette(st)
	state := snapshot.Results[(session.TurnFileRequest{Target: target, Action: "open"}).StateKey()]
	nativeAvailable = nativeAvailable && !state.NativeUnavailable
	apps, exists := snapshot.Results[session.TurnFileRequest{Target: target, Action: "applications"}]
	seat := ui.Box(c.Key(fmt.Sprintf("pill-apps:%s:%d:%v", snapshot.Scope.Binding, snapshot.Scope.Generation, target))).Width(0).Height(0)
	asked := ui.Local(seat, "asked", func() bool { return false })
	if nativeAvailable && run != nil && !exists && !*asked {
		*asked = true
		run(target, "applications")
	}
	if height <= 0 {
		height = 24
	}
	menuLabel := historyNavigationText("文件操作：", "File actions: ") + name
	var menu ui.Element
	ui.Row(c.Key("actions")).Width(42).Height(height).Margin(0, 10, 0, 0).Gap(0).Radius(8).Border(.5, pal.borderL4).Clip().Children(func() {
		main := ui.Button(c, "").Label(historyNavigationText("打开文件：", "Open file: ") + name).Tooltip(historyNavigationText("打开文件", "Open file")).Width(24).Height(height).Padding(0).BorderWidth(0).Radius(0).Background(ui.Transparent).Disabled(!nativeAvailable || state.Busy || apps.Busy || run == nil)
		main.Children(func() { paintOpenTargetAppIcon(c, name, apps.Applications) })
		if main.Hovered() {
			main.Background(pal.pillHover)
		}
		if main.Clicked() && run != nil {
			action := "reveal"
			if len(apps.Applications) > 0 {
				action = "open"
				for _, app := range apps.Applications {
					if app.Default {
						action = session.TurnFileApplicationAction(app.ID)
						break
					}
				}
			}
			run(target, action)
		}
		ui.Box(c).Width(1).Height(height).Background(pal.borderL4)
		menu = ui.Button(c, "").Label(menuLabel).Tooltip(menuLabel).Width(17).Height(height).Padding(0).BorderWidth(0).Radius(0).Background(ui.Transparent).Disabled(!nativeAvailable || state.Busy || run == nil)
		menu.Children(func() { ui.Icon(c, deliverableChevronDownIcon).Size(10, 10).TextColor(pal.chevronColor).PassThrough() })
		if menu.Hovered() {
			menu.Background(pal.pillHover)
		}
	})
	menu.Menu(func(menu *ui.Menu) {
		if menu.Item(i18n.T(i18n.KeyChatDeliveryOpenHost)).Disabled(!nativeAvailable || state.Busy || run == nil).Chosen() && nativeAvailable && !state.Busy && run != nil {
			run(target, "open")
		}
		if menu.Item(i18n.T(i18n.KeyChatDeliveryRevealHost)).Disabled(!nativeAvailable || state.Busy || run == nil).Chosen() && nativeAvailable && !state.Busy && run != nil {
			run(target, "reveal")
		}
		menu.Submenu(historyNavigationText("选择应用打开", "Open with…"), func(sub *ui.Menu) {
			for _, app := range apps.Applications {
				caption := strings.TrimSpace(app.Name)
				if caption == "" {
					caption = app.ID
				}
				if app.Default {
					caption += historyNavigationText("（默认）", " (default)")
				}
				if sub.Item(caption).Disabled(!nativeAvailable || state.Busy || run == nil).Chosen() && nativeAvailable && !state.Busy && run != nil {
					run(target, session.TurnFileApplicationAction(app.ID))
				}
			}
			if sub.Item(historyNavigationText("刷新应用列表", "Refresh applications")).Disabled(!nativeAvailable || apps.Busy || run == nil).Chosen() && nativeAvailable && !apps.Busy && run != nil {
				run(target, "applications")
			}
		})
	})
}

func paintOpenTargetAppIcon(c *ui.Context, name string, apps []session.TurnFileApplication) {
	for _, app := range apps {
		if app.Default && app.Icon != nil {
			iconID := strings.ToLower(strings.TrimSpace(*app.Icon))
			switch {
			case strings.Contains(iconID, "chrome") || strings.Contains(iconID, "browser") || strings.Contains(iconID, "edge"):
				ui.Image(c, openAppChromeSVG).Size(15, 15).PassThrough()
				return
			case strings.Contains(iconID, "code") || strings.Contains(iconID, "cursor") || strings.Contains(iconID, "terminal") || strings.Contains(iconID, "zed") || strings.Contains(iconID, "idea"):
				ui.Image(c, openAppCodeSVG).Size(15, 15).PassThrough()
				return
			}
		}
	}
	ext := attachmentExtension(name)
	switch ext {
	case "doc", "docx", "xls", "xlsx", "csv", "ppt", "pptx", "pdf", "html", "htm", "svg", "png", "jpg", "jpeg", "gif", "webp":
		ui.Image(c, openAppChromeSVG).Size(15, 15).PassThrough()
	default:
		if attachmentKindIcon(name, "") == attachmentIconCode {
			ui.Image(c, openAppCodeSVG).Size(15, 15).PassThrough()
			return
		}
		ui.Image(c, openAppChromeSVG).Size(15, 15).PassThrough()
	}
}

func paintDeliverableFileTileBadge(c *ui.Context, name string, accent ui.Color, edge float32) {
	if edge <= 0 {
		edge = 28
	}
	ext := attachmentExtension(name)
	switch ext {
	case "md", "markdown", "mdx":
		ui.Image(c, fileTypeMdSVG).Size(edge, edge).PassThrough()
	case "doc", "docx":
		ui.Image(c, fileTypeDocxSVG).Size(edge, edge).PassThrough()
	case "xls", "xlsx", "csv":
		ui.Image(c, fileTypeExcelSVG).Size(edge, edge).PassThrough()
	case "ppt", "pptx":
		ui.Image(c, fileTypePptSVG).Size(edge, edge).PassThrough()
	case "pdf":
		ui.Image(c, fileTypePdfSVG).Size(edge, edge).PassThrough()
	default:
		if name == "" || attachmentKindIcon(name, "") == attachmentIconCode {
			ui.Image(c, fileTypeCodeSVG).Size(edge, edge).PassThrough()
			return
		}
		ui.Image(c, fileTypeGenericSVG).Size(edge, edge).PassThrough()
	}
}

func fallbackToolDiffsForTurnSummary(call *session.ToolCallView) []toolFileDiff {
	if call == nil || call.State == "preparing" || toolSummaryHasCue(call.State) {
		return nil
	}
	if model := toolDiffModelFor(call); model != nil && len(model.Diffs) > 0 {
		return model.Diffs
	}
	args, ok := terminalModelObject(json.RawMessage(call.Arguments))
	if !ok {
		return nil
	}
	intended, ok := toolDiffModelIntent(call.Name, args)
	if !ok {
		return nil
	}
	if call.Result != nil {
		res := toolSummaryReadResult(call.Result)
		if res.failed || res.stopped {
			return nil
		}
		meta, _ := terminalModelObject(res.meta)
		if applied, valid := toolSummaryAppliedDiffs(meta["diffs"]); valid && len(applied) > 0 {
			return applied
		}
	}
	return []toolFileDiff{intended}
}

func fallbackTurnChangesSummary(snapshot session.ViewState, d session.TurnDelivery) *session.TurnChangesSummary {
	msgs := projectChatMessages(snapshot)
	start := 0
	end := len(msgs)
	foundEnd := false
	for i, msg := range msgs {
		if msg.EventType == "turn/end" {
			if (d.Changes != nil && uint64(msg.Seq) == d.Changes.Seq) || turnEndNumber(msg) == d.Turn {
				end = i
				foundEnd = true
				break
			}
			start = i + 1
		}
	}
	if !foundEnd && len(d.Files) == 0 {
		return nil
	}
	byDisplay := make(map[string]session.TurnChangedFile)
	for _, msg := range msgs[start:end] {
		if msg.Kind != session.MessageKindTool || msg.Tool == nil {
			continue
		}
		diffs := fallbackToolDiffsForTurnSummary(msg.Tool)
		for _, item := range diffs {
			display := relativeDeliverableDisplay(snapshot.SelectedCwd, item.Path)
			if display == "" {
				continue
			}
			oldText := ""
			if item.OldText != nil {
				oldText = *item.OldText
			}
			added, deleted := toolDiffCounts(oldText, item.NewText)
			cur := byDisplay[display]
			cur.Path = item.Path
			cur.Display = display
			cur.Added += added
			cur.Deleted += deleted
			byDisplay[display] = cur
		}
	}
	for _, f := range d.Files {
		display := relativeDeliverableDisplay(snapshot.SelectedCwd, f.Path)
		if display == "" {
			continue
		}
		if _, exists := byDisplay[display]; !exists && isBinaryDeliverablePath(f.Path) {
			byDisplay[display] = session.TurnChangedFile{
				Path:    f.Path,
				Display: display,
				Binary:  true,
			}
		}
	}
	if len(byDisplay) == 0 {
		return nil
	}
	files := make([]session.TurnChangedFile, 0, len(byDisplay))
	totalAdded, totalDeleted := 0, 0
	for _, item := range byDisplay {
		files = append(files, item)
		totalAdded += item.Added
		totalDeleted += item.Deleted
	}
	slices.SortFunc(files, func(a, b session.TurnChangedFile) int {
		return strings.Compare(a.Display, b.Display)
	})
	return &session.TurnChangesSummary{
		Turn:    d.Turn,
		Total:   len(files),
		Added:   totalAdded,
		Deleted: totalDeleted,
		Files:   files,
	}
}

func turnEndNumber(msg session.Message) uint64 {
	if len(msg.Raw) == 0 {
		return 0
	}
	var env struct {
		Data struct {
			Turn *uint64 `json:"turn"`
		} `json:"data"`
	}
	if json.Unmarshal(msg.Raw, &env) == nil && env.Data.Turn != nil {
		return *env.Data.Turn
	}
	return 0
}

func relativeDeliverableDisplay(cwd, targetPath string) string {
	cleanPath := strings.TrimSpace(strings.ReplaceAll(targetPath, `\`, "/"))
	if cleanPath == "" {
		return ""
	}
	cleanCwd := strings.TrimRight(strings.TrimSpace(strings.ReplaceAll(cwd, `\`, "/")), "/")
	if cleanCwd != "" && len(cleanPath) > len(cleanCwd)+1 && strings.EqualFold(cleanPath[:len(cleanCwd)], cleanCwd) && cleanPath[len(cleanCwd)] == '/' {
		return cleanPath[len(cleanCwd)+1:]
	}
	lower := strings.ToLower(cleanPath)
	for _, marker := range []string{"/.whaleisle/workspaces/", "/workspaces/"} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			rest := cleanPath[idx+len(marker):]
			if slash := strings.IndexByte(rest, '/'); slash >= 0 && slash+1 < len(rest) {
				return rest[slash+1:]
			}
		}
	}
	return cleanPath
}

func isBinaryDeliverablePath(filePath string) bool {
	switch attachmentExtension(filePath) {
	case "doc", "docx", "xls", "xlsx", "ppt", "pptx", "pdf",
		"png", "jpg", "jpeg", "gif", "webp", "bmp", "ico",
		"zip", "tar", "gz", "tgz", "7z", "rar",
		"mp3", "wav", "ogg", "mp4", "mov", "avi", "mkv", "webm",
		"exe", "dll", "so", "dylib", "bin", "db", "sqlite", "sqlite3":
		return true
	default:
		return false
	}
}

// TurnFileNativeActions retains explicit Host actions for changed-file rows.
func TurnFileNativeActions(c *ui.Context, target session.TurnFileTarget, snapshot session.TurnFilesSnapshot, enabled bool, run func(session.TurnFileTarget, string), st ToolCardStyle) {
	state := snapshot.Results[(session.TurnFileRequest{Target: target, Action: "open"}).StateKey()]
	enabled = enabled && !state.NativeUnavailable
	apps, exists := snapshot.Results[session.TurnFileRequest{Target: target, Action: "applications"}]
	seat := ui.Box(c.Key(fmt.Sprintf("native-apps:%s:%d:%v", snapshot.Scope.Binding, snapshot.Scope.Generation, target))).Width(0).Height(0)
	requested := ui.Local(seat, "requested", func() bool { return false })
	if enabled && run != nil && !exists && !*requested {
		*requested = true
		run(target, "applications")
	}
	ui.Row(c).Gap(6).Children(func() {
		for _, action := range []string{"open", "reveal"} {
			label := i18n.T(i18n.KeyChatDeliveryOpenHost)
			if action == "reveal" {
				label = i18n.T(i18n.KeyChatDeliveryRevealHost)
			}
			if state.Error != "" && state.Action == action {
				label = i18n.T(i18n.KeyChatDeliveryRetryAction) + " " + action
			}
			if stdButton(c, st.Colors, label, buttonGhost, !enabled || state.Busy || run == nil).Clicked() {
				run(target, action)
			}
		}
		ui.Button(c, "Open with…").FontSize(10).Disabled(!enabled || state.Busy || apps.Busy || run == nil).Menu(func(menu *ui.Menu) {
			for _, app := range apps.Applications {
				caption := strings.TrimSpace(app.Name)
				if caption == "" {
					caption = app.ID
				}
				if app.Default {
					caption += " (default)"
				}
				if menu.Item(caption).Disabled(!enabled || state.Busy || run == nil).Chosen() && enabled && !state.Busy && run != nil {
					run(target, session.TurnFileApplicationAction(app.ID))
				}
			}
			if menu.Item("Refresh applications").Disabled(!enabled || apps.Busy || run == nil).Chosen() && enabled && !apps.Busy && run != nil {
				run(target, "applications")
			}
		})
		status, failed, opacity, repaint := presentedMetadata(snapshot, state, "", c.Now())
		if apps.Busy && !state.Busy {
			status = historyNavigationText("正在读取应用…", "Loading applications…")
		}
		if apps.Error != "" && state.Error == "" && !state.Busy {
			status, failed = historyNavigationText("无法读取可用应用", "Could not read available applications"), true
		}
		ink := st.Muted
		if failed {
			ink = st.Err
		}
		ui.Box(c).Grow(1).MinWidth(0).Height(16).Tooltip(state.Error + "\n" + apps.Error).Children(func() { presentedStatusSeat(c, status, ink, opacity, repaint) })
	})
}
