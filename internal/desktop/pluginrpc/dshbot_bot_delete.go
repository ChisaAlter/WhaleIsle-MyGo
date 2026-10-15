package pluginrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode/utf16"
)

// These are read-only projections. The catalog credential triggerToken is never
// retained: deletion owns no credential and only uses the original Host command.
type DshbotDeleteCatalog struct {
	Revision               int
	Items                  []DshbotBotItem
	Routines, Tasks, Audit []map[string]json.RawMessage
}
type DshbotDeleteIdentity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}
type DshbotDeleteNamed struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type DshbotDeleteGroup struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Removed bool   `json:"removed"`
}
type DshbotDeleteTask struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
type DshbotDeleteDependencies struct {
	Groups   []DshbotDeleteGroup `json:"groups"`
	Routines []DshbotDeleteNamed `json:"routines"`
	Tasks    []DshbotDeleteTask  `json:"tasks"`
	Sources  []DshbotDeleteNamed `json:"sources"`
}
type DshbotDeletePreview struct {
	Item         DshbotDeleteIdentity     `json:"item"`
	Dependencies DshbotDeleteDependencies `json:"dependencies"`
	Preserves    []string                 `json:"preserves"`
	Warnings     []string                 `json:"warnings,omitempty"`
}
type DshbotBotDeletePlan struct {
	Catalog DshbotDeleteCatalog
	Preview DshbotDeletePreview
	Target  DshbotBotItem
	Impact  json.RawMessage
}
type DshbotBotDeleteCommand struct {
	Plan                       DshbotBotDeletePlan
	ConfirmedID, ConfirmedName string
	Acknowledged               bool
}
type DshbotBotDeleteResult struct {
	Attempted, Rejected               bool
	Removed                           string
	Catalog                           DshbotDeleteCatalog
	PreservedHistory, PreservedMemory bool
	Warnings                          []string
}

func cloneBotDelete[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	json.Unmarshal(b, &out)
	return out
}
func CloneDshbotBotDeletePlan(v DshbotBotDeletePlan) DshbotBotDeletePlan { return cloneBotDelete(v) }
func CloneDshbotBotDeleteCommand(v DshbotBotDeleteCommand) DshbotBotDeleteCommand {
	return cloneBotDelete(v)
}
func CloneDshbotDeleteCatalog(v DshbotDeleteCatalog) DshbotDeleteCatalog { return cloneBotDelete(v) }
func SameDshbotDeleteCatalog(a, b DshbotDeleteCatalog) bool {
	aRaw, _ := json.Marshal(a)
	bRaw, _ := json.Marshal(b)
	return sameCapabilityJSON(aRaw, bRaw)
}
func SameDshbotBotDeletePlan(a, b DshbotBotDeletePlan) bool {
	aRaw, _ := json.Marshal(a)
	bRaw, _ := json.Marshal(b)
	return sameCapabilityJSON(aRaw, bRaw)
}
func deleteString(row map[string]json.RawMessage, key string) (string, error) {
	var v string
	if json.Unmarshal(row[key], &v) != nil {
		return "", fmt.Errorf("incomplete deletion data: %s", key)
	}
	return v, nil
}
func deleteStrings(row map[string]json.RawMessage, key string) ([]string, error) {
	var v []string
	if json.Unmarshal(row[key], &v) != nil || v == nil {
		return nil, fmt.Errorf("incomplete deletion list: %s", key)
	}
	return v, nil
}
func deleteRows(raw json.RawMessage) ([]map[string]json.RawMessage, error) {
	var rows []map[string]json.RawMessage
	if json.Unmarshal(raw, &rows) != nil || rows == nil {
		return nil, fmt.Errorf("incomplete deletion catalog list")
	}
	for _, row := range rows {
		if row == nil {
			return nil, fmt.Errorf("invalid deletion catalog row")
		}
	}
	return rows, nil
}
func decodeDshbotDeleteCatalog(raw json.RawMessage) (DshbotDeleteCatalog, error) {
	basic, err := decodeDshbotBotCatalog(raw)
	if err != nil {
		return DshbotDeleteCatalog{}, err
	}
	var wire struct {
		Value struct{ Routines, Tasks, Audit json.RawMessage }
	}
	if json.Unmarshal(raw, &wire) != nil {
		return DshbotDeleteCatalog{}, fmt.Errorf("invalid deletion catalog")
	}
	out := DshbotDeleteCatalog{Revision: basic.Revision, Items: basic.Items}
	if out.Routines, err = deleteRows(wire.Value.Routines); err != nil {
		return out, err
	}
	if out.Tasks, err = deleteRows(wire.Value.Tasks); err != nil {
		return out, err
	}
	if out.Audit, err = deleteRows(wire.Value.Audit); err != nil {
		return out, err
	}
	return out, nil
}
func deleteDisplayName(item DshbotBotItem) string {
	if item.Kind != "room" && strings.TrimSpace(item.Title) != "" {
		return strings.TrimSpace(item.Title)
	}
	if strings.TrimSpace(item.Name) != "" {
		return strings.TrimSpace(item.Name)
	}
	return "Bot"
}
func deleteNotepadFile(id string) string {
	// JS /[^a-zA-Z0-9._-]/g operates on UTF-16 code units, including surrogates.
	var b strings.Builder
	for _, u := range utf16.Encode([]rune(id)) {
		if u >= 'a' && u <= 'z' || u >= 'A' && u <= 'Z' || u >= '0' && u <= '9' || u == '.' || u == '_' || u == '-' {
			b.WriteByte(byte(u))
		} else {
			b.WriteByte('_')
		}
	}
	safe := b.String()
	if safe == "" {
		safe = "unknown"
	}
	return "dshbot-routine/" + safe + ".md"
}
func deleteTerminal(s string) bool { return s == "completed" || s == "failed" || s == "cancelled" }

