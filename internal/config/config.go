// Package config reads and writes the desktop configuration file.
//
// The file is the same config.json the Electron shell kept in its user data
// directory (see reference/electron/src/main/config.js). Every field the
// Electron shell persists is decoded with its exact JSON name and default;
// the pet state blobs (pet, live2dPet) and the pluginRecovery object are
// carried through in the raw key/value map on ordinary saves, so the shells
// can share one data directory. Explicit native care/growth actions update
// only their shared live2dPet.stats/growth fields via pet_care.go/pet_growth.go;
// other pet fields remain untouched.
//
// Secrets (apiKey, baseUrl, githubToken, remoteToken, remoteRelayToken,
// remoteDevices) never live in config.json: the Electron shell strips them
// from the public layer and stores them in credentials.json, which the
// internal/credentials package handles.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"whaleisle/internal/petgrowth"
)

// DataDirName is the user data directory name shared with the Electron
// shell, so sessions in dsh-home stay where existing installs keep them.
const DataDirName = "Deepseek-Harness-Desktop"

// FileName is the configuration file inside the data directory.
const FileName = "config.json"

// DefaultTheme is the theme id of a first run: the deepseek family, which
// follows the OS light/dark preference. That matches the harness install
// default (ui-theme/src/theme-settings.ts DEFAULT_PREFERENCE = "system"); a
// first launch must not force a palette the user did not choose. A stored
// theme always wins, and an invalid one falls back here.
const DefaultTheme = "deepseek"

// DefaultGoPort and DefaultGoRemotePort keep WhaleIsle-MyGo on its own
// loopback service ports so it never collides with the Electron Whale Isle
// instance sharing the same config.json (3080 / 6767).
const (
	DefaultGoPort       = 3180
	DefaultGoRemotePort = 6867
)

// Remote relay topology constants mirrored from
// reference/electron/src/shared/lan.js (hardcoded product defaults).
const (
	defaultRelayEndpoint       = "ayase.cn:443"
	legacyDefaultRelayEndpoint = "125.124.85.212:8411"
	legacyDefaultPublicAppURL  = "http://125.124.85.212:3389/dshd"
	defaultPublicAppBaseURL    = "https://ayase.cn/dshd"
	legacyRelayPort            = "8411"
	remoteBindAddressDefault   = "127.0.0.1"
	dottedQuadPattern          = `^\d{1,3}(\.\d{1,3}){3}$`
	relayEndpointPattern       = `^[A-Za-z0-9.-]+(?::\d{1,5})?$`
	themePattern               = `^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`
)

var (
	dottedQuadRe  = regexp.MustCompile(dottedQuadPattern)
	relayEndpoint = regexp.MustCompile(relayEndpointPattern)
	themeID       = regexp.MustCompile(themePattern)
)

// Config holds the fields the Go shell reads. JSON names and defaults
// mirror DEFAULTS and the normalize* helpers in config.js.
type PluginInstallRegistry struct {
	Kind     string  `json:"kind"`
	Registry *string `json:"registry"`
	URL      string  `json:"url,omitempty"`
}

