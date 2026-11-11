import fs from 'node:fs';
import path from 'node:path';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import {pathToFileURL} from 'node:url';
import readline from 'node:readline';

async function runMcpBridge(server){
  if(!server?.command||!Array.isArray(server.args))throw Error('whaleisle MCP server was not injected');
  const env={...process.env};
  for(const entry of server.env||[])if(entry?.name)env[entry.name]=String(entry.value??'');
  const child=spawn(server.command,server.args,{env,stdio:['pipe','pipe','pipe']});
  const waiters=new Map();
  let seq=1;
  readline.createInterface({input:child.stdout,crlfDelay:Infinity}).on('line',line=>{
    try{const msg=JSON.parse(line);if(msg.id!==undefined)waiters.get(msg.id)?.(msg);waiters.delete(msg.id);}catch{}
  });
  const call=(method,params)=>new Promise((resolve,reject)=>{
    const id=seq++;
    const timer=setTimeout(()=>{waiters.delete(id);reject(Error(`MCP timeout: ${method}`));},10000);
    waiters.set(id,msg=>{clearTimeout(timer);msg.error?reject(Error(msg.error.message||method)):resolve(msg.result);});
    child.stdin.write(JSON.stringify({jsonrpc:'2.0',id,method,params})+'\n');
  });
  try{
    await call('initialize',{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'qa-local',version:'1'}});
    child.stdin.write(JSON.stringify({jsonrpc:'2.0',method:'notifications/initialized'})+'\n');
    const listed=await call('tools/list',{});
    const skills=await call('tools/call',{name:'whaleisle_skill_list',arguments:{}});
    const presented=await call('tools/call',{name:'whaleisle_present',arguments:{files:[{path:'qa-mcp-deliverable.html',description:'QA owned MCP deliverable'}]}} );
    child.stdin.end();
    await Promise.race([once(child,'exit'),new Promise(r=>setTimeout(r,3000))]);
    if(child.exitCode===null)child.kill();
    return {
      tools:(listed?.tools||[]).map(t=>t.name),
      skillListOk:!skills?.isError,
      presentOk:!presented?.isError,
      presentText:presented?.content?.[0]?.text||'',
    };
  }finally{
    if(child.exitCode===null){try{child.kill();}catch{}}
  }
}

