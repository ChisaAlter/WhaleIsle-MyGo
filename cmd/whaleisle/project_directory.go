package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

type projectDirectoryIntent struct {
	client *dsh.Client
	path   string
}

// The incoming Host path is never authority. A native directory-row/Hero click
// may authorize one exact registered path; otherwise the native dialog chooses.
func (s *mainState) selectProjectDirectory(ctx context.Context, suggested string) (string, error) {
	s.mu.Lock()
	win, sup, sessions, tunnel := s.win, s.sup, s.sessions, s.sshTunnel
	s.mu.Unlock()
	if win == nil || sup == nil {
		return "", errors.New(i18n.T(i18n.KeyCmdProjectDirectoryUnavailable))
	}
	if tunnel != nil {
		if _, status, _ := tunnel.Snapshot(); status == sshremote.TunnelConnected {
			return "", errors.New(i18n.T(i18n.KeyCmdProjectLocalDirectoryRemote))
		}
	}
	st := sup.Status()
	if st.State != harness.Ready || st.Endpoint.Origin == "" {
		return "", errors.New(i18n.T(i18n.KeyCmdWorkspaceHostUnavailable))
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	intent := s.projectFlow.directoryIntent
	if intent != nil && intent.client == client && s.projectFlow.client == client && strings.EqualFold(filepath.Clean(intent.path), filepath.Clean(suggested)) {
		s.projectFlow.directoryIntent = nil
	} else {
		intent = nil
	}
	s.mu.Unlock()
	if intent != nil && sessions != nil && client != nil && client.Endpoint.Origin == st.Endpoint.Origin && client.Endpoint.Cookie == st.Endpoint.Cookie {
		if choices, ready := sessions.CachedWorkspaceChoices(client); ready {
			for _, choice := range choices {
				if strings.EqualFold(filepath.Clean(choice.Path), filepath.Clean(intent.path)) {
					// The Project bridge still rechecks the Host registry and logical
					// authority before accepting the selected working directory.
					return choice.Path, nil
				}
			}
		}
	}
	if intent != nil {
		return "", errors.New("工作目录登记状态已变化，请刷新目录后重试。")
	}
	initial := ""
	if s.cfgStore != nil {
		initial = s.cfgStore.Get().Workspace
	}
	if directorypicker.SelectablePath(suggested) {
		initial = suggested
	}
	selected, err := directorypicker.SelectAndRegister(ctx, func() ([]string, error) {
		return mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: i18n.T(i18n.KeyCmdChooseProjectDirectory), DefaultPath: initial, Directory: true})
	}, func(ctx context.Context, chosen string) error {
		// A dialog may remain open while Host restarts; never register into a stale Host.
		current := sup.Status()
		if current.State != harness.Ready || current.Endpoint != st.Endpoint {
			return errors.New(i18n.T(i18n.KeyCmdWorkspaceHostSelectionChanged))
		}
		registerCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		return harness.EnsureWorkspace(registerCtx, nil, current.Endpoint, chosen)
	})
	if err != nil {
		return "", err
	}
	if s.cfgStore != nil {
		if err = s.cfgStore.Update(func(c *config.Config) { c.Workspace = selected }); err != nil {
			return "", errors.New(i18n.T(i18n.KeyCmdWorkspacePreferenceSaveFailed))
		}
	}
	if sessions != nil {
		sessions.SetWorkspaceCwd(selected)
	}
	win.Update(func() {})
	return selected, nil
}
