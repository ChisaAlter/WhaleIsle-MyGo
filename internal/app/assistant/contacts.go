package assistantctl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

// ContactsModel builds the Dshbot contacts and sections management model.
func (s *Service) ContactsModel(ep harness.Endpoint) assistantui.ContactsModel {
	if s == nil {
		return assistantui.ContactsModel{}
	}
	host := s.safeHost()
	client := s.roomClient(ep)
	originalAssistant := s.AssistantStore()
	roster := s.DshbotRoster()
	scope, binding := "", ""
	if client != nil {
		scope = endpointScope(ep.Origin, ep.Cookie)
		binding = clientBinding(client)
	}
	controls := &s.assistantContacts
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	rpc := host.PluginRPCClient(ep)
	current := func() bool {
		same := originalAssistant != nil && s.AssistantStore() == originalAssistant
		return same && client != nil && s.roomClient(ep) == client && controls.ViewCurrent(scope, snapshot.Generation)
	}
	updateRoster := func(catalog pluginrpc.DshbotContactCatalog) {
		host.RunOnMain(func() {
			if !current() {
				return
			}
			r := s.DshbotRoster()
			if r == nil {
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
			r.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{
				View: pluginrpc.DshbotSettingsView{
					NS:       "dsh-bot",
					Revision: catalog.Revision,
					Value:    pluginrpc.DshbotCatalogValue{Items: items, Sections: sections},
				},
			}, false, ep.Origin)
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
		s.UpdateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			writable, authErr := checkWriteAuthority(ctx, client, current, "contacts view changed")
			if !current() {
				return
			}
			catalog, err := rpc.DshbotContacts(ctx)
			if current() && controls.FinishLoad(ticket, catalog, authErr == nil && writable, err) {
				if err == nil {
					updateRoster(catalog)
				}
				s.UpdateAssistantTasks()
			}
		}()
	}
	model := assistantui.ContactsModel{
		State:     snapshot,
		Activity:  s.contactsActivity(ep, snapshot.Catalog),
		OnRefresh: refresh,
		OnMarkRead: func(id string, throughSeq int) {
			if !current() || !snapshot.Writable || throughSeq < 0 {
				return
			}
			ticket, ok := controls.BeginLoad(scope)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
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
						host.RunOnMain(func() {
							if current() {
								s.mu.Lock()
								run := s.botNotifications
								s.mu.Unlock()
								if run != nil && run.client == client {
									run.mu.Lock()
									applyBotReadView(run.contactActivity, view)
									run.mu.Unlock()
								}
							}
						})
					}
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(field, value string) {
			if current() {
				controls.Change(scope, field, value)
				s.UpdateAssistantTasks()
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
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if !current() {
					controls.Finish(ticket, frozen, pluginrpc.DshbotContactResult{}, fmt.Errorf("%s", i18n.T(i18n.KeyCmdContactsOwnerChanged)))
					return
				}
				result, err := rpc.DshbotChangeContacts(ctx, frozen)
				if controls.Finish(ticket, frozen, result, err) && current() {
					if err == nil {
						updateRoster(result.Catalog)
					}
					s.UpdateAssistantTasks()
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

// ObserveContacts suspends contacts polling/state when the contacts panel did not render.
func (s *Service) ObserveContacts(rendered bool) {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if !rendered || s.roomClient(ep) == nil {
		s.assistantContacts.Suspend()
	}
}

// Projects notification/session snapshots and read-only Room runtime observation
// while the original BotPage's native counterpart is actually mounted.
func (s *Service) contactsActivity(ep harness.Endpoint, catalog pluginrpc.DshbotContactCatalog) map[string]dshbotstore.ContactActivity {
	out := map[string]dshbotstore.ContactActivity{}
	host := s.safeHost()
	s.mu.Lock()
	run := s.botNotifications
	s.mu.Unlock()
	sessions := host.Sessions()
	client := s.roomClient(ep)
	if client == nil {
		return out
	}
	if run != nil && run.client == client && s.botNotificationCurrent(run) {
		activityCopy := run.snapshotContactActivity()
		for _, activity := range activityCopy {
			for _, row := range catalog.Items {
				if row.ID == activity.ID && row.SessionID == activity.SessionID && activity.Error == "" {
					out[row.ID] = dshbotstore.ContactActivity{
						Active:          activity.Attention,
						ActivityAt:      activity.ActivityAt,
						Preview:         activity.Preview,
						ReadInitialized: activity.ReadInitialized,
						ActivitySeq:     activity.ActivitySeq,
						LastSeenSeq:     activity.LastSeenSeq,
					}
					break
				}
			}
		}
	}
	if sessions != nil {
		for _, item := range sessions.ListedItems() {
			for _, row := range catalog.Items {
				own := item.SessionID == row.SessionID
				member := row.Kind == "room" && (item.ParentID == row.SessionID || pluginrpc.DshbotContactOwnsMember(row, item.SessionID))
				if !own && !member {
					continue
				}
				live := out[row.ID]
				live.Active = live.Active || item.Running || item.ContactError != ""
				// Original recency sorts by the canonical session's timestamp, not a
				// member timestamp. Member live state contributes only the active flag.
				if own && int64(item.UpdatedAt) > live.ActivityAt {
					live.ActivityAt = int64(item.UpdatedAt)
				}
				out[row.ID] = live
			}
		}
	}
	runtimes := s.contactRoomRuntimes(ep, catalog)
	runtime := s.assistantRoomRuntime.Snapshot()
	if runtime.Loaded && runtimes[runtime.Data.Room.ID].data.Room.ID == "" {
		id := runtime.Data.Room.ID
		if runtime.Scope == endpointScope(clientBinding(client), ep.Origin, ep.Cookie, id) {
			for _, row := range catalog.Items {
				if row.ID == id && row.SessionID == runtime.Data.Room.SessionID {
					out[id] = contactRoomActivity(out[id], runtime.Data)
					break
				}
			}
		}
	}
	for id, runtime := range runtimes {
		for _, row := range catalog.Items {
			if row.ID != id || row.Kind != "room" {
				continue
			}
			live := out[id]
			if row.SessionID == runtime.data.Room.SessionID {
				live = contactRoomActivity(live, runtime.data)
			}
			if runtime.err != "" {
				live.Active = true
				if live.Preview == "" {
					live.Preview = runtime.err
				}
			}
			out[id] = live
			break
		}
	}
	return out
}

func contactRoomActivity(live dshbotstore.ContactActivity, runtime pluginrpc.DshbotRoomRuntime) dshbotstore.ContactActivity {
	for _, member := range runtime.Members {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(member.Raw, &fields)
		status := strings.ToLower(member.Status)
		active := contactRuntimeTruth(fields["running"]) || contactRuntimeTruth(member.Pending) || contactRuntimeTruth(member.Error) || contactRuntimeTruth(member.LastError)
		for _, term := range []string{"running", "pending", "waiting", "approval", "question", "error", "failed"} {
			active = active || strings.Contains(status, term)
		}
		live.Active = live.Active || active
	}
	return live
}

func contactRuntimeTruth(raw json.RawMessage) bool {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		return v != ""
	default:
		return true
	}
}
