// Package settings loads dsh settings/describe namespaces and applies settings/update.
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Namespace is one settings form from describe.
type Namespace struct {
	NS               string
	Revision         float64
	Fields           []Field
	Value            json.RawMessage
	Schema           json.RawMessage
	Base             json.RawMessage
	User             json.RawMessage
	HasSecrets       bool
	SecretPaths      [][]string
	DraftRevision    float64
	HasDraftRevision bool
}

// ViewState is a UI-safe snapshot of the store.
type ViewState struct {
	ClientReady        bool
	Described          bool // This client has a fully accepted settings description.
	Loading            bool
	Saving             bool
	Writable           bool
	Err                string
	LastAction         string
	Namespaces         []Namespace
	SelectedNS         string
	Drafts             map[string]string // field key → draft for SelectedNS
	Resets             map[string]bool   // fields staged to inherit their base/default
	Dirty              bool
	PermissionDefaults PermissionDefaults
}

// Store tracks harness settings for the generic form renderer.
type Store struct {
	mu                     sync.Mutex
	client                 *dsh.Client
	describedClient        *dsh.Client
	notify                 func()
	visualReplyEnableCheck func(context.Context, string) error

	loading    bool
	saving     bool
	writable   bool
	err        error
	lastAction string

	namespaces                []Namespace
	selectedNS                string
	drafts                    map[string]map[string]string
	dirty                     map[string]map[string]bool
	resets                    map[string]map[string]bool
	generation                uint64
	origin                    string
	hostDrafts                map[string]map[string]map[string]string
	hostDirty                 map[string]map[string]map[string]bool
	hostResets                map[string]map[string]map[string]bool
	draftRevisions            map[string]float64
	draftSeeds                map[string]*draftSeed
	hostDraftRevisions        map[string]map[string]float64
	modelRoutes               []VisionRoute
	modelCatalogErr           string
	modelCatalogLoading       bool
	modelCatalogLoaded        bool
	modelCatalogRequest       uint64
	visionRoutes              []VisionRoute
	visionLoading             bool
	visionErr                 string
	permissionDefaults        PermissionDefaults
	webSearchKey              *WebSearchCredential
	themeBinding              *ThemeBinding
	instructionsWrite         *instructionsWrite
	onboardingCredentialEpoch uint64
	uploadNoticeSequence      uint64
	uploadNotice              string
	snapCache                 snapshotCache
}

type namespaceSnapKey struct {
	ns               string
	revision         float64
	draftRevision    float64
	hasDraftRevision bool
	fieldsPtr        *Field
	fieldsLen        int
	valuePtr         *byte
	valueLen         int
	schemaPtr        *byte
	schemaLen        int
	basePtr          *byte
	baseLen          int
	userPtr          *byte
	userLen          int
	hasSecrets       bool
	secretPathsPtr   *[]string
	secretPathsLen   int
}

type snapshotCache struct {
	valid   bool
	state   ViewState
	permPtr *PermissionDefaultOption
	permLen int
	nsKeys  []namespaceSnapKey
}

// New returns an empty settings store.
func New() *Store {
	return &Store{
		drafts:         map[string]map[string]string{},
		dirty:          map[string]map[string]bool{},
		resets:         map[string]map[string]bool{},
		draftRevisions: map[string]float64{},
	}
}

