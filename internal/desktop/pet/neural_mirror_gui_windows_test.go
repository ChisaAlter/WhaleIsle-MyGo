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

// This fixture uses TestMain's owned temporary profile/cache/CODEX_HOME. It never
// drives a user window or changes the real cursor. Only the owned page's canvas
// draw is observed, and its prototype is restored before that page is closed.
func TestRealPetNeuralMirror(t *testing.T) {
	if os.Getenv("WHALEISLE_PET_GUI") != "1" || os.Getenv("WHALEISLE_PET_GUI_MIRROR") != "1" {
		t.Skip("explicit owned neural mirror GUI fixture only")
	}
	dir := os.Getenv("WHALEISLE_PET_RENDER_ASSETS")
	if dir == "" {
		t.Fatal("mirror fixture requires explicit staged assets")
	}
	if _, err := loadRenderAssets(dir); err != nil {
		t.Fatalf("staged assets: %v", err)
	}
	previousRoot := petRenderRoot
	petRenderRoot = func() string { return dir }
	t.Cleanup(func() { petRenderRoot = previousRoot })
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	behavior := config.DefaultPetBehavior()
	behavior.Wander, behavior.SelfTalk = false, false
	if err := store.SaveDesktopPet(config.DesktopPet{Enabled: true, Neural: true, Behavior: behavior}); err != nil {
		t.Fatal(err)
	}
	var p *Companion
	var w *mygo.Window
	mygo.RunOnMain(func() { p = New(store); p.SetReady(true); w = p.win })
	if p == nil || w == nil || w.IsDestroyed() {
		t.Fatal("owned neural companion window was not created")
	}
	t.Cleanup(func() { mygo.RunOnMain(p.Close) })
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	wait := func(script string, ready func(map[string]any) bool) map[string]any {
		t.Helper()
		var last any
		var evalErr error
		for ctx.Err() == nil {
			last, evalErr = w.Page().EvalContext(ctx, script)
			if value, ok := last.(map[string]any); evalErr == nil && ok && ready(value) {
				return value
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("owned mirror page wait expired: last=%#v error=%v", last, evalErr)
		return nil
	}
	dom := wait(`(()=>({renderer:document.body.dataset.renderer||"",provider:document.body.dataset.provider||"",superResolution:document.body.dataset.superResolution||"",width:document.getElementById("sprite").getBoundingClientRect().width,height:document.getElementById("sprite").getBoundingClientRect().height}))()`, func(value map[string]any) bool { return value["renderer"] == "neural" })
	if dom["width"] != float64(240) || dom["height"] != float64(260) || dom["superResolution"] != "anime4k" {
		t.Fatalf("owned neural renderer geometry/SR changed: %#v", dom)
	}
	realPetEvidence["dom"] = dom
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := w.Page().EvalContext(cleanupCtx, `window.__petMirrorProbe?.restore()`); err != nil {
			t.Errorf("restore owned mirror observer: %v", err)
		}
	})
	if _, err := w.Page().EvalContext(ctx, neuralMirrorObserver); err != nil {
		t.Fatalf("install own-page mirror observer: %v", err)
	}
	var samples []map[string]any
	for _, step := range []struct {
		name   string
		facing int
	}{{"right", 1}, {"left", -1}, {"right-return", 1}} {
		if _, err := w.Page().EvalContext(ctx, fmt.Sprintf(`window.__petMirrorProbe.arm(%d)`, step.facing)); err != nil {
			t.Fatal(err)
		}
		var nativeFacing int
		var mode string
		mygo.RunOnMain(func() {
			// This is the production direction source: tickInteraction copies
			// motion.facing into interactionView.Facing, which the page polls.
			// Wandering is disabled and no stage/cursor position is changed.
			p.motion.facing = step.facing
			p.tickInteraction(time.Now())
			nativeFacing, mode = p.interaction.view.Facing, p.interaction.view.Mode
		})
		if nativeFacing != step.facing || mode != "" {
			t.Fatalf("production direction publish failed: face=%d mode=%q", nativeFacing, mode)
		}
		sample := wait(`window.__petMirrorProbe.sample`, func(value map[string]any) bool { return value["ready"] == true })
		record := make(map[string]any)
		for key, value := range sample {
			if !strings.HasSuffix(key, "Png") {
				record[key] = value
			}
		}
		record["nativeFacing"], record["name"] = nativeFacing, step.name
		samples = append(samples, record)
		realPetEvidence["neuralMirrorSamples"] = samples
		if message, _ := sample["error"].(string); message != "" {
			t.Fatalf("own-page pixel observer failed: %s", message)
		}
		if sample["facing"] != float64(step.facing) || sample["browserReferenceMismatches"] != float64(0) || sample["nontransparentPixels"].(float64) < 100 || sample["asymmetricChannels"].(float64) < 100 {
			t.Errorf("same-frame source/canvas mirror mismatch: %#v", record)
		}
		if sample["width"] != float64(240) || sample["height"] != float64(260) {
			t.Fatalf("mirror changed DOM size: %#v", record)
		}
		for _, key := range []string{"actualPng", "sourcePng", "referencePng", "byteReversePng", "differencePng"} {
			dataURL, ok := sample[key].(string)
			if !ok || !strings.HasPrefix(dataURL, "data:image/png;base64,") {
				t.Fatalf("own canvas capture %s missing", key)
			}
			image, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, "data:image/png;base64,"))
			if err != nil {
				t.Fatal(err)
			}
			savePetGUIImage(t, "neural-mirror-"+step.name+"-"+key, image)
			delete(sample, key)
		}
	}
	realPetEvidence["neuralMirrorSamples"] = samples
	t.Logf("owned WebView2 neural mirror: %v; same-frame reference Canvas feet formula matches right/left/right exactly; byte-reversal differences recorded separately; DOM stays 240x260", dom)
}

