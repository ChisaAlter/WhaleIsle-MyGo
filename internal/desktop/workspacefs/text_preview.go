package workspacefs

import (
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// TextPreviewLimit matches the original desktop Files text prefix. Keep this
// separate from DefaultMaxReadBytes: remote range requests retain their limit.
const TextPreviewLimit = 1024 * 1024

// ReadFileTextPreview reads a bounded, possibly truncated text prefix. Callers
// must retain Truncated and never use an incomplete preview as a save baseline.
// ReadFileText remains the strict reader for conflict checking before writes.
// The caller still owns JoinRoot authorization, exactly as for ReadFileText.
func ReadFileTextPreview(filePath string) (ReadTextResult, error) {
	filePath = filepath.Clean(filePath)
	st, err := os.Stat(filePath)
	if err != nil {
		return ReadTextResult{}, err
	}
	if !st.Mode().IsRegular() {
		return ReadTextResult{}, ErrNotFile
	}
	f, err := os.Open(filePath)
	if err != nil {
		return ReadTextResult{}, err
	}
	defer f.Close()
	st, err = f.Stat()
	if err != nil {
		return ReadTextResult{}, err
	}
	if !st.Mode().IsRegular() {
		return ReadTextResult{}, ErrNotFile
	}
	return readTextPrefix(f, st.Size())
}

func readTextPrefix(reader io.Reader, size int64) (ReadTextResult, error) {
	// One sentinel detects growth after Stat without ever reading the full file.
	data, err := io.ReadAll(io.LimitReader(reader, TextPreviewLimit+1))
	if err != nil {
		return ReadTextResult{}, err
	}
	overflow := len(data) > TextPreviewLimit
	truncated := overflow || size > TextPreviewLimit
	if overflow {
		data = data[:TextPreviewLimit]
	}
	if bytesContainsNUL(data) {
		return ReadTextResult{Size: size}, ErrBinary
	}
	if !utf8.Valid(data) && overflow {
		// A valid multibyte rune may straddle the byte cap. Omit only an
		// incomplete final rune; malformed bytes elsewhere remain an error.
		for n := 1; n < utf8.UTFMax && n <= len(data); n++ {
			prefix, tail := data[:len(data)-n], data[len(data)-n:]
			if !utf8.FullRune(tail) && utf8.Valid(prefix) {
				data = prefix
				break
			}
		}
	}
	if !utf8.Valid(data) {
		return ReadTextResult{Size: size}, ErrBinary
	}
	return ReadTextResult{Text: string(data), Truncated: truncated, Size: size}, nil
}
