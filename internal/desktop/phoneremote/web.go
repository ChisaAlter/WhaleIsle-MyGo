package phoneremote

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// MobileWebPort is the fixed SPA port used by the reference desktop shell.
const MobileWebPort = 3180

var mobileWebMIME = map[string]string{
	".html":  "text/html; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".svg":   "image/svg+xml",
	".json":  "application/json",
	".map":   "application/json",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".wasm":  "application/wasm",
}

// MobileWebAsset is one safely resolved SPA asset.
type MobileWebAsset struct {
	File string
	Type string
}

// MobileWebServerOptions configures CreateMobileWebServer.
type MobileWebServerOptions struct {
	Root        string
	BindAddress string
	Port        int
}

// ResolveMobileWebRoot locates the copied phone SPA. An explicit root wins;
// otherwise WHALEISLE_MOBILE_WEB, the executable's resources directory, and
// the current working directory are checked in that order.
func ResolveMobileWebRoot(root ...string) string {
	if len(root) > 0 && strings.TrimSpace(root[0]) != "" {
		return filepath.Clean(root[0])
	}
	if env := strings.TrimSpace(os.Getenv("WHALEISLE_MOBILE_WEB")); env != "" {
		return filepath.Clean(env)
	}
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "resources", "mobile", "web"),
			filepath.Join(exeDir, "mobile", "web"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "resources", "mobile", "web"),
			filepath.Join(cwd, "mobile", "web"),
		)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return filepath.Join("resources", "mobile", "web")
}

// ShouldProxyToHost reports whether a request belongs to the authenticated
// host API instead of the static SPA. The mobile server itself serves static
// assets only; callers can use this predicate when composing a gateway.
func ShouldProxyToHost(path string) bool {
	pathname := strings.SplitN(path, "?", 2)[0]
	return pathname == "/api" || strings.HasPrefix(pathname, "/api/")
}

// ResolveSPAAsset performs the same extension/fallback and traversal checks as
// the reference mobile-web.js. It decodes a URL path exactly once.
func ResolveSPAAsset(root, requestPath string) (MobileWebAsset, bool, error) {
	if strings.TrimSpace(root) == "" {
		return MobileWebAsset{}, false, errors.New("mobile web root required")
	}
	pathPart := strings.SplitN(requestPath, "?", 2)[0]
	if pathPart == "" {
		pathPart = "/"
	}
	decoded, err := url.PathUnescape(pathPart)
	if err != nil {
		return MobileWebAsset{}, false, fmt.Errorf("invalid mobile web path: %w", err)
	}
	if strings.IndexByte(decoded, 0) >= 0 {
		return MobileWebAsset{}, false, errors.New("invalid mobile web path")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return MobileWebAsset{}, false, err
	}
	rel := strings.TrimLeft(decoded, `/\\`)
	candidate := filepath.Clean(filepath.Join(rootAbs, filepath.FromSlash(rel)))
	inside, err := filepath.Rel(rootAbs, candidate)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) || filepath.IsAbs(inside) {
		return MobileWebAsset{}, false, nil
	}
	if strings.HasSuffix(strings.ToLower(candidate), ".test.js") {
		return MobileWebAsset{}, false, nil
	}
	if filepath.Ext(filepath.Base(decoded)) == "" {
		candidate = filepath.Join(rootAbs, "index.html")
	}
	info, err := os.Stat(candidate)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return MobileWebAsset{}, false, nil
		}
		return MobileWebAsset{}, false, err
	}
	if !info.Mode().IsRegular() {
		return MobileWebAsset{}, false, nil
	}
	typ := mobileWebMIME[strings.ToLower(filepath.Ext(candidate))]
	if typ == "" {
		typ = "application/octet-stream"
	}
	return MobileWebAsset{File: candidate, Type: typ}, true, nil
}

// CreateMobileWebServer returns an HTTP server serving the mobile pairing SPA.
// The offer stays in the URL fragment and therefore never reaches this
// server. GET and HEAD are supported; all other methods receive 405.
func CreateMobileWebServer(options ...MobileWebServerOptions) *http.Server {
	var opts MobileWebServerOptions
	if len(options) > 0 {
		opts = options[0]
	}
	root := ResolveMobileWebRoot(opts.Root)
	bind := strings.TrimSpace(opts.BindAddress)
	if bind == "" {
		bind = "127.0.0.1"
	}
	port := opts.Port
	if port <= 0 {
		port = MobileWebPort
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.Host != "" {
			if _, err := url.Parse("http://" + r.Host); err != nil {
				writeMobileError(w, http.StatusBadRequest, "Bad request")
				return
			}
		}
		asset, ok, err := ResolveSPAAsset(root, requestPath(r))
		if err != nil {
			writeMobileError(w, http.StatusBadRequest, "Bad request")
			return
		}
		if !ok {
			writeMobileError(w, http.StatusNotFound, "Not found")
			return
		}
		headers := mobileWebHeaders(asset.Type)
		for key, value := range headers {
			w.Header().Set(key, value)
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("Method not allowed"))
			return
		}
		body, readErr := os.ReadFile(asset.File)
		if readErr != nil {
			writeMobileError(w, http.StatusNotFound, "Not found")
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
	return &http.Server{Addr: net.JoinHostPort(bind, fmt.Sprintf("%d", port)), Handler: h}
}

// ListenMobileWebServer binds server exactly once and starts serving in a
// goroutine. The returned listener lets callers close the server during remote
// shutdown, while bind errors are returned synchronously.
func ListenMobileWebServer(server *http.Server, bindAddress string, port int) (net.Listener, error) {
	if server == nil {
		return nil, errors.New("mobile web server required")
	}
	if port <= 0 {
		port = MobileWebPort
	}
	if strings.TrimSpace(bindAddress) == "" {
		bindAddress = "127.0.0.1"
	}
	server.Addr = net.JoinHostPort(bindAddress, fmt.Sprintf("%d", port))
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, err
	}
	go func() {
		_ = server.Serve(listener)
	}()
	return listener, nil
}

// CloseMobileWebServer stops a server returned by CreateMobileWebServer. It
// is idempotent for a nil server and maps directly to the reference
// stopMobileWebServer lifecycle.
func CloseMobileWebServer(server *http.Server) error {
	if server == nil {
		return nil
	}
	return server.Close()
}

func requestPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "/"
	}
	if raw := r.URL.EscapedPath(); raw != "" {
		return raw
	}
	return r.URL.Path
}

func mobileWebHeaders(contentType string) map[string]string {
	cache := "no-cache"
	if strings.Contains(contentType, "html") {
		cache = "no-store"
	}
	return map[string]string{
		"content-type":           contentType,
		"cache-control":          cache,
		"x-content-type-options": "nosniff",
		"referrer-policy":        "no-referrer",
		"x-frame-options":        "DENY",
	}
}

func writeMobileError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Header().Set("x-content-type-options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
}
