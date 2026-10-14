package usage

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
)

func repairView(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	v := m.Store.Repair()
	st := m.Store.Stats()
	if v.Busy {
		ui.Text(c, usageText("正在修复会话日志：", "Repairing session log: ")+v.SessionID).TextColor(muted)
		ui.Text(c, usageText("请等待结果，暂不能重复修复或重新扫描。", "Please wait; repair and rescan are temporarily unavailable.")).FontSize(11).TextColor(muted)
		return
	}
	if v.Err != "" {
		ui.Text(c, usageText("修复未确认成功：", "Repair success not confirmed: ")+v.SessionID+" · "+v.Err).TextColor(accent)
		ui.Text(c, usageText("已有统计已保留。超时或连接中断不代表文件未改变；请先重新扫描，再按需重新确认修复。不会自动重试。", "Existing statistics retained. A timeout or disconnect does not prove the file is unchanged. Rescan first, then confirm a retry if needed. No automatic retries.")).FontSize(11).TextColor(muted)
	}
	if v.Result != nil {
		r := v.Result
		ui.Text(c, usageText("修复完成：", "Repair complete: ")+v.SessionID+fmt.Sprintf(" · %d events · %d → %d bytes", r.Repaired, r.BytesBefore, r.BytesAfter)).TextColor(fg)
		ui.Text(c, usageText("宿主备份：", "Host backup: ")+r.Backup).FontSize(11).TextColor(muted)
		ui.Text(c, usageText("请重新扫描以更新统计；若失败提示仍存在，请重启 dsh 清除宿主内存状态。此处不提供回滚。", "Rescan to update statistics. If the failure hint persists, restart dsh to clear its in-memory state. No rollback is provided here.")).FontSize(11).TextColor(muted)
	}
	if v.PendingID != "" {
		id := v.PendingID
		ui.Text(c, usageText("确认修复会话：", "Confirm repair for session: ")+id).Bold().TextColor(fg)
		ui.Text(c, usageText("宿主会解码该会话日志、从 0 重新编号事件，创建 .bak-时间戳 原件备份后重写并替换日志文件。这会修改磁盘日志；无预览接口，也不能在此撤销。", "The host will decode this session log, renumber events from zero, create a .bak-timestamp copy, then rewrite and replace the log file. This modifies the on-disk log. No preview endpoint or undo is available here.")).FontSize(12).TextColor(accent)
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, usageText("确认备份并修复此会话", "Confirm backup and repair this session")).Clicked() {
				m.Store.ConfirmRepair(context.Background(), id)
			}
			if ui.Button(c, usageText("取消", "Cancel")).Clicked() {
				m.Store.CancelRepair()
			}
		})
		return
	}
	if st.Overview == nil {
		return
	}
	for _, id := range st.Overview.Coverage.FailedSessionIDs {
		id := id
		if !m.Store.CanRepair(id) {
			continue
		}
		if ui.Button(c, usageText("修复日志：", "Repair log: ")+id).Clicked() {
			m.Store.PrepareRepair(id)
		}
	}
}
