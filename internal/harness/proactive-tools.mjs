import { setTimeout as delay } from 'node:timers/promises';
import {toolSource,installDispatchGate} from './proactive-provenance.mjs';

const eventType = 'mygo-proactive/binding';

export async function desktopCall(action, payload, signal) {
  const base = process.env.DSH_DESKTOP_INSTALL_URL;
  const token = process.env.DSH_DESKTOP_INSTALL_TOKEN;
  if (!base || !token) throw new Error('Native task control is unavailable');
  const response = await fetch(`${base}/desktop/proactive`, {
    method: 'POST', headers: {authorization:`Bearer ${token}`,'content-type':'application/json'},
    body: JSON.stringify({action,payload}), signal: signal ? AbortSignal.any([signal, AbortSignal.timeout(15000)]) : AbortSignal.timeout(15000),
  });
  const result = await response.json();
  if (!response.ok || result?.ok === false) throw new Error(result?.error || `Native task control failed (${response.status})`);
  return result;
}

export function bindingFor(agent) {
  const session=agent?.session??agent;
  const binding=session?.snapshotEvents().findLast(event=>event.type===eventType)?.data;
  return binding?.boundSessionId===session?.id?binding:undefined;
}

// The durable marker identifies a native task request, not every future prompt
// in this Session. A later user turn uses the ordinary Host permissions. Steers
// in the task's current turn keep its gate, as do delegated child Sessions.
export function executionBindingFor(agent) {
  const binding = bindingFor(agent);
  if (!binding || binding.boundSessionId !== binding.sessionId) return binding;
  const session = agent?.session ?? agent;
  const events = session.snapshotEvents();
  const start = events.findLastIndex(event => event.type === 'turn/start');
  let hasRequest = false;
  for (let index = events.length - 1; index > start; index--) {
    const event = events[index];
    if (event.type !== 'user/message') continue;
    const requestId = event.data?.source?.rpcId;
    if (requestId === binding.requestId) return binding;
    if (requestId) hasRequest = true;
  }
  return start >= 0 && hasRequest ? undefined : binding;
}

export async function bindTask(agent, payload) {
  const call = {taskId:payload.taskId,requestId:payload.requestId,sessionId:agent.id};
  const binding = await desktopCall('tool/bind',{call});
  const previous = bindingFor(agent);
  if (previous?.taskId!==binding.taskId || previous?.requestId!==binding.requestId) {
    agent.session.append(eventType,{taskId:binding.taskId,requestId:binding.requestId,sessionId:agent.id,boundSessionId:agent.id},{ignorable:true});
  }
  return binding;
}

export function installToolGate(ctx) {
  ctx.on('session/created',session=>{
    if (session.header.origin!=='subagent'||bindingFor(session)) return;
    const parent=ctx.sessions.get(session.header.parentSession);
    const owner=executionBindingFor(parent);
    if (owner) session.append(eventType,{...owner,boundSessionId:session.id},{ignorable:true});
  });
  ctx.effect(()=>installDispatchGate({binding:executionBindingFor,execute:async (exec,next)=>{
    const binding=executionBindingFor(exec.agent);
    if (!binding) return next();
    const call=JSON.parse(JSON.stringify({taskId:binding.taskId,requestId:binding.requestId,sessionId:binding.sessionId,callId:`${exec.agent.id}/${exec.callId}`,name:exec.name,source:toolSource(ctx.tools,exec),arguments:exec.arguments,cwd:exec.agent.session.header.cwd}));
    let lease;
    try {
      for (;;) {
        exec.signal.throwIfAborted();
        const decision=await desktopCall('tool/prepare',{call},exec.signal);
        if (decision.status==='pending'||decision.status==='busy') {await delay(400,undefined,{signal:exec.signal});continue;}
        if (decision.status!=='ready') throw new Error(decision.reason || `Task tool operation is ${decision.status}; it will not be repeated`);
        const claimed=await desktopCall('tool/claim',{call},exec.signal);
        if (claimed.status!=='allow'||!claimed.lease) throw new Error('This exact tool operation is no longer authorized');
        lease=claimed.lease;
        exec.signal.throwIfAborted();
        const result=await next();
        const outcome=result?.result??result;
        const status=exec.signal.aborted?'unknown':outcome?.isError?'failed':'succeeded';
        const error=outcome?.isError?String(outcome?.error?.message || 'Tool returned an error'):'';
        await desktopCall('tool/finish',{call,lease,status,error});
        return result;
      }
    } catch(error) {
      // A lost result is not evidence that an external effect did not happen.
      // Reporting it never grants permission to retry the effect.
      try {
        if (lease) await desktopCall('tool/finish',{call,lease,status:'unknown',error:String(error?.message??error)});
        else await desktopCall('tool/abandon',{call});
      } catch {}
      throw error;
    }
  }}));
}
