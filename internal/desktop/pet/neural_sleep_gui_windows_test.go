//go:build windows

package pet

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

// Only an explicitly enabled owned test process/profile is used. No input is
// sent to the user's desktop; the fixture changes only this Companion's rest.
func TestRealPetNeuralSleep(t *testing.T) {
	if os.Getenv("WHALEISLE_PET_GUI") != "1" || os.Getenv("WHALEISLE_PET_GUI_SLEEP") != "1" {
		t.Skip("owned sleep fixture only")
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
		previousOpen := openCompanionWindow
		openCompanionWindow = func(options mygo.WindowOptions) *mygo.Window {
			options.Page.PreloadScript = neuralSleepObserver
			if os.Getenv("WHALEISLE_PET_GUI_REST_POSE") == "1" {
				options.Page.PreloadScript += ";\n" + neuralRestPoseObserver
			}
			return mygo.NewWindow(options)
		}
		p = New(store)
		p.SetReady(true)
		w = p.win
		openCompanionWindow = previousOpen
	})
	if w == nil {
		t.Fatal("owned window missing")
	}
	t.Cleanup(func() { mygo.RunOnMain(p.Close) })
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
		if file := os.Getenv("WHALEISLE_PET_GUI_PROGRESS"); file != "" {
			_ = os.WriteFile(file, []byte(fmt.Sprintf("timeout last=%#v err=%v", last, err)), 0600)
		}
		t.Fatalf("sleep wait timeout last=%#v err=%v", last, err)
		return nil
	}
	dom := wait(`(()=>({renderer:document.body.dataset.renderer||"",provider:document.body.dataset.provider||"",sr:document.body.dataset.superResolution||""}))()`, func(v map[string]any) bool { return v["renderer"] == "neural" })
	realPetEvidence["dom"] = dom
	if file := os.Getenv("WHALEISLE_PET_GUI_PROGRESS"); file != "" {
		_ = os.WriteFile(file, []byte("neural ready\n"), 0600)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		_, _ = w.Page().EvalContext(cleanup, `window.__petSleepProbe?.restore()`)
	})
	if os.Getenv("WHALEISLE_PET_GUI_REST_POSE") == "1" {
		exerciseNeuralRestPose(t, ctx, p, w, wait)
	}
	var samples []map[string]any
	for _, step := range []struct {
		name  string
		sleep bool
		kind  string
	}{{"sleep-right", true, "sleep"}, {"wake-middle", false, "wake"}, {"awake", false, "awake"}, {"sleep-left", true, "sleep"}} {
		if _, err = w.Page().EvalContext(ctx, fmt.Sprintf(`window.__petSleepProbe.arm(%q)`, step.kind)); err != nil {
			t.Fatal(err)
		}
		if step.name == "sleep-left" {
			wait(`(()=>({frames:window.__petSleepProbe.frames||0}))()`, func(v map[string]any) bool { return v["frames"].(float64) >= 240 })
		}
		mygo.RunOnMain(func() {
			if step.name == "sleep-left" {
				p.motion.facing = -1
				p.tickInteraction(time.Now())
			}
			p.assetMu.Lock()
			if step.sleep {
				p.rest.sleep()
			} else {
				_ = p.rest.wake(nil)
			}
			p.assetMu.Unlock()
		})
		sample := wait(`window.__petSleepProbe.sample`, func(v map[string]any) bool { return v["ready"] == true })
		if sample["error"] != nil {
			t.Fatalf("sleep probe: %#v", sample)
		}
		if sample["matrixDelta"].(float64) > 0.01 || sample["hitDelta"].(float64) > 0.01 || sample["canvasBoundsDelta"].(float64) > 0.01 {
			t.Errorf("sleep geometry differs: matrix=%v hit=%v canvasBounds=%v frames=%v firstBox=%v latestBox=%v", sample["matrixDelta"], sample["hitDelta"], sample["canvasBoundsDelta"], sample["frames"], sample["firstBox"], sample["latestBox"])
		}
		if step.kind == "sleep" && (sample["markerDelta"] == nil || sample["markerDelta"].(float64) > 0.01) {
			t.Errorf("sleep marker is not anchored to the committed head: %v delta=%v", sample["marker"], sample["markerDelta"])
		}
		if step.kind != "awake" && (sample["individualRotate"] != "none" || sample["individualScale"] != "none" || sample["individualTranslate"] != "none" || sample["activeAnimations"] != float64(0)) {
			t.Errorf("sleep CSS composed twice: rotate=%v scale=%v translate=%v animations=%v", sample["individualRotate"], sample["individualScale"], sample["individualTranslate"], sample["activeAnimations"])
		}
		if step.kind != "awake" && (sample["target"] != "pet-body-hit" || sample["canvasPointerEvents"] != "none") {
			t.Errorf("sleep target ownership: %#v", sample)
		}
		if sample["stageWidth"] != float64(240) || sample["stageHeight"] != float64(260) {
			t.Errorf("stage pixels changed: %#v", sample)
		}
		for _, key := range []string{"actualPng", "referencePng", "differencePng"} {
			text, _ := sample[key].(string)
			data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(text, "data:image/png;base64,"))
			if err != nil {
				t.Fatal(err)
			}
			savePetGUIImage(t, "neural-sleep-"+step.name+"-"+key, data)
			delete(sample, key)
		}
		sample["name"] = step.name
		samples = append(samples, sample)
		realPetEvidence["neuralSleepSamples"] = samples
		if image, err := w.CapturePage(); err == nil {
			savePetGUIImage(t, "neural-sleep-"+step.name+"-window", image)
		} else {
			t.Error(err)
		}
	}
	t.Logf("owned neural sleep source-matrix and alpha-AABB comparisons: %v", samples)
}

