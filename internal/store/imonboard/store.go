// Package imonboard implements the original dsh-im channel setup contracts.
// Draft credentials and pairing artifacts exist only in memory.
package imonboard

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type Platform struct {
	ID, Label, Identity, Secret, Extra, Guide string
	QR                                        bool
}

var Platforms = []Platform{
	{"feishu", storelocale.T(storelocale.KeyStoreImonboardLark), "App ID", "App Secret", "", storelocale.T(storelocale.KeyStoreImonboardScanWithFeishuToCreateTheAppAdministr), true},
	{"weixin", storelocale.T(storelocale.KeyStoreImonboardWechat), "", "", "", storelocale.T(storelocale.KeyStoreImonboardScanTheQrCodeWithWechatIfRiskVerific), true},
	{"dingtalk", storelocale.T(storelocale.KeyStoreImonboardDingtalk), "Client ID", "Client Secret", "", storelocale.T(storelocale.KeyStoreImonboardScanWithDingtalkOrEnterExistingAppCre), true},
	{"qq", "QQ", "App ID", "App Secret", "", storelocale.T(storelocale.KeyStoreImonboardScanWithQqOrEnterOpenPlatformAppCred), true},
	{"wecom", storelocale.T(storelocale.KeyStoreImonboardWecom), "Bot ID", "Secret", "", storelocale.T(storelocale.KeyStoreImonboardScanWithWecomOrEnterSmartBotCredentia), true},
	{"telegram", "Telegram", "", "Bot Token", "", storelocale.T(storelocale.KeyStoreImonboardCreateTheBotThroughTelegramBotfatherFi), false},
	{"discord", "Discord", "", "Bot Token", "", storelocale.T(storelocale.KeyStoreImonboardCreateABotInTheDiscordDeveloperPortal), false},
	{"whatsapp", "WhatsApp", "", "", "", storelocale.T(storelocale.KeyStoreImonboardScanTheQrCodeFromLinkedDevicesInWhat), true},
	{"slack", "Slack", "", "Bot Token", "App Token", storelocale.T(storelocale.KeyStoreImonboardCreateAndInstallTheAppInSlackFirstTh), false},
}

func PlatformByID(id string) Platform {
	for _, p := range Platforms {
		if p.ID == id {
			return p
		}
	}
	return Platform{}
}

type Draft struct{ Identity, Secret, Extra, VerifyCode string }

