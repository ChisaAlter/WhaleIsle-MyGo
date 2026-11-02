package main

import (
	"context"
	"fmt"
	"strings"
	"whaleisle/internal/ui/i18n"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/store/session"
)

// assistantDshbotCurrent fences delayed catalog/open results to the same live
// client and enabled plugin. A normal chat selection is never a Bot identity.
func (s *mainState) assistantDshbotCurrent(ep harness.Endpoint, client *dsh.Client) bool {
	s.dshMu.Lock()
	same := client != nil && s.dshClient == client &&
		client.Endpoint.Origin == ep.Origin && client.Endpoint.Cookie == ep.Cookie
	s.dshMu.Unlock()
	s.mu.Lock()
	ready := s.bootStatus.State == harness.Ready &&
		s.bootStatus.Endpoint.Origin == ep.Origin && s.bootStatus.Endpoint.Cookie == ep.Cookie
	enabled := s.cfgStore != nil && s.cfgStore.Get().DshbotEnabled
	s.mu.Unlock()
	return same && ready && enabled
}

func (s *mainState) resolveAssistantDshbot(ctx context.Context, ep harness.Endpoint, sessions *session.Store, botID string) (*dsh.Client, string, error) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if sessions == nil || !s.assistantDshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotHostUnavailable))
	}
	botID = strings.TrimSpace(botID)
	if botID == "" {
		return nil, "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdSelectRosterBotOrRoom))
	}
	cli := s.pluginRPCClient(ep)
	catalog, err := cli.DshbotDescribeTyped(ctx, botID)
	if err != nil {
		return nil, "", err
	}
	var item pluginrpc.DshbotCatalogItem
	for _, candidate := range catalog.View.Value.Items {
		if candidate.ID == botID && !candidate.Hidden {
			item = candidate
			break
		}
	}
	if item.ID == "" {
		return nil, "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotProfileUnavailable))
	}
	if !s.assistantDshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("Host connection changed; refresh the roster")
	}
	sid, err := cli.DshbotOpen(ctx, item, catalog.View.Revision, sessions.ScratchCwd())
	if err != nil {
		return nil, "", err
	}
	if !s.assistantDshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("Host connection changed while opening the Bot")
	}
	return client, sid, nil
}

// Open always resolves bot/open; a cached session ID is not a Bot identity.
func (s *mainState) openAssistantDshbot(ep harness.Endpoint, botID string) {
	s.openAssistantConversation(ep, assiststore.ModeDshbot, botID, "")
}
