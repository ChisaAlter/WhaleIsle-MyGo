//go:build windows

package pet

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

// This fixture uses only TestMain's owned profile and page-local CDP input.
func TestRealPetNeuralPickup(t *testing.T) {
	if os.Getenv("WHALEISLE_PET_GUI") != "1" || os.Getenv("WHALEISLE_PET_GUI_PICKUP") != "1" {
		t.Skip("owned pickup fixture only")
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
			options.Page.PreloadScript = neuralPickupObserver
			return mygo.NewWindow(options)
		}
		p = New(store)
		p.SetReady(true)
		w = p.win
		openCompanionWindow = prior
	})
	if w == nil {
		t.Fatal("owned pickup window missing")
	}
	t.Cleanup(func() { mygo.RunOnMain(p.Close) })
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
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
		t.Fatalf("pickup wait failed last=%#v err=%v", last, err)
		return nil
	}
	realPetEvidence["dom"] = wait(`(()=>({renderer:document.body.dataset.renderer||"",provider:document.body.dataset.provider||"",sr:document.body.dataset.superResolution||""}))()`, func(v map[string]any) bool { return v["renderer"] == "neural" })
	// Stop just this fixture's native motion sampler; the owned snapshots below
	// retain production JSON transport and local DIP layout without OS input.
	var layout petLayout
	mygo.RunOnMain(func() {
		if p.interaction.timer != nil {
			p.interaction.timer.Stop()
			p.interaction.timer = nil
		}
		layout = p.layout
	})
	publish := func(mode string, dx, dy float64) {
		mygo.RunOnMain(func() {
			p.assetMu.Lock()
			defer p.assetMu.Unlock()
			p.interaction.mode = mode
			p.interaction.pressed = mode == "pickup"
			p.layout = layout
			p.layout.X += int(dx)
			p.layout.Y += int(dy)
			p.interaction.view.Mode = mode
			p.interaction.view.CursorX = float64(layout.X) + float64(layout.Width)/2
			p.interaction.view.CursorY = float64(layout.Y) + float64(layout.Height)*.2
			p.interaction.view.VX = 100
			p.interaction.view.Facing = -1
		})
	}
	// Navigation stays inside this owned Companion; native motion is stopped.
	// Trusted pointer events and capture come from this WebView's CDP input.
	var samples []map[string]any
	capture := func(name string) {
		t.Helper()
		if _, err = w.Page().EvalContext(ctx, `window.__petPickupProbe.arm()`); err != nil {
			t.Fatal(err)
		}
		sample := wait(`window.__petPickupProbe.sample`, func(v map[string]any) bool { return v["ready"] == true })
		if sample["error"] != nil {
			t.Fatal(sample["error"])
		}
		if sample["matrixDelta"].(float64) > .01 || sample["hitDelta"].(float64) > .01 || sample["poseDelta"].(float64) > 0.0000002 {
			t.Errorf("pickup atomic/source mismatch %#v", sample)
		}
		if sample["hasCapture"] != true {
			t.Errorf("original pointer capture lost %#v", sample)
		}
		text, _ := sample["png"].(string)
		data, e := base64.StdEncoding.DecodeString(strings.TrimPrefix(text, "data:image/png;base64,"))
		if e != nil {
			t.Fatal(e)
		}
		savePetGUIImage(t, "neural-pickup-"+name+"-canvas", data)
		delete(sample, "png")
		if image, e := w.CapturePage(); e == nil {
			savePetGUIImage(t, "neural-pickup-"+name+"-window", image)
		} else {
			t.Error(e)
		}
		sample["name"] = name
		samples = append(samples, sample)
		realPetEvidence["neuralPickupSamples"] = samples
	}
	point := wait(`(()=>{const r=document.getElementById('sprite').getBoundingClientRect();return {x:r.left+r.width*.5,y:r.top+r.height*.2}})()`, func(map[string]any) bool { return true })
	mouse := func(kind string, x, y float64, buttons int) {
		t.Helper()
		caller, ok := any(w.Page()).(interface {
			PetOwnedMouse(context.Context, string, float64, float64, int) error
		})
		if !ok {
			t.Fatal("owned input overlay missing")
		}
		if err = caller.PetOwnedMouse(ctx, kind, x, y, buttons); err != nil {
			t.Fatal(err)
		}
	}
	mouse("mouseMoved", point["x"].(float64), point["y"].(float64), 0)
	mouse("mousePressed", point["x"].(float64), point["y"].(float64), 1)
	publish("pickup", 0, 0)
	capture("canvas-held-early")
	wait(`(()=>({elapsed:window.__petPickupProbe.elapsed||0}))()`, func(v map[string]any) bool { return v["elapsed"].(float64) > .9 })
	capture("canvas-held-late")
	publish("throw", 30, 15)
	capture("canvas-throw-native-move")
	mouse("mouseReleased", point["x"].(float64), point["y"].(float64), 0)
	publish("", 30, 15)
	wait(`(()=>({ready:document.getElementById('pet-body-hit').hidden}))()`, func(v map[string]any) bool { return v["ready"] == true })
	publish("pickup", 0, 0)
	wait(`(()=>({ready:!document.getElementById('pet-body-hit').hidden}))()`, func(v map[string]any) bool { return v["ready"] == true })
	proxy := wait(`(()=>{const r=document.getElementById('pet-body-hit').getBoundingClientRect();return {x:r.left+r.width/2,y:r.top+r.height/2}})()`, func(map[string]any) bool { return true })
	mouse("mouseMoved", proxy["x"].(float64), proxy["y"].(float64), 0)
	mouse("mousePressed", proxy["x"].(float64), proxy["y"].(float64), 1)
	capture("proxy-held")
	mouse("mouseReleased", proxy["x"].(float64), proxy["y"].(float64), 0)
	publish("", 0, 0)
	wait(`(()=>({ready:document.getElementById('pet-body-hit').hidden}))()`, func(v map[string]any) bool { return v["ready"] == true })
	t.Logf("owned pickup samples %v", samples)
}

