package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/paths"
)

var bridgeChangeUI sync.Mutex
var bridgeChangeActive atomic.Bool

// Stage and inspect first without stopping the existing service. Review the
// exact version/impact, then use the existing Host admission and maintenance
// guards for the short lifecycle mutation.
func (s *mainState) changeWhaleBridge(win *mygo.Window, action string) {
	if !bridgeChangeUI.TryLock() {
		return
	}
	bridgeChangeActive.Store(true)
	route := s.updateDownloadRoute()
	s.mu.Lock()
	sup := s.sup
	s.launcherUpdate = resumeText("正在校验鲸桥版本和操作影响；现有实例保持运行…", "Verifying WhaleBridge version and impact; the current instance stays running…")
	s.mu.Unlock()
	generation := 0
	if sup != nil {
		generation = sup.Status().Generation
	}
	scopeValid := func() bool {
		s.mu.Lock()
		valid := s.sup == sup && !s.quitting && !s.launcherImportBusy
		s.mu.Unlock()
		return valid && (sup == nil || sup.Status().Generation == generation)
	}
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		defer func() {
			bridgeChangeActive.Store(false)
			bridgeChangeUI.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		var plan *desktopcomponents.BridgeChange
		var removal *desktopcomponents.BridgeUninstall
		var err error
		var result string
		switch action {
		case "update":
			plan, err = desktopcomponents.PrepareWhaleBridgeUpdate(context.Background(), paths.UserDataDir(), route)
		case "rollback":
			if !desktopcomponents.WhaleBridgeManagementSupported() {
				err = fmt.Errorf("鲸桥组件管理仅支持 Windows x64")
			} else {
				plan, err = desktopcomponents.PrepareWhaleBridgeRollback(paths.UserDataDir())
			}
		case "uninstall":
			removal, err = desktopcomponents.PrepareWhaleBridgeUninstall(context.Background(), paths.UserDataDir())
		case "stop":
		default:
			err = desktopcomponents.ErrUnknownAction
		}
		if err == nil && !scopeValid() {
			err = fmt.Errorf("运行环境已改变，请重新检查组件并确认")
		}
		cancelled := false
		if err == nil && (plan != nil || removal != nil) {
			var approved bool
			approved, err = confirmBridgeManagement(win, action, plan, removal)
			cancelled = err == nil && !approved
		}
		if err == nil && !cancelled && !scopeValid() {
			err = fmt.Errorf("确认期间运行环境已改变，请重新检查组件并确认")
		}
		if err == nil && !cancelled {
			s.mu.Lock()
			busy := s.sup != sup || s.taskRestartPending || s.quitting || s.launcherImportBusy
			if !busy {
				s.taskRestartPending = true
			}
			s.mu.Unlock()
			if busy {
				err = fmt.Errorf("其他生命周期操作正在进行，已保留现有实例和已校验版本")
			} else {
				did := s.withTaskProtection(win, sup, "component", func() error {
					if !scopeValid() {
						err = fmt.Errorf("运行环境已改变，请重新操作")
						return err
					}
					switch {
					case action == "stop":
						err = desktopcomponents.RunAction(paths.UserDataDir(), "whalebridge", "stop")
					case removal != nil:
						result, err = removal.Apply(context.Background())
					default:
						err = plan.Apply(context.Background())
					}
					return err
				})
				s.mu.Lock()
				s.taskRestartPending = false
				s.mu.Unlock()
				if !did && err == nil {
					err = fmt.Errorf("操作未执行或任务保护未完成；请查看启动器提示后重试")
				}
			}
		}
		s.mu.Lock()
		s.launcherUpdate = desktopcomponents.FormatActionError("whalebridge", action, err)
		if cancelled {
			s.launcherUpdate = resumeText("已取消；现有版本、连接和配置保持不变。", "Cancelled; the installed version, connections and configuration are unchanged.")
		} else if err == nil && removal != nil {
			s.launcherUpdate = resumeText("鲸桥 v", "WhaleBridge v") + removal.Version() + resumeText(" 已卸载并断开；供应商配置、账户、用量和聊天记录已保留。", " was uninstalled and disconnected. Provider settings, accounts, usage and chats were retained.")
			if removal.DefaultModel() {
				s.launcherUpdate += resumeText(" 请重新选择默认模型。", " Select a new default model.")
			}
			if result != "" {
				s.launcherUpdate += " " + result
			}
		} else if err == nil && plan != nil {
			s.launcherUpdate = resumeText("鲸桥已切换至 v", "WhaleBridge switched to v") + plan.Version() + resumeText("；用户数据和原运行状态已保留", "; user data and the original running state were retained")
		}
		s.mu.Unlock()
		s.refreshLauncherComponents()
	}()
}

func confirmBridgeManagement(win *mygo.Window, action string, plan *desktopcomponents.BridgeChange, removal *desktopcomponents.BridgeUninstall) (bool, error) {
	if win == nil {
		return false, fmt.Errorf("无法显示组件操作确认；未更改组件")
	}
	title, button, message := "", "", ""
	if removal != nil {
		title = resumeText("确认卸载鲸桥", "Confirm WhaleBridge uninstall")
		button = resumeText("卸载并保留用户数据", "Uninstall and keep user data")
		message = fmt.Sprintf(resumeText("目标：鲸桥 v%s。将停止服务、断开鲸桥渠道并删除程序文件；现有连接会中断。将移除桌面端的鲸桥网关凭据引用。支持外部代理托管的版本还会撤销鲸桥代管的代理接入设置。供应商配置和密钥、账户、用量和聊天记录会保留。官方断开命令如果中途失败，程序恢复不代表全部接入设置已恢复，请检查默认模型和受管代理。", "Target: WhaleBridge v%s. This stops the service, disconnects its channel and removes program files. Existing connections will be interrupted. The desktop WhaleBridge gateway credential reference is removed. Versions with managed external agents also undo their WhaleBridge-managed connection settings. Provider settings and keys, accounts, usage and chats are retained. If the official disconnect command fails partway, restoring program files does not restore every connection setting. Check the default model and managed agents."), removal.Version())
		if removal.DefaultModel() {
			message += "\n\n" + resumeText("当前默认模型使用鲸桥。卸载后必须重新选择默认模型。", "Your current default model uses WhaleBridge. Select a new default model after uninstalling.")
		}
	} else if plan != nil {
		title, button = resumeText("确认切换鲸桥版本", "Confirm WhaleBridge version change"), resumeText("切换版本", "Switch version")
		if action == "rollback" {
			title, button = resumeText("确认回滚鲸桥", "Confirm WhaleBridge rollback"), resumeText("回滚", "Roll back")
		}
		message = fmt.Sprintf(resumeText("目标：鲸桥 v%s。正在运行的服务会先停止再重新启动，连接会暂时中断；原本停止的组件会保持停止。配置和用户数据会保留；切换失败会尝试恢复原版本。", "Target: WhaleBridge v%s. A running service will stop and restart, temporarily interrupting connections. A stopped component stays stopped. Configuration and user data are retained; a failed switch attempts to restore the original version."), plan.Version())
	} else {
		return false, desktopcomponents.ErrUnknownAction
	}
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Title: title, Message: message, Buttons: []string{resumeText("取消", "Cancel"), button}, DefaultButton: 0, CancelButton: 0})
	return err == nil && answer.Button == 1, err
}
