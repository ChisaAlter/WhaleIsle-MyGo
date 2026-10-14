//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"unsafe"
)

// Read the IStream's logical contents, not GlobalSize's allocation capacity:
// rounded-up backing bytes are not part of the PNG and must not be disclosed.
// IStream::Seek slot5 and ISequentialStream::Read slot3 are the same supported
// interfaces already used by com.go and surface_drag.go. No disk stream is used.
func readPreviewPNG(stream uintptr) ([]byte, error) {
	if hr := comCall(stream, 5, 0, 0, 0); failed(hr) {
		return nil, hresultError("preview IStream::Seek", hr)
	}
	const limit = 64 * 1024 * 1024
	chunk := make([]byte, 64*1024)
	var data []byte
	for {
		var n uint32
		hr := comCall(stream, 3, uintptr(unsafe.Pointer(&chunk[0])), uintptr(len(chunk)), uintptr(unsafe.Pointer(&n)))
		if failed(hr) {
			return nil, hresultError("preview IStream::Read", hr)
		}
		if n > uint32(len(chunk)) || len(data)+int(n) > limit {
			return nil, errors.New("mygo: preview PNG exceeds memory budget")
		}
		data = append(data, chunk[:n]...)
		if hr == 1 || n == 0 {
			return data, nil
		}
	}
}
