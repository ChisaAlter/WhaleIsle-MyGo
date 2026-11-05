package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/desktop/mediapreview"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/projects"
)

func (s *mainState) capturePRViewOwner(root string) func() bool {
	s.mu.Lock()
	sessions, host := s.sessions, s.bootStatus
	s.mu.Unlock()
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	selected, cwd, epoch := "", "", uint64(0)
	if sessions != nil {
		v := sessions.SelectionSummary()
		selected, cwd, epoch = v.SelectedID, v.SelectedCwd, sessions.ComposerSessionGeneration()
	}
	currentRoot := func() string {
		s.mu.Lock()
		cfg := s.cfgStore
		s.mu.Unlock()
		value := ""
		if cfg != nil {
			value = cfg.Get().Workspace
		}
		if sessions != nil {
			if actual := sessions.SelectionSummary().SelectedCwd; actual != "" {
				value = actual
			}
		}
		return value
	}
	return func() bool {
		s.mu.Lock()
		current := s.sessions == sessions && s.bootStatus.Generation == host.Generation && s.bootStatus.Endpoint == host.Endpoint && s.bootStatus.State == host.State && !s.quitting
		s.mu.Unlock()
		s.dshMu.Lock()
		sameClient := s.dshClient == client
		s.dshMu.Unlock()
		if !current || !sameClient || strings.TrimSpace(root) != strings.TrimSpace(currentRoot()) {
			return false
		}
		if sessions != nil {
			now := sessions.SelectionSummary()
			return now.SelectedID == selected && now.SelectedCwd == cwd && sessions.ComposerSessionGeneration() == epoch
		}
		return true
	}
}

func (s *mainState) bindProjectNativeConsumers(win *mygo.Window, m *projects.Model) {
	m.OnOpenModelSettings = func() {
		s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: "models"})
		s.mu.Lock()
		set := s.settings
		s.mu.Unlock()
		if set != nil {
			set.RefreshDescribe(context.Background())
		}
	}
	s.mu.Lock()
	sessions, host := s.sessions, s.bootStatus
	s.mu.Unlock()
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if sessions == nil {
		return
	}
	selected := sessions.SelectedSessionID()
	navigation, err := sessions.CaptureComposerTarget(selected)
	accessCurrent := func() bool { return s.projectFileAccessCurrent(sessions, client, host) }
	admissionCurrent := func() bool { return err == nil && navigation.NavigationCurrent() && accessCurrent() }
	m.OnOpenAuthorizedFile = func(target projectflow.FileTarget) {
		if !admissionCurrent() {
			return
		}
		_, err := projectTargetRelative(target)
		if err != nil {
			sessions.SetLastAction(err.Error())
			return
		}
		checkPath := func(path string) error {
			if !accessCurrent() {
				return errors.New(resumeText("项目文件授权已失效，请从报告重新打开。", "Project file access expired. Reopen it from the report."))
			}
			copy := target
			copy.CanonicalPath = path
			_, err := projectTargetRelative(copy)
			return err
		}
		binding := fmt.Sprintf("project:%p:%d:%s:%s", client, host.Generation, target.ProjectID, target.CoordinatorSessionID)
		model := files.Model{ThemeID: m.ThemeID, SystemDark: m.SystemDark, ImageGeneration: uint64(host.Generation)}
		local := s.managedMarketProfile(host) != ""
		if local {
			model.OnSaveEdit = func(path, text string) error {
				if !accessCurrent() {
					return errors.New(resumeText("项目文件授权已失效，草稿已保留。", "Project file access expired. Your draft was kept."))
				}
				full, err := workspacefs.JoinRoot(target.Root, path)
				if err != nil {
					return err
				}
				if err = checkPath(full); err != nil {
					return err
				}
				return workspacefs.WriteFileText(full, text)
			}
		} else {
			model.ReadTextPageContext = func(ctx context.Context, path string, offset int64) (workspacefs.TextPage, error) {
				if err := checkPath(projectAbsolutePath(target.Root, path)); err != nil {
					return workspacefs.TextPage{}, err
				}
				return workspacefs.ReadRemoteTextPage(ctx, client, target.CoordinatorSessionID, projectAbsolutePath(target.Root, path), offset)
			}
			model.ListRemoteContext = func(ctx context.Context, path string) ([]workspacefs.Entry, error) {
				if !accessCurrent() {
					return nil, errors.New(resumeText("项目文件授权已失效。", "Project file access expired."))
				}
				return workspacefs.ListRemoteDirScoped(ctx, client, target.CoordinatorSessionID, projectAbsolutePath(target.Root, path))
			}
			model.ReadImageContext = func(ctx context.Context, path string, limit int64) ([]byte, error) {
				full := projectAbsolutePath(target.Root, path)
				if err := checkPath(full); err != nil {
					return nil, err
				}
				return mediapreview.ReadSnapshot(ctx, client, target.CoordinatorSessionID, full, int(limit), accessCurrent)
			}
		}
		model.OpenDocumentViewer = func(path string) (files.DocumentSurface, error) {
			path = projectAbsolutePath(target.Root, path)
			if err := checkPath(path); err != nil {
				return nil, err
			}
			url, err := desktopwebview.DocumentURL(client.Endpoint.Origin, target.CoordinatorSessionID, path)
			if err != nil {
				return nil, err
			}
			return desktopwebview.NewHost(desktopwebview.HostOptions{Parent: win, URL: url, Title: path, Untrusted: true, SessionOrigin: client.Endpoint.Origin, SessionCookie: client.Endpoint.Cookie, OnChange: win.Invalidate, OnFailure: s.onEmbeddedPageFailure, ShortcutPreferences: s.pageShortcutPreferences(), OnShortcutCloseWindow: s.queueProductShortcutClose})
		}
		// The Files owner, not the selected conversation, retains this target root
		// and runs its existing dirty/save-close admission for every occurrence.
		s.mu.Lock()
		if s.layout.Workspace == nil {
			s.layout.Workspace = &frame.WorkspacePanes{}
		}
		panes := s.layout.Workspace
		s.mu.Unlock()
		if err := panes.OpenAuthorizedProjectFile(target, binding, frame.ProjectFileAccess{Valid: accessCurrent, Model: model}); err != nil {
			sessions.SetLastAction(err.Error())
			return
		}
		s.selectMainPanel("")
		s.mu.Lock()
		s.filesPaneOpen, s.rightPaneTab = true, "files"
		s.mu.Unlock()
		if win != nil {
			win.Invalidate()
		}
	}
	if original := m.ObserverModel; original != nil {
		m.ObserverModel = func(o *projects.ObserverState) projects.Model {
			child := original(o)
			s.bindProjectNativeConsumers(win, &child)
			return child
		}
	}
}