type Config struct {
	GoPluginInstallRegistry *PluginInstallRegistry `json:"goPluginInstallRegistry,omitempty"`
	GoVoiceMicrophoneID     string                 `json:"goVoiceMicrophoneId,omitempty"`
	GoVoiceMicrophoneLabel  string                 `json:"goVoiceMicrophoneLabel,omitempty"`
	GoOpenInAppChoice       string                 `json:"goOpenInAppChoice,omitempty"`
	// DirectoryPickerUseNative remembers the explicit local Windows picker mode.
	DirectoryPickerUseNative bool `json:"directoryPickerUseNative"`
	// Workspace is the last folder a new session started in.
	Workspace string `json:"workspace"`
	// Host and Port are where dsh web listens; Port is the first port tried.
	Host string `json:"host"`
	Port int    `json:"port"`
	// GoPort and GoRemotePort let the Go shell bind dedicated ports (3180 / 6867)
	// without colliding with or overwriting the Electron shell's 3080 / 6767.
	GoPort       int `json:"goPort,omitempty"`
	GoRemotePort int `json:"goRemotePort,omitempty"`
	// DshBin overrides the dsh launcher (a bin.js or an executable).
	DshBin string `json:"dshBin"`
	// NodeBin overrides the Node.js used to run a bin.js launcher.
	NodeBin string `json:"nodeBin"`
	// DshVersion is the npm version used when dsh is run through npx.
	DshVersion string `json:"dshVersion"`
	// CloseToTray hides the main window instead of quitting.
	CloseToTray bool `json:"closeToTray"`
	// OpenAtLogin starts the shell when the user logs in.
	OpenAtLogin bool `json:"openAtLogin"`
	// OpenDevTools opens the renderer dev tools on boot.
	OpenDevTools bool `json:"openDevTools"`
	// Theme is a theme id matching ^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$.
	Theme string `json:"theme"`
	// AppearanceExplicit records that the user chose a color scheme in the
	// appearance settings. Until then the configured theme stays authoritative
	// even though the Host always reports its default "system" scheme.
	AppearanceExplicit bool `json:"appearanceExplicit"`
	// Locale is the UI language, "zh" or "en".
	Locale string `json:"locale"`

	// HideArchivedList hides only the sidebar archived section; archived data is retained.
	HideArchivedList bool `json:"hideArchivedList"`

	// Remote pairing daemon (phone/LAN remote), gated by the desktop build.
	RemoteEnabled bool `json:"remoteEnabled"`
	RemotePort    int  `json:"remotePort"`
	// RemoteMode is the phone gateway transport, "relay" or "lan". SSH is a
	// remote-workspace transport (RemoteWorkspaceEnabled), never a gateway mode
	// (reference/electron/src/main/config.js:235 keeps only "lan").
	RemoteMode          string `json:"remoteMode"`
	RemoteBindAddress   string `json:"remoteBindAddress"`
	RemoteLanTls        bool   `json:"remoteLanTls"`
	RemoteRelayUrl      string `json:"remoteRelayUrl"`
	RemoteRelayEndpoint string `json:"remoteRelayEndpoint"`
	RemoteRelayUseTls   bool   `json:"remoteRelayUseTls"`
	RemoteAppBaseUrl    string `json:"remoteAppBaseUrl"`

	// Harness supervision: restart dsh after an unexpected exit.
	HarnessAutoRestart        bool `json:"harnessAutoRestart"`
	HarnessRestartMaxAttempts int  `json:"harnessRestartMaxAttempts"`
	HarnessRestartBaseDelayMs int  `json:"harnessRestartBaseDelayMs"`

	// Launcher behaviour: what the desktop process does around dsh startup
	// and updates.
	QuitAfterStart      bool            `json:"quitAfterStart"`
	AutoStartDesktop    bool            `json:"autoStartDesktop"`
	AskOnUpdate         bool            `json:"askOnUpdate"`
	AutoDownloadUpdates bool            `json:"autoDownloadUpdates"`
	DownloadRoute       string          `json:"downloadRoute"` // "", "github" or "cnb"
	DisabledPlugins     []string        `json:"disabledPlugins"`
	Components          json.RawMessage `json:"components,omitempty"`

	// Built-in plugin and assistant toggles.
	DshbotEnabled bool `json:"dshbotEnabled"`
	// ProjectEnabled mounts the optional Project overlay on each start (default off).
	// Disabling never removes retained Project data or its prepared package.
	ProjectEnabled         bool `json:"projectEnabled"`
	WhaleAssistantEnabled  bool `json:"whaleAssistantEnabled"`
	RemoteWorkspaceEnabled bool `json:"remoteWorkspaceEnabled"`

	// SkipUserPlugins starts dsh with --skip-user-plugins (recovery mode).
	// Read from the separate top-level pluginRecovery key, which is carried
	// through unchanged on save (the Electron shell owns its shape).
	SkipUserPlugins bool `json:"-"`
	// SidebarWidth and TerminalHeight keep the Go-shell layout between runs.
	// These are Go-native keys, absent from the Electron DEFAULTS.
	SidebarWidth   float32 `json:"goSidebarWidth"`
	TerminalHeight float32 `json:"goTerminalHeight"`
	// HostTerminalShell is a client-local preference, validated by Host discovery.
	HostTerminalShell string `json:"goHostTerminalShell"`
	// WallpaperPath is an optional local image for the main window backdrop
	// (Go-native; Electron wallpaper lives in ui-theme host settings).
	WallpaperPath string `json:"goWallpaperPath"`
	// HideChatViewTabs hides ConversationRoot「对话/轨迹」tabs (Electron viewTabs=false).
	HideChatViewTabs bool `json:"goHideChatViewTabs"`
	// AutoAllowTools auto-answers approval/request (Electron tool auto-allow).
	AutoAllowTools bool `json:"goAutoAllowTools"`
	// PreviewNoticeAck is the last 预览版说明 copy version the user acknowledged,
	// the Go-side counterpart of the harness welcome-notice settings field.
	PreviewNoticeAck string `json:"goPreviewNoticeAck,omitempty"`
}

