package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

const DesktopOnboardingNamespace = "ui-settings-account"

// OnboardingAccount contains only the existing display-safe account facts.
// HasAPIKey is metadata; the introduction never reads a credential literal.
type OnboardingAccount struct {
	Enabled                   bool
	Status                    string
	Loading                   bool
	Error                     string
	HasAPIKey                 *bool
	CreditKnown, CreditFunded bool
}

type OnboardingProgress struct {
	Version        int    `json:"version"`
	Step           string `json:"step"`
	Purpose        string `json:"purpose"`
	Process        string `json:"process"`
	Completion     string `json:"completion"`
	Usage          string `json:"usage"`
	DeveloperTools bool   `json:"developerTools"`
}

type OnboardingView struct {
	Owner                                               uint64
	Visible, Saving, Loading, CreditKnown, CreditFunded bool
	CurrentCreditFunded                                 bool
	Progress                                            OnboardingProgress
	Error                                               string
}

type DesktopOnboarding struct {
	mu                sync.Mutex
	store             *Store
	client            *dsh.Client
	generation        uint64
	account           OnboardingAccount
	view              OnboardingView
	failed            *OnboardingProgress
	failedPreferences bool
	creditCaptured    bool
	draft             *OnboardingProgress
	updates           []OnboardingProgress
	updating          bool
	completion        string
	operation         uint64
	credentialEpoch   uint64
	hasAPIKey         *bool
	checkingKeys      bool
}

func NewDesktopOnboarding(s *Store) *DesktopOnboarding { return &DesktopOnboarding{store: s} }

func freshOnboarding() OnboardingProgress {
	return OnboardingProgress{Version: 1, Step: "welcome", Usage: "compact"}
}

// Sync follows accepted progress and safe account facts. A Host replacement or
// account sign-out retires the old introduction's pending UI settlements.
func (o *DesktopOnboarding) Sync(account OnboardingAccount) {
	s := o.store
	s.mu.Lock()
	client, generation, credentialEpoch := s.client, s.generation, s.onboardingCredentialEpoch
	described := client != nil && s.describedClient == client
	ns := s.findNamespaceLocked(DesktopOnboardingNamespace)
	progress := freshOnboarding()
	if ns != nil {
		_ = json.Unmarshal(ns.Value, &progress)
	}
	available, writable, loading, storeError := ns != nil, s.writable, s.loading, ""
	if s.err != nil {
		storeError = s.err.Error()
	}
	chatReady := s.findNamespaceLocked("ui-chat") != nil
	s.mu.Unlock()
	o.mu.Lock()
	if client != o.client || generation != o.generation {
		o.client, o.generation = client, generation
		o.failed, o.draft = nil, nil
		o.operation++
		o.updating, o.updates, o.completion = false, nil, ""
		o.creditCaptured = false
		o.view = OnboardingView{Owner: generation}
		o.hasAPIKey, o.checkingKeys = nil, false
	}
	if o.account.Status != account.Status {
		o.creditCaptured = false
		o.operation++
		o.view.Saving = false
		o.updating = false
		o.updates = nil
		o.completion = ""
		o.draft, o.failed = nil, nil
	}
	o.account = account
	if o.credentialEpoch != credentialEpoch {
		o.credentialEpoch = credentialEpoch
		o.hasAPIKey, o.checkingKeys = nil, false
		if o.failed != nil && o.failed.Completion == "api-key" {
			o.failed = nil
			o.view.Error = ""
		}
	}
	if account.HasAPIKey == nil {
		account.HasAPIKey = o.hasAPIKey
	}
	if o.draft != nil {
		progress = *o.draft
	}
	o.view.Progress = progress
	o.view.Loading = loading || !described && storeError == "" || account.Status == "" && account.Loading
	o.view.CreditKnown = account.CreditKnown
	o.view.CurrentCreditFunded = account.CreditFunded
	if !o.creditCaptured && account.Status == "credential-stored" && progress.Step == "credit" {
		o.view.CreditFunded = account.CreditFunded
		o.creditCaptured = true
	}
	o.view.Visible = account.Enabled && client != nil && account.Status == "credential-stored" && (!described || available && writable) && progress.Step != "done"
	if o.failed == nil {
		o.view.Error = ""
		if storeError != "" && (!described || loading) {
			o.view.Error = storeError
		}
	}
	auto := account.Enabled && client != nil && available && writable && chatReady && !loading && !account.Loading && account.Status == "signed-out" && account.HasAPIKey != nil && *account.HasAPIKey && progress.Step != "done" && !o.view.Saving && o.failed == nil
	checkKeys := account.Enabled && client != nil && available && chatReady && !loading && account.Status == "signed-out" && account.HasAPIKey == nil && !o.checkingKeys
	if checkKeys {
		o.checkingKeys = true
	}
	if auto {
		progress.Step, progress.Process, progress.Usage, progress.Completion, progress.DeveloperTools = "done", "standard", "detailed", "api-key", true
	}
	o.mu.Unlock()
	if checkKeys {
		go func() {
			present := s.readOnboardingAPIKeyPresence(client, generation)
			o.mu.Lock()
			if o.client != client || o.generation != generation || o.credentialEpoch != credentialEpoch {
				o.mu.Unlock()
				return
			}
			o.hasAPIKey, o.checkingKeys = &present, false
			o.mu.Unlock()
			s.fire()
		}()
	}
	if auto {
		o.start(progress, true, false)
	}
}

