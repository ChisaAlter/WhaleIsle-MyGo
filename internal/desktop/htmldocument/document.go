// Package htmldocument grants read-only access to one explicitly selected HTML
// snapshot. It never serves a directory, follows a requested filesystem path,
// proxies a Host, or receives account credentials.
package htmldocument

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"whaleisle/internal/desktop/htmlpreview"
)

const MaxBytes = 8 << 20
const Lifetime = 30 * time.Minute

type Document struct {
	Name      string
	Resources ResourceReport
	path      string
	identity  os.FileInfo
	data      []byte
	// Source bytes and directory identity belong to the original authorized read.
	// Interactive dependencies are frozen only after a separate user gesture.
	source, root, relative string
	remoteInteractive      func(context.Context) (string, ResourceReport, error)
	rootIdentity           os.FileInfo
	done                   chan struct{}
	origin, route          string
	valid                  func() bool
	server                 *http.Server
	timer                  *time.Timer
	mu                     sync.RWMutex
	closed                 bool
}

// OpenWorkspace confines reads with os.Root, including symlink/junction races.
// root is the initiating Session's authorized cwd, not the configured fallback.
func OpenWorkspace(root, raw string, valid func() bool) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("HTML request is no longer active")
	}
	if !filepath.IsAbs(root) || strings.TrimSpace(raw) == "" || strings.Contains(raw, "://") {
		return nil, errors.New("HTML requires the selected session's local workspace")
	}
	full := raw
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, raw)
	}
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, errors.New("HTML is outside the session workspace")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	info, err := r.Stat(rel)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("HTML is missing or is not a regular workspace file")
	}
	f, err := r.Open(rel)
	if err != nil {
		return nil, errors.New("HTML cannot be opened within the session workspace")
	}
	defer f.Close()
	return snapshot(f, r, rel, full, valid)
}

func snapshot(f *os.File, root *os.Root, relative, path string, valid func() bool) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("HTML request no longer belongs to the active session")
	}
	if !supported(path) {
		return nil, errors.New("only HTML documents can use this preview")
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("HTML is not a regular file")
	}
	if info.Size() < 1 || info.Size() > MaxBytes {
		return nil, fmt.Errorf("HTML preview limit is %d MiB", MaxBytes>>20)
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) || int64(len(data)) != info.Size() {
		return nil, errors.New("HTML changed while reading; open it again")
	}
	if len(data) == 0 || len(data) > MaxBytes || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return nil, errors.New("HTML preview requires a nonempty UTF-8 document")
	}
	rootIdentity, err := root.Stat(".")
	if err != nil {
		return nil, err
	}
	source := string(data)
	// Freeze only declared, bounded static dependencies before publishing. The
	// browser receives no filesystem reader, asset routes or directory authority.
	prepared, resources, err := prepareResources(root, relative, string(data), valid)
	if err != nil {
		return nil, err
	}
	data = []byte(htmlpreview.DocumentShell(prepared))
	if len(data) > htmlpreview.MaxPreparedHTMLBytes {
		return nil, errors.New("HTML document is too large after sandbox encoding")
	}
	if !valid() {
		return nil, errors.New("HTML session changed while reading")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	d := &Document{Name: filepath.Base(path), Resources: resources, path: path, identity: info, data: data, source: source, root: root.Name(), relative: relative, rootIdentity: rootIdentity, done: make(chan struct{}), origin: "http://" + listener.Addr().String(), route: "/" + hex.EncodeToString(secret) + "/document.html", valid: valid}
	d.server = &http.Server{Handler: http.HandlerFunc(d.serve), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	d.timer = time.AfterFunc(Lifetime, func() { d.Close() })
	go func() { _ = d.server.Serve(listener) }()
	return d, nil
}
func (d *Document) URL() string { return d.origin + d.route }
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
	if !d.valid() {
		d.Close()
		return false
	}
	return true
}
func (d *Document) AllowsURL(raw string) bool {
	u, err := url.Parse(raw)
	return d.Valid() && err == nil && u.User == nil && u.Scheme+"://"+u.Host == d.origin && u.Path == d.route && u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery
}
func (d *Document) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-DNS-Prefetch-Control", "off")
	if !d.Valid() {
		http.Error(w, "Preview expired", http.StatusGone)
		return
	}
	if r.Host != strings.TrimPrefix(d.origin, "http://") || r.URL.Path != d.route || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != d.origin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed {
		http.Error(w, "Preview expired", http.StatusGone)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="document.html"`)
	w.Header().Set("Content-Security-Policy", htmlpreview.DocumentCSP)
	http.ServeContent(w, r, "document.html", time.Time{}, bytes.NewReader(d.data))
}

// ExternalPath is only exposed to a separate explicit system-open gesture.
func (d *Document) ExternalPath() (string, error) {
	if d == nil || !d.valid() {
		return "", errors.New("HTML preview owner changed; reopen the document")
	}
	if d.identity == nil {
		return "", errors.New("Remote HTML: use Files external open on the Host")
	}
	current, err := os.Stat(d.path)
	if err != nil || !os.SameFile(d.identity, current) || current.Size() != d.identity.Size() || !current.ModTime().Equal(d.identity.ModTime()) {
		return "", errors.New("HTML source changed; reopen it before using the system viewer")
	}
	return d.path, nil
}
func (d *Document) Close() {
	if d == nil {
		return
	}
	// Close sockets before clearing bytes: a streaming reader holds the read lock.
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
	d.source = ""
	d.remoteInteractive = nil
	if d.done != nil {
		close(d.done)
	}
	if d.timer != nil {
		d.timer.Stop()
	}
}

func supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".htm", ".html", ".xhtml":
		return true
	}
	return false
}
