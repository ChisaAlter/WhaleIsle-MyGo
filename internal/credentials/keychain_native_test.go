package credentials

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
	"unsafe"
)

type cfFixtureObject struct {
	kind   uintptr
	refs   int
	bytes  []byte
	fields map[unsafe.Pointer]unsafe.Pointer
}

type keychainFixture struct {
	api                   *keychainAPI
	objects               map[unsafe.Pointer]*cfFixtureObject
	copyStatus, addStatus int32
	result                []byte
	resultKind            uintptr
	resultLength          int
	reads, adds           int
	last                  map[unsafe.Pointer][]byte
	added                 []byte
}

func newKeychainFixture(t *testing.T) *keychainFixture {
	t.Helper()
	f := &keychainFixture{objects: make(map[unsafe.Pointer]*cfFixtureObject), resultKind: 2, resultLength: -1}
	a := &keychainAPI{}
	f.api = a
	newObject := func(kind uintptr, data []byte) unsafe.Pointer {
		o := &cfFixtureObject{kind: kind, refs: 1, bytes: bytes.Clone(data)}
		p := unsafe.Pointer(o)
		f.objects[p] = o
		return p
	}
	for _, p := range []*unsafe.Pointer{&a.class, &a.genericPassword, &a.service, &a.account, &a.returnData, &a.booleanTrue, &a.matchLimit, &a.matchOne, &a.valueData, &a.keyCallbacks, &a.valueCallbacks} {
		*p = newObject(0, nil)
	}
	baseline := len(f.objects)
	a.stringCreate = func(_ unsafe.Pointer, p *byte, count int, encoding uint32, external bool) unsafe.Pointer {
		if encoding != 0x08000100 || external {
			t.Fatal("incorrect CF string ABI")
		}
		return newObject(1, unsafe.Slice(p, count))
	}
	a.dictCreate = func(_ unsafe.Pointer, capacity int, keys, values unsafe.Pointer) unsafe.Pointer {
		if capacity != 0 || keys != a.keyCallbacks || values != a.valueCallbacks {
			t.Fatal("dictionary does not retain native values")
		}
		p := newObject(3, nil)
		f.objects[p].fields = make(map[unsafe.Pointer]unsafe.Pointer)
		return p
	}
	a.dictSet = func(dict, key, value unsafe.Pointer) {
		o := f.objects[dict]
		if old := o.fields[key]; old != nil {
			a.release(old)
		}
		f.objects[value].refs++
		o.fields[key] = value
	}
	a.release = func(p unsafe.Pointer) {
		o := f.objects[p]
		if o == nil || o.refs <= 0 {
			t.Fatal("release of unowned CF object")
		}
		o.refs--
		if o.refs == 0 {
			for _, value := range o.fields {
				a.release(value)
			}
			clear(o.bytes)
			delete(f.objects, p)
		}
	}
	a.dataCreate = func(_ unsafe.Pointer, p *byte, count int) unsafe.Pointer { return newObject(2, unsafe.Slice(p, count)) }
	a.dataLength = func(p unsafe.Pointer) int {
		if f.resultLength >= 0 {
			return f.resultLength
		}
		return len(f.objects[p].bytes)
	}
	a.dataBytes = func(p unsafe.Pointer) *byte { return unsafe.SliceData(f.objects[p].bytes) }
	a.getType = func(p unsafe.Pointer) uintptr { return f.objects[p].kind }
	a.dataType = func() uintptr { return 2 }
	recordQuery := func(q unsafe.Pointer) {
		f.last = make(map[unsafe.Pointer][]byte)
		for key, value := range f.objects[q].fields {
			f.last[key] = bytes.Clone(f.objects[value].bytes)
		}
	}
	a.copyMatching = func(q unsafe.Pointer, result *unsafe.Pointer) int32 {
		f.reads++
		recordQuery(q)
		if f.result != nil {
			*result = newObject(f.resultKind, f.result)
		}
		return f.copyStatus
	}
	a.addItem = func(q, result unsafe.Pointer) int32 {
		if result != nil {
			t.Fatal("unexpected add result")
		}
		f.adds++
		recordQuery(q)
		f.added = bytes.Clone(f.last[a.valueData])
		return f.addStatus
	}
	t.Cleanup(func() {
		if len(f.objects) != baseline {
			t.Errorf("CF allocation leak: got %d, baseline %d", len(f.objects), baseline)
		}
	})
	return f
}

