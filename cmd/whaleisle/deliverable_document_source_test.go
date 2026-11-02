package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	browserpreviewui "whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/files"
)

func deliveryDocumentFixtureFile(t *testing.T, f *deliveryScopeFixture, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(f.cwd, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func deliveryDocumentWAV() []byte {
	b := make([]byte, 46)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], 38)
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], 8000)
	binary.LittleEndian.PutUint32(b[28:], 16000)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], 2)
	return b
}
func closeDeliveryResult(r browserpreviewui.DocumentResult) {
	if r.PDF != nil {
		r.PDF.Close()
	}
	if r.HTML != nil {
		r.HTML.Close()
	}
	if r.Media != nil {
		r.Media.Close()
	}
}

func TestLocalDeliveryDocumentReadsOnlyExactWorkspaceSource(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind files.DeliveryPreviewKind
		data []byte
	}{{"report.pdf", files.DeliveryPreviewPDF, []byte("%PDF-1.7\nlocal snapshot\n%%EOF")}, {"page.html", files.DeliveryPreviewHTML, []byte("<!doctype html><h1>Saved source</h1><script>fetch('https://invalid')</script>")}, {"sound.wav", files.DeliveryPreviewMedia, deliveryDocumentWAV()}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDeliveryScopeFixture(t)
			path := deliveryDocumentFixtureFile(t, f, tc.name, tc.data)
			source, err := f.state.localDeliverableDocumentSource(f.owner, path, tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			if !source.LocalDelivery || !source.LocalDeliveryValid() || source.AddressKind != localDeliveryAddress {
				t.Fatal("missing live local-only source")
			}
			result := f.state.readWorkspaceDocument(context.Background(), source)
			defer closeDeliveryResult(result)
			if result.Err != nil {
				t.Fatal(result.Err)
			}
			switch tc.kind {
			case files.DeliveryPreviewPDF:
				if result.PDF == nil || !result.PDF.Valid() {
					t.Fatal("no PDF")
				}
			case files.DeliveryPreviewHTML:
				if result.HTML == nil || !result.HTML.Valid() {
					t.Fatal("no HTML")
				}
			case files.DeliveryPreviewMedia:
				if result.Media == nil || !result.Media.Valid() {
					t.Fatal("no media")
				}
			}
			// Both persisted and missing-token routes remain local denials, even
			// with the same current owner. The HTTP client fixture rejects all RPC.
			raw, err := json.Marshal(source)
			if err != nil {
				t.Fatal(err)
			}
			var restored browserpreviewui.DocumentSource
			if err = json.Unmarshal(raw, &restored); err != nil {
				t.Fatal(err)
			}
			if f.state.workspaceDocumentReady(restored) {
				t.Fatal("descriptor resurrected permission")
			}
			denied := f.state.readWorkspaceDocument(context.Background(), restored)
			defer closeDeliveryResult(denied)
			if denied.Err == nil || denied.PDF != nil || denied.HTML != nil || denied.Media != nil {
				t.Fatal("missing permit loaded content")
			}
			f.store.Select("other")
			f.store.Select("same-id")
			if source.LocalDeliveryValid() || result.Valid() {
				t.Fatal("selection ABA retained document authority")
			}
			if result.PDF != nil && result.PDF.Valid() || result.HTML != nil && result.HTML.Valid() || result.Media != nil && result.Media.Valid() {
				t.Fatal("buffered document retained old selection")
			}
		})
	}
}

