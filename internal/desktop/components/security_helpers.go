package components

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const maxBridgeHTTPRedirects = 10

func isReparseOrSymlink(info os.FileInfo) bool {
	return info != nil && info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0
}

func validComponentVersionSegment(v string) bool {
	if v == "" || v == "." || v == ".." || len(v) > 128 {
		return false
	}
	if strings.ContainsAny(v, `/\:`+"\x00") || strings.Contains(v, "..") {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c < 0x20 || c == 0x7f || c == ' ' || c == '\t' {
			return false
		}
	}
	return true
}

func lstatNotReparse(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if isReparseOrSymlink(info) {
		return nil, fmt.Errorf("refusing symlink or junction path %s", path)
	}
	return info, nil
}

// writeFileAtomicRetry atomically writes data to path via a unique temp file,
// Sync, Close, and bounded rename retries without deleting the destination first.
func writeFileAtomicRetry(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if existing, err := os.Lstat(path); err == nil {
		if isReparseOrSymlink(existing) {
			return fmt.Errorf("refusing atomic overwrite of symlink or junction %s", path)
		}
		if existing.IsDir() {
			return fmt.Errorf("destination %s is a directory", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keepTmp := false
	defer func() {
		if !keepTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil && runtime.GOOS != "windows" {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	var renameErr error
	attempts := 1
	if runtime.GOOS == "windows" {
		attempts = 8
	}
	for attempt := 0; attempt < attempts; attempt++ {
		renameErr = os.Rename(tmpPath, path)
		if renameErr == nil {
			keepTmp = true
			return nil
		}
		if runtime.GOOS != "windows" || (!errors.Is(renameErr, os.ErrPermission) && !isWindowsSharingError(renameErr)) {
			break
		}
		time.Sleep(time.Duration(15*(attempt+1)) * time.Millisecond)
	}
	return renameErr
}

func isWindowsSharingError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "used by another process") ||
		strings.Contains(msg, "access is denied") ||
		strings.Contains(msg, "sharing violation")
}

func shouldStripChildEnvKey(key string, extraStrip map[string]struct{}) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" {
		return true
	}
	if _, ok := extraStrip[upper]; ok {
		return true
	}
	if strings.HasPrefix(upper, "DSH_DESKTOP_INSTALL_") {
		return true
	}
	if strings.HasSuffix(upper, "_CONTROL_TOKEN") || strings.HasSuffix(upper, "_WEB_RUNKEY") {
		return true
	}
	switch upper {
	case "NODE_OPTIONS",
		"NODE_EXTRA_CA_CERTS",
		"ELECTRON_RUN_AS_NODE",
		"MAGPIE_WEB_KEY",
		"MAGPIE_WEB_RUNKEY",
		"MAGPIE_TOKEN",
		"MAGPIE_AUTH_TOKEN",
		"MAGPIE_ADDR",
		"WHALEISLE_CONTROL_TOKEN",
		"CONTROL_PLANE_TOKEN",
		"DSH_CONTROL_TOKEN",
		"DSH_AUTH_TOKEN",
		"DSH_ACCESS_TOKEN",
		"DSH_SESSION_TOKEN",
		"DSH_API_TOKEN",
		"LAUNCHER_COMPONENT_ID",
		"LAUNCHER_COMPONENT_DATA_DIR",
		"WHALEBRIDGE_DSH_HOME":
		return true
	}
	return false
}

// sanitizeComponentEnv filters installer/runtime injection and control-plane
// secrets from base before appending explicit overrides.
func sanitizeComponentEnv(base []string, overrides ...string) []string {
	extraStrip := make(map[string]struct{}, len(overrides))
	for _, entry := range overrides {
		if k, _, ok := strings.Cut(entry, "="); ok {
			extraStrip[strings.ToUpper(strings.TrimSpace(k))] = struct{}{}
		}
	}
	out := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		k, _, ok := strings.Cut(entry, "=")
		if !ok || shouldStripChildEnvKey(k, extraStrip) {
			continue
		}
		out = append(out, entry)
	}
	return append(out, overrides...)
}

func bridgeNormalizedHost(u *url.URL) string {
	if u == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
}

func bridgeIsLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

func bridgeIsPrivateOrSpecialHost(host string) bool {
	h := strings.TrimSuffix(strings.TrimSpace(host), ".")
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func validateBridgeRemoteURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Hostname() == "" {
		return nil, errors.New("component URL must be a valid HTTPS URL without credentials")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "https" {
		return u, nil
	}
	if scheme == "http" && bridgeIsLoopbackHost(u.Hostname()) {
		return u, nil
	}
	return nil, fmt.Errorf("component URL must use HTTPS (got %q)", u.Scheme)
}

func bridgeAllowedCrossHostRedirect(firstHost, nextHost string) bool {
	switch {
	case firstHost == "github.com" || firstHost == "api.github.com" || strings.HasSuffix(firstHost, ".github.com"):
		return nextHost == "github.com" ||
			strings.HasSuffix(nextHost, ".github.com") ||
			strings.HasSuffix(nextHost, ".githubusercontent.com")
	case firstHost == "cnb.cool" || strings.HasSuffix(firstHost, ".cnb.cool"):
		return nextHost == "cnb.cool" || strings.HasSuffix(nextHost, ".cnb.cool")
	default:
		return false
	}
}

func bridgeHardenedHTTPClient(base *http.Client, defaultTimeout time.Duration) *http.Client {
	var client http.Client
	if base != nil {
		client = *base
	}
	if client.Timeout <= 0 {
		client.Timeout = defaultTimeout
	}
	prevCheck := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxBridgeHTTPRedirects {
			return errors.New("too many component download redirects")
		}
		nextURL, err := validateBridgeRemoteURL(req.URL.String())
		if err != nil {
			return fmt.Errorf("refusing unsafe component redirect: %w", err)
		}
		if len(via) > 0 {
			initial := via[0].URL
			prev := via[len(via)-1].URL
			if strings.EqualFold(prev.Scheme, "https") && !strings.EqualFold(nextURL.Scheme, "https") {
				return fmt.Errorf("refusing component redirect downgrade from %s to %s", prev.Scheme, nextURL.Scheme)
			}
			if strings.EqualFold(initial.Scheme, "https") && !strings.EqualFold(nextURL.Scheme, "https") {
				return fmt.Errorf("refusing component redirect downgrade from %s to %s", initial.Scheme, nextURL.Scheme)
			}
			firstHost := bridgeNormalizedHost(initial)
			nextHost := bridgeNormalizedHost(nextURL)
			if nextHost != firstHost {
				if !bridgeIsLoopbackHost(firstHost) && bridgeIsPrivateOrSpecialHost(nextHost) {
					return fmt.Errorf("refusing component redirect to internal host %q", nextHost)
				}
				if bridgeIsLoopbackHost(firstHost) && bridgeIsLoopbackHost(nextHost) {
					return fmt.Errorf("refusing cross-host loopback component redirect from %q to %q", firstHost, nextHost)
				}
				if !bridgeAllowedCrossHostRedirect(firstHost, nextHost) || !strings.EqualFold(nextURL.Scheme, "https") {
					return fmt.Errorf("refusing cross-host component redirect from %q to %q", firstHost, nextHost)
				}
			}
			if !strings.EqualFold(bridgeNormalizedHost(prev), nextHost) {
				req.Header.Del("Authorization")
				req.Header.Del("Cookie")
			}
		}
		if prevCheck != nil {
			return prevCheck(req, via)
		}
		return nil
	}
	return &client
}
