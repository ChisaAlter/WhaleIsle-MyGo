package assistant

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/ui/theme"
)

type ProactiveModel struct {
	Open, Busy                                             bool
	Error, LoadError                                       string
	Title, Description, Selected                           string
	State                                                  proactive.Snapshot
	OnToggle, OnCreate                                     func()
	OnEdit                                                 func(string, string)
	OnSelect, OnCancel                                     func(string)
	OnDecide                                               func(string, string)
	CanRun                                                 bool
	WorkingDirectory                                       string
	OnRun                                                  func(string, bool)
	OnInspect, OnComplete, OnOpenSession, OnCancelSchedule func(string)
	OnSchedule                                             func(string, string, string)
	Autonomy                                               int
	OnAutonomy                                             func(int)
	SourceRepository, SourceError                          string
	SourceBusy                                             bool
	OnSourceEdit                                           func(string)
	OnSourceAdd                                            func()
	OnSourceToggle                                         func(string, bool)
	OnSourceRefresh                                        func(string)
	OnNoticeOpen                                           func(proactive.Notice)
	OnNoticeRead, OnNoticeRetry                            func(string)
	DeliveryRoutes                                         []proactive.DeliveryRoute
	DeliveryError                                          string
	DeliveryBusy                                           bool
	OnDeliveryRefresh                                      func()
	OnDelivery                                             func(string, string, string)
}

func proactiveState(value string) string {
	zh := map[string]string{"discovered": "待处理", "triaged": "已整理", "queued": "等待执行", "planning": "规划中", "running": "执行中", "waiting_input": "需要补充", "waiting_approval": "等待批准", "resumed": "已批准，待继续", "verifying": "待核对结果", "delivered": "已交付", "failed": "未完成", "cancelled": "已取消", "scheduled": "已安排", "fired": "已触发"}
	if label := zh[value]; label != "" {
		return whaleTaskCopy(label, value)
	}
	return value
}

func proactiveValue(raw json.RawMessage) string {
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	return string(raw)
}

func proactiveReason(reason string) string {
	if reason == "Tool execution requires approval" {
		return whaleTaskCopy("工具操作等待你批准", reason)
	}
	zh := map[string]string{"Human approval required": "等待你批准此操作", "Approval granted": "你已批准此操作", "Approval rejected": "你已拒绝此操作", "Cancelled in task inbox": "你已取消此任务", "Execution authorized": "你已授权本次执行", "Result accepted by user": "你已确认任务完成", "Stopped by user": "你已停止执行"}
	if label := zh[reason]; label != "" {
		return whaleTaskCopy(label, reason)
	}
	return reason
}

func proactiveRisk(risk, class string) string {
	if class == "task-result" {
		class = whaleTaskCopy("任务结果", "Task result")
	}
	riskZH := map[string]string{"low": "低", "medium": "中", "high": "高", "review-required": "请核对具体操作"}
	classZH := map[string]string{"non-sensitive": "非敏感资料", "regular": "常规资料", "sensitive": "敏感资料", "tool-arguments": "工具参数"}
	if v := riskZH[risk]; v != "" {
		risk = whaleTaskCopy(v, risk)
	}
	if v := classZH[class]; v != "" {
		class = whaleTaskCopy(v, class)
	}
	return whaleTaskCopy("风险：", "Risk: ") + risk + " · " + class
}

func proactiveToolSource(source string) string {
	switch source {
	case "desktop:acp":
		return whaleTaskCopy("来源：外部 ACP Agent（desktop:acp）", "Source: external ACP agent (desktop:acp)")
	case "plugin:unclassified":
		return whaleTaskCopy("来源：未分类扩展工具（plugin:unclassified）", "Source: unclassified extension tool (plugin:unclassified)")
	case "harness:fs", "harness:fs-search":
		return whaleTaskCopy("来源：内置文件工具（"+source+"）", "Source: built-in file tool ("+source+")")
	default:
		if source != "" {
			return whaleTaskCopy("来源："+source, "Source: "+source)
		}
		return ""
	}
}

