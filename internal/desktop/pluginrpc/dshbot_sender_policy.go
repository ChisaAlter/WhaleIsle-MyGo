package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
)

// These are catalog Bot identities, never IM accounts or external recipients.
type DshbotSenderPolicyValues struct {
	Selected bool
	IDs      []string
}
type DshbotSenderPolicyStatus struct {
	DshbotCapabilityStatus
	CatalogItems      []json.RawMessage
	Peers             []DshbotCatalogItem
	SharedRoomPeerIDs []string
}
type DshbotSenderPolicyCommand struct {
	Expected DshbotSenderPolicyStatus
	Values   DshbotSenderPolicyValues
}

func CloneDshbotSenderPolicyValues(v DshbotSenderPolicyValues) DshbotSenderPolicyValues {
	v.IDs = append([]string{}, v.IDs...)
	return v
}
func SameDshbotSenderPolicyValues(a, b DshbotSenderPolicyValues) bool {
	return a.Selected == b.Selected && slices.Equal(a.IDs, b.IDs)
}
func CloneDshbotSenderPolicy(s DshbotSenderPolicyStatus) DshbotSenderPolicyStatus {
	s.DshbotCapabilityStatus = CloneDshbotCapabilities(s.DshbotCapabilityStatus)
	rows := s.CatalogItems
	s.CatalogItems = make([]json.RawMessage, len(rows))
	for i, r := range rows {
		s.CatalogItems[i] = append(json.RawMessage(nil), r...)
	}
	s.Peers = append([]DshbotCatalogItem{}, s.Peers...)
	for i := range s.Peers {
		s.Peers[i].MemberBotIDs = append([]string{}, s.Peers[i].MemberBotIDs...)
	}
	s.SharedRoomPeerIDs = append([]string{}, s.SharedRoomPeerIDs...)
	return s
}
func CloneDshbotSenderPolicyCommand(c DshbotSenderPolicyCommand) DshbotSenderPolicyCommand {
	c.Expected = CloneDshbotSenderPolicy(c.Expected)
	c.Values = CloneDshbotSenderPolicyValues(c.Values)
	return c
}
func SameDshbotSenderPolicy(a, b DshbotSenderPolicyStatus) bool {
	if !SameDshbotCapabilities(a.DshbotCapabilityStatus, b.DshbotCapabilityStatus) || !slices.Equal(a.PeerIDs, b.PeerIDs) || len(a.CatalogItems) != len(b.CatalogItems) {
		return false
	}
	for i, r := range a.CatalogItems {
		if !sameCapabilityJSON(r, b.CatalogItems[i]) {
			return false
		}
	}
	return true
}
func senderPolicyIDs(raw json.RawMessage) ([]string, error) {
	var ids []string
	if !validCapabilityUnicode(raw) || json.Unmarshal(raw, &ids) != nil || ids == nil {
		return nil, fmt.Errorf("sender IDs unavailable")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			return nil, fmt.Errorf("sender identity empty or duplicated")
		}
		seen[id] = true
	}
	return ids, nil
}
func DshbotSenderPolicyValue(s DshbotSenderPolicyStatus) (DshbotSenderPolicyValues, error) {
	ids, err := senderPolicyIDs(s.Profile["allowedSenderIds"])
	return DshbotSenderPolicyValues{Selected: len(ids) > 0, IDs: ids}, err
}
func decodeSenderCatalog(s *DshbotSenderPolicyStatus) error {
	if s.CatalogItems == nil {
		return fmt.Errorf("source catalog unavailable")
	}
	s.Peers = nil
	s.SharedRoomPeerIDs = nil
	seen := map[string]bool{}
	var rooms []DshbotCatalogItem
	target := 0
	for _, raw := range s.CatalogItems {
		var b DshbotCatalogItem
		if !validCapabilityUnicode(raw) || json.Unmarshal(raw, &b) != nil || b.ID == "" || seen[b.ID] || (b.Kind != "bot" && b.Kind != "room") {
			return fmt.Errorf("source catalog identity unavailable or ambiguous")
		}
		seen[b.ID] = true
		if b.ID == s.Bot.ID {
			target++
			if b.Kind != "bot" || b.SessionID != s.Bot.SessionID {
				return fmt.Errorf("source catalog target mismatch")
			}
		}
		if b.Kind == "room" {
			if b.MemberBotIDs == nil {
				return fmt.Errorf("room membership unavailable")
			}
			rooms = append(rooms, b)
		} else if b.ID != s.Bot.ID {
			s.Peers = append(s.Peers, b)
		}
	}
	if target != 1 {
		return fmt.Errorf("source catalog target missing")
	}
	peerIDs := []string{}
	for _, p := range s.Peers {
		peerIDs = append(peerIDs, p.ID)
	}
	if !slices.Equal(peerIDs, s.PeerIDs) {
		return fmt.Errorf("source catalog peers differ")
	}
	for _, p := range s.Peers {
		for _, r := range rooms {
			if slices.Contains(r.MemberBotIDs, s.Bot.ID) && slices.Contains(r.MemberBotIDs, p.ID) {
				s.SharedRoomPeerIDs = append(s.SharedRoomPeerIDs, p.ID)
				break
			}
		}
	}
	return nil
}
func (c *Client) DshbotSenderPolicy(ctx context.Context, id string) (DshbotSenderPolicyStatus, error) {
	var out DshbotSenderPolicyStatus
	if id == "" {
		return out, fmt.Errorf("select a Bot")
	}
	var response struct {
		View dshbotCapabilityView `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{"botId": id}, &response); err != nil {
		return out, fmt.Errorf("source catalog unavailable")
	}
	var err error
	out.DshbotCapabilityStatus, err = response.View.status(id, DshbotCapabilityCatalog{})
	if err != nil {
		return out, err
	}
	out.CatalogItems = response.View.Value.Items
	if err = decodeSenderCatalog(&out); err != nil {
		return out, err
	}
	return out, nil
}
func ValidateDshbotSenderPolicyCommand(c DshbotSenderPolicyCommand) error {
	if err := ValidateDshbotCapabilityProfile(c.Expected.DshbotCapabilityStatus); err != nil {
		return err
	}
	if c.Expected.WriteError != "" {
		return fmt.Errorf("profile cannot be preserved")
	}
	snapshot := CloneDshbotSenderPolicy(c.Expected)
	if err := decodeSenderCatalog(&snapshot); err != nil {
		return err
	}
	if c.Values.Selected != (len(c.Values.IDs) > 0) {
		return fmt.Errorf("select at least one Bot, or explicitly choose the default rule")
	}
	raw, _ := json.Marshal(c.Values.IDs)
	ids, err := senderPolicyIDs(raw)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == c.Expected.Bot.ID || !slices.Contains(snapshot.PeerIDs, id) {
			return fmt.Errorf("selected source is not an available peer Bot: %s", id)
		}
	}
	before, err := DshbotSenderPolicyValue(c.Expected)
	if err != nil {
		return err
	}
	if SameDshbotSenderPolicyValues(before, c.Values) {
		return fmt.Errorf("sender policy unchanged")
	}
	return nil
}

// Empty means all catalog-peer messages, but only shared-room task delegation.
// Hidden catalog Bots remain identities, exactly as the original consumers do.
func DshbotSenderPolicyAccess(s DshbotSenderPolicyStatus, v DshbotSenderPolicyValues) (messages, tasks []string) {
	if v.Selected {
		return append([]string{}, v.IDs...), append([]string{}, v.IDs...)
	}
	return append([]string{}, s.PeerIDs...), append([]string{}, s.SharedRoomPeerIDs...)
}
func DshbotSenderPolicyExpands(c DshbotSenderPolicyCommand) bool {
	old, err := DshbotSenderPolicyValue(c.Expected)
	if err != nil {
		return true
	}
	if old.Selected && !c.Values.Selected {
		return true
	} // Future catalog Bots also gain ordinary-message access.
	om, ot := DshbotSenderPolicyAccess(c.Expected, old)
	nm, nt := DshbotSenderPolicyAccess(c.Expected, c.Values)
	for _, id := range nm {
		if !slices.Contains(om, id) {
			return true
		}
	}
	for _, id := range nt {
		if !slices.Contains(ot, id) {
			return true
		}
	}
	return false
}
func DshbotSenderPolicyApplied(c DshbotSenderPolicyCommand, s DshbotCapabilityStatus) error {
	if err := ValidateDshbotCapabilityProfile(s); err != nil {
		return err
	}
	after, err := DshbotSenderPolicyValue(DshbotSenderPolicyStatus{DshbotCapabilityStatus: s})
	if err != nil || !SameDshbotSenderPolicyValues(after, c.Values) || s.WriteError != "" || s.Revision <= c.Expected.Revision || s.Bot.ID != c.Expected.Bot.ID || s.Bot.SessionID != c.Expected.Bot.SessionID || len(s.Profile) != len(c.Expected.Profile) {
		return fmt.Errorf("complete reviewed sender policy not confirmed; read back")
	}
	for key, before := range c.Expected.Profile {
		if key != "updatedAt" && key != "allowedSenderIds" && !sameCapabilityJSON(before, s.Profile[key]) {
			return fmt.Errorf("unrelated profile field changed: %s", key)
		}
	}
	return nil
}
func (c *Client) DshbotSaveSenderPolicy(ctx context.Context, cmd DshbotSenderPolicyCommand) (DshbotCapabilityReceipt, error) {
	if err := ValidateDshbotSenderPolicyCommand(cmd); err != nil {
		return DshbotCapabilityReceipt{}, err
	}
	raw, _ := json.Marshal(cmd.Values.IDs)
	return c.saveDshbotProfileField(ctx, cmd.Expected.DshbotCapabilityStatus, "allowedSenderIds", raw)
}

// Called only by the common writer's exact single-field allowlist branch.
func validateDshbotSenderField(expected DshbotCapabilityStatus, raw json.RawMessage) error {
	ids, err := senderPolicyIDs(raw)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == expected.Bot.ID || !slices.Contains(expected.PeerIDs, id) {
			return fmt.Errorf("sender must be an existing peer Bot")
		}
	}
	return nil
}