const neuralPickupObserver = `(()=>{
 const probe={ignoreNative:false,sample:null,wanted:false,current:null,pose:null,elapsed:0,pointer:null,arm(){this.sample=null;this.wanted=true;return true;}};
 window.addEventListener('pointerdown',event=>{probe.pointer={id:event.pointerId,node:event.target.id,trusted:event.isTrusted};},true);
 function hook(name,decorate){let value;Object.defineProperty(window,name,{configurable:true,get(){return value;},set(next){value=decorate(next);}});}
 hook('WhalePetInteraction',api=>{const create=api.create;api.create=function(...args){const c=create.apply(this,args),use=c.useNeuralRest;c.useNeuralRest=function(...args){const d=use.apply(this,args);return {frame(value){d.frame(value);const pose=new Float32Array(45);c.pose(pose);probe.pose=Array.from(pose);},get pickup(){return d.pickup;},get input(){return d.input;},close(){return d.close();}};};return c;};return api;});
 hook('WhaleTHA4',api=>{const open=api.open;api.open=async function(...args){const opened=await open.apply(this,args),run=opened.engine.frame;opened.engine.frame=async function(values){probe.inferencePose=Array.from(values);probe.inferenceExpected=probe.pose?.slice();return run.call(this,values);};return opened;};return api;});
 hook('WhalePetSleep',api=>{const create=api.create;return {...api,create(...args){const options=args[0],c=create.apply(this,args);return {...c,get active(){return c.active;},get pickupActive(){return c.pickupActive;},commitNeuralBody(pixels,token,input,paint){const success=c.commitNeuralBody(pixels,token,input,paint);if(!success)return success;probe.elapsed=token.frame?.elapsed||0;if(!probe.wanted||!token.frame?.hanging)return success;probe.wanted=false;queueMicrotask(()=>{try{const canvas=document.getElementById('sprite'),target=document.getElementById('pet-body-hit'),f=token.frame,layout=token.layout,evidence=c.pickupEvidence,box=evidence.box,w=layout.Width,h=layout.Height,a=f.a,cos=Math.cos(f.angle),sin=Math.sin(f.angle),px=f.cursorX,py=f.cursorY+12+f.dy;
 const expected=[cos*f.sx,sin*f.sx,-sin*f.sy,cos*f.sy,px-layout.X-cos*f.sx*w/2+sin*f.sy*h*.085,py-layout.Y-sin*f.sx*w/2-cos*f.sy*h*.085],m=new DOMMatrix(canvas.style.transform),actual=[m.a,m.b,m.c,m.d,m.e,m.f];
 const corners=[[box.x,box.y],[box.right,box.y],[box.x,box.bottom],[box.right,box.bottom]].map(([x,y])=>{const X=((x-60)*w/390-w/2)*f.sx,Y=((y-20)*h/492-h*.085)*f.sy;return {x:px+X*cos-Y*sin,y:py+X*sin+Y*cos};}),hit=WhalePetBody.current.hitRect(target),expectedHit={x:Math.min(...corners.map(p=>p.x)),y:Math.min(...corners.map(p=>p.y)),width:Math.max(...corners.map(p=>p.x))-Math.min(...corners.map(p=>p.x)),height:Math.max(...corners.map(p=>p.y))-Math.min(...corners.map(p=>p.y))};
 const pointer=probe.pointer,node=pointer?document.getElementById(pointer.node):null;probe.sample={ready:true,elapsed:f.elapsed,alpha:f.alpha,phase:f.phase,mode:f.mode,epoch:f.epoch,angle:f.angle,scaleY:f.sy,pose:probe.inferencePose,poseDelta:Math.max(...probe.inferencePose.map((v,i)=>Math.abs(v-probe.inferenceExpected[i]))),matrixDelta:Math.max(...actual.map((v,i)=>Math.abs(v-expected[i]))),hitDelta:Math.max(...['x','y','width','height'].map(k=>Math.abs(hit[k]-expectedHit[k]))),hasCapture:!!node?.hasPointerCapture(pointer.id),captureNode:pointer?.node,trusted:pointer?.trusted,frames:evidence.frames,matrix:actual,hit,box,png:canvas.toDataURL()};}catch(error){probe.sample={ready:true,error:String(error)};}});return success;}};}};});
 window.__petPickupProbe=probe;
})()`
