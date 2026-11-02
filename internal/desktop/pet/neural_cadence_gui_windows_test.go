//go:build windows

package pet

import (
	"context"
	"encoding/base64"
	"github.com/egoist/mygo"
	"os"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/config"
)

// This fixture observes only its own TestMain profile and WebView; it sends no
// OS input. Counts distinguish cached display presents from distinct RGBA.
func TestRealPetNeuralCadence(t *testing.T) {
	if os.Getenv("WHALEISLE_PET_GUI") != "1" || os.Getenv("WHALEISLE_PET_GUI_CADENCE") != "1" {
		t.Skip("owned cadence fixture only")
	}
	dir := os.Getenv("WHALEISLE_PET_RENDER_ASSETS")
	if _, err := loadRenderAssets(dir); err != nil {
		t.Fatal(err)
	}
	previous := petRenderRoot
	petRenderRoot = func() string { return dir }
	t.Cleanup(func() { petRenderRoot = previous })
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	behavior := config.DefaultPetBehavior()
	behavior.Wander = false
	behavior.SelfTalk = false
	if err = store.SaveDesktopPet(config.DesktopPet{Enabled: true, Neural: true, Behavior: behavior}); err != nil {
		t.Fatal(err)
	}
	var p *Companion
	var w *mygo.Window
	mygo.RunOnMain(func() {
		prior := openCompanionWindow
		openCompanionWindow = func(options mygo.WindowOptions) *mygo.Window {
			options.Page.PreloadScript = neuralCadenceObserver
			return mygo.NewWindow(options)
		}
		p = New(store)
		p.SetReady(true)
		w = p.win
		openCompanionWindow = prior
	})
	if w == nil {
		t.Fatal("owned cadence window missing")
	}
	t.Cleanup(func() { mygo.RunOnMain(p.Close) })
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	wait := func(script string, accept func(map[string]any) bool) map[string]any {
		t.Helper()
		var last any
		var err error
		for ctx.Err() == nil {
			last, err = w.Page().EvalContext(ctx, script)
			if value, ok := last.(map[string]any); err == nil && ok && accept(value) {
				return value
			}
			time.Sleep(35 * time.Millisecond)
		}
		t.Fatalf("cadence wait failed last=%#v err=%v", last, err)
		return nil
	}
	realPetEvidence["dom"] = wait(`(()=>({renderer:document.body.dataset.renderer||"",provider:document.body.dataset.provider||"",sr:document.body.dataset.superResolution||""}))()`, func(v map[string]any) bool { return v["renderer"] == "neural" })
	mygo.RunOnMain(func() { p.assetMu.Lock(); p.rest.sleep(); p.assetMu.Unlock() })
	wait(`window.__petCadenceProbe.status()`, func(v map[string]any) bool { return v["sleeping"] == true && v["particles"].(float64) > 0 })
	if _, err = w.Page().EvalContext(ctx, `window.__petCadenceProbe.arm()`); err != nil {
		t.Fatal(err)
	}
	sample := wait(`window.__petCadenceProbe.result()`, func(v map[string]any) bool { return v["ready"] == true })
	realPetEvidence["neuralCadence"] = sample
	for _, key := range []string{"inputDelta", "modelPoseDelta", "matrixDelta", "hitDelta"} {
		limit := 0.0000002
		if key == "matrixDelta" || key == "hitDelta" {
			limit = .01
		}
		if sample[key].(float64) > limit {
			t.Errorf("cadence %s differs: %v", key, sample[key])
		}
	}
	if sample["cachedPresents"].(float64) <= sample["inferenceCompleted"].(float64) {
		t.Errorf("cache display did not run between inferences: %#v", sample)
	}
	if sample["prepares"] != sample["accepted"] {
		t.Errorf("SR prepare must run once per accepted output: %#v", sample)
	}
	if sample["distinctRGBA"].(float64) > sample["inferenceCompleted"].(float64) {
		t.Errorf("distinct content count exceeds inference outputs: %#v", sample)
	}
	if sample["bodySamples"].(float64) < 20 || sample["particleSamples"].(float64) < 20 {
		t.Errorf("insufficient actual body/particle samples: %#v", sample)
	}
	for _, key := range []string{"canvasPng", "particlePng"} {
		text, _ := sample[key].(string)
		if text != "" {
			data, e := base64.StdEncoding.DecodeString(strings.TrimPrefix(text, "data:image/png;base64,"))
			if e != nil {
				t.Fatal(e)
			}
			savePetGUIImage(t, "neural-cadence-"+key, data)
		}
		delete(sample, key)
	}
	if image, e := w.CapturePage(); e == nil {
		savePetGUIImage(t, "neural-cadence-sleep-window", image)
	} else {
		t.Error(e)
	}
	mygo.RunOnMain(func() { p.assetMu.Lock(); _ = p.rest.wake(nil); p.assetMu.Unlock() })
	tail := wait(`window.__petCadenceProbe.status()`, func(v map[string]any) bool { return v["sleeping"] == false })
	realPetEvidence["wakeParticles"] = tail
	wait(`window.__petCadenceProbe.status()`, func(v map[string]any) bool { return v["sleeping"] == false && v["particles"] == float64(0) })
	t.Logf("owned cadence distinguishes display/inference/content counts: %#v", sample)
}

