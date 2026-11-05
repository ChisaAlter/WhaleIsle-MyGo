package main

import (
	"context"

	"whaleisle/internal/accountlogin"
	"whaleisle/internal/dsh/api"
)

func (s *mainState) startAccountSignIn() {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if client == nil || !s.accountLogin.Begin() {
		return
	}
	accountlogin.SetClientVersion(buildVersion)
	go func() {
		origin, err := accountlogin.CallbackOrigin(client.Endpoint.Origin)
		var view api.AccountView
		if err == nil {
			err = client.Call(context.Background(), "account/startSignIn", map[string]any{
				"client": accountlogin.ClientMetadata(), "callbackOrigin": origin, "loginSource": "desktop",
			}, &view)
		}
		s.dshMu.Lock()
		current := s.dshClient == client
		s.dshMu.Unlock()
		if !current {
			return
		}
		s.accountLogin.Started(view, err, s.openAccountAuthorizationBrowser)
		s.refreshReturnedAccount()
		s.updateProviderProfilesUI()
	}()
}

func (s *mainState) retryAccountSignIn() {
	s.mu.Lock()
	store := s.usage
	s.mu.Unlock()
	if store == nil {
		return
	}
	v := store.Snapshot()
	if !accountlogin.Active(v.AttemptPhase) || v.AttemptID == "" {
		s.startAccountSignIn()
		return
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if client == nil || !s.accountLogin.BeginCancel() {
		return
	}
	go func() {
		var out api.AccountView
		err := client.Call(context.Background(), "account/cancelSignIn", map[string]any{"attemptId": v.AttemptID}, &out)
		s.dshMu.Lock()
		current := s.dshClient == client
		s.dshMu.Unlock()
		if !current {
			return
		}
		if err != nil {
			s.accountLogin.CancelFailed(err.Error())
			s.reportAccountAction(err)
			return
		}
		s.accountLogin.Finish()
		s.startAccountSignIn()
	}()
}
