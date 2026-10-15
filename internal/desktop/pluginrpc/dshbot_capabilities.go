package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
)

var DshbotCapabilityGroups = []string{"tools", "skills", "mcp"}

type DshbotCapabilityPolicy struct {
	Mode  string   `json:"mode"`
	Names []string `json:"names"`
}
type DshbotCapabilityPolicies map[string]DshbotCapabilityPolicy
type DshbotCapabilityEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type DshbotCapabilityCatalog struct {
	Tools           []DshbotCapabilityEntry `json:"tools"`
	Skills          []DshbotCapabilityEntry `json:"skills"`
	MCP             []DshbotCapabilityEntry `json:"mcp"`
	Active          bool                    `json:"active"`
	SkillsAvailable bool                    `json:"skillsAvailable"`
}

func (c DshbotCapabilityCatalog) Entries(group string) []DshbotCapabilityEntry {
	switch group {
	case "tools":
		return c.Tools
	case "skills":
		return c.Skills
	case "mcp":
		return c.MCP
	}
	return nil
}

type DshbotCapabilityStatus struct {
	Bot        DshbotCatalogItem
	Revision   int
	Profile    map[string]json.RawMessage
	Policies   DshbotCapabilityPolicies
	Catalog    DshbotCapabilityCatalog
	PeerIDs    []string
	WriteError string
}
type dshbotCapabilityView struct {
	NS       string `json:"ns"`
	Revision int    `json:"revision"`
	Value    struct {
		Items []json.RawMessage `json:"items"`
	} `json:"value"`
}