func (o *DesktopOnboarding) Snapshot() OnboardingView {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.view
}

func (o *DesktopOnboarding) Update(change map[string]string) {
	o.mu.Lock()
	if o.view.Saving || !o.view.Visible || o.account.Status != "credential-stored" {
		o.mu.Unlock()
		return
	}
	p := o.view.Progress
	for key, value := range change {
		switch key {
		case "step":
			if value == "welcome" || value == "credit" || value == "purpose" || value == "process" {
				p.Step = value
			}
		case "purpose":
			if value == "" || value == "office" || value == "development" || value == "both" {
				p.Purpose = value
			}
		case "process":
			if value == "compact" || value == "standard" || value == "detailed" {
				p.Process = value
			}
		}
	}
	o.draft = &p
	o.view.Progress, o.view.Error = p, ""
	o.updates = append(o.updates, p)
	start := !o.updating
	o.updating = true
	client, generation, operation, status := o.client, o.generation, o.operation, o.account.Status
	o.mu.Unlock()
	o.store.fire()
	if start {
		go o.writeUpdates(client, generation, operation, status)
	}
}

// Step selections preview immediately while the Host mutations serialize.
// Completion joins this queue and never writes the done marker ahead of it.
func (o *DesktopOnboarding) writeUpdates(client *dsh.Client, generation, operation uint64, status string) {
	for {
		o.mu.Lock()
		if o.client != client || o.generation != generation || o.operation != operation || o.account.Status != status {
			o.mu.Unlock()
			return
		}
		if len(o.updates) == 0 {
			o.mu.Unlock()
			return
		}
		next := o.updates[0]
		o.updates = o.updates[1:]
		o.mu.Unlock()
		err := o.store.writeOnboardingFields(context.Background(), client, generation, DesktopOnboardingNamespace, onboardingFields(next))
		o.mu.Lock()
		if o.client != client || o.generation != generation || o.operation != operation || o.account.Status != status {
			o.mu.Unlock()
			return
		}
		if err != nil {
			o.failed = &next
			o.failedPreferences = false
			o.view.Error = err.Error()
		} else {
			o.failed = nil
			o.view.Error = ""
		}
		if len(o.updates) > 0 {
			o.mu.Unlock()
			o.store.fire()
			continue
		}
		o.updating, o.draft = false, nil
		reason := o.completion
		o.completion, o.view.Saving = "", false
		if err == nil {
			o.view.Progress = next
		}
		o.mu.Unlock()
		o.store.fire()
		if reason != "" && err == nil {
			o.Complete(reason)
		}
		return
	}
}

func (o *DesktopOnboarding) Complete(reason string) {
	o.mu.Lock()
	p := o.view.Progress
	if o.updating && !o.view.Saving && (reason == "completed" || reason == "skipped") {
		o.completion, o.view.Saving = reason, true
		o.mu.Unlock()
		o.store.fire()
		return
	}
	if !o.view.Visible || o.view.Saving || o.account.Status != "credential-stored" || p.Step == "welcome" || p.Step == "done" || reason != "completed" && reason != "skipped" {
		o.mu.Unlock()
		return
	}
	office := reason == "completed" && p.Purpose == "office"
	development := reason == "completed" && !office
	if development && p.Process == "" {
		o.mu.Unlock()
		return
	}
	p.Step, p.Completion = "done", reason
	if office {
		p.Process = "compact"
	} else if reason == "skipped" {
		p.Process = "standard"
	}
	p.Usage, p.DeveloperTools = "compact", false
	if development {
		p.Usage, p.DeveloperTools = "detailed", true
	}
	o.mu.Unlock()
	o.start(p, true, false)
}

