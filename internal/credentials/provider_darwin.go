//go:build darwin && cgo

package credentials

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

static CFMutableDictionaryRef credentialQuery(const char *account) {
 CFStringRef name = CFStringCreateWithCString(NULL, account, kCFStringEncodingUTF8);
 if (!name) return NULL;
 CFMutableDictionaryRef query = CFDictionaryCreateMutable(NULL, 0,
  &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 if (query) {
  CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
  CFDictionarySetValue(query, kSecAttrService, CFSTR("WhaleIsle-MyGo credentials AES-256-GCM"));
  CFDictionarySetValue(query, kSecAttrAccount, name);
 }
 CFRelease(name);
 return query;
}

static OSStatus credentialRead(const char *account, void **out, long *length) {
 CFMutableDictionaryRef query = credentialQuery(account);
 if (!query) return errSecAllocate;
 CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
 CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);
 CFTypeRef result = NULL;
 OSStatus status = SecItemCopyMatching(query, &result);
 CFRelease(query);
 if (status == errSecSuccess) {
  if (!result || CFGetTypeID(result) != CFDataGetTypeID()) status = errSecDecode;
  else {
   *length = CFDataGetLength((CFDataRef)result);
   // This item always contains one AES-256 key. Reject malformed lengths
   // before allocating or transferring bytes into Go.
   if (*length != 32) status = errSecDecode;
   else {
    *out = malloc(32);
    if (!*out) status = errSecAllocate;
    else memcpy(*out, CFDataGetBytePtr((CFDataRef)result), 32);
   }
  }
 }
 if (result) CFRelease(result);
 return status;
}

static OSStatus credentialAdd(const char *account, const void *key, long length) {
 CFMutableDictionaryRef query = credentialQuery(account);
 if (!query) return errSecAllocate;
 CFDataRef data = CFDataCreate(NULL, key, length);
 if (!data) { CFRelease(query); return errSecAllocate; }
 CFDictionarySetValue(query, kSecValueData, data);
 OSStatus status = SecItemAdd(query, NULL);
 CFRelease(data);
 CFRelease(query);
 return status;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type nativeKeychain struct{}

func platformCredentialProvider() credentialProvider {
	return credentialProvider{KeychainEnvelopeVersion, "macOS Keychain + AES-256-GCM", true, func(dir string, create bool) ([]byte, error) { return keychainKey(nativeKeychain{}, dir, create) }}
}

func keychainStatus(status C.OSStatus) error {
	switch status {
	case C.errSecSuccess:
		return nil
	case C.errSecItemNotFound:
		return errKeychainNotFound
	case C.errSecDuplicateItem:
		return errKeychainDuplicate
	default:
		return fmt.Errorf("credentials: macOS Keychain failed (OSStatus %d)", int32(status))
	}
}

func (nativeKeychain) get(account string) ([]byte, error) {
	name := C.CString(account)
	defer C.free(unsafe.Pointer(name))
	var data unsafe.Pointer
	var length C.long
	status := C.credentialRead(name, &data, &length)
	if data != nil {
		defer func() { C.memset(data, 0, C.size_t(length)); C.free(data) }()
	}
	if err := keychainStatus(status); err != nil {
		return nil, err
	}
	return C.GoBytes(data, C.int(length)), nil
}

func (nativeKeychain) add(account string, key []byte) error {
	name := C.CString(account)
	defer C.free(unsafe.Pointer(name))
	data := C.CBytes(key)
	defer func() { C.memset(data, 0, C.size_t(len(key))); C.free(data) }()
	return keychainStatus(C.credentialAdd(name, data, C.long(len(key))))
}
