package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf16"
)

// Pinned e9f584be profile-ops.js accepts caller IDs, but has no requestId
// idempotency contract. IDs correlate readback; they never authorize a retry.
type DshbotRoomDraft struct {
	Name, Description, WorkspaceID, ScratchCwd string
	MemberBotIDs                               []string
	MaxRounds, MaxSpeaks                       int
}
type DshbotRoomItem struct {
	DshbotCatalogItem
	Description string `json:"description"`
	MaxRounds   int    `json:"maxRounds"`
	MaxSpeaks   int    `json:"maxSpeaks"`
}
type DshbotRoomCatalog struct {
	Revision int
	Items    []DshbotRoomItem
}
type DshbotRoomCreateCommand struct {
	ID, SessionID string
	Expected      DshbotRoomCatalog
	Draft         DshbotRoomDraft
}
type DshbotRoomCreateResult struct {
	Attempted, Rejected bool
	Catalog             DshbotRoomCatalog
	Item                DshbotRoomItem
}

func CloneDshbotRoomCatalog(v DshbotRoomCatalog) DshbotRoomCatalog {
	out := v
	out.Items = make([]DshbotRoomItem, len(v.Items))
	copy(out.Items, v.Items)
	for i := range out.Items {
		out.Items[i].MemberBotIDs = append([]string(nil), v.Items[i].MemberBotIDs...)
	}
	return out
}
func CloneDshbotRoomCommand(v DshbotRoomCreateCommand) DshbotRoomCreateCommand {
	v.Expected = CloneDshbotRoomCatalog(v.Expected)
	v.Draft.MemberBotIDs = append([]string(nil), v.Draft.MemberBotIDs...)
	return v
}
func SameDshbotRoomCatalog(a, b DshbotRoomCatalog) bool {
	return reflect.DeepEqual(CloneDshbotRoomCatalog(a), CloneDshbotRoomCatalog(b))
}
func decodeDshbotRoomCatalog(raw json.RawMessage) (DshbotRoomCatalog, error) {
	var wire struct {
		NS       string `json:"ns"`
		Revision *int   `json:"revision"`
		Value    struct {
			Items *[]DshbotRoomItem `json:"items"`
		} `json:"value"`
	}
	if json.Unmarshal(raw, &wire) != nil || wire.NS != "dsh-bot" || wire.Revision == nil || *wire.Revision < 0 || wire.Value.Items == nil {
		return DshbotRoomCatalog{}, fmt.Errorf("Host returned no complete dsh-bot catalog/revision; use the existing Bot Web entry")
	}
	out := DshbotRoomCatalog{Revision: *wire.Revision, Items: *wire.Value.Items}
	seen, sessions := map[string]bool{}, map[string]bool{}
	for _, item := range out.Items {
		if strings.TrimSpace(item.ID) == "" || item.ID != strings.TrimSpace(item.ID) || item.SessionID == "" || item.SessionID != strings.TrimSpace(item.SessionID) || seen[item.ID] || sessions[item.SessionID] || (item.Kind != "bot" && item.Kind != "room") {
			return DshbotRoomCatalog{}, fmt.Errorf("catalog contains incomplete or ambiguous Bot identities")
		}
		seen[item.ID] = true
		sessions[item.SessionID] = true
	}
	return out, nil
}
func (c *Client) DshbotRoomCatalog(ctx context.Context) (DshbotRoomCatalog, error) {
	var out struct {
		View json.RawMessage `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{}, &out); err != nil {
		return DshbotRoomCatalog{}, err
	}
	return decodeDshbotRoomCatalog(out.View)
}
func ValidateDshbotRoomCreate(cmd DshbotRoomCreateCommand) error {
	d := cmd.Draft
	for _, f := range []struct {
		value    string
		limit    int
		required bool
	}{{cmd.ID, 200, true}, {cmd.SessionID, 240, true}, {d.Name, 120, true}, {d.Description, 12000, false}, {d.WorkspaceID, 2000, false}, {d.ScratchCwd, 2000, d.WorkspaceID == ""}} {
		if f.value != strings.TrimSpace(f.value) || (f.required && f.value == "") || len(utf16.Encode([]rune(f.value))) > f.limit {
			return fmt.Errorf("invalid room text/identity; name is required (120 UTF-16 units), description ≤12000, workspace/cwd ≤2000")
		}
	}
	if cmd.Expected.Revision < 0 {
		return fmt.Errorf("refresh the Host catalog before creating")
	}
	if d.MaxRounds < 1 || d.MaxRounds > 3 || d.MaxSpeaks < 1 || d.MaxSpeaks > 10 {
		return fmt.Errorf("group limits must be 1–3 rounds and 1–10 speaks")
	}
	if len(d.MemberBotIDs) < 2 || len(d.MemberBotIDs) > 6 {
		return fmt.Errorf("choose 2–6 available Bots")
	}
	available := map[string]bool{}
	for _, item := range cmd.Expected.Items {
		if item.ID == cmd.ID || item.SessionID == cmd.SessionID {
			return fmt.Errorf("room identity already exists; read back, do not create again")
		}
		if item.Kind == "room" && strings.EqualFold(strings.TrimSpace(item.Name), d.Name) {
			return fmt.Errorf("a group with this name already exists")
		}
		if item.Kind == "bot" && item.ID != "" && item.SessionID != "" {
			available[item.ID] = true
		}
	}
	seen := map[string]bool{}
	for _, id := range d.MemberBotIDs {
		if !available[id] || seen[id] {
			return fmt.Errorf("members must be distinct current Bot IDs, not rooms or names")
		}
		seen[id] = true
	}
	return nil
}
func DshbotCreatedRoom(cmd DshbotRoomCreateCommand, v DshbotRoomCatalog) (DshbotRoomItem, error) {
	if v.Revision <= cmd.Expected.Revision {
		return DshbotRoomItem{}, fmt.Errorf("no newer catalog receipt; read back before another action")
	}
	for _, row := range v.Items {
		if row.ID != cmd.ID {
			continue
		}
		d := cmd.Draft
		if row.Kind != "room" || row.SessionID != cmd.SessionID || row.Name != d.Name || row.Description != d.Description || row.WorkspaceID != d.WorkspaceID || row.MaxRounds != d.MaxRounds || row.MaxSpeaks != d.MaxSpeaks || !reflect.DeepEqual(row.MemberBotIDs, d.MemberBotIDs) {
			return DshbotRoomItem{}, fmt.Errorf("room identity or reviewed values differ; inspect the existing record")
		}
		return row, nil
	}
	return DshbotRoomItem{}, fmt.Errorf("no exact room record yet; the unknown request remains locked")
}
func (c *Client) DshbotCreateRoom(ctx context.Context, cmd DshbotRoomCreateCommand) (DshbotRoomCreateResult, error) {
	var out DshbotRoomCreateResult
	if err := ValidateDshbotRoomCreate(cmd); err != nil {
		out.Rejected = true
		return out, err
	}
	d := cmd.Draft
	all := map[string]any{"mode": "all", "names": []string{}}
	payload := map[string]any{"kind": "room", "id": cmd.ID, "sessionId": cmd.SessionID, "revision": cmd.Expected.Revision, "name": d.Name, "description": d.Description, "workspaceId": d.WorkspaceID, "scratchCwd": d.ScratchCwd, "memberBotIds": d.MemberBotIDs, "maxRounds": d.MaxRounds, "maxSpeaks": d.MaxSpeaks, "model": map[string]string{"provider": "", "model": "", "reasoningEffort": ""}, "capabilities": map[string]any{"tools": all, "skills": all, "mcp": all}, "allowedSenderIds": []string{}}
	var reply struct {
		View      json.RawMessage `json:"view"`
		Item      DshbotRoomItem  `json:"item"`
		SessionID string          `json:"sessionId"`
	}
	out.Attempted = true
	if err := c.Call(ctx, "dshbot", "bot/create", payload, &reply); err != nil {
		// Only exact known failures before scope.set release the local attempt. A
		// session-create/rollback error, timeout or malformed receipt stays unknown.
		message := err.Error()
		if strings.HasPrefix(message, "pluginrpc: ") && !strings.HasPrefix(message, "pluginrpc: HTTP ") {
			for _, known := range []string{"Bot catalog is read-only.", "Bot catalog changed. Refresh and try again.", "Group members must contain 2-6 available Bots.", "Group limits are invalid.", "Bot identity already exists."} {
				_, hostMessage, envelope := strings.Cut(strings.TrimPrefix(message, "pluginrpc: "), ": ")
				if envelope && hostMessage == known {
					out.Rejected = true
				}
			}
		}
		return out, err
	}
	var err error
	out.Catalog, err = decodeDshbotRoomCatalog(reply.View)
	if err != nil {
		return out, err
	}
	out.Item, err = DshbotCreatedRoom(cmd, out.Catalog)
	if err != nil {
		return out, err
	}
	if reply.SessionID != cmd.SessionID || !reflect.DeepEqual(reply.Item, out.Item) {
		return out, fmt.Errorf("creation receipt has inconsistent canonical identity; read back")
	}
	return out, nil
}
