package pluginui

import (
	"encoding/json"
	"os/exec"
	"testing"

	"whaleisle/internal/processutil"
)

// A frame paints in several passes and the final-placement overlay can run
// before the content that built the seats. That must not revoke a layout the
// page already acknowledged, or the handshake restarts on every frame.
func TestEndNativeFrameKeepsAcknowledgedLayoutUntilBuiltContentHasPainted(t *testing.T) {
	subject := Subject{Kind: "view", Name: "settings.section:fixture:plugin"}
	page := &bridgePage{id: 3}
	applied := &richLayout{page: page, revision: 7, subject: subject}
	owner := func(frame, declared, painted uint64) *Owner {
		return &Owner{selected: subject, page: page, state: Snapshot{Connected: true, Page: 3, Revision: 7, Subject: &subject}, rich: richState{enabled: true, begun: true, frame: frame, declared: declared, painted: painted, applied: applied}}
	}

	// The seat was built in this frame but its content has not painted yet.
	o := owner(5, 5, 4)
	o.EndNativeFrame()
	if o.rich.applied != applied {
		t.Fatal("an earlier overlay pass revoked a layout whose content is still to paint")
	}

	// The content painted in this frame and registered no seat (scrolled out of
	// view, clipped away): the layout is revoked.
	o = owner(5, 5, 5)
	o.EndNativeFrame()
	if o.rich.applied != nil {
		t.Fatal("a painted frame without a visible seat kept the layout")
	}

	// No seat was built in this frame at all (the page was left): revoked.
	o = owner(5, 4, 4)
	o.EndNativeFrame()
	if o.rich.applied != nil {
		t.Fatal("a frame without any built seat kept the layout")
	}

	// A masked owner always revokes, whatever the passes did.
	o = owner(5, 5, 4)
	o.masked = true
	o.EndNativeFrame()
	if o.rich.applied != nil {
		t.Fatal("a masked owner kept the layout")
	}
}

// Seating a node makes it fill its seat. Its geometry then reads as a full
// screen fragment, so measuring it again changes the published document with
// every layout and the page never settles.
func TestSeatedFragmentKeepsItsUnseatedMeasurement(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(string(nativeDocumentSource))
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const state={seated:false,geometry:{width:846,height:73.25,left:0,top:0},style:{position:'static',display:'block'}};
const context=vm.createContext({assert,state,innerWidth:564,innerHeight:73,getComputedStyle:()=>state.style});
context.whaleRichSurface={hidden:()=>false,portals:()=>[],applied:()=>state.seated,rect:()=>state.geometry};
vm.runInContext(` + string(source) + `,context);
vm.runInContext('(' + function(){
 const root={isConnected:true,hasAttribute:()=>false,getAttribute:()=>null,children:[],tagName:'DIV',matches:()=>false,querySelectorAll:()=>[]};
 const measure=()=>whaleNativeDocument(root,()=>'h',new Map(),{},{left:10})[0];
 const free=measure();
 assert.equal(free.kind,'rich');assert.equal(free.fullScreen,false);assert.equal(free.inline,false);
 assert.equal(free.width,0,'a plain fragment is sized by its seat, so its width is not published');
 assert.equal(free.height,73.25);
 // Seated: position:fixed and exactly the viewport size.
 state.seated=true;state.geometry={width:564,height:73.25,left:0,top:0};state.style={position:'fixed',display:'block'};
 assert.deepEqual(measure(),free,'a seated fragment must publish what it measured before it was seated');
 // Released again.
 state.seated=false;state.geometry={width:846,height:73.25,left:0,top:0};state.style={position:'static',display:'block'};
 assert.deepEqual(measure(),free);
}.toString() + ')()',context);`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("seated fragment measurement: %v\n%s", err, out)
	}
}
