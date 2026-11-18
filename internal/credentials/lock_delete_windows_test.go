//go:build windows

package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestCredentialsLockWaitsForWindowsDeletePendingHandle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, lockFileName)
	if err := os.WriteFile(path, []byte(`{"pid":1,"token":"retiring"}`), 0600); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(path)
	reader, err := windows.CreateFile(name, windows.GENERIC_READ|windows.DELETE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if reader != windows.InvalidHandle {
			_ = windows.CloseHandle(reader)
		}
	}()
	// Use the explicit legacy disposition, not DeleteFile: newer Windows can
	// unlink a DeleteFile name immediately even while a reader remains open.
	deleteFile := byte(1)
	if err := windows.SetFileInformationByHandle(reader, windows.FileDispositionInfo, &deleteFile, 1); err != nil {
		t.Fatal(err)
	}
	var info struct {
		AllocationSize, EndOfFile int64
		NumberOfLinks             uint32
		DeletePending, Directory  byte
		_                         [2]byte
	}
	if err := windows.GetFileInformationByHandleEx(reader, windows.FileStandardInfo, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		t.Fatal(err)
	}
	if info.DeletePending == 0 {
		t.Fatal("legacy file disposition did not mark the held handle delete-pending")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err == nil {
		f.Close()
		t.Fatal("fixture did not create a delete-pending pathname")
	}
	if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		t.Fatalf("expected Windows delete-pending denial, got %v", err)
	}
	closed := make(chan struct{})
	handle := reader
	reader = windows.InvalidHandle
	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = windows.CloseHandle(handle)
		close(closed)
	}()
	release, err := acquireCredentialsLock(dir)
	<-closed
	if err != nil {
		t.Fatalf("retired pathname prevented acquiring credentials lock: %v", err)
	}
	release()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("released lock remained: %v", err)
	}
}
