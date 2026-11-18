// Test-only Host consumer; copied into the acceptance profile, never bundled.
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {dirname, join} from 'node:path';
export const name = 'fixture-native-capability-probe';
export const inject = ['webServer', 'connection', 'tools', 'skills', 'agents', 'agentPresets'];
export function apply(ctx) {
  ctx.effect(() => ctx.webServer.register({kind: 'exact', path: '/api/fixtureNativeCapability/run', async handler(req, res) {
    const rejected = ctx.connection.requestRejection(req);
    if (rejected !== undefined) {res.writeHead(rejected);res.end();return;}
    let request;
    try {
      const chunks = [];
      for await (const chunk of req) chunks.push(chunk);
      request = JSON.parse(Buffer.concat(chunks).toString('utf8'));
      const input=request.payload.args.request;
      const agent=ctx.agents.get(input.sessionId);
      if(!agent)throw Error('Fixture session was not attached');
      const tool = await ctx.tools.execute({callId:'owned-native-fixture',name:'mcp__fixture__echo',arguments:{message:'ok'},signal:AbortSignal.timeout(10000)});
      if (tool.isError) throw Error(JSON.stringify(tool));
      const registry=ctx.agentPresets.serviceFor(agent,'skills')??ctx.skills;
      const skill = await registry.get('fixture-directory-skill', {scope:agent,cwd:input.cwd});
      if (!skill?.path) throw Error('Imported skill is unavailable');
      const script = await promisify(execFile)(process.execPath, [join(dirname(skill.path), 'scripts/task.js')], {cwd:dirname(skill.path),timeout:10000,windowsHide:true});
      res.writeHead(200, {'content-type':'application/json'});
      res.end(JSON.stringify({type:'server-response',rpcId:request.rpcId,result:{ok:true,value:{tool,script:script.stdout}}}));
    } catch (error) {
      res.writeHead(200, {'content-type':'application/json'});
      res.end(JSON.stringify({type:'server-response',rpcId:request?.rpcId??'invalid',result:{ok:false,error:{code:'fixture/failed',message:String(error)}}}));
    }
  }}));
}
