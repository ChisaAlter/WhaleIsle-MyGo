package office

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/state"
	"whaleisle/internal/store/storelocale"
)

type ViewState struct {
	Scope                                                 uint64
	Open, Available, Loaded, Fresh, Busy, Dirty, Conflict bool
	Host, Error, Notice, ReviewAction, UpdatedAt          string
	Status                                                Status
	Draft                                                 Draft
}

// Store keeps the mutable working view under mu and hands readers a
// published, immutable ViewState version. The public Scope is minted by a
// scope Lease — Bind and Hide renew it — and each async operation holds an
// op Lease whose invalidation drops stale completions. Status objects are
// fresh decodes assigned wholesale, so the published view can share them.
type Store struct {
	mu        sync.Mutex
	client    *pluginrpc.Client
	bindEpoch uint64
	notify    func()
	scope     state.LeaseOwner
	scopeL    state.Lease
	op        state.LeaseOwner
	opL       state.Lease
	opCancel  context.CancelFunc
	view      ViewState
	pub       state.Published[ViewState]
	base      [32]byte
	reviewed  [32]byte
}

// Bind is called by the IM store on every Host/account attachment transition.
// An account cookie is retained only by the original authenticated RPC client.
func (s *Store) Bind(client *pluginrpc.Client, epoch uint64, notify func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notify = notify
	if client == s.client && epoch == s.bindEpoch {
		return
	}
	s.op.Retire()
	s.scopeL = s.scope.Renew(context.Background())
	s.view = ViewState{Scope: s.scopeL.Generation(), Draft: defaultDraft(), Available: client != nil}
	s.base = [32]byte{}
	s.reviewed = [32]byte{}
	s.client = client
	s.bindEpoch = epoch
	if client != nil {
		if u, e := url.Parse(client.BaseURL); e == nil && u.Host != "" {
			s.view.Host = u.Scheme + "://" + u.Host
		}
	}
	s.publishLocked()
}

// Snapshot returns the current published version. The same *Version is
// shared by every reader until a write publishes the next revision; Value
// is immutable. A zero-value Store publishes its empty view lazily — the
// store is embedded inside the IM store without a constructor.
func (s *Store) Snapshot() *state.Version[ViewState] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	if s.pub.Load() == nil {
		s.publishLocked()
	}
	s.mu.Unlock()
	return s.pub.Load()
}

// Changed closes when the next version is published.
func (s *Store) Changed() <-chan struct{} { return s.pub.Changed() }

