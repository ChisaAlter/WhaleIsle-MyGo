//go:build darwin && !cgo

package credentials

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

func platformCredentialProvider() credentialProvider {
	return credentialProvider{KeychainEnvelopeVersion, "macOS Keychain + AES-256-GCM", true,
		func(dir string, create bool) ([]byte, error) { return keychainKey(nativeKeychain{}, dir, create) }}
}

type nativeKeychain struct{}

func (nativeKeychain) get(account string) ([]byte, error) {
	a, err := loadKeychainAPI()
	if err != nil {
		return nil, err
	}
	return a.read(credentialKeychainService, account, 32, 32)
}

func (nativeKeychain) add(account string, key []byte) error {
	a, err := loadKeychainAPI()
	if err != nil {
		return err
	}
	return a.add(account, key)
}

// Load only Apple's absolute framework paths, on first actual use. Successful
// handles live for the process lifetime because the function table retains them.
// Import/InspectStorage never reads an item or displays authentication UI.
var loadKeychainAPI = sync.OnceValues(func() (_ *keychainAPI, err error) {
	cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, fmt.Errorf("credentials: load CoreFoundation: %w", err)
	}
	security, err := purego.Dlopen("/System/Library/Frameworks/Security.framework/Security", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		_ = purego.Dlclose(cf)
		return nil, fmt.Errorf("credentials: load Security: %w", err)
	}
	defer func() {
		if err != nil {
			_ = purego.Dlclose(security)
			_ = purego.Dlclose(cf)
		}
	}()
	a := &keychainAPI{}
	for _, binding := range []struct {
		lib  uintptr
		name string
		fn   any
	}{
		{cf, "CFStringCreateWithBytes", &a.stringCreate},
		{cf, "CFDictionaryCreateMutable", &a.dictCreate},
		{cf, "CFDictionarySetValue", &a.dictSet},
		{cf, "CFDataCreate", &a.dataCreate},
		{cf, "CFDataGetLength", &a.dataLength},
		{cf, "CFDataGetBytePtr", &a.dataBytes},
		{cf, "CFGetTypeID", &a.getType},
		{cf, "CFDataGetTypeID", &a.dataType},
		{cf, "CFRelease", &a.release},
		{security, "SecItemCopyMatching", &a.copyMatching},
		{security, "SecItemAdd", &a.addItem},
	} {
		address, e := purego.Dlsym(binding.lib, binding.name)
		if e != nil {
			return nil, fmt.Errorf("credentials: resolve %s: %w", binding.name, e)
		}
		purego.RegisterFunc(binding.fn, address)
	}
	for _, binding := range []struct {
		lib      uintptr
		name     string
		to       *unsafe.Pointer
		indirect bool
	}{
		// The callbacks are exported structures; the remaining constants are
		// exported CFStringRef/CFBooleanRef variables and need one dereference.
		{cf, "kCFTypeDictionaryKeyCallBacks", &a.keyCallbacks, false},
		{cf, "kCFTypeDictionaryValueCallBacks", &a.valueCallbacks, false},
		{cf, "kCFBooleanTrue", &a.booleanTrue, true},
		{security, "kSecClass", &a.class, true},
		{security, "kSecClassGenericPassword", &a.genericPassword, true},
		{security, "kSecAttrService", &a.service, true},
		{security, "kSecAttrAccount", &a.account, true},
		{security, "kSecReturnData", &a.returnData, true},
		{security, "kSecMatchLimit", &a.matchLimit, true},
		{security, "kSecMatchLimitOne", &a.matchOne, true},
		{security, "kSecValueData", &a.valueData, true},
	} {
		address, e := purego.Dlsym(binding.lib, binding.name)
		if e != nil {
			return nil, fmt.Errorf("credentials: resolve %s: %w", binding.name, e)
		}
		p := *(*unsafe.Pointer)(unsafe.Pointer(&address))
		if binding.indirect {
			p = *(*unsafe.Pointer)(p)
		}
		if p == nil {
			return nil, fmt.Errorf("credentials: null native constant %s", binding.name)
		}
		*binding.to = p
	}
	return a, nil
})
