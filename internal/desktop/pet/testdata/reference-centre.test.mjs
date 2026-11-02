import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';

// Run the CURRENT reference functions read-only. No Electron process, source
// modification, user profile, model route or application API is involved.
const source=readFileSync(new URL('../../../../reference/electron/src/main/desktop-live2d.js',import.meta.url),'utf8');
const start=source.indexOf('  function displayForPoint(x, y) {');
const end=source.indexOf('\n  // The character\'s own screen position',start);
assert.ok(start>0&&end>start,'current reference function boundaries changed');
const dimensions=[...source.matchAll(/^const PET_(WIDTH|HEIGHT) = (\d+);$/gm)];
assert.equal(dimensions.length,2);
const declarations=dimensions.map(m=>`const PET_${m[1]}=${m[2]};`).join('\n');
const reference=`${declarations}\n${source.slice(start,end)}\nresult=overlayBounds();`;

function referenceDisplay(displays,x,y) {
  const context={screen:{getAllDisplays:()=>displays,getPrimaryDisplay:()=>displays[1]},petPosition:()=>({x,y}),finite:x=>Number.isFinite(Number(x))?Number(x):0,result:null};
  vm.runInNewContext(reference,context);
  return context.result.x;
}

test('current reference centre-first ignores mutable visual size',()=>{
  const displays=[{bounds:{x:-1920,y:0,width:1920,height:1080}},{bounds:{x:0,y:0,width:1920,height:1080}}];
  const records=[-121,-120,-100,100].map(x=>({x,selectedDisplayOrigin:referenceDisplay(displays,x,300)}));
  assert.deepEqual(records.map(v=>v.selectedDisplayOrigin),[-1920,0,0,0]);
  console.log(JSON.stringify({kind:'actual-reference-function-synthetic-screen',records}));
});

test('reference CSS physics constants match the port unit contract',()=>{
  const filename=new URL('../../../../reference/electron/src/renderer/pet-physics.js',import.meta.url);
  const physicsSource=readFileSync(filename,'utf8');
  const context={module:{exports:{}},Math};
  vm.runInNewContext(physicsSource,context);
  const physics=context.PetPhysics;
  const result=physics.throwStep(100,100,300,-200,.05,0,0,960,740);
  assert.equal(result.px,115);
  assert.equal(result.py,93.5);
  assert.equal(result.vx,300);
  assert.equal(result.vy,-130);
  const release=physics.estimateReleaseVelocity([[0,-400,100],[.05,-350,100],[.1,-300,100]],.1,physics.throwSpeedCap('standard'));
  assert.ok(Math.abs(release.vx-902.7055384769532)<1e-8);
  assert.equal(release.vy,0);
  console.log(JSON.stringify({kind:'actual-reference-physics',throw:result,release}));
});
