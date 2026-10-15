package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"unicode/utf8"
)

type DshbotRouteModel struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoningEffort"`
}
type DshbotRouteModelOption struct {
	Provider, Model, Label, DefaultEffort string
	Efforts                               []string
}
type DshbotRouteWorkspace struct{ ID, Path, Title string }
type DshbotRouteOptions struct {
	ModelsKnown, WorkspacesKnown, SessionKnown, WorkspaceLocked bool
	Default                                                     DshbotRouteModel
	Models                                                      []DshbotRouteModelOption
	Workspaces                                                  []DshbotRouteWorkspace
	ScratchCwd                                                  string
}
type DshbotRouteStatus struct {
	DshbotCapabilityStatus
	Options DshbotRouteOptions
}
type DshbotRouteValues struct {
	Model       DshbotRouteModel
	WorkspaceID string
}
type DshbotRouteCommand struct {
	Expected DshbotRouteStatus
	Values   DshbotRouteValues
}

func CloneDshbotRoute(s DshbotRouteStatus) DshbotRouteStatus {
	s.DshbotCapabilityStatus = CloneDshbotCapabilities(s.DshbotCapabilityStatus)
	s.Options.Models = append([]DshbotRouteModelOption(nil), s.Options.Models...)
	for i := range s.Options.Models {
		s.Options.Models[i].Efforts = append([]string(nil), s.Options.Models[i].Efforts...)
	}
	s.Options.Workspaces = append([]DshbotRouteWorkspace(nil), s.Options.Workspaces...)
	return s
}
func CloneDshbotRouteCommand(c DshbotRouteCommand) DshbotRouteCommand {
	c.Expected = CloneDshbotRoute(c.Expected)
	return c
}
func SameDshbotRoute(a, b DshbotRouteStatus) bool {
	return SameDshbotCapabilities(a.DshbotCapabilityStatus, b.DshbotCapabilityStatus) && reflect.DeepEqual(a.Options, b.Options)
}
func routeText(s string, max int, required bool) bool {
	return utf8.ValidString(s) && trimBotMemory(s) == s && WhaleTextLength(s) <= max && (!required || s != "")
}
func DshbotRouteValue(s DshbotRouteStatus) (DshbotRouteValues, error) {
	var v DshbotRouteValues
	model, err := capabilityObject(s.Profile["model"], "provider", "model", "reasoningEffort")
	if err != nil {
		return v, err
	}
	if v.Model.Provider, err = capabilityText(model, "provider", 200, false); err != nil {
		return v, err
	}
	if v.Model.Model, err = capabilityText(model, "model", 500, false); err != nil {
		return v, err
	}
	if v.Model.ReasoningEffort, err = capabilityText(model, "reasoningEffort", 100, false); err != nil {
		return v, err
	}
	v.WorkspaceID, err = capabilityText(s.Profile, "workspaceId", 2000, false)
	return v, err
}

