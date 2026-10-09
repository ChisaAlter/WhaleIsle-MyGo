//go:build windows

package credentials

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

// dataBlob mirrors CRYPT_DATA_BLOB: a DWORD length followed by a pointer
// to the bytes.
type dataBlob struct {
	cbData uint32
	pbData *byte
}

// dpapiProtect encrypts data with the current user's DPAPI context
// (CryptProtectData, no additional entropy, the mode Electron's os_crypt
// uses for the Local State key).
func dpapiProtect(data []byte) ([]byte, error) {
	return callCryptData(procCryptProtectData, data, "CryptProtectData")
}

// dpapiUnprotect reverses dpapiProtect (CryptUnprotectData).
func dpapiUnprotect(data []byte) ([]byte, error) {
	return callCryptData(procCryptUnprotectData, data, "CryptUnprotectData")
}

func callCryptData(proc *syscall.LazyProc, data []byte, what string) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("credentials: %s: empty input", what)
	}
	in := dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
	var out dataBlob
	ret, _, callErr := proc.Call(
		uintptr(unsafe.Pointer(&in)),
		0, // szDataDescr
		0, // pOptionalEntropy
		0, // pvReserved
		0, // pPromptStruct
		0, // dwFlags
		uintptr(unsafe.Pointer(&out)),
	)
	runtime.KeepAlive(in)
	if ret == 0 {
		return nil, fmt.Errorf("credentials: %s failed: %v", what, callErr)
	}
	defer func() {
		syscall.LocalFree(syscall.Handle(unsafe.Pointer(out.pbData)))
	}()
	result := make([]byte, out.cbData)
	copy(result, unsafe.Slice(out.pbData, out.cbData))
	return result, nil
}
