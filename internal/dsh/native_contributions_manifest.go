package dsh

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

type nativeContributionRelease struct {
	Version string  `json:"version"`
	Commit  *string `json:"commit,omitempty"`
}
type nativeContributionEntry struct {
	ID, URL, Rev     string
	Inject, External []string
	Immediately      bool
}
type nativeContributionBatch struct {
	Phase, URL, Rev string
	Entries         []string
}
type nativeContributionGraph struct {
	Rev     string
	Release *nativeContributionRelease
	Entries []nativeContributionEntry
	Batches []nativeContributionBatch
	index   map[string]int
}

var errNativeContributionManifest = errors.New("invalid native contribution graph")

// Mirrors the fixed source parseBootManifest wire checks: typed release and
// rows, unique entry ids/batch URLs, phases, and exactly one batch per entry.
// URLs remain opaque data. No URL in this manifest is fetched or executed.
func parseNativeContributionGraph(raw []byte) (*nativeContributionGraph, error) {
	object, ok := nativeContributionObject(raw)
	if !ok || len(raw) > nativeContributionLimit {
		return nil, errNativeContributionManifest
	}
	rev, ok := nativeContributionString(object["rev"])
	if !ok {
		return nil, errNativeContributionManifest
	}
	g := &nativeContributionGraph{Rev: rev, index: make(map[string]int)}
	if release, exists := object["release"]; exists {
		value, ok := nativeContributionObject(release)
		if !ok {
			return nil, errNativeContributionManifest
		}
		version, ok := nativeContributionString(value["version"])
		if !ok {
			return nil, errNativeContributionManifest
		}
		g.Release = &nativeContributionRelease{Version: version}
		if rawCommit, exists := value["commit"]; exists {
			commit, ok := nativeContributionString(rawCommit)
			if !ok {
				return nil, errNativeContributionManifest
			}
			g.Release.Commit = &commit
		}
	}
	entries, ok := nativeContributionArray(object["entries"])
	if !ok {
		return nil, errNativeContributionManifest
	}
	for _, rawEntry := range entries {
		row, ok := nativeContributionObject(rawEntry)
		if !ok {
			return nil, errNativeContributionManifest
		}
		id, idOK := nativeContributionString(row["id"])
		url, urlOK := nativeContributionString(row["url"])
		revision, revOK := nativeContributionString(row["rev"])
		if !idOK || !urlOK || !revOK {
			return nil, errNativeContributionManifest
		}
		if _, duplicate := g.index[id]; duplicate {
			return nil, errNativeContributionManifest
		}
		inject, injectOK := nativeContributionStrings(row["inject"], true)
		external, externalOK := nativeContributionStrings(row["external"], true)
		if !injectOK || !externalOK {
			return nil, errNativeContributionManifest
		}
		immediately := false
		if flag, exists := row["immediately"]; exists {
			flag = bytes.TrimSpace(flag)
			if string(flag) != "true" && string(flag) != "false" {
				return nil, errNativeContributionManifest
			}
			immediately = string(flag) == "true"
		}
		g.index[id] = len(g.Entries)
		g.Entries = append(g.Entries, nativeContributionEntry{ID: id, URL: url, Rev: revision, Inject: inject, External: external, Immediately: immediately})
	}
	batches, ok := nativeContributionArray(object["batches"])
	if !ok {
		return nil, errNativeContributionManifest
	}
	assigned, urls := make(map[string]bool), make(map[string]bool)
	for _, rawBatch := range batches {
		row, ok := nativeContributionObject(rawBatch)
		if !ok {
			return nil, errNativeContributionManifest
		}
		phase, phaseOK := nativeContributionString(row["phase"])
		url, urlOK := nativeContributionString(row["url"])
		revision, revOK := nativeContributionString(row["rev"])
		ids, idsOK := nativeContributionStrings(row["entries"], false)
		if !phaseOK || (phase != "bootstrap" && phase != "application") || !urlOK || !revOK || !idsOK || len(ids) == 0 || urls[url] {
			return nil, errNativeContributionManifest
		}
		urls[url] = true
		for _, id := range ids {
			if _, exists := g.index[id]; !exists || assigned[id] {
				return nil, errNativeContributionManifest
			}
			assigned[id] = true
		}
		g.Batches = append(g.Batches, nativeContributionBatch{Phase: phase, URL: url, Rev: revision, Entries: ids})
	}
	if len(assigned) != len(g.Entries) {
		return nil, errNativeContributionManifest
	}
	return g, nil
}

func nativeContributionObject(raw []byte) (map[string]json.RawMessage, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return nil, false
	}
	var value map[string]json.RawMessage
	if json.Unmarshal(raw, &value) != nil || value == nil {
		return nil, false
	}
	return value, true
}
func nativeContributionString(raw []byte) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '"' {
		return "", false
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}
func nativeContributionArray(raw []byte) ([]json.RawMessage, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return nil, false
	}
	var value []json.RawMessage
	if json.Unmarshal(raw, &value) != nil || len(value) > 4096 {
		return nil, false
	}
	return value, true
}
func nativeContributionStrings(raw []byte, optional bool) ([]string, bool) {
	if len(raw) == 0 && optional {
		return nil, true
	}
	values, ok := nativeContributionArray(raw)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(values))
	for _, raw := range values {
		value, ok := nativeContributionString(raw)
		if !ok {
			return nil, false
		}
		result = append(result, value)
	}
	return result, true
}

