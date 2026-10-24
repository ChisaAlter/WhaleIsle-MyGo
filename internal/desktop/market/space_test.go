package market

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func withFreeBytes(t *testing.T, value int64, err error) {
	t.Helper()
	previous := freeBytes
	freeBytes = func(string) (int64, error) { return value, err }
	t.Cleanup(func() { freeBytes = previous })
}

// TestCheckInstallSpaceReserve mirrors install-space.js:6-11: the preflight
// requires the install size plus a 256 MiB reserve.
func TestCheckInstallSpaceReserve(t *testing.T) {
	dir := t.TempDir()
	withFreeBytes(t, 2<<30, nil)
	if _, err := CheckInstallSpace(dir, 0); err != nil {
		t.Fatalf("2 GiB free rejected: %v", err)
	}
	// Exactly reserve+size passes, one byte less fails.
	withFreeBytes(t, InstallSpaceReserve+100, nil)
	if _, err := CheckInstallSpace(dir, 100); err != nil {
		t.Fatalf("exact space rejected: %v", err)
	}
	withFreeBytes(t, InstallSpaceReserve+99, nil)
	_, err := CheckInstallSpace(dir, 100)
	if err == nil {
		t.Fatal("insufficient space accepted")
	}
	for _, want := range []string{"磁盘空间不足", FormatGiB(InstallSpaceReserve + 99), FormatGiB(InstallSpaceReserve + 100)} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("message %q lacks %q", err, want)
		}
	}
	if !strings.Contains(FormatGiB(256<<20), "0.25 GiB") {
		t.Fatalf("FormatGiB(256 MiB) = %q, want 0.25 GiB", FormatGiB(256<<20))
	}
	if _, err := CheckInstallSpace(dir, -1); err == nil {
		t.Fatal("a negative size was accepted")
	}
}

// TestCheckInstallSpaceWalksUp is the "destination does not exist yet" case:
// the preflight must use the nearest existing ancestor's volume.
func TestCheckInstallSpaceWalksUp(t *testing.T) {
	base := t.TempDir()
	nested := base + string(os.PathSeparator) + "not" + string(os.PathSeparator) + "there"
	calls := []string{}
	previous := freeBytes
	freeBytes = func(dir string) (int64, error) {
		calls = append(calls, dir)
		if strings.HasPrefix(dir, nested) || dir == nested {
			return 0, os.ErrNotExist
		}
		return 8 << 30, nil
	}
	t.Cleanup(func() { freeBytes = previous })
	if _, err := CheckInstallSpace(nested, 1<<20); err != nil {
		t.Fatalf("ancestor walk failed: %v", err)
	}
	if len(calls) < 2 || !strings.HasSuffix(calls[0], "there") {
		t.Fatalf("probe order = %v, want the missing directory first", calls)
	}
}

// TestCheckInstallSpaceUnsupported reports an unsupported filesystem instead of
// pretending the volume has zero bytes.
func TestCheckInstallSpaceUnsupported(t *testing.T) {
	withFreeBytes(t, 0, ErrSpaceUnsupported)
	if _, err := CheckInstallSpace(t.TempDir(), 0); !errors.Is(err, ErrSpaceUnsupported) {
		t.Fatalf("unsupported filesystem reported %v", err)
	}
}
