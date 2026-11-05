package chat

import (
	"context"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type goalDockEditor struct {
	editing, pending     bool
	draft, error, action string
}

// GoalDockView retains the strip while an action is pending. Only acknowledged
// saves leave the editor; a failed CAS preserves the exact user's draft.
func GoalDockView(c *ui.Context, m PlanBoardModel, stores ...*session.Store) {
	g := m.Board.Goal
	if g == nil || g.Phase == "complete" || g.Phase == "completed" {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	scope := g.ID
	if len(stores) > 0 && stores[0] != nil {
		scope = stores[0].PlanHistoryBinding() + ":" + g.ID
	}
	root := ui.Row(c.Key("goal-dock:"+scope)).FillWidth().MinWidth(0).MinHeight(36).Shrink(0).Padding(4, 5, 4, 12).Gap(10).AlignItems(ui.Center).Radius(12).Draw(chrome.Glass(chrome.Color(colors.InputMajor), chrome.MaterialPanel, 12))
	state := ui.Local(root, "editor", func() goalDockEditor { return goalDockEditor{draft: g.Objective} })
	if !state.editing {
		root.Height(36)
	}
	var run func(context.Context, string, string) error
	if len(stores) > 0 && stores[0] != nil {
		run = stores[0].GoalDockAction(g.ID, g.Revision)
	}
	svc := c.Services()
	start := func(action string) {
		if state.pending {
			return
		}
		state.pending = true
		state.error = ""
		state.action = action
		draft := strings.TrimSpace(state.draft)
		if run == nil {
			switch action {
			case "edit":
				if m.OnEditGoal != nil {
					m.OnEditGoal(draft)
				}
				state.editing = false
			case "pause":
				if m.OnPauseGoal != nil {
					m.OnPauseGoal()
				}
			case "resume":
				if m.OnResumeGoal != nil {
					m.OnResumeGoal()
				}
			case "clear":
				if m.OnClearGoal != nil {
					m.OnClearGoal()
				}
			}
			state.pending = false
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := run(ctx, action, draft)
			mygo.RunOnMain(func() {
				state.pending = false
				state.action = ""
				if err != nil {
					state.error = err.Error()
				} else if action == "edit" {
					state.editing = false
				}
				svc.Invalidate()
			})
		}()
	}
	button := func(glyph, label, action string, disabled bool) {
		b := ui.Button(c, "").Label(label).Tooltip(label).Size(28, 28).Padding(0).Radius(14).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Disabled(state.pending || disabled)
		b.Children(func() {
			if state.pending && state.action == action {
				ui.Spinner(c).Size(14, 14)
			} else {
				ui.Icon(c, goalDockIcons[glyph]).Size(14, 14)
			}
		})
		if b.Clicked() {
			start(action)
		}
	}
	root.Children(func() {
		if state.editing {
			field := ui.TextAreaBase(c, &state.draft).Lines(1, 6).FontSize(13).FixedLineHeight(20).Padding(0).BorderWidth(0).Background(ui.Transparent).Label(referenceCopy("Goal objective", "目标内容")).FillWidth().Grow(1).MinWidth(0).ReadOnly(state.pending).AutoFocus()
			keys := ui.Local(field, "goal-keys", func() goalEditorKeys { return goalEditorKeys{} })
			composing := field.Composing()
			keys.observe(composing)
			field.HandleInput(func(ev ui.InputEvent) bool {
				return keys.handle(ev, composing, state.pending, func() {
					if strings.TrimSpace(state.draft) != "" {
						start("edit")
					}
				}, func() { state.editing = false })
			})
			if state.error != "" {
				ui.Text(c, state.error).FontSize(12).FixedLineHeight(20).TextColor(c.Theme().Danger).SingleLine().Tooltip(state.error).MinWidth(0).Shrink(1)
			}
			button("✓", referenceCopy("Save goal", "保存目标"), "edit", strings.TrimSpace(state.draft) == "")
			cancel := ui.Button(c, "").Label(referenceCopy("Cancel edit", "取消编辑")).Tooltip(referenceCopy("Cancel edit", "取消编辑")).Size(28, 28).Padding(0).Radius(14).BorderWidth(0).Background(ui.Transparent).Disabled(state.pending)
			cancel.Children(func() { ui.Icon(c, messageIconCloseOutline).Size(14, 14) })
			if cancel.Clicked() {
				state.editing = false
			}
			return
		}
		label := referenceCopy("Ongoing Goal", "进行中的目标")
		switch g.Phase {
		case "paused":
			label = referenceCopy("Paused Goal", "已暂停的目标")
		case "blocked":
			label = referenceCopy("Blocked Goal", "受阻的目标")
		default:
			if g.Activation == "disarmed" {
				label = referenceCopy("Inactive Goal", "已暂停的目标")
			}
		}
		ui.Icon(c, goalDockIcons["goal"]).Size(14, 14).TextColor(muted).Shrink(0)
		ui.Text(c, label).FontSize(13).FixedLineHeight(24).FontWeight(500).TextColor(fg).Shrink(0)
		ui.Text(c, g.Objective).FontSize(13).FixedLineHeight(20).TextColor(fg).Grow(1).MinWidth(0).SingleLine().Tooltip(g.Objective)
		if g.Phase == "blocked" && g.BlockedReason.Message != "" {
			root.Tooltip(g.BlockedReason.Message)
		}
		if state.error != "" {
			ui.Text(c, state.error).FontSize(12).FixedLineHeight(20).TextColor(c.Theme().Danger).SingleLine().MinWidth(0).Shrink(1).Tooltip(state.error)
		}
		if g.Phase == "active" && g.Activation == "armed" {
			button("Ⅱ", referenceCopy("Pause goal", "暂停目标"), "pause", false)
		}
		if g.Phase == "paused" || g.Phase == "active" && g.Activation == "disarmed" {
			button("▷", referenceCopy("Resume goal", "恢复目标"), "resume", false)
		}
		edit := ui.Button(c, "").Label(referenceCopy("Edit goal", "编辑目标")).Tooltip(referenceCopy("Edit goal", "编辑目标")).Size(28, 28).Padding(0).Radius(14).BorderWidth(0).Background(ui.Transparent).Disabled(state.pending)
		edit.Children(func() { ui.Icon(c, messageIconEditOutline).Size(14, 14) })
		if edit.Clicked() {
			state.draft = g.Objective
			state.editing = true
			state.error = ""
		}
		button("×", referenceCopy("Clear goal", "清除目标"), "clear", false)
	})
}

var goalDockIcons = map[string]*ui.SVG{
	"✓":    messageIconCheckOutline,
	"Ⅱ":    ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor"><rect x="3.5" y="2.5" width="3" height="11" rx="1"/><rect x="9.5" y="2.5" width="3" height="11" rx="1"/></svg>`)),
	"▷":    ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor"><path d="M4 2.5a.5.5 0 0 1 .8-.4l8 5.5a.5.5 0 0 1 0 .8l-8 5.5a.5.5 0 0 1-.8-.4Z"/></svg>`)),
	"×":    ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor"><path d="M2 4h12M6 2h4M4 4l.7 9.5h6.6L12 4M6.5 6v5M9.5 6v5"/></svg>`)),
	"goal": ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor"><circle cx="8" cy="8" r="6"/><circle cx="8" cy="8" r="3"/><path d="M8 8 14.5 1.5"/></svg>`)),
}

type goalEditorKeys struct{ composing, ended bool }

func (k *goalEditorKeys) observe(composing bool) {
	if k.composing && !composing {
		k.ended = true
	}
	k.composing = composing
}
func (k *goalEditorKeys) handle(ev ui.InputEvent, composing, pending bool, save, cancel func()) bool {
	switch ev.Kind {
	case ui.InputCompose:
		k.composing = ev.Text != ""
		k.ended = !k.composing
		return false
	case ui.InputText:
		if k.composing {
			k.composing = false
			k.ended = true
		}
		return false
	case ui.InputKeyUp:
		k.ended = false
		return false
	}
	if ev.Kind != ui.InputKeyDown || (ev.Key != ui.KeyEnter && ev.Key != ui.KeyEscape) {
		return false
	}
	if pending || composing || k.composing || k.ended {
		return true
	}
	if ev.Key == ui.KeyEnter && ev.Mods&ui.Shift != 0 {
		return false
	}
	if !ev.Repeat {
		if ev.Key == ui.KeyEnter {
			save()
		} else {
			cancel()
		}
	}
	return true
}
