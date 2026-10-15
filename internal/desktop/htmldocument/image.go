package htmldocument

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/desktop/htmlpreview"
	"whaleisle/internal/desktop/imagepreview"
)

// CanRunScripts is false for inert image snapshots. InteractiveSnapshot also
// rejects their absent HTML source/root identity, independently of this UI gate.
func (d *Document) CanRunScripts() bool {
	if d == nil {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return !d.closed && d.source != "" && d.rootIdentity != nil && d.identity != nil
}

// OpenImageWorkspace snapshots only the explicitly selected local image. The
// file is opened through os.Root, bounded, checked for changes, then detached
// from the filesystem. No script/resource preparation path is used.
func OpenImageWorkspace(root, raw string, valid func() bool) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("image request is no longer active")
	}
	if !filepath.IsAbs(root) || strings.TrimSpace(raw) == "" || strings.Contains(raw, "://") || !imagepreview.Supports(raw) {
		return nil, errors.New("image preview requires a supported local session file")
	}
	full := raw
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, raw)
	}
	rel, err := filepath.Rel(root, full)
	if err != nil || outsideAssetRoot(rel) {
		return nil, errors.New("image is outside the session workspace")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	f, err := r.Open(rel)
	if err != nil {
		return nil, errors.New("image cannot be opened within the session workspace")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > imagepreview.MaxBytes {
		return nil, errors.New("image must be a regular file within the 16 MiB byte budget")
	}
	data, err := io.ReadAll(io.LimitReader(f, imagepreview.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) || int64(len(data)) != info.Size() {
		return nil, errors.New("image changed while reading; open it again")
	}
	markup, err := imagepreview.Build(full, data)
	if err != nil {
		return nil, err
	}
	encoded := []byte(htmlpreview.DocumentShell(markup))
	if len(encoded) > htmlpreview.MaxPreparedHTMLBytes {
		return nil, errors.New("image exceeds the sandbox snapshot byte budget")
	}
	if !valid() {
		return nil, errors.New("image session changed while reading")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	// source/root identity intentionally stay empty: images can never be
	// promoted into the separately authorized JavaScript helper.
	d := &Document{
		Name: filepath.Base(full), Resources: ResourceReport{Files: 1, Bytes: len(data)},
		path: full, identity: info, data: encoded, done: make(chan struct{}),
		origin: "http://" + listener.Addr().String(),
		route:  "/" + hex.EncodeToString(secret) + "/document.html", valid: valid,
	}
	d.server = &http.Server{Handler: http.HandlerFunc(d.serve), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	d.timer = time.AfterFunc(Lifetime, func() { d.Close() })
	go func() { _ = d.server.Serve(listener) }()
	return d, nil
}
