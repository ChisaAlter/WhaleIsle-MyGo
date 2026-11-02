//go:build windows

package pet

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/egoist/mygo"
	"os"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/config"
)

func TestRealPetNeuralBoundary(t *testing.T) {
	if os.Getenv("WHALEISLE_PET_GUI") != "1" || os.Getenv("WHALEISLE_PET_GUI_BOUNDARY") != "1" {
		t.Skip("owned absolute feet/wake fixture only")
	}
	raw, err := os.ReadFile("reference/electron/src/renderer/pet-live2d.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	from, to := strings.Index(text, "function drawLive()"), strings.Index(text, "function drawRigPart(")
	if from < 0 || to <= from {
		t.Fatal("reference drawLive missing")
	}
	drawJSON, _ := json.Marshal(text[from:to])
	dir := os.Getenv("WHALEISLE_PET_RENDER_ASSETS")
	if _, err = loadRenderAssets(dir); err != nil {
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
			options.Page.PreloadScript = "window.__petBoundaryDraw=" + string(drawJSON) + ";\n" + neuralBoundaryObserver
			return mygo.NewWindow(options)
		}
		p = New(store)
		p.SetReady(true)
		w = p.win
		openCompanionWindow = prior
	})
	if w == nil {
		t.Fatal("owned boundary window missing")
	}
	t.Cleanup(func() { mygo.RunOnMain(p.Close) })
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
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
			time.Sleep(30 * time.Millisecond)
		}
		t.Fatalf("boundary wait failed last=%#v err=%v", last, err)
		return nil
	}
	realPetEvidence["dom"] = wait(`(()=>({renderer:document.body.dataset.renderer||"",provider:document.body.dataset.provider||"",sr:document.body.dataset.superResolution||""}))()`, func(v map[string]any) bool { return v["renderer"] == "neural" })
	mygo.RunOnMain(func() {
		if p.interaction.timer != nil {
			p.interaction.timer.Stop()
			p.interaction.timer = nil
		}
	})
	before := wait(`window.__petBoundaryProbe.normal()`, func(v map[string]any) bool { return v["ready"] == true })
	realPetEvidence["normalBefore"] = before
	for _, key := range []string{"feetDelta", "normalHitDelta", "baselineDelta"} {
		if before[key].(float64) > .01 {
			t.Errorf("normal absolute %s: %v", key, before[key])
		}
	}
	mygo.RunOnMain(func() { p.assetMu.Lock(); p.rest.sleep(); p.assetMu.Unlock() })
	wait(`window.__petBoundaryProbe.status()`, func(v map[string]any) bool { return v["phase"] == "sleep" && v["sleepSamples"].(float64) >= 25 })
	if image, e := w.CapturePage(); e == nil {
		savePetGUIImage(t, "neural-boundary-sleep-window", image)
	} else {
		t.Error(e)
	}
	mygo.RunOnMain(func() { p.assetMu.Lock(); _ = p.rest.wake(nil); p.assetMu.Unlock() })
	wait(`window.__petBoundaryProbe.status()`, func(v map[string]any) bool { return v["wakeFaded"] == true })
	after := wait(`window.__petBoundaryProbe.normal()`, func(v map[string]any) bool { return v["ready"] == true })
	realPetEvidence["normalAfter"] = after
	for _, key := range []string{"feetDelta", "normalHitDelta", "baselineDelta"} {
		if after[key].(float64) > .01 {
			t.Errorf("normal return absolute %s: %v", key, after[key])
		}
	}
	result := wait(`window.__petBoundaryProbe.result()`, func(v map[string]any) bool { return v["ready"] == true })
	realPetEvidence["neuralBoundary"] = result
	for _, key := range []string{"matrixDelta", "cornerDelta", "hitDelta", "feetDelta", "headDelta"} {
		if result[key].(float64) > .01 {
			t.Errorf("absolute source %s mismatch: %v", key, result[key])
		}
	}
	if result["alphaDelta"].(float64) > 1e-12 || result["inputDelta"].(float64) > 2e-7 {
		t.Errorf("wake expiry/input mismatch: %#v", result)
	}
	if result["expirySamples"].(float64) < 2 {
		t.Errorf("wake expiry boundary was not observed: %#v", result)
	}
	for _, key := range []string{"sleepPng", "wakePng"} {
		encoded, _ := result[key].(string)
		if encoded != "" {
			data, e := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, "data:image/png;base64,"))
			if e != nil {
				t.Fatal(e)
			}
			savePetGUIImage(t, "neural-boundary-"+key, data)
		}
		delete(result, key)
	}
	if image, e := w.CapturePage(); e == nil {
		savePetGUIImage(t, "neural-boundary-awake-window", image)
	} else {
		t.Error(e)
	}
	t.Logf("absolute reference drawLive and wake expiry: %#v", result)
}

