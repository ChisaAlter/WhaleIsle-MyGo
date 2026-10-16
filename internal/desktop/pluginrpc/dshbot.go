package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Dshbot HTTP control plane (vendored runtime/vendor/dshbot/lib/control-plane.js):
//   POST {origin}/dshbot/{endpoint}  Content-Type: application/json
//   Body: { type:"client-request", rpcId, method:<same as endpoint>, payload }
//   Response: { type:"server-response", rpcId, result:{ ok, value|error } }
//
// Read-oriented endpoints:
//   describe          payload {} or {botId}; value { view, capabilities }
//   group/runtime     payload {roomId}; value { roomId, sessionId, members[] }
//   bot/activity      payload {ids?}; value { entries[] }
//   routine/missed    payload {}; value { ids[] }
//   memory/get        payload { id }; value { botId, bot, user, limits }
//   bot/delete-preview payload { id }; value { item, dependencies, preserves }
//
// Credential mutation: bot/trigger-token payload {}; value { token }. This is
// Host-directory-wide get-or-create, not a read-only lookup; explicit consent only.
//
// Write endpoints (revision-gated where noted): catalog/items, bot/*, section/*,
// memory/replace, bot/objective/*, task/*, routine/*, bot/stop.
//
// There is no dshbot chat/pet RPC. Resolve the selected catalog item with bot/open,
// then send the unchanged prompt to that canonical harness session.
// Loopback-only: POST /dshbot-hook/routine/{id} + x-dshbot-token.

// DshbotSettingsView is the `view` field of describe (settings namespace dsh-bot).
type DshbotSettingsView struct {
	NS       string             `json:"ns"`
	Revision int                `json:"revision"`
	Value    DshbotCatalogValue `json:"value"`
	Schema   json.RawMessage    `json:"schema,omitempty"`
	Applies  string             `json:"applies,omitempty"`
}

// DshbotCatalogValue is the catalog document embedded in describe.view.value.
type DshbotCatalogValue struct {
	Items    []DshbotCatalogItem `json:"items"`
	Sections []DshbotSection     `json:"sections,omitempty"`
}

// DshbotCatalogItem is one bot or room row in the catalog.
type DshbotCatalogItem struct {
	ReadInitialized bool     `json:"readInitialized"`
	LastSeenSeq     int      `json:"lastSeenSeq"`
	Notify          *bool    `json:"notify"` // nil is unavailable evidence, never guessed true
	ID              string   `json:"id"`
	Kind            string   `json:"kind"`
	SessionID       string   `json:"sessionId"`
	WorkspaceID     string   `json:"workspaceId"`
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Hidden          bool     `json:"hidden"`
	Pinned          bool     `json:"pinned"`
	MemberBotIDs    []string `json:"memberBotIds"`
}

// DshbotSection is a roster grouping section.
type DshbotSection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DshbotDescribeResult is the value of dshbot/describe.
type DshbotDescribeResult struct {
	View         DshbotSettingsView `json:"view"`
	Capabilities json.RawMessage    `json:"capabilities,omitempty"`
}

// DshbotRosterRow is a UI-friendly bot or room entry from describe.
type DshbotRosterRow struct {
	MemberBotIDs []string
	ID           string
	Kind         string // "bot" or "room"
	Name         string
	Title        string
	SessionID    string
	Hidden       bool
	Pinned       bool
	MemberCount  int
	SectionID    string
	DisplayLabel string
}

// DshbotGroupRuntimeResult is the value of dshbot/group/runtime.
type DshbotGroupRuntimeResult struct {
	RoomID    string                     `json:"roomId"`
	SessionID string                     `json:"sessionId"`
	Members   []DshbotGroupRuntimeMember `json:"members"`
}

// DshbotGroupRuntimeMember is one live group-member runtime row.
type DshbotGroupRuntimeMember struct {
	RoomSessionID string `json:"roomSessionId"`
	BotID         string `json:"botId"`
	SessionID     string `json:"sessionId"`
	Status        string `json:"status"`
}

// DshbotActivityResult is the value of dshbot/bot/activity.
type DshbotActivityResult struct {
	Entries []DshbotActivityEntry `json:"entries"`
}

// DshbotActivityEntry summarizes session activity for one catalog item.
type DshbotActivityEntry struct {
	ReadInitialized        bool            `json:"readInitialized"`
	LastSeenSeq            int             `json:"lastSeenSeq"`
	NotificationShapeKnown bool            `json:"-"`
	Goal                   json.RawMessage `json:"goal"`
	LastRun                json.RawMessage `json:"lastRun"`
	BotID                  string          `json:"botId"`
	SessionID              string          `json:"sessionId"`
	Preview                string          `json:"preview"`
	Attention              bool            `json:"attention"`
	ActivityAt             int64           `json:"activityAt"`
	ActivitySeq            int             `json:"activitySeq"`
	Error                  string          `json:"error,omitempty"`
}

