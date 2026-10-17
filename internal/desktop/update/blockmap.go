package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// BlockEntry describes one hashed byte range of a release artifact.
type BlockEntry struct {
	Offset int64  `json:"offset"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// BlockMapInfo is the minimal blockmap JSON document:
//
//	{"fileSize":12345,"blocks":[{"offset":0,"size":4096,"sha256":"…"},…]}
//
// fileSize is optional. ApplyBlockPatch copies matching blocks from a local
// previous installer and Range-fetches the rest when the origin honors Range.
type BlockMapInfo struct {
	FileSize int64        `json:"fileSize,omitempty"`
	Blocks   []BlockEntry `json:"blocks"`
}

// ParseBlockMap unmarshals a blockmap JSON document (object with blocks[]).
func ParseBlockMap(data []byte) (BlockMapInfo, error) {
	var info BlockMapInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return BlockMapInfo{}, fmt.Errorf("invalid blockmap JSON: %w", err)
	}
	if err := validateBlockMap(info); err != nil {
		return BlockMapInfo{}, err
	}
	return info, nil
}

// DownloadBlockMap fetches and parses a blockmap JSON document from url.
func DownloadBlockMap(ctx context.Context, url string, client *http.Client) (BlockMapInfo, error) {
	if ctx.Err() != nil {
		return BlockMapInfo{}, ctx.Err()
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return BlockMapInfo{}, fmt.Errorf("empty blockmap URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return BlockMapInfo{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return BlockMapInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return BlockMapInfo{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return BlockMapInfo{}, fmt.Errorf("blockmap HTTP %d", resp.StatusCode)
	}
	return ParseBlockMap(body)
}

// CanIncremental reports whether an incremental (blockmap-assisted) update path
// is eligible for the given running version and feed.
//
// Heuristic only: true when a blockmap is present (inline BlockMap with at
// least one block, or non-empty BlockMapURL) AND the feed version shares the
// same major channel as currentVersion. False means callers should use the
// existing full-file DownloadAndStage path (ApplyIncremental also returns an
// error directing them there).
//
// When true, ApplyIncremental uses ApplyBlockPatch if OldPath exists; otherwise
// it full-downloads then VerifyBlockMap.
func CanIncremental(currentVersion string, feed Feed) bool {
	if !hasBlockMap(feed) {
		return false
	}
	current := strings.TrimSpace(currentVersion)
	latest := strings.TrimSpace(feed.Version)
	if current == "" || latest == "" {
		return false
	}
	return sameMajor(current, latest)
}

func hasBlockMap(feed Feed) bool {
	if strings.TrimSpace(feed.BlockMapURL) != "" {
		return true
	}
	return feed.BlockMap != nil && len(feed.BlockMap.Blocks) > 0
}

func sameMajor(a, b string) bool {
	ap := versionParts(a)
	bp := versionParts(b)
	if len(ap) == 0 || len(bp) == 0 {
		return false
	}
	return ap[0] == bp[0]
}

func validateBlockMap(info BlockMapInfo) error {
	if len(info.Blocks) == 0 {
		return fmt.Errorf("blockmap missing blocks")
	}
	for i, b := range info.Blocks {
		if b.Offset < 0 {
			return fmt.Errorf("block %d: negative offset", i)
		}
		if b.Size <= 0 {
			return fmt.Errorf("block %d: size must be > 0", i)
		}
		if strings.TrimSpace(b.SHA256) == "" {
			return fmt.Errorf("block %d: missing sha256", i)
		}
	}
	return nil
}

// VerifyBlockMap checks that filePath matches info: when FileSize > 0 the on-disk
// length must equal it, and each block's SHA-256 must match the bytes at
// [Offset, Offset+Size). Digests are compared case-insensitively as hex.
func VerifyBlockMap(filePath string, info BlockMapInfo) error {
	if err := validateBlockMap(info); err != nil {
		return err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	size := st.Size()
	if info.FileSize > 0 && size != info.FileSize {
		return fmt.Errorf("blockmap fileSize mismatch: got %d want %d", size, info.FileSize)
	}
	for i, b := range info.Blocks {
		if b.Offset+b.Size > size {
			return fmt.Errorf("block %d: range [%d,%d) exceeds file size %d", i, b.Offset, b.Offset+b.Size, size)
		}
		if _, err := f.Seek(b.Offset, io.SeekStart); err != nil {
			return fmt.Errorf("block %d: seek: %w", i, err)
		}
		h := sha256.New()
		if _, err := io.CopyN(h, f, b.Size); err != nil {
			return fmt.Errorf("block %d: read: %w", i, err)
		}
		got := hex.EncodeToString(h.Sum(nil))
		want := strings.ToLower(strings.TrimSpace(b.SHA256))
		if got != want {
			return fmt.Errorf("block %d: sha256 mismatch", i)
		}
	}
	return nil
}

// resolveBlockMap returns an inline map or downloads Feed.BlockMapURL.
// Prefer URL when set so remote documents can refresh independently of the feed.
func resolveBlockMap(ctx context.Context, feed Feed, client *http.Client) (BlockMapInfo, error) {
	if url := strings.TrimSpace(feed.BlockMapURL); url != "" {
		return DownloadBlockMap(ctx, url, client)
	}
	if feed.BlockMap != nil {
		info := *feed.BlockMap
		if err := validateBlockMap(info); err != nil {
			return BlockMapInfo{}, err
		}
		return info, nil
	}
	return BlockMapInfo{}, fmt.Errorf("no blockmap: use DownloadAndStage / DownloadAndStageOpts for full-file update")
}
