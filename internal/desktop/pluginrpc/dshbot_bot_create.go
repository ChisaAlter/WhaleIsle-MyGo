package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// e9f584be profile-ops.js accepts caller-owned id/sessionId for individual
// Bots. They correlate unknown outcomes; no requestId deduplication exists.
// This first native flow deliberately creates no objective, routine or peers.
type DshbotBotDraft struct {
	Name, Title, Description, WorkspaceID, ScratchCwd string
	Provider, Model, ReasoningEffort                  string
	MemoryReview, Notify                              bool
}
type DshbotBotItem struct {
	DshbotCatalogItem
	Profile map[string]json.RawMessage
}
type DshbotBotCatalog struct {
	Revision int
	Items    []DshbotBotItem
}
type DshbotBotCreateCommand struct {
	ID, SessionID string
	Expected      DshbotBotCatalog
	Draft         DshbotBotDraft
}
type DshbotBotCreateResult struct {
	Attempted, Rejected bool
	Catalog             DshbotBotCatalog
	Item                DshbotBotItem
}

func CloneDshbotBotItem(v DshbotBotItem) DshbotBotItem {
	v.MemberBotIDs = append([]string(nil), v.MemberBotIDs...)
	p := make(map[string]json.RawMessage, len(v.Profile))
	for k, raw := range v.Profile {
		p[k] = append(json.RawMessage(nil), raw...)
	}
	v.Profile = p
	return v
}
func CloneDshbotBotCatalog(v DshbotBotCatalog) DshbotBotCatalog {
	out := DshbotBotCatalog{Revision: v.Revision, Items: make([]DshbotBotItem, len(v.Items))}
	for i, item := range v.Items {
		out.Items[i] = CloneDshbotBotItem(item)
	}
	return out
}
func CloneDshbotBotCommand(v DshbotBotCreateCommand) DshbotBotCreateCommand {
	v.Expected = CloneDshbotBotCatalog(v.Expected)
	return v
}
func sameDshbotBotItem(a, b DshbotBotItem) bool {
	if a.ID != b.ID || a.SessionID != b.SessionID || a.Kind != b.Kind || len(a.Profile) != len(b.Profile) {
		return false
	}
	for key, raw := range a.Profile {
		if !sameCapabilityJSON(raw, b.Profile[key]) {
			return false
		}
	}
	return true
}
func SameDshbotBotCatalog(a, b DshbotBotCatalog) bool {
	if a.Revision != b.Revision || len(a.Items) != len(b.Items) {
		return false
	}
	for i := range a.Items {
		if !sameDshbotBotItem(a.Items[i], b.Items[i]) {
			return false
		}
	}
	return true
}
func decodeDshbotBotItem(raw json.RawMessage) (DshbotBotItem, error) {
	var out DshbotBotItem
	if !validCapabilityUnicode(raw) || json.Unmarshal(raw, &out.DshbotCatalogItem) != nil || json.Unmarshal(raw, &out.Profile) != nil || out.Profile == nil || out.ID == "" || out.ID != trimBotMemory(out.ID) || out.SessionID == "" || out.SessionID != trimBotMemory(out.SessionID) || (out.Kind != "bot" && out.Kind != "room") {
		return DshbotBotItem{}, fmt.Errorf("catalog contains incomplete Bot identities")
	}
	return out, nil
}
func decodeDshbotBotCatalog(raw json.RawMessage) (DshbotBotCatalog, error) {
	var wire struct {
		NS       string `json:"ns"`
		Revision *int   `json:"revision"`
		Value    struct {
			Items *[]json.RawMessage `json:"items"`
		} `json:"value"`
	}
	if !validCapabilityUnicode(raw) || json.Unmarshal(raw, &wire) != nil || wire.NS != "dsh-bot" || wire.Revision == nil || *wire.Revision < 0 || wire.Value.Items == nil {
		return DshbotBotCatalog{}, fmt.Errorf("Host returned no complete Bot catalog/revision; use the retained Bot Web entry")
	}
	out := DshbotBotCatalog{Revision: *wire.Revision, Items: []DshbotBotItem{}}
	ids, sessions := map[string]bool{}, map[string]bool{}
	for _, raw := range *wire.Value.Items {
		item, err := decodeDshbotBotItem(raw)
		if err != nil {
			return DshbotBotCatalog{}, err
		}
		if ids[item.ID] || sessions[item.SessionID] {
			return DshbotBotCatalog{}, fmt.Errorf("ambiguous Bot identity")
		}
		ids[item.ID] = true
		sessions[item.SessionID] = true
		out.Items = append(out.Items, item)
	}
	return out, nil
}
func (c *Client) DshbotBotCatalog(ctx context.Context) (DshbotBotCatalog, error) {
	var reply struct {
		View json.RawMessage `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{}, &reply); err != nil {
		return DshbotBotCatalog{}, err
	}
	return decodeDshbotBotCatalog(reply.View)
}
func ValidateDshbotBotCreate(cmd DshbotBotCreateCommand) error {
	d := cmd.Draft
	for _, f := range []struct {
		value    string
		max      int
		required bool
	}{{cmd.ID, 200, true}, {cmd.SessionID, 240, true}, {d.Name, 120, true}, {d.Title, 120, false}, {d.Description, 12000, false}, {d.WorkspaceID, 2000, false}, {d.ScratchCwd, 2000, d.WorkspaceID == ""}, {d.Provider, 200, false}, {d.Model, 500, false}, {d.ReasoningEffort, 100, false}} {
		if !utf8.ValidString(f.value) || f.value != trimBotMemory(f.value) || (f.required && f.value == "") || WhaleTextLength(f.value) > f.max {
			return fmt.Errorf("invalid Bot fields: require name and workspace or scratch cwd; respect the displayed text limits")
		}
	}
	if (d.Provider == "") != (d.Model == "") {
		return fmt.Errorf("Model provider and model must be selected together.")
	}
	if d.Provider == "" && d.ReasoningEffort != "" {
		return fmt.Errorf("Reasoning effort requires a selected model.")
	}
	if cmd.Expected.Revision < 0 {
		return fmt.Errorf("refresh the Bot catalog first")
	}
	for _, row := range cmd.Expected.Items {
		if row.ID == cmd.ID || row.SessionID == cmd.SessionID {
			return fmt.Errorf("Bot identity already exists; read back instead of creating again")
		}
		if row.Kind == "bot" && strings.EqualFold(trimBotMemory(row.Name), d.Name) {
			return fmt.Errorf("a Bot with this name already exists")
		}
	}
	return nil
}
func dshbotBotCreateProfile(cmd DshbotBotCreateCommand) map[string]json.RawMessage {
	d := cmd.Draft
	all := map[string]any{"mode": "all", "names": []string{}}
	values := map[string]any{"id": cmd.ID, "sessionId": cmd.SessionID, "kind": "bot", "name": d.Name, "title": d.Title, "description": d.Description, "workspaceId": d.WorkspaceID, "sectionId": "", "avatar": map[string]string{"kind": "blob", "shape": "circle", "color": "blue"}, "model": map[string]string{"provider": d.Provider, "model": d.Model, "reasoningEffort": d.ReasoningEffort}, "capabilities": map[string]any{"tools": all, "skills": all, "mcp": all}, "allowedSenderIds": []string{}, "memberBotIds": []string{}, "objective": "", "objectiveMaxRounds": 16, "memoryReview": d.MemoryReview, "notify": d.Notify, "maxRounds": 3, "maxSpeaks": 10, "hidden": false, "pinned": false}
	out := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		out[key], _ = json.Marshal(value)
	}
	return out
}
func DshbotCreatedBot(cmd DshbotBotCreateCommand, v DshbotBotCatalog) (DshbotBotItem, error) {
	if v.Revision <= cmd.Expected.Revision {
		return DshbotBotItem{}, fmt.Errorf("no newer catalog receipt; outcome remains locked")
	}
	for _, row := range v.Items {
		if row.ID != cmd.ID {
			continue
		}
		if row.Kind != "bot" || row.SessionID != cmd.SessionID {
			return DshbotBotItem{}, fmt.Errorf("Bot canonical identity differs")
		}
		for key, value := range dshbotBotCreateProfile(cmd) {
			matches := sameCapabilityJSON(value, row.Profile[key])
			if key == "avatar" {
				matches = sameDshbotAvatarReceipt(value, row.Profile[key])
			}
			if !matches {
				return DshbotBotItem{}, fmt.Errorf("Bot reviewed field %s differs or is missing; inspect the existing record", key)
			}
		}
		return CloneDshbotBotItem(row), nil
	}
	return DshbotBotItem{}, fmt.Errorf("no exact Bot record yet; missing readback never authorizes recreation")
}
func (c *Client) DshbotCreateBot(ctx context.Context, cmd DshbotBotCreateCommand) (DshbotBotCreateResult, error) {
	var out DshbotBotCreateResult
	if err := ValidateDshbotBotCreate(cmd); err != nil {
		out.Rejected = true
		return out, err
	}
	payload := dshbotBotCreateProfile(cmd)
	// These are Host-normalized personal-Bot defaults, never room-only inputs.
	for _, key := range []string{"memberBotIds", "maxRounds", "maxSpeaks", "hidden", "pinned"} {
		delete(payload, key)
	}
	payload["revision"], _ = json.Marshal(cmd.Expected.Revision)
	payload["scratchCwd"], _ = json.Marshal(cmd.Draft.ScratchCwd)
	var reply struct {
		View, Item json.RawMessage
		SessionID  string `json:"sessionId"`
	}
	out.Attempted = true
	if err := c.Call(ctx, "dshbot", "bot/create", payload, &reply); err != nil {
		// Only exact server errors known to precede scope.set release this draft.
		// Unknown model/session errors or rollback failures may be post-write.
		message := err.Error()
		if strings.HasPrefix(message, "pluginrpc: ") && !strings.HasPrefix(message, "pluginrpc: HTTP ") {
			_, host, envelope := strings.Cut(strings.TrimPrefix(message, "pluginrpc: "), ": ")
			if envelope {
				for _, known := range []string{"Bot catalog is read-only.", "Bot catalog changed. Refresh and try again.", "Bot identity already exists.", "Model provider and model must be selected together.", "Reasoning effort requires a selected model.", "No application default model is configured for this Bot. Select a Bot model first."} {
					if host == known {
						out.Rejected = true
					}
				}
			}
		}
		return out, err
	}
	var err error
	out.Catalog, err = decodeDshbotBotCatalog(reply.View)
	if err != nil {
		return out, err
	}
	out.Item, err = DshbotCreatedBot(cmd, out.Catalog)
	if err != nil {
		return out, err
	}
	item, err := decodeDshbotBotItem(reply.Item)
	if err != nil || reply.SessionID != cmd.SessionID || !sameDshbotCreatedItemReceipt(item, out.Item) {
		return out, fmt.Errorf("inconsistent Bot creation receipt; read back without retry")
	}
	return out, nil
}
