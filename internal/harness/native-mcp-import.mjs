// Create-only import uses the managed MCP owner's queue and file lock.
import {upsertRecord} from '__MYGO_MCP_DOCUMENT__';
export const name = 'mygo-mcp-import';
export const inject = ['webServer', 'connection', 'mcpServersFile'];
export function apply(ctx) {
  ctx.effect(() => ctx.webServer.register({kind:'prefix',path:'/api/mygoMCP',async handler(req,res) {
    const rejected=ctx.connection.requestRejection(req);
    if(rejected!==undefined){res.writeHead(rejected);res.end();return;}
    let request;
    const reply=result=>{res.writeHead(200,{'content-type':'application/json'});res.end(JSON.stringify({type:'server-response',rpcId:request?.rpcId??'invalid',result}));};
    try {
      const method=new URL(req.url??'/','http://dsh.internal').pathname.slice('/api/'.length);
      if(req.method!=='POST'||!['mygoMCP/capabilities','mygoMCP/import'].includes(method)){res.writeHead(404);res.end();return;}
      let size=0;const chunks=[];
      for await(const chunk of req){size+=chunk.length;if(size>4*1024*1024)throw Error('MCP import request too large');chunks.push(chunk);}
      request=JSON.parse(Buffer.concat(chunks).toString('utf8'));
      if(request.type!=='client-request'||typeof request.rpcId!=='string'||request.method!==method)throw Error('Invalid request');
      if(method==='mygoMCP/capabilities'){reply({ok:true,value:{createOnly:typeof ctx.mcpServersFile.mutate==='function'}});return;}
      const spec=request.payload?.args?.request?.spec;
      await ctx.mcpServersFile.mutate(document=>{
        if(document.servers.some(row=>row.id===spec?.id||row.serverName===spec?.serverName))throw Error('MCP identity already exists; edit it in MCP Settings');
        return upsertRecord(document,spec);
      });
      reply({ok:true,value:{created:true}});
    }catch(error){reply({ok:false,error:{code:'mygoMCP/rejected',message:String(error?.message??error),details:{}}});}
  }}));
}