func (s *Store) publishLocked() { s.pub.Publish(s.view) }
func (s *Store) fire() {
	s.mu.Lock()
	f := s.notify
	s.mu.Unlock()
	if f != nil {
		f()
	}
}
func (s *Store) Show(ctx context.Context) {
	s.mu.Lock()
	s.view.Open = true
	s.publishLocked()
	s.mu.Unlock()
	s.Refresh(ctx)
}
func (s *Store) Hide() {
	s.mu.Lock()
	s.op.Retire()
	s.scopeL = s.scope.Renew(context.Background())
	s.view.Scope = s.scopeL.Generation()
	s.view.Open = false
	s.view.Busy = false
	s.view.Fresh = false
	s.view.ReviewAction = ""
	s.view.Draft.DeviceToken = ""
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Edit(scope uint64, d Draft) bool {
	s.mu.Lock()
	if scope != s.view.Scope || s.view.Busy || s.view.ReviewAction != "" || !s.view.Open {
		s.mu.Unlock()
		return false
	}
	s.view.Draft = d
	s.view.Dirty = d != draftFor(s.view.Status)
	s.view.Notice = ""
	s.view.Error = ""
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	return true
}
func (s *Store) Discard(scope uint64) {
	s.mu.Lock()
	if scope != s.view.Scope || s.view.Busy || !s.view.Loaded {
		s.mu.Unlock()
		return
	}
	s.view.Draft = draftFor(s.view.Status)
	s.base = fingerprint(s.view.Status)
	s.view.Conflict = false
	s.view.Dirty = false
	s.view.ReviewAction = ""
	s.view.Error = ""
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Dismiss(scope uint64) {
	s.mu.Lock()
	if scope == s.view.Scope && !s.view.Busy {
		s.view.ReviewAction = ""
		s.publishLocked()
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) writableLocked(scope uint64) bool {
	return scope == s.view.Scope && s.client != nil && s.view.Open && s.view.Loaded && s.view.Fresh && !s.view.Busy && !s.view.Conflict
}
func (s *Store) Review(scope uint64, action string) bool {
	s.mu.Lock()
	if !s.writableLocked(scope) {
		s.mu.Unlock()
		return false
	}
	valid := true
	switch action {
	case "configure":
		if _, e := configurePayload(s.view.Draft, s.view.Status); e != nil {
			s.view.Error = e.Error()
			valid = false
		}
	case "test", "reconnect", "remove":
		if !s.view.Status.Configured || s.view.Dirty {
			s.view.Error = storelocale.T(storelocale.KeyStoreOfficeSaveOrDiscardTheDraftBeforeActingOnT)
			valid = false
		}
	default:
		valid = false
	}
	if valid {
		s.view.ReviewAction = action
		s.reviewed = fingerprint(s.view.Status)
		s.view.Error = ""
		s.view.Notice = ""
	}
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	return valid
}
func (s *Store) adoptLocked(next Status) {
	revision := fingerprint(next)
	if s.view.Dirty && s.view.Draft == draftFor(next) {
		s.view.Dirty = false
		s.base = revision
	}
	if s.view.Loaded && (s.view.Dirty || s.view.ReviewAction != "") && revision != s.base {
		s.view.Conflict = true
		s.view.ReviewAction = ""
		s.view.Draft.DeviceToken = ""
	}
	s.view.Status = next
	s.view.Loaded = true
	s.view.Fresh = true
	s.view.UpdatedAt = time.Now().Format(time.RFC3339)
	if !s.view.Dirty && !s.view.Conflict && s.view.ReviewAction == "" {
		s.view.Draft = draftFor(next)
		s.base = revision
	}
}
func (s *Store) beginLocked(ctx context.Context) state.Lease {
	parent, cancel := context.WithTimeout(ctx, 30*time.Second)
	s.opL = s.op.Renew(parent)
	s.opCancel = cancel
	s.view.Busy = true
	s.view.Error = ""
	s.view.Notice = ""
	return s.opL
}
func (s *Store) finishOpLocked() {
	if s.opCancel != nil {
		s.opCancel()
		s.opCancel = nil
	}
}
func (s *Store) currentLocked(scope uint64, op state.Lease, c *pluginrpc.Client) bool {
	return s.view.Scope == scope && s.opL == op && op.Valid() && s.client == c && s.view.Open
}
func readStatus(ctx context.Context, c *pluginrpc.Client) (Status, error) {
	var raw json.RawMessage
	if e := c.Call(ctx, "office", "connection.status", map[string]any{}, &raw); e != nil {
		return Status{}, e
	}
	v, _, e := parseStatus(raw)
	return v, e
}
func (s *Store) Refresh(ctx context.Context) {
	s.mu.Lock()
	if s.view.Busy || !s.view.Open {
		s.mu.Unlock()
		return
	}
	if s.client == nil {
		s.view.Error = storelocale.T(storelocale.KeyStoreOfficeTheHostIsNotConnectedRetryOnceTheHos)
		s.view.Fresh = false
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	c, scope := s.client, s.view.Scope
	op := s.beginLocked(ctx)
	ctx = op.Context()
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		v, e := readStatus(ctx, c)
		s.mu.Lock()
		if !s.currentLocked(scope, op, c) {
			s.mu.Unlock()
			return
		}
		s.finishOpLocked()
		s.view.Busy = false
		if e != nil {
			s.view.Fresh = false
			s.view.Error = officeCallError(e, storelocale.T(storelocale.KeyStoreOfficeCouldNotReadAiOfficeStatusRetryOrOpe), "")
		} else {
			s.adoptLocked(v)
		}
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
	}()
}

// Confirm performs one fresh read followed by at most one authorized mutation.
// The current Host has no CAS endpoint; preflight narrows, not eliminates, races.
func (s *Store) Confirm(ctx context.Context, scope uint64) bool {
	s.mu.Lock()
	if !s.writableLocked(scope) || s.view.ReviewAction == "" {
		s.mu.Unlock()
		return false
	}
	action := s.view.ReviewAction
	payload := map[string]any{}
	var e error
	if action == "configure" {
		payload, e = configurePayload(s.view.Draft, s.view.Status)
		if e != nil {
			s.view.Error = e.Error()
			s.publishLocked()
			s.mu.Unlock()
			s.fire()
			return false
		}
	} else if action == "remove" {
		payload["confirm"] = true
	}
	c, expected := s.client, s.reviewed
	op := s.beginLocked(ctx)
	ctx = op.Context()
	s.view.Draft.DeviceToken = ""
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		fresh, err := readStatus(ctx, c)
		s.mu.Lock()
		if !s.currentLocked(scope, op, c) {
			s.mu.Unlock()
			return
		}
		if err != nil || fingerprint(fresh) != expected {
			s.finishOpLocked()
			s.view.Busy = false
			s.view.ReviewAction = ""
			if err != nil {
				s.view.Fresh = false
				s.view.Error = officeCallError(err, storelocale.T(storelocale.KeyStoreOfficeTheLatestConfigurationCouldNotBeConfir), payloadToken(payload))
			} else {
				s.adoptLocked(fresh)
				s.view.Conflict = true
				s.view.Error = storelocale.T(storelocale.KeyStoreOfficeTheOfficeConfigurationWasChangedByAnot)
			}
			s.publishLocked()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.mu.Unlock()
		var raw json.RawMessage
		err = c.Call(ctx, "office", "connector."+action, payload, &raw)
		var next Status
		var tested bool
		if err == nil {
			next, tested, err = parseStatus(raw)
		}
		valid := err == nil
		if valid {
			switch action {
			case "test":
				valid = tested
			case "remove":
				valid = !next.Configured
			case "configure":
				valid = matchesConfiguration(next, payload)
			case "reconnect":
				valid = next.Configured && fingerprint(next) == expected
			}
		}
		s.mu.Lock()
		if !s.currentLocked(scope, op, c) {
			s.mu.Unlock()
			return
		}
		s.finishOpLocked()
		s.view.Busy = false
		s.view.ReviewAction = ""
		if !valid {
			s.view.Fresh = false
			s.view.Error = officeCallError(err, storelocale.T(storelocale.KeyStoreOfficeTheOutcomeIsUnconfirmedRefreshToVerify), payloadToken(payload))
			var rejected *pluginrpc.RPCError
			if errors.As(err, &rejected) && (action == "test" || rejected.Code == "bad-request") {
				// No mutation was accepted. The successful preflight remains
				// usable and the editable public draft is retained for correction.
				s.adoptLocked(fresh)
			}
		} else {
			if action == "configure" || action == "remove" {
				s.view.Dirty = false
				s.view.Conflict = false
			}
			s.adoptLocked(next)
			switch action {
			case "configure":
				if next.Connected {
					s.view.Notice = storelocale.T(storelocale.KeyStoreOfficeConfigurationSavedOfficeIsConnected)
				} else {
					s.view.Notice = storelocale.T(storelocale.KeyStoreOfficeConfigurationSavedOfficeIsNotConnected2)
				}
			case "reconnect":
				if next.Connected {
					s.view.Notice = storelocale.T(storelocale.KeyStoreOfficeOfficeReconnected)
				} else {
					s.view.Notice = storelocale.T(storelocale.KeyStoreOfficeReconnectionStartedOfficeIsNotConnected)
				}
			case "test":
				s.view.Notice = storelocale.T(storelocale.KeyStoreOfficeConnectionTestPassedThisDoesNotMeanA)
			case "remove":
				s.view.Notice = storelocale.T(storelocale.KeyStoreOfficeConnectionConfigurationRemoved)
			}
		}
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
	}()
	return true
}

func payloadToken(payload map[string]any) string {
	value, _ := payload["deviceToken"].(string)
	return value
}

func officeCallError(err error, fallback, token string) string {
	var rejected *pluginrpc.RPCError
	if !errors.As(err, &rejected) {
		return fallback
	}
	message := strings.TrimSpace(rejected.Message)
	if token != "" {
		message = strings.ReplaceAll(message, token, "[credential omitted]")
	}
	message = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, message)
	if len([]rune(message)) > 1600 {
		message = string([]rune(message)[:1600]) + "…"
	}
	code := strings.TrimSpace(rejected.Code)
	if code != "" && message != "" {
		return code + ": " + message
	}
	if message != "" {
		return message
	}
	if code != "" {
		return code
	}
	return fallback
}
func matchesConfiguration(s Status, p map[string]any) bool {
	return s.Configured && s.Config.BaseURL == p["baseUrl"] && s.Config.DeviceID == p["deviceId"] && s.Config.MaxConcurrency == p["maxConcurrency"] && s.Config.HeartbeatSeconds == p["heartbeatSeconds"] && reflect.DeepEqual(s.Config.Workspaces, p["workspaces"]) && reflect.DeepEqual(s.Config.InstructionPresets, p["instructionPresets"])
}
func ReviewSummary(v ViewState) string {
	target := v.Status.Config.BaseURL + " / " + v.Status.Config.DeviceID
	switch v.ReviewAction {
	case "configure":
		base, _ := normalizeBase(v.Draft.BaseURL)
		return storelocale.T(storelocale.KeyStoreOfficeAfterSavingToTheHostTheOldConnection) + base + " / " + strings.TrimSpace(v.Draft.DeviceID) + storelocale.T(storelocale.KeyStoreOfficeTheHostSendsTheDeviceIdProtocolVersio)
	case "test":
		return storelocale.T(storelocale.KeyStoreOfficeSendOneAuthenticatedHeartbeatProbeToTh) + target + storelocale.T(storelocale.KeyStoreOfficeTheProbeSendsTheDeviceIdMappingAliase)
	case "reconnect":
		return storelocale.T(storelocale.KeyStoreOfficeStopTheCurrentConnectionAndReconnectTo) + target + storelocale.T(storelocale.KeyStoreOfficeJobsAreReceivedAgainOnceTheConnection)
	case "remove":
		return storelocale.T(storelocale.KeyStoreOfficeStopTheConnectionAndRemoveTheConfigura) + target + storelocale.T(storelocale.KeyStoreOfficeJobsInProgressAreAffectedCompletedOper)
	}
	return ""
}
