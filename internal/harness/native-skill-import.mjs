// Desktop-authenticated local skill publication. Host owns the destination,
// skill registry, atomic publication and receipt; no arbitrary path is accepted.
import { mkdir, mkdtemp, readFile, writeFile, chmod, rename, rm, lstat } from 'node:fs/promises';
import { join, dirname } from 'node:path';
import { parseSkillMarkdown, renderSkillInvocationMarkdown } from '__MYGO_SKILL_FRONTMATTER__';
import { resolveDshHome } from '__MYGO_HOME_PATHS__';
import { FileSystemSkillProvider } from '__MYGO_SKILL_PROVIDER__';

export const name = 'mygo-skill-import';
export const inject = ['webServer', 'connection', 'skills'];
const receiptName = '.mygo-import.json';
const skillName = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const active = new Map();

function identity(request) {
  if (!request || typeof request.name !== 'string' || !skillName.test(request.name) || request.name.length > 64 || !/^[a-f0-9]{32}$/.test(request.requestId ?? '')) throw Error('Invalid skill import identity');
  return request;
}

function validatedFiles(files) {
  if (!Array.isArray(files) || files.length === 0 || files.length > 4096) throw Error('Skill import needs 1–4096 files');
  let size = 0;
  const seen = new Set();
  return files.map(file => {
    if (typeof file.path !== 'string' || !file.path || /[\\:\x00]/.test(file.path) || file.path.split('/').some(p => !p || p === '.' || p === '..' || /[. ]$/.test(p) || /^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\.|$)/i.test(p)) || file.path.toLowerCase() === receiptName) throw Error('Invalid relative skill file path');
    const key = file.path.toLowerCase();
    if (seen.has(key)) throw Error('Duplicate skill file path');
    seen.add(key);
    if (typeof file.data !== 'string' || !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(file.data)) throw Error('Invalid skill file data');
    const data = Buffer.from(file.data,'base64');
    size += data.length;
    if (size > 64*1024*1024) throw Error('Skill import exceeds 64 MiB');
    return {path:file.path,data,executable:file.executable===true};
  });
}

async function directoryReceipt(home, request) {
  const target=join(home,'skills',request.name);
  try {
    if ((await lstat(target)).isSymbolicLink()) return false;
    const receipt=JSON.parse(await readFile(join(target,receiptName),'utf8'));
    return receipt.requestId===request.requestId && receipt.name===request.name;
  } catch(error) {
    if (error.code==='ENOENT') return false;
    throw error;
  }
}

async function importedSkill(ctx,home,name) {
  // rc.4 presets own filesystem discovery. Verify through the same reader
  // without registering a global provider or starting background watchers.
  const path=join(home,'skills',name,'SKILL.md');
  const provider=new FileSystemSkillProvider(ctx,{invalidate(){},signal:new AbortController().signal},{includeDefaultRoots:false,watch:false});
  try {
    const skill=await provider.get({source:'user-dsh',locator:{path,directory:dirname(path)}},{});
    if(!skill||skill.name!==name)throw Error('Published skill cannot be read by the Host');
    return {name:skill.name,content:skill.content,modelInvocable:skill.invocation.modelInvocable,userInvocable:skill.invocation.userInvocable};
  } finally {await provider.dispose();}
}

