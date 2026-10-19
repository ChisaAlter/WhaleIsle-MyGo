package chrome

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestHostWallpaperDataURLDecodedInMemory(t *testing.T) {
	want := []byte{0x89, 'P', 'N', 'G', 0, 0xff}
	got, err := wallpaperImageBytes("data:image/png;base64," + base64.StdEncoding.EncodeToString(want))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("data URL treated as a disk path: %x %v", got, err)
	}
	got, err = wallpaperImageBytes("data:image/svg+xml,%3Csvg%3E%2B%3C/svg%3E")
	if err != nil || string(got) != "<svg>+</svg>" {
		t.Fatalf("non-base64 data changed bytes: %q %v", got, err)
	}
	if _, err = wallpaperImageBytes("data:image/png;base64,invalid!*"); err == nil {
		t.Fatal("invalid payload accepted")
	}
}