// Discovery is read-only and must come from the same authenticated Host. Missing
// or malformed values never manufacture a writable default or a local path.
func DecodeDshbotRouteModels(raw json.RawMessage, o *DshbotRouteOptions) error {
	o.ModelsKnown = false
	o.Models = nil
	var wire struct {
		Default *DshbotRouteModel `json:"default"`
		Groups  *[]struct {
			ID     string `json:"id"`
			Models []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Reasoning *struct {
					Efforts []struct {
						ID string `json:"id"`
					} `json:"efforts"`
					DefaultEffort string `json:"defaultEffort"`
				} `json:"reasoning"`
			} `json:"models"`
		} `json:"groups"`
	}
	if !validCapabilityUnicode(raw) || json.Unmarshal(raw, &wire) != nil || wire.Groups == nil || wire.Default == nil {
		return fmt.Errorf("model catalog unavailable; keep existing values or use Bot Web")
	}
	seen := map[[2]string]bool{}
	for _, g := range *wire.Groups {
		if !routeText(g.ID, 200, true) {
			return fmt.Errorf("invalid provider catalog")
		}
		for _, m := range g.Models {
			key := [2]string{g.ID, m.ID}
			if !routeText(m.ID, 500, true) || seen[key] {
				return fmt.Errorf("ambiguous model catalog")
			}
			seen[key] = true
			v := DshbotRouteModelOption{Provider: g.ID, Model: m.ID, Label: m.Name}
			if m.Reasoning != nil {
				v.DefaultEffort = m.Reasoning.DefaultEffort
				for _, e := range m.Reasoning.Efforts {
					if !routeText(e.ID, 100, true) || slices.Contains(v.Efforts, e.ID) {
						return fmt.Errorf("invalid effort catalog")
					}
					v.Efforts = append(v.Efforts, e.ID)
				}
				if v.DefaultEffort != "" && !slices.Contains(v.Efforts, v.DefaultEffort) {
					return fmt.Errorf("default effort unavailable")
				}
			}
			o.Models = append(o.Models, v)
		}
	}
	o.Default = *wire.Default
	o.ModelsKnown = true
	return nil
}
func DecodeDshbotRouteWorkspaces(raw json.RawMessage, o *DshbotRouteOptions) error {
	o.WorkspacesKnown = false
	o.Workspaces = nil
	o.ScratchCwd = ""
	var w struct {
		Type  string `json:"type"`
		Value *struct {
			ScratchCwd *string `json:"scratchCwd"`
			Items      *[]struct {
				ID    string `json:"workspaceId"`
				Path  string `json:"path"`
				Title string `json:"title"`
			} `json:"items"`
		} `json:"value"`
	}
	if !validCapabilityUnicode(raw) || json.Unmarshal(raw, &w) != nil || w.Type != "baseline" || w.Value == nil || w.Value.Items == nil || w.Value.ScratchCwd == nil {
		return fmt.Errorf("complete Host workspace baseline unavailable")
	}
	seen := map[string]bool{}
	for _, row := range *w.Value.Items {
		if !routeText(row.ID, 2000, true) || !routeText(row.Path, 2000, true) || seen[row.ID] {
			return fmt.Errorf("ambiguous Host workspace")
		}
		seen[row.ID] = true
		o.Workspaces = append(o.Workspaces, DshbotRouteWorkspace{ID: row.ID, Path: row.Path, Title: row.Title})
	}
	if !routeText(*w.Value.ScratchCwd, 2000, false) {
		return fmt.Errorf("invalid Host scratch directory")
	}
	o.ScratchCwd = *w.Value.ScratchCwd
	o.WorkspacesKnown = true
	return nil
}
func DecodeDshbotRouteSession(raw json.RawMessage, id string, o *DshbotRouteOptions) error {
	o.SessionKnown = false
	o.WorkspaceLocked = true
	var w struct {
		Items *[]struct {
			ID    string `json:"sessionId"`
			Blank *bool  `json:"blank"`
		} `json:"items"`
	}
	if json.Unmarshal(raw, &w) != nil || w.Items == nil {
		return fmt.Errorf("session list unavailable")
	}
	matches := 0
	for _, row := range *w.Items {
		if row.ID == id {
			matches++
			if row.Blank != nil {
				o.SessionKnown = true
				o.WorkspaceLocked = !*row.Blank
			}
		}
	}
	if matches != 1 || !o.SessionKnown {
		o.SessionKnown = false
		o.WorkspaceLocked = true
		return fmt.Errorf("canonical session history is unknown; workspace remains locked")
	}
	return nil
}
func validateRouteModel(m DshbotRouteModel, o DshbotRouteOptions) error {
	if !routeText(m.Provider, 200, false) || !routeText(m.Model, 500, false) || !routeText(m.ReasoningEffort, 100, false) || (m.Provider == "") != (m.Model == "") || m.Provider == "" && m.ReasoningEffort != "" {
		return fmt.Errorf("provider/model must be selected together with a valid effort")
	}
	if !o.ModelsKnown {
		return fmt.Errorf("refresh real Host model choices before changing model")
	}
	selected := m
	if selected.Provider == "" {
		selected = o.Default
		if selected.Provider == "" || selected.Model == "" {
			return fmt.Errorf("Host default model unavailable")
		}
	}
	for _, entry := range o.Models {
		if entry.Provider == selected.Provider && entry.Model == selected.Model {
			if selected.ReasoningEffort == "" || slices.Contains(entry.Efforts, selected.ReasoningEffort) {
				return nil
			}
			return fmt.Errorf("effort is not available for this model")
		}
	}
	return fmt.Errorf("model route is not in the current Host catalog")
}
func ValidateDshbotRouteCommand(c DshbotRouteCommand) error {
	s := c.Expected
	if err := ValidateDshbotCapabilityProfile(s.DshbotCapabilityStatus); err != nil {
		return err
	}
	if s.WriteError != "" {
		return fmt.Errorf("profile cannot be preserved")
	}
	before, err := DshbotRouteValue(s)
	if err != nil {
		return err
	}
	if before == c.Values {
		return fmt.Errorf("model and workspace are unchanged")
	}
	if before.Model != c.Values.Model {
		if err := validateRouteModel(c.Values.Model, s.Options); err != nil {
			return err
		}
	}
	if before.WorkspaceID != c.Values.WorkspaceID {
		if !s.Options.SessionKnown || s.Options.WorkspaceLocked {
			return fmt.Errorf("conversation history is present or unknown; workspace change is locked")
		}
		if !s.Options.WorkspacesKnown || !routeText(c.Values.WorkspaceID, 2000, false) {
			return fmt.Errorf("refresh real Host workspace choices")
		}
		found := false
		if c.Values.WorkspaceID == "" {
			found = routeText(s.Options.ScratchCwd, 2000, true)
		} else {
			for _, w := range s.Options.Workspaces {
				if w.ID == c.Values.WorkspaceID {
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("workspace is no longer available on this Host")
		}
	}
	return nil
}

var dshbotRouteSessionID = regexp.MustCompile(`^session-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func DshbotRouteApplied(c DshbotRouteCommand, s DshbotCapabilityStatus) error {
	before, err := DshbotRouteValue(c.Expected)
	if err != nil {
		return err
	}
	value, err := DshbotRouteValue(DshbotRouteStatus{DshbotCapabilityStatus: s})
	if err != nil || value != c.Values || s.WriteError != "" || s.Bot.ID != c.Expected.Bot.ID || s.Revision <= c.Expected.Revision {
		return fmt.Errorf("reviewed route values are not confirmed; read back")
	}
	if err := ValidateDshbotCapabilityProfile(s); err != nil {
		return err
	}
	moved := before.WorkspaceID != c.Values.WorkspaceID
	if moved {
		if s.Bot.SessionID == c.Expected.Bot.SessionID || !dshbotRouteSessionID.MatchString(s.Bot.SessionID) {
			return fmt.Errorf("new canonical session is not confirmed")
		}
	} else if s.Bot.SessionID != c.Expected.Bot.SessionID {
		return fmt.Errorf("unexpected canonical session change")
	}
	if len(s.Profile) != len(c.Expected.Profile) {
		return fmt.Errorf("profile shape changed; inspect in Bot Web")
	}
	for k, v := range c.Expected.Profile {
		if k == "updatedAt" || k == "model" || k == "workspaceId" || moved && k == "sessionId" {
			continue
		}
		if !sameCapabilityJSON(v, s.Profile[k]) {
			return fmt.Errorf("unrelated profile field changed: %s", k)
		}
	}
	return nil
}
func (c *Client) DshbotSaveRoute(ctx context.Context, cmd DshbotRouteCommand) (DshbotCapabilityReceipt, error) {
	if err := ValidateDshbotRouteCommand(cmd); err != nil {
		return DshbotCapabilityReceipt{}, err
	}
	model, _ := json.Marshal(cmd.Values.Model)
	workspace, _ := json.Marshal(cmd.Values.WorkspaceID)
	return c.saveDshbotProfileFields(ctx, cmd.Expected.DshbotCapabilityStatus, map[string]json.RawMessage{"model": model, "workspaceId": workspace}, cmd)
}
