package main

import (
	"fmt"
	"github.com/egoist/mygo"
	"path/filepath"
	"sync"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

var launcherComponentPending sync.Map

type launcherComponentActivity struct {
	Pending map[string]string
	Status  string
}

func (s *mainState) launcherComponentActivitySnapshot() launcherComponentActivity {
	activity := launcherComponentActivity{Pending: map[string]string{}}
	launcherComponentPending.Range(func(key, value any) bool {
		activity.Pending[key.(string)] = value.(string)
		return true
	})
	s.mu.Lock()
	activity.Status = s.launcherComponentStatus
	s.mu.Unlock()
	return activity
}

func (s *mainState) configureGenericComponents() {
	s.mu.Lock()
	store := s.cfgStore
	s.mu.Unlock()
	node := ""
	if store != nil {
		node = store.Get().NodeBin
	}
	node, err := harness.ResolveComponentNode(node)
	if err != nil {
		node = ""
	}
	desktopcomponents.ConfigureGeneric(paths.UserDataDir(), desktopcomponents.GenericOptions{
		NodeBin:     node,
		SamplesRoot: filepath.Join(filepath.Dir(paths.VendorDir()), "components", "samples"),
		OnProgress: func(id, action string, p desktopcomponents.GenericProgress) {
			s.mu.Lock()
			s.launcherUpdate = fmt.Sprintf("%s · %s · %s %d%% %s", id, action, p.Phase, p.Percent, p.Message)
			s.launcherComponentStatus = s.launcherUpdate
			if p.Op == "supervise" {
				s.launcherComponentsFresh = false
			}
			win := s.win
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
	})
}

// Share the launcher's existing per-component admission across generic and
// built-in components. The returned release belongs to the existing worker.
func (s *mainState) claimLauncherComponentAction(win *mygo.Window, id, action string) (release func(), claimed bool) {
	if _, loaded := launcherComponentPending.LoadOrStore(id, action); loaded {
		return nil, false
	}
	s.mu.Lock()
	s.launcherComponentStatus = ""
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	return func() {
		launcherComponentPending.Delete(id)
		if win != nil {
			win.Update(func() {})
		}
	}, true
}

func (s *mainState) runGenericComponentAction(win *mygo.Window, id, action string) {
	release, claimed := s.claimLauncherComponentAction(win, id, action)
	if !claimed {
		return
	}
	go func() {
		defer release()
		s.mu.Lock()
		quitting := s.quitting
		var row desktopcomponents.Row
		for _, candidate := range s.launcherComponents {
			if candidate.ID == id {
				row = candidate
				break
			}
		}
		s.mu.Unlock()
		if quitting {
			return
		}
		if action == "uninstall" || action == "rollback" {
			if win == nil {
				return
			}
			title := resumeText("卸载组件", "Uninstall component")
			message := fmt.Sprintf(resumeText("将停止并删除组件「%s」及其程序文件，其配置数据会保留。", "Stop and remove %s and its program files. Configuration data will be retained."), row.Name)
			button := resumeText("卸载", "Uninstall")
			if action == "rollback" {
				title = resumeText("回滚组件", "Roll back component")
				button = resumeText("回滚", "Roll back")
				message = fmt.Sprintf(resumeText("将把组件「%s」回滚到 v%s，当前进程会先停止。", "Roll back %s to v%s. Its running process will be stopped first."), row.Name, row.PreviousVersion)
			}
			answer, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Title: title, Message: message, Buttons: []string{resumeText("取消", "Cancel"), button}, DefaultButton: 0, CancelButton: 0})
			if err != nil || answer.Button != 1 {
				return
			}
		}
		s.mu.Lock()
		quitting = s.quitting
		s.mu.Unlock()
		if quitting {
			return
		}
		_, err := desktopcomponents.RunGenericAction(paths.UserDataDir(), id, action)
		s.mu.Lock()
		s.launcherUpdate = desktopcomponents.FormatActionError(id, action, err)
		s.launcherComponentStatus = s.launcherUpdate
		s.mu.Unlock()
		s.refreshLauncherComponents()
		if win != nil {
			win.Update(func() {})
		}
	}()
}
