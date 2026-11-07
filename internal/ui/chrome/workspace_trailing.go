package chrome

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const WorkspaceTrailingHeight float32 = 32
const WorkspaceTrailingTop float32 = 8

// WorkspaceTrailingModel is the shell.titlebar.trailing producer. Session/log
// actions stay with the shell owner (order 10); Git is order 20 and panels 40.
// WorkspaceRoot is the retained main-view Session cwd, not an arbitrary pane.
type WorkspaceTrailingModel struct {
	Git                                           *GitBadge
	Status                                        *git.RepositoryStatus
	WorkspaceRoot, Density                        string
	GitLoaded, GitBusy, ManagedSession, HideGit   bool
	ChangeRequestOpen                             bool
	ChangeRequestShort                            string
	OnGitAction                                   func(action string)
	OnGitMenuOpen                                 func()
	RenderBranches                                func(*ui.Context, ui.Element, *bool)
	RenderGitActivity                             func(*ui.Context)
	ShowTerminal, TerminalAvailable, TerminalOpen bool
	ShowSurfaces, SurfacesOpen                    bool
	OnToggleTerminal, OnToggleSurfaces            func()
}
type captionGitAction struct {
	id, label, icon, hint string
	disabled              bool
}

func captionWord(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}

// WorkspaceTrailing returns a 32px row at y=8 of the 48px control plate. The
// surrounding shell title row is min52/auto; it must not duplicate these controls
// in the conversation row.
// Controllers/dialogs remain owned and mounted outside visibility preferences.
func WorkspaceTrailing(c *ui.Context, m WorkspaceTrailingModel) {
	ui.Row(c.Key("workspace-titlebar-trailing")).Height(WorkspaceTrailingHeight).Shrink(0).AlignItems(ui.Center).Gap(8).Children(func() {
		if !m.HideGit && !m.ManagedSession && m.GitLoaded && strings.TrimSpace(m.WorkspaceRoot) != "" && m.Status != nil {
			captionGit(c, m)
		}
		if m.ShowTerminal || m.ShowSurfaces {
			ui.Row(c).Height(32).Gap(4).Children(func() {
				if m.ShowTerminal {
					captionPanelToggle(c, "terminal", captionWord("切换终端抽屉", "Toggle terminal drawer"), captionWord("终端抽屉不可用", "Terminal drawer unavailable"), "Ctrl+`", m.TerminalOpen, m.TerminalAvailable && m.OnToggleTerminal != nil, m.OnToggleTerminal)
				}
				if m.ShowSurfaces {
					captionPanelToggle(c, "surfaces", captionWord("切换右侧栏", "Toggle right panel"), "", "Ctrl+\\", m.SurfacesOpen, m.OnToggleSurfaces != nil, m.OnToggleSurfaces)
				}
			})
		}
	})
	if !m.ManagedSession && m.RenderGitActivity != nil {
		m.RenderGitActivity(c)
	}
}
func captionPanelToggle(c *ui.Context, icon, label, unavailable, chord string, pressed, enabled bool, run func()) {
	hint := label + " (" + chord + ")"
	if !enabled && unavailable != "" {
		hint = unavailable
	}
	b := ui.ButtonBase(c).Width(32).Height(32).Padding(0).BorderWidth(0).Radius(4).Justify(ui.Center).AlignItems(ui.Center).Background(ui.Transparent).Label(label).Tooltip(hint).Checked(pressed).Disabled(!enabled)
	ink := c.Theme().TextMuted
	if pressed {
		ink = c.Theme().Text
		b.Background(c.Theme().Surface)
	}
	b.Children(func() { ui.Icon(c, workspaceTrailingIcons[icon]).Size(14, 14).TextColor(ink) })
	if b.Clicked() && run != nil {
		run()
	}
}
func captionGit(c *ui.Context, m WorkspaceTrailingModel) {
	if !m.Status.IsRepo {
		a := captionGitAction{id: "init", label: captionWord("初始化 Git", "Initialize Git"), icon: "branch", disabled: m.GitBusy}
		captionGitButton(c, m, a, m.Density != "compact").Radius(14).Border(.5, c.Theme().Border)
		return
	}
	toolbar := ui.Row(c.Key("caption-git-toolbar")).Label("Git toolbar").Height(27).Shrink(0).Gap(0).Border(.5, c.Theme().Border).Radius(14).Clip().AlignItems(ui.Center)
	revealed := ui.Local(toolbar, "revealed-parts", func() captionGitReveal { return captionGitReveal{} })
	if !toolbar.Hovered() {
		revealed.observe(false, false, false)
	}
	toolbar.Children(func() {
		branch := m.Status.Branch
		if branch == "" && m.Git != nil {
			branch = m.Git.Branch
		}
		if branch == "" {
			branch = captionWord("选择分支", "Select branch")
		}
		maxWidth := float32(180)
		if m.Density == "cozy" {
			maxWidth = 120
		}
		branchPart := ui.Row(c.Key("caption-git-branch")).Height(27).Shrink(0).Gap(0)
		branchPart.Children(func() {
			padding := float32(10)
			if m.Density == "compact" {
				padding = 8
			}
			anchor := ui.ButtonBase(c).Height(27).MaxWidth(maxWidth).Padding(0, padding).Gap(0).BorderWidth(0).Radius(0).Background(ui.Transparent).Label(captionWord("切换分支", "Switch branch")).Tooltip(branch).Disabled(m.GitBusy || m.RenderBranches == nil).FocusRing(false)
			branchOpen := ui.Local(anchor, "branch-open", func() bool { return false })
			if m.GitBusy {
				*branchOpen = false
			}
			if anchor.Clicked() {
				*branchOpen = !*branchOpen
			}
			if m.RenderBranches != nil {
				m.RenderBranches(c, anchor, branchOpen)
			}
			anchor.Expanded(*branchOpen)
			revealed.observe(toolbar.Hovered(), branchPart.Hovered(), false)
			shown := revealed.branch || anchor.FocusVisible() || *branchOpen
			anchor.Children(func() {
				ui.Icon(c, workspaceTrailingIcons["branch"]).Size(14, 14).TextColor(c.Theme().Text)
				captionGitLabel(c, anchor, branch, "branch", shown, maxWidth-2*padding-32)
				ui.Icon(c, workspaceTrailingIcons["chevron"]).Size(14, 14).Margin(0, 0, 0, 4).TextColor(c.Theme().Text)
			})
		})
		captionGitRule(c)
		quick := resolveCaptionQuick(m)
		// A disabled button cannot own a toolkit tooltip. Keep its reason on
		// the enabled presentation wrapper, as the reference span does.
		actionPart := ui.Row(c.Key("caption-git-action")).Height(27).Shrink(0).Gap(0).Tooltip(quick.hint)
		actionPart.Children(func() {
			primary := captionGitControl(c, m, quick, true).Gap(0)
			if m.Density == "compact" {
				primary.Padding(0, 8)
			}
			captionGitRule(c)
			trigger := ui.ButtonBase(c).Width(24).Height(27).Padding(0).Justify(ui.Center).AlignItems(ui.Center).BorderWidth(0).Radius(0).Background(ui.Transparent).Label(captionWord("Git 操作选项", "Git action options")).Disabled(m.GitBusy).FocusRing(false)
			trigger.Children(func() { ui.Icon(c, workspaceTrailingIcons["chevron"]).Size(14, 14).TextColor(c.Theme().Text) })
			open := ui.Local(trigger, "menu-open", func() bool { return false })
			if m.GitBusy {
				*open = false
			}
			if trigger.Clicked() {
				*open = !*open
				if *open && m.OnGitMenuOpen != nil {
					m.OnGitMenuOpen()
				}
			}
			ui.PopoverBase(c.Key("caption-git-menu"), trigger, open, func(panel ui.Element) {
				panel = MenuSurface(panel.MinWidth(144).MaxWidth(360).Padding(4).Gap(0).Role(ui.RoleMenu), c.Theme().Surface, c.Theme().Border, 16)
				// The menu fades in over the popover token, as every floating card
				// does: a PopoverBase panel cannot play PopoverTransition (it is born
				// in the frame it opens), so Reveal runs the enter by hand.
				panel = PopoverReveal(c, panel)
				panel.Children(func() {
					for _, action := range captionMenuActions(m) {
						a := action
						hint := ""
						if a.disabled {
							hint = a.hint
						}
						ui.Row(c).FillWidth().Height(36).Shrink(0).Tooltip(hint).Children(func() {
							row := captionGitButton(c, m, a, true).FillWidth().Justify(ui.Start).Height(36).Padding(6, 8).Role(ui.RoleMenuItem).Radius(8)
							if row.Clicked() {
								*open = false
							}
						})
					}
					if m.Status.Branch == "" {
						ui.Text(c, captionWord("当前处于 detached HEAD", "Currently on detached HEAD")).FillWidth().Padding(8, 10).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
					} else if !m.Status.Dirty && m.Status.Behind > 0 && m.Status.Ahead == 0 {
						ui.Text(c, captionWord("分支落后，请先拉取", "Branch is behind; pull first")).FillWidth().Padding(8, 10).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
					}
				})
			})
			trigger.Expanded(*open)
			revealed.observe(toolbar.Hovered(), false, actionPart.Hovered())
			shown := revealed.action || primary.FocusVisible() || trigger.FocusVisible() || *open
			primary.Children(func() {
				ui.Icon(c, workspaceTrailingIcons[quick.icon]).Size(14, 14).TextColor(c.Theme().Text)
				captionGitLabel(c, primary, quick.label, "action", shown, 0)
			})
			if primary.Clicked() {
				captionRun(m, quick.id)
			}
		})
	})
}
func captionGitRule(c *ui.Context) {
	ui.Box(c).Width(.5).Height(16).Shrink(0).Background(c.Theme().Border)
}
func captionGitControl(c *ui.Context, m WorkspaceTrailingModel, a captionGitAction, label bool) ui.Element {
	disabled := a.disabled || !captionCanRun(m, a.id)
	hint := ""
	if a.disabled {
		hint = a.hint
	}
	b := ui.ButtonBase(c).Height(27).Padding(0, 10).Gap(4).BorderWidth(0).Radius(0).Background(ui.Transparent).Label(a.label).Tooltip(hint).Disabled(disabled).AlignItems(ui.Center).Justify(ui.Center).FocusRing(false)
	if !label {
		b.Padding(0, 8)
		if hint == "" {
			b.Tooltip(a.label)
		}
	}
	return b
}
func captionGitButton(c *ui.Context, m WorkspaceTrailingModel, a captionGitAction, label bool) ui.Element {
	b := captionGitControl(c, m, a, label)
	b.Children(func() {
		ui.Icon(c, workspaceTrailingIcons[a.icon]).Size(14, 14).TextColor(c.Theme().Text)
		if label {
			ui.Text(c, a.label).SingleLine().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Text)
		}
	})
	if b.Clicked() {
		captionRun(m, a.id)
	}
	return b
}
func captionCanRun(m WorkspaceTrailingModel, id string) bool {
	if id == "" {
		return false
	}
	if m.OnGitAction != nil {
		return true
	}
	if m.Git == nil {
		return false
	}
	switch id {
	case "commit":
		return m.Git.OnCommit != nil
	case "push":
		return m.Git.OnPush != nil
	case "pull":
		return m.Git.OnPull != nil
	case "open_pr":
		return m.Git.OnPRView != nil
	case "create_pr":
		return m.Git.OnPRCreate != nil
	}
	return false
}
func captionRun(m WorkspaceTrailingModel, id string) {
	if m.OnGitAction != nil {
		m.OnGitAction(id)
		return
	}
	if m.Git == nil {
		return
	}
	var f func()
	switch id {
	case "commit":
		f = m.Git.OnCommit
	case "push":
		f = m.Git.OnPush
	case "pull":
		f = m.Git.OnPull
	case "open_pr":
		f = m.Git.OnPRView
	case "create_pr":
		f = m.Git.OnPRCreate
	}
	if f != nil {
		f()
	}
}
func captionPRShort(m WorkspaceTrailingModel) string {
	if m.ChangeRequestShort != "" {
		return m.ChangeRequestShort
	}
	return "PR"
}
func captionMenuActions(m WorkspaceTrailingModel) []captionGitAction {
	s := m.Status
	items := []captionGitAction{{"commit", "Commit", "commit", captionWord("请先修改工作区文件", "Make changes before committing"), m.GitBusy || !s.Dirty}}
	if s.HasOrigin {
		items = append(items, captionGitAction{"push", "Push", "push", captionWord("需要可推送的本地分支及提交，且分支不能落后", "Requires local commits on a branch that is not behind upstream"), m.GitBusy || !s.CanPush()})
		id, label := "create_pr", "Create "+captionPRShort(m)
		disabled := !s.CanCreatePR()
		if m.ChangeRequestOpen {
			id = "open_pr"
			label = "View " + captionPRShort(m)
			disabled = false
		}
		items = append(items, captionGitAction{id, label, "pr", captionWord("请先提交更改，并确认分支含有待合并提交", "Commit changes and ensure the branch has commits to merge"), m.GitBusy || disabled})
	} else {
		items = append(items, captionGitAction{"publish", "Publish repository", "push", "", m.GitBusy})
	}
	return items
}

