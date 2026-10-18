package mediapreview

import (
	"bytes"
	"encoding/binary"
	"errors"
	"path"
	"strings"
)

func extension(name string) string {
	return strings.ToLower(path.Ext(strings.ReplaceAll(name, `\`, "/")))
}
func Supports(name string) bool {
	switch extension(name) {
	case ".mp3", ".aac", ".wav", ".flac", ".m4a", ".ogg", ".oga", ".mp4", ".m4v", ".mov", ".webm", ".ogv":
		return true
	}
	return false
}

// Validate is a container/type gate, not a codec or decoder-memory guarantee.
// Unknown containers, playlists and mismatched extensions never reach a browser.
// Playback support remains a property of the installed WebView2 runtime.
func Validate(name string, b []byte) (kind, mime string, err error) {
	bad := errors.New("unsupported or mismatched media container; use a compatible file (maximum 32 MiB)")
	if len(b) < 8 || len(b) > MaxBytes {
		return "", "", bad
	}
	ext := extension(name)
	switch ext {
	case ".wav":
		if len(b) < 44 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" || uint64(binary.LittleEndian.Uint32(b[4:]))+8 != uint64(len(b)) {
			break
		}
		fmtOK, dataOK := false, false
		for p := 12; p+8 <= len(b); {
			n := uint64(binary.LittleEndian.Uint32(b[p+4:]))
			end := uint64(p) + 8 + n
			if end > uint64(len(b)) {
				return "", "", bad
			}
			if string(b[p:p+4]) == "fmt " && n >= 16 {
				format := binary.LittleEndian.Uint16(b[p+8:])
				channels := binary.LittleEndian.Uint16(b[p+10:])
				rate := binary.LittleEndian.Uint32(b[p+12:])
				fmtOK = (format == 1 || format == 3 || format == 0xfffe) && channels > 0 && channels <= 32 && rate > 0 && rate <= 384000
			}
			if string(b[p:p+4]) == "data" && n > 0 {
				dataOK = true
			}
			p = int(end + (n & 1))
		}
		if fmtOK && dataOK {
			return "audio", "audio/wav", nil
		}
	case ".flac":
		if len(b) >= 42 && string(b[:4]) == "fLaC" && b[4]&0x7f == 0 && b[5] == 0 && b[6] == 0 && b[7] == 34 {
			return "audio", "audio/flac", nil
		}
	case ".mp3":
		p := 0
		if string(b[:3]) == "ID3" {
			if len(b) < 10 || b[3] < 2 || b[3] > 4 {
				return "", "", bad
			}
			n := 0
			for _, v := range b[6:10] {
				if v&0x80 != 0 {
					return "", "", bad
				}
				n = (n << 7) | int(v)
			}
			p = 10 + n
			if b[3] == 4 && b[5]&0x10 != 0 {
				p += 10
			}
		}
		if p+4 <= len(b) && b[p] == 0xff && b[p+1]&0xe0 == 0xe0 && b[p+1]&0x18 != 8 && b[p+1]&6 != 0 && b[p+2]&0xf0 != 0 && b[p+2]&0xf0 != 0xf0 && b[p+2]&0x0c != 0x0c {
			return "audio", "audio/mpeg", nil
		}
	case ".aac":
		n := int(b[3]&3)<<11 | int(b[4])<<3 | int(b[5]>>5)
		if b[0] == 0xff && b[1]&0xf6 == 0xf0 && b[2]&0x3c != 0x3c && n >= 7 && n <= len(b) {
			return "audio", "audio/aac", nil
		}
	case ".ogg", ".oga", ".ogv":
		if len(b) < 28 || string(b[:4]) != "OggS" || b[4] != 0 || b[5]&2 == 0 {
			break
		}
		count := int(b[26])
		if count == 0 || 27+count > len(b) {
			break
		}
		n := 0
		for _, v := range b[27 : 27+count] {
			n += int(v)
			if v < 255 {
				break
			}
		}
		p := 27 + count
		if n < 7 || p+n > len(b) {
			break
		}
		packet := b[p : p+n]
		if ext == ".ogv" && bytes.HasPrefix(packet, []byte{0x80, 't', 'h', 'e', 'o', 'r', 'a'}) {
			return "video", "video/ogg", nil
		}
		if ext != ".ogv" && (bytes.HasPrefix(packet, []byte{1, 'v', 'o', 'r', 'b', 'i', 's'}) || bytes.HasPrefix(packet, []byte("OpusHead")) || bytes.HasPrefix(packet, []byte{0x7f, 'F', 'L', 'A', 'C'})) {
			return "audio", "audio/ogg", nil
		}
	case ".webm":
		if bytes.Equal(b[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) && bytes.Contains(b[:min(len(b), 4096)], []byte{0x42, 0x82, 0x84, 'w', 'e', 'b', 'm'}) {
			return "video", "video/webm", nil
		}
	case ".mp4", ".m4v", ".m4a", ".mov":
		if validBMFF(b, ext == ".mov") {
			if ext == ".m4a" {
				return "audio", "audio/mp4", nil
			}
			if ext == ".mov" {
				return "video", "video/quicktime", nil
			}
			return "video", "video/mp4", nil
		}
	}
	return "", "", bad
}
func validBMFF(b []byte, mov bool) bool {
	if len(b) < 24 || string(b[4:8]) != "ftyp" {
		return false
	}
	brandOK, moov, mdat := false, false, false
	for p, boxes := 0, 0; p < len(b); boxes++ {
		if boxes > 65536 || p+8 > len(b) {
			return false
		}
		n := uint64(binary.BigEndian.Uint32(b[p:]))
		header := 8
		if n == 1 {
			if p+16 > len(b) {
				return false
			}
			n = binary.BigEndian.Uint64(b[p+8:])
			header = 16
		}
		if n == 0 {
			n = uint64(len(b) - p)
		}
		if n < uint64(header) || n > uint64(len(b)-p) {
			return false
		}
		end := p + int(n)
		tag := string(b[p+4 : p+8])
		if tag == "ftyp" {
			if p != 0 || n < 16 || n > 4096 {
				return false
			}
			for i := p + 8; i+4 <= end; i += 4 {
				if i == p+12 {
					continue
				}
				brand := string(b[i : i+4])
				if mov {
					brandOK = brandOK || brand == "qt  "
				} else {
					switch brand {
					case "isom", "iso2", "mp41", "mp42", "M4A ", "M4V ", "avc1", "dash":
						brandOK = true
					}
				}
			}
		}
		moov = moov || tag == "moov"
		mdat = mdat || tag == "mdat"
		p = end
	}
	return brandOK && moov && mdat
}
