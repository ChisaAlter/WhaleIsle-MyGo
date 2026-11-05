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
	"whaleisle/internal/store/storelocale"
)

type ViewState struct {
	Scope                                                 uint64
	Open, Available, Loaded, Fresh, Busy, Dirty, Conflict bool
	Host, Error, Notice, ReviewAction, UpdatedAt          string
	Status                                                Status
	Draft                                                 Draft
}
type Store struct {
	mu        sync.Mutex
	client    *pluginrpc.Client
	bindEpoch uint64
	notify    func()
	cancel    context.CancelFunc
	operation uint64
	state     ViewState
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
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.operation++
	scope := s.state.Scope + 1
	s.state = ViewState{Scope: scope, Draft: defaultDraft(), Available: client != nil}
	s.base = [32]byte{}
	s.reviewed = [32]byte{}
	s.client = client
	s.bindEpoch = epoch
	if client != nil {
		if u, e := url.Parse(client.BaseURL); e == nil && u.Host != "" {
			s.state.Host = u.Scheme + "://" + u.Host
		}
	}
}
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.state
	v.Status.Config.Workspaces = copyMap(v.Status.Config.Workspaces)
	v.Status.Config.InstructionPresets = copyMap(v.Status.Config.InstructionPresets)
	if v.Status.Health.Error != nil {
		e := *v.Status.Health.Error
		v.Status.Health.Error = &e
	}
	return v
}
func copyMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
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
	s.state.Open = true
	s.mu.Unlock()
	s.Refresh(ctx)
}
func (s *Store) Hide() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.operation++
	s.state.Scope++
	s.state.Open = false
	s.state.Busy = false
	s.state.Fresh = false
	s.state.ReviewAction = ""
	s.state.Draft.DeviceToken = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Edit(scope uint64, d Draft) bool {
	s.mu.Lock()
	if scope != s.state.Scope || s.state.Busy || s.state.ReviewAction != "" || !s.state.Open {
		s.mu.Unlock()
		return false
	}
	s.state.Draft = d
	s.state.Dirty = d != draftFor(s.state.Status)
	s.state.Notice = ""
	s.state.Error = ""
	s.mu.Unlock()
	s.fire()
	return true
}
func (s *Store) Discard(scope uint64) {
	s.mu.Lock()
	if scope != s.state.Scope || s.state.Busy || !s.state.Loaded {
		s.mu.Unlock()
		return
	}
	s.state.Draft = draftFor(s.state.Status)
	s.base = fingerprint(s.state.Status)
	s.state.Conflict = false
	s.state.Dirty = false
	s.state.ReviewAction = ""
	s.state.Error = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Dismiss(scope uint64) {
	s.mu.Lock()
	if scope == s.state.Scope && !s.state.Busy {
		s.state.ReviewAction = ""
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) writableLocked(scope uint64) bool {
	return scope == s.state.Scope && s.client != nil && s.state.Open && s.state.Loaded && s.state.Fresh && !s.state.Busy && !s.state.Conflict
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
		if _, e := configurePayload(s.state.Draft, s.state.Status); e != nil {
			s.state.Error = e.Error()
			valid = false
		}
	case "test", "reconnect", "remove":
		if !s.state.Status.Configured || s.state.Dirty {
			s.state.Error = storelocale.T(storelocale.KeyStoreOfficeSaveOrDiscardTheDraftBeforeActingOnT)
			valid = false
		}
	default:
		valid = false
	}
	if valid {
		s.state.ReviewAction = action
		s.reviewed = fingerprint(s.state.Status)
		s.state.Error = ""
		s.state.Notice = ""
	}
	s.mu.Unlock()
	s.fire()
	return valid
}
func (s *Store) adoptLocked(next Status) {
	revision := fingerprint(next)
	if s.state.Loaded && (s.state.Dirty || s.state.ReviewAction != "") && revision != s.base {
		s.state.Conflict = true
		s.state.ReviewAction = ""
		s.state.Draft.DeviceToken = ""
	}
	s.state.Status = next
	s.state.Loaded = true
	s.state.Fresh = true
	s.state.UpdatedAt = time.Now().Format(time.RFC3339)
	if !s.state.Dirty && !s.state.Conflict && s.state.ReviewAction == "" {
		s.state.Draft = draftFor(next)
		s.base = revision
	}
}
func (s *Store) beginLocked(ctx context.Context) (context.Context, uint64) {
	if s.cancel != nil {
		s.cancel()
	}
	ctx, s.cancel = context.WithTimeout(ctx, 30*time.Second)
	s.operation++
	s.state.Busy = true
	s.state.Error = ""
	s.state.Notice = ""
	return ctx, s.operation
}
func (s *Store) currentLocked(scope, op uint64, c *pluginrpc.Client) bool {
	return s.state.Scope == scope && s.operation == op && s.client == c && s.state.Open
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
	if s.state.Busy || !s.state.Open {
		s.mu.Unlock()
		return
	}
	if s.client == nil {
		s.state.Error = storelocale.T(storelocale.KeyStoreOfficeTheHostIsNotConnectedRetryOnceTheHos)
		s.state.Fresh = false
		s.mu.Unlock()
		s.fire()
		return
	}
	c, scope := s.client, s.state.Scope
	ctx, op := s.beginLocked(ctx)
	s.mu.Unlock()
	s.fire()
	go func() {
		v, e := readStatus(ctx, c)
		s.mu.Lock()
		if !s.currentLocked(scope, op, c) {
			s.mu.Unlock()
			return
		}
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
		s.state.Busy = false
		if e != nil {
			s.state.Fresh = false
			s.state.Error = officeCallError(e, storelocale.T(storelocale.KeyStoreOfficeCouldNotReadAiOfficeStatusRetryOrOpe), "")
		} else {
			s.adoptLocked(v)
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// Confirm performs one fresh read followed by at most one authorized mutation.
// The current Host has no CAS endpoint; preflight narrows, not eliminates, races.
func (s *Store) Confirm(ctx context.Context, scope uint64) bool {
	s.mu.Lock()
	if !s.writableLocked(scope) || s.state.ReviewAction == "" {
		s.mu.Unlock()
		return false
	}
	action := s.state.ReviewAction
	payload := map[string]any{}
	var e error
	if action == "configure" {
		payload, e = configurePayload(s.state.Draft, s.state.Status)
		if e != nil {
			s.state.Error = e.Error()
			s.mu.Unlock()
			s.fire()
			return false
		}
	} else if action == "remove" {
		payload["confirm"] = true
	}
	c, expected := s.client, s.reviewed
	ctx, op := s.beginLocked(ctx)
	s.state.Draft.DeviceToken = ""
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
			if s.cancel != nil {
				s.cancel()
				s.cancel = nil
			}
			s.state.Busy = false
			s.state.ReviewAction = ""
			if err != nil {
				s.state.Fresh = false
				s.state.Error = officeCallError(err, storelocale.T(storelocale.KeyStoreOfficeTheLatestConfigurationCouldNotBeConfir), payloadToken(payload))
			} else {
				s.adoptLocked(fresh)
				s.state.Conflict = true
				s.state.Error = storelocale.T(storelocale.KeyStoreOfficeTheOfficeConfigurationWasChangedByAnot)
			}
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
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
		s.state.Busy = false
		s.state.ReviewAction = ""
		if !valid {
			s.state.Fresh = false
			s.state.Error = officeCallError(err, storelocale.T(storelocale.KeyStoreOfficeTheOutcomeIsUnconfirmedRefreshToVerify), payloadToken(payload))
			var rejected *pluginrpc.RPCError
			if errors.As(err, &rejected) {
				// No mutation was accepted. The successful preflight remains
				// usable and the editable public draft is retained for correction.
				s.adoptLocked(fresh)
			}
		} else {
			if action == "configure" || action == "remove" {
				s.state.Dirty = false
				s.state.Conflict = false
			}
			s.adoptLocked(next)
			switch action {
			case "configure":
				if next.Connected {
					s.state.Notice = storelocale.T(storelocale.KeyStoreOfficeConfigurationSavedOfficeIsConnected)
				} else {
					s.state.Notice = storelocale.T(storelocale.KeyStoreOfficeConfigurationSavedOfficeIsNotConnected2)
				}
			case "reconnect":
				if next.Connected {
					s.state.Notice = storelocale.T(storelocale.KeyStoreOfficeOfficeReconnected)
				} else {
					s.state.Notice = storelocale.T(storelocale.KeyStoreOfficeReconnectionStartedOfficeIsNotConnected)
				}
			case "test":
				s.state.Notice = storelocale.T(storelocale.KeyStoreOfficeConnectionTestPassedThisDoesNotMeanA)
			case "remove":
				s.state.Notice = storelocale.T(storelocale.KeyStoreOfficeConnectionConfigurationRemoved)
			}
		}
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