// A retained Project resource outlives the chat selection that admitted it.
// Its typed coordinator/root descriptor is immutable; only its owning Host and
// client lease can revoke the runtime readers and checked local save callback.
func (s *mainState) projectFileAccessCurrent(sessions *session.Store, client *dsh.Client, host harness.Status) bool {
	s.mu.Lock()
	same := s.sessions == sessions && s.bootStatus.Generation == host.Generation && s.bootStatus.Endpoint == host.Endpoint && s.bootStatus.State == harness.Ready && !s.quitting
	s.mu.Unlock()
	s.dshMu.Lock()
	sameClient := s.dshClient == client
	s.dshMu.Unlock()
	return same && sameClient
}

func projectAbsolutePath(root, value string) string {
	value = strings.ReplaceAll(value, `\`, "/")
	if strings.HasPrefix(value, "/") || len(value) > 1 && value[1] == ':' {
		return value
	}
	return strings.TrimRight(strings.ReplaceAll(root, `\`, "/"), "/") + "/" + value
}
func projectTargetRelative(t projectflow.FileTarget) (string, error) {
	fail := errors.New(resumeText("项目文件路径不属于报告授权的工作目录。", "The Project file is outside the report's authorized root."))
	if t.ProjectID == "" || t.CoordinatorSessionID == "" || strings.ContainsRune(t.Root, 0) || strings.ContainsRune(t.CanonicalPath, 0) {
		return "", fail
	}
	rel, ok := files.WorkspaceRelativePath(t.Root, t.CanonicalPath)
	if !ok || rel == "." {
		return "", fail
	}
	if err := workspacefs.ValidateRelativePath(rel); err != nil {
		return "", err
	}
	return rel, nil
}
