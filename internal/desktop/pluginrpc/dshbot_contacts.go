package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Contacts use the same authoritative catalog and revision as profile editing.
// Section mutations are original section/* commands; only pinning uses items CAS.
type DshbotContactSection struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}
type DshbotContactCatalog struct {
	DshbotBotCatalog
	Sections []DshbotContactSection
}
type DshbotSectionSnapshot struct {
	Section   DshbotContactSection `json:"section"`
	Index     int                  `json:"index"`
	MemberIDs []string             `json:"memberIds"`
}
type DshbotContactCommand struct {
	Expected                               DshbotContactCatalog
	Action, ID, SessionID, Name, SectionID string
	Delta                                  int
	Pinned                                 bool
	UpdatedAt                              int64
	Snapshot                               *DshbotSectionSnapshot
}
type DshbotContactResult struct {
	Attempted, Rejected bool
	Catalog             DshbotContactCatalog
	Snapshot            *DshbotSectionSnapshot
}

func NewDshbotContactSectionID() string { return "section-" + newRPCID() }
func CloneDshbotContactCatalog(v DshbotContactCatalog) DshbotContactCatalog {
	v.DshbotBotCatalog = CloneDshbotBotCatalog(v.DshbotBotCatalog)
	v.Sections = append([]DshbotContactSection(nil), v.Sections...)
	return v
}
func CloneDshbotSectionSnapshot(v *DshbotSectionSnapshot) *DshbotSectionSnapshot {
	if v == nil {
		return nil
	}
	out := *v
	out.MemberIDs = append([]string(nil), v.MemberIDs...)
	return &out
}
func CloneDshbotContactCommand(v DshbotContactCommand) DshbotContactCommand {
	v.Expected = CloneDshbotContactCatalog(v.Expected)
	v.Snapshot = CloneDshbotSectionSnapshot(v.Snapshot)
	return v
}
func DshbotContactText(v DshbotBotItem, field string) string {
	var out string
	_ = json.Unmarshal(v.Profile[field], &out)
	return strings.TrimSpace(out)
}
func DshbotContactNumber(v DshbotBotItem, field string) float64 {
	var out float64
	_ = json.Unmarshal(v.Profile[field], &out)
	return out
}
func DshbotContactSource(v DshbotBotItem) (string, string) {
	id, label := DshbotContactText(v, "sourceId"), DshbotContactText(v, "sourceLabel")
	var source struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	_ = json.Unmarshal(v.Profile["source"], &source)
	// Match the original nullish fallback, including an explicitly empty string.
	if _, ok := v.Profile["sourceId"]; !ok || string(v.Profile["sourceId"]) == "null" {
		id = strings.TrimSpace(source.ID)
	}
	if _, ok := v.Profile["sourceLabel"]; !ok || string(v.Profile["sourceLabel"]) == "null" {
		label = strings.TrimSpace(source.Label)
	}
	if label == "" {
		label = id
	}
	return id, label
}
func decodeDshbotContactCatalog(raw json.RawMessage) (DshbotContactCatalog, error) {
	bots, err := decodeDshbotBotCatalog(raw)
	if err != nil {
		return DshbotContactCatalog{}, err
	}
	var wire struct {
		Value struct {
			Sections []DshbotContactSection `json:"sections"`
		} `json:"value"`
	}
	if err = json.Unmarshal(raw, &wire); err != nil {
		return DshbotContactCatalog{}, err
	}
	return DshbotContactCatalog{DshbotBotCatalog: bots, Sections: wire.Value.Sections}, nil
}
func (c *Client) DshbotContacts(ctx context.Context) (DshbotContactCatalog, error) {
	var out struct {
		View json.RawMessage `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{}, &out); err != nil {
		return DshbotContactCatalog{}, err
	}
	return decodeDshbotContactCatalog(out.View)
}
func DshbotContactSectionSnapshot(v DshbotContactCatalog, id string) *DshbotSectionSnapshot {
	for i, section := range v.Sections {
		if section.ID == id {
			out := &DshbotSectionSnapshot{Section: section, Index: i, MemberIDs: []string{}}
			for _, row := range v.Items {
				if DshbotContactText(row, "sectionId") == id {
					out.MemberIDs = append(out.MemberIDs, row.ID)
				}
			}
			return out
		}
	}
	return nil
}
func contactTarget(cmd DshbotContactCommand) (DshbotBotItem, error) {
	for _, row := range cmd.Expected.Items {
		if row.ID == cmd.ID && row.SessionID == cmd.SessionID {
			return row, nil
		}
	}
	return DshbotBotItem{}, fmt.Errorf("联系人已变化，请刷新目录 / Refresh the contact identity")
}
func contactPinItems(cmd DshbotContactCommand) []map[string]json.RawMessage {
	maxOrder := float64(0)
	for _, row := range cmd.Expected.Items {
		if row.Pinned {
			if order := DshbotContactNumber(row, "pinOrder"); order > maxOrder {
				maxOrder = order
			}
		}
	}
	order := float64(0)
	if cmd.Pinned {
		order = maxOrder + 1
	}
	items := make([]map[string]json.RawMessage, len(cmd.Expected.Items))
	for i, row := range cmd.Expected.Items {
		items[i] = CloneDshbotBotItem(row).Profile
		if row.ID == cmd.ID {
			items[i]["pinned"], _ = json.Marshal(cmd.Pinned)
			items[i]["pinOrder"], _ = json.Marshal(order)
			items[i]["updatedAt"], _ = json.Marshal(cmd.UpdatedAt)
		}
	}
	return items
}
func contactPayload(cmd DshbotContactCommand) (string, map[string]any, error) {
	payload := map[string]any{"revision": cmd.Expected.Revision}
	if cmd.Expected.Revision < 0 {
		return "", nil, fmt.Errorf("请先刷新目录 / Refresh the catalog first")
	}
	switch cmd.Action {
	case "pin":
		if _, err := contactTarget(cmd); err != nil {
			return "", nil, err
		}
		payload["items"] = contactPinItems(cmd)
		return "catalog/items", payload, nil
	case "create", "rename":
		name := strings.TrimSpace(cmd.Name)
		if name == "" || WhaleTextLength(name) > 120 {
			return "", nil, fmt.Errorf("分组名称需为 1–120 字符 / Section name: 1–120 characters")
		}
		if cmd.ID == "" {
			return "", nil, fmt.Errorf("分组 ID 不可为空")
		}
		for _, section := range cmd.Expected.Sections {
			if section.ID != cmd.ID && strings.EqualFold(strings.TrimSpace(section.Name), name) {
				return "", nil, fmt.Errorf("分组名称已存在 / Section name already exists")
			}
		}
		payload["id"], payload["name"] = cmd.ID, name
	case "move":
		if cmd.Delta != -1 && cmd.Delta != 1 {
			return "", nil, fmt.Errorf("移动方向无效")
		}
		payload["id"], payload["delta"] = cmd.ID, cmd.Delta
	case "assign":
		if _, err := contactTarget(cmd); err != nil {
			return "", nil, err
		}
		payload["itemId"], payload["sectionId"] = cmd.ID, cmd.SectionID
	case "delete":
		payload["id"] = cmd.ID
	case "restore":
		if cmd.Snapshot == nil {
			return "", nil, fmt.Errorf("没有可恢复的分组快照 / No section snapshot")
		}
		payload["snapshot"] = cmd.Snapshot
	default:
		return "", nil, fmt.Errorf("不支持的联系人操作 / Unsupported contacts action")
	}
	return "section/" + cmd.Action, payload, nil
}

// Used only after an explicit readback when the original response was lost.
func DshbotContactApplied(cmd DshbotContactCommand, v DshbotContactCatalog) bool {
	if v.Revision <= cmd.Expected.Revision {
		return false
	}
	section := func(id string) *DshbotContactSection {
		for _, s := range v.Sections {
			if s.ID == id {
				out := s
				return &out
			}
		}
		return nil
	}
	switch cmd.Action {
	case "create", "rename":
		s := section(cmd.ID)
		return s != nil && s.Name == strings.TrimSpace(cmd.Name)
	case "move":
		before := DshbotContactSectionSnapshot(cmd.Expected, cmd.ID)
		if before == nil {
			return false
		}
		i := before.Index + cmd.Delta
		return i >= 0 && i < len(v.Sections) && v.Sections[i].ID == cmd.ID
	case "assign":
		for _, row := range v.Items {
			if row.ID == cmd.ID && row.SessionID == cmd.SessionID {
				return DshbotContactText(row, "sectionId") == cmd.SectionID
			}
		}
	case "delete":
		if section(cmd.ID) != nil {
			return false
		}
		for _, row := range v.Items {
			if DshbotContactText(row, "sectionId") == cmd.ID {
				return false
			}
		}
		return true
	case "restore":
		if cmd.Snapshot == nil {
			return false
		}
		restored := section(cmd.Snapshot.Section.ID)
		if restored == nil || *restored != cmd.Snapshot.Section {
			return false
		}
		for _, id := range cmd.Snapshot.MemberIDs {
			for _, row := range v.Items {
				if row.ID == id && DshbotContactText(row, "sectionId") != restored.ID {
					return false
				}
			}
		}
		return true
	case "pin":
		expected := contactPinItems(cmd)
		if len(expected) != len(v.Items) {
			return false
		}
		for i, raw := range expected {
			b, _ := json.Marshal(raw)
			row, err := decodeDshbotBotItem(b)
			if err != nil || !sameDshbotBotItem(row, v.Items[i]) {
				return false
			}
		}
		return true
	}
	return false
}
func (c *Client) DshbotChangeContacts(ctx context.Context, cmd DshbotContactCommand) (DshbotContactResult, error) {
	var out DshbotContactResult
	endpoint, payload, err := contactPayload(cmd)
	if err != nil {
		out.Rejected = true
		return out, err
	}
	var reply struct {
		View     json.RawMessage        `json:"view"`
		Snapshot *DshbotSectionSnapshot `json:"snapshot"`
	}
	out.Attempted = true
	if err = c.Call(ctx, "dshbot", endpoint, payload, &reply); err != nil {
		message := err.Error()
		if strings.HasPrefix(message, "pluginrpc: ") && !strings.HasPrefix(message, "pluginrpc: HTTP ") {
			_, host, ok := strings.Cut(strings.TrimPrefix(message, "pluginrpc: "), ": ")
			out.Rejected = ok && (host == "Bot catalog is read-only." || host == "Bot catalog changed. Refresh and try again.")
		}
		return out, err
	}
	out.Catalog, err = decodeDshbotContactCatalog(reply.View)
	out.Snapshot = CloneDshbotSectionSnapshot(reply.Snapshot)
	return out, err
}
