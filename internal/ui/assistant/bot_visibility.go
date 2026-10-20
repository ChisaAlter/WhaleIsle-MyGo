package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

const botVisibilityEffects = "隐藏只改变联系人列表的可见性，可在本块勾选“显示隐藏项”后恢复。不会停用 Bot 或群聊，不会停止会话、目标、例行任务、消息或任务委派，也不等于关闭通知；通知仍按 notify、权限及消费者条件处理。群聊成员、历史和资料保持原样。保存通过 catalog/items 将审阅时完整 items 数组连同 revision 发送当前 Host，只改目标 hidden 与 updatedAt；该原版路由不直接调用 bot/update 的 rename/presentation/requestSync。现有后台工作仍可继续，目录并发变化会拒绝或留下待核实结果。本入口不额外启动、停止或打开会话。"

type BotVisibilityModel struct {
	State                          dshbotstore.BotVisibilitySnapshot
	HostLabel                      string
	OnRefresh, OnConfirm, OnCancel func()
	OnShowHidden                   func(bool)
	OnReview, OnAdopt              func(id, session string)
	OnAcknowledge                  func(bool)
}

func visibilityKind(kind string) string {
	if kind == "room" {
		return i18n.T(i18n.KeyMiscAssistantRoom)
	}
	return "Bot"
}
func visibilityIdentity(v pluginrpc.DshbotBotItem) string {
	return fmt.Sprintf("%s · %s · %s\nID: %s · canonical session: %s · hidden=%t", visibilityKind(v.Kind), v.Name, v.Title, v.ID, v.SessionID, v.Hidden)
}
func botVisibilityReviewText(host string, cmd pluginrpc.DshbotVisibilityCommand) string {
	target, err := pluginrpc.DshbotVisibilityTarget(cmd)
	if err != nil {
		return err.Error()
	}
	keys := []string{}
	for k := range target.Profile {
		if k != "hidden" && k != "updatedAt" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantVisibilityDiff), host, cmd.Expected.Revision, visibilityIdentity(target), target.Hidden, cmd.Hidden, string(target.Profile["updatedAt"]), cmd.UpdatedAt, strings.Join(keys, ", "), len(cmd.Expected.Items))
}
func botVisibilityBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.BotVisibility.OnRefresh == nil {
		return
	}
	botVisibilityView(c, m.BotVisibility, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, border, accent)
}
func botVisibilityView(c *ui.Context, m BotVisibilityModel, blocked bool, fg, muted, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("bot-visibility-management")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantVisibilityTitle)).Bold().TextColor(fg)
		ui.Text(c, "此管理块独立于普通列表；显示隐藏项仅改变本地查看，不会恢复或修改 Host 目录。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).TextColor(muted).Selectable()
		}
		if s.FocusID != "" {
			label := i18n.T(i18n.KeyMiscAssistantTargetAwaitsFreshCatalog)
			if s.Loaded {
				label = i18n.T(i18n.KeyMiscAssistantExactTargetNotFound)
				for _, v := range s.Catalog.Items {
					if v.ID == s.FocusID && v.SessionID == s.FocusSessionID {
						label = i18n.T(i18n.KeyMiscAssistantExactTarget) + visibilityIdentity(v)
						break
					}
				}
			}
			ui.Text(c, label+"\nID: "+s.FocusID+" · canonical session: "+s.FocusSessionID).TextColor(accent).Selectable()
		}
		if s.Result.ID != "" {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantLastConfirmedResult)+visibilityIdentity(s.Result)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		shown := s.ShowHidden
		if ui.Checkbox(c, &shown, i18n.T(i18n.KeyMiscAssistantShowHiddenLocally)).Changed() && m.OnShowHidden != nil {
			m.OnShowHidden(shown)
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReadFullCatalog)).Disabled(blocked || s.Scope == "").Clicked() && m.OnRefresh != nil {
			m.OnRefresh()
		}
		if s.Uncertain {
			ui.Text(c, "原写入保持未知，不能自动或手动重发；读回不等于再次保存。锁仅在本进程保留，重启不是安全重试证明。").TextColor(accent).Selectable()
			if s.Pending != nil {
				ui.Text(c, botVisibilityReviewText(m.HostLabel, *s.Pending)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
			}
			if s.Candidate != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAdoptExactReadback)).Disabled(blocked).Clicked() && m.OnAdopt != nil {
				m.OnAdopt(s.Candidate.ID, s.Candidate.SessionID)
			}
			return
		}
		if !s.Loaded {
			ui.Text(c, "先读取当前 Host 完整目录与写权限；隐藏后的目标也可在此找回。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable4)).TextColor(accent)
		}
		if s.Review != nil {
			r := s.Review
			ui.Column(c.Key("visibility-confirm")).Border(1, accent).Padding(10).Gap(8).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancel)).Disabled(blocked).Clicked() && m.OnCancel != nil {
					m.OnCancel()
				}
				ui.Text(c, botVisibilityReviewText(m.HostLabel, r.Command)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
				ui.Text(c, botVisibilityEffects).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, i18n.T(i18n.KeyMiscAssistantIConfirmThisTargetAndEffects)).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				label := i18n.T(i18n.KeyMiscAssistantConfirmUnhide)
				if r.Command.Hidden {
					label = i18n.T(i18n.KeyMiscAssistantConfirmHide)
				}
				if ui.Button(c, label).Disabled(blocked || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
					m.OnConfirm()
				}
			})
			return
		}
		count := 0
		for _, row := range s.Catalog.Items {
			if row.Hidden && !s.ShowHidden {
				continue
			}
			count++
			v := row
			ui.Column(c.Key("visibility-row-" + v.ID)).Gap(4).Children(func() {
				ui.Text(c, visibilityIdentity(v)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
				label := i18n.T(i18n.KeyMiscAssistantReviewHide)
				if v.Hidden {
					label = i18n.T(i18n.KeyMiscAssistantReviewUnhide)
				}
				if ui.Button(c, label).Disabled(blocked || !s.Writable).Clicked() && m.OnReview != nil {
					m.OnReview(v.ID, v.SessionID)
				}
			})
		}
		if count == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoMatchingItemsShowHiddenToRecover)).TextColor(muted)
		}
	})
}
