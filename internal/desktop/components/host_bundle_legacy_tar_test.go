package components

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// A '\x00' typeflag is the pre-POSIX regular-file marker (tar.TypeRegA).
// archive/tar's Reader in Go 1.27 normalizes it to tar.TypeReg, so the host
// bundle extractor does not need a TypeRegA case; this test pins both halves
// of that claim.
func TestHostBundleLegacyTypeRegAEntryStillExtracts(t *testing.T) {
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	body := []byte("legacy entry")
	if err := tw.WriteHeader(&tar.Header{Name: "runtime/host", Mode: 0755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	// The typeflag sits at offset 156 of the ustar header; flip the first
	// header's regular marker back to the legacy NUL form, then refresh the
	// header checksum (bytes 148-155, computed with the field itself blank).
	header := raw.Bytes()[:512]
	header[156] = 0
	sum := 0
	for i, b := range header {
		if i >= 148 && i < 156 {
			sum += ' '
		} else {
			sum += int(b)
		}
	}
	copy(header[148:156], fmt.Sprintf("%06o\x00 ", sum))

	var packed bytes.Buffer
	gz := gzip.NewWriter(&packed)
	if _, err := gz.Write(raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := os.WriteFile(archive, packed.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	inflate, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := tar.NewReader(inflate).Next()
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Typeflag != tar.TypeReg {
		t.Fatalf("Reader reported legacy NUL typeflag as %#x, want TypeReg", parsed.Typeflag)
	}

	destination := t.TempDir()
	if err := extractHostBundle(context.Background(), archive, destination); err != nil {
		t.Fatal(err)
	}
	extracted, err := os.ReadFile(filepath.Join(destination, "runtime", "host"))
	if err != nil {
		t.Fatal(err)
	}
	if string(extracted) != string(body) {
		t.Fatalf("extracted %q, want %q", extracted, body)
	}
}
