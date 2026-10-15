package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// e9f584be profile-ops duplicate generates new identities inside the Host.
// Neither a caller id/sessionId nor requestId can correlate a lost receipt.
var DshbotBotDuplicateConfigKeys = []string{"title", "description", "avatar", "workspaceId", "model", "capabilities", "allowedSenderIds", "objective", "objectiveMaxRounds", "memoryReview", "notify", "sectionId"}

type DshbotBotDuplicateCommand struct {
	SourceID, Name, ScratchCwd string
	Expected                   DshbotBotCatalog
	AllowObjectiveExecution    bool // local explicit consent only; never a wire field
}
type DshbotBotDuplicateResult struct {
	Attempted, Rejected bool
	Catalog             DshbotBotCatalog
	Item                DshbotBotItem
	Warnings            []string
}

func CloneDshbotBotDuplicateCommand(v DshbotBotDuplicateCommand) DshbotBotDuplicateCommand {
	v.Expected = CloneDshbotBotCatalog(v.Expected)
	return v
}
func DshbotBotDuplicateSource(cmd DshbotBotDuplicateCommand) (DshbotBotItem, error) {
	var source DshbotBotItem
	found := 0
	for _, item := range cmd.Expected.Items {
		if item.ID == cmd.SourceID {
			source = item
			found++
		}
	}
	if found != 1 || source.Kind != "bot" || source.Hidden || source.SessionID == "" {
		return source, fmt.Errorf("select a current visible personal Bot")
	}
	policies, err := decodeCapabilityPolicies(source.Profile["capabilities"])
	if err != nil {
		return source, err
	}
	status := DshbotCapabilityStatus{Bot: source.DshbotCatalogItem, Revision: cmd.Expected.Revision, Profile: source.Profile, Policies: policies}
	for _, item := range cmd.Expected.Items {
		if item.ID != source.ID && item.Kind == "bot" {
			status.PeerIDs = append(status.PeerIDs, item.ID)
		}
	}
	if err := ValidateDshbotCapabilityProfile(status); err != nil {
		return source, fmt.Errorf("source cannot be duplicated without normalization uncertainty: %w", err)
	}
	allowed := append(append([]string{}, DshbotBotDuplicateConfigKeys...), "id", "sessionId", "kind", "name", "pinned", "hidden", "pinOrder", "memberBotIds", "maxRounds", "maxSpeaks", "inbox", "readInitialized", "lastSeenSeq", "createdAt", "updatedAt", "holds", "holdCheckpoint")
	for key := range source.Profile {
		if !slices.Contains(allowed, key) {
			return source, fmt.Errorf("unknown source field %s; use the existing Bot Web entry rather than silently dropping configuration", key)
		}
	}
	return CloneDshbotBotItem(source), nil
}
func DshbotBotDuplicateHasObjective(cmd DshbotBotDuplicateCommand) bool {
	for _, item := range cmd.Expected.Items {
		if item.ID == cmd.SourceID {
			var goal string
			json.Unmarshal(item.Profile["objective"], &goal)
			return trimBotMemory(goal) != ""
		}
	}
	return false
}
func ValidateDshbotBotDuplicate(cmd DshbotBotDuplicateCommand) error {
	source, err := DshbotBotDuplicateSource(cmd)
	if err != nil {
		return err
	}
	if cmd.Expected.Revision < 0 || !utf8.ValidString(cmd.Name) || cmd.Name == "" || cmd.Name != trimBotMemory(cmd.Name) || WhaleTextLength(cmd.Name) > 120 {
		return fmt.Errorf("enter a distinct name of at most 120 UTF-16 units and refresh the original catalog")
	}
	if !utf8.ValidString(cmd.ScratchCwd) || cmd.ScratchCwd != trimBotMemory(cmd.ScratchCwd) || WhaleTextLength(cmd.ScratchCwd) > 2000 || (source.WorkspaceID == "" && cmd.ScratchCwd == "") {
		return fmt.Errorf("scratch cwd is required without a workspace and must be at most 2000 UTF-16 units")
	}
	for _, item := range cmd.Expected.Items {
		if item.Kind == "bot" && strings.EqualFold(trimBotMemory(item.Name), cmd.Name) {
			return fmt.Errorf("a personal Bot with this name already exists")
		}
	}
	return nil
}
func dshbotDuplicateCandidate(cmd DshbotBotDuplicateCommand, row DshbotBotItem) bool {
	source, err := DshbotBotDuplicateSource(cmd)
	if err != nil {
		return false
	}
	if row.ID == "" || row.SessionID == "" || row.Kind != "bot" || row.Name != cmd.Name {
		return false
	}
	for _, old := range cmd.Expected.Items {
		if old.ID == row.ID || old.SessionID == row.SessionID {
			return false
		}
	}
	for _, key := range DshbotBotDuplicateConfigKeys {
		if !sameCapabilityJSON(source.Profile[key], row.Profile[key]) {
			return false
		}
	}
	for key, value := range map[string]any{"kind": "bot", "name": cmd.Name, "hidden": false, "pinned": false, "pinOrder": 0, "inbox": []any{}, "memberBotIds": []string{}, "readInitialized": true, "lastSeenSeq": 0, "maxRounds": 3, "maxSpeaks": 10} {
		raw, _ := json.Marshal(value)
		if !sameCapabilityJSON(raw, row.Profile[key]) {
			return false
		}
	}
	for _, key := range []string{"createdAt", "updatedAt"} {
		var timestamp float64
		if json.Unmarshal(row.Profile[key], &timestamp) != nil || timestamp <= 0 {
			return false
		}
	}
	return true
}