const neuralSleepObserver = `(()=>{
 let canvas=null,target=null,body=null;
 const proto=CanvasRenderingContext2D.prototype,draw=proto.drawImage,put=proto.putImageData;
 let raw=null,lastBox=null,firstBox=null,latestBox=null,frames=0,pending=false;
 const probe={get frames(){return frames;},wanted:null,sample:null,arm(kind){this.wanted=kind;this.sample=null;return true;},restore(){proto.drawImage=draw;proto.putImageData=put;delete window.__petSleepProbe;return true;}};
 function putObserved(image,...args){if(this.canvas.width===512&&this.canvas.height===512)raw=new Uint8ClampedArray(image.data);return put.call(this,image,...args);}
 function observed(...args){const result=draw.apply(this,args);if(this.canvas.id!=='sprite')return result;canvas=this.canvas;target=document.getElementById('pet-body-hit');body=WhalePetBody.current;if(raw){let x=512,y=512,right=-1,bottom=-1;for(let py=0;py<512;py+=2)for(let px=0;px<512;px+=2)if(raw[(py*512+px)*4+3]>16){x=Math.min(x,px);y=Math.min(y,py);right=Math.max(right,px);bottom=Math.max(bottom,py);}latestBox={x,y,right,bottom};frames++;if(!lastBox||frames%240===0)lastBox={...latestBox};firstBox=firstBox||{...latestBox};}if(!probe.wanted||pending)return result;pending=true;queueMicrotask(()=>{pending=false;if(!probe.wanted)return;try{sample();}catch(error){probe.sample={ready:true,error:String(error)};probe.wanted=null;}});return result;}
 function sample(){
  const computed=getComputedStyle(canvas),m=new DOMMatrix(canvas.style.transform),active=!target.hidden;
  const angle=Math.atan2(m.b,m.a),sy=Math.hypot(m.c,m.d),w=canvas.offsetWidth,h=canvas.offsetHeight;
  const left=parseFloat(getComputedStyle(canvas).left),top=parseFloat(getComputedStyle(canvas).top);
  const dy=m.b*w/2+m.d*h+m.f-h;
  if(probe.wanted==='sleep'&&(!active||angle>-.99))return;
  if(probe.wanted==='wake'&&(!active||Math.abs(angle)>.00001||sy<1.025))return;
  if(probe.wanted==='awake'&&active)return;
  const frame=canvas.getContext('2d').getImageData(0,0,canvas.width,canvas.height);
  const source=document.createElement('canvas');source.width=canvas.width;source.height=canvas.height;source.getContext('2d').putImageData(frame,0,0);
  let facing=1; // Read the visible reflection against the original raw alpha asymmetry.
  if(raw){let normal=0,mirror=0;for(let y=0;y<512;y+=8)for(let x=60;x<450;x+=8){const ix=Math.min(canvas.width-1,Math.max(0,Math.floor((x-60)*canvas.width/390))),iy=Math.min(canvas.height-1,Math.max(0,Math.floor((y-20)*canvas.height/492)));normal+=Math.abs(raw[(y*512+x)*4+3]-frame.data[(iy*canvas.width+ix)*4+3]);mirror+=Math.abs(raw[(y*512+x)*4+3]-frame.data[(iy*canvas.width+canvas.width-1-ix)*4+3]);}facing=mirror<normal?-1:1;}
  let box={x:0,y:0,right:512,bottom:512};
  if(raw){let x=512,y=512,right=-1,bottom=-1;for(let py=0;py<512;py+=2)for(let px=0;px<512;px+=2)if(raw[(py*512+px)*4+3]>16){x=Math.min(x,px);y=Math.min(y,py);right=Math.max(right,px);bottom=Math.max(bottom,py);}if(right>=0)box={x,y,right,bottom};}
  // Production measures first and every240 accepted frames. The probe can only
  // compare its first raw box to the retained reference until that checkpoint.
  box=lastBox||box;
  const cos=Math.cos(angle),sin=Math.sin(angle);
  const expected=[cos,sin,-sin*sy,cos*sy,w/2-cos*w/2+sin*h*sy,h+dy-sin*w/2-cos*h*sy];
  const actual=[m.a,m.b,m.c,m.d,m.e,m.f];const matrixDelta=Math.max(...actual.map((v,i)=>Math.abs(v-expected[i])));
  const points=[[box.x,box.y],[box.right,box.y],[box.x,box.bottom],[box.right,box.bottom]].map(([x,y])=>{const px=((x-60)*w/390-w/2)*facing,py=((y-20)*h/492-h)*sy;return{x:left+w/2+px*cos-py*sin,y:top+h+dy+px*sin+py*cos};});
  const reference={x:Math.min(...points.map(p=>p.x)),y:Math.min(...points.map(p=>p.y)),right:Math.max(...points.map(p=>p.x)),bottom:Math.max(...points.map(p=>p.y))};reference.width=reference.right-reference.x;reference.height=reference.bottom-reference.y;
  const hit=body.hitRect(target),hitDelta=active?Math.max(...['x','y','width','height'].map(k=>Math.abs(hit[k]-reference[k]))):0;
  const corners=[[0,0],[w,0],[0,h],[w,h]].map(([x,y])=>({x:left+m.a*x+m.c*y+m.e,y:top+m.b*x+m.d*y+m.f}));
  const bounds=canvas.getBoundingClientRect(),canvasBoundsDelta=Math.max(Math.abs(bounds.left-Math.min(...corners.map(p=>p.x))),Math.abs(bounds.top-Math.min(...corners.map(p=>p.y))),Math.abs(bounds.right-Math.max(...corners.map(p=>p.x))),Math.abs(bounds.bottom-Math.max(...corners.map(p=>p.y))));
  const make=()=>{const c=document.createElement('canvas');c.width=700;c.height=650;return c;},out=make(),ref=make(),diff=make(),cx=out.getContext('2d'),rx=ref.getContext('2d');
  cx.translate(250,150);cx.transform(...actual);cx.drawImage(source,0,0,w,h);
  rx.translate(250+w/2,150+h+dy);rx.rotate(angle);rx.scale(1,sy);rx.drawImage(source,-w/2,-h,w,h);
  const a=cx.getImageData(0,0,700,650),r=rx.getImageData(0,0,700,650),d=diff.getContext('2d').createImageData(700,650);let mismatches=0,maxDelta=0;
  for(let i=0;i<a.data.length;i++){const delta=Math.abs(a.data[i]-r.data[i]);if(delta){mismatches++;maxDelta=Math.max(maxDelta,delta);const p=i-i%4;d.data[p]=255;d.data[p+2]=255;d.data[p+3]=255;}}
  diff.getContext('2d').putImageData(d,0,0);
  const center=active?{x:hit.x+hit.width/2,y:hit.y+hit.height/2}:null;
  const marker=[...document.body.children].find(node=>node.textContent==='Z z z'),headX=(box.x+box.right)/2,headY=box.y,px=((headX-60)*w/390-w/2)*facing,py=((headY-20)*h/492-h)*sy;
  const markerExpected={x:left+w/2+px*cos-py*sin+18,y:top+h+dy+px*sin+py*cos-12};
  const markerDelta=marker?Math.max(Math.abs(parseFloat(marker.style.left)-markerExpected.x),Math.abs(parseFloat(marker.style.top)-markerExpected.y)):null;
  probe.sample={ready:true,phase:probe.wanted,marker:marker?{left:marker.style.left,top:marker.style.top}:null,markerExpected,markerDelta,angle,dy,sy,facing,matrix:actual,computedMatrix:computed.transform,individualRotate:computed.rotate,individualScale:computed.scale,individualTranslate:computed.translate,activeAnimations:canvas.getAnimations().length,matrixDelta,hitDelta,canvasBoundsDelta,box,firstBox,latestBox,frames,hit,reference,stageWidth:w,stageHeight:h,backingWidth:canvas.width,backingHeight:canvas.height,canvasPointerEvents:getComputedStyle(canvas).pointerEvents,target:center?document.elementFromPoint(center.x,center.y)?.id:'',normalizedAnchorOffsetCSS:2,pixelMismatchChannels:mismatches,pixelMaxDelta:maxDelta,actualPng:out.toDataURL(),referencePng:ref.toDataURL(),differencePng:diff.toDataURL()};probe.wanted=null;
 }
 proto.drawImage=observed;proto.putImageData=putObserved;window.__petSleepProbe=probe;return true;
})()`

