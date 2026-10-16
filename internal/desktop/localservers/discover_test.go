package localservers

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"testing"
)

func TestDiscoverFixedPortsAndSortedURLs(t *testing.T) {
	var mu sync.Mutex
	var visited []int
	got := discover(context.Background(), func(_ context.Context, port int) bool {
		mu.Lock()
		visited = append(visited, port)
		mu.Unlock()
		return port == 8080 || port == 3000
	})
	sort.Ints(visited)
	wantPorts := []int{3000, 3001, 3333, 4173, 4200, 4321, 5000, 5173, 5174, 5175, 5500, 8000, 8080, 8081, 8888, 9000}
	if !reflect.DeepEqual(visited, wantPorts) {
		t.Fatalf("probed ports %v", visited)
	}
	want := []Server{{URL: "http://127.0.0.1:3000", Port: 3000}, {URL: "http://127.0.0.1:8080", Port: 8080}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("servers = %#v", got)
	}
}
func TestDiscoverCanceledDiscardsResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := discover(ctx, func(context.Context, int) bool { return true })
	if len(got) != 0 {
		t.Fatalf("canceled discovery returned %#v", got)
	}
}