func proactiveBlock(c *ui.Context, p ProactiveModel, fg, muted, border, accent ui.Color) {
	ui.Column(c.Key("proactive-inbox")).FillWidth().MinWidth(0).Gap(12).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, whaleTaskCopy("任务收件箱", "Task inbox")).FontSize(theme.InterfaceSize(18)).FontWeight(600).TextColor(fg).Grow(1)
			if p.OnToggle != nil && ui.Button(c, whaleTaskCopy("返回助理", "Back to assistant")).Clicked() {
				p.OnToggle()
			}
		})
		if p.LoadError != "" {
			ui.Text(c, whaleTaskCopy("无法读取任务记录，原文件已保留。", "Task records could not be loaded. The original file is preserved.")).TextColor(accent)
			ui.Text(c, p.LoadError).TextColor(muted).Selectable()
			return
		}
		if p.Error != "" {
			ui.Text(c, p.Error).TextColor(accent).Selectable()
		}
		proactiveNotices(c, p, fg, muted, border, accent)
		proactiveSources(c, p, fg, muted, border, accent)
		pending := false
		for _, a := range p.State.Outbox {
			if a.Status != "pending" {
				continue
			}
			if !pending {
				ui.Text(c, whaleTaskCopy("需要你决定", "Your decision")).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(fg)
				pending = true
			}
			ui.Column(c.Key("approval-"+a.ID)).FillWidth().MinWidth(0).Border(.5, border).Radius(12).Padding(12).Gap(8).Children(func() {
				for _, t := range p.State.Tasks {
					if t.ID == a.TaskID {
						ui.Text(c, t.Title).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
						break
					}
				}
				action := a.Action
				if a.Delivery != nil {
					action = whaleTaskCopy("发送到飞书会话", "Send to Feishu")
				}
				ui.Text(c, action).FontSize(theme.InterfaceSize(13)).TextColor(fg).Selectable()
				if a.Delivery != nil {
					proactiveDeliveryPreview(c, *a.Delivery, fg, muted)
				} else if a.ToolCall != nil {
					if label := proactiveToolSource(a.ToolCall.Source); label != "" {
						ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
					var pretty strings.Builder
					var value any
					decoder := json.NewDecoder(strings.NewReader(string(a.ToolCall.Arguments)))
					decoder.UseNumber()
					if decoder.Decode(&value) == nil {
						raw, _ := json.MarshalIndent(value, "", "  ")
						pretty.Write(raw)
					} else {
						pretty.Write(a.ToolCall.Arguments)
					}
					ui.Text(c, pretty.String()).FontSize(theme.InterfaceSize(13)).TextColor(fg).Selectable()
					ui.Text(c, whaleTaskCopy("批准仅适用于这次工具调用，现有权限检查仍然生效。", "Approval applies to this tool call only; existing permission checks still apply.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				} else if len(a.Target) > 0 && string(a.Target) != "null" {
					ui.Text(c, proactiveValue(a.Target)).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
				}
				ui.Text(c, proactiveRisk(a.Risk, a.DataClass)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				ui.Row(c).Gap(8).Wrap().Children(func() {
					if ui.Button(c, whaleTaskCopy("批准此操作", "Approve action")).Disabled(p.Busy || p.OnDecide == nil).Clicked() {
						p.OnDecide(a.ID, "approved")
					}
					if ui.Button(c, whaleTaskCopy("拒绝", "Reject")).Disabled(p.Busy || p.OnDecide == nil).Clicked() {
						p.OnDecide(a.ID, "rejected")
					}
				})
			})
		}
		form := ui.Column(c.Key("proactive-create")).FillWidth().MinWidth(0).Gap(8)
		creating := ui.Local(form, "open", func() bool { return len(p.State.Tasks) == 0 || p.Title != "" })
		seen := ui.Local(form, "count", func() int { return len(p.State.Tasks) })
		if len(p.State.Tasks) > *seen && p.Title == "" && !p.Busy {
			*creating = false
		}
		*seen = len(p.State.Tasks)
		form.Children(func() {
			ui.Row(c).Children(func() {
				if ui.Button(c, whaleTaskCopy("记录任务", "Add task")).Clicked() {
					*creating = !*creating
				}
			})
			if !*creating {
				return
			}
			for _, field := range []struct{ key, zh, en, value string }{{"title", "任务", "Task", p.Title}, {"description", "目标与完成条件", "Goal and completion criteria", p.Description}} {
				value := field.value
				label := whaleTaskCopy(field.zh, field.en)
				ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				ui.TextArea(c.Key("proactive-"+field.key), &value).Label(label).FillWidth().MinWidth(0).Lines(1, 4).Disabled(p.Busy).OnChange(func() {
					if p.OnEdit != nil {
						p.OnEdit(field.key, value)
					}
				})
			}
			ui.Text(c, whaleTaskCopy("自主级别", "Autonomy")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			ui.Row(c).Gap(6).Wrap().Children(func() {
				for level := 1; level <= 5; level++ {
					if ui.Button(c, fmt.Sprintf("L%d", level)).Disabled(p.Busy || p.OnAutonomy == nil || level == p.Autonomy).Clicked() {
						p.OnAutonomy(level)
					}
				}
			})
			ui.Text(c, whaleTaskCopy("L1 只读；L2–L5 可在工作目录内修改。级别不改变现有工具审批规则。", "L1 is read-only; L2–L5 can edit within the working directory. The level does not change existing tool approval rules.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if ui.Button(c, whaleTaskCopy("加入收件箱", "Add to inbox")).Disabled(p.Busy || strings.TrimSpace(p.Title) == "" || p.OnCreate == nil).Clicked() {
				p.OnCreate()
			}
		})
		if len(p.State.Tasks) == 0 {
			ui.Text(c, whaleTaskCopy("把需要跟进的事记在这里。任务进展和待批准的操作会保留在此处。", "Keep work to follow up here, along with its progress and approval requests.")).FontSize(theme.InterfaceSize(13)).TextColor(muted)
			return
		}
		for i := len(p.State.Tasks) - 1; i >= 0; i-- {
			t := p.State.Tasks[i]
			ui.Column(c.Key("proactive-task-"+t.ID)).FillWidth().MinWidth(0).Gap(6).Padding(10, 0).Children(func() {
				ui.Row(c).FillWidth().Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
					ui.Text(c, t.Title).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg).Grow(1)
					ui.Text(c, proactiveState(t.State)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					if ui.Button(c, whaleTaskCopy("详情", "Details")).Disabled(p.OnSelect == nil).Clicked() {
						if p.Selected == t.ID {
							p.OnSelect("")
						} else {
							p.OnSelect(t.ID)
						}
					}
				})
				if p.Selected == t.ID {
					if t.Description != "" {
						ui.Text(c, t.Description).FontSize(theme.InterfaceSize(13)).TextColor(fg).Selectable()
					}
					ui.Text(c, fmt.Sprintf("%s · L%d", t.Source, t.Autonomy)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					if t.SourceURL != "" {
						ui.Text(c, t.SourceURL).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
					}
					if t.SourceItem != nil {
						state := whaleTaskCopy("来源仍开放", "Source is open")
						if t.SourceItem.State == "closed" {
							state = whaleTaskCopy("来源已关闭", "Source is closed")
						}
						ui.Text(c, state+" · "+t.SourceItem.Kind).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						if t.Execution != nil && (t.Title != t.SourceItem.Title || t.Description != t.SourceItem.Description) {
							ui.Text(c, whaleTaskCopy("来源已更新，本次执行仍保留启动时内容。", "The source changed; this execution keeps the content it started with.")).FontSize(theme.InterfaceSize(12)).TextColor(accent)
							ui.Text(c, t.SourceItem.Title+"\n"+t.SourceItem.Description).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
						}
					}
					if t.LastReason != "" {
						ui.Text(c, proactiveReason(t.LastReason)).TextColor(muted).Selectable()
					}
					proactiveExecution(c, p, t, fg, muted, border)
					for _, evidence := range t.LastEvidence {
						ui.Text(c, proactiveValue(evidence)).TextColor(muted).Selectable()
					}
					for _, a := range p.State.Outbox {
						if a.TaskID == t.ID && a.Status != "pending" {
							status := a.Status
							switch status {
							case "approved":
								status = whaleTaskCopy("已批准", "Approved")
							case "rejected":
								status = whaleTaskCopy("已拒绝", "Rejected")
							case "cancelled":
								status = whaleTaskCopy("已取消", "Cancelled")
							}
							action := a.Action
							if a.Delivery != nil {
								action = whaleTaskCopy("发送结果到飞书", "Send result to Feishu")
							}
							ui.Text(c, status+" · "+action).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
							if a.Delivery != nil {
								proactiveDeliveryReceipt(c, *a.Delivery, fg, muted)
							}
							if a.ToolCall != nil {
								effect := map[string]string{"pending": "等待执行", "ready": "等待执行", "executing": "执行已发起，等待回执", "succeeded": "执行成功", "failed": "执行失败", "unknown": "执行结果未知，不会自动重试", "cancelled": "执行已取消", "denied": "执行被拒绝"}[a.Effect]
								ui.Text(c, whaleTaskCopy(effect, a.Effect)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
								if a.EffectError != "" {
									ui.Text(c, a.EffectError).TextColor(muted).Selectable()
								}
							}
							if a.Note != "" {
								ui.Text(c, a.Note).TextColor(muted).Selectable()
							}
						}
					}
					for _, j := range p.State.Schedules {
						if j.TaskID == t.ID {
							at := j.At
							if parsed, err := time.Parse(time.RFC3339Nano, j.At); err == nil {
								at = parsed.Local().Format("2006-01-02 15:04")
							}
							ui.Row(c).Gap(8).Wrap().Children(func() {
								ui.Text(c, proactiveState(j.Status)+" · "+at).FontSize(theme.InterfaceSize(12)).TextColor(muted)
								if j.IntervalMS != nil {
									ui.Text(c, fmt.Sprintf(whaleTaskCopy("每 %d 分钟", "Every %d min"), *j.IntervalMS/60000)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
								}
								if j.Status == "scheduled" && p.OnCancelSchedule != nil && ui.Button(c, whaleTaskCopy("取消安排", "Cancel schedule")).Disabled(p.Busy).Clicked() {
									p.OnCancelSchedule(j.ID)
								}
							})
						}
					}
					if !t.Terminal() {
						ui.Row(c).Children(func() {
							if ui.Button(c, whaleTaskCopy("取消任务", "Cancel task")).Disabled(p.Busy || p.OnCancel == nil).Clicked() {
								p.OnCancel(t.ID)
							}
						})
					}
				}
				ui.Box(c).FillWidth().Height(.5).Background(border)
			})
		}
	})
}

func proactiveExecution(c *ui.Context, p ProactiveModel, t proactive.Task, fg, muted, border ui.Color) {
	if t.Execution != nil {
		e := t.Execution
		ui.Text(c, e.Spec.Cwd).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
		if e.Summary != "" {
			ui.Text(c, e.Summary).FontSize(theme.InterfaceSize(13)).TextColor(fg).Selectable()
		}
		if e.Error != "" {
			ui.Text(c, e.Error).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
		}
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if ui.Button(c, whaleTaskCopy("查看执行会话", "Open execution")).Disabled(p.OnOpenSession == nil).Clicked() {
				p.OnOpenSession(e.SessionID)
			}
			if !t.Terminal() && e.Phase == "unknown" && ui.Button(c, whaleTaskCopy("核对执行状态", "Check execution")).Disabled(!p.CanRun || p.OnInspect == nil).Clicked() {
				p.OnInspect(t.ID)
			}
			if !t.Terminal() && !e.Unsettled() && t.ApprovalID == "" {
				if ui.Button(c, whaleTaskCopy("继续任务", "Continue task")).Disabled(p.Busy || !p.CanRun || p.OnRun == nil).Clicked() {
					p.OnRun(t.ID, true)
				}
				if t.State == "verifying" && ui.Button(c, whaleTaskCopy("确认完成", "Accept result")).Disabled(p.Busy || p.OnComplete == nil).Clicked() {
					p.OnComplete(t.ID)
				}
			}
		})
		proactiveDeliveryForm(c, p, t, fg, muted, border)
	}
	if t.Terminal() || t.ApprovalID != "" || t.Execution != nil {
		return
	}
	ui.Text(c, whaleTaskCopy("工作目录：", "Working directory: ")+p.WorkingDirectory).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
	if t.Autonomy == 1 {
		ui.Text(c, whaleTaskCopy("L1：只读执行，工具权限请求仍需你确认。", "L1: read-only execution; tool permission requests still need your decision.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	ui.Row(c).Gap(8).Wrap().Children(func() {
		if ui.Button(c, whaleTaskCopy("开始执行", "Start task")).Disabled(p.Busy || !p.CanRun || p.OnRun == nil).Clicked() {
			p.OnRun(t.ID, false)
		}
	})
	form := ui.Column(c.Key("schedule-" + t.ID)).FillWidth().MinWidth(0).Gap(6)
	open := ui.Local(form, "open", func() bool { return false })
	at := ui.Local(form, "at", func() string { return time.Now().Add(time.Hour).Format("2006-01-02 15:04") })
	repeat := ui.Local(form, "repeat", func() string { return "" })
	form.Children(func() {
		ui.Row(c).Children(func() {
			if ui.Button(c, whaleTaskCopy("安排执行时间", "Schedule task")).Clicked() {
				*open = !*open
			}
		})
		if !*open {
			return
		}
		ui.Text(c, whaleTaskCopy("执行时间（本地时间，年-月-日 时:分）", "Run at (local time, YYYY-MM-DD HH:MM)")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		ui.TextInput(c.Key("schedule-at"), at).Label(whaleTaskCopy("执行时间", "Run at")).FillWidth().Height(36).Radius(8).Border(.5, border).Disabled(p.Busy)
		ui.Text(c, whaleTaskCopy("重复间隔（分钟，留空只执行一次）", "Repeat interval (minutes, leave blank for once)")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		ui.TextInput(c.Key("schedule-repeat"), repeat).Label(whaleTaskCopy("重复间隔", "Repeat interval")).FillWidth().Height(36).Radius(8).Border(.5, border).Disabled(p.Busy)
		ui.Text(c, whaleTaskCopy("应用与 Host 就绪时执行。错过的多次安排合并为一次，每次结果单独保留。", "Runs while the app and Host are ready. Missed occurrences coalesce into one run; each result is kept separately.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		ui.Row(c).Children(func() {
			if ui.Button(c, whaleTaskCopy("保存执行安排", "Save schedule")).Disabled(p.Busy || !p.CanRun || p.OnSchedule == nil).Clicked() {
				p.OnSchedule(t.ID, *at, *repeat)
			}
		})
	})
}