// Observe after the real draw while its transformation is still active. Compare
// against both an independent byte-reversal of the same source and the reference
// feet-pivot Canvas formula. Preserve quantization differences instead of
// treating byte reversal and browser Canvas sampling as interchangeable.
const neuralMirrorObserver = `(()=>{
 const canvas=document.getElementById('sprite'), proto=CanvasRenderingContext2D.prototype;
 const original=proto.drawImage;
 const surface=()=>{const out=document.createElement('canvas');out.width=canvas.width;out.height=canvas.height;return out;};
 const sourceCanvas=surface(),referenceCanvas=surface(),differenceCanvas=surface(),byteReverseCanvas=surface();
 const sourceContext=sourceCanvas.getContext('2d',{willReadFrequently:true}),referenceContext=referenceCanvas.getContext('2d');
 const probe={sample:null,wanted:null,arm(facing){this.sample=null;this.wanted=facing;return true;},restore(){if(proto.drawImage===observed)proto.drawImage=original;delete window.__petMirrorProbe;return true;}};
 function observed(...args){
  const result=original.apply(this,args);
  if(this.canvas!==canvas||probe.wanted===null||document.body.dataset.renderer!=='neural')return result;
  const m=this.getTransform(),facing=m.a<0?-1:1;
  if(facing!==probe.wanted)return result;
  probe.wanted=null;
  try{
   const width=canvas.width,height=canvas.height;
   sourceContext.clearRect(0,0,width,height);original.apply(sourceContext,args);
   referenceContext.clearRect(0,0,width,height);referenceContext.save();
   // Reference drawLive uses the feet pivot; translating it into this bounded
   // frame is equivalent to draw(src,-w/2,-h,w,h) at (w/2,h).
   referenceContext.translate(width/2,height);referenceContext.scale(facing,1);
   if(args.length===5)original.call(referenceContext,args[0],-width/2,-height,width,height);
   else original.call(referenceContext,args[0],...args.slice(1,5),-width/2,-height,width,height);
   referenceContext.restore();
   const source=sourceContext.getImageData(0,0,width,height).data,actual=this.getImageData(0,0,width,height).data,reference=referenceContext.getImageData(0,0,width,height).data;
   const diff=differenceCanvas.getContext('2d').createImageData(width,height),reverse=byteReverseCanvas.getContext('2d').createImageData(width,height);
   let mismatchChannels=0,maxDelta=0,asymmetricChannels=0,nontransparentPixels=0,browserReferenceMismatches=0;
   const byChannel=[0,0,0,0],byAlpha={transparent:0,feathered:0,opaque:0},locations=[];
   let minX=width,minY=height,maxX=-1,maxY=-1;
   for(let y=0;y<height;y++)for(let x=0;x<width;x++){
    const i=(y*width+x)*4,opposite=(y*width+width-1-x)*4,j=facing<0?opposite:i;
    if(actual[i+3]>=4)nontransparentPixels++;
    const changes=[];
    for(let c=0;c<4;c++){
     reverse.data[i+c]=source[j+c];
     const delta=Math.abs(actual[i+c]-source[j+c]);
     if(delta){mismatchChannels++;maxDelta=Math.max(maxDelta,delta);byChannel[c]++;changes.push({channel:c,actual:actual[i+c],expected:source[j+c],delta});}
     if(actual[i+c]!==reference[i+c])browserReferenceMismatches++;
     if(source[i+c]!==source[opposite+c])asymmetricChannels++;
    }
    if(changes.length){
     byAlpha[source[j+3]===0?'transparent':source[j+3]===255?'opaque':'feathered']++;
     minX=Math.min(minX,x);minY=Math.min(minY,y);maxX=Math.max(maxX,x);maxY=Math.max(maxY,y);
     diff.data[i]=255;diff.data[i+1]=changes.some(v=>v.channel===3)?255:0;diff.data[i+2]=255;diff.data[i+3]=255;
     if(locations.length<64)locations.push({x,y,alpha:actual[i+3],sourceAlpha:source[j+3],changes});
    }
   }
   differenceCanvas.getContext('2d').putImageData(diff,0,0);byteReverseCanvas.getContext('2d').putImageData(reverse,0,0);
   const r=canvas.getBoundingClientRect();
   probe.sample={ready:true,facing,matrix:[m.a,m.b,m.c,m.d,m.e,m.f],actualContext:this.getContextAttributes?.(),referenceContext:referenceContext.getContextAttributes?.(),sourceContext:sourceContext.getContextAttributes?.(),browserReferenceMismatches,mismatchChannels,maxDelta,byChannel,byAlpha,mismatchBounds:maxX<0?null:{minX,minY,maxX,maxY},locations,asymmetricChannels,nontransparentPixels,width:r.width,height:r.height,backingWidth:width,backingHeight:height,renderer:document.body.dataset.renderer,provider:document.body.dataset.provider,superResolution:document.body.dataset.superResolution,actualPng:canvas.toDataURL('image/png'),sourcePng:sourceCanvas.toDataURL('image/png'),referencePng:referenceCanvas.toDataURL('image/png'),byteReversePng:byteReverseCanvas.toDataURL('image/png'),differencePng:differenceCanvas.toDataURL('image/png')};
  }catch(error){probe.sample={ready:true,error:String(error)};}
  return result;
 }
 proto.drawImage=observed;window.__petMirrorProbe=probe;return true;
})()`