const neuralBoundaryObserver = `(()=>{
 const probe={sleepSamples:0,wakeSamples:0,expirySamples:0,matrixDelta:0,cornerDelta:0,hitDelta:0,feetDelta:0,headDelta:0,alphaDelta:0,inputDelta:0,wakeFaded:false,phase:null,previous:null};
 const reference=new Function('ctx2d','frame','layout','const rigT=frame.time,settings={opacity:1},srReady=false,outCanvas={},stillCtl={alpha:1},smooth=x=>x*x*(3-2*x),clamp01=x=>Math.max(0,Math.min(1,x)),fx={rot:0,dx:0,dy:0,sx:1,sy:1},liveFx={rot:frame.angle,dx:0,dy:frame.dy,sx:1,sy:frame.sy,pivot:null,stars:false},dragging=false,thrown=false,physPoint=null,petW=()=>layout.Width,petH=()=>layout.Height,drawPos={x:layout.X,y:layout.Y},facing=frame.facing,GRAB_FRAC=.085;'+window.__petBoundaryDraw+'\ndrawLive();');
 function hook(name,decorate){let value;Object.defineProperty(window,name,{configurable:true,get(){return value;},set(next){value=decorate(next);}});}
 hook('WhaleTHA4',api=>({...api,async open(...args){const opened=await api.open(...args),run=opened.engine.frame;opened.engine.frame=async function(values){const before=Array.from(values),pixels=await run.call(this,values);probe.inputDelta=Math.max(probe.inputDelta,...before.map((v,i)=>Math.abs(v-values[i])));return pixels;};return opened;}}));
 hook('WhalePetSleep',api=>({...api,create(...args){const c=api.create(...args);probe.controller=c;function check(token){if(!c.active||!token||!(token.sleeping||token.waking))return;const canvas=document.getElementById('sprite'),layout=JSON.parse(document.body.dataset.layout),box=c.evidence.box,ctx={matrix:[1,0,0,1,0,0],save(){},restore(){},translate(x,y){this.mul([1,0,0,1,x,y]);},rotate(a){this.mul([Math.cos(a),Math.sin(a),-Math.sin(a),Math.cos(a),0,0]);},scale(x,y){this.mul([x,0,0,y,0,0]);},mul([a,b,c,d,e,f]){const[x,y,z,w,u,v]=this.matrix;this.matrix=[x*a+z*b,y*a+w*b,x*c+z*d,y*c+w*d,x*e+z*f+u,y*e+w*f+v];},drawImage(_src,x,y){this.mul([1,0,0,1,x,y]);}};reference(ctx,{angle:token.angle,dy:token.dy,sy:token.sy,facing:token.facing,time:token.time/1000},layout);const expected=ctx.matrix,m=new DOMMatrix(canvas.style.transform),actual=[m.a,m.b,m.c,m.d,m.e+layout.X,m.f+layout.Y];if(token.facing<0){actual[4]+=actual[0]*layout.Width;actual[5]+=actual[1]*layout.Width;actual[0]*=-1;actual[1]*=-1;}probe.matrixDelta=Math.max(probe.matrixDelta,...actual.map((v,i)=>Math.abs(v-expected[i])));const point=(matrix,x,y)=>({x:matrix[0]*x+matrix[2]*y+matrix[4],y:matrix[1]*x+matrix[3]*y+matrix[5]}),corners=[[box.x,box.y],[box.right,box.y],[box.x,box.bottom],[box.right,box.bottom]].map(([x,y])=>[(x-60)*layout.Width/390,(y-20)*layout.Height/492]),refCorners=corners.map(p=>point(expected,...p)),actualCorners=corners.map(p=>point(actual,...p));probe.cornerDelta=Math.max(probe.cornerDelta,...refCorners.flatMap((p,i)=>[Math.abs(p.x-actualCorners[i].x),Math.abs(p.y-actualCorners[i].y)]));const hit=WhalePetBody.current.hitRect(WhalePetBody.current.target),rect={x:Math.min(...refCorners.map(p=>p.x)),y:Math.min(...refCorners.map(p=>p.y)),width:Math.max(...refCorners.map(p=>p.x))-Math.min(...refCorners.map(p=>p.x)),height:Math.max(...refCorners.map(p=>p.y))-Math.min(...refCorners.map(p=>p.y))};probe.hitDelta=Math.max(probe.hitDelta,...['x','y','width','height'].map(k=>Math.abs(hit[k]-rect[k])));const feet=point(actual,layout.Width/2,layout.Height);probe.feetDelta=Math.max(probe.feetDelta,Math.abs(feet.x-layout.X-layout.Width/2),Math.abs(feet.y-layout.Y-layout.Height+2-token.dy));const head=point(expected,((box.x+box.right)/2-60)*layout.Width/390,(box.y-20)*layout.Height/492);probe.headDelta=Math.max(probe.headDelta,Math.abs(head.x-c.evidence.head.x),Math.abs(head.y-c.evidence.head.y));if(token.sleeping){probe.sleepSamples++;if(!probe.sleepPng&&token.rest?.phase==='sleep')probe.sleepPng=canvas.toDataURL();}else{probe.wakeSamples++;if(!probe.wakePng&&token.sy>1.02)probe.wakePng=canvas.toDataURL();}}
 return {...c,get active(){return c.active;},get evidence(){return c.evidence;},get pickupActive(){return c.pickupActive;},tick(now){c.tick(now);const token=c.snapshot({}),r=token.rest;let target=r.sleeping||token.waking?1:0;if(probe.previous?.phase==='wake'&&!r.sleeping){target=probe.previous.target;let alpha=probe.previous.alpha+(target-probe.previous.alpha)*.35;if(!target&&alpha<.005)alpha=0;probe.alphaDelta=Math.max(probe.alphaDelta,Math.abs(alpha-r.alpha));if(now>r.started*1000+1100)target=0;if(Math.abs(now-r.started*1000-1100)<100)probe.expirySamples++;if(r.phase===null&&r.alpha===0)probe.wakeFaded=true;}probe.previous={phase:r.phase,alpha:r.alpha,target:r.phase==='wake'?target:r.sleeping?1:0};probe.phase=r.phase;},presentCommitted(content,token,input,paint){const ok=c.presentCommitted(content,token,input,paint);if(ok)check(token);return ok;},commitRest(...args){const ok=c.commitRest(...args);if(ok)check(args[1]);return ok;}};}}));
 probe.status=()=>({sleepSamples:probe.sleepSamples,wakeSamples:probe.wakeSamples,wakeFaded:probe.wakeFaded,phase:probe.phase});probe.normal=()=>{const canvas=document.getElementById('sprite'),body=WhalePetBody.current,layout=JSON.parse(document.body.dataset.layout),r=canvas.getBoundingClientRect(),hit=body.hitRect(canvas),m=new DOMMatrix(getComputedStyle(canvas).transform),translate=getComputedStyle(canvas).translate.split(' ').map(parseFloat),origin=getComputedStyle(canvas).transformOrigin.split(' ').map(parseFloat),feetX=layout.X+(translate[0]||0)+origin[0]+m.a*(layout.Width/2-origin[0])+m.c*(layout.Height-origin[1])+m.e,feetY=layout.Y+(translate[1]||0)+origin[1]+m.b*(layout.Width/2-origin[0])+m.d*(layout.Height-origin[1])+m.f;const value={ready:!probe.controller.active&&canvas.style.translate==='0px -2px'&&Math.max(Math.abs(m.a-1),Math.abs(m.b),Math.abs(m.c),Math.abs(m.d-1),Math.abs(m.e),Math.abs(m.f))<.000001,feetDelta:Math.max(Math.abs(feetX-layout.X-layout.Width/2),Math.abs(feetY-layout.Y-layout.Height+2)),normalHitDelta:Math.max(...['x','y','width','height'].map(k=>Math.abs(hit[k]-r[k]))),baselineDelta:Math.max(Math.abs(r.top-layout.Y+2),Math.abs(r.left-layout.X),Math.abs(r.width-layout.Width),Math.abs(r.height-layout.Height)),box:{x:r.x,y:r.y,width:r.width,height:r.height},hit,translate:canvas.style.translate,dpr:devicePixelRatio};return value;};probe.result=()=>({ready:probe.wakeFaded,sleepSamples:probe.sleepSamples,wakeSamples:probe.wakeSamples,expirySamples:probe.expirySamples,matrixDelta:probe.matrixDelta,cornerDelta:probe.cornerDelta,hitDelta:probe.hitDelta,feetDelta:probe.feetDelta,headDelta:probe.headDelta,alphaDelta:probe.alphaDelta,inputDelta:probe.inputDelta,sleepPng:probe.sleepPng||'',wakePng:probe.wakePng||''});window.__petBoundaryProbe=probe;
})()`
