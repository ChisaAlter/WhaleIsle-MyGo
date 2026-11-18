package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestAgentNavigationImagesAsyncReturnsBeforeQueuedIO(t *testing.T) {
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
	s := New()
	s.Attach(client, nil)
	defer s.Detach()
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "continuable"}
	s.mu.Lock()
	s.selectedID = address.ParentSessionID
	s.items = []Item{{SessionID: address.ParentSessionID, AgentAvailable: true}}
	s.planHistoryBinding = fmt.Sprintf("%p\x00%s", client, s.selectedID)
	s.agentNavigation = agentNavigationState{client: client, mux: s.mux, generation: s.followGeneration, binding: s.planHistoryBinding, root: s.selectedID, view: AgentNavigationView{Token: 12, SessionID: address.ChildSessionID, Address: address}}
	s.mu.Unlock()
	file := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(file, []byte("\x89PNG\r\n\x1a\nimage"), 0600); err != nil {
		t.Fatal(err)
	}
	childImageWorkers <- struct{}{}
	childImageWorkers <- struct{}{}
	blocked := true
	defer func() {
		if blocked {
			<-childImageWorkers
			<-childImageWorkers
		}
	}()
	returned := make(chan struct{})
	go func() { s.AddAgentNavigationImagesAsync(12, address, []string{file}, nil); close(returned) }()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("native admission waited for file worker")
	}
	if !s.AgentNavigationSnapshot().ImagesPreparing {
		t.Fatal("preparation state not published")
	}
	<-childImageWorkers
	<-childImageWorkers
	blocked = false
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && s.AgentNavigationSnapshot().ImagesPreparing {
		time.Sleep(time.Millisecond)
	}
	if got := s.AgentNavigationSnapshot(); got.ImagesPreparing || len(got.PromptImages) != 1 || got.PromptImages[0] != file {
		t.Fatal("worker result was not returned to its owner")
	}
}