func (o *DesktopOnboarding) Retry() {
	o.mu.Lock()
	if o.view.Saving || o.updating {
		o.mu.Unlock()
		return
	}
	p, preferences := o.failed, o.failedPreferences
	o.mu.Unlock()
	if p != nil {
		o.start(*p, preferences, false)
	} else {
		o.store.RefreshDescribe(context.Background())
	}
}

func (o *DesktopOnboarding) start(next OnboardingProgress, preferences, preview bool) {
	o.mu.Lock()
	if o.view.Saving || o.client == nil {
		o.mu.Unlock()
		return
	}
	client, generation, status, operation := o.client, o.generation, o.account.Status, o.operation
	o.view.Saving, o.view.Error = true, ""
	if preview {
		o.draft = &next
		o.view.Progress = next
	}
	o.mu.Unlock()
	o.store.fire()
	go func() {
		ctx := context.Background()
		current := func() bool {
			o.mu.Lock()
			defer o.mu.Unlock()
			return o.client == client && o.generation == generation && o.account.Status == status && o.operation == operation
		}
		var err error
		if preferences {
			err = o.store.writeOnboardingFields(ctx, client, generation, "ui-chat", map[string]any{"transcriptView": next.Process, "performanceUsage": next.Usage})
			if err == nil && current() {
				err = o.store.writeOnboardingFields(ctx, client, generation, "ui-settings", map[string]any{"enabled": next.DeveloperTools})
			}
		}
		if err == nil && current() {
			err = o.store.writeOnboardingFields(ctx, client, generation, DesktopOnboardingNamespace, onboardingFields(next))
		}
		o.mu.Lock()
		if o.client != client || o.generation != generation || o.account.Status != status || o.operation != operation {
			o.mu.Unlock()
			return
		}
		o.view.Saving = false
		o.draft = nil
		if err != nil {
			o.failed = &next
			o.failedPreferences = preferences
			o.view.Error = err.Error()
		} else {
			o.failed = nil
			o.view.Progress = next
			if next.Step == "done" {
				o.view.Visible = false
			}
		}
		o.mu.Unlock()
		o.store.fire()
	}()
}

func onboardingFields(p OnboardingProgress) map[string]any {
	nullable := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	return map[string]any{"version": p.Version, "step": p.Step, "purpose": nullable(p.Purpose), "process": nullable(p.Process), "completion": nullable(p.Completion), "usage": p.Usage, "developerTools": p.DeveloperTools}
}

// Each phase preserves untouched user fields and accepts only the Host's
// returned values. The durable done marker is written after active preferences.
func (s *Store) writeOnboardingFields(ctx context.Context, client *dsh.Client, generation uint64, namespace string, values map[string]any) error {
	s.mu.Lock()
	ns := s.findNamespaceLocked(namespace)
	if s.client != client || s.generation != generation || s.describedClient != client || ns == nil || !s.writable || s.saving || s.loading {
		s.mu.Unlock()
		return fmt.Errorf("Onboarding settings are unavailable; retry after reconnecting")
	}
	revision := ns.Revision
	ops := []json.RawMessage{}
	for key, value := range values {
		raw, _ := json.Marshal(value)
		ops = append(ops, fieldOperation("set", []string{key}, raw))
	}
	s.saving = true
	s.mu.Unlock()
	s.fire()
	var out api.SettingsNamespaceView
	err := client.Call(ctx, "settings/mutate", map[string]any{"ns": namespace, "expectedRevision": revision, "ops": ops}, &out)
	s.mu.Lock()
	if s.client != client || s.generation != generation {
		s.mu.Unlock()
		return fmt.Errorf("Onboarding Host changed")
	}
	s.saving = false
	if err == nil {
		if current := s.findNamespaceLocked(namespace); current != nil {
			current.Revision = out.Revision
			current.Value = append(json.RawMessage(nil), out.Value...)
			current.Base = append(json.RawMessage(nil), out.Base...)
			current.User = append(json.RawMessage(nil), out.User...)
			current.HasSecrets = hasSetSecrets(out.Secrets)
			current.SecretPaths = configuredSecretPaths(out.Secrets)
			s.seedDraftsLocked(*current)
		} else {
			err = fmt.Errorf("Onboarding settings are no longer available")
		}
	}
	if err == nil {
		var accepted map[string]json.RawMessage
		_ = json.Unmarshal(out.Value, &accepted)
		for key, value := range values {
			raw, _ := json.Marshal(value)
			if !sameJSON(accepted[key], raw) {
				err = fmt.Errorf("Onboarding settings were not accepted; choices retained for retry")
				break
			}
		}
	}
	if err != nil {
		s.err = err
		s.lastAction = err.Error()
	}
	s.mu.Unlock()
	s.fire()
	return err
}