// Attach binds a dsh client and an optional UI refresh hook.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	s.uploadNotice = ""
	s.generation++
	s.permissionDefaults = PermissionDefaults{}
	s.modelRoutes = nil
	s.modelCatalogErr = ""
	s.modelCatalogLoading = false
	s.modelCatalogLoaded = false
	s.modelCatalogRequest++
	s.visionRoutes = nil
	s.visionLoading = false
	s.visionErr = ""
	if client != nil && s.origin != client.Endpoint.Origin {
		if s.hostDrafts == nil {
			s.hostDrafts = map[string]map[string]map[string]string{}
			s.hostDirty = map[string]map[string]map[string]bool{}
			s.hostResets = map[string]map[string]map[string]bool{}
		}
		s.hostDrafts[s.origin] = s.drafts
		s.hostDirty[s.origin] = s.dirty
		s.hostResets[s.origin] = s.resets
		if s.hostDraftRevisions == nil {
			s.hostDraftRevisions = map[string]map[string]float64{}
		}
		s.hostDraftRevisions[s.origin] = s.draftRevisions
		s.origin = client.Endpoint.Origin
		s.drafts = s.hostDrafts[s.origin]
		s.dirty = s.hostDirty[s.origin]
		s.resets = s.hostResets[s.origin]
		s.draftRevisions = s.hostDraftRevisions[s.origin]
		if s.draftRevisions == nil {
			s.draftRevisions = map[string]float64{}
		}
		if s.resets == nil {
			s.resets = map[string]map[string]bool{}
		}
		if s.drafts == nil {
			s.drafts = map[string]map[string]string{}
		}
		if s.dirty == nil {
			s.dirty = map[string]map[string]bool{}
		}
		s.namespaces = nil
		s.selectedNS = ""
		s.writable = false
		s.lastAction = ""
	}
	s.saving = false
	s.loading = false
	s.client = client
	s.notify = notify
	s.err = nil
	s.mu.Unlock()
}

// Detach clears the client when harness stops.
func (s *Store) Detach() {
	s.mu.Lock()
	s.generation++
	s.permissionDefaults = PermissionDefaults{}
	s.modelRoutes = nil
	s.modelCatalogErr = ""
	s.modelCatalogLoading = false
	s.modelCatalogLoaded = false
	s.modelCatalogRequest++
	s.visionRoutes = nil
	s.visionLoading = false
	s.visionErr = ""
	s.client = nil
	s.describedClient = nil
	s.loading = false
	s.saving = false
	s.namespaces = nil
	s.selectedNS = ""
	s.mu.Unlock()
	s.fire()
}

// Snapshot copies observable state for one frame.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

// NamespaceValue returns the raw JSON value for one namespace without cloning all namespaces.
func (s *Store) NamespaceValue(ns string) json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item := s.findNamespaceLocked(ns); item != nil {
		return item.Value
	}
	return nil
}

// EnsureLoaded starts the one initial describe for a connected Host. A refused
// read stays visible until the explicit settings/onboarding Retry action.
func (s *Store) EnsureLoaded(ctx context.Context) {
	s.mu.Lock()
	needed := s.client != nil && s.describedClient != s.client && !s.loading && s.err == nil
	s.mu.Unlock()
	if needed {
		s.RefreshDescribe(ctx)
	}
}

func makeNamespaceSnapKey(ns Namespace, draftRevisions map[string]float64) namespaceSnapKey {
	drev, hasDrev := draftRevisions[ns.NS]
	return namespaceSnapKey{
		ns:               ns.NS,
		revision:         ns.Revision,
		draftRevision:    drev,
		hasDraftRevision: hasDrev,
		fieldsPtr:        unsafe.SliceData(ns.Fields),
		fieldsLen:        len(ns.Fields),
		valuePtr:         unsafe.SliceData(ns.Value),
		valueLen:         len(ns.Value),
		schemaPtr:        unsafe.SliceData(ns.Schema),
		schemaLen:        len(ns.Schema),
		basePtr:          unsafe.SliceData(ns.Base),
		baseLen:          len(ns.Base),
		userPtr:          unsafe.SliceData(ns.User),
		userLen:          len(ns.User),
		hasSecrets:       ns.HasSecrets,
		secretPathsPtr:   unsafe.SliceData(ns.SecretPaths),
		secretPathsLen:   len(ns.SecretPaths),
	}
}

