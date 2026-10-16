package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) assistantContactsModel(ep harness.Endpoint) assistantui.ContactsModel {
	client := s.assistantRoomClient(ep)
	s.mu.Lock()
	originalAssistant := s.assistant
	roster := s.dshbotRoster
	s.mu.Unlock()
	scope, binding := "", ""
	if client != nil {
		scope = assistantVisibilityScope(ep)
		binding = fmt.Sprintf("%p", client)
	}
	controls := &s.assistantContacts
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	rpc := s.pluginRPCClient(ep)
	current := func() bool {
		s.mu.Lock()
		same := originalAssistant != nil && s.assistant == originalAssistant
		s.mu.Unlock()
		return same && client != nil && s.assistantRoomClient(ep) == client && controls.ViewCurrent(scope, snapshot.Generation)
	}
	updateRoster := func(catalog pluginrpc.DshbotContactCatalog) {
		mygo.RunOnMain(func() {
			if !current() {
				return
			}
			s.mu.Lock()
			roster := s.dshbotRoster
			s.mu.Unlock()
			if roster == nil {
				return
			}
			items := make([]pluginrpc.DshbotCatalogItem, len(catalog.Items))
			for i, row := range catalog.Items {
				items[i] = row.DshbotCatalogItem
			}
			sections := make([]pluginrpc.DshbotSection, len(catalog.Sections))
			for i, section := range catalog.Sections {
				sections[i] = pluginrpc.DshbotSection{ID: section.ID, Name: section.Name}
			}
			roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{View: pluginrpc.DshbotSettingsView{NS: "dsh-bot", Revision: catalog.Revision, Value: pluginrpc.DshbotCatalogValue{Items: items, Sections: sections}}}, false, ep.Origin)
		})
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := controls.BeginLoad(scope)
		if !ok {
			return
		}
		s.updateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var authority struct {
				Writable bool `json:"writable"`
			}
			authorityErr := client.Call(ctx, "settings/describe", nil, &authority)
			if !current() {
				return
			}
			catalog, err := rpc.DshbotContacts(ctx)
			if current() && controls.FinishLoad(ticket, catalog, authorityErr == nil && authority.Writable, err) {
				if err == nil {
					updateRoster(catalog)
				}
				s.updateAssistantTasks()
			}
		}()
	}
	model := assistantui.ContactsModel{State: snapshot, Activity: s.assistantContactsActivity(ep, snapshot.Catalog), OnRefresh: refresh,
		OnMarkRead: func(id string, throughSeq int) {
			if !current() || !snapshot.Writable || throughSeq < 0 {
				return
			}
			ticket, ok := controls.BeginLoad(scope)
			if !ok {
				return
			}
			s.updateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if !current() {
					return
				}
				view, err := rpc.DshbotMarkRead(ctx, []pluginrpc.DshbotReadMark{{ID: id, ThroughSeq: throughSeq}})
				catalog := snapshot.Catalog
				if err == nil {
					catalog, err = rpc.DshbotContacts(ctx)
				}
				if current() && controls.FinishLoad(ticket, catalog, snapshot.Writable, err) {
					if err == nil {
						updateRoster(catalog)
						mygo.RunOnMain(func() {
							if current() {
								s.mu.Lock()
								run := s.botNotifications
								s.mu.Unlock()
								if run != nil && run.client == client {
									applyBotReadView(run.contactActivity, view)
								}
							}
						})
					}
					s.updateAssistantTasks()
				}
			}()
		},
		OnChange: func(field, value string) {
			if current() {
				controls.Change(scope, field, value)
				s.updateAssistantTasks()
			}
		},
		OnAction: func(cmd pluginrpc.DshbotContactCommand) {
			if !current() {
				return
			}
			cmd.Expected = snapshot.Catalog
			cmd.UpdatedAt = time.Now().UnixMilli()
			if cmd.Action == "create" && cmd.ID == "" {
				cmd.ID = pluginrpc.NewDshbotContactSectionID()
			}
			ticket, frozen, ok := controls.Begin(scope, cmd)
			if !ok {
				return
			}
			s.updateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if !current() {
					controls.Finish(ticket, frozen, pluginrpc.DshbotContactResult{}, fmt.Errorf("原 Host/账户已切换，操作未发送"))
					return
				}
				result, err := rpc.DshbotChangeContacts(ctx, frozen)
				if controls.Finish(ticket, frozen, result, err) && current() {
					if err == nil {
						updateRoster(result.Catalog)
					}
					s.updateAssistantTasks()
				}
			}()
		},
	}
	newerCatalog := false
	if roster != nil && snapshot.Loaded {
		cached := roster.Snapshot()
		newerCatalog = cached.Origin == ep.Origin && cached.Revision > snapshot.Catalog.Revision
	}
	if scope != "" && (!snapshot.Loaded || newerCatalog) && !snapshot.Busy && snapshot.Error == "" {
		refresh()
	}
	return model
}
func (s *mainState) observeAssistantContacts(rendered bool) {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if !rendered || s.assistantRoomClient(ep) == nil {
		s.assistantContacts.Suspend()
	}
}