const neuralCadenceObserver = `(()=>{
 const probe={armed:false,start:0,now:0,inferenceStarted:0,inferenceCompleted:0,accepted:0,prepares:0,cachedPresents:0,bodySamples:0,particleSamples:0,inputDelta:0,modelPoseDelta:0,matrixDelta:0,hitDelta:0,hashes:new Set(),modelHash:new Map(),sleeping:false,driver:null,controller:null};
 const nativeRAF=window.requestAnimationFrame.bind(window);window.requestAnimationFrame=callback=>nativeRAF(now=>{probe.now=now;callback(now);});
 const hash=values=>{let h=2166136261;for(let i=0;i<values.length;i++){h^=values[i];h=Math.imul(h,16777619);}return h>>>0;};
 function hook(name,decorate){let value;Object.defineProperty(window,name,{configurable:true,get(){return value;},set(next){value=decorate(next);}});}
 hook('WhaleTHA4',api=>({...api,async open(...args){const opened=await api.open(...args),run=opened.engine.frame;opened.engine.frame=async function(values){const before=Array.from(values);if(probe.armed)probe.inferenceStarted++;const pixels=await run.call(this,values);probe.inputDelta=Math.max(probe.inputDelta,...before.map((v,i)=>Math.abs(v-values[i])));if(pixels){const key=hash(pixels);probe.modelHash.set(key,before);probe.latestHash=key;if(probe.armed){probe.inferenceCompleted++;probe.hashes.add(key);}}return pixels;};return opened;}}));
 hook('WhalePetSuperResolution',api=>({...api,create(...args){const c=api.create(...args);return {...c,get ready(){return c.ready;},prepare(image){if(probe.armed)probe.prepares++;return c.prepare(image);}};}}));
 hook('WhalePetCareAnimation',api=>({...api,create(...args){const c=api.create(...args),use=c.useNeuralParticles;c.useNeuralParticles=function(...args){const driver=use.apply(c,args);return {...driver,frame(now,state){driver.frame(now,state);probe.sleeping=!!state?.sleeping;if(probe.armed){const nodes=[...document.querySelectorAll('span')].filter(n=>n.textContent==='Z');probe.particleSamples+=nodes.length;const node=nodes.find(n=>Number(n.style.opacity)>.35);if(node&&!probe.particlePng){const out=document.createElement('canvas');out.width=100;out.height=100;const ctx=out.getContext('2d');ctx.fillStyle=node.style.color;ctx.font=getComputedStyle(node).font;ctx.globalAlpha=Number(node.style.opacity);ctx.fillText('Z',25,65);probe.particlePng=out.toDataURL();}}}};};return c;}}));
 hook('WhalePetSleep',api=>({...api,create(...args){const c=api.create(...args);probe.controller=c;const check=token=>{if(!probe.armed||!c.active)return;const canvas=document.getElementById('sprite'),m=new DOMMatrix(canvas.style.transform),box=c.evidence.box,layout=JSON.parse(document.body.dataset.layout||'{}'),w=layout.Width,h=layout.Height,co=Math.cos(token.angle),si=Math.sin(token.angle),sy=token.sy,expected=[co,si,-si*sy,co*sy,w/2-co*w/2+si*h*sy,h+token.dy-si*w/2-co*h*sy],actual=[m.a,m.b,m.c,m.d,m.e,m.f];probe.matrixDelta=Math.max(probe.matrixDelta,...expected.map((v,i)=>Math.abs(v-actual[i])));const corners=[[box.x,box.y],[box.right,box.y],[box.x,box.bottom],[box.right,box.bottom]].map(([x,y])=>{const px=((x-60)*w/390-w/2)*token.facing,py=((y-20)*h/492-h)*sy;return{x:layout.X+w/2+px*co-py*si,y:layout.Y+h+token.dy+px*si+py*co};}),hit=WhalePetBody.current.hitRect(WhalePetBody.current.target),expectedHit={x:Math.min(...corners.map(p=>p.x)),y:Math.min(...corners.map(p=>p.y)),width:Math.max(...corners.map(p=>p.x))-Math.min(...corners.map(p=>p.x)),height:Math.max(...corners.map(p=>p.y))-Math.min(...corners.map(p=>p.y))};probe.hitDelta=Math.max(probe.hitDelta,...['x','y','width','height'].map(k=>Math.abs(hit[k]-expectedHit[k])));probe.bodySamples++;};return {...c,get active(){return c.active;},get evidence(){return c.evidence;},get pickupActive(){return c.pickupActive;},commitRest(...args){const accepted=c.commitRest(...args);if(accepted&&probe.armed)probe.accepted++;return accepted;},presentCommitted(content,token,input,paint){const accepted=c.presentCommitted(content,token,input,paint);if(accepted&&probe.armed){probe.cachedPresents++;const expected=probe.modelHash.get(probe.latestHash);if(expected)probe.modelPoseDelta=Math.max(probe.modelPoseDelta,...content.modelPose.map((v,i)=>Math.abs(v-expected[i])));check(token);}return accepted;}};}}));
 probe.arm=()=>{probe.armed=true;probe.start=performance.now();for(const key of ['inferenceStarted','inferenceCompleted','accepted','prepares','cachedPresents','bodySamples','particleSamples','inputDelta','modelPoseDelta','matrixDelta','hitDelta'])probe[key]=0;probe.hashes.clear();return true;};
 probe.status=()=>({sleeping:probe.sleeping,particles:[...document.querySelectorAll('span')].filter(n=>n.textContent==='Z').length});
 probe.result=()=>{if(!probe.armed||performance.now()-probe.start<4000)return {ready:false};probe.armed=false;return {ready:true,elapsed:performance.now()-probe.start,inferenceStarted:probe.inferenceStarted,inferenceCompleted:probe.inferenceCompleted,accepted:probe.accepted,prepares:probe.prepares,cachedPresents:probe.cachedPresents,distinctRGBA:probe.hashes.size,bodySamples:probe.bodySamples,particleSamples:probe.particleSamples,inputDelta:probe.inputDelta,modelPoseDelta:probe.modelPoseDelta,matrixDelta:probe.matrixDelta,hitDelta:probe.hitDelta,canvasPng:document.getElementById('sprite').toDataURL(),particlePng:probe.particlePng||''};};
 window.__petCadenceProbe=probe;
})()`
