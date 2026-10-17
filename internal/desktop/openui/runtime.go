package openui

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

const ViewerSHA256 = "ee5470ac1f67c50e69e554f6be5d7202625eef572b1f272f2276a1e250876961"

// LoadViewer reads an explicitly supplied viewer.html/app.asar, or the user's
// existing standard desktop install. It does not download/install/execute it.
func LoadViewer(explicit string) (string, error) {
	paths := []string{}
	if explicit != "" {
		paths = append(paths, explicit)
	} else if runtime.GOOS == "darwin" {
		paths = append(paths, "/Applications/Whale Isle.app/Contents/Resources/app.asar")
		if home, e := os.UserHomeDir(); e == nil {
			paths = append(paths, filepath.Join(home, "Applications/Whale Isle.app/Contents/Resources/app.asar"))
		}
	} else if runtime.GOOS == "windows" {
		for _, root := range []string{os.Getenv("LOCALAPPDATA"), os.Getenv("ProgramFiles")} {
			if root != "" {
				paths = append(paths, filepath.Join(root, "Programs", "Whale Isle", "resources", "app.asar"), filepath.Join(root, "Whale Isle", "resources", "app.asar"))
			}
		}
	}
	for _, path := range paths {
		data, e := readViewer(path)
		if e != nil {
			if explicit != "" {
				return "", e
			}
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != ViewerSHA256 {
			return "", errors.New("OpenUI viewer version/hash is not the supported audited runtime")
		}
		return string(data), nil
	}
	return "", errors.New("OpenUI runtime is not bundled; set WHALEISLE_OPENUI_VIEWER to your supported viewer.html or existing app.asar")
}

type asarNode struct {
	Files    map[string]asarNode `json:"files"`
	Offset   string              `json:"offset"`
	Size     int64               `json:"size"`
	Unpacked bool                `json:"unpacked"`
	Link     string              `json:"link"`
}

func readViewer(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("OpenUI runtime path must be absolute")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return nil, errors.New("OpenUI runtime must be a regular file")
	}
	if filepath.Ext(path) != ".asar" {
		if info.Size() > MaxViewerBytes {
			return nil, errors.New("OpenUI viewer exceeds size limit")
		}
		return io.ReadAll(io.LimitReader(f, MaxViewerBytes+1))
	}
	var h [16]byte
	if _, e = io.ReadFull(f, h[:]); e != nil {
		return nil, e
	}
	headerSize := int64(binary.LittleEndian.Uint32(h[4:8]))
	jsonSize := int64(binary.LittleEndian.Uint32(h[12:16]))
	if binary.LittleEndian.Uint32(h[:4]) != 4 || headerSize < 8 || headerSize > 8<<20 || jsonSize < 2 || jsonSize > headerSize-8 {
		return nil, errors.New("invalid ASAR header")
	}
	raw := make([]byte, jsonSize)
	if _, e = io.ReadFull(f, raw); e != nil {
		return nil, e
	}
	var node asarNode
	if e = json.Unmarshal(raw, &node); e != nil {
		return nil, e
	}
	for _, part := range []string{"src", "renderer", "openui", "viewer.html"} {
		var ok bool
		node, ok = node.Files[part]
		if !ok {
			return nil, errors.New("ASAR does not contain OpenUI viewer")
		}
	}
	offset, e := strconv.ParseInt(node.Offset, 10, 64)
	if e != nil || offset < 0 || node.Size <= 0 || node.Size > MaxViewerBytes || node.Unpacked || node.Link != "" || offset > info.Size()-8-headerSize-node.Size {
		return nil, errors.New("invalid ASAR viewer entry")
	}
	data := make([]byte, node.Size)
	_, e = f.ReadAt(data, 8+headerSize+offset)
	return data, e
}
