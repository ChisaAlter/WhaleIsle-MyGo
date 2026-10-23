//go:build windows && (amd64 || arm64)

package windows

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func previewOwnedDirectory(t *testing.T, path string) *previewEnvironment {
	t.Helper()
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	p, err := newPreviewEnvironment(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.releaseDirectoryIdentity)
	return p
}

func previewMarker(t *testing.T, directory string) string {
	t.Helper()
	path := filepath.Join(directory, "keep.txt")
	if err := os.WriteFile(path, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func previewAssertMarker(t *testing.T, path string) {
	t.Helper()
	if data, err := os.ReadFile(path); err != nil || string(data) != "preserve" {
		t.Fatalf("unowned marker changed: data%q err%v", data, err)
	}
}

func TestPreviewCleanupRemovesOriginalContentsAndCanRepeat(t *testing.T) {
	p := previewOwnedDirectory(t, filepath.Join(t.TempDir(), "owned"))
	if err := os.MkdirAll(filepath.Join(p.path, "cache", "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	previewMarker(t, filepath.Join(p.path, "cache", "nested"))
	// More than one Windows enumeration buffer worth of ordinary entries.
	for i := range 1100 {
		if err := os.WriteFile(filepath.Join(p.path, fmt.Sprintf("cache-%04d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	p.removeOwnedDirectory()
	if _, err := os.Lstat(p.path); !os.IsNotExist(err) {
		t.Fatalf("original folder retained: %v", err)
	}
	p.removeOwnedDirectory()
}

func TestPreviewCleanupRefusesMissingFileAndUnknownIdentity(t *testing.T) {
	for _, replacement := range []string{"missing", "file", "unknown-identity", "wrong-id"} {
		t.Run(replacement, func(t *testing.T) {
			root := t.TempDir()
			p := previewOwnedDirectory(t, filepath.Join(root, "owned"))
			moved := filepath.Join(root, "moved")
			if err := os.Rename(p.path, moved); err != nil {
				t.Fatal(err)
			}
			original := previewMarker(t, moved)
			switch replacement {
			case "file":
				if err := os.WriteFile(p.path, []byte("replacement"), 0600); err != nil {
					t.Fatal(err)
				}
			case "unknown-identity", "wrong-id":
				if err := os.Mkdir(p.path, 0700); err != nil {
					t.Fatal(err)
				}
				previewMarker(t, p.path)
				if replacement == "unknown-identity" {
					p.identity = previewDirectoryIdentity{}
				} else {
					p.identity.file[0] ^= 1
				}
			}
			p.removeOwnedDirectory()
			previewAssertMarker(t, original)
			switch replacement {
			case "file":
				if data, err := os.ReadFile(p.path); err != nil || string(data) != "replacement" {
					t.Fatalf("replacement file removed: %q %v", data, err)
				}
			case "unknown-identity", "wrong-id":
				previewAssertMarker(t, filepath.Join(p.path, "keep.txt"))
			}
		})
	}
}

func TestPreviewCleanupPinRejectsRenameAndReleasesOnClose(t *testing.T) {
	root := t.TempDir()
	p := previewOwnedDirectory(t, filepath.Join(root, "owned"))
	marker := previewMarker(t, p.path)
	cleanup, err := lockPreviewDirectory(p.path, p.identity)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.close()
	moved := filepath.Join(root, "moved")
	if err := os.Rename(p.path, moved); err == nil {
		t.Fatal("verified cleanup root was renamed under its pin")
	}
	previewAssertMarker(t, marker)
	if err := cleanup.remove(); err != nil {
		t.Fatal(err)
	}
	cleanup.close()
	cleanup.close()
	p.releaseDirectoryIdentity()
	if _, err := os.Lstat(p.path); !os.IsNotExist(err) {
		t.Fatalf("handle disposition did not remove original: %v", err)
	}
	if err := os.Mkdir(p.path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(p.path, moved); err != nil {
		t.Fatalf("cleanup pin leaked: %v", err)
	}
}

func TestPreviewCleanupRetainsInUseFolderAndTerminatesOwner(t *testing.T) {
	p := previewOwnedDirectory(t, filepath.Join(t.TempDir(), "owned"))
	marker := previewMarker(t, p.path)
	file, err := os.Open(marker)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	p.removeOwnedDirectory()
	previewAssertMarker(t, marker)
	file.Close()
	if p.identityHandle != 0 || p.identity.valid {
		t.Fatal("failed cleanup retained identity")
	}
	p.removeOwnedDirectory()
	previewAssertMarker(t, marker)
	// Only a newly established owner may retry once the old file object
	// was released; an exit callback never schedules path-based retries.
	fresh, err := newPreviewEnvironment(p.path)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.releaseDirectoryIdentity()
	fresh.removeOwnedDirectory()
	if _, err := os.Lstat(p.path); !os.IsNotExist(err) {
		t.Fatalf("fresh verified cleanup failed: %v", err)
	}
}

func TestPreviewIdentityLifetimeAllowsRenameAndEndsOnClose(t *testing.T) {
	root := t.TempDir()
	p := previewOwnedDirectory(t, filepath.Join(root, "owned"))
	handle := p.identityHandle
	moved := filepath.Join(root, "moved")
	if err := os.Rename(p.path, moved); err != nil {
		t.Fatal("original identity blocked browser-era rename", err)
	}
	id, err := previewIdentityFromHandle(handle)
	if err != nil || id != p.identity {
		t.Fatalf("original identity changed after rename: %v", err)
	}
	marker := previewMarker(t, moved)
	p.close()
	p.close()
	if p.identity.valid || p.identityHandle != 0 {
		t.Fatal("close without exit subscription retained identity")
	}
	if _, err := previewIdentityFromHandle(handle); err == nil {
		t.Fatal("released identity handle is still open")
	}
	p.path = moved
	p.removeOwnedDirectory()
	previewAssertMarker(t, marker)
}

func TestPreviewIdentityStaysAliveUntilPendingCreationSettles(t *testing.T) {
	p := previewOwnedDirectory(t, filepath.Join(t.TempDir(), "owned"))
	handle := p.identityHandle
	p.creating = true
	p.close()
	if !p.identity.valid || p.identityHandle != handle {
		t.Fatal("close released identity during pending environment creation")
	}
	if _, err := previewIdentityFromHandle(handle); err != nil {
		t.Fatal("pending original file object lost", err)
	}
	// Asynchronous creation failure after the window has already closed.
	p.creating = false
	p.removeOwnedDirectory()
	if p.identity.valid || p.identityHandle != 0 {
		t.Fatal("completion retained original identity")
	}
	if _, err := os.Lstat(p.path); !os.IsNotExist(err) {
		t.Fatalf("creation failure cleanup: %v", err)
	}
}

func TestPreviewIdentityLateCreationSuccessAfterCloseReleasesWithoutSubscription(t *testing.T) {
	p := previewOwnedDirectory(t, filepath.Join(t.TempDir(), "owned"))
	marker := previewMarker(t, p.path)
	p.creating = true
	p.close()
	// A successful late environment is released before an exit subscription
	// or controller is requested; no browser-exit event is needed to close.
	p.creating = false
	p.close()
	if p.identity.valid || p.identityHandle != 0 {
		t.Fatal("late completion leaked original identity")
	}
	previewAssertMarker(t, marker)
}

func TestPreviewExitCleanupEndsRetryAndReleasesIdentity(t *testing.T) {
	p := previewOwnedDirectory(t, filepath.Join(t.TempDir(), "owned"))
	w := &window{closed: true, isolated: p}
	// Closed native window needs no HWND or COM object; exercise the actual
	// exit completion and subsequent stale retry callbacks on its real folder.
	w.previewBrowserExited(p)
	if !p.closed || p.exiting || p.identity.valid || p.identityHandle != 0 {
		t.Fatal("exit did not terminate original owner")
	}
	if _, err := os.Lstat(p.path); !os.IsNotExist(err) {
		t.Fatalf("exit folder not removed: %v", err)
	}
	w.closed = false
	w.createWebView()
	w.creationFailed(eFail)
	if w.webViewFails != 0 {
		t.Fatal("private preview restarted after terminal browser exit")
	}
}

func TestPreviewExitDuringControllerSetupWaitsForOuterCallback(t *testing.T) {
	p := previewOwnedDirectory(t, filepath.Join(t.TempDir(), "owned"))
	marker := previewMarker(t, p.path)
	w := &window{closed: true, isolated: p}
	p.controllerSetup = true
	w.previewBrowserExited(p)
	if !p.exitPending || !p.closed || p.identityHandle == 0 {
		t.Fatal("reentrant exit did not preserve setup owner")
	}
	previewAssertMarker(t, marker)
	p.controllerSetup = false
	p.exitPending = false
	w.previewBrowserExited(p)
	if p.identityHandle != 0 || p.identity.valid {
		t.Fatal("outer completion retained owner")
	}
	if _, err := os.Lstat(p.path); !os.IsNotExist(err) {
		t.Fatalf("deferred exit cleanup failed: %v", err)
	}
}

func TestPreviewIdentityCloseKeepsSubscribedOriginalUntilExit(t *testing.T) {
	p := previewOwnedDirectory(t, filepath.Join(t.TempDir(), "owned"))
	marker := previewMarker(t, p.path)
	// No COM method is invoked: this nonzero token represents the lifetime
	// of an established exit subscription while env is already released.
	p.events = 1
	p.browserStarted = true
	p.close()
	if !p.identity.valid || p.identityHandle == 0 {
		t.Fatal("close disposed subscribed original")
	}
	previewAssertMarker(t, marker)
	p.events = 0
	p.removeOwnedDirectory()
	if p.identity.valid || p.identityHandle != 0 {
		t.Fatal("browser-exit cleanup retained identity")
	}
	if _, err := os.Lstat(p.path); !os.IsNotExist(err) {
		t.Fatalf("exit cleanup failed: %v", err)
	}
}

func TestPreviewIdentityRejectsRelativeMissingFileAndInvalidHandle(t *testing.T) {
	if _, err := capturePreviewDirectoryIdentity("."); err == nil {
		t.Fatal("accepted relative identity")
	}
	root := t.TempDir()
	if _, err := capturePreviewDirectoryIdentity(filepath.Join(root, "missing")); err == nil {
		t.Fatal("accepted missing identity")
	}
	file := previewMarker(t, root)
	if _, err := capturePreviewDirectoryIdentity(file); err == nil {
		t.Fatal("accepted ordinary file identity")
	}
	if id, err := previewIdentityFromHandle(syscall.InvalidHandle); err == nil || id.valid {
		t.Fatal("accepted invalid handle identity")
	}
}

// Junctions do not require the developer-mode/symlink privilege. The fixture
// uses native reparse data so this path is exercised on ordinary Windows CI.
func previewJunction(t *testing.T, link, target string) {
	t.Helper()
	if err := os.Mkdir(link, 0700); err != nil {
		t.Fatal(err)
	}
	name, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(h)
	sub, err := syscall.UTF16FromString(`\??\` + target)
	if err != nil {
		t.Fatal(err)
	}
	printName, err := syscall.UTF16FromString(target)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 16+2*(len(sub)+len(printName)))
	binary.LittleEndian.PutUint32(buf, 0xa0000003) // IO_REPARSE_TAG_MOUNT_POINT
	binary.LittleEndian.PutUint16(buf[4:], uint16(len(buf)-8))
	binary.LittleEndian.PutUint16(buf[10:], uint16(2*(len(sub)-1)))
	binary.LittleEndian.PutUint16(buf[12:], uint16(2*len(sub)))
	binary.LittleEndian.PutUint16(buf[14:], uint16(2*(len(printName)-1)))
	for i, value := range append(sub, printName...) {
		binary.LittleEndian.PutUint16(buf[16+2*i:], value)
	}
	var returned uint32
	if err := syscall.DeviceIoControl(h, 0x900a4, &buf[0], uint32(len(buf)), nil, 0, &returned, nil); err != nil {
		t.Fatal(err)
	} // FSCTL_SET_REPARSE_POINT
}

func TestPreviewCleanupRefusesJunctionRootAndDoesNotTraverseChild(t *testing.T) {
	for _, replaceRoot := range []bool{true, false} {
		t.Run(fmt.Sprintf("root-%t", replaceRoot), func(t *testing.T) {
			root := t.TempDir()
			outside := filepath.Join(root, "outside")
			if err := os.Mkdir(outside, 0700); err != nil {
				t.Fatal(err)
			}
			marker := previewMarker(t, outside)
			p := previewOwnedDirectory(t, filepath.Join(root, "owned"))
			link := filepath.Join(p.path, "link")
			if replaceRoot {
				if err := os.Rename(p.path, filepath.Join(root, "original")); err != nil {
					t.Fatal(err)
				}
				link = p.path
			}
			previewJunction(t, link, outside)
			if replaceRoot {
				if _, err := capturePreviewDirectoryIdentity(link); err == nil {
					t.Fatal("junction accepted as original")
				}
			}
			p.removeOwnedDirectory()
			previewAssertMarker(t, marker)
			_, err := os.Lstat(p.path)
			if replaceRoot && err != nil {
				t.Fatalf("replacement junction removed: %v", err)
			}
			if !replaceRoot && !os.IsNotExist(err) {
				t.Fatalf("original with child junction retained: %v", err)
			}
		})
	}
}

func TestPreviewCleanupWorksWithLongUnicodePath(t *testing.T) {
	root := t.TempDir()
	for range 4 {
		root = filepath.Join(root, strings.Repeat("鲸屿", 20))
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	p := previewOwnedDirectory(t, filepath.Join(root, "owned"))
	previewMarker(t, p.path)
	p.removeOwnedDirectory()
	if _, err := os.Lstat(p.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("long Unicode cleanup: %v", err)
	}
}