// The native projection supplements the original preview; it never replaces
// the Host's deletion algorithm. Dynamic activity and notepads are not locked.
func BuildDshbotBotDeletePlan(c DshbotDeleteCatalog, p DshbotDeletePreview, id string) (DshbotBotDeletePlan, error) {
	fail := func() (DshbotBotDeletePlan, error) {
		return DshbotBotDeletePlan{}, fmt.Errorf("deletion preview/catalog is incomplete or changed; refresh without deleting")
	}
	if c.Revision < 0 || c.Routines == nil || c.Tasks == nil || c.Audit == nil {
		return fail()
	}
	var target DshbotBotItem
	found := 0
	ids := map[string]bool{}
	for _, item := range c.Items {
		if item.ID == "" || ids[item.ID] || item.SessionID == "" || (item.Kind != "bot" && item.Kind != "room") {
			return fail()
		}
		ids[item.ID] = true
		if item.ID == id {
			target = item
			found++
		}
	}
	if found != 1 || target.Kind != "bot" || target.Hidden || target.Name == "" {
		return fail()
	}
	expected := DshbotDeletePreview{Item: DshbotDeleteIdentity{ID: id, Name: deleteDisplayName(target), Kind: "bot"}, Dependencies: DshbotDeleteDependencies{Groups: []DshbotDeleteGroup{}, Routines: []DshbotDeleteNamed{}, Tasks: []DshbotDeleteTask{}, Sources: []DshbotDeleteNamed{}}, Preserves: []string{"conversation-history", "memory"}, Warnings: p.Warnings}
	routines, tasks, groups, sources, inboxes, notepads, releases, retained := []any{}, []any{}, []any{}, []any{}, []any{}, []any{}, []any{}, []any{}
	removedRoutines, cancelledTasks, removedItems := map[string]bool{}, map[string]bool{}, map[string]bool{id: true}
	seenRoutines, seenTasks := map[string]bool{}, map[string]bool{}
	for _, r := range c.Routines {
		rid, e := deleteString(r, "id")
		owner, e2 := deleteString(r, "botId")
		name, e3 := deleteString(r, "name")
		if e != nil || e2 != nil || e3 != nil || rid == "" || seenRoutines[rid] {
			return fail()
		}
		seenRoutines[rid] = true
		if owner == id {
			removedRoutines[rid] = true
			routines = append(routines, r)
			expected.Dependencies.Routines = append(expected.Dependencies.Routines, DshbotDeleteNamed{ID: rid, Name: name})
			shared := []any{}
			for _, other := range c.Routines {
				oid, _ := deleteString(other, "id")
				if oid != rid && deleteNotepadFile(oid) == deleteNotepadFile(rid) {
					shared = append(shared, other)
				}
			}
			notepads = append(notepads, map[string]any{"routineId": rid, "fileRelativeToHostHome": deleteNotepadFile(rid), "otherRoutinesSharingFile": shared, "effect": "permanent removal of entire file at execution time; no content revision or undo"})
		}
	}
	for _, task := range c.Tasks {
		tid, e := deleteString(task, "id")
		status, e2 := deleteString(task, "status")
		from, e3 := deleteString(task, "fromId")
		to, e4 := deleteString(task, "toId")
		if e != nil || e2 != nil || e3 != nil || e4 != nil || tid == "" || seenTasks[tid] {
			return fail()
		}
		seenTasks[tid] = true
		if !deleteTerminal(status) && (from == id || to == id) {
			if _, e := deleteRows(task["events"]); e != nil {
				return fail()
			}
			cancelledTasks[tid] = true
			tasks = append(tasks, map[string]any{"before": task, "afterStatus": "cancelled", "error": "Bot profile was removed.", "appendedEvent": map[string]any{"type": "cancelled", "actorId": "user", "detail": "Bot profile removed"}})
			expected.Dependencies.Tasks = append(expected.Dependencies.Tasks, DshbotDeleteTask{ID: tid, Status: status})
		}
	}
	for _, item := range c.Items {
		members, e := deleteStrings(item.Profile, "memberBotIds")
		senders, e2 := deleteStrings(item.Profile, "allowedSenderIds")
		if e != nil || e2 != nil {
			return fail()
		}
		if item.Kind == "room" && slices.Contains(members, id) {
			after := []string{}
			for _, m := range members {
				if m != id {
					after = append(after, m)
				}
			}
			removed := len(after) < 2
			removedItems[item.ID] = removed
			expected.Dependencies.Groups = append(expected.Dependencies.Groups, DshbotDeleteGroup{ID: item.ID, Name: deleteDisplayName(item), Removed: removed})
			groups = append(groups, map[string]any{"id": item.ID, "name": item.Name, "canonicalSession": item.SessionID, "beforeMemberIds": members, "afterMemberIds": after, "removeRoomCatalogEntry": removed})
			if removed {
				retained = append(retained, map[string]any{"roomId": item.ID, "roomSessionId": item.SessionID, "memberBotIds": members, "effect": "group member Sessions remain hidden and intact"})
			} else {
				releases = append(releases, map[string]any{"role": "member", "roomId": item.ID, "roomSessionId": item.SessionID, "botId": id})
			}
		}
		if item.Kind != "room" && slices.Contains(senders, id) {
			after := []string{}
			for _, sender := range senders {
				if sender != id {
					after = append(after, sender)
				}
			}
			expected.Dependencies.Sources = append(expected.Dependencies.Sources, DshbotDeleteNamed{ID: item.ID, Name: deleteDisplayName(item)})
			sources = append(sources, map[string]any{"id": item.ID, "name": item.Name, "beforeAllowedSenderIds": senders, "afterAllowedSenderIds": after})
		}
	}
	for _, item := range c.Items {
		mail, e := deleteRows(item.Profile["inbox"])
		if e != nil {
			return fail()
		}
		drop := []any{}
		for _, m := range mail {
			task, _ := deleteString(m, "taskId")
			routine, _ := deleteString(m, "routineId")
			if removedItems[item.ID] || cancelledTasks[task] || removedRoutines[routine] {
				drop = append(drop, m)
			}
		}
		if len(drop) > 0 {
			inboxes = append(inboxes, map[string]any{"ownerId": item.ID, "ownerName": item.Name, "entriesRemoved": drop})
		}
		if removedItems[item.ID] {
			releases = append(releases, map[string]any{"role": item.Kind, "itemId": item.ID, "canonicalSession": item.SessionID, "stopAttempted": true})
		}
	}
	if !reflect.DeepEqual(expected, p) {
		return fail()
	}
	prune := len(c.Audit) + 1 - 200
	if prune < 0 {
		prune = 0
	}
	oldAudit := c.Audit[:prune]
	impact := map[string]any{"targetCatalogProfileRemoved": target.Profile, "groups": groups, "routineCatalogEntriesRemoved": routines, "routineNotepadFilesPermanentlyRemoved": notepads, "unfinishedTasksCancelled": tasks, "peerSourcePoliciesChanged": sources, "inboxEntriesRemoved": inboxes, "sessionPresentationsReleased": releases, "disbandedGroupMemberSessionsRetained": retained, "auditEntriesPruned": oldAudit, "auditAppended": map[string]any{"type": "bot.deleted", "fromId": "user", "toId": id, "maxEntries": 200}, "preserves": p.Preserves}
	body, e := json.MarshalIndent(impact, "", "  ")
	if e != nil {
		return fail()
	}
	return DshbotBotDeletePlan{Catalog: CloneDshbotDeleteCatalog(c), Preview: cloneBotDelete(p), Target: CloneDshbotBotItem(target), Impact: body}, nil
}
func ValidateDshbotBotDelete(cmd DshbotBotDeleteCommand) error {
	p, e := BuildDshbotBotDeletePlan(cmd.Plan.Catalog, cmd.Plan.Preview, cmd.Plan.Target.ID)
	if e != nil {
		return e
	}
	if !SameDshbotBotDeletePlan(p, cmd.Plan) {
		return fmt.Errorf("deletion review changed")
	}
	if !cmd.Acknowledged || cmd.ConfirmedID != p.Target.ID || cmd.ConfirmedName != p.Target.Name {
		return fmt.Errorf("confirm irreversible effects and type the exact original Bot name and ID for this deletion")
	}
	return nil
}
func (c *Client) DshbotBotDeletionCatalog(ctx context.Context) (DshbotDeleteCatalog, error) {
	var reply struct{ View json.RawMessage }
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{}, &reply); err != nil {
		return DshbotDeleteCatalog{}, err
	}
	return decodeDshbotDeleteCatalog(reply.View)
}
func (c *Client) DshbotBotDeletePlan(ctx context.Context, id string) (DshbotBotDeletePlan, error) {
	before, e := c.DshbotBotDeletionCatalog(ctx)
	if e != nil {
		return DshbotBotDeletePlan{}, e
	}
	var raw json.RawMessage
	if e = c.Call(ctx, "dshbot", "bot/delete-preview", map[string]any{"id": id}, &raw); e != nil {
		return DshbotBotDeletePlan{}, e
	}
	var preview DshbotDeletePreview
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if !validCapabilityUnicode(raw) {
		return DshbotBotDeletePlan{}, fmt.Errorf("invalid deletion preview text")
	}
	if e = decoder.Decode(&preview); e != nil {
		return DshbotBotDeletePlan{}, e
	}
	after, e := c.DshbotBotDeletionCatalog(ctx)
	if e != nil {
		return DshbotBotDeletePlan{}, e
	}
	if !SameDshbotDeleteCatalog(before, after) {
		return DshbotBotDeletePlan{}, fmt.Errorf("catalog changed during deletion preview; refresh without deleting")
	}
	return BuildDshbotBotDeletePlan(after, preview, id)
}
func (c *Client) DshbotDeleteBot(ctx context.Context, cmd DshbotBotDeleteCommand) (DshbotBotDeleteResult, error) {
	var out DshbotBotDeleteResult
	if err := ValidateDshbotBotDelete(cmd); err != nil {
		out.Rejected = true
		return out, err
	}
	var reply struct {
		View             json.RawMessage
		Removed          string   `json:"removed"`
		PreservedHistory *bool    `json:"preservedHistory"`
		PreservedMemory  *bool    `json:"preservedMemory"`
		Warnings         []string `json:"warnings"`
	}
	out.Attempted = true
	err := c.Call(ctx, "dshbot", "bot/delete", map[string]any{"id": cmd.Plan.Target.ID, "revision": cmd.Plan.Catalog.Revision, "confirmCascade": true}, &reply)
	if err != nil {
		message := err.Error()
		if strings.HasPrefix(message, "pluginrpc: ") && !strings.HasPrefix(message, "pluginrpc: HTTP ") {
			_, host, ok := strings.Cut(strings.TrimPrefix(message, "pluginrpc: "), ": ")
			if ok && slices.Contains([]string{"Bot catalog is read-only.", "Bot catalog changed. Refresh and try again.", "Bot profile is unavailable.", "Delete requires an explicit dependency confirmation."}, host) {
				out.Rejected = true
			}
		}
		return out, err
	}
	out.Warnings = append([]string(nil), reply.Warnings...)
	out.Removed = reply.Removed
	out.PreservedHistory = reply.PreservedHistory != nil && *reply.PreservedHistory
	out.PreservedMemory = reply.PreservedMemory != nil && *reply.PreservedMemory
	out.Catalog, err = decodeDshbotDeleteCatalog(reply.View)
	if err != nil {
		return out, err
	}
	return out, ValidateDshbotBotDeleteReceipt(cmd, out)
}
func ValidateDshbotBotDeleteReceipt(cmd DshbotBotDeleteCommand, out DshbotBotDeleteResult) error {
	if out.Removed != cmd.Plan.Target.ID || out.Catalog.Revision <= cmd.Plan.Catalog.Revision || !out.PreservedHistory || !out.PreservedMemory {
		return fmt.Errorf("incomplete deletion receipt; outcome remains unknown")
	}
	for _, item := range out.Catalog.Items {
		if item.ID == cmd.Plan.Target.ID {
			return fmt.Errorf("deleted identity still present in receipt; outcome remains unknown")
		}
	}
	return nil
}

