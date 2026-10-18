//go:build darwin && cgo

package credentials

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

// Read only the original application's documented identity. Never add an item,
// enumerate unrelated accounts, or change an existing item's ACL.
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
*/
import "C"
import "unsafe"

func decryptSafeStorage(dir string, blob []byte) ([]byte, error) {
	// Reject unknown/truncated formats before asking the OS for any secret.
	if len(blob) < 19 || string(blob[:3]) != "v10" || (len(blob)-3)%16 != 0 {
		return openLegacyMac(blob, nil)
	}
	var data unsafe.Pointer
	var length C.long
	status := C.legacyWhaleRead(&data, &length)
	if data != nil {
		defer func() { C.memset(data, 0, C.size_t(length)); C.free(data) }()
	}
	if err := keychainStatus(status); err != nil {
		return nil, err
	}
	password := C.GoBytes(data, C.int(length))
	defer clear(password)
	return openLegacyMac(blob, password)
}
