// A downward-only permission bridge for native L1 task Sessions. Normal task
// admission, model requests, cancellation and history use published Session APIs.
export const name = 'mygo-proactive-readonly';
import { bindTask, installToolGate } from './tools.mjs';
import {deliveryRoutes,sendDelivery} from './delivery.mjs';
export const inject = ['webServer', 'connection', 'agents', 'approval', 'tools', 'sessions'];

export function apply(ctx) {
  installToolGate(ctx);
  ctx.effect(() => ctx.webServer.register({
    kind: 'prefix', path: '/native-proactive',
    async handler(req, res) {
      const rejection = ctx.connection.requestRejection(req);
      if (rejection !== undefined) { res.writeHead(rejection); res.end('rejected'); return; }
      const path = new URL(req.url ?? '/', 'http://dsh.internal').pathname;
      const method=path.slice('/native-proactive/'.length);
      if (req.method !== 'POST' || !['read-only','bind','delivery/routes','delivery/send'].includes(method)) { res.writeHead(404); res.end('not found'); return; }
      let request;
      const reply = result => {
        res.writeHead(200, {'content-type':'application/json'});
        res.end(JSON.stringify({type:'server-response', rpcId:request?.rpcId ?? 'invalid', result}));
      };
      try {
        let size = 0;
        const chunks = [];
        for await (const chunk of req) {
          size += chunk.length;
          if (size > 65536) throw new Error('request too large');
          chunks.push(chunk);
        }
        request = JSON.parse(Buffer.concat(chunks).toString('utf8'));
        if (request?.type !== 'client-request' || typeof request.rpcId !== 'string' || request.method !== method) throw new Error('invalid request');
        if (method==='delivery/routes') {reply({ok:true,value:await deliveryRoutes(ctx)});return;}
        if (method==='delivery/send') {
          if (typeof request.payload?.approvalId !== 'string') throw Error('approval ID required');
          reply({ok:true,value:await sendDelivery(ctx,request.payload.approvalId)});return;
        }
        if (typeof request.payload?.sessionId !== 'string') throw Error('session ID required');
        const agent = ctx.agents.get(request.payload.sessionId);
        if (!agent) throw new Error('session is not attached');
        if (method==='bind') {reply({ok:true,value:await bindTask(agent,request.payload)});return;}
        if (agent.status !== 'idle' || agent.inbox.nextTurn.length || agent.inbox.nextStep.length) throw new Error('session must be idle before selecting permissions');
        agent.session.append('sandbox/mode', {mode:'read-only'});
        ctx.approval.setPolicy(agent, 'ask');
        reply({ok:true, value:{sandbox:'read-only', approval:'ask'}});
      } catch (error) {
        reply({ok:false, error:{code:'native-proactive/rejected', message:String(error?.message ?? error), details:{}}});
      }
    }
  }));
}
