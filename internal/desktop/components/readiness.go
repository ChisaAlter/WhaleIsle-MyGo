package components


import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
	"whaleisle/internal/store/storelocale"
)

type bridgeState struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	PID     int    `json:"pid"`
	Ready   bool   `json:"ready"`
	URL     string `json:"url"`
}

var managementKeyPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func readBridgeState(path string) (bridgeState, error) {
	var st bridgeState
	b, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if json.Unmarshal(b, &st) != nil {
		return st, errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeStateFile))
	}
	return st, nil
}

// verifyBridgeReady uses only the authenticated loopback status endpoint. Errors
// never contain the management URL, token, response body, or credentials.
func verifyBridgeReady(ctx context.Context, st bridgeState, version string, pid int, client *http.Client) error {
	if st.ID != "whalebridge" || !st.Ready || st.PID <= 0 || st.PID != pid || st.Version != version {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeProcessPidOrVersionDoes))
	}
	resp, err := bridgeLocalRequest(ctx, st, http.MethodGet, "status", nil, client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeStatusEndpointHttpD), resp.StatusCode))
	}
	var status struct {
		PID     int    `json:"pid"`
		Version string `json:"version"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&status) != nil || status.PID != pid || status.Version != version {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeStatusEndpointReportsADi))
	}
	return nil
}

// Both lifecycle calls share the same exact-loopback, no-proxy, no-redirect
// authentication boundary. Never place the management key in request URLs.
func bridgeLocalRequest(ctx context.Context, st bridgeState, method, route string, body io.Reader, client *http.Client) (*http.Response, error) {
	u, err := bridgeManagementURL(st.URL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://"+u.Host+"/api/whalebridge/"+route, body)
	if err != nil {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeLocalRequest))
	}
	req.AddCookie(&http.Cookie{Name: "magpie_web_" + u.Port(), Value: u.Query().Get("k")})
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	safe := *client
	safe.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	safe.Transport = &http.Transport{Proxy: nil, DisableKeepAlives: true}
	resp, err := safe.Do(req)
	if err != nil {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeLocalManagementEndpointIs))
	}
	return resp, nil
}

func bridgeMaintenance(ctx context.Context, st bridgeState, on bool, client *http.Client) error {
	b, _ := json.Marshal(map[string]bool{"on": on})
	resp, err := bridgeLocalRequest(ctx, st, http.MethodPost, "maintenance", bytes.NewReader(b), client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeIsStillGeneratingAReplyRetr))
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeMaintenanceEndpointHttpD), resp.StatusCode))
	}
	var result struct {
		Active *int `json:"active"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result) != nil || result.Active == nil || on && *result.Active != 0 {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeMaintenanceStateIsUnconfi))
	}
	return nil
}

func waitBridgeReady(ctx context.Context, path, version string, pid int, exited <-chan error, client *http.Client) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-exited:
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeStartProcessExitedCheckT))
		default:
		}
		st, err := readBridgeState(path)
		if err == nil && st.Ready {
			if err := verifyBridgeReady(ctx, st, version, pid, client); err != nil {
				return err
			}
			select {
			case <-exited:
				return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeStartProcessExitedCheckT))
			default:
				return nil
			}
		}
		if err != nil && !os.IsNotExist(err) {
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsCouldNotReadTheWhalebridgeReadinessSta))
		}
		select {
		case <-ctx.Done():
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeStartTimedOutOrWasCancelled))
		case <-exited:
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeStartProcessExitedCheckT))
		case <-ticker.C:
		}
	}
}

// bridgeManagementURL is shared by API calls and the explicit UI opener.
func bridgeManagementURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil || !managementKeyPattern.MatchString(u.Query().Get("k")) {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeLocalManagementAddressIs))
	}
	return u, nil
}
