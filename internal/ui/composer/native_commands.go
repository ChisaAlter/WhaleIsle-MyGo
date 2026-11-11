package composer

import (
	"context"
	"errors"
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	"sync"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type NativeCommandOwner = session.NativeCommandOwner
type NativeCommandOption = session.NativeCommandOption

func nativeCommandOwner(s session.ViewState, name string) NativeCommandOwner {
	g := s.NativeContributions
	return NativeCommandOwner{HostGeneration: s.ComposerHostGeneration, SessionGeneration: s.ComposerSessionGeneration, PermissionCatalogRevision: s.PermissionCatalogRevision, RegistrationGeneration: s.NativeContributionsGeneration, PortGeneration: s.NativeCommandGeneration(name), SessionID: s.SelectedID, Revision: g.CommandRevision(name), ReleaseVersion: g.ReleaseVersion, ReleaseCommit: g.ReleaseCommit}
}
func nativeCommandRegistered(s session.ViewState, name string) bool {
	g := s.NativeContributions
	if !g.Known {
		return false
	}
	switch name {
	case "file":
		return g.FileCommand
	case "model":
		return g.ModelCommand
	case "permission":
		return g.PermissionCommand
	case "feedback":
		return g.FeedbackCommand
	}
	return false
}
func nativeCommandAvailable(s session.ViewState, name string) bool {
	if !s.ClientReady || s.SelectedID == "" || !nativeCommandRegistered(s, name) {
		return false
	}
	for _, i := range s.Items {
		if i.SessionID == s.SelectedID && i.Origin == "subagent" {
			return false
		}
	}
	if name == "permission" && s.SelectedPermission == "" {
		return false
	}
	if name == "permission" || name == "feedback" {
		for _, d := range s.SlashCommands {
			if d.Name == name {
				return true
			}
		}
		return false
	}
	return true
}
func nativeOwnerCurrent(s session.ViewState, o NativeCommandOwner, name string) bool {
	now := nativeCommandOwner(s, name)
	if name != "permission" {
		now.PermissionCatalogRevision = o.PermissionCatalogRevision
	}
	if now != o || !nativeCommandRegistered(s, name) || !s.ClientReady || s.SelectedID == "" {
		return false
	}
	if name == "permission" {
		return s.SelectedPermission != ""
	}
	return nativeCommandAvailable(s, name)
}
func nativeSlashCatalog(m Model) ([]SlashCommand, error) {
	if !m.Snapshot.SlashCommandsKnown {
		return nil, nil
	}
	rows := FromAPIList(m.Snapshot.SlashCommands)
	for _, name := range []string{"file", "model"} {
		if !nativeCommandAvailable(m.Snapshot, name) || name == "file" && (m.Inert || m.OnNativeCommandAction == nil || m.State != nil && (m.State.Sending || m.State.Preparing > 0)) {
			continue
		}
		for _, r := range rows {
			if r.Line == "/"+name {
				return nil, errors.New("ui-commands: contribution /" + name + " collides with a host command")
			}
		}
		label, description := composerText("文件", "File"), ""
		if name == "model" {
			label = composerText("模型", "Model")
			description = composerText("选择本会话使用的模型", "Select the model for this conversation")
		}
		rows = append(rows, SlashCommand{Line: "/" + name, Label: label, Description: description})
	}
	return rows, nil
}

// builtinSlashFace mirrors the vendor ui-commands client face: first-party
// commands carry a localized label and description while every other row
// keeps its catalog text. Identity follows the descriptor's definitionId —
// a scoped override that reuses a built-in name keeps its own copy. The
// native file/model ports append their own labels in nativeSlashCatalog.
func builtinSlashFace(definitionID string) (label, description string, ok bool) {
	switch definitionID {
	case "@deepseek-ai/dsh-command-goal":
		return composerText("目标", "Goal"),
			composerText("设置或查看长期任务目标", "Set or view the goal for a long-running task"), true
	case "@deepseek-ai/dsh-plan-mode":
		return composerText("计划", "Plan"),
			composerText("进入或退出计划模式", "Enter or leave plan mode"), true
	case "@deepseek-ai/dsh-command-feedback":
		return composerText("反馈", "Feedback"),
			composerText("发送关于当前会话的反馈", "Record feedback about this session"), true
	case "@deepseek-ai/dsh-command-compact":
		return composerText("压缩", "Compact"),
			composerText("压缩以上对话内容", "Compact older conversation history"), true
	case "@deepseek-ai/dsh-permission-presets":
		return composerText("权限", "Permission"),
			composerText("切换权限预设（沙箱模式与审批策略）", "Switch the permission preset (sandbox mode + approval policy)"), true
	case "@deepseek-ai/dsh-session-log-export":
		return composerText("下载日志", "Export"),
			composerText("将当前会话内容导出为 ZIP", "Download this Session log as a ZIP archive"), true
	}
	return "", "", false
}

// A menu segment owns a revision and exact span. Bare Enter follows the source's
// token equality CAS. Neither path restores text after a failed or retired pick.
type nativeCommandSegment struct {
	text     string
	token    TokenRange
	revision uint64
	bare     bool
	client   *DraftClient
}

func (s nativeCommandSegment) consume(d *DraftState) {
	if d == nil {
		return
	}
	token := s.token
	if s.bare {
		if strings.TrimSpace(d.Text) != s.text {
			return
		}
		token = TokenRange{0, len(d.Text)}
	} else if d.Text != s.text || s.client == nil || d.Client != s.client || s.client.Revision() != s.revision {
		return
	}
	if token.Start < 0 || token.End > len(d.Text) || token.End < token.Start {
		return
	}
	if d.Client != nil {
		d.Client.ReplaceText(&ui.TextInputRange{Start: ui.UTF16Len(d.Text[:token.Start]), End: ui.UTF16Len(d.Text[:token.End])}, "")
	} else {
		d.Text = ReplaceToken(d.Text, token, "")
	}
}

type nativeCommandReceipt struct {
	mu           sync.Mutex
	rows         []NativeCommandOption
	err          error
	loaded, done bool
}
type nativeCommandPopup struct {
	owner                             NativeCommandOwner
	name                              string
	segment                           nativeCommandSegment
	ctx                               context.Context
	cancel                            context.CancelFunc
	receipt                           *nativeCommandReceipt
	rows                              []NativeCommandOption
	query, errorText                  string
	active                            int
	loading, started, busy, composing bool
	reveal                            bool
	pointerX, pointerY                float32
	pointerPresent                    bool
	confirmation                      *NativeCommandOption
	acknowledged                      bool
}

// RetireNativeCommands is called when the retained draft leaves the active
// composer, including navigation and window hiding. Workers keep no UI lease.
func (d *DraftState) RetireNativeCommands() {
	if d != nil {
		d.ModelCommand.close()
		d.PermissionCommand.close()
	}
}

func (p *nativeCommandPopup) close() {
	if p != nil && p.cancel != nil {
		p.cancel()
	}
}
func (d *DraftState) openNativeCommand(s session.ViewState, name string, segment nativeCommandSegment) {
	d.ModelCommand.close()
	d.PermissionCommand.close()
	model := Model{State: d, Draft: &d.Text, Snapshot: s, TextInput: d.Client}
	activeSuggestion(model, d.Client)
	dismissSuggestion(model, d.Client)
	ctx, cancel := context.WithCancel(context.Background())
	p := &nativeCommandPopup{owner: nativeCommandOwner(s, name), name: name, segment: segment, ctx: ctx, cancel: cancel, active: -1}
	state := ModelCommandState{Open: true, SessionID: s.SelectedID, popup: p}
	if name == "model" {
		d.ModelCommand = state
	} else {
		d.PermissionCommand = state
	}
	d.Error = ""
}
func (p *nativeCommandPopup) load(m Model) {
	p.started, p.loading, p.errorText = true, true, ""
	receipt := &nativeCommandReceipt{}
	p.receipt = receipt
	go func() {
		var rows []NativeCommandOption
		var err error
		if m.OnLoadNativeCommandOptions == nil {
			err = errors.New("Command option reader unavailable")
		} else {
			rows, err = m.OnLoadNativeCommandOptions(p.ctx, p.owner, p.name)
		}
		receipt.mu.Lock()
		receipt.rows, receipt.err, receipt.loaded = rows, err, true
		receipt.mu.Unlock()
		if m.OnNativeCommandChanged != nil {
			m.OnNativeCommandChanged()
		}
	}()
}
func (p *nativeCommandPopup) settle(m Model, row NativeCommandOption) {
	if p.busy || p.loading || !nativeOwnerCurrent(m.Snapshot, p.owner, p.name) {
		return
	}
	if row.Failure != "" {
		p.errorText = row.Failure
		return
	}
	if row.Confirmation && !p.acknowledged {
		copy := row
		p.confirmation = &copy
		return
	}
	p.busy, p.errorText = true, ""
	acknowledged := p.acknowledged
	receipt := &nativeCommandReceipt{}
	p.receipt = receipt
	go func() {
		var err error
		switch p.name {
		case "model":
			if row.Model == nil || m.OnSelectNativeCommandModel == nil {
				err = errors.New("Model selection unavailable")
			} else {
				err = m.OnSelectNativeCommandModel(p.ctx, p.owner, *row.Model)
			}
		case "permission":
			if m.OnSelectNativeCommandPermission == nil {
				err = errors.New("Permission selection unavailable")
			} else {
				err = m.OnSelectNativeCommandPermission(p.ctx, p.owner, row.ID, acknowledged)
			}
		}
		receipt.mu.Lock()
		receipt.err, receipt.done = err, true
		receipt.mu.Unlock()
		if m.OnNativeCommandChanged != nil {
			m.OnNativeCommandChanged()
		}
	}()
}
func (p *nativeCommandPopup) poll(d *DraftState) bool {
	if p.receipt == nil {
		return false
	}
	r := p.receipt
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loaded {
		p.loading = false
		p.rows = append([]NativeCommandOption(nil), r.rows...)
		for i := range p.rows {
			row := &p.rows[i]
			if p.name == "permission" {
				row.Label = nativeOptionLabel(*row)
				if row.ID == session.PermissionAutoReview {
					row.Detail = composerText("无沙箱运行；每次原生工具调用和 PTC 内层调用前由同一模型进行实验性审查。", "Run without a sandbox after an experimental same-model review of every native tool call and PTC inner call.")
				}
			}
			if row.ID == "failure/deepseek-account" {
				row.Label = composerText("DeepSeek 账户", "DeepSeek account")
			}
		}
		p.reveal = true
		p.active = -1
		if r.err != nil {
			p.errorText = r.err.Error()
		}
		r.loaded = false
	}
	if r.done {
		p.busy = false
		r.done = false
		if r.err != nil {
			p.errorText = r.err.Error()
			p.confirmation = nil
			p.acknowledged = false
			return false
		}
		p.segment.consume(d)
		return true
	}
	return false
}
func nativePopupRows(p *nativeCommandPopup) []NativeCommandOption {
	q := strings.ToLower(strings.TrimSpace(p.query))
	if q == "" {
		return p.rows
	}
	if p.name != "model" {
		out := []NativeCommandOption{}
		for _, r := range p.rows {
			if strings.Contains(strings.ToLower(r.Label), q) || strings.Contains(strings.ToLower(r.Detail), q) {
				out = append(out, r)
			}
		}
		return out
	}
	// Fuzzy ranking is local to each provider. Ungrouped failure rows stay in
	// their original final group and remain error actions rather than routes.
	groups := [][]NativeCommandOption{}
	indices := map[string]int{}
	for _, r := range p.rows {
		index, ok := indices[r.Group]
		if !ok {
			index = len(groups)
			indices[r.Group] = index
			groups = append(groups, nil)
		}
		groups[index] = append(groups[index], r)
	}
	type match struct {
		row    NativeCommandOption
		score  int
		prefix bool
	}
	out := []NativeCommandOption{}
	for _, group := range groups {
		matches := []match{}
		for _, r := range group {
			if score, ok := modelNameScore(r.Label, q); ok {
				matches = append(matches, match{r, score, strings.HasPrefix(strings.ToLower(r.Label), q)})
			}
		}
		sort.SliceStable(matches, func(i, j int) bool {
			if matches[i].prefix != matches[j].prefix {
				return matches[i].prefix
			}
			return matches[i].score > matches[j].score
		})
		for _, m := range matches {
			out = append(out, m.row)
		}
	}
	return out
}
func paintNativeCommandPopup(c *ui.Context, m Model, card ui.Element, colors theme.Colors, panel, border, fg, muted ui.Color) {
	if m.State == nil {
		return
	}
	hover := chrome.Color(colors.Hover)
	caption := chrome.Color(colors.Caption)
	for _, state := range []*ModelCommandState{&m.State.ModelCommand, &m.State.PermissionCommand} {
		p := state.popup
		if !state.Open || p == nil {
			state.close()
			continue
		}
		if !nativeOwnerCurrent(m.Snapshot, p.owner, p.name) || m.Editing || m.Inert {
			state.close()
			continue
		}
		if p.poll(m.State) {
			state.close()
			m.State.EditFocus = true
			// This helper is built after the editor. Schedule the pass that consumes
			// its explicit focus request rather than waiting for unrelated input.
			c.Services().Invalidate()
			continue
		}
		if !p.started {
			p.load(m)
		}
		visible := nativePopupRows(p)
		if p.active < 0 || p.active >= len(visible) {
			p.active = 0
			for i, r := range visible {
				if r.Active {
					p.active = i
					break
				}
			}
		}
		dismiss := func() {
			state.close()
			m.State.EditFocus = true
			c.Services().Invalidate()
		}
		if p.confirmation != nil {
			paintNativePermissionConfirmation(c, m, p, panel, fg)
			continue
		}
		height := float32(320)
		if bounds := card.Bounds(); bounds.W > 0 {
			height = min(height, max(float32(0), bounds.Y-4))
		}
		pane := ui.Column(c.Key("native-command:"+p.name)).Absolute().Left(0).Right(0).BottomPercent(100).Margin(0, 0, 4, 0).MaxHeight(height).Clip().Padding(4).Gap(0)
		pane = menuSurface(c, pane, colors)
		pointerX, pointerY, pointerPresent := pane.PointerPosition()
		pointerX, pointerY = pointerX+pane.Bounds().X, pointerY+pane.Bounds().Y
		pointerMoved := pointerPresent && (!p.pointerPresent || pointerX != p.pointerX || pointerY != p.pointerY)
		p.pointerX, p.pointerY, p.pointerPresent = pointerX, pointerY, pointerPresent
		if !p.composing && (pane.Shortcut(0, ui.KeyEscape) || pane.Shortcut(ui.Shift, ui.KeyTab)) {
			dismiss()
			return
		}
		pane.Children(func() {
			ui.TextInput(c, &p.query).Label("Search command options").Placeholder(nativePopupSearchPlaceholder(p.name)).FillWidth().AutoFocus().
				Margin(2, 2, 3, 2).Padding(5, 7).Radius(12).BorderWidth(0).Background(ui.Transparent).FontSize(12).
				Disabled(p.busy || p.confirmation != nil).OnChange(func() { p.active = 0; p.reveal = true }).HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputCompose {
					p.composing = ev.Text != ""
					return false
				}
				if ev.Kind != ui.InputKeyDown || p.composing || p.busy || p.loading || p.confirmation != nil || ev.Mods != 0 {
					return false
				}
				switch ev.Key {
				case ui.KeyDown:
					if len(visible) > 0 {
						p.active = (p.active + 1) % len(visible)
						p.reveal = true
					}
					return true
				case ui.KeyUp:
					if len(visible) > 0 {
						p.active = (p.active + len(visible) - 1) % len(visible)
						p.reveal = true
					}
					return true
				case ui.KeyEnter, ui.KeyTab:
					if len(visible) > 0 {
						p.settle(m, visible[p.active])
						return true
					}
					return false
				}
				return false
			})
			if p.loading {
				ui.Text(c, composerText("正在加载…", "Loading…")).FontSize(12).TextColor(muted).Padding(5, 7)
			}
			if p.errorText != "" {
				ui.Text(c, p.errorText).FontSize(12).TextColor(fg).MaxHeight(48).Padding(5, 7)
				if !p.busy && !p.loading && len(p.rows) == 0 && ui.Button(c, composerText("重试", "Retry")).Clicked() {
					p.load(m)
				}
			}
			if !p.loading && len(visible) == 0 {
				ui.Text(c, composerText("没有可用选项", "No matching options")).FontSize(12).TextColor(muted).Padding(5, 7)
			}
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(max(float32(0), height-48)).Gap(0).Children(func() {
				group := ""
				for i, row := range visible {
					row := row
					if row.Group != "" && row.Group != group {
						group = row.Group
						label := row.GroupLabel
						if group == "deepseek-account" {
							label = composerText("DeepSeek 账户", "DeepSeek account")
						}
						ui.Text(c, label).FontSize(11).FontWeight(500).FixedLineHeight(16).TextColor(caption).Padding(5, 8, 2, 8)
					}
					button := ui.Button(c, "").Label(row.Label).FillWidth().MinHeight(34).Padding(5, 7).Radius(12).
						AlignItems(ui.Center).Justify(ui.Start).Gap(6).BorderWidth(0).Background(ui.Transparent).
						Disabled(p.busy || p.loading || p.confirmation != nil)
					if pointerMoved && button.Hovered() && !p.busy && !p.loading {
						p.active, p.reveal = i, false
					}
					if i == p.active {
						button = button.Background(hover)
						if p.reveal {
							button.ScrollIntoView()
						}
					}
					button.Children(func() {
						ui.Column(c).FillWidth().Gap(2).Children(func() {
							ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
								if p.name == "model" && row.CatalogLabel != "" {
									ModelLabel(c, m.Snapshot, row.CatalogLabel, ModelLabelOption, row.Active, fg, 12).Grow(1).MinWidth(0)
								} else {
									ui.Text(c, nativeOptionLabel(row)).TextColor(fg).FontSize(12).FixedLineHeight(18).Grow(1)
								}
								if row.ID == session.PermissionAutoReview {
									ui.Text(c, "EXP").FontSize(8).FixedLineHeight(10).FontWeight(500).TextColor(muted)
								}
								if row.Active {
									ui.Icon(c, checkIcon).Size(16, 16).TextColor(fg).Shrink(0)
								}
							})
							if row.ID == session.PermissionAutoReview {
								row.Detail = composerText("无沙盒运行；每次原生工具调用和 PTC 内层调用前由同一模型进行实验性审查。", "Run without a sandbox after an experimental same-model review of every native tool call and PTC inner call.")
							}
							if row.Detail != "" {
								ui.Text(c, row.Detail).FontSize(11).FixedLineHeight(16).TextColor(muted)
							}
						})
					})
					if button.Clicked() {
						p.active = i
						p.settle(m, row)
					}
				}
			})
		})
		p.reveal = false
		if pane.PressedOutside() {
			state.close()
		}
	}
}

func nativePopupSearchPlaceholder(name string) string {
	if name == "model" {
		return composerText("搜索模型…", "Search models…")
	}
	return composerText("搜索…", "Search…")
}
