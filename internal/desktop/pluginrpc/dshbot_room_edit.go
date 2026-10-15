package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
)

// The room update endpoint accepts a whole profile, not a patch. Retain raw
// unknown values and refuse shapes its pinned normalizer cannot preserve.
type DshbotRoomEditStatus struct {
	Catalog    DshbotRoomCatalog
	Item       DshbotRoomItem
	Profile    map[string]json.RawMessage
	Sections   []string
	WriteError string
}
type DshbotRoomEditValues struct {
	MemberBotIDs         []string
	MaxRounds, MaxSpeaks int
}
type DshbotRoomEditCommand struct {
	Expected DshbotRoomEditStatus
	Values   DshbotRoomEditValues
}
type DshbotRoomEditResult struct {
	Attempted bool
	Status    DshbotRoomEditStatus
	Warnings  []string
}

func CloneDshbotRoomEdit(v DshbotRoomEditStatus) DshbotRoomEditStatus {
	v.Catalog = CloneDshbotRoomCatalog(v.Catalog)
	v.Item.MemberBotIDs = append([]string(nil), v.Item.MemberBotIDs...)
	v.Sections = append([]string(nil), v.Sections...)
	p := map[string]json.RawMessage{}
	for k, raw := range v.Profile {
		p[k] = append(json.RawMessage(nil), raw...)
	}
	v.Profile = p
	return v
}
func CloneDshbotRoomEditCommand(v DshbotRoomEditCommand) DshbotRoomEditCommand {
	v.Expected = CloneDshbotRoomEdit(v.Expected)
	v.Values.MemberBotIDs = append([]string(nil), v.Values.MemberBotIDs...)
	return v
}
func SameDshbotRoomEdit(a, b DshbotRoomEditStatus) bool {
	return reflect.DeepEqual(CloneDshbotRoomEdit(a), CloneDshbotRoomEdit(b))
}
func DshbotRoomEditProfile(cmd DshbotRoomEditCommand) map[string]json.RawMessage {
	p := CloneDshbotRoomEdit(cmd.Expected).Profile
	p["memberBotIds"], _ = json.Marshal(cmd.Values.MemberBotIDs)
	p["maxRounds"], _ = json.Marshal(cmd.Values.MaxRounds)
	p["maxSpeaks"], _ = json.Marshal(cmd.Values.MaxSpeaks)
	return p
}
func decodeDshbotRoomEdit(raw json.RawMessage, id string) (DshbotRoomEditStatus, error) {
	var out DshbotRoomEditStatus
	catalog, err := decodeDshbotRoomCatalog(raw)
	if err != nil {
		return out, err
	}
	var wire struct {
		Value struct {
			Items    []json.RawMessage `json:"items"`
			Sections []struct {
				ID string `json:"id"`
			} `json:"sections"`
		} `json:"value"`
	}
	if json.Unmarshal(raw, &wire) != nil {
		return out, fmt.Errorf("complete profile unavailable")
	}
	out.Catalog = catalog
	for _, section := range wire.Value.Sections {
		out.Sections = append(out.Sections, section.ID)
	}
	for i, row := range catalog.Items {
		if row.ID == id {
			if row.Kind != "room" {
				return out, fmt.Errorf("select an existing Bot group")
			}
			out.Item = row
			if json.Unmarshal(wire.Value.Items[i], &out.Profile) != nil || out.Profile == nil {
				return out, fmt.Errorf("complete room profile unavailable")
			}
			if err = validateDshbotRoomProfile(out); err != nil {
				out.WriteError = err.Error()
			}
			return out, nil
		}
	}
	return out, fmt.Errorf("selected room is unavailable")
}
func (c *Client) DshbotRoomEdit(ctx context.Context, id string) (DshbotRoomEditStatus, error) {
	var r struct {
		View json.RawMessage `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{"botId": id}, &r); err != nil {
		return DshbotRoomEditStatus{}, err
	}
	return decodeDshbotRoomEdit(r.View, id)
}
func roomRawNumber(p map[string]json.RawMessage, k string, min, max float64, integer bool) error {
	var n float64
	if p[k] == nil || string(p[k]) == "null" || json.Unmarshal(p[k], &n) != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < min || n > max || integer && math.Trunc(n) != n {
		return fmt.Errorf("%s cannot be preserved", k)
	}
	return nil
}
func roomRawStrings(raw json.RawMessage) error {
	var x []string
	if string(raw) == "null" || json.Unmarshal(raw, &x) != nil || x == nil {
		return fmt.Errorf("missing string array")
	}
	return nil
}
func roomRawBool(p map[string]json.RawMessage, k string) error {
	var b bool
	if p[k] == nil || string(p[k]) == "null" || json.Unmarshal(p[k], &b) != nil {
		return fmt.Errorf("%s cannot be preserved", k)
	}
	return nil
}
func roomRawObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var p map[string]json.RawMessage
	if json.Unmarshal(raw, &p) != nil || p == nil {
		return nil, fmt.Errorf("missing object")
	}
	return p, nil
}
func roomHolds(raw json.RawMessage) error {
	p, err := roomRawObject(raw)
	if err != nil {
		return err
	}
	for _, v := range p {
		h, err := roomRawObject(v)
		if err != nil {
			return err
		}
		for _, k := range []string{"byMessageId", "thread"} {
			var s string
			if h[k] == nil || string(h[k]) == "null" || json.Unmarshal(h[k], &s) != nil {
				return fmt.Errorf("incomplete hold")
			}
		}
		if roomRawBool(h, "noted") != nil || roomRawNumber(h, "at", -math.MaxFloat64, math.MaxFloat64, false) != nil {
			return fmt.Errorf("incomplete hold")
		}
	}
	return nil
}
func validateDshbotRoomProfile(s DshbotRoomEditStatus) error {
	p := s.Profile
	raw, _ := json.Marshal(p)
	if !validCapabilityUnicode(raw) || s.Item.ID == "" || s.Item.Kind != "room" || s.Item.SessionID == "" || s.Catalog.Revision < 0 {
		return fmt.Errorf("room profile is incomplete")
	}
	for _, f := range []struct {
		k        string
		max      int
		required bool
	}{{"id", 200, true}, {"kind", 20, true}, {"sessionId", 240, true}, {"name", 120, true}, {"title", 120, false}, {"description", 12000, false}, {"workspaceId", 2000, false}, {"sectionId", 200, false}, {"objective", 2000, false}} {
		if _, err := capabilityText(p, f.k, f.max, f.required); err != nil {
			return err
		}
	}
	var item DshbotRoomItem
	if json.Unmarshal(raw, &item) != nil || !reflect.DeepEqual(item, s.Item) {
		return fmt.Errorf("room identity/profile mismatch")
	}
	for _, k := range []string{"pinned", "hidden", "readInitialized", "memoryReview", "notify"} {
		if err := roomRawBool(p, k); err != nil {
			return err
		}
	}
	for _, k := range []string{"pinOrder", "createdAt", "updatedAt"} {
		if err := roomRawNumber(p, k, -math.MaxFloat64, math.MaxFloat64, false); err != nil {
			return err
		}
	}
	if roomRawNumber(p, "lastSeenSeq", 0, 9007199254740991, true) != nil {
		return fmt.Errorf("read state cannot be preserved")
	}
	for k, want := range map[string]string{"kind": `"room"`, "title": `""`, "objective": `""`, "objectiveMaxRounds": `16`, "memoryReview": `true`, "notify": `true`, "allowedSenderIds": `[]`} {
		if !sameCapabilityJSON(p[k], json.RawMessage(want)) {
			return fmt.Errorf("room %s would be normalized; use Bot Web", k)
		}
	}
	section, _ := capabilityText(p, "sectionId", 200, false)
	if section != "" && !slices.Contains(s.Sections, section) {
		return fmt.Errorf("room section unavailable")
	}
	model, err := capabilityObject(p["model"], "provider", "model", "reasoningEffort")
	if err != nil {
		return fmt.Errorf("unknown model fields would be lost")
	}
	for _, k := range []string{"provider", "model", "reasoningEffort"} {
		if !sameCapabilityJSON(model[k], json.RawMessage(`""`)) {
			return fmt.Errorf("room model would be reset")
		}
	}
	avatar, err := capabilityObject(p["avatar"], "kind", "shape", "color", "dataUrl", "crop")
	if err != nil {
		return fmt.Errorf("unknown avatar fields would be lost")
	}
	av := map[string]string{}
	for _, k := range []string{"kind", "shape", "color", "dataUrl", "crop"} {
		if avatar[k] == nil || string(avatar[k]) == "null" || json.Unmarshal(avatar[k], new(string)) != nil {
			return fmt.Errorf("avatar defaults unavailable")
		}
		var text string
		json.Unmarshal(avatar[k], &text)
		av[k] = text
	}
	if av["kind"] == "blob" {
		if !slices.Contains([]string{"circle", "soft", "square", "pill", "triangle", "hex", "cloud", "drop"}, av["shape"]) || !slices.Contains([]string{"ink", "brown", "red", "orange", "yellow", "green", "teal", "blue", "purple", "pink", "grey"}, av["color"]) || av["dataUrl"] != "" || av["crop"] != "" {
			return fmt.Errorf("avatar would be normalized")
		}
	} else if av["kind"] == "image" {
		url := strings.ToLower(av["dataUrl"])
		if !(strings.HasPrefix(url, "data:image/jpeg;base64,") || strings.HasPrefix(url, "data:image/jpg;base64,") || strings.HasPrefix(url, "data:image/png;base64,")) || WhaleTextLength(av["dataUrl"]) > 120000 || (av["crop"] != "circle" && av["crop"] != "square") || av["shape"] != "" || av["color"] != "" {
			return fmt.Errorf("avatar would be normalized")
		}
	} else {
		return fmt.Errorf("unknown avatar would be replaced")
	}
	// For rooms capabilities are retained from current, and the non-strict
	// schema preserves extensions. Require all known defaults already present.
	caps, err := roomRawObject(p["capabilities"])
	if err != nil {
		return err
	}
	for _, group := range []string{"tools", "skills", "mcp"} {
		g, err := roomRawObject(caps[group])
		if err != nil {
			return err
		}
		var mode string
		if json.Unmarshal(g["mode"], &mode) != nil || (mode != "all" && mode != "selected") || roomRawStrings(g["names"]) != nil {
			return fmt.Errorf("capability defaults unavailable")
		}
	}
	if err := roomHolds(p["holds"]); err != nil {
		return fmt.Errorf("hold state cannot be preserved")
	}
	if cp, ok := p["holdCheckpoint"]; ok {
		h, err := roomRawObject(cp)
		if err != nil || roomRawNumber(h, "version", 1, 9007199254740991, true) != nil || roomRawNumber(h, "processedThroughSeq", -1, 9007199254740991, true) != nil || roomHolds(h["holds"]) != nil || roomRawStrings(h["processedEventIds"]) != nil {
			return fmt.Errorf("checkpoint cannot be preserved")
		}
	}
	var inbox []map[string]json.RawMessage
	if json.Unmarshal(p["inbox"], &inbox) != nil || inbox == nil {
		return fmt.Errorf("inbox unavailable")
	}
	for _, entry := range inbox {
		for _, k := range []string{"routineId", "runId", "kind", "fromId", "fromName", "text", "taskId", "task", "successCriteria", "previousOutput", "notepad", "watchContent"} {
			var s string
			if entry[k] == nil || string(entry[k]) == "null" || json.Unmarshal(entry[k], &s) != nil {
				return fmt.Errorf("inbox defaults unavailable")
			}
		}
		if roomRawStrings(entry["constraints"]) != nil || roomRawBool(entry, "priority") != nil || roomRawBool(entry, "silentAllowed") != nil || roomRawNumber(entry, "timestampMs", -math.MaxFloat64, math.MaxFloat64, false) != nil {
			return fmt.Errorf("inbox cannot be preserved")
		}
		var contexts []map[string]json.RawMessage
		if json.Unmarshal(entry["contextOutputs"], &contexts) != nil || contexts == nil {
			return fmt.Errorf("inbox contexts unavailable")
		}
		for _, x := range contexts {
			for _, k := range []string{"name", "output"} {
				var s string
				if x[k] == nil || string(x[k]) == "null" || json.Unmarshal(x[k], &s) != nil {
					return fmt.Errorf("inbox contexts incomplete")
				}
			}
		}
	}
	if roomRawStrings(p["memberBotIds"]) != nil || roomRawNumber(p, "maxRounds", 1, 3, true) != nil || roomRawNumber(p, "maxSpeaks", 1, 10, true) != nil {
		return fmt.Errorf("current group values unavailable")
	}
	return nil
}
func ValidateDshbotRoomEdit(cmd DshbotRoomEditCommand) error {
	if cmd.Expected.WriteError != "" {
		return fmt.Errorf("profile cannot be preserved: %s", cmd.Expected.WriteError)
	}
	if err := validateDshbotRoomProfile(cmd.Expected); err != nil {
		return err
	}
	v := cmd.Values
	if v.MaxRounds < 1 || v.MaxRounds > 3 || v.MaxSpeaks < 1 || v.MaxSpeaks > 10 || len(v.MemberBotIDs) < 2 || len(v.MemberBotIDs) > 6 {
		return fmt.Errorf("choose 2–6 Bots, 1–3 rounds and 1–10 speaks")
	}
	available := map[string]bool{}
	for _, row := range cmd.Expected.Catalog.Items {
		if row.Kind == "bot" {
			available[row.ID] = true
		}
	}
	seen := map[string]bool{}
	for _, id := range v.MemberBotIDs {
		if !available[id] || seen[id] {
			return fmt.Errorf("members must be distinct current Bot identities")
		}
		seen[id] = true
	}
	return nil
}
func DshbotRoomEditApplied(cmd DshbotRoomEditCommand, fresh DshbotRoomEditStatus) error {
	if fresh.Item.ID != cmd.Expected.Item.ID || fresh.Item.SessionID != cmd.Expected.Item.SessionID || fresh.Catalog.Revision <= cmd.Expected.Catalog.Revision {
		return fmt.Errorf("no newer canonical room receipt")
	}
	desired := DshbotRoomEditProfile(cmd)
	if len(desired) != len(fresh.Profile) {
		return fmt.Errorf("profile shape changed; inspect Bot Web")
	}
	for k, v := range desired {
		if k != "updatedAt" && !sameCapabilityJSON(v, fresh.Profile[k]) {
			return fmt.Errorf("profile differs at %s; outcome remains unknown", k)
		}
	}
	if err := validateDshbotRoomProfile(fresh); err != nil {
		return err
	}
	return nil
}
func (c *Client) DshbotSaveRoomEdit(ctx context.Context, cmd DshbotRoomEditCommand) (DshbotRoomEditResult, error) {
	var out DshbotRoomEditResult
	if err := ValidateDshbotRoomEdit(cmd); err != nil {
		return out, err
	}
	payload := DshbotRoomEditProfile(cmd)
	payload["revision"], _ = json.Marshal(cmd.Expected.Catalog.Revision)
	var reply struct {
		View      json.RawMessage `json:"view"`
		Item      json.RawMessage `json:"item"`
		SessionID string          `json:"sessionId"`
		Warnings  []string        `json:"warnings"`
	}
	out.Attempted = true
	if err := c.Call(ctx, "dshbot", "bot/update", payload, &reply); err != nil {
		return out, fmt.Errorf("group save outcome unknown; read back without retry")
	}
	out.Warnings = append([]string(nil), reply.Warnings...)
	var err error
	out.Status, err = decodeDshbotRoomEdit(reply.View, cmd.Expected.Item.ID)
	if err != nil {
		return out, err
	}
	var receiptItem DshbotRoomItem
	if json.Unmarshal(reply.Item, &receiptItem) != nil || reply.SessionID != cmd.Expected.Item.SessionID || !reflect.DeepEqual(receiptItem, out.Status.Item) {
		return out, fmt.Errorf("inconsistent canonical group receipt")
	}
	return out, DshbotRoomEditApplied(cmd, out.Status)
}
