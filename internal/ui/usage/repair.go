package usage

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func repairView(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	v := m.Store.Repair()
	if v.Busy {
		if v.Rescanning {
			ui.Text(c, usageText("正在强制重扫，验证修复后的统计…", "Forcing a rescan to verify post-repair statistics…")).TextColor(muted)
		} else {
			ui.Text(c, usageText("正在修复会话日志：", "Repairing session log: ")+v.SessionID).TextColor(muted)
			if v.StopRequested {
				ui.Text(c, usageText("已停止后续排队；等待当前请求的真实结果，再重扫。已发出的宿主修复不能撤销。", "Queue stopped; waiting for the current request’s actual result, then rescanning. A submitted host repair cannot be undone.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
			} else if widgets.Button(c, colors, usageText("停止后续修复", "Stop queued repairs"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
				m.Store.StopRepairQueue()
			}
		}
	}
	if len(v.Outcomes) > 0 {
		succeeded, events := 0, int64(0)
		for _, o := range v.Outcomes {
			if o.Result != nil {
				succeeded++
				events += o.Result.Repaired
			}
		}
		ui.Text(c, fmt.Sprintf(usageText("修复结果：已确认 %d 个（共 %d 个）请求成功，%d 个事件；目标 %d 个会话。", "Repair results: %d of %d attempted requests confirmed, %d events; %d target sessions."), succeeded, len(v.Outcomes), events, len(v.Targets))).TextColor(fg)
		for _, o := range v.Outcomes {
			if o.Result != nil {
				r := o.Result
				ui.Text(c, o.SessionID+fmt.Sprintf(" · %d events · %d → %d bytes", r.Repaired, r.BytesBefore, r.BytesAfter)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
				ui.Text(c, usageText("宿主备份：", "Host backup: ")+r.Backup).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			} else {
				ui.Text(c, usageText("未确认成功：", "Success not confirmed: ")+o.SessionID+" · "+o.Err).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
			}
		}
	}
	if v.Err != "" {
		if len(v.Outcomes) == 0 {
			ui.Text(c, v.Err).TextColor(accent)
		}
		ui.Text(c, usageText("失败请求未自动重试。超时或断线不代表宿主没有完成写入；只按重扫结果决定下一步。", "Failed requests are not retried automatically. Timeout or disconnect does not prove the host did not finish writing; use the rescan result to decide what to do next.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
	}
	if len(v.Unattempted) > 0 && (!v.Busy || v.StopRequested) {
		ui.Text(c, usageText("尚未发出修复：", "Repairs not submitted: ")+strings.Join(v.Unattempted, ", ")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if v.RescanErr != "" {
		ui.Text(c, usageText("修复请求结果已保留，但重扫失败，旧统计仍在显示：", "Repair request results retained, but verification failed; previous statistics remain visible: ")+v.RescanErr).TextColor(accent)
		if !v.Busy && widgets.Button(c, colors, usageText("重新扫描以验证", "Rescan to verify"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			m.Store.RefreshStats(context.Background(), true)
		}
	} else if v.RescanComplete {
		if v.RescanStale {
			ui.Text(c, usageText("重扫返回了过期缓存，尚不能确认修复后的完整状态，请再重扫。", "The rescan returned stale data; post-repair state is not yet confirmed. Rescan again.")).TextColor(accent)
		} else if len(v.RemainingFailed) > 0 {
			ui.Text(c, usageText("重扫后仍有失败日志：", "Failed logs remain after the rescan: ")+strings.Join(v.RemainingFailed, ", ")).TextColor(accent)
			ui.Text(c, usageText("已确认修复的日志不会自动重复写入；若失败提示仍在，请重启 dsh 清理宿主内存状态后再扫描。", "Confirmed repairs will not be written again automatically. If failures persist, restart dsh to clear its in-memory state, then rescan.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else {
			ui.Text(c, usageText("重扫未发现失败日志；每条修复请求的结果如上。其他统计覆盖情况见下方。", "The rescan found no failed logs; individual repair results are shown above. See other coverage details below.")).TextColor(fg)
		}
	}
	if v.Busy {
		return
	}
	if len(v.PendingIDs) > 0 {
		ids := v.PendingIDs
		ui.Text(c, fmt.Sprintf(usageText("确认修复以下 %d 个失败会话：", "Confirm repair of these %d failed sessions:"), len(ids))).Bold().TextColor(fg)
		ui.Text(c, strings.Join(ids, ", ")).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		ui.Text(c, usageText("宿主将逐条解码日志、从 0 重新编号事件、创建 .bak-时间戳 备份后替换文件。这会修改磁盘日志，无预览和撤销接口。完成后自动强制重扫；失败时停止后续排队。切换 Host 或停止排队不能撤销已发出的修复。", "The host will sequentially decode logs, renumber events from zero, save .bak-timestamp backups, and replace files. This modifies on-disk logs; no preview or undo endpoint exists. A forced rescan follows; a failure stops queued repairs. Switching hosts or stopping the queue cannot undo submitted repairs.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
		ui.Row(c).Gap(6).Children(func() {
			if widgets.Button(c, colors, usageText("确认备份、修复并重扫", "Confirm backup, repair and rescan"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Clicked() {
				if v.PendingID != "" {
					m.Store.ConfirmRepair(context.Background(), v.PendingID)
				} else {
					m.Store.ConfirmRepairAll(context.Background(), ids)
				}
			}
			if widgets.Button(c, colors, usageText("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
				m.Store.CancelRepair()
			}
		})
		return
	}
	candidates := m.Store.RepairCandidates()
	if len(candidates) > 0 && widgets.Button(c, colors, fmt.Sprintf(usageText("批量修复 %d 个失败日志并重扫", "Repair %d failed logs and rescan"), len(candidates)), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Clicked() {
		m.Store.PrepareRepairAll()
	}
	for _, id := range candidates {
		if ui.Button(c, usageText("修复日志：", "Repair log: ")+id).Clicked() {
			m.Store.PrepareRepair(id)
		}
	}
}
