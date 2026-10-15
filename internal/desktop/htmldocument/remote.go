package htmldocument

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"time"
	"unicode/utf8"
	"whaleisle/internal/desktop/htmlpreview"
)

// OpenBytes serves a static Host snapshot. Relative dependencies remain unavailable;
// the browser receives neither Host credentials nor filesystem/network authority.
func OpenBytes(path string, data []byte, valid func() bool) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("HTML request is no longer active")
	}
	if len(data) == 0 || len(data) > MaxBytes || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return nil, errors.New("HTML preview requires a nonempty UTF-8 document within 8 MiB")
	}
	data = []byte(htmlpreview.DocumentShell(string(data)))
	if len(data) > htmlpreview.MaxPreparedHTMLBytes {
		return nil, errors.New("HTML document is too large after sandbox encoding")
	}
	resources := ResourceReport{Bytes: len(data), Skipped: 1, Warnings: []string{"远端静态快照；相对资源和脚本未载入"}}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	d := &Document{Name: filepath.Base(path), Resources: resources, data: data, done: make(chan struct{}), origin: "http://" + listener.Addr().String(), route: "/" + hex.EncodeToString(secret) + "/document.html", valid: valid}
	d.server = &http.Server{Handler: http.HandlerFunc(d.serve), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	d.timer = time.AfterFunc(Lifetime, func() { d.Close() })
	go func() { _ = d.server.Serve(listener) }()
	return d, nil
}
