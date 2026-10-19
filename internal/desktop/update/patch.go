package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// PatchOptions configures ApplyBlockPatch.
type PatchOptions struct {
	// Zero retains the legacy 512 MiB bound. Native catalog downloads pass their
	// existing 8 GiB installer limit through reconstruction and full fallback.
	MaxBytes          int64
	OnProgress        func(int64, int64)
	OnPhase           func(StagePhase)
	OldPath           string
	Feed              Feed
	BlockMap          BlockMapInfo
	DestDir           string
	Start             bool
	HTTPClient        *http.Client
	RequireSignature  bool
	ExpectedPublisher string
}

// ApplyBlockPatch reconstructs Feed.URL into DestDir by copying unchanged
// blocks from oldPath (matched by offset + SHA-256 against blockMap) and
// fetching only missing/changed ranges via HTTP Range.
//
// If the origin does not honor Range, it falls back to a full GET of Feed.URL
// then VerifyBlockMap. Verification failure is fail-closed (staged file removed).
func ApplyBlockPatch(ctx context.Context, oldPath string, feed Feed, blockMap BlockMapInfo) (ApplyResult, error) {
	return ApplyBlockPatchOpts(ctx, PatchOptions{
		OldPath:  oldPath,
		Feed:     feed,
		BlockMap: blockMap,
	})
}

// ApplyBlockPatchOpts is ApplyBlockPatch with dest, client, and signature options.
func ApplyBlockPatchOpts(ctx context.Context, opt PatchOptions) (ApplyResult, error) {
	if ctx.Err() != nil {
		return ApplyResult{}, ctx.Err()
	}
	if err := validateBlockMap(opt.BlockMap); err != nil {
		return ApplyResult{}, err
	}
	oldPath := strings.TrimSpace(opt.OldPath)
	if oldPath == "" {
		return ApplyResult{}, fmt.Errorf("empty old installer path")
	}
	old, err := os.Open(oldPath)
	if err != nil {
		return ApplyResult{}, err
	}
	defer old.Close()
	st, err := old.Stat()
	if err != nil {
		return ApplyResult{}, err
	}
	if !st.Mode().IsRegular() {
		return ApplyResult{}, fmt.Errorf("old installer is not a regular file")
	}
	oldSize := st.Size()
	releaseURL := strings.TrimSpace(opt.Feed.URL)
	if releaseURL == "" {
		return ApplyResult{}, fmt.Errorf("empty release URL")
	}
	client := opt.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}

	need := make([]int, 0, len(opt.BlockMap.Blocks))
	for i, b := range opt.BlockMap.Blocks {
		if !blockReusable(old, oldSize, b) {
			need = append(need, i)
		}
	}

	if len(need) > 0 {
		first := opt.BlockMap.Blocks[need[0]]
		reportStagePhase(ctx, opt.OnPhase, StageDownloading)
		data, err := FetchRange(ctx, releaseURL, first.Offset, first.Size, client)
		if errors.Is(err, ErrRangeUnsupported) {
			return downloadFullAndVerify(ctx, opt)
		}
		if err != nil {
			return ApplyResult{}, err
		}
		return reconstructFromBlocks(ctx, opt, old, oldSize, need, data)
	}
	return reconstructFromBlocks(ctx, opt, old, oldSize, nil, nil)
}

