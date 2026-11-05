package main

import (
	"context"
	"errors"

	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/ui/i18n"
)

// sshConnectOperation owns one UI attempt, including any queued host-key dialog.
// Its context only controls connection establishment; the tunnel owns the
// lifetime of an established connection independently.
type sshConnectOperation struct {
	owner  *mainState
	epoch  uint64
	tunnel *sshremote.TunnelSession
	ctx    context.Context
	cancel context.CancelFunc
}

// Caller holds s.mu. Saving or replacing an attempt must invalidate it before
// changing the tunnel configuration, so an old prompt cannot trust a new host.
func (s *mainState) invalidateSSHConnectLocked() {
	s.sshConnectEpoch++
	if s.sshConnectCancel != nil {
		s.sshConnectCancel()
		s.sshConnectCancel = nil
	}
}

// Caller holds s.mu.
func (s *mainState) beginSSHConnectLocked(cfg sshremote.TunnelConfig) *sshConnectOperation {
	s.invalidateSSHConnectLocked()
	if s.sshTunnel == nil {
		s.sshTunnel = sshremote.NewTunnelSession()
	}
	// A replacement attempt may start before the canceled handshake returns.
	// Retire only that pending handshake, never an established connection.
	if _, status, _ := s.sshTunnel.Snapshot(); status == sshremote.TunnelConnecting {
		s.sshTunnel.Disconnect()
	}
	s.sshTunnel.SetConfig(cfg)
	s.sshTunnel.SetTrustOnFirstUse(false)
	tunnel := s.sshTunnel
	tunnel.OnChange(func() {
		s.mu.Lock()
		if s.sshTunnel != tunnel || s.quitting {
			s.mu.Unlock()
			return
		}
		_, status, detail := tunnel.Snapshot()
		if detail != "" {
			s.sshTunnelMsg = detail
		} else if status == sshremote.TunnelIdle {
			s.sshTunnelMsg = lifecycleText("SSH 连接已断开", "SSH connection closed")
		}
		win := s.win
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	s.sshConnectCancel = cancel
	return &sshConnectOperation{owner: s, epoch: s.sshConnectEpoch, tunnel: s.sshTunnel, ctx: ctx, cancel: cancel}
}

func (op *sshConnectOperation) currentLocked() bool {
	return op.owner.sshConnectEpoch == op.epoch && op.owner.sshTunnel == op.tunnel && !op.owner.quitting && op.ctx.Err() == nil
}

func (op *sshConnectOperation) current() bool {
	op.owner.mu.Lock()
	defer op.owner.mu.Unlock()
	return op.currentLocked()
}

// Host-key persistence is short and serialized with Save/Disconnect. Never
// hold this lock across a dialog or a network call.
func (op *sshConnectOperation) accept(unknown *sshremote.UnknownHostKey, write func(*sshremote.UnknownHostKey) error) (bool, error) {
	op.owner.mu.Lock()
	defer op.owner.mu.Unlock()
	if !op.currentLocked() {
		return false, nil
	}
	return true, write(unknown)
}

func (op *sshConnectOperation) publish(msg string) bool {
	op.owner.mu.Lock()
	defer op.owner.mu.Unlock()
	if !op.currentLocked() {
		return false
	}
	_, _, currentError := op.tunnel.Snapshot()
	if currentError != "" {
		msg = currentError
	}
	op.owner.sshTunnelMsg = msg
	op.owner.sshConnectCancel = nil
	op.cancel()
	return true
}

// These effects are separate so the real asynchronous flow can be exercised
// without an OS dialog, user keys, or an SSH server.
type sshConnectEffects struct {
	connect func(context.Context) error
	prompt  func(*sshremote.UnknownHostKey, func() bool) bool
	changed func(*sshremote.HostKeyChanged, func() bool)
	accept  func(*sshremote.UnknownHostKey) error
	redraw  func()
}

func (op *sshConnectOperation) run(e sshConnectEffects) {
	defer op.cancel()
	if !op.current() {
		return
	}
	err := e.connect(op.ctx)
	if !op.current() {
		return
	}
	msg := i18n.T(i18n.KeyRemoteSSHConnected)
	if err != nil {
		var changed *sshremote.HostKeyChanged
		var unknown *sshremote.UnknownHostKey
		switch {
		case errors.As(err, &changed):
			e.changed(changed, op.current)
			msg = i18n.T(i18n.KeyRemoteSSHHostChangedStatus) + ": " + changed.Error()
		case errors.As(err, &unknown):
			if e.prompt(unknown, op.current) {
				owned, aerr := op.accept(unknown, e.accept)
				if !owned {
					return
				}
				if aerr != nil {
					msg = aerr.Error()
				} else {
					if !op.current() {
						return
					}
					// Connect also rejects a pre-canceled context before touching
					// its state, closing the check-to-call cancellation boundary.
					if err = e.connect(op.ctx); err != nil {
						msg = err.Error()
					}
				}
			} else {
				msg = i18n.T(i18n.KeyRemoteSSHHostDeclined)
			}
		default:
			msg = err.Error()
		}
	}
	if op.publish(msg) && e.redraw != nil {
		e.redraw()
	}
}
