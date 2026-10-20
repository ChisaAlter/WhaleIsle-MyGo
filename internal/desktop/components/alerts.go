package components

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
	"whaleisle/internal/desktop/bridgealerts"
	"whaleisle/internal/store/storelocale"
)

type bridgeAlertSource struct {
	root            string
	client          *http.Client
	processIdentity func(int, string) (string, error)
	open            func(string) error
}

// NewWhaleBridgeAlerts never starts a service or reads account/provider files.
// Its only credential access is the existing managed state.json web key.
func NewWhaleBridgeAlerts(userData string) *bridgealerts.Monitor {
	return bridgealerts.New(&bridgeAlertSource{root: filepath.Join(userData, "components", "whalebridge"), client: &http.Client{Timeout: 2 * time.Second}, processIdentity: bridgeAlertProcessIdentity, open: openURL})
}
func bridgeAlertAccess(st bridgeState) (*url.URL, error) {
	u, err := url.Parse(st.URL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil || !managementKeyPattern.MatchString(u.Query().Get("k")) {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeLocalManagementAddressIs))
	}
	return u, nil
}
func (s *bridgeAlertSource) capture(ctx context.Context) (bridgeState, string, error) {
	if err := ctx.Err(); err != nil {
		return bridgeState{}, "", err
	}
	st, err := readBridgeState(filepath.Join(s.root, "data", "state.json"))
	if os.IsNotExist(err) {
		return bridgeState{}, "", nil
	}
	if err != nil {
		return bridgeState{}, "", errors.New(storelocale.T(storelocale.KeyStoreComponentsCouldNotReadTheLocalWhalebridgeState))
	}
	inst, err := checkedBridgeInstalled(s.root)
	if errors.Is(err, ErrNotInstalled) {
		return bridgeState{}, "", nil
	}
	if err != nil {
		return bridgeState{}, "", errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeInstallRecord))
	}
	if st.ID != "whalebridge" || !st.Ready || st.PID <= 0 || st.Version != inst.Version {
		return bridgeState{}, "", errors.New(storelocale.T(storelocale.KeyStoreComponentsTheLocalWhalebridgeInstanceIdentityDoes))
	}
	if _, err = bridgeAlertAccess(st); err != nil {
		return bridgeState{}, "", err
	}
	identity, err := s.processIdentity(st.PID, filepath.Join(s.root, "versions", inst.Version, bridgeAsset))
	if err != nil {
		return bridgeState{}, "", errors.New(storelocale.T(storelocale.KeyStoreComponentsCouldNotVerifyTheInstalledWhalebridgeP))
	}
	raw, _ := json.Marshal(struct {
		State     bridgeState
		Installed installedFile
		Process   string
	}{st, inst, identity})
	return st, fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}
func (s *bridgeAlertSource) Current(ctx context.Context) (string, error) {
	_, key, err := s.capture(ctx)
	return key, err
}
func (s *bridgeAlertSource) verified(ctx context.Context, expected string) (bridgeState, error) {
	st, key, err := s.capture(ctx)
	if err != nil || key == "" || key != expected {
		return bridgeState{}, bridgealerts.ErrNotClaimed
	}
	if err := verifyBridgeReady(ctx, st, st.Version, st.PID, s.client); err != nil {
		return bridgeState{}, bridgealerts.ErrNotClaimed
	}
	current, key, err := s.capture(ctx)
	if err != nil || key != expected || current != st {
		return bridgeState{}, bridgealerts.ErrNotClaimed
	}
	return st, nil
}
func (s *bridgeAlertSource) Claim(ctx context.Context, key string) (bridgealerts.Batch, error) {
	// Share the existing lifecycle gate, so a local stop/switch cannot race claim.
	if !bridgeStartMu.TryLock() {
		return bridgealerts.Batch{}, bridgealerts.ErrNotClaimed
	}
	defer bridgeStartMu.Unlock()
	st, err := s.verified(ctx, key)
	if err != nil {
		return bridgealerts.Batch{}, err
	}
	u, _ := bridgeAlertAccess(st)
	// Fixed upstream service.alerts contract: POST, no body/query/since/ack.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+u.Host+"/api/subscriptions/alerts/claim", nil)
	if err != nil {
		return bridgealerts.Batch{}, bridgealerts.ErrNotClaimed
	}
	req.AddCookie(&http.Cookie{Name: "magpie_web_" + u.Port(), Value: u.Query().Get("k")})
	req.Header.Set("Content-Type", "application/json")
	safe := *s.client
	safe.Jar = nil
	safe.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	safe.Transport = &http.Transport{Proxy: nil, DisableKeepAlives: true}
	resp, err := safe.Do(req)
	if err != nil {
		return bridgealerts.Batch{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeReminderClaimResultIsUnc))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return bridgealerts.Batch{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeReminderEndpointReturnedA))
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, (256<<10)+1))
	if err != nil || len(b) > 256<<10 {
		return bridgealerts.Batch{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeReminderResultCouldNotBe))
	}
	return decodeBridgeAlertReceipt(b)
}

func decodeBridgeAlertReceipt(b []byte) (bridgealerts.Batch, error) {
	var wire struct {
		Sequence *int64                 `json:"sequence"`
		Alerts   *[]bridgealerts.Record `json:"alerts"`
	}
	if json.Unmarshal(b, &wire) != nil || wire.Sequence == nil || wire.Alerts == nil {
		return bridgealerts.Batch{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeReminderResultFormat))
	}
	return bridgealerts.Batch{Sequence: *wire.Sequence, Alerts: *wire.Alerts}, nil
}
func (s *bridgeAlertSource) Open(ctx context.Context, key string) error {
	if !bridgeStartMu.TryLock() {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeIsSwitchingViewTheReminders))
	}
	defer bridgeStartMu.Unlock()
	st, err := s.verified(ctx, key)
	if err != nil {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeInstanceChangedNoOtherIn))
	}
	u, _ := bridgeAlertAccess(st)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// Never use an alert URL, current generic manage fallback, or executable command.
	return s.open("http://" + u.Host + "/?k=" + url.QueryEscape(u.Query().Get("k")))
}