var configJSONFields = func() map[string]bool {
	fields := map[string]bool{}
	typ := reflect.TypeFor[Config]()
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			fields[name] = true
		}
	}
	return fields
}()

// mergeKnownConfig replaces all typed fields, including those omitted after an
// explicit reset. Only keys outside Config remain opaque and carried through.
func mergeKnownConfig(raw, known map[string]json.RawMessage) map[string]json.RawMessage {
	merged := make(map[string]json.RawMessage, len(raw)+len(known))
	for key, value := range raw {
		if !configJSONFields[key] {
			merged[key] = value
		}
	}
	for key, value := range known {
		merged[key] = value
	}
	return merged
}

// Defaults returns the configuration of a first run. Values mirror
// DEFAULTS in reference/electron/src/main/config.js, except the theme, which
// starts on the shipped dark palette (DefaultTheme).
func Defaults() Config {
	return Config{
		Host:                      "127.0.0.1",
		Port:                      3080,
		DshVersion:                "0.2.0-rc.2",
		CloseToTray:               true, // DEFAULT_CLOSE_TO_TRAY
		OpenAtLogin:               false,
		OpenDevTools:              false,
		Theme:                     DefaultTheme,
		Locale:                    "zh",
		RemoteEnabled:             false,
		RemotePort:                6767,
		RemoteMode:                "relay",
		RemoteBindAddress:         remoteBindAddressDefault,
		RemoteLanTls:              false,
		RemoteRelayUrl:            "",
		RemoteRelayEndpoint:       "",
		RemoteRelayUseTls:         false,
		RemoteAppBaseUrl:          "",
		HarnessAutoRestart:        true,
		HarnessRestartMaxAttempts: 3,
		HarnessRestartBaseDelayMs: 1000,
		QuitAfterStart:            true,
		AutoStartDesktop:          true,
		AskOnUpdate:               true,
		AutoDownloadUpdates:       true,
		DownloadRoute:             "",
		DshbotEnabled:             false,
		WhaleAssistantEnabled:     true,
		RemoteWorkspaceEnabled:    true,
		SidebarWidth:              280,
		TerminalHeight:            240,
	}
}

// Store loads and saves one config.json, keeping keys it does not know.
type Store struct {
	path string

	mu               sync.Mutex
	raw              map[string]json.RawMessage
	cfg              Config
	cfgSnap          atomic.Pointer[Config]
	baseline         fileBaseline
	shortcutReload   *shortcutDocumentSnapshot
	petRawCache      json.RawMessage
	petStatsRawCache json.RawMessage
	petGrowthCache   petgrowth.State
	petCacheValid    bool
	petGrowthSettled bool
	// shortcutGen advances whenever this store changes the shortcut document.
	// Readers cache their derived values and only reread the file when it does.
	shortcutGen atomic.Uint64
	// firstRun records that no config.json existed when the store opened, so
	// callers can apply first-run appearance without overriding a saved one.
	firstRun bool
	// rename replaces the temp-file rename step; nil means os.Rename. It is
	// a per-instance seam so tests can inject a deterministic write failure.
	rename func(oldpath, newpath string) error
	// commitLock takes the cross-process advisory lock around the final
	// compare-and-replace; nil means acquireCommitLock. It is a per-instance
	// seam so tests can inject a lock failure or skip OS locking entirely.
	commitLock func(path string) (func(), error)
}

// cloneConfig returns a Config whose mutable fields share no storage with c,
// preserving the nil-versus-empty distinction. It is used at every ownership
// boundary (Get, the update callback, and the committed value).
func cloneConfig(c Config) Config {
	out := c
	if c.GoPluginInstallRegistry != nil {
		registry := *c.GoPluginInstallRegistry
		if registry.Registry != nil {
			address := *registry.Registry
			registry.Registry = &address
		}
		out.GoPluginInstallRegistry = &registry
	}
	if c.DisabledPlugins != nil {
		out.DisabledPlugins = make([]string, len(c.DisabledPlugins))
		copy(out.DisabledPlugins, c.DisabledPlugins)
	}
	if c.Components != nil {
		out.Components = make(json.RawMessage, len(c.Components))
		copy(out.Components, c.Components)
	}
	return out
}

