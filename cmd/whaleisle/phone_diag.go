package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/paths"
)

// PhoneDiagEnv enables the token-free remote snapshot record used by acceptance
// runs. It is opt-in so a normal launch writes nothing.
const PhoneDiagEnv = "WHALEISLE_REMOTE_DIAG"

// phoneDiag is the observable shape of the phone runtime without the one-time
// pairing token: enough to prove ready/listening/offer without persisting a
// secret that a phone could use.
type phoneDiag struct {
	UpdatedAt      string `json:"updatedAt"`
	Status         string `json:"status"`
	Listening      bool   `json:"listening"`
	RelayConnected bool   `json:"relayConnected"`
	Message        string `json:"message,omitempty"`
	HasPairingURL  bool   `json:"hasPairingURL"`
	PairingURLHash string `json:"pairingURLSha256,omitempty"`
	Scheme         string `json:"scheme,omitempty"`
	Host           string `json:"host,omitempty"`
	Port           string `json:"port,omitempty"`
	FragmentPrefix string `json:"fragmentPrefix,omitempty"`
	OfferVersion   int    `json:"offerVersion,omitempty"`
	OfferExpiresMs int64  `json:"offerExpiresAtMs,omitempty"`
	RelayEndpoint  string `json:"relayEndpoint,omitempty"`
	RelayUseTLS    bool   `json:"relayUseTls,omitempty"`
	ExpiresAt      string `json:"expiresAt,omitempty"`
	Devices        int    `json:"devices"`
	PairingImage   bool   `json:"pairingImage"`
}

// describePhoneRuntime turns a snapshot into the diag record. The pairing token
// is never copied: only its hash and the URL/offer metadata are kept.
func describePhoneRuntime(snap phoneremote.RuntimeState) phoneDiag {
	d := phoneDiag{
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
		Status:         snap.Status,
		Listening:      snap.Listening,
		RelayConnected: snap.RelayConnected,
		Message:        snap.Message,
		Devices:        len(snap.Devices),
		PairingImage:   snap.PairingImage != nil,
	}
	if !snap.ExpiresAt.IsZero() {
		d.ExpiresAt = snap.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if snap.PairingURL == "" {
		return d
	}
	d.HasPairingURL = true
	sum := sha256.Sum256([]byte(snap.PairingURL))
	d.PairingURLHash = hex.EncodeToString(sum[:])
	if u, err := url.Parse(snap.PairingURL); err == nil {
		d.Scheme, d.Host, d.Port = u.Scheme, u.Hostname(), u.Port()
		if i := strings.Index(u.Fragment, "="); i > 0 {
			d.FragmentPrefix = "#" + u.Fragment[:i+1]
			if raw, err := base64.RawURLEncoding.DecodeString(u.Fragment[i+1:]); err == nil {
				var offer struct {
					V    int `json:"v"`
					Auth struct {
						Expires int64 `json:"expiresAtMs"`
					} `json:"authBootstrap"`
					Relay struct {
						Endpoint string `json:"endpoint"`
						UseTLS   bool   `json:"useTls"`
					} `json:"relay"`
				}
				if json.Unmarshal(raw, &offer) == nil {
					d.OfferVersion = offer.V
					d.OfferExpiresMs = offer.Auth.Expires
					d.RelayEndpoint = offer.Relay.Endpoint
					d.RelayUseTLS = offer.Relay.UseTLS
				}
			}
		}
	}
	return d
}

// startPhoneDiagSampler starts the bounded diag sampler once per process when
// the acceptance env is set. It never runs for a normal launch.
func (s *mainState) startPhoneDiagSampler(rt *phoneremote.Runtime) {
	if strings.TrimSpace(os.Getenv(PhoneDiagEnv)) == "" {
		return
	}
	s.mu.Lock()
	already := s.phoneDiagSampling
	s.phoneDiagSampling = true
	s.mu.Unlock()
	if already {
		return
	}
	go s.samplePhoneDiag(rt)
}

// writePhoneDiag records the snapshot when PhoneDiagEnv is set. Best effort: a
// failure to write must never disturb the remote lifecycle.
func (s *mainState) writePhoneDiag(snap phoneremote.RuntimeState) {
	if strings.TrimSpace(os.Getenv(PhoneDiagEnv)) == "" {
		return
	}
	data, err := json.MarshalIndent(describePhoneRuntime(snap), "", "  ")
	if err != nil {
		return
	}
	dir := paths.RuntimeDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "remote-pairing.json"), append(data, '\n'), 0o600)
}

// samplePhoneDiag keeps the diag record current while the acceptance env is set:
// refreshPhoneRemote returns as soon as the daemon reports "starting", and the
// ready/pairing transition happens later in the runtime goroutine. Bounded and
// debug-only; it never writes the offer token.
func (s *mainState) samplePhoneDiag(rt *phoneremote.Runtime) {
	for i := 0; i < 300; i++ {
		time.Sleep(time.Second)
		snap := rt.Snapshot()
		s.writePhoneDiag(snap)
		if snap.Status == "offline" || snap.Status == "disabled" {
			return
		}
	}
}
