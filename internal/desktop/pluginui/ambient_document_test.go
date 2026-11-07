package pluginui

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"whaleisle/internal/processutil"
)

// An ambient shell entry is consumed as absolute rich fragments: each occupied
// box (flow children, fixed descendants, owned portals) keeps its authored
// coordinates plus the measure-time viewport for edge anchoring.
func TestAmbientDocumentSeatsOccupiedBoxesAtAuthoredCoordinates(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	source, err := json.Marshal(string(nativeDocumentSource))
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const context={assert,getComputedStyle:node=>node.style||{position:'static'},window:{innerWidth:800,innerHeight:600},
 whaleRichSurface:{rect:n=>n.r,hidden:n=>!!n.hide,portals:o=>o.ps||[],applied:()=>false},
 document:{querySelectorAll:()=>[]}};
vm.createContext(context);vm.runInContext(` + string(source) + `,context);
vm.runInContext('(' + function(){
 const make=(r,position)=>({isConnected:true,r,style:{position},tagName:'DIV',children:[],querySelectorAll:()=>[],getAttribute:()=>null,matches:()=>false});
 const child=make({left:10,top:560,width:300,height:40},'static'),blank=make({left:0,top:0,width:0,height:0},'static'),ball=make({left:760,top:500,width:28,height:28},'fixed');
 const portal=make({left:200,top:100,width:400,height:300},'fixed');portal.getAttribute=k=>k==='aria-modal'?'true':null;
 const shield=make({left:0,top:0,width:800,height:600},'fixed');
 const root={isConnected:true,children:[child,blank],querySelectorAll:sel=>sel==='*'?[ball,shield]:[],getAttribute:()=>null};
 const actions=new Map(),ids=new Map();let seq=0;const identify=n=>{if(!ids.has(n))ids.set(n,'h'+(++seq));return ids.get(n)};
 const nodes=whaleAmbientDocument(root,identify,actions,{root,ps:[portal]});
 assert.equal(nodes.length,4);
 for(const n of nodes){assert.equal(n.kind,'rich');assert.equal(n.richType,'fragment');assert.equal(n.popup,true);assert.equal(n.ambient,true);assert.equal(n.vw,800);assert.equal(n.vh,600);assert(actions.has(n.handle))}
 const byId=Object.fromEntries([...ids].map(([n,id])=>[id,n])),at=n=>nodes.find(v=>v.handle===ids.get(n));
 assert.equal(at(child).x,10);assert.equal(at(child).y,560);
 assert.equal(at(ball).x,760);assert.equal(at(ball).y,500);
 assert.equal(at(portal).modal,true);assert.equal(at(portal).x,200);
 assert.equal(at(shield).fullScreen,true,'full-viewport fixed layer must claim the surface');
 assert.equal(at(blank),undefined,'zero-size box seated');
 assert.equal(whaleAmbientDocument({isConnected:true,hide:true,children:[],querySelectorAll:()=>[],getAttribute:()=>null},identify,actions,{}).length,0,'hidden entry emitted');
}.toString() + ')()',context);`
	cmd := exec.Command(node, "-")
	cmd.Stdin = strings.NewReader(script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ambient document: %v\n%s", err, out)
	}
}

func TestAmbientFixedSeatRetainsLiveDragAndPointerEventsInTopLayer(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	docSource, err := json.Marshal(string(nativeDocumentSource))
	if err != nil {
		t.Fatal(err)
	}
	richSource, err := json.Marshal(string(richControlsSource))
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const head={appendChild:el=>{context.maskStyle=el.textContent}};
const docEl={dataset:{},hasAttribute:()=>false,setAttribute:(k,v)=>{docEl.dataset[k]=v},removeAttribute:k=>{delete docEl.dataset[k]}};
const context={assert,maskStyle:'',CSS:{escape:s=>s},requestAnimationFrame:cb=>cb(),
 getComputedStyle:node=>node.computed||{position:'fixed',display:'block',visibility:'visible',opacity:'1',borderTopWidth:'0px',borderRightWidth:'0px',borderBottomWidth:'0px',borderLeftWidth:'0px',paddingTop:'0px',paddingRight:'0px',paddingBottom:'0px',paddingLeft:'0px',backgroundColor:'rgba(0, 0, 0, 0)',color:'rgb(0, 0, 0)',font:'13px sans-serif',lineHeight:'18px',textAlign:'start',boxSizing:'border-box'},
 window:{innerWidth:1440,innerHeight:863},
 document:{head,documentElement:docEl,body:{},createElement:()=>({setAttribute(){},textContent:''}),querySelectorAll:()=>[]}};
vm.createContext(context);
vm.runInContext(` + string(richSource) + `,context);
vm.runInContext(` + string(docSource) + `,context);
vm.runInContext('(' + function(){
 const makeFixed=(rect)=>({
  isConnected:true,tagName:'DIV',children:[],
  computed:{position:'fixed',display:'block',visibility:'visible',opacity:'1',borderTopWidth:'0px',borderRightWidth:'0px',borderBottomWidth:'0px',borderLeftWidth:'0px',paddingTop:'0px',paddingRight:'0px',paddingBottom:'0px',paddingLeft:'0px',backgroundColor:'rgba(0, 0, 0, 0)',color:'rgb(0, 0, 0)',font:'13px sans-serif',lineHeight:'18px',textAlign:'start',boxSizing:'border-box'},
  rect,
  attrs:new Map(),
  props:new Map(),
  getBoundingClientRect(){return this.rect},
  getClientRects(){return [this.rect]},
  querySelectorAll(){return []},
  contains(other){return other===this||(this.nested||[]).includes(other)},
  hasAttribute(k){return this.attrs.has(k)},
  getAttribute(k){return this.attrs.has(k)?this.attrs.get(k):null},
  setAttribute(k,v){this.attrs.set(k,String(v))},
  removeAttribute(k){this.attrs.delete(k)},
  matches(sel){return sel===':popover-open'&&this.popoverOpen===true},
  showPopover(){this.popoverOpen=true},
  hidePopover(){this.popoverOpen=false},
  style:{
   setProperty(k,v,p){nodeProps.set(k,{v,p})},
   getPropertyValue(k){return nodeProps.get(k)?.v||''},
   getPropertyPriority(k){return nodeProps.get(k)?.p||''},
   removeProperty(k){nodeProps.delete(k)}
  }
 });
 let nodeProps=new Map();
 const dock=makeFixed({left:1080,top:52,width:352,height:500});
 const nestedFixed=makeFixed({left:1090,top:60,width:28,height:28});
 dock.nested=[nestedFixed];
 const star=makeFixed({left:1400,top:54,width:28,height:28});
 star.setAttribute('aria-hidden','true');
 const root={isConnected:true,children:[],querySelectorAll:sel=>sel==='*'?[dock,nestedFixed,star]:[],getAttribute:()=>null,contains:()=>true};
 const actions=new Map(),ids=new Map();let seq=0;
 const identify=n=>{if(!ids.has(n))ids.set(n,'h'+(++seq));return ids.get(n)};
 const nodes=whaleAmbientDocument(root,identify,actions,{root});
 assert.equal(nodes.length,2,'nested fixed child inside a seated dock must not be double-seated');
 for(const rec of actions.values()){assert.equal(rec.owner.ambient,true,'ambient owner flag must be set')}
 let scheduled=0,drained=0,acked=null;
 whaleRichSurface.bind(()=>({
  revision:1,
  subject:{kind:'navigation'},
  actions,
  valid:()=>true,
  schedule:()=>{scheduled++},
  drain:()=>{drained++}
 }));
 // Simulate dock moving mid-drag (live rect is at left=900 while in-flight seat was left=1080).
 dock.rect={left:900,top:80,width:352,height:500};
 whaleRichSurface.ack({revision:1,token:42,subject:{kind:'navigation'},seats:[
  {handle:ids.get(dock),x:1080,y:52,w:352,h:500,cl:0,ct:0,cr:0,cb:0},
  {handle:ids.get(star),x:1400,y:54,w:28,h:28,cl:0,ct:0,cr:0,cb:0}
 ]},msg=>{acked=msg});
 assert(!maskStyle.includes('[aria-hidden="true"]'),'mask CSS must not hide aria-hidden decorative seats like .dshpad-star-anchor');
 assert(maskStyle.includes(':where([data-whale-rich-seat])'),'mask CSS must use zero-specificity :where() so pointer-events:none on .dshpad-star-anchor wins');
 assert.equal(acked?.ok,true,'mid-drag ambient fixed seat must acknowledge ok instead of failing top-layer geometry');
 assert.equal(nodeProps.has('left'),false,'ambient fixed seat must not have inline left!important locked');
 assert.equal(nodeProps.has('top'),false,'ambient fixed seat must not have inline top!important locked');
 assert(scheduled>=1,'drifted ambient fixed seat must schedule a fresh geometry publish');
 assert(drained>=1,'rich surface must drain observer records after layout');
}.toString() + ')()',context);`
	cmd := exec.Command(node, "-")
	cmd.Stdin = strings.NewReader(script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ambient fixed seat: %v\n%s", err, out)
	}
}
