// Package settings loads dsh settings/describe namespaces and applies settings/update.
package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/state"
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
	HasDocument        bool
	Described          bool // This client has a fully accepted settings description.
	Loading            bool
	Saving             bool
	PreferencePending  map[string]bool // namespace.field; independent immediate preference writes
	PreferenceErrors   map[string]string
	Writable           bool
	Err                string
	ErrDetail          string // unlocalized text behind Err: the log and details only, never the sentence
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

	loading     bool
	saving      bool
	hasDocument bool
	writable    bool
	err         error
	lastAction  string

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
	providerPage              uint64
	providerDirectory         api.ListConfigurableProvidersResult
	providerRegistered        api.ListProvidersResult
	providerDirectoryErr      string
	providerDirectoryLoaded   bool
	providerDirectoryLoading  bool
	modelCatalog              api.ModelCatalog
	modelRoutes               []VisionRoute
	modelCatalogErr           string
	modelCatalogLoading       bool
	modelCatalogLoaded        bool
	modelCatalogRequest       uint64
	describeRequest           uint64
	visionRoutes              []VisionRoute
	visionLoading             bool
	visionErr                 string
	permissionDefaults        PermissionDefaults
	webSearchKey              *WebSearchCredential
	deepSeekKeyRef            string
	deepSeekKeyConfigured     bool
	deepSeekKeyWritable       bool
	deepSeekKeyLoading        bool
	deepSeekKeyErr            string
	themeBinding              *ThemeBinding
	instructionsWrite         *instructionsWrite
	preferenceWrites          map[string]*preferenceWriteQueue
	preferenceErrors          map[string]string
	patchBoolQueues           map[string]*patchBoolQueue
	onboardingCredentialEpoch uint64
	uploadNoticeSequence      uint64
	uploadNotice              string
	pub                       state.Published[ViewState]
	viewStale                 bool // a change viewMatchesLocked does not cover; publish on the next unlock
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
	s.preferenceWrites = nil
	s.preferenceErrors = nil
	s.patchBoolQueues = nil
	s.viewStale = true
	s.uploadNotice = ""
	s.generation++
	s.hasDocument = false
	s.permissionDefaults = PermissionDefaults{}
	s.modelRoutes = nil
	s.modelCatalog = api.ModelCatalog{}
	s.providerDirectory = nil
	s.providerRegistered = nil
	s.providerDirectoryErr = ""
	s.providerDirectoryLoaded, s.providerDirectoryLoading = false, false
	s.modelCatalogErr = ""
	s.modelCatalogLoading = false
	s.modelCatalogLoaded = false
	s.modelCatalogRequest++
	s.describeRequest++
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
	s.unlockPublish()
}

// Detach clears the client when harness stops.
func (s *Store) Detach() {
	s.mu.Lock()
	s.preferenceWrites = nil
	s.preferenceErrors = nil
	s.patchBoolQueues = nil
	s.viewStale = true
	s.generation++
	s.hasDocument = false
	s.permissionDefaults = PermissionDefaults{}
	s.modelRoutes = nil
	s.modelCatalog = api.ModelCatalog{}
	s.providerDirectory = nil
	s.providerRegistered = nil
	s.providerDirectoryErr = ""
	s.providerDirectoryLoaded, s.providerDirectoryLoading = false, false
	s.modelCatalogErr = ""
	s.modelCatalogLoading = false
	s.modelCatalogLoaded = false
	s.modelCatalogRequest++
	s.describeRequest++
	s.visionRoutes = nil
	s.visionLoading = false
	s.visionErr = ""
	s.client = nil
	s.describedClient = nil
	s.loading = false
	s.saving = false
	s.namespaces = nil
	s.selectedNS = ""
	s.unlockPublish()
	s.fire()
}

// Snapshot returns the current published version. The same *Version is
// shared by every reader until a change publishes the next revision; its
// Value is immutable. Namespace payloads (Fields, Value, Schema, Base, User,
// SecretPaths) are shared with the store, which replaces them wholesale and
// never edits them in place.
func (s *Store) Snapshot() *state.Version[ViewState] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	s.viewStale = true
	s.unlockPublish()
	return s.pub.Load()
}

// Changed closes when the next version is published.
func (s *Store) Changed() <-chan struct{} { return s.pub.Changed() }

