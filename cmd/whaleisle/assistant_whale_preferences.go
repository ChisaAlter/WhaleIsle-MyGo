package main

import (
	"context"
	"fmt"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) assistantWhalePreferencesModel(ep harness.Endpoint) assistantui.WhalePreferencesModel {
	client, host := s.whaleSettingsOwner(ep)
	editor := &s.whaleSettings.Preferences
	if host != "" {
		editor.Bind(host)
	}
	current := func() bool {
		now, key := s.whaleSettingsOwner(ep)
		return client != nil && now == client && host != "" && key == host
	}
	run := func(cmd *assiststore.WhalePreferencesCommand) {
		if !current() {
			return
		}
		ticket, ok := editor.Begin(host, cmd != nil)
		if !ok {
			return
		}
		s.refreshWhaleSettingsUI()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rpc := s.pluginRPCClient(ep)
			var value pluginrpc.WhalePreferences
			var catalog *api.ModelCatalog
			var modelErr string
			execute := func() error {
				if !current() || !editor.Current(ticket) {
					return fmt.Errorf("Whale editor changed")
				}
				if cmd != nil {
					var err error
					switch cmd.Action {
					case "model":
						err = rpc.WhaleSaveModel(ctx, cmd.Route)
					case "im":
						err = rpc.WhaleSaveIMDefault(ctx, cmd.IMDefault)
					case "skill":
						err = rpc.WhaleToggleSkill(ctx, cmd.Skill, cmd.Enabled)
					default:
						return fmt.Errorf("unknown Whale preference")
					}
					if err != nil {
						return err
					}
				}
				if !current() || !editor.Current(ticket) {
					return fmt.Errorf("Whale editor changed")
				}
				var err error
				value, err = rpc.WhalePreferences(ctx)
				if err != nil {
					return err
				}
				if cmd == nil {
					catalog = &api.ModelCatalog{}
					if !current() || !editor.Current(ticket) {
						return fmt.Errorf("Whale editor changed")
					}
					if e := client.Call(ctx, "session/modelCatalog", map[string]any{}, catalog); e != nil {
						modelErr = e.Error()
					}
				}
				return nil
			}
			err := execute()
			if current() {
				editor.Finish(ticket, value, catalog, modelErr, cmd, err)
				s.refreshWhaleSettingsUI()
			}
		}()
	}
	state := editor.Snapshot()
	if host == "" {
		state = assiststore.WhalePreferencesSnapshot{}
	}
	return assistantui.WhalePreferencesModel{
		State:     state,
		OnRefresh: func() { run(nil) },
		OnEdit: func(route pluginrpc.WhaleModelRoute, im bool) {
			if current() {
				editor.Edit(host, route, im)
				s.refreshWhaleSettingsUI()
			}
		},
		OnSave: func(action string) {
			if !current() {
				return
			}
			draft := editor.Snapshot()
			run(&assiststore.WhalePreferencesCommand{Action: action, Route: draft.Route, IMDefault: draft.IMDefault})
		},
		OnToggleSkill: func(name string, enabled bool) {
			run(&assiststore.WhalePreferencesCommand{Action: "skill", Skill: name, Enabled: enabled})
		},
	}
}
