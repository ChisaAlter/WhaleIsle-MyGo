package pluginrpc

import (
	"encoding/json"
	"fmt"
	"slices"
)

// The e9 room deletion branch is a soft-disband, not deletion of member Bots.
// Refuse malformed cross-room/peer references instead of consenting to a wider
// cascade than the native room review describes.
func dshbotRoomDeleteMembers(c DshbotDeleteCatalog, target DshbotBotItem) ([]DshbotBotItem, error) {
	ids, err := deleteStrings(target.Profile, "memberBotIds")
	if err != nil {
		return nil, err
	}
	out := []DshbotBotItem{}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			return nil, fmt.Errorf("room deletion requires unambiguous member Bot identities")
		}
		seen[id] = true
		found := false
		for _, item := range c.Items {
			if item.ID == id {
				if item.Kind != "bot" || item.SessionID == "" {
					return nil, fmt.Errorf("room member is not a personal Bot")
				}
				out = append(out, CloneDshbotBotItem(item))
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("room member identity is unavailable; refresh without deleting")
		}
	}
	return out, nil
}
func addDshbotRoomDeleteImpact(c DshbotDeleteCatalog, target DshbotBotItem, impact map[string]any) error {
	members, err := dshbotRoomDeleteMembers(c, target)
	if err != nil {
		return err
	}
	for _, item := range c.Items {
		ids, e := deleteStrings(item.Profile, "memberBotIds")
		senders, e2 := deleteStrings(item.Profile, "allowedSenderIds")
		if e != nil || e2 != nil {
			return fmt.Errorf("incomplete room deletion associations")
		}
		if item.Kind == "room" && slices.Contains(ids, target.ID) || item.Kind == "bot" && slices.Contains(senders, target.ID) {
			return fmt.Errorf("room ID appears in an unexpected member/source association; wider cascading deletion is not authorized")
		}
	}
	retained := []any{}
	memberIDs := []string{}
	for _, member := range members {
		memberIDs = append(memberIDs, member.ID)
		retained = append(retained, map[string]any{"id": member.ID, "name": member.Name, "title": member.Title, "canonicalSession": member.SessionID, "catalogEntryRemoved": false, "inboxEffect": "only related room task/routine entries may be removed; see inboxEntriesRemoved"})
	}
	impact["targetKind"] = "room"
	impact["memberBotCatalogEntriesRetained"] = retained
	impact["disbandedGroupMemberSessionsRetained"] = []any{map[string]any{"roomId": target.ID, "roomSessionId": target.SessionID, "memberBotIds": memberIDs, "effect": "group member Sessions remain hidden and intact; member Bot canonical Sessions also remain"}}
	impact["roomStopEffects"] = map[string]any{"roomId": target.ID, "roomSessionId": target.SessionID, "memberBotIds": memberIDs, "effects": []string{"abort active member turns", "advance room turn epoch", "attempt to cancel room agent with keepInbox true before catalog cleanup"}, "failureBoundary": "best effort; cancellation may precede later catalog/presentation/notepad failure"}
	impact["historyPreservationIsNotRecovery"] = "retained Session history/memory does not restore removed room configuration, routines, task state, inbox entries, audit pruning or notepad files"
	return nil
}
func validateDshbotRoomDeleteReceipt(cmd DshbotBotDeleteCommand, result DshbotBotDeleteResult) error {
	members, err := dshbotRoomDeleteMembers(cmd.Plan.Catalog, cmd.Plan.Target)
	if err != nil {
		return err
	}
	for _, member := range members {
		found := false
		for _, item := range result.Catalog.Items {
			if item.ID == member.ID && item.Kind == "bot" && item.SessionID == member.SessionID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("room deletion receipt does not retain reviewed member Bot %s and its canonical identity; outcome remains unknown", member.ID)
		}
	}
	return nil
}
func dshbotRoomDeleteObservation(p DshbotBotDeletePlan, c DshbotDeleteCatalog) []any {
	ids, _ := deleteStrings(p.Target.Profile, "memberBotIds")
	out := []any{}
	for _, id := range ids {
		state := "absent"
		var current map[string]json.RawMessage
		for _, item := range c.Items {
			if item.ID == id {
				state = "present"
				current = item.Profile
				break
			}
		}
		out = append(out, map[string]any{"reviewedMemberBotId": id, "catalogState": state, "currentProfile": current, "causality": "observation only; the original room deletion attempt remains unknown"})
	}
	return out
}
