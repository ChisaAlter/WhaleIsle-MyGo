package projectflow

import (
	"fmt"
	"testing"
)

func TestProcessFollowReplacesSnapshotAndFoldsDurableEvents(t *testing.T) {
	var state ProcessSnapshot
	var err error
	state, err = FoldProcessFrame(state, []byte(`{"type":"snapshot","cursor":7,"records":[{"type":"event","event":{"type":"user/message","seq":4,"data":{"text":"first"}}}],"hasMore":true}`))
	if err != nil || !state.Live || state.Cursor != 7 || len(state.Records) != 1 {
		t.Fatal(state, err)
	}
	state, err = FoldProcessFrame(state, []byte(`{"type":"event","event":{"type":"assistant/message","seq":8,"data":{"text":"arrived live"}}}`))
	if err != nil || state.Cursor != 8 || len(state.Records) != 2 {
		t.Fatal(state, err)
	}
	state, err = FoldProcessFrame(state, []byte(`{"type":"event","event":{"type":"assistant/message","seq":8,"data":{"text":"duplicate"}}}`))
	if err != nil || len(state.Records) != 2 {
		t.Fatal("duplicate was appended", err)
	}
	// A reconnect snapshot is authoritative; it must not append to an old tail.
	state, err = FoldProcessFrame(state, []byte(`{"type":"snapshot","cursor":9,"records":[],"hasMore":false}`))
	if err != nil || len(state.Records) != 0 || state.Cursor != 9 || state.HasMore {
		t.Fatal(state, err)
	}
	for n := 10; n <= ProcessRecordLimit+10; n++ {
		state, err = FoldProcessFrame(state, []byte(fmt.Sprintf(`{"type":"event","event":{"type":"tool/end","seq":%d,"data":{}}}`, n)))
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(state.Records) != ProcessRecordLimit || !state.Truncated || !state.HasMore {
		t.Fatal("unbounded live process tail")
	}
}