// Candidates are configuration matches, never proof of which request created
// them. Even a single candidate or an empty catalog cannot settle an unknown.
func DshbotBotDuplicateCandidates(cmd DshbotBotDuplicateCommand, v DshbotBotCatalog) []DshbotBotItem {
	out := []DshbotBotItem{}
	if v.Revision <= cmd.Expected.Revision {
		return out
	}
	for _, row := range v.Items {
		if dshbotDuplicateCandidate(cmd, row) {
			out = append(out, CloneDshbotBotItem(row))
		}
	}
	return out
}
func DshbotDuplicatedBot(cmd DshbotBotDuplicateCommand, v DshbotBotCatalog, id, sessionID string) (DshbotBotItem, error) {
	if v.Revision <= cmd.Expected.Revision || id == "" || sessionID == "" {
		return DshbotBotItem{}, fmt.Errorf("no complete duplicate receipt")
	}
	for _, row := range v.Items {
		if row.ID == id && row.SessionID == sessionID && dshbotDuplicateCandidate(cmd, row) {
			return CloneDshbotBotItem(row), nil
		}
	}
	return DshbotBotItem{}, fmt.Errorf("duplicate receipt identity/configuration mismatch; original attempt remains unknown")
}
func (c *Client) DshbotDuplicateBot(ctx context.Context, cmd DshbotBotDuplicateCommand) (DshbotBotDuplicateResult, error) {
	var out DshbotBotDuplicateResult
	if err := ValidateDshbotBotDuplicate(cmd); err != nil {
		out.Rejected = true
		return out, err
	}
	if DshbotBotDuplicateHasObjective(cmd) && !cmd.AllowObjectiveExecution {
		out.Rejected = true
		return out, fmt.Errorf("confirm separately that the copied objective may execute on agent creation/idle or later opening")
	}
	var reply struct {
		View, Item json.RawMessage
		SessionID  string   `json:"sessionId"`
		Warnings   []string `json:"warnings"`
	}
	out.Attempted = true
	err := c.Call(ctx, "dshbot", "bot/duplicate", map[string]any{"id": cmd.SourceID, "revision": cmd.Expected.Revision, "name": cmd.Name, "scratchCwd": cmd.ScratchCwd}, &reply)
	if err != nil {
		message := err.Error()
		if strings.HasPrefix(message, "pluginrpc: ") && !strings.HasPrefix(message, "pluginrpc: HTTP ") {
			_, host, envelope := strings.Cut(strings.TrimPrefix(message, "pluginrpc: "), ": ")
			if envelope {
				for _, known := range []string{"Bot catalog is read-only.", "Bot catalog changed. Refresh and try again.", "Bot profile is unavailable."} {
					if host == known {
						out.Rejected = true
					}
				}
			}
		}
		return out, err
	}
	out.Warnings = append([]string(nil), reply.Warnings...)
	out.Catalog, err = decodeDshbotBotCatalog(reply.View)
	if err != nil {
		return out, err
	}
	item, err := decodeDshbotBotItem(reply.Item)
	if err != nil {
		return out, err
	}
	out.Item, err = DshbotDuplicatedBot(cmd, out.Catalog, item.ID, reply.SessionID)
	if err != nil {
		return out, err
	}
	if !sameDshbotCreatedItemReceipt(item, out.Item) {
		return out, fmt.Errorf("duplicate item and catalog receipts disagree")
	}
	return out, nil
}

// An explicit open must still address the inspected configuration, especially
// its objective; volatile inbox/activity changes do not identify another Bot.
func SameDshbotBotDuplicateOpenTarget(a, b DshbotBotItem) bool {
	if a.ID == "" || a.ID != b.ID || a.SessionID == "" || a.SessionID != b.SessionID || a.Kind != "bot" || b.Kind != "bot" || b.Hidden || a.Name != b.Name {
		return false
	}
	for _, key := range DshbotBotDuplicateConfigKeys {
		if !sameCapabilityJSON(a.Profile[key], b.Profile[key]) {
			return false
		}
	}
	return true
}
