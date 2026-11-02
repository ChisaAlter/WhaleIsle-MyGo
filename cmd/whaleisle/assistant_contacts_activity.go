package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
)

// Projects notification/session snapshots and read-only Room runtime observation
// while the original BotPage's native counterpart is actually mounted.
func (s *mainState) assistantContactsActivity(ep harness.Endpoint, catalog pluginrpc.DshbotContactCatalog) map[string]dshbotstore.ContactActivity {
	out := map[string]dshbotstore.ContactActivity{}
	s.mu.Lock()
	run, sessions := s.botNotifications, s.sessions
	s.mu.Unlock()
	client := s.assistantRoomClient(ep)
	if client == nil {
		return out
	}
	if run != nil && run.client == client && s.botNotificationCurrent(run) {
		for _, activity := range run.contactActivity {
			for _, row := range catalog.Items {
				if row.ID == activity.ID && row.SessionID == activity.SessionID && activity.Error == "" {
					out[row.ID] = dshbotstore.ContactActivity{Active: activity.Attention, ActivityAt: activity.ActivityAt, Preview: activity.Preview, ReadInitialized: activity.ReadInitialized, ActivitySeq: activity.ActivitySeq, LastSeenSeq: activity.LastSeenSeq}
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
	runtimes := s.assistantContactRoomRuntimes(ep, catalog)
	runtime := s.assistantRoomRuntime.Snapshot()
	if runtime.Loaded && runtimes[runtime.Data.Room.ID].data.Room.ID == "" {
		id := runtime.Data.Room.ID
		digest := sha256.Sum256([]byte(fmt.Sprintf("%p\x00%s\x00%s\x00%s", client, ep.Origin, ep.Cookie, id)))
		if runtime.Scope == fmt.Sprintf("%x", digest) {
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
