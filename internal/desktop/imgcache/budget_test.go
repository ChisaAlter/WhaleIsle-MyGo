package imgcache

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCacheEnforcesMemoryEntryAndDiskByteBudgets(t *testing.T) {
	payload := bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 1024) // 4 KiB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	c := New(dir)
	c.SetHTTPClient(srv.Client())
	c.SetMaxEntries(8)
	c.SetDiskBudget(int64(len(payload) * 5)) // room for at most 5 files

	for i := 0; i < 15; i++ {
		u := fmt.Sprintf("%s/img-%02d.png", srv.URL, i)
		if _, err := c.Fetch(context.Background(), u); err != nil {
			t.Fatalf("Fetch %d failed: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	c.mu.Lock()
	memCount := len(c.entries)
	c.mu.Unlock()
	if memCount > 8 {
		t.Fatalf("in-memory entries exceeded LRU cap: %d > 8", memCount)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var diskBytes int64
	for _, de := range entries {
		info, err := de.Info()
		if err != nil {
			t.Fatal(err)
		}
		diskBytes += info.Size()
	}
	if diskBytes > int64(len(payload)*5) {
		t.Fatalf("disk cache exceeded budget: %d > %d", diskBytes, len(payload)*5)
	}
}