// Open reads the configuration in dir, creating nothing until Save.
// A missing file yields the defaults. Keys absent from an existing file
// keep their default value; explicitly stored values (including false and
// zero) are honored as written.
func Open(dir string) (*Store, error) {
	s := &Store{path: filepath.Join(dir, FileName), raw: map[string]json.RawMessage{}}
	s.setCfgLocked(Defaults())
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		s.firstRun = true
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	s.baseline = fileBaseline{exists: true, data: data}
	// A tool- or editor-written config may start with a UTF-8 BOM, which
	// encoding/json rejects as `invalid character '\ufeff' looking for beginning
	// of value`. The bytes on disk are kept in baseline for the next save; only
	// the parse sees the BOM stripped.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(data, &s.raw); err != nil {
		return nil, fmt.Errorf("config: %s: %w", s.path, err)
	}
	if err := unmarshalKnownConfig(data, s.raw, &s.cfg); err != nil {
		return nil, fmt.Errorf("config: %s: %w", s.path, err)
	}
	if recovery, ok := s.raw["pluginRecovery"]; ok {
		var r struct {
			SkipUserPlugins bool `json:"skipUserPlugins"`
		}
		if json.Unmarshal(recovery, &r) == nil {
			s.cfg.SkipUserPlugins = r.SkipUserPlugins
		}
	}
	s.setCfgLocked(normalize(s.cfg))
	return s, nil
}

func (s *Store) setCfgLocked(c Config) {
	s.cfg = c
	snap := cloneConfig(c)
	s.cfgSnap.Store(&snap)
}

// unmarshalKnownConfig decodes top-level Config fields without re-scanning
// large non-Config blobs (such as a multi-megabyte live2dPet food ledger).
func unmarshalKnownConfig(data []byte, raw map[string]json.RawMessage, cfg *Config) error {
	if len(raw["live2dPet"]) < 4096 && len(raw["pet"]) < 4096 && len(raw[workspaceNavigationKey]) < 4096 {
		return json.Unmarshal(data, cfg)
	}
	filtered := make(map[string]json.RawMessage, len(raw))
	for k, v := range raw {
		switch k {
		case "live2dPet", "pet", workspaceNavigationKey:
			continue
		default:
			filtered[k] = v
		}
	}
	small, err := json.Marshal(filtered)
	if err != nil {
		return json.Unmarshal(data, cfg)
	}
	return json.Unmarshal(small, cfg)
}

// Path returns the file the store reads and writes.
func (s *Store) Path() string { return s.path }

// IsFirstRun reports whether config.json was absent when this store opened.
// The shell uses it to apply first-run appearance (dark chrome and the ambient
// backdrop) once, without overriding a profile that has already saved its own
// appearance or theme.
func (s *Store) IsFirstRun() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.firstRun
}

// Get returns a detached snapshot of the current configuration.
func (s *Store) Get() Config {
	if snap := s.cfgSnap.Load(); snap != nil {
		return cloneConfig(*snap)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneConfig(s.cfg)
}

// Update changes the configuration with fn and writes it to disk. Unknown
// keys (pet, live2dPet, pluginRecovery, ...) are carried through unchanged.
//
// Update refuses to write when config.json no longer matches what this store
// last read, returning ErrConflict. It never merges, reloads or retries the
// callback on its own.
func (s *Store) Update(fn func(*Config)) error {
	return s.UpdateChecked(func(c *Config) error {
		fn(c)
		return nil
	})
}

// UpdateChecked validates and changes a detached candidate under the store lock.
// A callback error leaves both the current configuration and the file unchanged.
// The same baseline checks and atomic write used by Update protect the commit.
func (s *Store) UpdateChecked(fn func(*Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Refuse to build on a stale snapshot before running the callback, so a
	// conflicting update has no side effects. The final check and replacement
	// are locked together after the callback has finished.
	if err := s.checkBaseline(); err != nil {
		return err
	}

	candidate := cloneConfig(s.cfg)
	if err := fn(&candidate); err != nil {
		return err
	}
	if candidate.SkipUserPlugins != s.cfg.SkipUserPlugins {
		// SkipUserPlugins is projected from pluginRecovery, never written
		// back as a top-level key.
		candidate.SkipUserPlugins = s.cfg.SkipUserPlugins
	}
	// The committed value must not share mutable storage with anything the
	// callback may still hold.
	next := normalize(cloneConfig(candidate))
	encoded, err := json.Marshal(next)
	if err != nil {
		return err
	}
	var known map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &known); err != nil {
		return err
	}
	merged := mergeKnownConfig(s.raw, known)
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}

	// The final compare-and-replace is serialized across all stores in this
	// process, catching an edit made while the callback or serialization ran.
	unlock, err := s.lockCommit()
	if err != nil {
		return err
	}
	defer unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	if err := writeAtomic(s.path, data, s.rename); err != nil {
		return err
	}
	s.raw = merged
	s.setCfgLocked(next)
	s.baseline = fileBaseline{exists: true, data: data}
	return nil
}

