package pluginrpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"whaleisle/internal/store/storelocale"
)

type DshbotRoomRuntimeMember struct {
	RoomSessionID   string          `json:"roomSessionId"`
	BotID           string          `json:"botId"`
	SessionID       string          `json:"sessionId"`
	Status          string          `json:"status"`
	LastError       json.RawMessage `json:"lastError"`
	Error           json.RawMessage `json:"error"`
	Pending         json.RawMessage `json:"pending"`
	UpdatedAt       json.RawMessage `json:"updatedAt"`
	Raw             json.RawMessage `json:"-"`
	Name, OpenError string          `json:"-"`
}
type DshbotRoomRuntime struct {
	Catalog DshbotRoomCatalog
	Room    DshbotRoomItem
	Members []DshbotRoomRuntimeMember
	Raw     json.RawMessage
}

func CloneDshbotRoomRuntime(v DshbotRoomRuntime) DshbotRoomRuntime {
	v.Catalog = CloneDshbotRoomCatalog(v.Catalog)
	v.Room.MemberBotIDs = append([]string(nil), v.Room.MemberBotIDs...)
	v.Raw = append(json.RawMessage(nil), v.Raw...)
	v.Members = append([]DshbotRoomRuntimeMember(nil), v.Members...)
	for i := range v.Members {
		m := &v.Members[i]
		m.Raw = append(json.RawMessage(nil), m.Raw...)
		m.LastError = append(json.RawMessage(nil), m.LastError...)
		m.Error = append(json.RawMessage(nil), m.Error...)
		m.Pending = append(json.RawMessage(nil), m.Pending...)
		m.UpdatedAt = append(json.RawMessage(nil), m.UpdatedAt...)
	}
	return v
}

// Exact e9 root member identities; do not substitute a personal Bot session or
// invent a subagent address. Legacy aliases remain readable after upgrade.
func dshbotRoomMemberIdentity(roomSessionID, botID, sessionID string) bool {
	const prefix = "dshbot-group-member-v3-"
	if strings.HasPrefix(sessionID, prefix) {
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(sessionID, prefix))
		if err != nil || !validCapabilityUnicode(raw) {
			return false
		}
		var pair []string
		return json.Unmarshal(raw, &pair) == nil && len(pair) == 2 && pair[0] == roomSessionID && pair[1] == botID
	}
	digest := sha256.Sum256([]byte("root-v2\x00" + roomSessionID + "\x00" + botID))
	return sessionID == fmt.Sprintf("dshbot-group-member-%x", digest[:20])
}
func roomRuntimeOpenError(v DshbotRoomRuntime, m DshbotRoomRuntimeMember) string {
	if m.RoomSessionID != v.Room.SessionID || m.BotID == "" {
		return storelocale.T(storelocale.KeyDesktopPluginrpcMemberOwnershipUnavailable)
	}
	if !slices.Contains(v.Room.MemberBotIDs, m.BotID) {
		return storelocale.T(storelocale.KeyDesktopPluginrpcHistoricalMemberNotInCurrentRoom)
	}
	found := false
	for _, bot := range v.Catalog.Items {
		if bot.ID == m.BotID && bot.Kind == "bot" {
			found = true
		}
	}
	if !found {
		return storelocale.T(storelocale.KeyDesktopPluginrpcCurrentBotIdentityUnavailable)
	}
	if m.SessionID == "" {
		return storelocale.T(storelocale.KeyDesktopPluginrpcMemberSessionUnavailable)
	}
	if !dshbotRoomMemberIdentity(v.Room.SessionID, m.BotID, m.SessionID) {
		return storelocale.T(storelocale.KeyDesktopPluginrpcSessionDoesNotIdentifyThisRoomMemb)
	}
	return ""
}
func decodeDshbotRoomRuntime(catalog DshbotRoomCatalog, id string, raw json.RawMessage) (DshbotRoomRuntime, error) {
	out := DshbotRoomRuntime{Catalog: CloneDshbotRoomCatalog(catalog), Raw: append(json.RawMessage(nil), raw...)}
	for _, row := range catalog.Items {
		if row.ID == id && row.Kind == "room" && !row.Hidden {
			out.Room = row
			break
		}
	}
	if out.Room.ID == "" {
		return out, fmt.Errorf("selected room unavailable in current Host catalog")
	}
	var wire struct {
		RoomID    string             `json:"roomId"`
		SessionID string             `json:"sessionId"`
		Members   *[]json.RawMessage `json:"members"`
	}
	if !validCapabilityUnicode(raw) || json.Unmarshal(raw, &wire) != nil || wire.RoomID != id || wire.SessionID != out.Room.SessionID || wire.Members == nil {
		return out, fmt.Errorf("group/runtime unavailable or does not match the selected canonical room")
	}
	seenBots, seenSessions := map[string]int{}, map[string]int{}
	for _, r := range *wire.Members {
		var m DshbotRoomRuntimeMember
		var obj map[string]json.RawMessage
		if json.Unmarshal(r, &obj) != nil || obj == nil || json.Unmarshal(r, &m) != nil {
			return out, fmt.Errorf("group/runtime contains an invalid member record")
		}
		m.Raw = append(json.RawMessage(nil), r...)
		m.Name = m.BotID
		for _, bot := range catalog.Items {
			if bot.ID == m.BotID && bot.Kind == "bot" {
				m.Name = bot.Name
				break
			}
		}
		m.OpenError = roomRuntimeOpenError(out, m)
		out.Members = append(out.Members, m)
		seenBots[m.BotID]++
		seenSessions[m.SessionID]++
	}
	for i := range out.Members {
		m := &out.Members[i]
		if seenBots[m.BotID] > 1 || m.SessionID != "" && seenSessions[m.SessionID] > 1 {
			m.OpenError = storelocale.T(storelocale.KeyDesktopPluginrpcAmbiguousRuntimeIdentity)
		}
	}
	return out, nil
}

