import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import {pathToFileURL} from 'node:url';
import {Readable} from 'node:stream';

test('MCP import rejects a competing identity under the real Host file lock', async () => {
  const work=await fs.mkdtemp(path.join(os.tmpdir(),'mygo-mcp-import-'));
  try {
    const base=path.resolve('runtime-rc4/vendor/deepseek-harness/packages/mcp/mcp-servers-file/lib/types');
    const {McpServersFile}=await import(pathToFileURL(path.join(base,'service.js')).href);
    const {parseDocument}=await import(pathToFileURL(path.join(base,'document.js')).href);
    let source=await fs.readFile(new URL('./native-mcp-import.mjs',import.meta.url),'utf8');
    source=source.replace("'__MYGO_MCP_DOCUMENT__'",JSON.stringify(pathToFileURL(path.join(base,'document.js')).href));
    const module=await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
    const filename=path.join(work,'mcp-servers.yaml');
    const createHandler=()=>{
      // Use the real Host mutation, read, serialization and cross-process lock;
      // disabled records avoid starting external MCP children in this unit test.
      const owner=Object.create(McpServersFile.prototype);
      Object.assign(owner,{spec:{filename},ready:Promise.resolve(),operations:Promise.resolve(),closed:false,children:new Map(),document:{servers:[]},ctx:{logger:{warn(){}}}});
      let handler;
      module.apply({effect:fn=>fn(),connection:{requestRejection:()=>undefined},mcpServersFile:owner,webServer:{register:route=>{assert.equal(route.path,'/api/mygoMCP');handler=route.handler;return()=>{}}}});
      return handler;
    };
    const invoke=async(handler,command)=>{
      const spec={id:'shared',serverName:'shared',transport:'stdio',command,enabled:false};
      const req=Readable.from([Buffer.from(JSON.stringify({type:'client-request',rpcId:command,method:'mygoMCP/import',payload:{args:{request:{spec}}}}))]);
      req.method='POST';req.url='/api/mygoMCP/import';
      const res={writeHead(){},end(body){this.body=JSON.parse(body)}};
      await handler(req,res);return res.body;
    };
    const results=await Promise.all([invoke(createHandler(),'first'),invoke(createHandler(),'second')]);
    assert.equal(results.filter(row=>row.result.ok).length,1);
    const winner=results.find(row=>row.result.ok).rpcId;
    const stored=parseDocument(await fs.readFile(filename,'utf8'));
    assert.equal(stored.servers.length,1);
    assert.equal(stored.servers[0].command,winner);
    assert.match(results.find(row=>!row.result.ok).result.error.message,/already exists/);
  } finally {
    assert.equal(path.dirname(work),os.tmpdir());
    assert.ok(path.basename(work).startsWith('mygo-mcp-import-'));
    await fs.rm(work,{recursive:true,force:true});
  }
});
