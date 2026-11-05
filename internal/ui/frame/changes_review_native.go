package frame

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
	"whaleisle/internal/store/session"
)

func changesReviewNativeMenu(c *ui.Context, target session.TurnFileTarget, snapshot session.TurnFilesSnapshot, enabled bool, ensure func(session.TurnFileTarget, string) session.TurnFileResult, run func(session.TurnFileTarget, string)) {
	state := snapshot.Results[(session.TurnFileRequest{Target: target, Action: "open"}).StateKey()]
	enabled = enabled && !state.NativeUnavailable
	apps := session.TurnFileResult{}
	if enabled {
		apps = ensure(target, "applications")
	}
	button := reviewToolbarButton(c.Key(fmt.Sprintf("review-host-%s-%d-%v", snapshot.Scope.Binding, snapshot.Scope.Generation, target)), paneLabel("Host 文件操作", "Host file actions"), conversationHeaderIcons["EllipsisOutline"])
	last := ui.Local(button, "native-feedback", func() string { return "" })
	message := state.Error
	kind := "error"
	if message == "" && state.Ready && !state.Busy && !state.Unavailable && !state.NativeUnavailable {
		message = paneLabel("已交给 Host：", "Handed to Host: ") + state.Action
		kind = "success"
	}
	if message != "" && message != *last {
		c.AddToast(ui.Toast{ID: "changes-review-host", Title: message, Type: kind, Timeout: 5 * time.Second})
	}
	*last = message
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
}
