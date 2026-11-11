// Isolated Host fixture: real WhaleTasks/Feishu binding, local SDK recorder only.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {pathToFileURL} from 'node:url';
export const name='qa-proactive-channel';
export const inject=['webServer','connection'];
export async function apply(ctx){
  const vendor=process.env.WHALEISLE_QA_VENDOR;
  const marker=path.join(process.env.WHALEISLE_QA_WORKSPACE,'qa-feishu-effects.jsonl');
  const {createFeishuTaskBinding}=await import(pathToFileURL(path.join(vendor,'dsh-im/plugin-src/host/channels/feishu/whale-tasks.mjs')));
  let service,ready=false,mode='sent',setup;
  const bot={id:'qa-bot',appId:'qa-public-app',createdAt:'qa-created',connectedAt:'qa-connected',ownerOpenIds:['qa-owner']};
  const route={channel:'feishu',botId:bot.id,actor:'qa-owner',chatId:'qa-chat',messageId:'qa-original',incarnation:crypto.createHash('sha256').update(JSON.stringify([bot.id,bot.appId,bot.createdAt,bot.connectedAt])).digest('hex')};
  const abort=new AbortController();ctx.effect(()=>()=>abort.abort());
  async function install(){
    const current=ctx.get('whaleTasks');if(!current?.available)return;
    if(setup)return setup;
    setup=(async()=>{
      service=current;
      const scope=service.scope.get();await service.scope.set({...scope,imDefault:true,sessionId:scope.sessionId||'qa-resident'},scope);
      await service.setOwner(bot.id,'qa-owner');
      await service.store.put({id:'qa-source-task',revision:1,generation:1,origin:{eventId:'qa-origin',sessionId:service.scope.get().sessionId,messageId:'qa-original',rpcId:'qa-inbound',actor:route.actor,kind:'im',route,text:'/task {}',files:[]},goal:'已授权的 QA 飞书会话',criteria:[],inputs:[{id:'input-1',path:'qa.txt',name:'qa.txt',hash:'qa-hash',bytes:2,text:'QA'}],model:{provider:'qa',model:'qa'},state:'completed',hold:false,createdAt:1,updatedAt:1,error:'',effects:[],history:[]});
      const sdk={im:{message:{create:async request=>{
        fs.appendFileSync(marker,JSON.stringify(request)+'\n');
        if(mode==='unknown')return {code:0,data:{}};
        return {code:0,data:{message_id:'qa-message-'+fs.readFileSync(marker,'utf8').trim().split('\n').length}};
      }}}};
      createFeishuTaskBinding(ctx,{botId:bot.id,botConfig:bot,workspaces:{has:()=>true,agentPresetFor:()=>null},configStore:{list:()=>[bot]}}).attach({client:sdk,signal:abort.signal,allowedSenderOpenIds:new Set(['qa-owner'])});
      ready=true;
    })();
    return setup;
  }
  ctx.on('whale-tasks/ready',()=>void install());await install();
  ctx.effect(()=>ctx.webServer.register({kind:'prefix',path:'/qa-proactive-channel',async handler(req,res){
    const rejection=ctx.connection.requestRejection(req);if(rejection!==undefined){res.writeHead(rejection);res.end();return;}
    let body='';for await(const chunk of req)body+=chunk;
    const request=JSON.parse(body);
    try{
      await install();
      if(request.method==='mode')mode=request.payload.mode;
      if(request.method==='revoke')await service.revokeOwner(bot.id);
      if(request.method==='restore')await service.setOwner(bot.id,'qa-owner');
      const effects=fs.existsSync(marker)?fs.readFileSync(marker,'utf8').trim().split('\n').filter(Boolean).map(line=>JSON.parse(line)):[];
      res.writeHead(200,{'content-type':'application/json'});res.end(JSON.stringify({type:'server-response',rpcId:request.rpcId,result:{ok:true,value:{ready,effects}}}));
    }catch(error){res.writeHead(200,{'content-type':'application/json'});res.end(JSON.stringify({type:'server-response',rpcId:request.rpcId,result:{ok:false,error:{code:'qa',message:error.message}}}));}
  }}));
}