func (v dshbotCapabilityView) status(id string, catalog DshbotCapabilityCatalog) (DshbotCapabilityStatus, error) {
	out := DshbotCapabilityStatus{Revision: v.Revision, Catalog: catalog}
	matches := 0
	for _, raw := range v.Value.Items {
		var b DshbotCatalogItem
		if json.Unmarshal(raw, &b) != nil {
			return out, fmt.Errorf("Bot catalog unavailable")
		}
		if b.Kind != "room" && b.ID != id {
			out.PeerIDs = append(out.PeerIDs, b.ID)
		}
		if b.ID == id {
			out.Bot = b
			matches++
			if !validCapabilityUnicode(raw) {
				out.WriteError = "资料包含不能无损解码的 Unicode，保留 Web 入口 / Unsupported Unicode; use existing Web editor"
			}
			if json.Unmarshal(raw, &out.Profile) != nil {
				return out, fmt.Errorf("Bot profile unavailable")
			}
		}
	}
	if v.NS != "dsh-bot" || v.Revision < 0 || matches != 1 || out.Bot.Kind == "room" || out.Bot.Hidden || out.Bot.SessionID == "" {
		return out, fmt.Errorf("selected Bot unavailable")
	}
	policies, err := decodeCapabilityPolicies(out.Profile["capabilities"])
	out.Policies = policies
	if err == nil {
		err = ValidateDshbotCapabilityProfile(out)
	}
	if err != nil {
		out.WriteError = "无法证明资料无损，保留 Web 编辑入口 / Profile cannot be safely round-tripped: " + err.Error()
	}
	return out, nil
}
func CloneDshbotCapabilityPolicies(p DshbotCapabilityPolicies) DshbotCapabilityPolicies {
	if p == nil {
		return nil
	}
	out := DshbotCapabilityPolicies{}
	for group, v := range p {
		v.Names = append([]string{}, v.Names...)
		out[group] = v
	}
	return out
}
func CloneDshbotCapabilities(s DshbotCapabilityStatus) DshbotCapabilityStatus {
	out := s
	out.Bot.MemberBotIDs = append([]string(nil), s.Bot.MemberBotIDs...)
	out.Profile = map[string]json.RawMessage{}
	for k, v := range s.Profile {
		out.Profile[k] = append(json.RawMessage(nil), v...)
	}
	out.Policies = CloneDshbotCapabilityPolicies(s.Policies)
	out.PeerIDs = append([]string(nil), s.PeerIDs...)
	out.Catalog.Tools = append([]DshbotCapabilityEntry{}, s.Catalog.Tools...)
	out.Catalog.Skills = append([]DshbotCapabilityEntry{}, s.Catalog.Skills...)
	out.Catalog.MCP = append([]DshbotCapabilityEntry{}, s.Catalog.MCP...)
	return out
}
func SameDshbotCapabilities(a, b DshbotCapabilityStatus) bool {
	if a.Revision != b.Revision || a.Bot.ID != b.Bot.ID || a.Bot.SessionID != b.Bot.SessionID || len(a.Profile) != len(b.Profile) || !reflect.DeepEqual(a.Catalog, b.Catalog) {
		return false
	}
	for k, v := range a.Profile {
		if !sameCapabilityJSON(v, b.Profile[k]) {
			return false
		}
	}
	return true
}
func (c *Client) DshbotCapabilities(ctx context.Context, id string) (DshbotCapabilityStatus, error) {
	var out DshbotCapabilityStatus
	if id == "" {
		return out, fmt.Errorf("select a Bot")
	}
	var response struct {
		View         dshbotCapabilityView `json:"view"`
		Capabilities json.RawMessage      `json:"capabilities"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{"botId": id}, &response); err != nil {
		return out, fmt.Errorf("Bot capability discovery unavailable")
	}
	catalog := DshbotCapabilityCatalog{Active: true, SkillsAvailable: true}
	if !validCapabilityUnicode(response.Capabilities) || json.Unmarshal(response.Capabilities, &catalog) != nil || catalog.Tools == nil || catalog.Skills == nil || catalog.MCP == nil {
		return out, fmt.Errorf("capability discovery incomplete")
	}
	for _, group := range DshbotCapabilityGroups {
		seen := map[string]bool{}
		for _, e := range catalog.Entries(group) {
			if e.Name == "" || seen[e.Name] || trimBotMemory(e.Name) != e.Name || WhaleTextLength(e.Name) > 300 {
				return out, fmt.Errorf("capability discovery malformed")
			}
			seen[e.Name] = true
		}
	}
	return response.View.status(id, catalog)
}

type DshbotCapabilityCommand struct {
	Expected DshbotCapabilityStatus
	Desired  DshbotCapabilityPolicies
}
type DshbotCapabilityReceipt struct {
	Status    DshbotCapabilityStatus
	Warnings  []string
	Attempted bool
}

func ValidateDshbotCapabilityCommand(cmd DshbotCapabilityCommand) error {
	if err := ValidateDshbotCapabilityProfile(cmd.Expected); err != nil {
		return err
	}
	if cmd.Expected.WriteError != "" || !cmd.Expected.Catalog.Active {
		return fmt.Errorf("capability editing unavailable")
	}
	raw, _ := json.Marshal(cmd.Desired)
	policies, err := decodeCapabilityPolicies(raw)
	if err != nil {
		return err
	}
	for _, group := range DshbotCapabilityGroups {
		p := policies[group]
		if p.Mode != "selected" {
			continue
		}
		if group == "skills" && !cmd.Expected.Catalog.SkillsAvailable && len(p.Names) > 0 {
			return fmt.Errorf("Skills are unavailable")
		}
		available := []string{}
		for _, e := range cmd.Expected.Catalog.Entries(group) {
			available = append(available, e.Name)
		}
		for _, name := range p.Names {
			if !slices.Contains(available, name) {
				return fmt.Errorf("selected %s is no longer available: %s", group, name)
			}
		}
	}
	if reflect.DeepEqual(cmd.Expected.Policies, cmd.Desired) {
		return fmt.Errorf("capabilities are unchanged")
	}
	return nil
}
func (c *Client) DshbotSaveCapabilities(ctx context.Context, cmd DshbotCapabilityCommand) (DshbotCapabilityReceipt, error) {
	var out DshbotCapabilityReceipt
	if err := ValidateDshbotCapabilityCommand(cmd); err != nil {
		return out, err
	}
	payload := CloneDshbotCapabilities(cmd.Expected).Profile
	payload["capabilities"], _ = json.Marshal(cmd.Desired)
	payload["revision"], _ = json.Marshal(cmd.Expected.Revision)
	var response struct {
		View     dshbotCapabilityView `json:"view"`
		Warnings []string             `json:"warnings"`
	}
	out.Attempted = true
	if err := c.Call(ctx, "dshbot", "bot/update", payload, &response); err != nil {
		return out, fmt.Errorf("capability save outcome uncertain; read back before another save")
	}
	out.Warnings = append([]string(nil), response.Warnings...)
	var err error
	out.Status, err = response.View.status(cmd.Expected.Bot.ID, cmd.Expected.Catalog)
	if err != nil {
		return out, err
	}
	if out.Status.Bot.SessionID != cmd.Expected.Bot.SessionID || out.Status.Revision <= cmd.Expected.Revision || !reflect.DeepEqual(out.Status.Policies, cmd.Desired) {
		return out, fmt.Errorf("capability receipt differs; read back")
	}
	for k, v := range cmd.Expected.Profile {
		if k != "capabilities" && k != "updatedAt" && !sameCapabilityJSON(v, out.Status.Profile[k]) {
			return out, fmt.Errorf("profile field changed in receipt; read back: %s", k)
		}
	}
	return out, nil
}
