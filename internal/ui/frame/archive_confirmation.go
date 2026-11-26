package frame

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func archiveText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func archiveConfirmation(c *ui.Context, m *Model) {
	if m.Store == nil {
		return
	}
	store := m.Store
	request := store.ArchiveConfirmation()
	if request == nil {
		return
	}
	t := tk(c)
	open := true
	title := archiveText("停止并归档此会话？", "Stop and archive this session?")
	desc := fmt.Sprintf(archiveText("“%s”仍有正在进行的工作。归档会先停止这些工作；之后可在侧栏筛选“全部对话（显示已归档）”中恢复会话，被停止的工作不会自动继续。", "“%s” still has work in progress. Archiving first stops that work; the session can later be restored from the “All conversations (show archived)” filter in the sidebar, but stopped work does not resume automatically."), request.Title)
	confirmLabel := archiveText("停止并归档", "Stop and archive")
	if request.Busy {
		confirmLabel = archiveText("正在停止并归档…", "Stopping and archiving…")
	}
	token := request.Token
	busy := request.Busy
	coss.AlertDialog(c, &open, coss.DialogOptions{
		Title:           title,
		Description:     desc,
		MaxWidth:        480,
		DismissDisabled: busy,
		FooterVariant:   coss.FooterBare,
		Footer: func() {
			if coss.Button(c, archiveText("取消", "Cancel"), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Disabled: busy,
			}).Clicked() {
				store.CancelArchive(token)
			}
			if coss.Button(c, confirmLabel, coss.ButtonOptions{
				Variant:  coss.ButtonDestructive,
				Disabled: busy,
			}).Clicked() {
				store.ConfirmArchive(context.Background(), token)
			}
		},
	}, func() {
		ui.Column(c).FillWidth().Gap(design.Sp(2)).Children(func() {
			frameText(c, archiveText("将被停止的工作", "Work that will be stopped"), design.TextXS, design.WeightMedium, t.Foreground)
			for _, activity := range request.Activity {
				frameText(c, archiveActivityLine(activity), design.TextXS, design.WeightNormal, t.Foreground)
			}
			if request.Error != "" {
				coss.Alert(c, "", request.Error, coss.AlertOptions{Variant: coss.AlertError})
			}
		})
	})
	if !open && !busy {
		store.CancelArchive(token)
	}
}

func archiveActivityLine(activity session.ArchiveActivity) string {
	names := make([]string, 0, len(activity.Items))
	for _, item := range activity.Items {
		name := item.Label
		if name == "" {
			name = item.ID
		}
		names = append(names, name)
	}
	kind := activity.Kind
	switch kind {
	case "turn":
		return archiveText("进行中的回合", "Turn in progress")
	case "subagent":
		return fmt.Sprintf(archiveText("%d 个运行中的子智能体：%s", "%d running subagents: %s"), len(names), strings.Join(names, ", "))
	case "job":
		return fmt.Sprintf(archiveText("%d 个后台任务：%s", "%d background jobs: %s"), len(names), strings.Join(names, ", "))
	case "schedule":
		return fmt.Sprintf(archiveText("%d 条定时提醒：%s", "%d scheduled reminders: %s"), len(names), strings.Join(names, ", "))
	}
	return fmt.Sprintf(archiveText("%d 项其他工作（%s）", "%d other work items (%s)"), len(names), kind)
}