func reconstructFromBlocks(ctx context.Context, opt PatchOptions, old *os.File, oldSize int64, need []int, firstRange []byte) (ApplyResult, error) {
	info := opt.BlockMap
	client := opt.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	releaseURL := strings.TrimSpace(opt.Feed.URL)

	destDir := opt.DestDir
	if destDir == "" {
		destDir = filepath.Join(os.TempDir(), "whaleisle-updates")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ApplyResult{}, err
	}
	dest := stageFileName(destDir, releaseURL)
	part := dest
	if sameFilePath(opt.OldPath, dest) {
		part = dest + ".part"
	}

	outSize := info.FileSize
	if outSize <= 0 {
		for _, b := range info.Blocks {
			if end := b.Offset + b.Size; end > outSize {
				outSize = end
			}
		}
	}
	maxBytes := opt.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 512 << 20
	}
	if outSize > maxBytes {
		return ApplyResult{}, fmt.Errorf("reconstructed size %d exceeds limit", outSize)
	}

	f, err := os.Create(part)
	if err != nil {
		return ApplyResult{}, err
	}
	cleanup := true
	defer func() {
		_ = f.Close()
		if cleanup {
			_ = os.Remove(part)
			if part != dest {
				_ = os.Remove(dest)
			}
		}
	}()
	if err := f.Truncate(outSize); err != nil {
		return ApplyResult{}, err
	}

	needSet := make(map[int]struct{}, len(need))
	for _, i := range need {
		needSet[i] = struct{}{}
	}
	copied, fetched := 0, 0
	var completed int64
	progress := func(n int64) {
		completed += n
		if opt.OnProgress != nil {
			opt.OnProgress(completed, outSize)
		}
	}
	for i, b := range info.Blocks {
		if ctx.Err() != nil {
			return ApplyResult{}, ctx.Err()
		}
		if _, err := f.Seek(b.Offset, io.SeekStart); err != nil {
			return ApplyResult{}, fmt.Errorf("block %d: seek dest: %w", i, err)
		}
		if _, ok := needSet[i]; !ok {
			if _, err := old.Seek(b.Offset, io.SeekStart); err != nil {
				return ApplyResult{}, fmt.Errorf("block %d: seek old: %w", i, err)
			}
			if _, err := io.CopyN(f, old, b.Size); err != nil {
				return ApplyResult{}, fmt.Errorf("block %d: copy old: %w", i, err)
			}
			copied++
			progress(b.Size)
			continue
		}
		var data []byte
		if fetched == 0 && firstRange != nil {
			data = firstRange
		} else {
			var err error
			data, err = FetchRange(ctx, releaseURL, b.Offset, b.Size, client)
			if err != nil {
				return ApplyResult{}, fmt.Errorf("block %d: %w", i, err)
			}
		}
		if int64(len(data)) != b.Size {
			return ApplyResult{}, fmt.Errorf("block %d: range length mismatch", i)
		}
		if _, err := f.Write(data); err != nil {
			return ApplyResult{}, fmt.Errorf("block %d: write: %w", i, err)
		}
		fetched++
		progress(b.Size)
	}
	if err := f.Close(); err != nil {
		return ApplyResult{}, err
	}

	if part != dest {
		if err := os.Rename(part, dest); err != nil {
			return ApplyResult{}, err
		}
	}

	reportStagePhase(ctx, opt.OnPhase, StageVerifying)
	sum, n, err := hashFileBounded(dest, maxBytes)
	if err != nil {
		return ApplyResult{}, err
	}
	if opt.Feed.Size > 0 && n != opt.Feed.Size {
		return ApplyResult{}, fmt.Errorf("size mismatch: got %d want %d", n, opt.Feed.Size)
	}
	expect := strings.ToLower(strings.TrimSpace(opt.Feed.SHA256))
	if expect != "" && expect != sum {
		return ApplyResult{}, fmt.Errorf("sha256 mismatch")
	}
	if err := VerifyBlockMap(dest, info); err != nil {
		return ApplyResult{}, fmt.Errorf("blockmap verify failed: %w", err)
	}

	sig, err := CheckSignature(ctx, dest, VerifyOptions{
		RequireSignature:  opt.RequireSignature,
		ExpectedPublisher: opt.ExpectedPublisher,
	})
	if err != nil {
		return ApplyResult{}, err
	}

	if err := ctx.Err(); err != nil {
		return ApplyResult{}, err
	}
	cleanup = false
	out := ApplyResult{
		Path:          dest,
		Message:       fmt.Sprintf("staged %s; blockmap verified (block patch; copied %d, fetched %d", dest, copied, fetched),
		SHA256:        sum,
		Size:          n,
		Signature:     sig.Status,
		Publisher:     sig.Publisher,
		UsedRange:     fetched > 0,
		CopiedBlocks:  copied,
		FetchedBlocks: fetched,
	}
	if fetched > 0 {
		out.Message += " via Range)"
	} else {
		out.Message += ")"
	}
	if expect != "" {
		out.Message += "; sha256 ok"
	}
	if sig.Status == SignatureValid {
		label := "signature"
		if runtime.GOOS == "windows" {
			label = "authenticode"
		} else if runtime.GOOS == "darwin" {
			label = "codesign"
		}
		out.Message += "; " + label + " ok"
	}
	if opt.Start && CanStartStaged(out) {
		return StartStaged(out)
	}
	return out, nil
}

func downloadFullAndVerify(ctx context.Context, opt PatchOptions) (ApplyResult, error) {
	res, err := DownloadAndStageOpts(ctx, StageOptions{
		ReleaseURL:        opt.Feed.URL,
		DestDir:           opt.DestDir,
		Start:             false,
		HTTPClient:        opt.HTTPClient,
		ExpectSHA256:      opt.Feed.SHA256,
		ExpectSize:        opt.Feed.Size,
		MaxBytes:          opt.MaxBytes,
		OnProgress:        opt.OnProgress,
		OnPhase:           opt.OnPhase,
		RequireSignature:  opt.RequireSignature,
		ExpectedPublisher: opt.ExpectedPublisher,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	if err := VerifyBlockMap(res.Path, opt.BlockMap); err != nil {
		_ = os.Remove(res.Path)
		return ApplyResult{}, fmt.Errorf("blockmap verify failed: %w", err)
	}
	res.Message += "; blockmap verified (full download; Range unsupported)"
	res.UsedRange = false
	if opt.Start && CanStartStaged(res) {
		return StartStaged(res)
	}
	return res, nil
}

func blockReusable(old *os.File, oldSize int64, b BlockEntry) bool {
	if b.Offset < 0 || b.Size <= 0 || b.Offset+b.Size > oldSize {
		return false
	}
	if _, err := old.Seek(b.Offset, io.SeekStart); err != nil {
		return false
	}
	h := sha256.New()
	if _, err := io.CopyN(h, old, b.Size); err != nil {
		return false
	}
	got := hex.EncodeToString(h.Sum(nil))
	want := strings.ToLower(strings.TrimSpace(b.SHA256))
	return got == want
}

func hashFile(path string) (string, int64, error) { return hashFileBounded(path, 512<<20) }

func hashFileBounded(path string, maxBytes int64) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxBytes+1))
	if err != nil {
		return "", 0, err
	}
	if n > maxBytes {
		return "", 0, fmt.Errorf("installer exceeds size limit")
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func stageFileName(destDir, releaseURL string) string {
	name := filepath.Base(strings.Split(releaseURL, "?")[0])
	if name == "" || name == "." || name == "/" {
		name = "WhaleIsle-update.exe"
	}
	return filepath.Join(destDir, name)
}

func sameFilePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return strings.EqualFold(aa, bb)
}
