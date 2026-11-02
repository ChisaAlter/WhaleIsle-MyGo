package credentials

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

const credentialKeychainService = "WhaleIsle-MyGo credentials AES-256-GCM"

// keychainAPI follows the public Security/CoreFoundation headers. CFIndex is a
// signed pointer-sized integer, CFTypeID is unsigned, and OSStatus is int32.
// Only synchronous calls receive Go memory; CFStringCreateWithBytes and
// CFDataCreate copy it. Dictionaries retain their native values with CF callbacks.
// This table is also the seam for fixtures which never open a user's Keychain.
type keychainAPI struct {
	stringCreate func(unsafe.Pointer, *byte, int, uint32, bool) unsafe.Pointer
	dictCreate   func(unsafe.Pointer, int, unsafe.Pointer, unsafe.Pointer) unsafe.Pointer
	dictSet      func(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer)
	dataCreate   func(unsafe.Pointer, *byte, int) unsafe.Pointer
	dataLength   func(unsafe.Pointer) int
	dataBytes    func(unsafe.Pointer) *byte
	getType      func(unsafe.Pointer) uintptr
	dataType     func() uintptr
	release      func(unsafe.Pointer)
	copyMatching func(unsafe.Pointer, *unsafe.Pointer) int32
	addItem      func(unsafe.Pointer, unsafe.Pointer) int32

	keyCallbacks, valueCallbacks                             unsafe.Pointer
	class, genericPassword, service, account                 unsafe.Pointer
	returnData, booleanTrue, matchLimit, matchOne, valueData unsafe.Pointer
}

func nativeKeychainStatus(status int32) error {
	switch status {
	case 0:
		return nil
	case -25300: // errSecItemNotFound, SecBase.h
		return errKeychainNotFound
	case -25299: // errSecDuplicateItem
		return errKeychainDuplicate
	default:
		return fmt.Errorf("credentials: macOS Keychain failed (OSStatus %d)", status)
	}
}

func (a *keychainAPI) string(s string) unsafe.Pointer {
	b := []byte(s)
	// UTF-8 is 0x08000100 in CFString.h. Explicit lengths prevent embedded NUL
	// bytes from changing a service/account identity.
	p := a.stringCreate(nil, unsafe.SliceData(b), len(b), 0x08000100, false)
	runtime.KeepAlive(b)
	return p
}

func (a *keychainAPI) query(service, account string) (unsafe.Pointer, error) {
	s := a.string(service)
	if s == nil {
		return nil, errors.New("credentials: cannot allocate Keychain service")
	}
	defer a.release(s)
	n := a.string(account)
	if n == nil {
		return nil, errors.New("credentials: cannot allocate Keychain account")
	}
	defer a.release(n)
	q := a.dictCreate(nil, 0, a.keyCallbacks, a.valueCallbacks)
	if q == nil {
		return nil, errors.New("credentials: cannot allocate Keychain query")
	}
	a.dictSet(q, a.class, a.genericPassword)
	a.dictSet(q, a.service, s)
	a.dictSet(q, a.account, n)
	return q, nil
}

func (a *keychainAPI) read(service, account string, minLength, maxLength int) ([]byte, error) {
	q, err := a.query(service, account)
	if err != nil {
		return nil, err
	}
	defer a.release(q)
	a.dictSet(q, a.returnData, a.booleanTrue)
	a.dictSet(q, a.matchLimit, a.matchOne)
	var result unsafe.Pointer
	status := a.copyMatching(q, &result)
	if result != nil {
		defer a.release(result)
	}
	if err := nativeKeychainStatus(status); err != nil {
		return nil, err
	}
	if result == nil || a.getType(result) != a.dataType() {
		return nil, errors.New("credentials: Keychain returned non-data content")
	}
	length := a.dataLength(result)
	if length < minLength || length > maxLength {
		return nil, errors.New("credentials: Keychain returned an invalid data length")
	}
	p := a.dataBytes(result)
	if p == nil {
		return nil, errors.New("credentials: Keychain returned a null data pointer")
	}
	// Copy before releasing the CFData owner; never return native storage or
	// modify immutable CFData belonging to Security.framework.
	return append([]byte(nil), unsafe.Slice(p, length)...), nil
}

func (a *keychainAPI) add(account string, key []byte) error {
	if len(key) != 32 {
		return errors.New("credentials: invalid Keychain key length")
	}
	q, err := a.query(credentialKeychainService, account)
	if err != nil {
		return err
	}
	defer a.release(q)
	data := a.dataCreate(nil, unsafe.SliceData(key), len(key))
	runtime.KeepAlive(key)
	if data == nil {
		return errors.New("credentials: cannot allocate Keychain key data")
	}
	defer a.release(data)
	a.dictSet(q, a.valueData, data)
	return nativeKeychainStatus(a.addItem(q, nil))
}

func openLegacyMacWithKeychain(blob []byte, read func() ([]byte, error)) ([]byte, error) {
	// Validate the original v10 block envelope before querying any OS secret.
	if len(blob) < 19 || string(blob[:3]) != "v10" || (len(blob)-3)%16 != 0 {
		return openLegacyMac(blob, nil)
	}
	secret, err := read()
	defer clear(secret)
	if err != nil {
		return nil, err
	}
	return openLegacyMac(blob, secret)
}