// NamespaceValue returns the raw JSON value for one namespace without cloning all namespaces.
func (s *Store) NamespaceValue(ns string) json.RawMessage {
	s.mu.Lock()
	defer s.unlockPublish()
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
	s.unlockPublish()
	if needed {
		s.RefreshDescribe(ctx)
	}
}

// unlockPublish publishes the view when the working state no longer matches
// the current version, then releases mu. Every critical section ends with
// it, so writes publish on the write path and reads never build a view. The
// match is a content comparison of the published value against the working
// fields (shared payloads compare in constant time); state the comparison
// does not cover — queued preference and theme writes — marks viewStale.
func (s *Store) unlockPublish() {
	if v := s.pub.Load(); v == nil || s.viewStale || !s.viewMatchesLocked(&v.Value) {
		s.pub.Publish(s.buildViewLocked())
		s.viewStale = false
	}
	s.mu.Unlock()
}

func (s *Store) errTextLocked() string {
	if s.err != nil {
		return s.err.Error()
	}
	return ""
}

// viewMatchesLocked reports whether st still describes the working state.
// It allocates nothing.
func (s *Store) viewMatchesLocked(st *ViewState) bool {
	clientReady := s.client != nil
	described := clientReady && s.describedClient == s.client
	if st.ClientReady != clientReady ||
		st.HasDocument != s.hasDocument ||
		st.Described != described ||
		st.Loading != s.loading ||
		st.Saving != (s.saving || s.themeWriteInFlightLocked(s.selectedNS)) ||
		st.Writable != s.writable ||
		st.LastAction != s.lastAction ||
		st.SelectedNS != s.selectedNS ||
		st.Dirty != (len(s.dirty[s.selectedNS]) > 0) ||
		st.Err != s.errTextLocked() ||
		st.ErrDetail != writeRefusalDetail(s.err) {
		return false
	}
	if st.PermissionDefaults.DefaultPreset != s.permissionDefaults.DefaultPreset ||
		st.PermissionDefaults.Loaded != s.permissionDefaults.Loaded ||
		st.PermissionDefaults.Err != s.permissionDefaults.Err ||
		!slices.Equal(st.PermissionDefaults.Options, s.permissionDefaults.Options) {
		return false
	}
	if len(st.Namespaces) != len(s.namespaces) {
		return false
	}
	for i := range s.namespaces {
		if !namespaceMatches(&st.Namespaces[i], &s.namespaces[i], s.draftRevisions) {
			return false
		}
	}
	if !flagsMatch(st.Resets, s.resets[s.selectedNS]) {
		return false
	}
	var drafts map[string]string
	if s.selectedNS != "" {
		drafts = s.drafts[s.selectedNS]
	}
	return maps.Equal(st.Drafts, drafts)
}

func flagsMatch(a, b map[string]bool) bool {
	return maps.Equal(a, b)
}

// namespaceMatches compares a published namespace with the working one by
// content; payloads the store shares compare in constant time.
func namespaceMatches(pub, cur *Namespace, draftRevisions map[string]float64) bool {
	drev, hasDrev := draftRevisions[cur.NS]
	return pub.NS == cur.NS && pub.Revision == cur.Revision &&
		pub.DraftRevision == drev && pub.HasDraftRevision == hasDrev &&
		pub.HasSecrets == cur.HasSecrets &&
		bytes.Equal(pub.Value, cur.Value) && bytes.Equal(pub.Schema, cur.Schema) &&
		bytes.Equal(pub.Base, cur.Base) && bytes.Equal(pub.User, cur.User) &&
		slices.EqualFunc(pub.Fields, cur.Fields, fieldEqual) &&
		slices.EqualFunc(pub.SecretPaths, cur.SecretPaths, slices.Equal[[]string])
}

func fieldEqual(a, b Field) bool {
	return a.Key == b.Key && a.Label == b.Label && a.Description == b.Description &&
		a.Kind == b.Kind && a.Required == b.Required && a.Role == b.Role &&
		slices.Equal(a.EnumOptions, b.EnumOptions)
}