// DshbotDescribeTyped calls describe and decodes the catalog view.
func (c *Client) DshbotDescribeTyped(ctx context.Context, botID string) (DshbotDescribeResult, error) {
	payload := map[string]any{}
	if id := strings.TrimSpace(botID); id != "" {
		payload["botId"] = id
	}
	var out DshbotDescribeResult
	if err := c.Call(ctx, "dshbot", "describe", payload, &out); err != nil {
		return DshbotDescribeResult{}, fmt.Errorf("pluginrpc: decode dshbot describe: %w", err)
	}
	if out.View.NS == "" {
		out.View.NS = "dsh-bot"
	}
	return out, nil
}

// RosterRowsFromDescribe builds sorted roster rows (bots then rooms; pinned first).
func RosterRowsFromDescribe(res DshbotDescribeResult, includeHidden bool) []DshbotRosterRow {
	items := res.View.Value.Items
	rows := make([]DshbotRosterRow, 0, len(items))
	for _, item := range items {
		if item.Hidden && !includeHidden {
			continue
		}
		kind := strings.TrimSpace(item.Kind)
		if kind == "" {
			kind = "bot"
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = item.ID
		}
		title := strings.TrimSpace(item.Title)
		label := name
		if kind == "bot" && title != "" {
			label = name + " · " + title
		}
		if kind == "room" {
			label = name
		}
		rows = append(rows, DshbotRosterRow{
			ID:           item.ID,
			Kind:         kind,
			Name:         name,
			Title:        title,
			SessionID:    strings.TrimSpace(item.SessionID),
			Hidden:       item.Hidden,
			Pinned:       item.Pinned,
			MemberCount:  len(item.MemberBotIDs),
			MemberBotIDs: append([]string(nil), item.MemberBotIDs...),
			DisplayLabel: label,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Pinned != rows[j].Pinned {
			return rows[i].Pinned && !rows[j].Pinned
		}
		if rows[i].Kind != rows[j].Kind {
			if rows[i].Kind == "bot" {
				return true
			}
			if rows[j].Kind == "bot" {
				return false
			}
		}
		return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
	})
	return rows
}

// DshbotGroupRuntime calls dshbot/group/runtime for live room member status.
func (c *Client) DshbotGroupRuntime(ctx context.Context, roomID string) (DshbotGroupRuntimeResult, error) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return DshbotGroupRuntimeResult{}, fmt.Errorf("pluginrpc: empty roomId")
	}
	var out DshbotGroupRuntimeResult
	err := c.Call(ctx, "dshbot", "group/runtime", map[string]any{"roomId": roomID}, &out)
	return out, err
}

// DshbotActivityTyped calls bot/activity (all items when ids is nil/empty).
func (c *Client) DshbotActivityTyped(ctx context.Context, ids []string) (DshbotActivityResult, error) {
	payload := map[string]any{}
	if len(ids) > 0 {
		payload["ids"] = ids
	}
	var out DshbotActivityResult
	err := c.Call(ctx, "dshbot", "bot/activity", payload, &out)
	return out, err
}

// DshbotOpen resolves or restores the chosen Bot/room's canonical session.
// The client includes the catalog revision as the original UI does, but the
// server's open operation is not CAS-gated; validate its returned identity.
func (c *Client) DshbotOpen(ctx context.Context, item DshbotCatalogItem, revision int, scratchCwd string) (string, error) {
	id, expectedSession := strings.TrimSpace(item.ID), strings.TrimSpace(item.SessionID)
	if id == "" || expectedSession == "" {
		return "", fmt.Errorf("pluginrpc: Bot profile has no canonical session")
	}
	var out struct {
		Item      DshbotCatalogItem `json:"item"`
		SessionID string            `json:"sessionId"`
	}
	err := c.Call(ctx, "dshbot", "bot/open", map[string]any{
		"id": id, "revision": revision, "scratchCwd": strings.TrimSpace(scratchCwd),
	}, &out)
	if err != nil {
		return "", err
	}
	if out.Item.ID != id || strings.TrimSpace(out.SessionID) != expectedSession || strings.TrimSpace(out.Item.SessionID) != expectedSession {
		return "", fmt.Errorf("pluginrpc: Bot session changed; refresh the roster and try again")
	}
	return expectedSession, nil
}
