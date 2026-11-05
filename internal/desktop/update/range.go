package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrRangeUnsupported means the origin ignored a Range request (typically HTTP
// 200 with a full body) or otherwise does not honor byte ranges. Callers should
// fall back to a full GET.
var ErrRangeUnsupported = errors.New("HTTP Range not supported")

const maxRangeBytes = 64 << 20

// FetchRange GETs [offset, offset+size) from url using an HTTP Range request.
//
// A 206 Partial Content response returns exactly size bytes. HTTP 200 (server
// ignored Range) returns ErrRangeUnsupported so callers can fall back to a
// full-file download. Other non-2xx statuses are errors.
func FetchRange(ctx context.Context, url string, offset, size int64, client *http.Client) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty URL")
	}
	if offset < 0 || size <= 0 {
		return nil, fmt.Errorf("invalid range offset=%d size=%d", offset, size)
	}
	if size > maxRangeBytes {
		return nil, fmt.Errorf("range size %d exceeds %d", size, maxRangeBytes)
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute, Transport: updateTransport()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	end := offset + size - 1
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusPartialContent:
		data, err := io.ReadAll(io.LimitReader(resp.Body, size+1))
		if err != nil {
			return nil, err
		}
		if int64(len(data)) != size {
			return nil, fmt.Errorf("range length mismatch: got %d want %d", len(data), size)
		}
		return data, nil
	case http.StatusOK:
		return nil, ErrRangeUnsupported
	case http.StatusRequestedRangeNotSatisfiable:
		return nil, fmt.Errorf("HTTP 416 range not satisfiable")
	default:
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("range HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("range HTTP %d (expected 206)", resp.StatusCode)
	}
}
