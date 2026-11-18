//go:build darwin && cgo

package credentials

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

// Read only the original application's documented identity. Existing items and
// their ACLs are never changed; a new profile may add this exact shared identity.
static OSStatus legacyWhaleRead(void **out, long *length) {
 const void *keys[] = {kSecClass, kSecAttrService, kSecAttrAccount, kSecReturnData, kSecMatchLimit};
 const void *values[] = {kSecClassGenericPassword, CFSTR("Whale Isle Safe Storage"), CFSTR("Whale Isle"), kCFBooleanTrue, kSecMatchLimitOne};
 CFDictionaryRef query = CFDictionaryCreate(NULL, keys, values, 5, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 if (!query) return errSecAllocate;
 CFTypeRef result = NULL;
 OSStatus status = SecItemCopyMatching(query, &result);
 CFRelease(query);
 if (status == errSecSuccess) {
  if (!result || CFGetTypeID(result) != CFDataGetTypeID()) status = errSecDecode;
  else {
   *length = CFDataGetLength((CFDataRef)result);
   if (*length <= 0 || *length > 4096) status = errSecDecode;
   else {
    *out = malloc(*length);
    if (!*out) status = errSecAllocate;
    else memcpy(*out, CFDataGetBytePtr((CFDataRef)result), *length);
   }
  }
 }
 if (result) CFRelease(result);
 return status;
}

static OSStatus legacyWhaleAdd(const void *password, long length) {
 CFMutableDictionaryRef query = CFDictionaryCreateMutable(NULL, 0,
  &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 if (!query) return errSecAllocate;
 CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
 CFDictionarySetValue(query, kSecAttrService, CFSTR("Whale Isle Safe Storage"));
 CFDictionarySetValue(query, kSecAttrAccount, CFSTR("Whale Isle"));
 CFDataRef data = CFDataCreate(NULL, password, length);
 if (!data) { CFRelease(query); return errSecAllocate; }
 CFDictionarySetValue(query, kSecValueData, data);
 OSStatus status = SecItemAdd(query, NULL);
 CFRelease(data);
 CFRelease(query);
 return status;
}
*/
import "C"
import "unsafe"

func decryptSafeStorage(dir string, blob []byte) ([]byte, error) {
	return openLegacyMacWithKeychain(blob, readMacSafeStoragePassword)
}

func readMacSafeStoragePassword() ([]byte, error) {
	var data unsafe.Pointer
	var length C.long
	status := C.legacyWhaleRead(&data, &length)
	if data != nil {
		defer func() { C.memset(data, 0, C.size_t(length)); C.free(data) }()
	}
	if err := keychainStatus(status); err != nil {
		return nil, err
	}
	return C.GoBytes(data, C.int(length)), nil
}

func macSafeStorageSeal(_ string, plaintext []byte, create bool) (string, []byte, error) {
	password, err := getMacSafeStoragePassword(create, readMacSafeStoragePassword, func(password []byte) error {
		data := C.CBytes(password)
		defer func() { C.memset(data, 0, C.size_t(len(password))); C.free(data) }()
		return keychainStatus(C.legacyWhaleAdd(data, C.long(len(password))))
	})
	defer clear(password)
	if err != nil {
		return "", nil, err
	}
	payload, err := sealLegacyMac(plaintext, password)
	return EnvelopeVersion, payload, err
}
