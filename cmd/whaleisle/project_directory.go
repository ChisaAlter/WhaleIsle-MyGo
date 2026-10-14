package main

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/harness"
)

// The incoming Host path is never authority. Only the native dialog's explicit
// choice may reach workspace/create; cancellation leaves configuration intact.
func (s *mainState) selectProjectDirectory(ctx context.Context, _ string) (string, error) {
	s.mu.Lock()
	win, sup, sessions, tunnel := s.win, s.sup, s.sessions, s.sshTunnel
	s.mu.Unlock()
	if win == nil || sup == nil {
		return "", errors.New("desktop workspace selection unavailable")
	}
	if tunnel != nil {
		if _, status, _ := tunnel.Snapshot(); status == sshremote.TunnelConnected {
			return "", errors.New("local Project directory selection unavailable in remote mode")
		}
	}
	st := sup.Status()
	if st.State != harness.Ready || st.Endpoint.Origin == "" {
		return "", errors.New("workspace Host unavailable")
	}
	initial := ""
	if s.cfgStore != nil {
		initial = s.cfgStore.Get().Workspace
	}
	selected, err := directorypicker.SelectAndRegister(ctx, func() ([]string, error) {
		return mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: "选择 Project 工作目录", DefaultPath: initial, Directory: true})
	}, func(ctx context.Context, chosen string) error {
		// A dialog may remain open while Host restarts; never register into a stale Host.
		current := sup.Status()
		if current.State != harness.Ready || current.Endpoint != st.Endpoint {
			return errors.New("workspace Host changed during directory selection")
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
			return "", errors.New("workspace preference could not be saved")
		}
	}
	if sessions != nil {
		sessions.SetWorkspaceCwd(selected)
	}
	win.Update(func() {})
	return selected, nil
}