func exerciseNeuralRestPose(t *testing.T, ctx context.Context, p *Companion, w *mygo.Window, wait func(string, func(map[string]any) bool) map[string]any) {
	t.Helper()
	rest := func(value bool) {
		mygo.RunOnMain(func() {
			p.assetMu.Lock()
			defer p.assetMu.Unlock()
			if value {
				p.rest.sleep()
			} else {
				_ = p.rest.wake(nil)
			}
		})
	}
	var samples []map[string]any
	capture := func(name, phase string, minElapsed, maxElapsed float64) map[string]any {
		t.Helper()
		if file := os.Getenv("WHALEISLE_PET_GUI_PROGRESS"); file != "" {
			_ = os.WriteFile(file, []byte("waiting "+name+"\n"), 0600)
		}
		if _, err := w.Page().EvalContext(ctx, fmt.Sprintf(`window.__petRestProbe.arm(%q,%f,%f)`, phase, minElapsed, maxElapsed)); err != nil {
			t.Fatal(err)
		}
		sample := wait(`window.__petRestProbe.sample`, func(v map[string]any) bool { return v["ready"] == true })
		if sample["error"] != nil {
			t.Fatalf("rest pose observer: %v", sample["error"])
		}
		if sample["inputDelta"].(float64) > 0.0000002 || sample["afterDelta"].(float64) > 0 {
			t.Errorf("rest pose input crossed a frame barrier: %#v", sample)
		}
		if sample["phase"] != phase {
			t.Errorf("rest phase=%v want=%s", sample["phase"], phase)
		}
		values := sample["pose"].([]any)
		if phase == "sleep" {
			for ch := 0; ch < 39; ch++ {
				if ch != 12 && ch != 13 && (values[ch].(float64) > 0.000001 || values[ch].(float64) < -0.000001) {
					t.Errorf("sleep facial residue ch%d=%v", ch, values[ch])
				}
			}
			if values[12] != float64(1) || values[13] != float64(1) {
				t.Errorf("sleep eye closure=%v/%v", values[12], values[13])
			}
		}
		if phase == "wake" && sample["alpha"].(float64) > .999 && values[16].(float64) < .899 {
			t.Errorf("wake expression not composed after state: %v", values[16])
		}
		text, _ := sample["canvasPng"].(string)
		image, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(text, "data:image/png;base64,"))
		if err != nil {
			t.Fatal(err)
		}
		savePetGUIImage(t, "neural-rest-"+name+"-canvas", image)
		delete(sample, "canvasPng")
		if image, err := w.CapturePage(); err == nil {
			savePetGUIImage(t, "neural-rest-"+name+"-window", image)
		} else {
			t.Error(err)
		}
		sample["name"] = name
		samples = append(samples, sample)
		realPetEvidence["neuralRestPoseSamples"] = samples
		if file := os.Getenv("WHALEISLE_PET_GUI_PROGRESS"); file != "" {
			_ = os.WriteFile(file, []byte("captured "+name+"\n"), 0600)
		}
		return sample
	}
	if _, err := w.Page().EvalContext(ctx, `window.__petRestProbe.arm("sleep-enter",.4,.9)`); err != nil {
		t.Fatal(err)
	}
	rest(true)
	capture("enter-middle", "sleep-enter", .4, .9)
	capture("settled", "sleep", 1.4, 100)
	if _, err := w.Page().EvalContext(ctx, `window.__petRestProbe.breathe(5200)`); err != nil {
		t.Fatal(err)
	}
	breath := wait(`window.__petRestProbe.breathSample`, func(v map[string]any) bool { return v["ready"] == true })
	realPetEvidence["neuralRestBreathing"] = breath
	if breath["count"].(float64) < 20 || breath["min44"].(float64) > -.45 || breath["max44"].(float64) < .45 || breath["rawHashes"].(float64) < 3 {
		t.Errorf("sleep did not carry a full changing neural breath: %#v", breath)
	}
	capture("breath-later", "sleep", 1.4, 100)
	if _, err := w.Page().EvalContext(ctx, `window.__petRestProbe.arm("wake",.3,.9)`); err != nil {
		t.Fatal(err)
	}
	rest(false)
	capture("wake-middle", "wake", .3, .9)
	wait(`(()=>({ready:window.__petRestProbe.phase==null&&document.getElementById('pet-body-hit').hidden}))()`, func(v map[string]any) bool { return v["ready"] == true })
	if _, err := w.Page().EvalContext(ctx, `window.__petRestProbe.stop()`); err != nil {
		t.Error(err)
	}
}

