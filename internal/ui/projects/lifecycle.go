package projects

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func restoreProjectOrigin(c *ui.Context, m Model) {
	if m.presentation != nil {
		if origin := m.presentation.Rows[m.presentation.Origin]; origin != nil {
			origin.Focus(c)
		}
	}
}

func projectRenameDialog(c *ui.Context, m Model) {
	open := true
	_, viewportHeight := c.Size()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	close := func() {
		if !m.Busy && m.OnCancelRename != nil {
			m.OnCancelRename()
			restoreProjectOrigin(c, m)
		}
	}
	ui.DialogBase(c.Key("rename/"+m.RenameID), &open, func(back, panel ui.Element) {
		chrome.DialogReveal(c, back, panel)
		panel.Label(tr("重命名项目", "Rename Project")).Width(380).MaxWidthPercent(94).MaxHeight(max(1, viewportHeight-48)).MinHeight(0).Clip().Padding(0, 0, 24, 0).Radius(28).Background(chrome.Color(colors.Field)).Draw(chrome.Glass(chrome.Color(colors.Field), chrome.MaterialPanel, 28)).Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .2)).Children(func() {
			projectModalHeader(c, m, tr("重命名项目", "Rename Project"), close, m.Busy, false)
			available := false
			for _, p := range m.Listing.Projects {
				if p.ID == m.RenameID {
					available = true
				}
			}
			blocked := m.Busy || !m.Listing.Available || !available || m.RenameDraft == nil
			submit := false
			ui.Scroll(c).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(44, viewportHeight-240)).Padding(0, 24).Gap(12).Children(func() {
				if m.RenameDraft != nil {
					input := ui.TextInput(c.Key(m.RenameID), m.RenameDraft).Label(tr("项目名称", "Project name")).FillWidth().MinWidth(0).Height(44).FontSize(theme.InterfaceSize(14)).Disabled(m.Busy)
					first := ui.Local(input, "initial-focus", func() bool { return true })
					if *first {
						input.Focus().SetTextSelection(0, utf8.RuneCountInString(*m.RenameDraft))
						*first = false
					}
					blocked = blocked || strings.TrimSpace(*m.RenameDraft) == "" || utf8.RuneCountInString(strings.TrimSpace(*m.RenameDraft)) > 240
					submit = input.Submitted()
				}
				if !available {
					ui.Text(c, tr("项目不再可用。", "This Project is no longer available.")).FontSize(theme.InterfaceSize(12))
				}
				if m.Error != "" {
					ui.Text(c, m.Error).FontSize(theme.InterfaceSize(12))
				}
			})
			ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Wrap().Padding(20, 24, 0, 24).Gap(8).Children(func() {
				ui.Box(c).Grow(1)
				if ui.Button(c, tr("取消", "Cancel")).MinWidth(72).Disabled(m.Busy).Clicked() {
					close()
				}
				if ui.PrimaryButton(c, tr("保存", "Save")).MinWidth(72).Disabled(blocked).Clicked() || submit && !blocked {
					if m.OnRename != nil {
						m.OnRename(m.RenameID, *m.RenameDraft)
					}
				}
			})
		})
	})
	if !open {
		close()
	}
}

func projectArchiveDialog(c *ui.Context, m Model) {
	open := true
	_, viewportHeight := c.Size()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	close := func() {
		if !m.Busy && m.OnCancelArchive != nil {
			m.OnCancelArchive()
			restoreProjectOrigin(c, m)
		}
	}
	ui.DialogBase(c.Key("archive/"+m.ArchiveCandidate.ID), &open, func(back, panel ui.Element) {
		chrome.DialogReveal(c, back, panel)
		panel.Label(tr("停止并归档项目", "Stop work and archive Project")).Width(380).MaxWidthPercent(94).MaxHeight(max(1, viewportHeight-48)).MinHeight(0).Clip().Padding(0, 0, 24, 0).Radius(28).Background(chrome.Color(colors.Field)).Draw(chrome.Glass(chrome.Color(colors.Field), chrome.MaterialPanel, 28)).Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .2)).Children(func() {
			first := ui.Local(panel, "archive-initial-focus", func() bool { return true })
			projectModalHeader(c, m, tr("停止并归档项目", "Stop work and archive Project"), close, m.Busy, *first)
			*first = false
			ui.Scroll(c).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(44, viewportHeight-240)).Padding(0, 24).Gap(12).Children(func() {
				archiveConfirmation(c, m)
				if m.Error != "" {
					ui.Text(c, m.Error).FontSize(theme.InterfaceSize(12))
				}
			})
			ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Wrap().Padding(20, 24, 0, 24).Gap(8).Children(func() {
				ui.Box(c).Grow(1)
				if ui.Button(c, tr("取消", "Cancel")).Disabled(m.Busy).Clicked() {
					close()
				}
				if ui.Button(c, tr("停止工作并归档", "Stop work and archive")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnConfirmArchive != nil {
					m.OnConfirmArchive()
				}
			})
		})
	})
	if !open {
		close()
	}
}

func archiveConfirmation(c *ui.Context, m Model) {
	p := m.ArchiveCandidate
	ui.Text(c, p.Title).FontSize(theme.InterfaceSize(14)).FontWeight(500)
	ui.Text(c, tr("后台工作将停止。工作目录、对话和成果将保留。", "Background work will stop. Files, conversation and results are retained.")).FontSize(theme.InterfaceSize(14))
	if p.Activity.State == "" {
		ui.Text(c, tr("Host 未提供完整活动信息；确认后将尝试停止此项目的工作。", "Host activity details are incomplete; confirming archive will attempt to stop this Project’s work.")).FontSize(theme.InterfaceSize(12))
	} else {
		ui.Text(c, fmt.Sprintf(tr("Lead 运行：%t；成员运行：%d；排队：%d；准备中：%d；状态：%s", "Lead running: %t; members running: %d; queued: %d; preparing: %d; state: %s"), p.Activity.CoordinatorRunning, p.Activity.Running, p.Activity.Queued, p.Activity.Provisioning, p.Activity.State)).FontSize(theme.InterfaceSize(12))
	}
}
