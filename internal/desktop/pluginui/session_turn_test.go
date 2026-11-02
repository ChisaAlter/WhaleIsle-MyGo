package pluginui

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"testing"

	"whaleisle/internal/processutil"
)

func TestSessionTurnMarkerPreservesZeroAndRejectsMissingOrInvalid(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict'),context={};vm.createContext(context);vm.runInContext(` + strconv.Quote(string(sessionSource)) + `,context);
	function read(raw,exists=true){return context.whaleSessionTurnOf({closest:()=>exists?{getAttribute:()=>raw}:null})}
	for(const raw of [null,undefined,'',' ','-1','2.5','NaN','Infinity','9007199254740992','0x10','1e2','+1']){const r=read(raw);assert.equal(r.hasTurn,false,String(raw));assert.equal(r.turn,0)}
	assert.equal(read('0',false).hasTurn,false);
	for(const [raw,want] of [['0',0],['1',1],['25',25],[' 7 ',7]]){const r=read(raw);assert.equal(r.hasTurn,true,raw);assert.equal(r.turn,want,raw)}
	console.log('Session turn presence: absent/empty/zero/positive/invalid passed');`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("turn marker: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}

func TestSessionOccurrenceAttributionUpdatesWithoutReplacingHandle(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
	const sent=[],cleanups=[];let mutation,options,turn='0',call='call-a',canonicalTurn='0',endSeq='19';
	const entry={options:{label:'Same content',id:'entry'},registrant:'fixture'};
	const marker={isConnected:true,textContent:'Body',hidden:false,setAttribute(){},removeAttribute(){},contains:()=>false,querySelectorAll:()=>[],scrollHeight:100,getAttribute:name=>name==='data-whale-turn-tail-turn'?canonicalTurn:name==='data-whale-turn-tail-end-seq'?endSeq:null,querySelector:()=>null,closest:selector=>selector==='[data-turn-tail]'?{getAttribute:()=>turn}:selector==='[data-chat-call-id]'?{getAttribute:()=>call}:null};
	const root={getAttribute:()=> 's',querySelectorAll:()=>[marker],contains:()=>true};
	const binding={session:{getSnapshot:()=>({openState:'open'}),subscribe:()=>()=>{}}};
	const socket={readyState:1,addEventListener(type,fn){this[type]=fn},send:message=>sent.push(JSON.parse(message)),close(){}};
	const document={head:{appendChild(){}},documentElement:{},createElement:()=>({remove(){}}),querySelectorAll:()=>[root],querySelector:()=>root};
	const ctx={effect:fn=>cleanups.push(fn()),get:()=>({openSession(){}}),slots:{entriesOfSlot:slot=>slot==='main.conversation'||slot==='conversation.chat.turnTail'?[entry]:[]},sessions:{binding:()=>binding,list:{getSnapshot:()=>({byId:{s:{}}}),subscribe:()=>()=>{}}},layout:{panelInfo:{getSnapshot:()=>({activePanelId:null}),subscribe:()=>()=>{}}},locale:{getSnapshot:()=>({active:'en'}),subscribe:()=>()=>{}},on:()=>()=>{}};
	class Socket{constructor(){return socket}static OPEN=1}
	const context={ctx,window:{__whaleSessionNoOnboarding:true,__whaleSessionEntryOf:(node,slot)=>slot==='conversation.chat.turnTail'?entry:null,addEventListener(){},removeEventListener(){}},location:{search:'?desktopNativeSession=1',href:'http://fixture/'},document,URL,URLSearchParams,WebSocket:Socket,queueMicrotask:fn=>fn(),requestAnimationFrame:fn=>fn(),MutationObserver:class{constructor(fn){mutation=fn}observe(node,config){options=config}disconnect(){}}};
	vm.createContext(context);vm.runInContext(` + strconv.Quote(string(sessionSource)) + `,context);vm.runInContext('whaleBindSessionUI(ctx)',context);
	socket.message({data:JSON.stringify({type:'select',subject:{kind:'session',sessionId:'s',epoch:'1:2'}})});
	let row=sent.at(-1).outlets[0];assert.equal(row.turn,0);assert.equal(row.hasTurn,true);assert.equal(row.callId,'call-a');assert.equal(row.hasTurnTail,true);assert.equal(row.turnTailSeq,19);const handle=row.handle,revision=sent.at(-1).revision;
	assert(options.attributeFilter.includes('data-turn-tail'));assert(options.attributeFilter.includes('data-chat-call-id'));assert(options.attributeFilter.includes('data-whale-turn-tail-end-seq'));
	const choose=(request,overrides={})=>socket.message({data:JSON.stringify({type:'sessionSurface',revision:sent.at(-1).revision,subject:{kind:'session',sessionId:'s',epoch:'1:2'},handle,turnTail:{turn:0,seq:19},request,...overrides})});
	choose('first');assert.equal(sent.at(-1).sessionSurfaceRequest,'first');assert.equal(sent.at(-1).sessionSurfaceHandle,handle);
	const firstAck=sent.at(-1).revision;choose('second');assert.equal(sent.at(-1).sessionSurfaceRequest,'second');assert(sent.at(-1).revision>firstAck);
	const secondAck=sent.at(-1).revision;
	choose('stale-revision',{revision:firstAck});assert.equal(sent.at(-1).revision,secondAck);
	choose('foreign-epoch',{subject:{kind:'session',sessionId:'s',epoch:'1:3'}});assert.equal(sent.at(-1).sessionSurfaceRequest,'second');
	endSeq='20';choose('retargeted');assert.equal(sent.at(-1).sessionSurfaceRequest,'second');endSeq='19';
	socket.message({data:JSON.stringify({type:'surface',slot:''})});assert.equal(sent.at(-1).sessionSurfaceRequest,'');assert.equal(sent.at(-1).sessionSurfaceHandle,'');
	turn='4';canonicalTurn='4';endSeq='29';call='call-b';mutation();row=sent.at(-1).outlets[0];assert.equal(row.handle,handle);assert.equal(row.turn,4);assert.equal(row.hasTurn,true);assert.equal(row.callId,'call-b');assert.equal(row.hasTurnTail,true);assert.equal(row.turnTailSeq,29);assert(sent.at(-1).revision>revision);
	turn=null;call='';mutation();row=sent.at(-1).outlets[0];assert.equal(row.handle,handle);assert.equal(row.hasTurn,false);assert.equal(row.turn,0);assert.equal(row.callId,'');assert.equal(row.hasTurnTail,false);
	cleanups.reverse().forEach(fn=>fn());console.log('Session live attribution update keeps original occurrence handle');`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("occurrence metadata: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}

func TestSessionTurnPresenceWireIsBackwardCompatible(t *testing.T) {
	for _, tc := range []struct {
		wire  string
		turn  int
		known bool
	}{{`{"handle":"a","slot":"conversation.chat.turnTail","turn":0,"hasTurn":true}`, 0, true}, {`{"handle":"a","slot":"conversation.chat.turnTail","turn":3}`, 3, true}, {`{"handle":"a","slot":"conversation.chat.turnTail"}`, 0, false}, {`{"handle":"a","slot":"conversation.chat.turnTail","turn":0}`, 0, false}} {
		var row SessionOutlet
		if err := json.Unmarshal([]byte(tc.wire), &row); err != nil {
			t.Fatal(err)
		}
		if row.Turn != tc.turn || (row.HasTurn || row.Turn > 0) != tc.known || !validSessionOutlets([]SessionOutlet{row}) {
			t.Fatalf("presence lost: %s -> %+v", tc.wire, row)
		}
		encoded, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		var roundtrip SessionOutlet
		if err := json.Unmarshal(encoded, &roundtrip); err != nil {
			t.Fatal(err)
		}
		if roundtrip.HasTurn != row.HasTurn || roundtrip.Turn != row.Turn {
			t.Fatalf("zero presence lost after re-encoding: %s", encoded)
		}
	}
	if validSessionOutlets([]SessionOutlet{{Handle: "a", Slot: "conversation.chat.turnTail", Turn: -1, HasTurn: true}}) {
		t.Fatal("negative turn accepted")
	}
}
