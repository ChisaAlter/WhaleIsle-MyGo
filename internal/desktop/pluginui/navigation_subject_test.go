package pluginui

import (
	"encoding/json"
	"os/exec"
	"testing"

	"whaleisle/internal/processutil"
)

// Go leaves an empty Subject name out of the JSON it sends, so the page used to
// receive a main-navigation subject without a name and then died on
// requested.name.startsWith(...): every later snapshot was lost and a selected
// plugin page stayed on "Loading plugin view…" for good.
func TestNavigationSubjectFromGoAlwaysHasAName(t *testing.T) {
	var sent map[string]any
	raw, err := json.Marshal(map[string]any{"type": "select", "subject": Subject{Kind: "navigation"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &sent); err != nil {
		t.Fatal(err)
	}
	if _, has := sent["subject"].(map[string]any)["name"]; has {
		t.Fatal("this test documents that Go omits the empty name; the page must cope")
	}

	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(string(navigationSource))
	if err != nil {
		t.Fatal(err)
	}
	subject, err := json.Marshal(sent["subject"])
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const context=vm.createContext({assert,sent:` + string(subject) + `});
vm.runInContext(` + string(source) + `,context);
vm.runInContext('(' + function(){
 const subject=whaleNavigationSubject(sent);
 assert.equal(subject.kind,'navigation');
 assert.equal(subject.name,'');
 assert.equal(subject.name.startsWith('settings:'),false);
 assert.equal(whaleNavigationSubject({kind:'view',name:'main:a:b'}).name,'main:a:b');
}.toString() + ')()',context);`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("navigation subject: %v\n%s", err, out)
	}
}