// buildViewLocked assembles the next published view. The namespace array is
// copied so later element edits stay private; the payload slices inside are
// shared, since the store only ever replaces them.
func (s *Store) buildViewLocked() ViewState {
	clientReady := s.client != nil
	st := ViewState{
		PreferencePending:  s.preferencePendingLocked(),
		PreferenceErrors:   maps.Clone(s.preferenceErrors),
		PermissionDefaults: s.permissionDefaults.clone(),
		ClientReady:        clientReady,
		HasDocument:        s.hasDocument,
		Described:          clientReady && s.describedClient == s.client,
		Loading:            s.loading,
		Saving:             s.saving || s.themeWriteInFlightLocked(s.selectedNS),
		Writable:           s.writable,
		Err:                s.errTextLocked(),
		ErrDetail:          writeRefusalDetail(s.err),
		LastAction:         s.lastAction,
		SelectedNS:         s.selectedNS,
		Drafts:             map[string]string{},
		Resets:             copyFlags(s.resets[s.selectedNS]),
		Dirty:              len(s.dirty[s.selectedNS]) > 0,
	}
	st.Namespaces = slices.Clone(s.namespaces)
	for i := range st.Namespaces {
		st.Namespaces[i].DraftRevision, st.Namespaces[i].HasDraftRevision = s.draftRevisions[st.Namespaces[i].NS]
	}
	if s.selectedNS != "" {
		maps.Copy(st.Drafts, s.drafts[s.selectedNS])
	}
	return st
}

// SelectedNamespace returns the active namespace tab without cloning all namespaces.
func (s *Store) SelectedNamespace() string {
	s.mu.Lock()
	defer s.unlockPublish()
	return s.selectedNS
}

// Select chooses the active namespace tab.
func (s *Store) Select(ns string) {
	s.mu.Lock()
	if s.selectedNS == ns {
		s.unlockPublish()
		return
	}
	s.selectedNS = ns
	s.ensureDraftsLocked(ns)
	s.unlockPublish()
	s.fire()
}

// SetDraft updates one field draft for the selected namespace.
func (s *Store) SetDraft(key, value string) {
	s.mu.Lock()
	defer s.unlockPublish()
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
	s.refreshDescribe(ctx, false)
}

func (s *Store) refreshDescribe(ctx context.Context, background bool) {
	s.mu.Lock()
	client := s.client
	generation := s.generation
	if client == nil {
		s.unlockPublish()
		return
	}
	alreadyDescribed := s.describedClient == client
	if !background || !alreadyDescribed {
		s.loading = true
		s.err = nil
	}
	s.describeRequest++
	req := s.describeRequest
	if !alreadyDescribed {
		s.describedClient = nil
	}
	s.unlockPublish()
	s.fire()

	go func() {
		var out api.SettingsDescribeValue
		err := client.Call(ctx, "settings/describe", nil, &out)
		defaults := PermissionDefaults{}
		if err == nil {
			defaults = loadPermissionDefaults(ctx, client, out)
		}

		s.mu.Lock()
		if s.generation != generation || s.describeRequest != req {
			s.unlockPublish()
			return
		}
		s.loading = false
		if err != nil {
			if !background || s.describedClient != client {
				s.describedClient = nil
				s.err = err
			}
			s.unlockPublish()
			s.fire()
			return
		}
		s.hasDocument = out.HasDocument
		s.writable = out.Writable
		s.permissionDefaults = defaults
		sameHost := s.describedClient == client
		nsList := make([]Namespace, 0, len(out.Namespaces))
		for _, item := range out.Namespaces {
			fields, fErr := BuildFields(item.Schema)
			if fErr != nil {
				if !background || !sameHost {
					s.describedClient = nil
					s.err = fErr
				}
				s.unlockPublish()
				s.fire()
				return
			}
			nextNS := Namespace{
				NS:          item.Ns,
				Revision:    item.Revision,
				Fields:      fields,
				Value:       append(json.RawMessage(nil), item.Value...),
				Schema:      append(json.RawMessage(nil), item.Schema...),
				Base:        append(json.RawMessage(nil), item.Base...),
				User:        append(json.RawMessage(nil), item.User...),
				HasSecrets:  hasSetSecrets(item.Secrets),
				SecretPaths: configuredSecretPaths(item.Secrets),
			}
			// Never let an in-flight describe response regress a namespace that
			// already landed a higher revision on the same Host.
			if sameHost {
				if cur := s.findNamespaceLocked(item.Ns); cur != nil && cur.Revision > nextNS.Revision {
					nextNS = *cur
				}
			}
			nsList = append(nsList, nextNS)
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
		s.unlockPublish()
		s.fire()
	}()
}

// SaveReplace posts settings/replace for the selected namespace with a full section map.
// Empty section fails closed with Err/LastAction (no RPC).
func (s *Store) SaveReplace(ctx context.Context, section map[string]json.RawMessage) {
	s.mu.Lock()
	name := s.selectedNS
	s.unlockPublish()
	s.saveReplace(ctx, name, section)
}
func (s *Store) saveReplace(ctx context.Context, name string, section map[string]json.RawMessage) {
	s.saveReplacePrepared(ctx, name, section, nil)
}
func (s *Store) saveReplacePrepared(ctx context.Context, name string, section map[string]json.RawMessage, receipt *replaceDraftReceipt) {
	s.mu.Lock()
	if err := s.replaceReceiptErrorLocked(name, receipt); err != nil {
		s.err, s.lastAction = err, err.Error()
		s.unlockPublish()
		s.fire()
		return
	}
	client := s.client
	generation := s.generation
	ns := s.findNamespaceLocked(name)
	if s.saving || s.themeWriteInFlightLocked(name) || client == nil || ns == nil {
		s.unlockPublish()
		return
	}
	if ns.HasSecrets {
		s.err = fmt.Errorf("This section contains hidden secrets. Use Save to preserve them; full replacement is unavailable")
		s.lastAction = s.err.Error()
		s.unlockPublish()
		s.fire()
		return
	}
	if name == CustomInstructionsNamespace {
		var text string
		_ = json.Unmarshal(section[CustomInstructionsField], &text)
		if err := validateCustomInstructions(name, map[string]string{CustomInstructionsField: text}); err != nil {
			s.err = err
			s.unlockPublish()
			s.fire()
			return
		}
	}
	if len(section) == 0 {
		s.err = fmt.Errorf("settings replace rejected: empty section")
		s.lastAction = s.err.Error()
		s.unlockPublish()
		s.fire()
		return
	}
	if !s.writable {
		s.lastAction = "settings read-only"
		s.unlockPublish()
		s.fire()
		return
	}
	submitted := copyDrafts(s.drafts[ns.NS])
	submittedResets := copyFlags(s.resets[ns.NS])
	rev := s.draftRevisionLocked(*ns)
	nsName := ns.NS
	s.saving = true
	s.err = nil
	s.unlockPublish()
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
			s.unlockPublish()
			return
		}
		s.saving = false
		if err != nil {
			s.err = err
			s.lastAction = err.Error()
			s.unlockPublish()
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
		s.unlockPublish()
		s.fire()
	}()
}

