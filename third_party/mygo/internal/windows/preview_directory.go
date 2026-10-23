//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	previewReadAttributes = 0x80
	previewDeleteAccess   = 0x10000
	previewDisposition    = 4 // FILE_INFO_BY_HANDLE_CLASS.FileDispositionInfo
)

var (
	procPreviewSetFileInformation = kernel32.NewProc("SetFileInformationByHandle")
	procPreviewGetFileInformation = kernel32.NewProc("GetFileInformationByHandleEx")
)

// Path-based FileInfo on Windows can defer its file-ID query until SameFile.
// Keep the ID obtained from the original open handle instead of a lazy path.
type previewDirectoryIdentity struct {
	volume uint64
	file   [16]byte
	valid  bool
}

func previewIdentityFromHandle(handle syscall.Handle) (previewDirectoryIdentity, error) {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(handle, &info); err != nil {
		return previewDirectoryIdentity{}, err
	}
	if info.FileAttributes&syscall.FILE_ATTRIBUTE_DIRECTORY == 0 || info.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return previewDirectoryIdentity{}, errors.New("mygo: preview folder must be a real directory")
	}
	// ReFS requires the full 128-bit ID; its legacy 64-bit index need not
	// uniquely identify a file. Query both volume and ID from this handle.
	var id struct {
		Volume uint64
		File   [16]byte
	}
	result, _, err := procPreviewGetFileInformation.Call(uintptr(handle), 18, uintptr(unsafe.Pointer(&id)), unsafe.Sizeof(id)) // FileIdInfo
	if result == 0 {
		if err == syscall.Errno(0) {
			err = syscall.EINVAL
		}
		return previewDirectoryIdentity{}, err
	}
	return previewDirectoryIdentity{volume: id.Volume, file: id.File, valid: true}, nil
}

func openPreviewDirectory(path string, access, share uint32) (syscall.Handle, previewDirectoryIdentity, error) {
	if !filepath.IsAbs(path) {
		return syscall.InvalidHandle, previewDirectoryIdentity{}, errors.New("mygo: preview folder must be absolute")
	}
	path = filepath.Clean(path)
	// CreateFile does not perform os.Open's long-path normalization.
	if !strings.HasPrefix(path, `\\?\`) {
		if strings.HasPrefix(path, `\\`) {
			path = `\\?\UNC\` + path[2:]
		} else {
			path = `\\?\` + path
		}
	}
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return syscall.InvalidHandle, previewDirectoryIdentity{}, err
	}
	handle, err := syscall.CreateFile(name, access, share, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return syscall.InvalidHandle, previewDirectoryIdentity{}, err
	}
	identity, err := previewIdentityFromHandle(handle)
	if err != nil {
		syscall.CloseHandle(handle)
		return syscall.InvalidHandle, previewDirectoryIdentity{}, err
	}
	return handle, identity, nil
}

func newPreviewEnvironment(path string) (*previewEnvironment, error) {
	handle, identity, err := openPreviewDirectory(path, previewReadAttributes, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE)
	if err != nil {
		return nil, err
	}
	// Keeping the original file object alive prevents its ID being recycled
	// after deletion. Full sharing permits browser use and ordinary rename.
	return &previewEnvironment{path: path, identity: identity, identityHandle: handle}, nil
}

func (p *previewEnvironment) releaseDirectoryIdentity() {
	if p.identityHandle != 0 && p.identityHandle != syscall.InvalidHandle {
		syscall.CloseHandle(p.identityHandle)
		p.identityHandle = 0
	}
	// Cleanup is terminal once the original file object is released. A
	// later call must not trust an ID the filesystem is free to recycle.
	p.identity = previewDirectoryIdentity{}
}

func capturePreviewDirectoryIdentity(path string) (previewDirectoryIdentity, error) {
	handle, identity, err := openPreviewDirectory(path, previewReadAttributes, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE)
	if err != nil {
		return previewDirectoryIdentity{}, err
	}
	defer syscall.CloseHandle(handle)
	return identity, nil
}

// The pin denies delete sharing for the verified directory throughout cleanup.
// Children are accessed through an independently verified Root, and the final
// directory is removed by the pin itself, never by re-resolving its path.
type previewDirectoryCleanup struct {
	pin  syscall.Handle
	root *os.Root
}

func lockPreviewDirectory(path string, identity previewDirectoryIdentity) (*previewDirectoryCleanup, error) {
	if !identity.valid {
		return nil, errors.New("mygo: preview folder identity is missing")
	}
	pin, current, err := openPreviewDirectory(path, previewReadAttributes|previewDeleteAccess, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE)
	if err != nil {
		return nil, err
	}
	cleanup := &previewDirectoryCleanup{pin: pin}
	ok := false
	defer func() {
		if !ok {
			cleanup.close()
		}
	}()
	if current != identity {
		return nil, errors.New("mygo: preview folder was replaced")
	}
	// OpenRoot(path) omits delete sharing on Windows, conflicting with the
	// DELETE access of pin. Root.OpenRoot uses handle-relative, shared opens.
	parent, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	cleanup.root, err = parent.OpenRoot(filepath.Base(path))
	parent.Close()
	if err != nil {
		return nil, err
	}
	file, err := cleanup.root.Open(".")
	if err != nil {
		return nil, err
	}
	rootIdentity, err := previewIdentityFromHandle(syscall.Handle(file.Fd()))
	file.Close()
	if err != nil {
		return nil, err
	}
	if rootIdentity != identity {
		return nil, errors.New("mygo: preview cleanup root was replaced")
	}
	ok = true
	return cleanup, nil
}

func (p *previewDirectoryCleanup) close() {
	if p.root != nil {
		p.root.Close()
		p.root = nil
	}
	if p.pin != syscall.InvalidHandle {
		syscall.CloseHandle(p.pin)
		p.pin = syscall.InvalidHandle
	}
}

func (p *previewDirectoryCleanup) remove() error {
	file, err := p.root.Open(".")
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		// Root.RemoveAll removes a reparse entry without traversing outside
		// this root, including when an entry changes during deletion.
		if err := p.root.RemoveAll(name); err != nil {
			return err
		}
	}
	// FILE_DISPOSITION_INFO contains one BOOLEAN. The pin stays open until
	// close, so no path swap can redirect deletion of the final directory.
	remove := byte(1)
	result, _, err := procPreviewSetFileInformation.Call(uintptr(p.pin), previewDisposition, uintptr(unsafe.Pointer(&remove)), unsafe.Sizeof(remove))
	if result == 0 {
		if err == syscall.Errno(0) {
			return syscall.EINVAL
		}
		return err
	}
	return nil
}