func sameNativeContributionRelease(left, right *nativeContributionRelease) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if left.Version != right.Version {
		return false
	}
	if left.Commit == nil || right.Commit == nil {
		return left.Commit == nil && right.Commit == nil
	}
	return *left.Commit == *right.Commit
}

const (
	nativeConversation   = "@deepseek-ai/dsh-client-ui-conversation"
	nativeModelSelection = "@deepseek-ai/dsh-client-ui-model-selection"
	nativePermissions    = "@deepseek-ai/dsh-client-ui-permission-presets"
	nativeFeedback       = "@deepseek-ai/dsh-client-ui-message-feedback"
	nativeLabel1017      = "@deepseek-ai/dsh-client-ui-deepseek-model-label-1017"
	nativeCommands       = "@deepseek-ai/dsh-client-ui-commands"
)

// Only these five source package owners map to registered Go adapters. Their
// fixed package-manifest bases remain necessary even if a received row omits
// inject. Declared inject edges must exist too; /client is the original module
// system's one normalized export suffix. Presence never means React ACTIVE.
func (g *nativeContributionGraph) available(id string, required ...string) bool {
	present := func(id string) bool {
		index, exists := g.index[id]
		if !exists {
			return false
		}
		for _, dependency := range g.Entries[index].Inject {
			if _, exists := g.index[strings.TrimSuffix(dependency, "/client")]; !exists {
				return false
			}
		}
		return true
	}
	if !present(id) {
		return false
	}
	for _, id := range required {
		if !present(id) {
			return false
		}
	}
	return true
}

func (g *nativeContributionGraph) snapshot() NativeContributions {
	const (
		connection = "@deepseek-ai/dsh-client-connection"
		sessionAPI = "@deepseek-ai/dsh-api-session-controller"
		locale     = "@deepseek-ai/dsh-client-locale"
		renderer   = "@deepseek-ai/dsh-client-ui-renderer"
		remotes    = "@deepseek-ai/dsh-api-remotes"
	)
	conversationBase := []string{nativeConversation, connection, sessionAPI, "@deepseek-ai/dsh-client-file-upload", locale, "@deepseek-ai/dsh-client-ui-layout", renderer, "@deepseek-ai/dsh-client-ui-session", "@deepseek-ai/dsh-client-ui-settings", "@deepseek-ai/dsh-client-ui-workspace"}
	commandBase := append([]string{nativeCommands, remotes, locale, "@deepseek-ai/dsh-client-ui-input-trigger"}, conversationBase...)
	modelBase := append([]string{nativeModelSelection, sessionAPI, locale, remotes}, commandBase...)
	permissionBase := append([]string{nativePermissions}, commandBase...)
	feedbackBase := append([]string{nativeFeedback, renderer}, commandBase...)
	conversation := g.available(conversationBase[0], conversationBase[1:]...)
	commands := conversation && g.available(commandBase[0], commandBase[1:]...)
	model := commands && g.available(modelBase[0], modelBase[1:]...)
	// Retain the whole graph identity for diagnostics. Popup ownership instead
	// uses the port identities below: ClientEntries.reconcile keeps unchanged
	// entry fibers when an unrelated plugin or delivery batch changes.
	canonical, _ := json.Marshal(g)
	digest := sha256.Sum256(canonical)
	snapshot := NativeContributions{
		Known: true, Revision: hex.EncodeToString(digest[:]),
		FileCommand:       commands,
		ModelCommand:      model,
		PermissionCommand: commands && g.available(permissionBase[0], permissionBase[1:]...),
		FeedbackCommand:   commands && g.available(feedbackBase[0], feedbackBase[1:]...),
		ModelLabel1017:    conversation && model && g.available(nativeLabel1017, nativeModelSelection, renderer, locale),
	}
	if snapshot.FileCommand {
		snapshot.fileRevision = g.commandRevision(commandBase)
	}
	if snapshot.ModelCommand {
		snapshot.modelRevision = g.commandRevision(modelBase)
	}
	if snapshot.PermissionCommand {
		snapshot.permissionRevision = g.commandRevision(permissionBase)
	}
	if snapshot.FeedbackCommand {
		snapshot.feedbackRevision = g.commandRevision(feedbackBase)
	}
	if snapshot.ModelLabel1017 {
		snapshot.modelLabelRevision = g.commandRevision([]string{nativeLabel1017})
	}
	if g.Release != nil {
		snapshot.ReleaseVersion = g.Release.Version
		if g.Release.Commit != nil {
			snapshot.ReleaseCommit = *g.Release.Commit
		}
	}
	return snapshot
}

// Only graph rows are revisioned code. External requests may instead name
// shell-supplied static modules; they are not invented as entries or fetched.
// A visited set bounds cycles to the already validated graph's row count.
func (g *nativeContributionGraph) commandRevision(roots []string) string {
	type target struct{ ID, Rev string }
	seen := make(map[string]bool)
	pending := append([]string(nil), roots...)
	targets := make([]target, 0, len(roots))
	for len(pending) > 0 {
		id := strings.TrimSuffix(pending[len(pending)-1], "/client")
		pending = pending[:len(pending)-1]
		if seen[id] {
			continue
		}
		seen[id] = true
		index, exists := g.index[id]
		if !exists {
			continue
		}
		row := g.Entries[index]
		targets = append(targets, target{row.ID, row.Rev})
		pending = append(pending, row.Inject...)
		pending = append(pending, row.External...)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	canonical, _ := json.Marshal(targets)
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:])
}
