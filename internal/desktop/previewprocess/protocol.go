// Package previewprocess contains the opt-in, out-of-process HTML preview.
// It does not enable or replace the default static document preview.
package previewprocess

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	protocolVersion = 1
	maxHeaderBytes = 4096
	maxHTMLBytes = 25 * 1024 * 1024
)

var ErrDisabled = errors.New("interactive preview helper is disabled")

// Options is supplied by trusted application code, never by document content.
// HelperPath must identify the separately built, trusted whaleisle-preview.exe.
type Options struct {
	Enabled bool
	HelperPath string
}

// Document contains an already authorized, resource-frozen snapshot. This
// package never opens workspace paths or resolves document resource URLs.
type Document struct {
	HTML string
	Width int
	Appearance string
}

// Session owns the entire helper job until Close, cancellation, process exit,
// a failed containment check, or the thirty-minute lifetime limit.
type Session struct {
	done chan struct{}
	once sync.Once
	closeFn func() error
	err error
}

func (s *Session) Done() <-chan struct{} { return s.done }
func (s *Session) Close() error {
	s.once.Do(func() { s.err = s.closeFn(); close(s.done) })
	return s.err
}

// OpenExperimental requires explicit opt-in. Success means this particular
// process tree passed the OS token/job gate and loaded its document. It does
// not change htmlpreview.Readiness, nor authorize in-process script execution.
func OpenExperimental(ctx context.Context, opt Options, doc Document) (*Session, error) {
	if !opt.Enabled { return nil, ErrDisabled }
	if err := ctx.Err(); err != nil { return nil, err }
	if err := validateDocument(doc); err != nil { return nil, err }
	if strings.TrimSpace(opt.HelperPath) == "" { return nil, errors.New("missing trusted preview helper") }
	return openPlatform(ctx, opt, doc)
}

func validateDocument(d Document) error {
	if len(d.HTML) == 0 || len(d.HTML) > maxHTMLBytes || !utf8.ValidString(d.HTML) || strings.TrimSpace(d.HTML) == "" {
		return errors.New("invalid prepared HTML snapshot")
	}
	if d.Width < 240 || d.Width > 1600 || (d.Appearance != "light" && d.Appearance != "dark") {
		return errors.New("invalid preview dimensions or appearance")
	}
	return nil
}

type packet struct {
	Version int `json:"version"`
	Kind string `json:"kind"`
	Nonce string `json:"nonce"`
	SID string `json:"sid,omitempty"`
	HelperPID uint32 `json:"helperPID,omitempty"`
	BrowserPID uint32 `json:"browserPID,omitempty"`
	Bytes int `json:"bytes,omitempty"`
	Width int `json:"width,omitempty"`
	Appearance string `json:"appearance,omitempty"`
}

func writePacket(w io.Writer, p packet) error {
	raw, err := json.Marshal(p)
	if err != nil { return err }
	if len(raw) == 0 || len(raw) > maxHeaderBytes { return errors.New("invalid preview protocol header") }
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], uint32(len(raw)))
	if err := writeAll(w, size[:]); err != nil { return err }
	return writeAll(w, raw)
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) != 0 {
		n, err := w.Write(data)
		if n < 0 || n > len(data) { return errors.New("invalid writer count") }
		data = data[n:]
		if err != nil { return err }
		if n == 0 { return io.ErrShortWrite }
	}
	return nil
}

func readPacket(r io.Reader) (packet, error) {
	var p packet
	var size [4]byte
	if _, err := io.ReadFull(r, size[:]); err != nil { return p, err }
	n := binary.LittleEndian.Uint32(size[:])
	if n == 0 || n > maxHeaderBytes { return p, errors.New("invalid preview protocol header size") }
	raw := make([]byte, int(n))
	if _, err := io.ReadFull(r, raw); err != nil { return p, err }
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil { return p, err }
	if err := decoder.Decode(new(any)); err != io.EOF { return p, errors.New("trailing preview protocol data") }
	return p, nil
}

func expect(p packet, kind, nonce string) error {
	if p.Version != protocolVersion || p.Kind != kind || p.Nonce != nonce || len(nonce) != 64 {
		return fmt.Errorf("unexpected preview protocol phase %q", p.Kind)
	}
	return nil
}

func sendDocument(w io.Writer, nonce string, d Document) error {
	if err := validateDocument(d); err != nil { return err }
	if err := writePacket(w, packet{Version: protocolVersion, Kind: "document", Nonce: nonce, Bytes: len(d.HTML), Width: d.Width, Appearance: d.Appearance}); err != nil { return err }
	return writeAll(w, []byte(d.HTML))
}

func receiveDocument(r io.Reader, nonce string) (Document, error) {
	p, err := readPacket(r)
	if err != nil { return Document{}, err }
	if err := expect(p, "document", nonce); err != nil { return Document{}, err }
	if p.Bytes < 1 || p.Bytes > maxHTMLBytes { return Document{}, errors.New("invalid document byte count") }
	body := make([]byte, p.Bytes)
	if _, err := io.ReadFull(r, body); err != nil { return Document{}, err }
	d := Document{HTML: string(body), Width: p.Width, Appearance: p.Appearance}
	return d, validateDocument(d)
}