func (s *Store) snapCacheHitLocked(errStr string, clientReady, described, dirty bool) bool {
	c := &s.snapCache
	if !c.valid {
		return false
	}
	st := &c.state
	if st.ClientReady != clientReady ||
		st.Described != described ||
		st.Loading != s.loading ||
		st.Saving != s.saving ||
		st.Writable != s.writable ||
		st.LastAction != s.lastAction ||
		st.SelectedNS != s.selectedNS ||
		st.Dirty != dirty ||
		st.Err != errStr {
		return false
	}
	if st.PermissionDefaults.DefaultPreset != s.permissionDefaults.DefaultPreset ||
		st.PermissionDefaults.Loaded != s.permissionDefaults.Loaded ||
		st.PermissionDefaults.Err != s.permissionDefaults.Err ||
		c.permLen != len(s.permissionDefaults.Options) ||
		c.permPtr != unsafe.SliceData(s.permissionDefaults.Options) {
		return false
	}
	if len(c.nsKeys) != len(s.namespaces) || len(st.Namespaces) != len(s.namespaces) {
		return false
	}
	for i := range s.namespaces {
		if c.nsKeys[i] != makeNamespaceSnapKey(s.namespaces[i], s.draftRevisions) {
			return false
		}
	}
	resets := s.resets[s.selectedNS]
	if len(st.Resets) != len(resets) {
		return false
	}
	for k, v := range resets {
		if cv, ok := st.Resets[k]; !ok || cv != v {
			return false
		}
	}
	var drafts map[string]string
	if s.selectedNS != "" {
		drafts = s.drafts[s.selectedNS]
	}
	if len(st.Drafts) != len(drafts) {
		return false
	}
	for k, v := range drafts {
		if cv, ok := st.Drafts[k]; !ok || cv != v {
			return false
		}
	}
	return true
}

func (s *Store) snapshotLocked() ViewState {
	clientReady := s.client != nil
	described := clientReady && s.describedClient == s.client
	dirty := len(s.dirty[s.selectedNS]) > 0
	errStr := ""
	if s.err != nil {
		errStr = s.err.Error()
	}
	if s.snapCacheHitLocked(errStr, clientReady, described, dirty) {
		return s.snapCache.state
	}
	st := ViewState{
		PermissionDefaults: s.permissionDefaults.clone(),
		ClientReady:        clientReady,
		Described:          described,
		Loading:            s.loading,
		Saving:             s.saving,
		Writable:           s.writable,
		Err:                errStr,
		LastAction:         s.lastAction,
		SelectedNS:         s.selectedNS,
		Drafts:             map[string]string{},
		Resets:             copyFlags(s.resets[s.selectedNS]),
		Dirty:              dirty,
	}
	nsKeys := make([]namespaceSnapKey, len(s.namespaces))
	st.Namespaces = append([]Namespace(nil), s.namespaces...)
	for i := range st.Namespaces {
		nsKeys[i] = makeNamespaceSnapKey(s.namespaces[i], s.draftRevisions)
		st.Namespaces[i].DraftRevision, st.Namespaces[i].HasDraftRevision = s.draftRevisions[st.Namespaces[i].NS]
		st.Namespaces[i].Fields = append([]Field(nil), st.Namespaces[i].Fields...)
		st.Namespaces[i].Value = append(json.RawMessage(nil), st.Namespaces[i].Value...)
		st.Namespaces[i].Base = append(json.RawMessage(nil), st.Namespaces[i].Base...)
		st.Namespaces[i].User = append(json.RawMessage(nil), st.Namespaces[i].User...)
		st.Namespaces[i].SecretPaths = cloneSecretPaths(st.Namespaces[i].SecretPaths)
	}
	if s.selectedNS != "" {
		if d, ok := s.drafts[s.selectedNS]; ok {
			for k, v := range d {
				st.Drafts[k] = v
			}
		}
	}
	s.snapCache = snapshotCache{
		valid:   true,
		state:   st,
		permPtr: unsafe.SliceData(s.permissionDefaults.Options),
		permLen: len(s.permissionDefaults.Options),
		nsKeys:  nsKeys,
	}
	return st
}

// SelectedNamespace returns the active namespace tab without cloning all namespaces.
func (s *Store) SelectedNamespace() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.selectedNS
}

