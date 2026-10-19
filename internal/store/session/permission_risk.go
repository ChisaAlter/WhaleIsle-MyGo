package session

import (
	"context"
	"whaleisle/internal/dsh"
)

const PermissionFullAccess = "danger-full-access"
const PermissionAutoReview = "auto"

type PermissionRiskConfirmation struct {
	Token                   uint64
	SessionID, Label, Value string
	Acknowledged            bool
}
type permissionRiskOperation struct {
	view       PermissionRiskConfirmation
	client     *dsh.Client
	generation uint64
}

func permissionNeedsConfirmation(value string) bool {
	return value == PermissionFullAccess || value == PermissionAutoReview
}
func (s *Store) resetPermissionRiskLocked() {
	s.permissionRiskToken++
	s.permissionRisk = nil
}
func (s *Store) permissionRiskCurrentLocked(token uint64) bool {
	p := s.permissionRisk
	return p != nil && p.view.Token == token && p.client == s.client &&
		p.view.SessionID == s.selectedID && p.generation == s.followGeneration && s.permissionProjectionCurrentLocked()
}
func (s *Store) PermissionRiskConfirmation() *PermissionRiskConfirmation {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.permissionRisk == nil || !s.permissionRiskCurrentLocked(s.permissionRisk.view.Token) {
		return nil
	}
	out := s.permissionRisk.view
	return &out
}
func (s *Store) AcknowledgePermissionRisk(token uint64, acknowledged bool) {
	s.mu.Lock()
	if !s.permissionRiskCurrentLocked(token) {
		s.mu.Unlock()
		return
	}
	s.permissionRisk.view.Acknowledged = acknowledged
	s.mu.Unlock()
	s.fire()
}
func (s *Store) CancelPermissionRisk(token uint64) {
	s.mu.Lock()
	if s.permissionRisk == nil || s.permissionRisk.view.Token != token {
		s.mu.Unlock()
		return
	}
	s.resetPermissionRiskLocked()
	s.mu.Unlock()
	s.fire()
}

// Confirmation consumes one token before sending. It never changes the picker
// optimistically; the existing session projection remains the only authority.
func (s *Store) ConfirmPermissionRisk(ctx context.Context, token uint64) {
	s.mu.Lock()
	if ctx.Err() != nil || !s.permissionRiskCurrentLocked(token) || !s.permissionRisk.view.Acknowledged || s.permissionProjection.Pending {
		s.mu.Unlock()
		return
	}
	p := *s.permissionRisk
	opt, exists := permissionOptionByValue(p.view.Value, s.permissionOptions)
	if !exists {
		s.resetPermissionRiskLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	s.resetPermissionRiskLocked()
	if opt.Value == s.permissionProjection.Value {
		s.mu.Unlock()
		s.fire()
		return
	}
	s.permissionProjection.Pending = true
	s.permissionsErr = nil
	s.mu.Unlock()
	s.fire()
	go s.runPermissionPreset(ctx, p.client, p.view.SessionID, p.generation, opt)
}