async function publish(ctx, home, input) {
  const request=identity(input);
  if (active.has(request.name)) throw Error('A skill import is already in progress');
  active.set(request.name,true);
  let stage;
  try {
    if (await directoryReceipt(home,request)) return {installed:true,name:request.name};
    const files=validatedFiles(request.files);
    const markdown=files.find(file=>file.path==='SKILL.md');
    if (!markdown) throw Error('Skill directory must contain SKILL.md');
    const parsed=parseSkillMarkdown(markdown.data.toString('utf8'));
    if (parsed.data.name!==request.name || typeof parsed.data.description!=='string' || !parsed.data.description.trim()) throw Error('SKILL.md name and description must match the inspected skill');
    if (await ctx.skills.get(request.name,{})) throw Error('Skill already exists');
    const root=join(home,'skills');
    await mkdir(root,{recursive:true,mode:0o700});
    if ((await lstat(root)).isSymbolicLink()) throw Error('Skill root is redirected');
    const target=join(root,request.name);
    try {await lstat(target);throw Error('Skill directory already exists');} catch(error){if(error.code!=='ENOENT')throw error;}
    const work=join(home,'runtime','skill-import');
    await mkdir(work,{recursive:true,mode:0o700});
    stage=await mkdtemp(join(work,'import-'));
    for(const file of files){
      const dest=join(stage,...file.path.split('/'));
      await mkdir(dirname(dest),{recursive:true,mode:0o700});
      let data=file.data;
      if(file.path==='SKILL.md' && typeof request.enabled==='boolean'){
        data=Buffer.from(renderSkillInvocationMarkdown({existingData:parsed.data,content:parsed.body,modelInvocable:request.enabled && parsed.data['disable-model-invocation']!==true,userInvocable:request.enabled && parsed.data['user-invocable']!==false}));
      }
      await writeFile(dest,data,{flag:'wx',mode:file.executable?0o700:0o600});
      if(file.executable)await chmod(dest,0o700);
    }
    await writeFile(join(stage,receiptName),JSON.stringify({name:request.name,requestId:request.requestId}),{flag:'wx',mode:0o600});
    // Recheck conflicts immediately before publication. Existing content is
    // never removed to make an import succeed.
    if(await ctx.skills.get(request.name,{}))throw Error('Skill appeared during import');
    try {await lstat(target);throw Error('Skill directory appeared during import');} catch(error){if(error.code!=='ENOENT')throw error;}
    await rename(stage,target);stage=undefined;
    ctx.skills.invalidate();
    await importedSkill(ctx,home,request.name);
    return {installed:true,name:request.name};
  } finally {
    active.delete(request.name);
    if(stage)await rm(stage,{recursive:true,force:true});
  }
}

export function apply(ctx) {
  ctx.effect(()=>ctx.webServer.register({
    kind:'prefix',path:'/api/mygoSkills',
    async handler(req,res){
      const rejection=ctx.connection.requestRejection(req);
      if(rejection!==undefined){res.writeHead(rejection);res.end('rejected');return;}
      const method=new URL(req.url??'/','http://dsh.internal').pathname.slice('/api/'.length);
      if(req.method!=='POST'||!['mygoSkills/capabilities','mygoSkills/import','mygoSkills/status'].includes(method)){res.writeHead(404);res.end();return;}
      let request;
      const reply=result=>{res.writeHead(200,{'content-type':'application/json'});res.end(JSON.stringify({type:'server-response',rpcId:request?.rpcId??'invalid',result}));};
      try{
        let size=0;const chunks=[];
        for await(const chunk of req){size+=chunk.length;if(size>90*1024*1024)throw Error('Skill import request too large');chunks.push(chunk);}
        request=JSON.parse(Buffer.concat(chunks).toString('utf8'));
        if(request.type!=='client-request'||typeof request.rpcId!=='string'||request.method!==method)throw Error('Invalid request');
        const home=resolveDshHome();
        if(method==='mygoSkills/capabilities'){reply({ok:true,value:{importDirectory:true}});return;}
        const input=request.payload?.args?.request;
        if(method==='mygoSkills/status'){
          identity(input);
          const installed=await directoryReceipt(home,input);
          reply({ok:true,value:{installed,...installed?{detail:await importedSkill(ctx,home,input.name)}:{}}});return;
        }
        reply({ok:true,value:await publish(ctx,home,input)});
      }catch(error){reply({ok:false,error:{code:'mygoSkills/rejected',message:String(error?.message??error),details:{}}});}
    }
  }));
}