// Select chooses the active namespace tab.
func (s *Store) Select(ns string) {
	s.mu.Lock()
	if s.selectedNS == ns {
		s.mu.Unlock()
		return
	}
	s.selectedNS = ns
	s.ensureDraftsLocked(ns)
	s.mu.Unlock()
	s.fire()
}

// SetDraft updates one field draft for the selected namespace.
func (s *Store) SetDraft(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.selectedNS == "" {
		return
	}
	s.beginDraftLocked(s.selectedNS)
	if s.drafts[s.selectedNS] == nil {
		s.drafts[s.selectedNS] = map[string]string{}
	}
	s.drafts[s.selectedNS][key] = value
	if s.dirty == nil {
		s.dirty = map[string]map[string]bool{}
	}
	if s.dirty[s.selectedNS] == nil {
		s.dirty[s.selectedNS] = map[string]bool{}
	}
	s.dirty[s.selectedNS][key] = true
	delete(s.resets[s.selectedNS], key)
}

// RefreshDescribe loads settings/describe on a background goroutine.
func (s *Store) RefreshDescribe(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	generation := s.generation
	if client == nil {
		s.mu.Unlock()
		return
	}
	s.loading = true
	if s.describedClient != client {
		s.describedClient = nil
	}
	s.err = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		var out api.SettingsDescribeValue
		err := client.Call(ctx, "settings/describe", nil, &out)
		defaults := PermissionDefaults{}
		if err == nil {
			defaults = loadPermissionDefaults(ctx, client, out)
		}

		s.mu.Lock()
		if s.generation != generation {
			s.mu.Unlock()
			return
		}
		s.loading = false
		if err != nil {
			s.describedClient = nil
			s.err = err
			s.mu.Unlock()
			s.fire()
			return
		}
		s.writable = out.Writable
		s.permissionDefaults = defaults
		nsList := make([]Namespace, 0, len(out.Namespaces))
		for _, item := range out.Namespaces {
			fields, fErr := BuildFields(item.Schema)
			if fErr != nil {
				s.describedClient = nil
				s.err = fErr
				s.mu.Unlock()
				s.fire()
				return
			}
			nsList = append(nsList, Namespace{
				NS:          item.Ns,
				Revision:    item.Revision,
				Fields:      fields,
				Value:       append(json.RawMessage(nil), item.Value...),
				Schema:      append(json.RawMessage(nil), item.Schema...),
				Base:        append(json.RawMessage(nil), item.Base...),
				User:        append(json.RawMessage(nil), item.User...),
				HasSecrets:  hasSetSecrets(item.Secrets),
				SecretPaths: configuredSecretPaths(item.Secrets),
			})
		}
		s.namespaces = nsList
		s.onboardingCredentialEpoch++
		s.describedClient = client
		if s.selectedNS == "" && len(nsList) > 0 {
			s.selectedNS = nsList[0].NS
		}
		for _, ns := range nsList {
			s.seedDraftsLocked(ns)
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// SaveReplace posts settings/replace for the selected namespace with a full section map.
// Empty section fails closed with Err/LastAction (no RPC).
func (s *Store) SaveReplace(ctx context.Context, section map[string]json.RawMessage) {
	s.mu.Lock()
	name := s.selectedNS
	s.mu.Unlock()
	s.saveReplace(ctx, name, section)
}
func (s *Store) saveReplace(ctx context.Context, name string, section map[string]json.RawMessage) {
	s.mu.Lock()
	client := s.client
	generation := s.generation
	ns := s.findNamespaceLocked(name)
	if s.saving || client == nil || ns == nil {
		s.mu.Unlock()
		return
	}
	if ns.HasSecrets {
		s.err = fmt.Errorf("This section contains hidden secrets. Use Save to preserve them; full replacement is unavailable")
		s.lastAction = s.err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	if name == CustomInstructionsNamespace {
		var text string
		_ = json.Unmarshal(section[CustomInstructionsField], &text)
		if err := validateCustomInstructions(name, map[string]string{CustomInstructionsField: text}); err != nil {
			s.err = err
			s.mu.Unlock()
			s.fire()
			return
		}
	}
	if len(section) == 0 {
		s.err = fmt.Errorf("settings replace rejected: empty section")
		s.lastAction = s.err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	if !s.writable {
		s.lastAction = "settings read-only"
		s.mu.Unlock()
		s.fire()
		return
	}
	submitted := copyDrafts(s.drafts[ns.NS])
	submittedResets := copyFlags(s.resets[ns.NS])
	rev := s.draftRevisionLocked(*ns)
	nsName := ns.NS
	s.saving = true
	s.err = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		args := map[string]any{
			"ns":               nsName,
			"section":          section,
			"expectedRevision": rev,
		}
		var out api.SettingsNamespaceView
		err := s.verifyVisualReplyEnable(ctx, nsName, visualReplyEnabled(section) || submittedResets["enabled"], client.Endpoint.Origin, generation)
		if err == nil {
			err = client.Call(ctx, "settings/replace", args, &out)
		}
		s.mu.Lock()
		if s.generation != generation {
			s.mu.Unlock()
			return
		}
		s.saving = false
		if err != nil {
			s.err = err
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.lastAction = "settings replaced"
		s.err = nil
		for i := range s.namespaces {
			if s.namespaces[i].NS == nsName {
				s.namespaces[i].Revision = out.Revision
				s.namespaces[i].Value = append(json.RawMessage(nil), out.Value...)
				s.namespaces[i].Base = append(json.RawMessage(nil), out.Base...)
				s.namespaces[i].User = append(json.RawMessage(nil), out.User...)
				s.namespaces[i].HasSecrets = hasSetSecrets(out.Secrets)
				s.namespaces[i].SecretPaths = configuredSecretPaths(out.Secrets)
				seed := s.namespaces[i]
				s.acknowledgeDraftsLocked(seed, submitted, submittedResets)
				break
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// SaveReplaceFromDrafts replaces the stored user layer, never the resolved
// effective layer. Redacted sections cannot safely be reconstructed wholesale.
func (s *Store) SaveReplaceFromDrafts(ctx context.Context) {
	s.mu.Lock()
	ns := s.findNamespaceLocked(s.selectedNS)
	if s.client == nil || ns == nil || s.saving || s.loading {
		s.mu.Unlock()
		return
	}
	if !s.writable {
		s.lastAction = "settings read-only"
		s.mu.Unlock()
		s.fire()
		return
	}
	section, err := replacementFromDrafts(*ns, s.drafts[ns.NS], s.dirty[ns.NS], s.resets[ns.NS])
	name := ns.NS
	s.mu.Unlock()
	if err != nil {
		s.mu.Lock()
		s.err = err
		s.lastAction = err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	s.saveReplace(ctx, name, section)
}

// SaveMutate posts settings/mutate ops for one field set (path replace) when
// the generic form prefers mutate over patch update.
func (s *Store) SaveMutate(ctx context.Context, ops []json.RawMessage) {
	s.mu.Lock()
	name := s.selectedNS
	s.mu.Unlock()
	s.saveMutate(ctx, name, ops)
}

func (s *Store) saveMutate(ctx context.Context, name string, ops []json.RawMessage) {
	s.saveMutateOwned(ctx, name, ops, nil, 0)
}

func (s *Store) saveMutateOwned(ctx context.Context, name string, ops []json.RawMessage, owner *dsh.Client, ownerGeneration uint64) {
	s.mu.Lock()
	client := s.client
	generation := s.generation
	ns := s.findNamespaceLocked(name)
	if s.saving || client == nil || ns == nil || len(ops) == 0 || (owner != nil && (client != owner || generation != ownerGeneration)) {
		s.mu.Unlock()
		return
	}
	if !s.writable {
		s.lastAction = "settings read-only"
		s.mu.Unlock()
		s.fire()
		return
	}
	// Direct path writes are independent of the generic editor drafts.
	submitted := map[string]string{}
	rev := ns.Revision
	nsName := ns.NS
	if nsName == "session-log-deepseek" {
		s.uploadNotice = ""
	}
	s.saving = true
	s.err = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		args := map[string]any{
			"ns":               nsName,
			"ops":              ops,
			"expectedRevision": rev,
		}
		var out api.SettingsNamespaceView
		err := s.verifyVisualReplyEnable(ctx, nsName, visualReplyMutationEnables(ops), client.Endpoint.Origin, generation)
		if err == nil {
			err = client.Call(ctx, "settings/mutate", args, &out)
		}
		if err == nil && nsName == "session-log-deepseek" && !namespaceWriteLanded(out, nil, ops) {
			err = fmt.Errorf("Session log upload preference was not accepted")
		}
		s.mu.Lock()
		if s.generation != generation {
			s.mu.Unlock()
			return
		}
		s.saving = false
		if nsName == "session-log-deepseek" {
			s.uploadNoticeSequence++
			s.uploadNotice = "saved"
			if err != nil {
				s.uploadNotice = "failed"
			}
		}
		if err != nil {
			s.err = err
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.lastAction = "settings mutated"
		s.err = nil
		for i := range s.namespaces {
			if s.namespaces[i].NS == nsName {
				s.namespaces[i].Revision = out.Revision
				s.namespaces[i].Value = append(json.RawMessage(nil), out.Value...)
				s.namespaces[i].Base = append(json.RawMessage(nil), out.Base...)
				s.namespaces[i].User = append(json.RawMessage(nil), out.User...)
				s.namespaces[i].HasSecrets = hasSetSecrets(out.Secrets)
				s.namespaces[i].SecretPaths = configuredSecretPaths(out.Secrets)
				seed := s.namespaces[i]
				s.acknowledgeDraftsLocked(seed, submitted)
				break
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// SaveUpdate persists only staged fields. Structured edits use path mutations
// against the stored user section, preserving redacted and unrelated values.
func (s *Store) SaveUpdate(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	generation := s.generation
	ns := s.findNamespaceLocked(s.selectedNS)
	if s.saving || s.loading || client == nil || ns == nil {
		s.mu.Unlock()
		return
	}
	if !s.writable {
		s.lastAction = "settings read-only"
		s.mu.Unlock()
		s.fire()
		return
	}
	captured := *ns
	captured.Revision = s.draftRevisionLocked(captured)
	ns = &captured
	drafts := copyDrafts(s.drafts[ns.NS])
	resets := copyFlags(s.resets[ns.NS])
	if !resets[CustomInstructionsField] {
		if err := validateCustomInstructions(ns.NS, drafts); err != nil {
			s.err = err
			s.mu.Unlock()
			s.fire()
			return
		}
	}
	patch, ops, submitted, err := draftWrite(*ns, drafts, s.dirty[ns.NS], resets)
	if err != nil {
		s.err = err
		s.lastAction = err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	if len(patch) == 0 && len(ops) == 0 {
		s.acknowledgeDraftsLocked(*ns, submitted, resets)
		s.lastAction = "No settings changes"
		s.err = nil
		s.mu.Unlock()
		s.fire()
		return
	}
	s.saving = true
	s.err = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		method := "settings/update"
		args := map[string]any{"ns": ns.NS, "patch": patch, "expectedRevision": ns.Revision}
		enables := visualReplyEnabled(patch)
		if len(ops) > 0 {
			method = "settings/mutate"
			delete(args, "patch")
			args["ops"] = ops
			enables = visualReplyMutationEnables(ops)
		}
		var out api.SettingsNamespaceView
		err := s.verifyVisualReplyEnable(ctx, ns.NS, enables, client.Endpoint.Origin, generation)
		if err == nil {
			err = client.Call(ctx, method, args, &out)
		}
		s.mu.Lock()
		if s.generation != generation {
			s.mu.Unlock()
			return
		}
		s.saving = false
		if err != nil {
			s.err = err
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.lastAction = "settings saved"
		s.err = nil
		for i := range s.namespaces {
			if s.namespaces[i].NS == ns.NS {
				s.namespaces[i].Revision = out.Revision
				s.namespaces[i].Value = append(json.RawMessage(nil), out.Value...)
				s.namespaces[i].Base = append(json.RawMessage(nil), out.Base...)
				s.namespaces[i].User = append(json.RawMessage(nil), out.User...)
				s.namespaces[i].HasSecrets = hasSetSecrets(out.Secrets)
				s.namespaces[i].SecretPaths = configuredSecretPaths(out.Secrets)
				s.acknowledgeDraftsLocked(s.namespaces[i], submitted, resets)
				break
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ViewTabsEnabled reports harness viewTabs when a known namespace exposes it.
// ok is false when the field is absent (caller keeps goHideChatViewTabs).
func (s *Store) ViewTabsEnabled() (enabled bool, ok bool) {
	candidates := []struct{ ns, key string }{
		{"ui-conversation", "viewTabs"},
		{"conversation", "viewTabs"},
		{"submissionPolicy", "viewTabs"},
		{"ui", "viewTabs"},
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range candidates {
		ns := s.findNamespaceLocked(c.ns)
		if ns == nil || len(ns.Value) == 0 {
			continue
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(ns.Value, &m) != nil {
			continue
		}
		raw, found := m[c.key]
		if !found {
			continue
		}
		var b bool
		if json.Unmarshal(raw, &b) != nil {
			continue
		}
		return b, true
	}
	return false, false
}

// SyncViewTabs pushes Electron conversation/submissionPolicy viewTabs into harness
// settings when those namespaces are present. Shell still keeps goHideChatViewTabs.
func (s *Store) SyncViewTabs(ctx context.Context, show bool) {
	candidates := []struct{ ns, key string }{
		{"ui-conversation", "viewTabs"},
		{"conversation", "viewTabs"},
		{"submissionPolicy", "viewTabs"},
		{"ui", "viewTabs"},
	}
	for _, c := range candidates {
		s.mu.Lock()
		ns := s.findNamespaceLocked(c.ns)
		has := false
		if ns != nil {
			for _, f := range ns.Fields {
				if f.Key == c.key {
					has = true
					break
				}
			}
			if !has && len(ns.Value) > 0 {
				var m map[string]json.RawMessage
				if json.Unmarshal(ns.Value, &m) == nil {
					if _, ok := m[c.key]; ok {
						has = true
					}
				}
			}
		}
		s.mu.Unlock()
		if has {
			s.PatchBool(ctx, c.ns, c.key, show)
			return
		}
	}
}

// PatchBool posts settings/update for one bool key in a namespace.
func (s *Store) PatchBool(ctx context.Context, nsName, key string, value bool) {
	s.mu.Lock()
	client := s.client
	ns := s.findNamespaceLocked(nsName)
	if client == nil || ns == nil || !s.writable {
		s.mu.Unlock()
		return
	}
	rev := ns.Revision
	s.mu.Unlock()

	raw, err := PatchValue(FieldBool, fmt.Sprintf("%v", value))
	if err != nil {
		return
	}
	go func() {
		args := map[string]any{
			"ns":               nsName,
			"patch":            map[string]json.RawMessage{key: raw},
			"expectedRevision": rev,
		}
		var out api.SettingsNamespaceView
		err := client.Call(ctx, "settings/update", args, &out)
		s.mu.Lock()
		if err != nil {
			s.err = err
			s.lastAction = "viewTabs sync: " + err.Error()
		} else {
			s.lastAction = "viewTabs synced"
			for i := range s.namespaces {
				if s.namespaces[i].NS == nsName {
					s.namespaces[i].Revision = out.Revision
					s.namespaces[i].Value = append(json.RawMessage(nil), out.Value...)
					s.namespaces[i].Base = append(json.RawMessage(nil), out.Base...)
					s.namespaces[i].User = append(json.RawMessage(nil), out.User...)
					s.namespaces[i].HasSecrets = hasSetSecrets(out.Secrets)
					s.namespaces[i].SecretPaths = configuredSecretPaths(out.Secrets)
					break
				}
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) findNamespaceLocked(ns string) *Namespace {
	for i := range s.namespaces {
		if s.namespaces[i].NS == ns {
			return &s.namespaces[i]
		}
	}
	return nil
}

func (s *Store) ensureDraftsLocked(ns string) {
	if n := s.findNamespaceLocked(ns); n != nil {
		s.seedDraftsLocked(*n)
	}
}

func (s *Store) beginDraftLocked(name string) {
	if len(s.dirty[name]) > 0 {
		return
	}
	if ns := s.findNamespaceLocked(name); ns != nil {
		if s.draftRevisions == nil {
			s.draftRevisions = map[string]float64{}
		}
		s.draftRevisions[name] = ns.Revision
	}
}

func (s *Store) draftRevisionLocked(ns Namespace) float64 {
	if revision, ok := s.draftRevisions[ns.NS]; ok && len(s.dirty[ns.NS]) > 0 {
		return revision
	}
	return ns.Revision
}

// draftSeed caches the drafts one namespace value yields. BindNamespace reseeds
// on every render pass while the value rarely changes, so a field is decoded
// once per value rather than once per frame.
type draftSeed struct {
	value  string
	drafts map[draftField]string
}

type draftField struct {
	key  string
	kind FieldKind
}

func (s *Store) seedDraftsLocked(ns Namespace) {
	if s.drafts[ns.NS] == nil {
		s.drafts[ns.NS] = map[string]string{}
	}
	seed := s.draftSeeds[ns.NS]
	if seed == nil || seed.value != string(ns.Value) {
		if s.draftSeeds == nil {
			s.draftSeeds = map[string]*draftSeed{}
		}
		seed = &draftSeed{value: string(ns.Value), drafts: map[draftField]string{}}
		s.draftSeeds[ns.NS] = seed
	}
	var root map[string]json.RawMessage
	decoded, read := false, false
	for _, f := range ns.Fields {
		if s.dirty[ns.NS][f.Key] {
			continue
		}
		field := draftField{f.Key, f.Kind}
		draft, ok := seed.drafts[field]
		if !ok {
			if !read {
				root, decoded = draftRoot(ns.Value)
				read = true
			}
			draft = rootDraft(root, decoded, f.Key, f.Kind)
			seed.drafts[field] = draft
		}
		s.drafts[ns.NS][f.Key] = draft
	}
}

func copyDrafts(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *Store) fire() {
	s.mu.Lock()
	fn := s.notify
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
	s.flushQueuedTheme()
	s.flushQueuedInstructions()
}

// Only the submitted draft is acknowledged; edits made during a write survive.
func (s *Store) acknowledgeDraftsLocked(ns Namespace, submitted map[string]string, resetSnapshots ...map[string]bool) {
	var resets map[string]bool
	if len(resetSnapshots) > 0 {
		resets = resetSnapshots[0]
	}
	for key, value := range submitted {
		if s.drafts[ns.NS][key] == value && s.resets[ns.NS][key] == resets[key] {
			delete(s.dirty[ns.NS], key)
			delete(s.resets[ns.NS], key)
		}
	}
	s.seedDraftsLocked(ns)
	if len(s.dirty[ns.NS]) == 0 {
		delete(s.draftRevisions, ns.NS)
	} else if len(submitted) > 0 {
		s.draftRevisions[ns.NS] = ns.Revision
	}
}

// DiscardDrafts abandons only the selected namespace and reloads its current revision.
func (s *Store) DiscardDrafts(ctx context.Context) {
	s.mu.Lock()
	ns := s.selectedNS
	delete(s.dirty, ns)
	delete(s.resets, ns)
	delete(s.drafts, ns)
	delete(s.draftRevisions, ns)
	s.ensureDraftsLocked(ns)
	s.mu.Unlock()
	s.RefreshDescribe(ctx)
}