func TestNativeKeychainReadAddOwnership(t *testing.T) {
	f := newKeychainFixture(t)
	f.result = bytes.Repeat([]byte{0x7b}, 32)
	got, err := f.api.read(credentialKeychainService, "profile-test", 32, 32)
	if err != nil || !bytes.Equal(got, f.result) {
		t.Fatalf("read: %x %v", got, err)
	}
	if string(f.last[f.api.service]) != credentialKeychainService || string(f.last[f.api.account]) != "profile-test" {
		t.Fatal("wrong Keychain identity")
	}
	if len(f.last) != 5 {
		t.Fatal("read query must request a single data item")
	}
	if err := f.api.add("profile-test", got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.added, got) || len(f.last) != 4 || f.reads != 1 || f.adds != 1 {
		t.Fatal("unexpected add behavior")
	}
	before := f.adds
	if f.api.add("profile-test", got[:31]) == nil || f.adds != before {
		t.Fatal("malformed key reached Keychain")
	}
	f.addStatus = -25299
	if !errors.Is(f.api.add("profile-test", got), errKeychainDuplicate) {
		t.Fatal("duplicate status lost")
	}
}

func TestNativeKeychainRejectsMalformedAndDenied(t *testing.T) {
	for _, tc := range []struct {
		name   string
		length int
		kind   uintptr
		status int32
		want   error
	}{
		{"missing", 32, 2, -25300, errKeychainNotFound},
		{"denied", 32, 2, -25293, nil},
		{"not-data", 32, 1, 0, nil},
		{"short", 31, 2, 0, nil},
		{"long", 33, 2, 0, nil},
		{"negative", -2, 2, 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newKeychainFixture(t)
			f.result, f.resultKind, f.copyStatus = make([]byte, 32), tc.kind, tc.status
			if tc.length == -2 {
				f.api.dataLength = func(unsafe.Pointer) int { return -1 }
			} else {
				f.resultLength = tc.length
			}
			got, err := f.api.read(credentialKeychainService, "profile-test", 32, 32)
			if got != nil || err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
				t.Fatalf("accepted malformed or denied result: %v", err)
			}
			if f.adds != 0 {
				t.Fatal("read wrote Keychain")
			}
		})
	}
}

func TestNativeLegacyLookupAndSecretWipe(t *testing.T) {
	blob, _ := hex.DecodeString("7631305b4e217df97c3793d529753da8a42be0f19c7913ee062272e72b8667f4f24b0a")
	secret := []byte("synthetic-whale-fixture")
	reads := 0
	read := func() ([]byte, error) { reads++; return secret, nil }
	if _, err := openLegacyMacWithKeychain(blob[:3], read); err == nil || reads != 0 {
		t.Fatal("malformed legacy envelope queried Keychain")
	}
	got, err := openLegacyMacWithKeychain(blob, read)
	if err != nil || string(got) != `{"apiKey":"synthetic-only"}` {
		t.Fatalf("legacy decode: %v", err)
	}
	if !bytes.Equal(secret, make([]byte, len(secret))) {
		t.Fatal("legacy secret retained")
	}
	f := newKeychainFixture(t)
	f.result = []byte("synthetic-whale-fixture")
	_, err = f.api.read("Whale Isle Safe Storage", "Whale Isle", 1, 4096)
	if err != nil || f.adds != 0 || string(f.last[f.api.service]) != "Whale Isle Safe Storage" || string(f.last[f.api.account]) != "Whale Isle" {
		t.Fatal("legacy identity or read-only contract changed")
	}
}