func TestLocalDeliveryDocumentHostRestartAndSourceReplacement(t *testing.T) {
	for _, mode := range []string{"restart", "client", "root", "replace", "modify", "outside-link"} {
		t.Run(mode, func(t *testing.T) {
			f := newDeliveryScopeFixture(t)
			path := deliveryDocumentFixtureFile(t, f, "source.html", []byte("<p>original</p>"))
			source, err := f.state.localDeliverableDocumentSource(f.owner, path, files.DeliveryPreviewHTML)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "restart":
				st := f.status
				st.Generation++
				setDeliverableFixtureField(t, f.supervisor, "status", st)
				f.state.bootStatus = st
			case "client":
				f.state.dshClient = nil
			case "root":
				if err = os.Rename(f.cwd, f.cwd+"-old"); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.RemoveAll(f.cwd); _ = os.Rename(f.cwd+"-old", f.cwd) })
				if err = os.Mkdir(f.cwd, 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(path, []byte("<p>original</p>"), 0600); err != nil {
					t.Fatal(err)
				}
			case "replace":
				replacement := filepath.Join(f.cwd, "replacement")
				if err = os.WriteFile(replacement, []byte("<p>original</p>"), 0600); err != nil {
					t.Fatal(err)
				}
				if err = os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err = os.Rename(replacement, path); err != nil {
					t.Fatal(err)
				}
			case "modify":
				if err = os.WriteFile(path, []byte("<p>modified longer source</p>"), 0600); err != nil {
					t.Fatal(err)
				}
			case "outside-link":
				outside := filepath.Join(t.TempDir(), "else.html")
				if err = os.WriteFile(outside, []byte("<p>elsewhere</p>"), 0600); err != nil {
					t.Fatal(err)
				}
				if err = os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(outside, path); err != nil {
					t.Skip("symlink unavailable", err)
				}
			}
			if source.LocalDeliveryValid() {
				t.Fatal("replaced owner/source remained authorized")
			}
			r := f.state.readWorkspaceDocument(context.Background(), source)
			defer closeDeliveryResult(r)
			if r.Err == nil || r.HTML != nil {
				t.Fatal("stale source read")
			}
		})
	}
}

func TestLocalDeliveryDocumentRejectsInvalidPathsAndKinds(t *testing.T) {
	f := newDeliveryScopeFixture(t)
	good := deliveryDocumentFixtureFile(t, f, "good.pdf", []byte("%PDF-1.7\n%%EOF"))
	outside := filepath.Join(t.TempDir(), "good.pdf")
	if err := os.WriteFile(outside, []byte("%PDF-1.7\n%%EOF"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path string
		kind files.DeliveryPreviewKind
	}{{"good.pdf", files.DeliveryPreviewPDF}, {outside, files.DeliveryPreviewPDF}, {"file://" + good, files.DeliveryPreviewPDF}, {f.cwd, files.DeliveryPreviewPDF}, {good, files.DeliveryPreviewHTML}, {good, "unknown"}, {filepath.Join(f.cwd, ".git", "secret.pdf"), files.DeliveryPreviewPDF}} {
		if _, err := f.state.localDeliverableDocumentSource(f.owner, tc.path, tc.kind); err == nil {
			t.Fatalf("accepted %q kind=%s", tc.path, tc.kind)
		}
	}
	// A container cannot be accepted without an owning window; nil is not a
	// shortcut for silently doing nothing or sending the file to another opener.
	if err := f.state.openLocalDeliverableDocument(f.owner, good, files.DeliveryPreviewPDF); err == nil {
		t.Fatal("nil window returned accepted")
	}
}

func TestLocalDeliveryMediaBoundsAndContainerValidation(t *testing.T) {
	f := newDeliveryScopeFixture(t)
	for _, mode := range []string{"mismatch", "oversize", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			data := []byte("<html>not media</html>")
			if mode == "canceled" {
				data = deliveryDocumentWAV()
			}
			path := deliveryDocumentFixtureFile(t, f, mode+".wav", data)
			if mode == "oversize" {
				if err := os.Truncate(path, 32<<20+1); err != nil {
					t.Fatal(err)
				}
			}
			source, err := f.state.localDeliverableDocumentSource(f.owner, path, files.DeliveryPreviewMedia)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if mode == "canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			r := f.state.readWorkspaceDocument(ctx, source)
			defer closeDeliveryResult(r)
			if r.Err == nil || r.Media != nil {
				t.Fatalf("admitted %s", mode)
			}
			if mode == "canceled" && !errors.Is(r.Err, context.Canceled) {
				t.Fatal(r.Err)
			}
		})
	}
}
