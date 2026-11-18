import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import { pathToFileURL } from 'node:url';
import { Readable } from 'node:stream';

test('Host adapter publishes complete skill once, retains metadata, rejects paths and cleans staging', async () => {
  const work = await fs.mkdtemp(path.join(os.tmpdir(),'mygo-skill-adapter-'));
  try {
    const frontmatter=path.resolve('runtime-rc4/vendor/deepseek-harness/packages/host/skill-inventory/lib/types/frontmatter.js');
    let source=await fs.readFile(new URL('./native-skill-import.mjs',import.meta.url),'utf8');
    source=source.replace("'__MYGO_SKILL_FRONTMATTER__'",JSON.stringify(pathToFileURL(frontmatter).href));
    source=source.replace("'__MYGO_SKILL_PROVIDER__'",JSON.stringify(pathToFileURL(path.resolve('runtime-rc4/vendor/deepseek-harness/packages/skill/skill-filesystem/lib/types/index.js')).href));
    const homeModule='export const resolveDshHome=()=>'+JSON.stringify(work)+';';
    source=source.replace("'__MYGO_HOME_PATHS__'",JSON.stringify('data:text/javascript;base64,'+Buffer.from(homeModule).toString('base64')));
    const module=await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
    let handler,invalidations=0;
    module.apply({
      effect:fn=>fn(),get(){},logger:{warn(message){throw Error(message)}},connection:{requestRejection:req=>req.headers.authorization==='owned-fixture'?undefined:401},
      webServer:{register:route=>{assert.equal(route.path,'/api/mygoSkills');handler=route.handler;return()=>{};}},
      skills:{invalidate(){invalidations++},async get(){return undefined}}
    });
    const invoke=async(method,payload,authorized=true)=>{
      const req=Readable.from([Buffer.from(JSON.stringify({type:'client-request',rpcId:'fixture',method,payload:{args:{request:payload}}}))]);
      req.method='POST';req.url='/api/'+method;req.headers={authorization:authorized?'owned-fixture':'wrong'};
      const res={code:0,body:'',writeHead(code){this.code=code},end(body=''){this.body=body}};
      await handler(req,res);return {code:res.code,body:res.code===200?JSON.parse(res.body):res.body};
    };
    assert.equal((await invoke('mygoSkills/capabilities',{},false)).code,401);
    const markdown='---\nname: fixture-skill\ndescription: Fixture\nmetadata:\n  owner: preserved\n---\n\nUse scripts/run.js\n';
    const files=[{path:'SKILL.md',data:Buffer.from(markdown).toString('base64')},{path:'scripts/run.js',data:Buffer.from('console.log("resource")').toString('base64'),executable:true}];
    const request={name:'fixture-skill',requestId:'a'.repeat(32),files};
    assert.equal((await invoke('mygoSkills/import',request)).body.result.ok,true);
    assert.equal(await fs.readFile(path.join(work,'skills/fixture-skill/SKILL.md'),'utf8'),markdown);
    assert.equal(await fs.readFile(path.join(work,'skills/fixture-skill/scripts/run.js'),'utf8'),'console.log("resource")');
    assert.equal((await invoke('mygoSkills/import',request)).body.result.ok,true);
    assert.equal(invalidations,1,'same request must not republish');
    assert.equal((await invoke('mygoSkills/status',request)).body.result.value.installed,true);
    const conflict=await invoke('mygoSkills/import',{...request,requestId:'b'.repeat(32)});
    assert.equal(conflict.body.result.ok,false);
    for(const name of ['../outside','C:/outside','scripts\\escape','Scripts/run.js']){
      const bad=await invoke('mygoSkills/import',{...request,name:'invalid-skill',requestId:'c'.repeat(32),files:[...files,{path:name,data:''}]});
      assert.equal(bad.body.result.ok,false);
    }
    assert.deepEqual(await fs.readdir(path.join(work,'runtime/skill-import')),[]);
    assert.equal((await invoke('mygoSkills/import',{...request,name:'disabled-skill',requestId:'d'.repeat(32),enabled:false,files:[{path:'SKILL.md',data:Buffer.from(markdown.replace('fixture-skill','disabled-skill')).toString('base64')}]})).body.result.ok,true);
    const disabled=await fs.readFile(path.join(work,'skills/disabled-skill/SKILL.md'),'utf8');
    assert.match(disabled,/disable-model-invocation: true/);assert.match(disabled,/user-invocable: false/);assert.match(disabled,/owner: preserved/);
  } finally {
    // This fixture created work and only regular files/directories under it.
    assert.equal(path.dirname(work),os.tmpdir());
    assert.ok(path.basename(work).startsWith('mygo-skill-adapter-'));
    await fs.rm(work,{recursive:true,force:true});
  }
});