const neuralRestPoseObserver = `(()=>{
 const probe={phase:null,sample:null,breathSample:null,wanted:null,current:null,sequence:0,breath:null,arm(phase,min,max){this.sample=null;this.wanted={phase,min,max};return true;},breathe(ms){this.breath={until:performance.now()+ms,min:Infinity,max:-Infinity,count:0,hashes:new Set()};this.breathSample=null;return true;},stop(){this.wanted=null;this.breath=null;return true;}};
 function intercept(name,decorate){let value;Object.defineProperty(window,name,{configurable:true,get(){return value;},set(next){value=decorate(next);}});}
 intercept('WhalePetInteraction',api=>{const create=api.create;api.create=function(...args){const controller=create.apply(this,args),use=controller.useNeuralRest;controller.useNeuralRest=function(...values){const driver=use.apply(this,values);return {frame(frame){driver.frame(frame);const pose=new Float32Array(45);controller.pose(pose);probe.current={frame,pose:Array.from(pose),sequence:++probe.sequence};probe.phase=frame.phase;},close(){probe.phase=null;probe.current=null;return driver.close();}};};return controller;};return api;});
 intercept('WhaleTHA4',api=>{const open=api.open;api.open=async function(...args){const result=await open.apply(this,args),engine=result.engine,run=engine.frame;engine.frame=async function(values){const current=probe.current,input=Array.from(values),inputDelta=current?Math.max(...input.map((v,i)=>Math.abs(v-current.pose[i]))):0;const pixels=await run.call(this,values),afterDelta=Math.max(...input.map((v,i)=>Math.abs(v-values[i])));if(!pixels||!current)return pixels;let hash=2166136261;for(let i=0;i<pixels.length;i++)hash=Math.imul(hash^pixels[i],16777619);hash=hash>>>0;const frame=current.frame;
   const breath=probe.breath;if(breath&&frame.phase==='sleep'){breath.min=Math.min(breath.min,input[44]);breath.max=Math.max(breath.max,input[44]);breath.count++;breath.hashes.add(hash);if(performance.now()>=breath.until){probe.breathSample={ready:true,count:breath.count,min44:breath.min,max44:breath.max,rawHashes:breath.hashes.size};probe.breath=null;}}
   const wanted=probe.wanted;if(wanted&&frame.phase===wanted.phase&&frame.elapsed>=wanted.min&&frame.elapsed<=wanted.max){probe.wanted=null;queueMicrotask(()=>queueMicrotask(()=>{try{const canvas=document.getElementById('sprite'),target=document.getElementById('pet-body-hit');probe.sample={ready:true,phase:frame.phase,elapsed:frame.elapsed,alpha:frame.alpha,pose:input,inputDelta,afterDelta,sequence:current.sequence,revision:frame.revision,rawHash:hash,backingWidth:canvas.width,backingHeight:canvas.height,bodyTargetHidden:target.hidden,canvasTransform:canvas.style.transform,canvasPng:canvas.toDataURL()};}catch(error){probe.sample={ready:true,error:String(error)};}}));}return pixels;};return result;};return api;});
 window.__petRestProbe=probe;
})()`
