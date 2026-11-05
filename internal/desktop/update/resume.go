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
)

type partialIdentity struct {
	URL, SHA256 string
	Size        int64
}

func receiveUpdatePayload(ctx context.Context, opt StageOptions, client *http.Client, work string, maxBytes int64) (int64, string, error) {
	var offset int64
	identity := partialIdentity{opt.ReleaseURL, strings.ToLower(opt.ExpectSHA256), opt.ExpectSize}
	if opt.Resume {
		if identity.SHA256 == "" || identity.Size <= 0 {
			return 0, "", fmt.Errorf("resuming requires the exact installer digest and size")
		}
		var previous partialIdentity
		if raw, err := os.ReadFile(work + ".json"); err == nil && json.Unmarshal(raw, &previous) == nil && previous == identity {
			if st, err := os.Stat(work); err == nil && st.Mode().IsRegular() && st.Size() <= identity.Size {
				offset = st.Size()
			}
		}
		if offset == 0 {
			if err := os.Remove(work); err != nil && !os.IsNotExist(err) {
				return 0, "", err
			}
		}
		raw, _ := json.Marshal(identity)
		if err := os.WriteFile(work+".json", raw, 0600); err != nil {
			return 0, "", err
		}
	}
	var body io.ReadCloser
	if offset == 0 || offset != opt.ExpectSize {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, opt.ReleaseURL, nil)
		if err != nil {
			return 0, "", err
		}
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, "", err
		}
		body = resp.Body
		defer body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return 0, "", fmt.Errorf("download HTTP %d", resp.StatusCode)
		}
		if offset > 0 {
			if resp.StatusCode == http.StatusOK {
				offset = 0
			} else if resp.StatusCode != http.StatusPartialContent || !strings.HasPrefix(resp.Header.Get("Content-Range"), fmt.Sprintf("bytes %d-", offset)) {
				return 0, "", fmt.Errorf("installer resume range mismatch")
			}
		}
	}
	h := sha256.New()
	if offset > 0 {
		f, err := os.Open(work)
		if err != nil {
			return 0, "", err
		}
		n, err := io.CopyN(h, f, offset)
		f.Close()
		if err != nil || n != offset {
			return 0, "", fmt.Errorf("partial installer changed while resuming")
		}
	}
	if opt.OnProgress != nil {
		opt.OnProgress(offset, opt.ExpectSize)
	}
	n := offset
	if body != nil {
		flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		if offset > 0 {
			flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
		}
		f, err := os.OpenFile(work, flags, 0600)
		if err != nil {
			return 0, "", err
		}
		progress := &stageProgressWriter{notify: func(received, total int64) {
			if opt.OnProgress != nil {
				opt.OnProgress(offset+received, total)
			}
		}, total: opt.ExpectSize}
		written, copyErr := io.Copy(io.MultiWriter(f, h, progress), io.LimitReader(body, maxBytes-offset+1))
		n += written
		closeErr := f.Close()
		if copyErr == nil {
			copyErr = closeErr
		}
		if copyErr == nil {
			copyErr = ctx.Err()
		}
		if copyErr == nil && n > maxBytes {
			copyErr = fmt.Errorf("installer exceeds download size limit")
		}
		if copyErr != nil {
			if !opt.Resume {
				_ = os.Remove(work)
			}
			return n, "", copyErr
		}
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}
