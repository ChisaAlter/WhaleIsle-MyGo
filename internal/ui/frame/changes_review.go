package frame

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/diff"
)

// Only selectors/presentation survive layout persistence. Auth, snapshots,
// in-flight work and native-action outcomes stay in the session Store.
type ChangesReviewState struct {
	Host      string                 `json:"host"`
	SessionID string                 `json:"sessionId"`
	Target    session.TurnFileTarget `json:"target"`
	Split     bool                   `json:"split"`
	Wrap      bool                   `json:"wrap"`
	scope     session.TurnFileScope
	requested map[session.TurnFileRequest]bool
}

func (r *ChangesReviewState) bind(snapshot session.ViewState) bool {
	scope := snapshot.TurnFiles.Scope
	if scope.Binding == "" || scope.SessionID != r.SessionID || snapshot.SelectedID != r.SessionID || snapshot.TurnFiles.HostLabel != r.Host {
		return false
	}
	if r.scope.Binding != "" && r.scope != scope {
		return false
	}
	for _, d := range session.TurnDeliveries(snapshot.Messages) {
		if d.Changes != nil && d.Changes.Turn == r.Target.Turn && d.Changes.Seq == r.Target.Seq && r.Target.Kind == "changes" {
			r.scope = scope
			return true
		}
	}
	return false
}
func (w *WorkspacePanes) OpenChangesReview(snapshot session.ViewState, target session.TurnFileTarget) bool {
	r := &ChangesReviewState{Host: snapshot.TurnFiles.HostLabel, SessionID: snapshot.SelectedID, Target: target, Split: true}
	if !r.bind(snapshot) {
		return false
	}
	if r.Target.Index < 0 {
		r.Target.Index = 0
	}
	if w.Root == nil {
		w.syncRoute("files")
	}
	var found *WorkspacePane
	w.Root.eachLeaf(func(n *WorkspacePane) {
		old := n.ChangesReview
		if found == nil && old != nil && old.Host == r.Host && old.SessionID == r.SessionID && old.Target.Seq == target.Seq && old.Target.Turn == target.Turn {
			found = n
		}
	})
	if found == nil {
		found = w.Root.find(w.Active)
		found.ChangesReview = r
	} else {
		found.ChangesReview.Target.Index = r.Target.Index
		found.ChangesReview.scope = r.scope
		found.ChangesReview.requested = nil
	}
	w.selectKind(found, "changes-review")
	w.focusPending = found.ID
	return true
}
func OpenChangesReview(m Model, scope session.TurnFileScope, target session.TurnFileTarget) {
	if m.Store == nil || m.Sizes == nil {
		return
	}
	snapshot := m.Store.Snapshot()
	if scope != snapshot.TurnFiles.Scope {
		return
	}
	if m.Sizes.Workspace == nil {
		m.Sizes.Workspace = &WorkspacePanes{}
	}
	if !m.Sizes.Workspace.OpenChangesReview(snapshot, target) {
		return
	}
	if m.RightPaneTab != nil {
		*m.RightPaneTab = "changes-review"
	}
	if !m.RightPaneOpen && m.OnToggleTools != nil {
		m.OnToggleTools()
	}
}
func copyChangesReviewPane(target, source *WorkspacePane) {
	if target.Kind != "changes-review" || source.Kind != "changes-review" || source.ChangesReview == nil {
		return
	}
	copy := *source.ChangesReview
	copy.requested = nil
	target.ChangesReview = &copy
}
func changesReviewTitle(m Model) string {
	if m.workspacePane != nil && m.workspacePane.ChangesReview != nil {
		return fmt.Sprintf("%s %d", paneLabel("本轮变更", "Turn changes"), m.workspacePane.ChangesReview.Target.Turn)
	}
	return paneLabel("本轮变更", "Turn changes")
}
func changesReviewPane(c *ui.Context, m Model) {
	if m.Store == nil || m.workspacePane == nil || m.workspacePane.ChangesReview == nil {
		ui.Text(c, paneLabel("从本轮交付卡打开变更", "Open changes from a turn delivery card"))
		return
	}
	r := m.workspacePane.ChangesReview
	snapshot := m.Store.Snapshot()
	if !r.bind(snapshot) {
		ui.Text(c, paneLabel("请返回此变更所属的 Host 与会话并重新打开", "Return to the source Host and session and reopen this review"))
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	st := diff.Style{FontSize: 11, FG: chrome.Color(colors.FG), Muted: chrome.Color(colors.Muted), Hunk: chrome.Color(colors.Accent), Add: ui.RGB(40, 160, 90), Del: ui.RGB(210, 70, 70)}
	scope := r.scope
	run := func(target session.TurnFileTarget, action string) {
		go m.Store.RunTurnFileAction(context.Background(), scope, session.TurnFileRequest{Target: target, Action: action})
	}
	ensure := func(target session.TurnFileTarget, action string) session.TurnFileResult {
		key := session.TurnFileRequest{Target: target, Action: action}
		result, exists := snapshot.TurnFiles.Results[key]
		if r.requested == nil {
			r.requested = map[session.TurnFileRequest]bool{}
		}
		if !exists && !r.requested[key] {
			r.requested[key] = true
			run(target, action)
		}
		return result
	}
	base := r.Target
	base.Index = -1
	summary := ensure(base, "summary")
	host := ensure(base, "host")
	ui.Text(c, fmt.Sprintf("%s · %s · event %d", r.Host, r.SessionID, r.Target.Seq)).FontSize(10).TextColor(st.Muted).Selectable()
	ui.Text(c, paneLabel("本轮起止快照，并非当前文件", "Turn-start/end snapshot, not current file contents")).FontSize(10).TextColor(st.Muted)
	status := func(result session.TurnFileResult, target session.TurnFileTarget, action string) bool {
		if result.Error != "" {
			ui.Text(c, result.Error).TextColor(st.Muted)
			if ui.Button(c, paneLabel("重试", "Retry")).Disabled(result.Busy).Clicked() {
				run(target, action)
			}
			return false
		}
		if result.Busy || !result.Ready {
			ui.Text(c, paneLabel("读取中…", "Loading…")).TextColor(st.Muted)
			return false
		}
		return true
	}
	if !status(summary, base, "summary") || summary.Summary == nil {
		return
	}
	files := summary.Summary.Files
	if len(files) == 0 {
		ui.Text(c, paneLabel("本轮没有文件变更", "No changed files in this turn"))
		return
	}
	if r.Target.Index < 0 || r.Target.Index >= len(files) {
		r.Target.Index = 0
	}
	ui.Row(c).Wrap().Gap(4).Children(func() {
		for i, file := range files {
			if ui.Button(c, fmt.Sprintf("%s +%d −%d", file.Display, file.Added, file.Deleted)).Disabled(i == r.Target.Index).Clicked() {
				r.Target.Index = i
			}
		}
	})
	target := r.Target
	ui.Row(c).Wrap().Gap(4).Children(func() {
		label := paneLabel("左右对比", "Split view")
		if r.Split {
			label = paneLabel("单栏对比", "Unified view")
		}
		if ui.Button(c, label).Clicked() {
			r.Split = !r.Split
		}
		label = paneLabel("自动换行", "Wrap lines")
		if r.Wrap {
			label = paneLabel("取消换行", "No wrap")
		}
		if ui.Button(c, label).Clicked() {
			r.Wrap = !r.Wrap
		}
		if ui.Button(c, paneLabel("打开完整文件", "Open full file")).Disabled(m.OnOpenDeliverable == nil).Clicked() {
			if path, ok := m.Store.ChangesReviewPath(r.scope, target); ok {
				m.OnOpenDeliverable(path)
			}
		}
		native := snapshot.TurnFiles.Results[(session.TurnFileRequest{Target: target, Action: "open"}).StateKey()]
		for _, action := range []string{"open", "reveal"} {
			label := paneLabel("在 Host 打开", "Open on Host")
			if action == "reveal" {
				label = paneLabel("在 Host 定位", "Reveal on Host")
			}
			if ui.Button(c, label).Disabled(host.Host == nil || !host.Host.Available || native.Busy).Clicked() {
				run(target, action)
			}
		}
		if native.Busy {
			ui.Text(c, native.Action+"…")
		}
		if native.Error != "" {
			ui.Text(c, native.Error)
		}
		if native.Ready {
			ui.Text(c, paneLabel("Host 已接受文件动作", "Host accepted file action"))
		}
	})
	if host.Error != "" {
		ui.Text(c, host.Error)
		if ui.Button(c, paneLabel("重试 Host 信息", "Retry Host info")).Clicked() {
			run(base, "host")
		}
	}
	result := ensure(target, "diff")
	if !status(result, target, "diff") || result.Diff == nil {
		return
	}
	content := result.Diff
	if content.Kind != "text" {
		ui.Text(c, content.Kind+paneLabel("：没有文本对比", ": no text comparison"))
		return
	}
	if content.Before != nil && !*content.Before {
		ui.Text(c, paneLabel("本轮新建文件", "File created this turn"))
	}
	if content.After != nil && !*content.After {
		ui.Text(c, paneLabel("本轮删除文件", "File deleted this turn"))
	}
	if content.Coarse != nil && *content.Coarse {
		ui.Text(c, paneLabel("Host 按整文件替换显示", "Host supplied whole-file replacement"))
	}
	patch, truncated, hasAdd, hasDel := reviewPatch(*content)
	if truncated {
		ui.Text(c, paneLabel("仅显示前 5000 行", "Showing the first 5000 lines"))
	}
	if patch == "" {
		ui.Text(c, paneLabel("没有文本差异", "No text changes"))
		return
	}
	ui.Scroll(c).Fill().MinWidth(0).Children(func() { diffsurface.RecordedPatchView(c, patch, r.Split && hasAdd && hasDel, r.Wrap, st) })
}
func reviewPatch(content session.TurnFileDiff) (string, bool, bool, bool) {
	var b strings.Builder
	count := 0
	added, deleted := false, false
	for _, h := range content.Hunks {
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines)
		for _, line := range h.Lines {
			if count == 5000 {
				return b.String(), true, added, deleted
			}
			count++
			added = added || strings.HasPrefix(line, "+")
			deleted = deleted || strings.HasPrefix(line, "-")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String(), false, added, deleted
}
