package openui

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenUIBoundaryAndResourceFixture(t *testing.T) {
	id := Identity("host-a", "session-a", "message-a", 12)
	if id == Identity("host-b", "session-a", "message-a", 12) || id == Identity("host-a", "session-a", "message-a", 13) {
		t.Fatal("owner/block collision")
	}
	state := json.RawMessage(`{"value":3}`)
	r := Render{InstanceID: id, Program: "main=Text(\"</script>\")", State: state}
	shell, err := Shell("<html>synthetic viewer only</html>", r, 900, "dark")
	if err != nil {
		t.Fatal(err)
	}
	for _, needed := range []string{"connect-src &#39;none&#39;", "worker-src &#39;none&#39;", "event.source!==frame.contentWindow", "event.origin!=='null'", "__whaleOpenUIDrain", "allow-scripts allow-forms"} {
		if !strings.Contains(shell, needed) {
			t.Fatal("missing boundary", needed)
		}
	}
	if strings.Contains(shell, "main=Text(\"</script>") {
		t.Fatal("program escaped parent script")
	}
	raw, _ := json.Marshal(Event{Channel: "whale-openui", Version: 1, InstanceID: id, Type: "state", State: state})
	if _, err = ValidateEvent(raw, id); err != nil {
		t.Fatal(err)
	}
	if _, err = ValidateEvent(raw, "other"); err == nil {
		t.Fatal("cross instance event")
	}
	if ValidateState(json.RawMessage(`[]`)) == nil {
		t.Fatal("nonobject state")
	}
	root := t.TempDir()
	if err = SaveState(root, id, state); err != nil {
		t.Fatal(err)
	}
	got, err := ReadState(root, id)
	if err != nil || string(got) != string(state) {
		t.Fatal("state persistence", err)
	}
	viewer := []byte("synthetic resource")
	node := asarNode{Files: map[string]asarNode{"src": {Files: map[string]asarNode{"renderer": {Files: map[string]asarNode{"openui": {Files: map[string]asarNode{"viewer.html": {Offset: "0", Size: int64(len(viewer))}}}}}}}}}
	header, _ := json.Marshal(node)
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint32(buf, 4)
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(header)+8))
	binary.LittleEndian.PutUint32(buf[8:], uint32(len(header)+4))
	binary.LittleEndian.PutUint32(buf[12:], uint32(len(header)))
	buf = append(buf, header...)
	buf = append(buf, viewer...)
	path := filepath.Join(root, "app.asar")
	os.WriteFile(path, buf, 0600)
	got, err = readViewer(path)
	if err != nil || string(got) != string(viewer) {
		t.Fatal("static ASAR extraction", err)
	}
	if _, err = LoadViewer(path); err == nil {
		t.Fatal("unverified runtime accepted")
	}
}