// Both calls are reads. Refreshing runtime never opens/follows a session,
// discovers a synthetic run RPC, replays prompts, or answers interactions.
func (c *Client) DshbotRoomRuntime(ctx context.Context, id string) (DshbotRoomRuntime, error) {
	catalog, err := c.DshbotRoomCatalog(ctx)
	if err != nil {
		return DshbotRoomRuntime{}, err
	}
	var raw json.RawMessage
	if err = c.Call(ctx, "dshbot", "group/runtime", map[string]any{"roomId": id}, &raw); err != nil {
		return DshbotRoomRuntime{}, err
	}
	return decodeDshbotRoomRuntime(catalog, id, raw)
}
func DshbotRoomRuntimeTarget(v DshbotRoomRuntime, sessionID string) (DshbotRoomRuntimeMember, error) {
	var found *DshbotRoomRuntimeMember
	for _, m := range v.Members {
		if m.SessionID != sessionID {
			continue
		}
		if found != nil {
			return DshbotRoomRuntimeMember{}, fmt.Errorf("ambiguous member session")
		}
		copy := m
		found = &copy
	}
	if found == nil {
		return DshbotRoomRuntimeMember{}, fmt.Errorf("member session is no longer in this runtime snapshot")
	}
	if found.OpenError != "" {
		return *found, fmt.Errorf("%s", found.OpenError)
	}
	if err := roomRuntimeOpenError(v, *found); err != "" {
		return *found, fmt.Errorf("%s", err)
	}
	return *found, nil
}
func DshbotRoomRuntimeJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "—"
	}
	var b bytes.Buffer
	if json.Indent(&b, raw, "", "  ") != nil {
		return string(raw)
	}
	return b.String()
}
func roomRuntimeValuePresent(raw json.RawMessage) bool {
	v := strings.TrimSpace(string(raw))
	return v != "" && v != "null" && v != "false" && v != `""` && v != "[]" && v != "{}"
}
func DshbotRoomRuntimeAttention(m DshbotRoomRuntimeMember) bool {
	s := strings.ToLower(m.Status)
	return m.OpenError != "" || roomRuntimeValuePresent(m.LastError) || roomRuntimeValuePresent(m.Error) || roomRuntimeValuePresent(m.Pending) || strings.Contains(s, "pending") || strings.Contains(s, "waiting") || strings.Contains(s, "approval") || strings.Contains(s, "question") || strings.Contains(s, "failed") || strings.Contains(s, "error") || strings.Contains(s, "timeout") || strings.Contains(s, "timed-out") || strings.Contains(s, "capped") || strings.Contains(s, "limit") || s == "held"
}
