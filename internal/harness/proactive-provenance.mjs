// Loaded before the CLI so registrations carry the actual Cordis callback,
// not a tool-supplied name, description or provenance field.
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {existsSync,realpathSync} from 'node:fs';

const origins=new WeakMap(),callbacks=new Map(),authorized=new WeakMap(),consumed=new WeakSet();
let gate,acpRejector;

export function registerKnownToolCallback(callback,source){
  if(typeof callback==='function'&&source)callbacks.set(callback,source);
}

export function installRuntimeProvenance(proto){
  const register=proto.register,dispatch=proto.dispatchScheduledExecution,body=proto.dispatchToolBody;
  proto.register=function(definition){
    const callback=this.ctx?.fiber?.runtime?.callback;
    const undo=register.call(this,definition);
    origins.set(definition,{callback,source:callbacks.get(callback)??'plugin:unclassified'});
    return undo;
  };
  proto.dispatchScheduledExecution=async function(exec){
    if(!gate?.binding(exec.agent))return dispatch.call(this,exec);
    const definition=this.get(exec.name,exec.agent);
    const identity={definition,name:exec.name,args:JSON.stringify(exec.arguments)};
    try{
      return await gate.execute(exec,async()=>{
        if(this.get(exec.name,exec.agent)!==definition)throw Error('Tool definition changed during task approval');
        authorized.set(exec,identity);
        try{return await dispatch.call(this,exec);}finally{authorized.delete(exec);}
      });
    }catch(error){
      // A denied ACP proxy must also reject the waiting external permission;
      // otherwise the Host would finish its tool while ACP stayed blocked.
      if(origins.get(definition)?.source==='desktop:acp')await acpRejector?.(exec);
      const message=String(error?.message??error);
      return {kind:'post-result',result:{content:[{type:'text',text:message}],isError:true,error:{message}}};
    }
  };
  proto.dispatchToolBody=function(exec){
    const identity=authorized.get(exec);
    if(identity&&(identity.definition!==this.get(exec.name,exec.agent)||identity.name!==exec.name||identity.args!==JSON.stringify(exec.arguments)))throw Error('Tool identity or arguments changed after task approval');
    return body.call(this,exec);
  };
}

const root=process.env.DSH_HARNESS_ROOT;
if(root){
  const load=target=>import(pathToFileURL(realpathSync(target)).href);
  for(const [relative,source] of [
    ['packages/fs/tool-fs/lib/index.js','harness:fs'],
    ['packages/fs/tool-fs-search/lib/index.js','harness:fs-search'],
  ]){
    const module=await load(path.join(root,relative));registerKnownToolCallback(module.apply,source);
  }
  const acp=path.join(root,'..','dsh-acp','lib','index.js');
  if(existsSync(acp))registerKnownToolCallback((await load(acp)).apply,'desktop:acp');
  const {ToolRuntime}=await load(path.join(root,'packages/core/tools/lib/index.js'));
  installRuntimeProvenance(ToolRuntime.prototype);
}

export function toolSource(tools,exec){return origins.get(tools.get(exec.name,exec.agent))?.source??'plugin:unclassified';}
export function installDispatchGate(value){gate=value;return()=>{if(gate===value)gate=undefined;};}
const api=Object.freeze({
  binding:agent=>gate?.binding(agent),
  consumeAuthorization(exec){
    if(!exec||!authorized.has(exec)||consumed.has(exec))return false;
    consumed.add(exec);
    return true;
  },
  setAcpRejector(fn){acpRejector=fn;return()=>{if(acpRejector===fn)acpRejector=undefined;};},
});
if(!Object.getOwnPropertyDescriptor(globalThis,Symbol.for('whaleisle.nativeTaskTools'))){
  Object.defineProperty(globalThis,Symbol.for('whaleisle.nativeTaskTools'),{value:api,configurable:false});
}