if(process.argv[2]==='configure'){
  const [home,vendor,node]=process.argv.slice(3);
  const p=await import(pathToFileURL(path.join(vendor,'dsh-acp/lib/presets.js')));
  p.saveAgentsConfig([
    ...p.BUILTIN_AGENT_PRESETS.map(a=>({...a,enabled:false})),
    {id:'qa-local',name:'QA Local ACP',registrySource:'local',command:node,args:[process.argv[1],'serve'],env:{QA_ACP_ROOT:path.dirname(process.argv[1])},enabled:true,permissionPolicy:'ask',models:[{id:'mock-fast',name:'QA Fast'},{id:'mock-pro',name:'QA Pro'}]},
  ],path.join(home,'acp-agents.json'));
}else if(process.argv[2]==='serve'){
  const root=process.env.QA_ACP_ROOT;
  const record=v=>fs.appendFileSync(path.join(root,'acp-protocol.jsonl'),JSON.stringify({...v,pid:process.pid})+'\n');
  const send=v=>process.stdout.write(JSON.stringify({jsonrpc:'2.0',...v})+'\n');
  const result=(id,value)=>send({id,result:value});
  const update=(sessionId,update)=>send({method:'session/update',params:{sessionId,update}});
  const options=[{configId:'model',name:'Model',category:'model',type:'select',currentValue:'mock-fast',options:['mock-fast','mock-pro'].map(value=>({value,name:value}))},{configId:'reasoning_effort',name:'Reasoning effort',category:'thought_level',type:'select',currentValue:'medium',options:['low','medium','high'].map(value=>({value,name:value}))}];
  const setup={models:{currentModelId:'mock-fast',availableModels:[{modelId:'mock-fast',name:'QA Fast'},{modelId:'mock-pro',name:'QA Pro'}]},configOptions:options};
  const pending=new Map();let next=10000;let waiting;let mcpServer;
  const ask=params=>new Promise(resolve=>{const id=next++;pending.set(id,resolve);send({id,method:'session/request_permission',params})});
  record({event:'started'});
  readline.createInterface({input:process.stdin,crlfDelay:Infinity}).on('line',async line=>{
    let msg;try{msg=JSON.parse(line)}catch{return}
    if(!msg.method){pending.get(msg.id)?.(msg.result);pending.delete(msg.id);return}
    const {id,method,params={}}=msg;
    const logged=structuredClone(params);
    for(const server of logged.mcpServers||[])if(server.env)server.env=server.env.map(e=>({name:e.name,value:'[owned ephemeral value omitted]'}));
    record({event:method,params:logged});
    if(method==='initialize')return result(id,{protocolVersion:2,info:{name:'qa-local',version:'1'},capabilities:{session:{}},authMethods:[]});
    if(method==='session/new'){
      mcpServer=(params.mcpServers||[]).find(s=>s.name==='whaleisle')||mcpServer;
      return result(id,{sessionId:'qa-acp-persisted',...setup});
    }
    if(method==='session/resume'){
      mcpServer=(params.mcpServers||[]).find(s=>s.name==='whaleisle')||mcpServer;
      return result(id,{sessionId:params.sessionId,...setup});
    }
    if(method==='session/set_config_option')return result(id,{configOptions:options.map(o=>({...o,currentValue:o.configId===params.configId?params.value:o.currentValue}))});
    if(method==='session/set_model'||method==='session/set_mode')return result(id,{});
    if(method==='session/cancel'){
      if(waiting){update(waiting,{sessionUpdate:'state_update',state:'idle',stopReason:'cancelled'});waiting=undefined}
      if(id!==undefined)result(id,{});return;
    }
    if(method!=='session/prompt')return result(id,{});
    const text=params.prompt.map(x=>x.text||'').join('');const sid=params.sessionId;
    result(id,{});
    update(sid,{sessionUpdate:'agent_thought_chunk',messageId:'thought-'+id,content:{type:'text',text:'QA_ACP_THOUGHT '+text}});
    if(text.includes('QA_ACP_WAIT')){waiting=sid;update(sid,{sessionUpdate:'agent_message_chunk',messageId:'reply-'+id,content:{type:'text',text:'QA_ACP_WAITING'}});return}
    let suffix='';
    if(text.includes('QA_ACP_MCP')){
      try{
        const mcp=await runMcpBridge(mcpServer);
        record({event:'mcp-bridge-result',...mcp});
        if(mcp.tools.includes('whaleisle_present')&&mcp.tools.includes('whaleisle_skill_list')&&mcp.skillListOk&&mcp.presentOk)suffix=' QA_ACP_MCP_OK';
      }catch(error){
        record({event:'mcp-bridge-error',message:String(error?.message??error)});
      }
    }
    if(text.includes('QA_ACP_TOOL')){
      const toolCall={toolCallId:'qa-write-'+id,title:'QA ACP write owned file',kind:'edit',status:'pending',rawInput:{path:path.join(root,'acp-effect.txt')}};
      update(sid,{sessionUpdate:'tool_call',...toolCall});
      const answer=await ask({sessionId:sid,title:toolCall.title,subject:{type:'tool_call',toolCall},options:[{optionId:'allow',name:'Allow once',kind:'allow_once'},{optionId:'deny',name:'Reject once',kind:'reject_once'}]});
      const allowed=answer?.outcome?.optionId==='allow';record({event:'permission-result',allowed});
      if(allowed)fs.appendFileSync(path.join(root,'acp-effect.txt'),'effect\n');
      update(sid,{sessionUpdate:'tool_call_update',...toolCall,status:allowed?'completed':'failed',content:[{type:'content',content:{type:'text',text:allowed?'QA_ACP_ALLOWED':'QA_ACP_DENIED'}}]});
    }
    update(sid,{sessionUpdate:'agent_message_chunk',messageId:'reply-'+id,content:{type:'text',text:'QA_ACP_REPLY '+text+suffix}});
    update(sid,{sessionUpdate:'state_update',state:'idle',stopReason:'end_turn'});
  });
}
