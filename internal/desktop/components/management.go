package components

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// WhaleBridgeManagementAddress returns the validated loopback management
// address of a ready WhaleBridge instance. The embedded management window
// (port of reference/electron/src/main/whalebridge-window.js) loads exactly this
// address and refuses every other document, so no caller-supplied URL and no
// remote origin can reach it.
func WhaleBridgeManagementAddress(userData string) (string, error) {
	if strings.TrimSpace(userData) == "" {
		return "", errors.New("whalebridge: empty user data directory")
	}
	root := filepath.Join(userData, "components", "whalebridge")
	b, err := os.ReadFile(filepath.Join(root, "data", "state.json"))
	if err != nil {
		return "", err
	}
	var st struct {
		URL   string `json:"url"`
		Ready bool   `json:"ready"`
	}
	if json.Unmarshal(b, &st) != nil || !st.Ready || strings.TrimSpace(st.URL) == "" {
		return "", errors.New("whalebridge management page is not ready")
	}
	u, err := bridgeManagementURL(st.URL)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
