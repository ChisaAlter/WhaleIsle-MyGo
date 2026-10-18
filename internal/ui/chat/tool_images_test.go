package chat

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestToolImagesFixture(t *testing.T) {
	image := func(id string) json.RawMessage {
		return json.RawMessage(`{"type":"image","attachment":{"attachmentId":"` + id + `","mediaType":"image/png","bytes":100,"width":2,"height":3}}`)
	}
	content := []json.RawMessage{json.RawMessage(`{"type":"text","text":"<path>/original.png</path>"}`), image("second"), image("first")}
	raw, _ := json.Marshal(map[string]any{"data": map[string]any{"message": map[string]any{"content": content}}})
	card := ParseToolCard("tool/result", raw)
	if !reflect.DeepEqual(card.Images, []string{"attachment:second", "attachment:first"}) || !strings.Contains(card.Detail, "/original.png") {
		t.Fatalf("%+v", card)
	}
	content = append(content, json.RawMessage(`{"type":"image","attachment":{"attachmentId":"bad","mediaType":"image/png","bytes":1,"width":0,"height":1}}`))
	if refs := toolImageReferences(content); refs != nil {
		t.Fatalf("partial gallery: %v", refs)
	}
	raw, _ = json.Marshal(map[string]any{"data": map[string]any{"message": map[string]any{"content": content}}})
	if card := ParseToolCard("tool/result", raw); len(card.Images) != 0 || !strings.Contains(card.Detail, "attachmentId") {
		t.Fatalf("missing fallback: %+v", card)
	}

	var cache attachmentLoadCache
	var calls atomic.Int32
	entered, release, done := make(chan struct{}, 2), make(chan struct{}), make(chan struct{}, 2)
	fetch := func(context.Context, string) (string, error) {
		n := calls.Add(1)
		entered <- struct{}{}
		if n == 1 {
			<-release
			return "", errors.New("unavailable")
		}
		return "fixture.png", nil
	}
	invalidate := func() { done <- struct{}{} }
	cache.ensure("first", fetch, invalidate)
	<-entered
	for range 20 {
		cache.ensure("first", fetch, invalidate)
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate in-flight request")
	}
	close(release)
	<-done
	if state := cache.ensure("first", fetch, invalidate); state.err == nil {
		t.Fatal("missing failed state")
	}
	if calls.Load() != 1 {
		t.Fatal("error auto-retried")
	}
	cache.retry("first")
	cache.ensure("first", fetch, invalidate)
	<-entered
	<-done
	if state := cache.ensure("first", fetch, invalidate); state.path != "fixture.png" || state.err != nil {
		t.Fatalf("retry: %+v", state)
	}
}
