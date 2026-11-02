package petgrowth

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func resetScanCache() {
	scanCache.Lock()
	scanCache.root, scanCache.dirs = "", nil
	scanCache.Unlock()
}

func TestMergeScanOrderCacheAndBudget(t *testing.T) {
	resetScanCache()
	defer resetScanCache()
	id := strings.Repeat("a", 64)
	other := strings.Repeat("b", 64)
	first := []scanResultDir{{R: "one", S: "s1", D: map[string]int64{id: 1}}, {R: "two", S: "s2", D: map[string]int64{id: 2, other: 5}}}
	got, err := mergeScan("root", first, true)
	if err != nil || got[id] != 2 || got[other] != 5 {
		t.Fatalf("later directory must win: %v %v", got, err)
	}
	cached := []scanResultDir{{R: "one", S: "s1"}, {R: "two", S: "s2"}}
	if again, err := mergeScan("root", cached, true); err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("unchanged directories from cache: %v %v", again, err)
	}
	if _, err := mergeScan("root", []scanResultDir{{R: "one", S: "stale"}}, true); err == nil {
		t.Fatal("a cached claim with another signature was accepted")
	}
	if _, err := mergeScan("root", cached, true); err == nil {
		t.Fatal("cache survived an out-of-step scan")
	}
	if _, err := mergeScan("root", first[:1], false); err == nil {
		t.Fatal("an incomplete scan returned samples")
	}
	if got, err := mergeScan("root", []scanResultDir{{R: "one", S: "s1"}, {R: "two", S: "s2", D: map[string]int64{other: 5}}}, true); err != nil || got[id] != 1 || got[other] != 5 {
		t.Fatalf("finished directory of an incomplete scan was not kept: %v %v", got, err)
	}
}

func scanTestNode(t *testing.T) string {
	t.Helper()
	candidates := []string{os.Getenv("WHALEISLE_TEST_NODE")}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "..", "..", "runtime", "node", "node.exe"))
	}
	if p, err := exec.LookPath("node"); err == nil {
		candidates = append(candidates, p)
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if exec.Command(c, "-e", "process.exit(typeof require('zlib').zstdCompressSync==='function'?0:1)").Run() == nil {
			return c
		}
	}
	t.Skip("no node runtime with zstd")
	return ""
}

func writeLogFrame(t *testing.T, node, file string, events ...string) {
	t.Helper()
	cmd := exec.Command(node, "-e", "const fs=require('fs'),z=require('zlib');fs.appendFileSync(process.argv[1],z.zstdCompressSync(Buffer.from(fs.readFileSync(0,'utf8'))))", file)
	cmd.Stdin = strings.NewReader(strings.Join(events, "\n") + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("write log: %v %s", err, out)
	}
}

func usageEvent(seq, turn, step, tokens int) string {
	return fmt.Sprintf(`{"type":"assistant/message","seq":%d,"data":{"turn":%d,"step":%d,"usage":{"inputTokens":%d,"outputTokens":1}}}`, seq, turn, step, tokens)
}

func TestScanIncrementalMatchesFullScan(t *testing.T) {
	node := scanTestNode(t)
	resetScanCache()
	defer resetScanCache()
	home := t.TempDir()
	for _, dir := range []string{"ws/session-a", "ws/session-b"} {
		if err := os.MkdirAll(filepath.Join(home, "sessions", filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	a := filepath.Join(home, "sessions", "ws", "session-a", "session.v4.jsonl.zstd")
	b := filepath.Join(home, "sessions", "ws", "session-b", "session.v4.jsonl.zstd")
	writeLogFrame(t, node, a, `{"type":"session/header","seq":0}`, usageEvent(1, 1, 1, 10), usageEvent(2, 1, 1, 20))
	writeLogFrame(t, node, b, usageEvent(1, 1, 1, 7))
	first, err := Scan(context.Background(), node, home)
	if err != nil || len(first) != 2 {
		t.Fatalf("first scan: %v %v", first, err)
	}
	writeLogFrame(t, node, b, usageEvent(2, 2, 1, 3))
	incremental, err := Scan(context.Background(), node, home)
	if err != nil || len(incremental) != 3 {
		t.Fatalf("incremental scan: %v %v", incremental, err)
	}
	resetScanCache()
	full, err := Scan(context.Background(), node, home)
	if err != nil || !reflect.DeepEqual(incremental, full) {
		t.Fatalf("incremental %v differs from full %v (%v)", incremental, full, err)
	}
}