// checkBaseline compares the on-disk file against the recorded baseline. It
// final pre-write check must run under lockCommit so cooperating writers cannot
// change the file between that comparison and replacement.
func (s *Store) checkBaseline() error {
	ok, err := s.baseline.matches(s.path)
	if err != nil {
		return err
	}
	if !ok {
		return ErrConflict
	}
	return nil
}

// normalize mirrors the clamp/validation helpers of config.js:
// normalizeHarnessRecovery, normalizeLauncherSettings, normalizeRemoteConfig,
// normalizeRemoteBindAddress, normalizeRelayEndpoint,
// normalizePublicAppBaseUrl and the theme/locale checks from
// normalizeRendererConfigPatch. Boolean and already-stored values are
// trusted as written (Open prefills defaults for keys absent from the file).
func normalize(c Config) Config {
	d := Defaults()
	if c.Host == "" {
		c.Host = d.Host
	}
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = d.Port
	}
	if c.DshVersion == "" {
		c.DshVersion = d.DshVersion
	}
	if !themeID.MatchString(c.Theme) {
		c.Theme = d.Theme
	}
	if c.Locale != "en" && c.Locale != "zh" {
		c.Locale = d.Locale
	}

	// Remote: port 1024-65535, mode lan|relay, loopback or explicit bind
	// address, relay endpoint resolved against the built-in relay.
	if c.RemotePort < 1024 || c.RemotePort > 65535 {
		c.RemotePort = d.RemotePort
	}
	// Only "lan" is kept; every other value (including a stale "ssh") is the
	// relay transport, exactly as normalizeRemoteConfig does
	// (reference/electron/src/main/config.js:235).
	switch c.RemoteMode {
	case "lan":
	default:
		c.RemoteMode = "relay"
	}
	c.RemoteBindAddress = normalizeBindAddress(c.RemoteBindAddress)
	endpoint := normalizeRelayEndpoint(c.RemoteRelayUrl)
	useBuiltIn := endpoint == "" || endpoint == legacyDefaultRelayEndpoint
	if useBuiltIn {
		endpoint = defaultRelayEndpoint
		c.RemoteRelayUseTls = true
	}
	c.RemoteRelayEndpoint = endpoint
	c.RemoteRelayUrl = endpoint
	c.RemoteAppBaseUrl = normalizePublicAppBaseURL(c.RemoteAppBaseUrl, endpoint)
	// The built-in and legacy default origins are never persisted as the SPA
	// landing host (normalizeRemoteConfig maps them back to "").
	if c.RemoteAppBaseUrl == defaultPublicAppBaseURL || c.RemoteAppBaseUrl == legacyDefaultPublicAppURL {
		c.RemoteAppBaseUrl = ""
	}

	// Harness restart: attempts 1..10, base delay 500..30000 ms.
	if c.HarnessRestartMaxAttempts < 1 || c.HarnessRestartMaxAttempts > 10 {
		c.HarnessRestartMaxAttempts = d.HarnessRestartMaxAttempts
	}
	if c.HarnessRestartBaseDelayMs < 500 || c.HarnessRestartBaseDelayMs > 30000 {
		c.HarnessRestartBaseDelayMs = d.HarnessRestartBaseDelayMs
	}

	// Launcher: download route enum; the alias-removal pass of
	// normalizeDisabledPlugins (dsh-im/dshbot/dsh-whale/dsh-remote/usage-panel/
	// task-control aliases) is Electron-desktop specific and not replicated.
	if c.DownloadRoute == "gitee" {
		c.DownloadRoute = "cnb"
	}
	if c.DownloadRoute != "github" && c.DownloadRoute != "cnb" {
		c.DownloadRoute = ""
	}
	c.DisabledPlugins = normalizeDisabledPlugins(c.DisabledPlugins)

	// Go-shell layout clamps (goSidebarWidth / goTerminalHeight).
	// Sidebar matches ui-layout SIDEBAR_MIN/MAX (264–420 DIP).
	if c.SidebarWidth < 264 {
		c.SidebarWidth = d.SidebarWidth
	}
	if c.SidebarWidth > 420 {
		c.SidebarWidth = 420
	}
	if c.TerminalHeight < 80 {
		c.TerminalHeight = d.TerminalHeight
	}
	return c
}

