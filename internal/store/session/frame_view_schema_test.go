package session

import (
	"encoding/json"
	"reflect"
	"testing"

	"whaleisle/internal/dsh/events"
)

// A frame view shares only the request-header tool schemas with its source;
// Snapshot clones keep copying them, and every other frame field stays private.
func TestFrameViewSharesOnlyToolSchemas(t *testing.T) {
	params := map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}}}
	header := TrajectoryRequestHeader{Seq: 1, Header: RequestInspectionHeader{Tools: []events.RequestHeaderDataHeaderToolsItem{{Name: "search", Parameters: params}}}}
	in := ViewState{
		TrajectoryRequests:       []TrajectoryRequest{{ID: "r", Header: &header, Events: []json.RawMessage{json.RawMessage(`{"type":"x"}`)}}},
		TrajectoryRequestHeaders: []TrajectoryRequestHeader{header},
	}
	same := func(a map[string]any) bool {
		return reflect.ValueOf(a).UnsafePointer() == reflect.ValueOf(params).UnsafePointer()
	}

	frame := cloneFrameViewState(in)
	if !same(frame.TrajectoryRequests[0].Header.Header.Tools[0].Parameters) || !same(frame.TrajectoryRequestHeaders[0].Header.Tools[0].Parameters) {
		t.Fatal("frame view copied the tool schemas")
	}
	if frame.TrajectoryRequests[0].Header == &header || &frame.TrajectoryRequests[0].Header.Header.Tools[0] == &header.Header.Tools[0] || &frame.TrajectoryRequestHeaders[0].Header.Tools[0] == &in.TrajectoryRequestHeaders[0].Header.Tools[0] {
		t.Fatal("frame view aliased the request header")
	}
	frame.TrajectoryRequests[0].Events[0][0] ^= 1
	if string(in.TrajectoryRequests[0].Events[0]) != `{"type":"x"}` {
		t.Fatal("frame view aliased trajectory events")
	}

	full := cloneViewState(in)
	if same(full.TrajectoryRequests[0].Header.Header.Tools[0].Parameters) || same(full.TrajectoryRequestHeaders[0].Header.Tools[0].Parameters) {
		t.Fatal("snapshot clone shared the tool schemas")
	}
	if !reflect.DeepEqual(full.TrajectoryRequests[0].Header.Header.Tools[0].Parameters, params) {
		t.Fatal("snapshot clone changed the tool schema")
	}
}