// SaveReplaceFromDrafts replaces the stored user layer, never the resolved
// effective layer. Redacted sections cannot safely be reconstructed wholesale.
func (s *Store) SaveReplaceFromDrafts(ctx context.Context) {
	s.mu.Lock()
	ns := s.findNamespaceLocked(s.selectedNS)
	if s.client == nil || ns == nil || s.saving || s.themeWriteInFlightLocked(s.selectedNS) || s.loading {
		s.unlockPublish()
		return
	}
	if !s.writable {
		s.lastAction = "settings read-only"
		s.unlockPublish()
		s.fire()
		return
	}
	section, err := replacementFromDrafts(*ns, s.drafts[ns.NS], s.dirty[ns.NS], s.resets[ns.NS])
	name := ns.NS
	s.unlockPublish()
	if err != nil {
		s.mu.Lock()
		s.err = err
		s.lastAction = err.Error()
		s.unlockPublish()
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
	s.unlockPublish()
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
	if s.saving || s.themeWriteInFlightLocked(name) || client == nil || ns == nil || len(ops) == 0 || (owner != nil && (client != owner || generation != ownerGeneration)) {
		s.unlockPublish()
		return
	}
	if !s.writable {
		s.lastAction = "settings read-only"
		s.unlockPublish()
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
	s.unlockPublish()
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
			s.unlockPublish()
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
			s.unlockPublish()
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
		s.unlockPublish()
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
	if s.saving || s.themeWriteInFlightLocked(s.selectedNS) || s.loading || client == nil || ns == nil {
		s.unlockPublish()
		return
	}
	if !s.writable {
		s.lastAction = "settings read-only"
		s.unlockPublish()
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
			s.unlockPublish()
			s.fire()
			return
		}
	}
	patch, ops, submitted, err := draftWrite(*ns, drafts, s.dirty[ns.NS], resets)
	if err != nil {
		s.err = err
		s.lastAction = err.Error()
		s.unlockPublish()
		s.fire()
		return
	}
	if len(patch) == 0 && len(ops) == 0 {
		s.acknowledgeDraftsLocked(*ns, submitted, resets)
		s.lastAction = "No settings changes"
		s.err = nil
		s.unlockPublish()
		s.fire()
		return
	}
	s.saving = true
	s.err = nil
	s.unlockPublish()
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
			s.unlockPublish()
			return
		}
		s.saving = false
		if err != nil {
			s.err = err
			s.lastAction = err.Error()
			s.unlockPublish()
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
		s.unlockPublish()
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
	defer s.unlockPublish()
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

type patchBoolMutation struct {
	ctx context.Context
	key string
	raw json.RawMessage
}

type patchBoolQueue struct {
	client     *dsh.Client
	generation uint64
	pending    []patchBoolMutation
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
	s.mu.Lock()
	defer s.unlockPublish()
	for _, c := range candidates {
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
		if has {
			s.enqueuePatchBoolLocked(ctx, c.ns, c.key, show)
			return
		}
	}
}

// PatchBool posts settings/update for one bool key in a namespace.
func (s *Store) PatchBool(ctx context.Context, nsName, key string, value bool) {
	s.mu.Lock()
	defer s.unlockPublish()
	s.enqueuePatchBoolLocked(ctx, nsName, key, value)
}

func (s *Store) enqueuePatchBoolLocked(ctx context.Context, nsName, key string, value bool) {
	client := s.client
	generation := s.generation
	ns := s.findNamespaceLocked(nsName)
	if client == nil || s.describedClient != client || ns == nil || !s.writable {
		return
	}
	raw, err := PatchValue(FieldBool, fmt.Sprintf("%v", value))
	if err != nil {
		return
	}
	if s.patchBoolQueues == nil {
		s.patchBoolQueues = map[string]*patchBoolQueue{}
	}
	q := s.patchBoolQueues[nsName]
	start := q == nil || q.client != client || q.generation != generation
	if start {
		q = &patchBoolQueue{client: client, generation: generation}
		s.patchBoolQueues[nsName] = q
	}
	q.pending = append(q.pending, patchBoolMutation{ctx: ctx, key: key, raw: raw})
	if start {
		go s.flushPatchBoolQueue(nsName, q)
	}
}

func (s *Store) flushPatchBoolQueue(nsName string, q *patchBoolQueue) {
	for {
		s.mu.Lock()
		if s.client != q.client || s.generation != q.generation || s.describedClient != q.client || s.patchBoolQueues[nsName] != q || len(q.pending) == 0 {
			s.unlockPublish()
			return
		}
		item := q.pending[0]
		ns := s.findNamespaceLocked(nsName)
		if ns == nil || !s.writable {
			delete(s.patchBoolQueues, nsName)
			s.unlockPublish()
			return
		}
		rev := ns.Revision
		s.unlockPublish()

		args := map[string]any{
			"ns":               nsName,
			"patch":            map[string]json.RawMessage{item.key: item.raw},
			"expectedRevision": rev,
		}
		var out api.SettingsNamespaceView
		err := q.client.Call(item.ctx, "settings/update", args, &out)

		s.mu.Lock()
		if s.client != q.client || s.generation != q.generation || s.describedClient != q.client || s.patchBoolQueues[nsName] != q {
			s.unlockPublish()
			return
		}
		if len(q.pending) > 0 {
			q.pending = q.pending[1:]
		}
		if err != nil {
			s.err = err
			s.lastAction = "viewTabs sync: " + err.Error()
		} else {
			s.err = nil
			s.lastAction = "viewTabs synced"
			for i := range s.namespaces {
				if s.namespaces[i].NS == nsName {
					if s.namespaces[i].Revision <= out.Revision {
						s.namespaces[i].Revision = out.Revision
						s.namespaces[i].Value = append(json.RawMessage(nil), out.Value...)
						s.namespaces[i].Base = append(json.RawMessage(nil), out.Base...)
						s.namespaces[i].User = append(json.RawMessage(nil), out.User...)
						s.namespaces[i].HasSecrets = hasSetSecrets(out.Secrets)
						s.namespaces[i].SecretPaths = configuredSecretPaths(out.Secrets)
						s.acknowledgeDraftsLocked(s.namespaces[i], map[string]string{})
					}
					break
				}
			}
		}
		s.viewStale = true
		done := len(q.pending) == 0
		if done {
			delete(s.patchBoolQueues, nsName)
		}
		s.unlockPublish()
		s.fire()
		if done {
			return
		}
	}
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
	s.unlockPublish()
	s.RefreshDescribe(ctx)
}