// normalizeBindAddress mirrors normalizeRemoteBindAddress: the loopback
// default, the explicit wildcard, or one dotted-quad IPv4 with octets
// 0-255; anything else falls back to the default.
func normalizeBindAddress(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "0.0.0.0" {
		return raw
	}
	if !dottedQuadRe.MatchString(raw) {
		return remoteBindAddressDefault
	}
	for _, part := range strings.Split(raw, ".") {
		var octet int
		if _, err := fmt.Sscanf(part, "%d", &octet); err != nil || octet < 0 || octet > 255 {
			return remoteBindAddressDefault
		}
	}
	return raw
}

// normalizeRelayEndpoint mirrors normalizeRelayEndpoint in
// reference/electron/src/shared/lan.js: a hostname[:port] endpoint, scheme
// and path stripped, chisacode.sh rejected, port defaulting to 443.
func normalizeRelayEndpoint(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	lower := raw
	if idx := strings.Index(lower, "://"); idx >= 0 && (lower[:idx] == "http" || lower[:idx] == "https") {
		raw = raw[idx+3:]
	}
	if idx := strings.Index(raw, "/"); idx >= 0 {
		raw = raw[:idx]
	}
	if raw == "" || strings.Contains(strings.ToLower(raw), "chisacode.sh") {
		return ""
	}
	if !relayEndpoint.MatchString(raw) {
		return ""
	}
	if strings.Contains(raw, ":") {
		return raw
	}
	return raw + ":443"
}

// normalizePublicAppBaseURL mirrors normalizePublicAppBaseURL in lan.js:
// a public http(s) SPA origin (+path) for the Away-mode QR; loopback,
// link-local/CGNAT, private-LAN and relay hosts collapse to "".
func normalizePublicAppBaseURL(value, relayEndpoint string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	hostname := u.Hostname()
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return ""
	}
	if isIPv4Hostname(hostname) && (isVirtualOrLinkLocalIPv4(hostname) || isPrivateLANIPv4(hostname)) {
		return ""
	}
	hostPort := hostname
	if u.Port() != "" {
		hostPort += ":" + u.Port()
	}
	if u.Port() == legacyRelayPort || (relayEndpoint != "" && hostPort == relayEndpoint) {
		return ""
	}
	return u.Scheme + "://" + u.Host + strings.TrimSuffix(u.Path, "/")
}

func isIPv4Hostname(hostname string) bool {
	parts := strings.Split(hostname, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) < 1 || len(part) > 3 || !isDigits(part) {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isVirtualOrLinkLocalIPv4 mirrors isVirtualOrLinkLocalIpv4: link-local
// 169.254.0.0/16 and CGNAT 100.64.0.0/10.
func isVirtualOrLinkLocalIPv4(address string) bool {
	a, b, ok := octets(address)
	if !ok {
		return true
	}
	if a == 169 && b == 254 {
		return true
	}
	if a == 100 && b >= 64 && b <= 127 {
		return true
	}
	return false
}

// isPrivateLANIPv4 mirrors isPrivateLanIpv4: RFC1918 space.
func isPrivateLANIPv4(address string) bool {
	a, b, ok := octets(address)
	if !ok {
		return false
	}
	if a == 10 {
		return true
	}
	if a == 192 && b == 168 {
		return true
	}
	if a == 172 && b >= 16 && b <= 31 {
		return true
	}
	return false
}

func octets(address string) (first, second int, ok bool) {
	parts := strings.Split(address, ".")
	if len(parts) != 4 {
		return 0, 0, false
	}
	nums := make([]int, 4)
	for i, part := range parts {
		n := 0
		if part == "" || !isDigits(part) {
			return 0, 0, false
		}
		for _, r := range part {
			n = n*10 + int(r-'0')
		}
		nums[i] = n
	}
	return nums[0], nums[1], true
}

// normalizeDisabledPlugins trims, drops empties and dedupes, mirroring the
// original desktop config, including its built-in Market alias removal.
func normalizeDisabledPlugins(list []string) []string {
	seen := make(map[string]struct{}, len(list))
	out := make([]string, 0, len(list))
	for _, name := range list {
		name = strings.TrimSpace(name)
		if name == "" || name == "dshmarket" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