// Pinned ui-git/git-logic.ts resolveQuickAction, evaluated from the same retained
// Session's status snapshot. Actions name existing controller review paths.
func resolveCaptionQuick(m WorkspaceTrailingModel) captionGitAction {
	a := captionGitAction{label: "Commit", icon: "commit", disabled: true}
	if m.GitBusy {
		a.hint = captionWord("Git 操作进行中", "Git action in progress")
		return a
	}
	s := m.Status
	if s == nil {
		a.hint = captionWord("Git 状态暂不可用", "Git status unavailable")
		return a
	}
	if s.Branch == "" {
		a.hint = captionWord("请先创建并切换到分支", "Create and check out a branch first")
		return a
	}
	makeAction := func(id, label, icon string) captionGitAction {
		return captionGitAction{id: id, label: label, icon: icon}
	}
	view := func() captionGitAction {
		return makeAction("open_pr", "View "+captionPRShort(m), "pr")
	}
	push := func() captionGitAction {
		if s.AheadUnreliable && s.Ahead == 0 {
			return makeAction("push", "Push", "push")
		}
		if m.ChangeRequestOpen || s.IsDefault {
			id := "push"
			if s.IsDefault {
				id = "commit_push"
			}
			return makeAction(id, "Push", "push")
		}
		return makeAction("create_pr", "Push & create "+captionPRShort(m), "push")
	}
	if s.Dirty {
		if !s.HasUpstream && !s.HasOrigin {
			return makeAction("commit", a.label, "commit")
		}
		if m.ChangeRequestOpen || s.IsDefault {
			return makeAction("commit_push", "Commit & push", "push")
		}
		return makeAction("commit_push_pr", "Commit, push & "+captionPRShort(m), "push")
	}
	if !s.HasUpstream {
		if !s.HasOrigin {
			if m.ChangeRequestOpen && s.Ahead == 0 {
				return view()
			}
			return makeAction("publish", "Publish repository", "push")
		}
		if s.Ahead == 0 && !s.AheadUnreliable {
			if m.ChangeRequestOpen {
				return view()
			}
			a.label = "Push"
			a.icon = "push"
			a.hint = captionWord("没有待推送的本地提交", "No local commits to push")
			return a
		}
		return push()
	}
	if s.Ahead > 0 && s.Behind > 0 {
		a.label = "Sync branch"
		a.icon = "push"
		a.hint = captionWord("分支已分叉，请先合并或变基", "Branch has diverged; merge or rebase first")
		return a
	}
	if s.Behind > 0 {
		return makeAction("pull", "Pull", "push")
	}
	if s.Ahead > 0 || s.AheadUnreliable {
		return push()
	}
	if m.ChangeRequestOpen && s.HasUpstream {
		return view()
	}
	if s.AheadOfDefault > 0 && !s.IsDefault {
		return makeAction("create_pr", "Create "+captionPRShort(m), "push")
	}
	a.hint = captionWord("分支已是最新，无需操作", "Branch is up to date")
	return a
}