// Readback is evidence about catalog state only. Presence or absence does not
// prove which request acted, runtime cancellation, notepad removal, or rollback.
func DshbotBotDeleteObservation(p DshbotBotDeletePlan, c DshbotDeleteCatalog) string {
	targetState := "absent"
	var target any
	groups, routines, tasks, sources := []any{}, []any{}, []any{}, []any{}
	for _, item := range c.Items {
		if item.ID == p.Target.ID {
			targetState = "present"
			target = item.Profile
		}
		for _, g := range p.Preview.Dependencies.Groups {
			if g.ID == item.ID {
				groups = append(groups, item.Profile)
			}
		}
		for _, s := range p.Preview.Dependencies.Sources {
			if s.ID == item.ID {
				sources = append(sources, item.Profile)
			}
		}
	}
	for _, r := range c.Routines {
		id, _ := deleteString(r, "id")
		for _, prior := range p.Preview.Dependencies.Routines {
			if prior.ID == id {
				routines = append(routines, r)
			}
		}
	}
	for _, task := range c.Tasks {
		id, _ := deleteString(task, "id")
		for _, prior := range p.Preview.Dependencies.Tasks {
			if prior.ID == id {
				tasks = append(tasks, task)
			}
		}
	}
	b, _ := json.MarshalIndent(map[string]any{"outcome": "unknown; readback cannot establish this attempt's effects or safe retry", "latestCatalogRevision": c.Revision, "originalTargetId": p.Target.ID, "originalCanonicalSession": p.Target.SessionID, "targetCatalogState": targetState, "currentTarget": target, "currentRelatedGroups": groups, "currentRelatedRoutines": routines, "currentRelatedTasks": tasks, "currentRelatedSources": sources, "runtimeAndNotepadEffects": "unknown; no automatic retry or rollback"}, "", "  ")
	return string(b)
}
