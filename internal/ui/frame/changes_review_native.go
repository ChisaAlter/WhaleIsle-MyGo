package frame

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets"
)

// changesReviewNativeMenu draws the Host file-actions button and its menu. It
// returns the failure the last action reported, for the pane to write beside
// the diff: a failure is never a toast.
func changesReviewNativeMenu(c *ui.Context, target session.TurnFileTarget, snapshot session.TurnFilesSnapshot, enabled bool, ensure func(session.TurnFileTarget, string) session.TurnFileResult, run func(session.TurnFileTarget, string)) string {
	state := snapshot.Results[(session.TurnFileRequest{Target: target, Action: "open"}).StateKey()]
	enabled = enabled && !state.NativeUnavailable
	apps := session.TurnFileResult{}
	if enabled {
		apps = ensure(target, "applications")
	}
	button := reviewToolbarButton(c.Key(fmt.Sprintf("review-host-%s-%d-%v", snapshot.Scope.Binding, snapshot.Scope.Generation, target)), paneLabel("Host 文件操作", "Host file actions"), conversationHeaderIcons["EllipsisOutline"])
	// A handed-to-Host outcome reads as a muted caption beside the button that
	// caused it, fading after a few seconds; a failure stays on widgets.Alert
	// where the caller writes it beside the diff.
	outcome := ""
	if state.Error == "" && state.Ready && !state.Busy && !state.Unavailable && !state.NativeUnavailable {
		outcome = paneLabel("已交给 Host：", "Handed to Host: ") + state.Action
	}
	if widgets.AckLive(c, "changes-review-host", outcome, widgets.StatusDuration) {
		ui.Text(c, outcome).FontSize(11).TextColor(c.Theme().TextMuted)
	}
	button.Menu(func(menu *ui.Menu) {
		for _, action := range []struct{ key, label string }{{"open", paneLabel("在 Host 打开", "Open on Host")}, {"reveal", paneLabel("在 Host 定位", "Reveal on Host")}} {
			if menu.Item(action.label).Disabled(!enabled || state.Busy).Chosen() {
				run(target, action.key)
			}
		}
		menu.Submenu(paneLabel("打开方式", "Open with"), func(sub *ui.Menu) {
			if apps.Busy {
				sub.Item(paneLabel("正在读取应用…", "Loading applications…")).Disabled(true)
			}
			if apps.Error != "" {
				sub.Item(apps.Error).Disabled(true)
			}
			for _, app := range apps.Applications {
				caption := strings.TrimSpace(app.Name)
				if caption == "" {
					caption = app.ID
				}
				if app.Default {
					caption += paneLabel("（默认）", " (default)")
				}
				if sub.Item(caption).Disabled(!enabled || state.Busy || apps.Busy).Chosen() {
					run(target, session.TurnFileApplicationAction(app.ID))
				}
			}
			sub.Separator()
			if sub.Item(paneLabel("刷新应用列表", "Refresh applications")).Disabled(!enabled || apps.Busy).Chosen() {
				run(target, "applications")
			}
		})
		if state.Busy {
			menu.Item(paneLabel("正在交给 Host…", "Handing to Host…")).Disabled(true)
		}
		if state.Error != "" {
			menu.Separator()
			menu.Item(state.Error).Disabled(true)
		}
	})
	return state.Error
}