// Public presentation fields from the original connection.status snapshot.
// Keep credential bindings, tokens and raw transport payloads out of this view.
type LastMessageError struct {
	Code        string   `json:"code"`
	Reason      string   `json:"reason"`
	Message     string   `json:"message"`
	ReferenceID string   `json:"referenceId"`
	At          *float64 `json:"at"`
}
type Bot struct {
	BotID       string `json:"botId"`
	Workspace   string `json:"workspace"`
	AgentPreset string `json:"agentPreset"`
	State       string `json:"state"`
	Connected   bool   `json:"connected"`
	Bot         struct {
		Name string `json:"name"`
	} `json:"bot"`
	Health struct {
		Summary       string          `json:"summary"`
		LastCheckedAt json.RawMessage `json:"lastCheckedAt"`
	} `json:"health"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	LastMessageError              LastMessageError `json:"lastMessageError"`
	GroupResponseMode             string           `json:"groupResponseMode"`
	GroupMessagePermissionGranted bool             `json:"groupMessagePermissionGranted"`
	AccessPolicy                  struct {
		AccessMode     string   `json:"accessMode"`
		AllowedUsers   []string `json:"allowedUsers"`
		AllowedNumbers []string `json:"allowedNumbers"`
	} `json:"accessPolicy"`
}
type Provision struct {
	Submitted bool   `json:"submitted"`
	Message   string `json:"message"`
	Error     struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	AttemptID            string          `json:"attemptId"`
	ProvisioningID       string          `json:"provisioningId"`
	Status               string          `json:"status"`
	State                string          `json:"state"`
	BotID                string          `json:"botId"`
	Operation            string          `json:"operation"`
	QR                   string          `json:"qrCodeDataUrl"`
	URL                  string          `json:"verificationUrl"`
	ExpiresAt            json.RawMessage `json:"expiresAt"`
	ExpireIn             float64         `json:"expireIn"`
	ExpiresIn            float64         `json:"expiresIn"`
	PollIntervalMS       int             `json:"pollIntervalMs"`
	VerificationRequired bool            `json:"verificationRequired"`
	Deadline             time.Time       `json:"-"`
	StartedAt            time.Time       `json:"-"`
}

// A submitted Feishu app update is already being applied by the platform.
// The QR deadline and cancel/replacement actions no longer govern that phase.
func (p Provision) Applying() bool {
	return p.Operation != "" && (p.Submitted || p.Status == "connecting" || p.Status == "scanned") && p.Status != "failed" && p.Status != "cancelled" && p.Status != "expired" && p.Status != "connected"
}

func (p Provision) Cancelable() bool {
	return !((p.Operation == "callback_repair" || p.Operation == "group_message_permission") && p.Applying())
}

type View struct {
	TestNotice                      string
	StatusError                     string
	Open, Available, Busy, Loaded   bool
	Scope                           uint64
	Channel, Notice, Error, Host    string
	Draft                           Draft
	Bots                            []Bot
	Provision                       Provision
	Review, Target, Mode, Allowlist string
	PolicyDrafts                    map[string]string
	WorkspaceDrafts                 map[string]string
	Presets                         []Preset
}
type Store struct {
	mu               sync.Mutex
	client           *pluginrpc.Client
	notify           func()
	view             View
	cancel           context.CancelFunc
	statusCancel     context.CancelFunc
	statusGeneration uint64
	statusReady      bool
}

func (s *Store) fire() {
	s.mu.Lock()
	f := s.notify
	s.mu.Unlock()
	if f != nil {
		f()
	}
}
func (s *Store) Bind(client *pluginrpc.Client, epoch uint64, notify func()) {
	s.mu.Lock()
	if s.statusCancel != nil {
		s.statusCancel()
		s.statusCancel = nil
	}
	s.statusReady = false
	s.statusGeneration++
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	old, ch, attempt := s.client, s.view.Channel, s.view.Provision.AttemptID
	s.client = client
	s.notify = notify
	s.view = View{Available: client != nil, Scope: s.view.Scope + 1}
	if client != nil {
		if u, err := url.Parse(client.BaseURL); err == nil {
			s.view.Host = u.Scheme + "://" + u.Host
		}
	}
	s.mu.Unlock()
	if old != nil && attempt != "" {
		go cancelAttempt(old, ch, attempt)
	}
}
func cancelAttempt(p *pluginrpc.Client, ch, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = p.Call(ctx, ch, "provision.cancel", map[string]any{"attemptId": id}, &json.RawMessage{})
}
func (s *Store) Snapshot() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.view
	v.Bots = append([]Bot(nil), v.Bots...)
	for i := range v.Bots {
		v.Bots[i].Health.LastCheckedAt = append(json.RawMessage(nil), v.Bots[i].Health.LastCheckedAt...)
		if v.Bots[i].LastMessageError.At != nil {
			at := *v.Bots[i].LastMessageError.At
			v.Bots[i].LastMessageError.At = &at
		}
	}
	v.PolicyDrafts = map[string]string{}
	for k, value := range s.view.PolicyDrafts {
		v.PolicyDrafts[k] = value
	}
	v.WorkspaceDrafts = map[string]string{}
	for k, value := range s.view.WorkspaceDrafts {
		v.WorkspaceDrafts[k] = value
	}
	v.Presets = append([]Preset(nil), s.view.Presets...)
	return v
}
func (s *Store) Show(ch string) {
	if PlatformByID(ch).ID == "" {
		return
	}
	s.Hide()
	s.mu.Lock()
	s.view.Open = true
	s.view.Channel = ch
	var statusContext context.Context
	if s.client != nil {
		statusContext, s.statusCancel = context.WithCancel(context.Background())
	}
	client, scope := s.client, s.view.Scope
	s.mu.Unlock()
	s.Refresh()
	if statusContext != nil {
		go s.pollConnectionStatus(statusContext, client, scope, ch)
	}
}
func (s *Store) Hide() {
	s.mu.Lock()
	if s.statusCancel != nil {
		s.statusCancel()
		s.statusCancel = nil
	}
	s.statusReady = false
	s.statusGeneration++
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	p, ch, id := s.client, s.view.Channel, s.view.Provision.AttemptID
	available := p != nil
	scope := s.view.Scope + 1
	host := s.view.Host
	s.view = View{Available: available, Scope: scope, Host: host}
	s.mu.Unlock()
	if p != nil && id != "" {
		go cancelAttempt(p, ch, id)
	}
	s.fire()
}
func (s *Store) Edit(scope uint64, d Draft) {
	s.mu.Lock()
	if s.view.Scope == scope && !s.view.Busy {
		s.view.Draft = d
	}
	s.mu.Unlock()
}
func (s *Store) Review(scope uint64, action, target, mode, allowlist string) {
	s.mu.Lock()
	if scope == s.view.Scope && !s.view.Busy {
		s.view.Review = action
		s.view.Target = target
		s.view.Mode = mode
		s.view.Allowlist = allowlist
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) EditPolicy(scope uint64, id, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if scope == s.view.Scope && !s.view.Busy {
		if s.view.PolicyDrafts == nil {
			s.view.PolicyDrafts = map[string]string{}
		}
		s.view.PolicyDrafts[id] = text
	}
}
func (s *Store) Dismiss() {
	s.mu.Lock()
	if !s.view.Busy {
		s.view.Review = ""
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Refresh() {
	s.mu.Lock()
	provision := s.view.Provision
	if s.client != nil && s.view.Open && !s.view.Busy && provision.AttemptID != "" && provision.Status != "failed" && provision.Status != "cancelled" && provision.Status != "expired" {
		if s.cancel != nil {
			s.cancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		client, scope, channel := s.client, s.view.Scope, s.view.Channel
		s.view.Error = ""
		s.mu.Unlock()
		s.fire()
		// Observation resumes the same attempt; it never replays a mutation.
		go s.poll(ctx, client, scope, channel, provision)
		return
	}
	s.mu.Unlock()
	s.run("connection.status", map[string]any{}, false)
}
func (s *Store) Confirm() {
	s.mu.Lock()
	v := s.view
	if v.Channel == "feishu" && v.Provision.Applying() && (v.Review == "provision.begin" || v.Review == "provision.cancel" && !v.Provision.Cancelable()) {
		s.view.Review = ""
		s.mu.Unlock()
		s.fire()
		return
	}
	if v.Review != "bot.delete" {
		s.view.Review = ""
	}
	s.mu.Unlock()
	payload := map[string]any{}
	method := v.Review
	pairing := false
	switch method {
	case "provision.begin":
		// Refresh an existing targeted authorization as that same operation/Bot,
		// not as a new application registration.
		if v.Channel == "feishu" && v.Provision.BotID != "" {
			switch v.Provision.Operation {
			case "callback_repair":
				method = "bot.callback-repair.begin"
			case "group_message_permission":
				method = "bot.group-message-permission.begin"
			}
			if method != "provision.begin" {
				payload["botId"] = v.Provision.BotID
			}
		}
		if v.Channel != "whatsapp" {
			if method == "provision.begin" {
				payload["locale"] = "zh-CN"
			}
		}
		pairing = true
	case "bot.callback-repair.begin", "bot.group-message-permission.begin":
		payload["botId"] = v.Target
		pairing = true
	case "bot.bind-credentials":
		switch v.Channel {
		case "feishu", "qq":
			payload = map[string]any{"appId": strings.TrimSpace(v.Draft.Identity), "appSecret": strings.TrimSpace(v.Draft.Secret)}
		case "dingtalk":
			payload = map[string]any{"clientId": strings.TrimSpace(v.Draft.Identity), "clientSecret": strings.TrimSpace(v.Draft.Secret)}
		case "wecom":
			payload = map[string]any{"botId": strings.TrimSpace(v.Draft.Identity), "secret": strings.TrimSpace(v.Draft.Secret)}
		case "telegram", "discord":
			payload["token"] = strings.TrimSpace(v.Draft.Secret)
		case "slack":
			payload = map[string]any{"botToken": strings.TrimSpace(v.Draft.Secret), "appToken": strings.TrimSpace(v.Draft.Extra)}
		default:
			return
		}
	case "provision.verify":
		if !ValidVerificationCode(v.Draft.VerifyCode) {
			s.mu.Lock()
			if s.view.Scope == v.Scope {
				s.view.Error = "请输入手机微信显示的 4–8 位数字配对码。"
			}
			s.mu.Unlock()
			s.fire()
			return
		}
		payload = map[string]any{"attemptId": v.Provision.AttemptID, "verifyCode": strings.TrimSpace(v.Draft.VerifyCode)}
		pairing = true
	case "provision.cancel":
		payload["attemptId"] = v.Provision.AttemptID
	case "bot.reconnect-test":
		if PlatformByID(v.Channel).ID == "" {
			return
		}
		method = "bot.reconnect"
		payload = map[string]any{"botId": v.Target, "sendTest": true}
	case "bot.reconnect":
		payload = map[string]any{"botId": v.Target, "sendTest": false}
	case "bot.delete":
		payload = map[string]any{"botId": v.Target, "confirm": true}
	case "bot.disconnect":
		payload["botId"] = v.Target
	case "bot.group-response-mode.set":
		payload = map[string]any{"botId": v.Target, "groupResponseMode": v.Mode}
	case "bot.access-policy.set":
		payload = map[string]any{"botId": v.Target, "accessMode": v.Mode}
		values, message := normalizeAccessEntries(v.Channel, v.Allowlist)
		if message != "" {
			s.mu.Lock()
			if s.view.Scope == v.Scope {
				s.view.Error = message
			}
			s.mu.Unlock()
			s.fire()
			return
		}
		if v.Channel == "telegram" {
			payload["allowedUsers"] = values
		} else {
			payload["allowedNumbers"] = values
		}
	default:
		return
	}
	s.run(method, payload, pairing)
}
func (s *Store) run(method string, payload map[string]any, pairing bool) {
	s.runScoped(0, method, payload, pairing)
}
func (s *Store) runScoped(expectedScope uint64, method string, payload map[string]any, pairing bool) {
	s.mu.Lock()
	if s.client == nil || s.view.Busy || !s.view.Open || expectedScope != 0 && expectedScope != s.view.Scope {
		s.mu.Unlock()
		return
	}
	if method == "bot.workspace.set" || method == "bot.preset.set" {
		if s.view.Review != "" || s.view.Provision.AttemptID != "" {
			s.mu.Unlock()
			return
		}
		if message := s.validateBindingLocked(method, payload); message != "" {
			s.view.Error = message
			s.mu.Unlock()
			s.fire()
			return
		}
	}
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	p, scope, ch, previous := s.client, s.view.Scope, s.view.Channel, s.view.Provision
	s.statusGeneration++
	s.view.Busy = true
	s.view.Error = ""
	s.view.Notice = storelocale.T(storelocale.KeyStoreImonboardWaitingForTheHost)
	if method == "bot.reconnect" {
		s.view.TestNotice = ""
	}
	s.mu.Unlock()
	s.fire()
	go func() {
		call := func(method string, args any, out any) error {
			c, done := context.WithTimeout(ctx, 30*time.Second)
			defer done()
			return p.Call(c, ch, method, args, out)
		}
		var raw json.RawMessage
		if pairing && method != "provision.verify" && previous.AttemptID != "" {
			if err := call("provision.cancel", map[string]any{"attemptId": previous.AttemptID}, &raw); err != nil {
				s.finish(scope, storelocale.T(storelocale.KeyStoreImonboardCouldNotCancelThePreviousSetupRefresh), nil)
				return
			}
		}
		if err := call(method, payload, &raw); err != nil {
			s.finish(scope, storelocale.T(storelocale.KeyStoreImonboardHostOperationFailedCheckThePlatformCon)+": "+publicOperationError(err, payload), nil)
			return
		}
		// A definite accepted mutation clears sensitive input. A rejected or
		// failed call keeps the in-memory form intact for correction in place.
		s.mu.Lock()
		if s.view.Scope == scope {
			if method == "bot.bind-credentials" {
				s.view.Draft = Draft{}
			}
			if method == "provision.verify" {
				s.view.Draft.VerifyCode = ""
			}
			if method == "bot.delete" {
				s.view.Review = ""
			}
		}
		s.mu.Unlock()
		testNotice, testError := "", ""
		if method == "bot.reconnect" && payload["sendTest"] == true {
			testNotice, testError = reconnectTestResult(raw, reconnectBotID(payload))
			s.mu.Lock()
			if s.view.Scope == scope {
				s.view.TestNotice = testNotice
			}
			s.mu.Unlock()
		}
		if method == "provision.cancel" {
			var result struct {
				Status string `json:"status"`
			}
			_ = json.Unmarshal(raw, &result)
			if result.Status == "connecting" || result.Status == "connected" {
				pairing = true
			}
		}
		if pairing {
			base := Provision{}
			if method == "provision.verify" || method == "provision.cancel" {
				base = previous
			}
			provision := decodeProvision(raw, base)
			if method == "bot.callback-repair.begin" || method == "bot.group-message-permission.begin" {
				expected := "callback_repair"
				if method == "bot.group-message-permission.begin" {
					expected = "group_message_permission"
				}
				if provision.Operation != expected || provision.BotID != payload["botId"] {
					s.finish(scope, "Host returned an authorization for a different operation or Bot", nil)
					return
				}
			}
			if provision.AttemptID == "" {
				s.finish(scope, storelocale.T(storelocale.KeyStoreImonboardTheHostReturnedNoValidSetupTask), nil)
				return
			}
			s.mu.Lock()
			if scope != s.view.Scope {
				s.mu.Unlock()
				cancelAttempt(p, ch, provision.AttemptID)
				return
			}
			s.view.Busy = false
			s.view.Provision = provision
			s.view.Notice = storelocale.T(storelocale.KeyStoreImonboardWaitingForPlatformAuthorizationConnectio)
			s.mu.Unlock()
			s.fire()
			s.poll(ctx, p, scope, ch, provision)
			return
		}
		if method == "provision.cancel" {
			s.mu.Lock()
			if scope == s.view.Scope {
				s.view.Provision = Provision{}
				s.view.Draft.VerifyCode = ""
			}
			s.mu.Unlock()
		}
		if method != "connection.status" {
			if err := call("connection.status", map[string]any{}, &raw); err != nil {
				s.finish(scope, storelocale.T(storelocale.KeyStoreImonboardTheActionWasSubmittedButTheStatusRefr), nil)
				return
			}
		}
		if previous.AttemptID == "" {
			var status struct {
				Provisioning json.RawMessage `json:"provisioning"`
				Snapshot     *struct {
					Provisioning json.RawMessage `json:"provisioning"`
				} `json:"snapshot"`
			}
			_ = json.Unmarshal(raw, &status)
			active := status.Provisioning
			if status.Snapshot != nil {
				active = status.Snapshot.Provisioning
			}
			if len(active) > 0 && string(active) != "null" {
				previous = decodeProvision(active, Provision{})
				if previous.Status == "connected" {
					previous = Provision{}
				}
				s.mu.Lock()
				if scope == s.view.Scope {
					s.view.Provision = previous
				}
				s.mu.Unlock()
			}
		}
		s.finish(scope, "", raw)
		if testNotice != "" || testError != "" {
			s.mu.Lock()
			if s.view.Scope == scope {
				s.view.TestNotice = testNotice
				if testError != "" {
					s.view.Error = testError
				}
			}
			s.mu.Unlock()
			s.fire()
		}
		if method != "provision.cancel" && previous.AttemptID != "" {
			s.poll(ctx, p, scope, ch, previous)
		}
	}()
}
func (s *Store) finish(scope uint64, message string, raw json.RawMessage) {
	s.mu.Lock()
	if scope != s.view.Scope {
		s.mu.Unlock()
		return
	}
	s.view.Busy = false
	s.view.Error = message
	if message == "" {
		bots, presets, ok := decodeConnectionStatus(raw)
		if !ok {
			s.view.Error = storelocale.T(storelocale.KeyStoreImonboardHostStatusFormatIsUnusable)
		} else {
			s.applyConnectionStatusLocked(bots, presets)
			s.view.Loaded = true
			s.statusReady = true
			s.view.StatusError = ""
			s.view.Notice = storelocale.T(storelocale.KeyStoreImonboardHostConnectionStatusRefreshed)
		}
	}
	s.mu.Unlock()
	s.fire()
}

// Match the original channel card's ready-state 15s status observation. This
// reads only the captured Host/channel; actions and provisioning own their RPCs.
func (s *Store) pollConnectionStatus(ctx context.Context, client *pluginrpc.Client, scope uint64, channel string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		s.mu.Lock()
		if ctx.Err() != nil || s.client != client || s.view.Scope != scope || !s.view.Open || s.view.Channel != channel {
			s.mu.Unlock()
			return
		}
		if !s.statusReady || s.view.Busy || s.view.Review != "" || s.view.Provision.AttemptID != "" {
			s.mu.Unlock()
			continue
		}
		generation := s.statusGeneration
		s.mu.Unlock()
		callContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		var raw json.RawMessage
		err := client.Call(callContext, channel, "connection.status", map[string]any{}, &raw)
		cancel()
		bots, presets, valid := decodeConnectionStatus(raw)
		message := ""
		if err != nil {
			message = storelocale.T(storelocale.KeyStoreImonboardConnectionStatusRefreshFailedRefresh)
		} else if !valid {
			message = storelocale.T(storelocale.KeyStoreImonboardIncompleteHostConnectionStatus)
		}
		s.mu.Lock()
		if ctx.Err() != nil || s.client != client || s.view.Scope != scope || s.statusGeneration != generation || !s.view.Open || s.view.Channel != channel || s.view.Busy || s.view.Review != "" || s.view.Provision.AttemptID != "" {
			s.mu.Unlock()
			continue
		}
		s.view.StatusError = message
		if message == "" {
			s.applyConnectionStatusLocked(bots, presets)
		}
		s.mu.Unlock()
		s.fire()
	}
}
func decodeProvision(raw json.RawMessage, previous Provision) Provision {
	var wrap struct {
		Provisioning json.RawMessage `json:"provisioning"`
		Status       string          `json:"status"`
		State        string          `json:"state"`
		Message      string          `json:"message"`
	}
	_ = json.Unmarshal(raw, &wrap)
	value := raw
	if len(wrap.Provisioning) > 0 && string(wrap.Provisioning) != "null" {
		value = wrap.Provisioning
	}
	p := previous
	_ = json.Unmarshal(value, &p)
	if wrap.Message != "" {
		p.Message = wrap.Message
		p.Error.Message = wrap.Message
	}
	if p.AttemptID == "" {
		p.AttemptID = p.ProvisioningID
	}
	if wrap.Status != "" {
		p.Status = wrap.Status
	} else if wrap.State != "" {
		p.Status = wrap.State
	}
	if p.Status == "" {
		p.Status = "pending"
	}
	if p.StartedAt.IsZero() {
		p.StartedAt = time.Now()
	}
	if len(p.ExpiresAt) > 0 {
		var ms float64
		var text string
		if json.Unmarshal(p.ExpiresAt, &ms) == nil {
			p.Deadline = time.UnixMilli(int64(ms))
		} else if json.Unmarshal(p.ExpiresAt, &text) == nil {
			p.Deadline, _ = time.Parse(time.RFC3339, text)
		}
	}
	if p.Deadline.IsZero() {
		seconds := p.ExpireIn
		if seconds == 0 {
			seconds = p.ExpiresIn
		}
		if seconds == 0 {
			seconds = 300
		}
		p.Deadline = time.Now().Add(time.Duration(seconds) * time.Second)
	}
	if p.PollIntervalMS < 500 {
		p.PollIntervalMS = 1000
	}
	if p.PollIntervalMS > 10000 {
		p.PollIntervalMS = 10000
	}
	return p
}
func (s *Store) poll(ctx context.Context, p *pluginrpc.Client, scope uint64, ch string, provision Provision) {
	for {
		if ctx.Err() != nil {
			return
		}
		if provision.Status == "connected" {
			s.mu.Lock()
			if scope != s.view.Scope || ctx.Err() != nil {
				s.mu.Unlock()
				return
			}
			if scope == s.view.Scope {
				s.view.Provision = Provision{}
				s.view.Notice = storelocale.T(storelocale.KeyStoreImonboardPlatformAuthorizationCompleteReadingConn)
			}
			s.mu.Unlock()
			s.Refresh()
			return
		}
		terminal := provision.Status == "failed" || provision.Status == "cancelled" || provision.Status == "expired"
		if terminal || !provision.Applying() && time.Now().After(provision.Deadline) {
			s.mu.Lock()
			if scope == s.view.Scope && ctx.Err() == nil {
				if !terminal {
					provision.Status = "expired"
				}
				provision.QR = ""
				provision.URL = ""
				s.view.Provision = provision
				s.view.Notice = storelocale.T(storelocale.KeyStoreImonboardSetupStopped) + provision.Status
			}
			s.mu.Unlock()
			s.fire()
			return
		}
		timer := time.NewTimer(time.Duration(provision.PollIntervalMS) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		var raw json.RawMessage
		c, done := context.WithTimeout(ctx, 30*time.Second)
		err := p.Call(c, ch, "provision.poll", map[string]any{"attemptId": provision.AttemptID}, &raw)
		done()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			s.finish(scope, "暂时无法读取接入状态。刷新会继续核对此接入任务，不会重新提交授权。", nil)
			return
		}
		provision = decodeProvision(raw, provision)
		s.mu.Lock()
		if scope != s.view.Scope || ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		s.view.Provision = provision
		s.mu.Unlock()
		s.fire()
	}
}
