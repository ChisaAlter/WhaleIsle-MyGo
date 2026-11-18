package assistantctl

import (
	"context"
	"fmt"
	"strings"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// DshbotCurrent fences delayed catalog/open results to the same live client and
// enabled Dshbot plugin. A normal chat selection is never a Bot identity.
func (s *Service) DshbotCurrent(ep harness.Endpoint, client *dsh.Client) bool {
	if s == nil {
		return false
	}
	host := s.safeHost()
	liveClient := host.CurrentClient()
	same := client != nil && liveClient == client &&
		client.Endpoint.Origin == ep.Origin && client.Endpoint.Cookie == ep.Cookie
	if !same {
		return false
	}
	bootEp, bootState, _ := host.BootEndpoint()
	cfgStore := host.Config()
	ready := bootState == harness.Ready &&
		bootEp.Origin == ep.Origin && bootEp.Cookie == ep.Cookie
	enabled := cfgStore != nil && cfgStore.Get().DshbotEnabled
	return ready && enabled
}

// ResolveDshbot verifies catalog ownership and opens the canonical Bot/room session.
func (s *Service) ResolveDshbot(ctx context.Context, ep harness.Endpoint, sessions *session.Store, botID string) (*dsh.Client, string, error) {
	if s == nil {
		return nil, "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotHostUnavailable))
	}
	host := s.safeHost()
	client := host.CurrentClient()
	if sessions == nil || !s.DshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotHostUnavailable))
	}
	botID = strings.TrimSpace(botID)
	if botID == "" {
		return nil, "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdSelectRosterBotOrRoom))
	}
	cli := host.PluginRPCClient(ep)
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
	if !s.DshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("Host connection changed; refresh the roster")
	}
	sid, err := cli.DshbotOpen(ctx, item, catalog.View.Revision, sessions.ScratchCwd())
	if err != nil {
		return nil, "", err
	}
	if !s.DshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("Host connection changed while opening the Bot")
	}
	return client, sid, nil
}

// OpenDshbot resolves bot/open for the requested Bot/room ID and navigates to it.
func (s *Service) OpenDshbot(ep harness.Endpoint, botID string) {
	if s == nil {
		return
	}
	s.OpenConversation(ep, assiststore.ModeDshbot, botID, "")
}
