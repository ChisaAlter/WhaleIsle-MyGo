package mediapreview

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Lifetime = 30 * time.Minute

type Document struct {
	Name, Kind          string
	mime, origin, route string
	data                []byte
	valid               func() bool
	server              *http.Server
	timer               *time.Timer
	deadline            time.Time
	done                chan struct{}
	mu                  sync.RWMutex
	closed              bool
}

func Open(name string, data []byte, valid func() bool) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("media Host/session is no longer active")
	}
	kind, mime, err := Validate(name, data)
	if err != nil {
		return nil, err
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	d := &Document{Name: path.Base(strings.ReplaceAll(name, `\`, "/")), Kind: kind, mime: mime, origin: "http://" + listener.Addr().String(), route: "/" + hex.EncodeToString(secret), data: append([]byte(nil), data...), valid: valid, deadline: time.Now().Add(Lifetime), done: make(chan struct{})}
	d.server = &http.Server{Handler: http.HandlerFunc(d.serve), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	d.timer = time.AfterFunc(Lifetime, d.Close)
	go func() { _ = d.server.Serve(listener) }()
	return d, nil
}
func (d *Document) URL() string           { return d.origin + d.route + "/player.html" }
func (d *Document) MediaURL() string      { return d.origin + d.route + "/media" }
func (d *Document) Done() <-chan struct{} { return d.done }
func (d *Document) Valid() bool {
	if d == nil {
		return false
	}
	d.mu.RLock()
	closed := d.closed
	d.mu.RUnlock()
	if closed {
		return false
	}
	if !time.Now().Before(d.deadline) || d.valid == nil || !d.valid() {
		d.Close()
		return false
	}
	return true
}
func (d *Document) Close() {
	if d == nil {
		return
	}
	if d.server != nil {
		_ = d.server.Close()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	d.closed = true
	d.data = nil
	close(d.done)
	if d.timer != nil {
		d.timer.Stop()
	}
}
func (d *Document) csp() string {
	return "default-src 'none'; script-src 'none'; style-src 'unsafe-inline'; media-src " + d.MediaURL() + "; connect-src 'none'; img-src 'none'; font-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
}
func (d *Document) shell() string {
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="referrer" content="no-referrer"><style>html,body{margin:0;width:100%;height:100%;background:#202124;color:#ddd;font:13px sans-serif}body{display:flex;flex-direction:column}p{padding:8px 12px;margin:0}main{display:flex;align-items:center;justify-content:center;flex:1;min-height:0}video{max-width:100%;max-height:100%}audio{width:90%}</style></head><body><p>` + html.EscapeString(d.Name) + ` · 只读媒体快照 / Read-only media snapshot</p><main><` + d.Kind + ` controls preload="metadata" controlslist="nodownload noremoteplayback" disableremoteplayback disablepictureinpicture src="` + d.MediaURL() + `"></` + d.Kind + `></main><p>最大 32 MiB，30 分钟有效。编解码器由当前 WebView2 决定；若无法播放，请关闭并使用 Files 的外部打开。 / Maximum 32 MiB; expires in 30 minutes. Codec support depends on WebView2. If playback fails, close and use Open external in Files.</p></body></html>`
}
func (d *Document) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-DNS-Prefetch-Control", "off")
	w.Header().Set("Content-Security-Policy", d.csp())
	if !d.Valid() {
		http.Error(w, "Media preview expired", http.StatusGone)
		return
	}
	if r.Host != strings.TrimPrefix(d.origin, "http://") || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || (r.URL.Path != d.route+"/player.html" && r.URL.Path != d.route+"/media") || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != d.origin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed {
		http.Error(w, "Media preview expired", http.StatusGone)
		return
	}
	if r.URL.Path == d.route+"/player.html" {
		body := d.shell()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		if r.Method == "GET" {
			_, _ = w.Write([]byte(body))
		}
		return
	}
	size := len(d.data)
	start, end := 0, size-1
	status := http.StatusOK
	if values := r.Header.Values("Range"); len(values) > 1 {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
		http.Error(w, "Only one byte range is supported", 416)
		return
	}
	if raw := r.Header.Get("Range"); raw != "" {
		var ok bool
		start, end, ok = singleRange(raw, size)
		if !ok {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
			http.Error(w, "Invalid byte range", 416)
			return
		}
		status = http.StatusPartialContent
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	}
	w.Header().Set("Content-Type", d.mime)
	w.Header().Set("Content-Disposition", `inline; filename="media`+extension(d.Name)+`"`)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
	w.WriteHeader(status)
	if r.Method == "GET" {
		_, _ = w.Write(d.data[start : end+1])
	}
}
func singleRange(raw string, size int) (int, int, bool) {
	if !strings.HasPrefix(raw, "bytes=") || strings.Contains(raw, ",") {
		return 0, 0, false
	}
	a, b, ok := strings.Cut(strings.TrimPrefix(raw, "bytes="), "-")
	if !ok {
		return 0, 0, false
	}
	number := func(s string) (int, bool) {
		if s == "" || strings.Trim(s, "0123456789") != "" {
			return 0, false
		}
		n, err := strconv.ParseUint(s, 10, 63)
		return int(n), err == nil && n <= uint64(MaxBytes)
	}
	if a == "" {
		n, ok := number(b)
		if !ok || n <= 0 {
			return 0, 0, false
		}
		return max(0, size-n), size - 1, true
	}
	start, ok := number(a)
	if !ok || start >= size {
		return 0, 0, false
	}
	end := size - 1
	if b != "" {
		end, ok = number(b)
		if !ok || end < start || end >= size {
			return 0, 0, false
		}
	}
	return start, end, true
}
