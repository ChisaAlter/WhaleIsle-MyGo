package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

func exerciseDshbot(ctx context.Context, client *dsh.Client, store *session.Store, workspace string, pass func(string, string), restart func() (*dsh.Client, *session.Store, error)) error {
	plugin := &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	var preset api.AgentPresetDocument
	if err := client.Call(ctx, "agentPresets/read", map[string]any{"agentPreset": "dshbot-room"}, &preset); err != nil {
		return err
	}
	// Healthy hidden presets intentionally do not appear in the selection roster.
	if preset.AgentPreset != "dshbot-room" || !strings.Contains(preset.Content, "dshbot/ask-participant") {
		return errors.New("retained dshbot room preset is unavailable")
	}
	pass("dshbot-room-preset-registered", "original dshbot room composition registered with the rc.4 live preset registry")
	for _, id := range []string{"qa-bot-one", "qa-bot-two"} {
		catalog, err := plugin.DshbotBotCatalog(ctx)
		if err != nil {
			return err
		}
		_, err = plugin.DshbotCreateBot(ctx, pluginrpc.DshbotBotCreateCommand{ID: id, SessionID: id + "-session", Expected: catalog, Draft: pluginrpc.DshbotBotDraft{Name: id, Description: "Synthetic loopback-only acceptance bot", ScratchCwd: workspace}})
		if err != nil {
			return fmt.Errorf("create %s: %w", id, err)
		}
	}
	pass("dshbot-personal-contacts-created", "two contacts and canonical sessions created through production native RPC with catalog receipts")
	catalog, err := plugin.DshbotRoomCatalog(ctx)
	if err != nil {
		return err
	}
	created, err := plugin.DshbotCreateRoom(ctx, pluginrpc.DshbotRoomCreateCommand{ID: "qa-room", SessionID: "qa-room-session", Expected: catalog, Draft: pluginrpc.DshbotRoomDraft{Name: "Owned compatibility room", ScratchCwd: workspace, MemberBotIDs: []string{"qa-bot-one", "qa-bot-two"}, MaxRounds: 1, MaxSpeaks: 2}})
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	roomID, err := plugin.DshbotOpen(ctx, created.Item.DshbotCatalogItem, created.Catalog.Revision, workspace)
	if err != nil {
		return fmt.Errorf("open room: %w", err)
	}
	pass("dshbot-room-created-opened", "native room creation/open preserves the selected canonical session and two member identities")
	store.Select(roomID)
	if err = await(ctx, func() (bool, error) {
		v := store.Snapshot()
		if v.HistoryError != "" {
			return false, errors.New(v.HistoryError)
		}
		return v.SelectedID == roomID && v.HistoryReady && !v.HistoryLoading, nil
	}); err != nil {
		return err
	}
	if err = store.SubmitComposer(ctx, roomID, helloPrompt, nil, 0); err != nil {
		return err
	}
	var runtime pluginrpc.DshbotRoomRuntime
	if err = await(ctx, func() (bool, error) {
		runtime, err = plugin.DshbotRoomRuntime(ctx, "qa-room")
		if err != nil {
			return false, err
		}
		if len(runtime.Members) != 2 {
			return false, nil
		}
		for _, member := range runtime.Members {
			if member.Status != "idle" && member.Status != "completed" {
				return false, nil
			}
		}
		return !store.Snapshot().PromptBusy, nil
	}); err != nil {
		return fmt.Errorf("room dispatch: %w", err)
	}
	readMember := func(id string) error {
		store.Select(id)
		return await(ctx, func() (bool, error) {
			v := store.Snapshot()
			if v.HistoryError != "" {
				return false, errors.New(v.HistoryError)
			}
			for _, m := range v.Messages {
				if m.Role == "assistant" && strings.Contains(m.Body, "QA_REPLY_OK") && !m.Streaming && m.Seq > 0 {
					return true, nil
				}
			}
			return false, nil
		})
	}
	for _, member := range runtime.Members {
		if err = readMember(member.SessionID); err != nil {
			return fmt.Errorf("member %s response: %w", member.BotID, err)
		}
	}
	pass("dshbot-room-member-turns", "both room members have persisted responses from the local model in their actual member sessions")
	client, store, err = restart()
	if err != nil {
		return err
	}
	plugin = &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	restored, err := plugin.DshbotRoomCatalog(ctx)
	if err != nil {
		return err
	}
	if len(restored.Items) != 3 {
		return fmt.Errorf("restored catalog has %d entries", len(restored.Items))
	}
	for _, member := range runtime.Members {
		if err = readMember(member.SessionID); err != nil {
			return err
		}
	}
	pass("dshbot-room-restart-restores", "fresh Host restores both contacts, room, member identities and persisted replies")
	return nil
}
