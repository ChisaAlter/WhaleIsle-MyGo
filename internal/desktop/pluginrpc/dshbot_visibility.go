package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// catalog/items is the original visibility command, not bot/update. Its CAS
// replaces the entire items array; all other catalog fields remain Host-owned.
type DshbotVisibilityCommand struct {
	Expected      DshbotBotCatalog
	ID, SessionID string
	Hidden        bool
	UpdatedAt     int64
}
type DshbotVisibilityResult struct {
	Attempted, Rejected bool
	Catalog             DshbotBotCatalog
	Item                DshbotBotItem
}

func CloneDshbotVisibilityCommand(c DshbotVisibilityCommand) DshbotVisibilityCommand {
	c.Expected = CloneDshbotBotCatalog(c.Expected)
	return c
}
func DshbotVisibilityTarget(c DshbotVisibilityCommand) (DshbotBotItem, error) {
	for _, v := range c.Expected.Items {
		if v.ID == c.ID && v.SessionID == c.SessionID {
			return CloneDshbotBotItem(v), nil
		}
	}
	return DshbotBotItem{}, fmt.Errorf("exact catalog target is missing; refresh without choosing another item")
}
func ValidateDshbotVisibilityCommand(c DshbotVisibilityCommand) error {
	if c.Expected.Revision < 0 || c.ID == "" || c.SessionID == "" || c.UpdatedAt <= 0 || c.UpdatedAt > 9007199254740991 {
		return fmt.Errorf("invalid catalog identity, revision or reviewed timestamp")
	}
	ids, sessions := map[string]bool{}, map[string]bool{}
	for _, row := range c.Expected.Items {
		raw, err := json.Marshal(row.Profile)
		if err != nil {
			return err
		}
		decoded, err := decodeDshbotBotItem(raw)
		if err != nil || decoded.ID != row.ID || decoded.SessionID != row.SessionID || decoded.Kind != row.Kind || decoded.Hidden != row.Hidden || ids[row.ID] || sessions[row.SessionID] {
			return fmt.Errorf("catalog contains ambiguous or unsupported identities")
		}
		ids[row.ID], sessions[row.SessionID] = true, true
		hidden := strings.TrimSpace(string(row.Profile["hidden"]))
		if hidden != "true" && hidden != "false" {
			return fmt.Errorf("catalog hidden value is not an explicit boolean")
		}
		var stamp json.Number
		if json.Unmarshal(row.Profile["updatedAt"], &stamp) != nil {
			return fmt.Errorf("catalog timestamp is missing")
		}
		if _, err := stamp.Float64(); err != nil {
			return fmt.Errorf("catalog timestamp is invalid")
		}
	}
	target, err := DshbotVisibilityTarget(c)
	if err != nil {
		return err
	}
	if target.Hidden == c.Hidden {
		return fmt.Errorf("visibility is unchanged")
	}
	return nil
}
func dshbotVisibilityItems(c DshbotVisibilityCommand) []map[string]json.RawMessage {
	out := make([]map[string]json.RawMessage, len(c.Expected.Items))
	for i, row := range c.Expected.Items {
		out[i] = CloneDshbotBotItem(row).Profile
		if row.ID == c.ID {
			out[i]["hidden"], _ = json.Marshal(c.Hidden)
			out[i]["updatedAt"], _ = json.Marshal(c.UpdatedAt)
		}
	}
	return out
}

// A matching boolean alone cannot prove a full-array save. Require a newer
// revision and every item, identity, order and raw nested field to match.
func DshbotVisibilityApplied(c DshbotVisibilityCommand, v DshbotBotCatalog) (DshbotBotItem, error) {
	if err := ValidateDshbotVisibilityCommand(c); err != nil {
		return DshbotBotItem{}, err
	}
	if v.Revision <= c.Expected.Revision || len(v.Items) != len(c.Expected.Items) {
		return DshbotBotItem{}, fmt.Errorf("no exact newer catalog receipt; read back without retry")
	}
	wanted := dshbotVisibilityItems(c)
	var target DshbotBotItem
	for i, row := range v.Items {
		raw, _ := json.Marshal(wanted[i])
		expected, err := decodeDshbotBotItem(raw)
		if err != nil || !sameDshbotBotItem(expected, row) {
			return DshbotBotItem{}, fmt.Errorf("catalog item/order/profile differs; outcome remains unknown")
		}
		if row.ID == c.ID {
			target = CloneDshbotBotItem(row)
		}
	}
	return target, nil
}
func (c *Client) DshbotSetVisibility(ctx context.Context, cmd DshbotVisibilityCommand) (DshbotVisibilityResult, error) {
	var out DshbotVisibilityResult
	if err := ValidateDshbotVisibilityCommand(cmd); err != nil {
		out.Rejected = true
		return out, err
	}
	var reply struct {
		View json.RawMessage `json:"view"`
	}
	out.Attempted = true
	err := c.Call(ctx, "dshbot", "catalog/items", map[string]any{"revision": cmd.Expected.Revision, "items": dshbotVisibilityItems(cmd)}, &reply)
	if err != nil {
		// Both exact errors occur in checkRevision before scope.set. Other failures
		// may follow persistence; they never authorize an automatic resend.
		message := err.Error()
		if strings.HasPrefix(message, "pluginrpc: ") && !strings.HasPrefix(message, "pluginrpc: HTTP ") {
			_, host, envelope := strings.Cut(strings.TrimPrefix(message, "pluginrpc: "), ": ")
			out.Rejected = envelope && (host == "Bot catalog is read-only." || host == "Bot catalog changed. Refresh and try again.")
		}
		return out, err
	}
	out.Catalog, err = decodeDshbotBotCatalog(reply.View)
	if err != nil {
		return out, err
	}
	out.Item, err = DshbotVisibilityApplied(cmd, out.Catalog)
	return out, err
}
