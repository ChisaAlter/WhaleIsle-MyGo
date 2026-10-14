// Package pdfpreview grants read-only access to one explicitly selected PDF
// snapshot. It never serves a directory, follows a requested filesystem path,
// proxies a Host, or receives account credentials.
package pdfpreview

import (
	"bytes"
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
)

const MaxBytes = 32 << 20
const Lifetime = 30 * time.Minute

type Document struct {
	Name          string
	path          string
	identity      os.FileInfo
	data          []byte
	origin, route string
	valid         func() bool
	server        *http.Server
	timer         *time.Timer
	mu            sync.RWMutex
	closed        bool
}

// OpenWorkspace confines reads with os.Root, including symlink/junction races.
// root is the initiating Session's authorized cwd, not the configured fallback.
func OpenWorkspace(root, raw string, valid func() bool) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("PDF request is no longer active")
	}
	if !filepath.IsAbs(root) || strings.TrimSpace(raw) == "" || strings.Contains(raw, "://") {
		return nil, errors.New("PDF requires the selected session's local workspace")
	}
	full := raw
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, raw)
	}
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, errors.New("PDF is outside the session workspace")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	f, err := r.Open(rel)
	if err != nil {
		return nil, errors.New("PDF cannot be opened within the session workspace")
	}
	defer f.Close()
	return snapshot(f, full, valid)
}

// OpenGenerated is only for the exact result of a still-current authenticated
// Office conversion request. It does not grant access to that result's directory.
func OpenGenerated(path string, valid func() bool) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("PDF conversion request is no longer active")
	}
	if !filepath.IsAbs(path) {
		return nil, errors.New("conversion returned no local PDF path")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("conversion PDF is missing or is not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("conversion PDF changed before opening")
	}
	return snapshot(f, path, valid)
}

func snapshot(f *os.File, path string, valid func() bool) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("PDF request no longer belongs to the active session")
	}
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return nil, errors.New("only PDF documents can use this preview")
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("PDF is not a regular file")
	}
	if info.Size() < 8 || info.Size() > MaxBytes {
		return nil, fmt.Errorf("PDF preview limit is %d MiB", MaxBytes>>20)
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) || int64(len(data)) != info.Size() {
		return nil, errors.New("PDF changed while reading; open it again")
	}
	// Do not give renamed HTML a local document origin. The native PDF renderer
	// still validates the PDF structure; scripts/network are disabled separately.
	tail := bytes.TrimSpace(data)
	if len(data) > MaxBytes || !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 8 || (data[5] != '1' && data[5] != '2') || data[6] != '.' || data[7] < '0' || data[7] > '9' || !bytes.HasSuffix(tail, []byte("%%EOF")) {
		return nil, errors.New("file is not a supported PDF document")
	}
	if !valid() {
		return nil, errors.New("PDF session changed while reading")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	d := &Document{Name: filepath.Base(path), path: path, identity: info, data: data, origin: "http://" + listener.Addr().String(), route: "/" + hex.EncodeToString(secret) + "/document.pdf", valid: valid}
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
	return d.Valid() && err == nil && u.User == nil && u.Scheme+"://"+u.Host == d.origin && u.Path == d.route && u.RawPath == "" && u.RawQuery == ""
}
func (d *Document) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !d.Valid() {
		http.Error(w, "Preview expired", http.StatusGone)
		return
	}
	if r.Host != strings.TrimPrefix(d.origin, "http://") || r.URL.Path != d.route || r.URL.RawPath != "" || r.URL.RawQuery != "" || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != d.origin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
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
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="document.pdf"`)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'none'; connect-src 'none'; form-action 'none'; base-uri 'none'; frame-ancestors 'none'")
	http.ServeContent(w, r, "document.pdf", time.Time{}, bytes.NewReader(d.data))
}

// ExternalPath is only exposed to a separate explicit system-open gesture.
func (d *Document) ExternalPath() (string, error) {
	if d == nil || !d.valid() {
		return "", errors.New("PDF preview owner changed; reopen the document")
	}
	current, err := os.Stat(d.path)
	if err != nil || !os.SameFile(d.identity, current) || current.Size() != d.identity.Size() || !current.ModTime().Equal(d.identity.ModTime()) {
		return "", errors.New("PDF source changed; reopen it before using the system viewer")
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
	if d.timer != nil {
		d.timer.Stop()
	}
}
