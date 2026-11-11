import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import {applyAcpCompatibility,drainAcpTurnEvents,cancelAcpTurns,constrainAcpTaskPolicy,verifyAcp} from './session-runtime-acp.mjs';
import {bindTurnCancellation} from './acp-turn-cancellation.mjs';
import {effectiveAutoAllow,taskRuntimePolicy,guardTaskClientMutation,settleTaskPermission,registerTaskAcpRejector} from './acp-task-policy.mjs';

test('ACP stage repairs internal T3 slugs, drains turn events, forwards cancel and binds task policy',()=>{
  const parent=fs.realpathSync(os.tmpdir());
  const root=fs.mkdtempSync(path.join(parent,'mygo-acp-package-'));
  try{
    fs.mkdirSync(path.join(root,'lib'));
    for(const name of ['t3-auth.js','t3-session.js'])fs.writeFileSync(path.join(root,'lib',name),'const instanceId=`dsh:${agent.id}`; const provider=`acp:${agent.id}`;');
    const patch=applyAcpCompatibility(root);
    assert.equal(patch.changes.length,2);
    for(const name of ['t3-auth.js','t3-session.js'])assert.equal(fs.readFileSync(path.join(root,'lib',name),'utf8'),'const instanceId=`dsh-${agent.id}`; const provider=`acp:${agent.id}`;');
    fs.appendFileSync(path.join(root,'lib/t3-session.js'),[
      '',
      '.then((result) => {',
      '        this.promptDone = true;',
      '    this.promptPromise.catch(() => {});',
      "      this.agentConfig.permissionPolicy === 'auto-allow'",
      '        this._resolvePermission(params, runtimePolicy, requestContext),',
      '      handleWriteTextFile: async (request) => {',
      '          const disposition = t3.AcpClientPolicy.acpMcpToolApprovalElicitationDisposition(',
      '            runtimePolicy,',
      '            params,',
      '          );',
      "          if (disposition === 'allow') return { action: 'accept', content: {} };",
      '      handlers.handleCreateTerminal = (request) =>',
      '        toPromise(this.terminals.create(request));',
      "    if (disposition === 'allow') return autoAllowedOutcome(params);",
      '      const decisionDeferred = createDeferred();',
      '  async _resolve() {',
      '    if (x) {',
      '    }',
      '    return autoAllowedOutcome(params);',
      '  }',
    ].join('\n'));
    assert.equal(drainAcpTurnEvents(root).changes.length,1);
    assert.ok(fs.readFileSync(path.join(root,'lib/t3-session.js'),'utf8').includes('await api.drainEvents();\n        this.promptDone'));
    fs.writeFileSync(path.join(root,'lib/index.js'),[
      '      await conn.ensureReady(callOptions.signal);',
      '      await conn.syncModelAndEffort(callOptions.model, callOptions.reasoningEffort, callOptions.signal);',
      "          const approvalService = safeGetService(ctx, 'approval');",
      '          const targetAgent = exec.agent ?? tracked.dshAgent;',
      '  const manager = createAcpAgentsManager(ctx, config);',
    ].join('\n'));
    assert.equal(cancelAcpTurns(root).changes.length,2);
    const policyPatch=constrainAcpTaskPolicy(root);
    assert.equal(policyPatch.changes.length,2);
    fs.writeFileSync(path.join(root,'package.json'),JSON.stringify({name:'dsh-acp-agents',version:'0.1.0'}));
    fs.writeFileSync(path.join(root,'mygo-acp-source.json'),JSON.stringify({package:'dsh-acp-agents',version:'0.1.0',files:['lib/t3-auth.js','lib/t3-session.js','lib/index.js','lib/mygo-turn-cancellation.js','lib/mygo-task-policy.js'],compatibility:[patch,policyPatch]}));
    assert.equal(verifyAcp(root).package,'dsh-acp-agents');
    fs.unlinkSync(path.join(root,'lib/t3-session.js'));
    assert.throws(()=>verifyAcp(root),/ENOENT/);
  }finally{
    assert.equal(path.dirname(fs.realpathSync(root)),parent);
    assert.ok(path.basename(root).startsWith('mygo-acp-package-'));
    fs.rmSync(root,{recursive:true,force:true});
  }
});

test('Host abort forwards cancel once and retains ownership until the ACP prompt settles',async()=>{
  let finish,cancels=0;
  const controller=new AbortController(),connection={activeTurn:null};
  const turn={signal:controller.signal,connection,promptDone:false,promptPromise:new Promise(resolve=>{finish=resolve})};
  connection.activeTurn=turn;
  bindTurnCancellation(turn,{cancel:async()=>{cancels++}});
  controller.abort();await Promise.resolve();
  assert.equal(cancels,1);assert.equal(connection.activeTurn,turn);
  const pending=connection.pendingCancellation;
  finish();await pending;
  assert.equal(connection.activeTurn,null);assert.equal(connection.pendingCancellation,null);
});

test('Native task turns suppress ACP auto-allow, gate client mutations and settle external permissions once',async()=>{
  const sym=Symbol.for('whaleisle.nativeTaskTools');
  const prevDesc=Object.getOwnPropertyDescriptor(globalThis,sym);
  let activeAgent=null,allowedExec=null,rejector=null;
  const consumed=new WeakSet();
  const fakeBridge={
    binding:agent=>(agent&&agent===activeAgent?{taskId:'t1',requestId:'r1',sessionId:'s1',boundSessionId:'s1'}:undefined),
    consumeAuthorization:exec=>{
      if(!exec||exec!==allowedExec||consumed.has(exec))return false;
      consumed.add(exec);return true;
    },
    setAcpRejector:fn=>{rejector=fn;return()=>{if(rejector===fn)rejector=null;};},
  };
  if(prevDesc)throw Error('unexpected global nativeTaskTools in isolated ACP unit test');
  Object.defineProperty(globalThis,sym,{value:fakeBridge,configurable:true});
  try{
    const taskAgent={id:'task-session'},ordinaryAgent={id:'ordinary-session'};
    activeAgent=taskAgent;
    const conn={cwd:'C:\\ws',agentConfig:{permissionPolicy:'auto-allow'},dshAgent:taskAgent};
    assert.equal(effectiveAutoAllow(conn),false);
    assert.deepEqual(taskRuntimePolicy(conn),{runtimeMode:'approval-required',cwd:'C:\\ws',approvalPolicy:'on-request',sandboxPolicy:{type:'workspaceWrite'}});
    assert.throws(()=>guardTaskClientMutation(conn,'fs/write_text_file'),/Task-governed/);

    // Ordinary turn restores configured auto-allow on the same connection.
    conn.dshAgent=ordinaryAgent;
    assert.equal(effectiveAutoAllow(conn),true);
    assert.equal(taskRuntimePolicy(conn).runtimeMode,'full-access');
    guardTaskClientMutation(conn,'fs/write_text_file');

    // Approved task execution authorizes exactly one ACP permission resolution.
    const exec={callId:'acp-1',agent:taskAgent};
    allowedExec=exec;
    const outcomes=[];
    assert.equal(settleTaskPermission(exec,taskAgent,{resolve:v=>outcomes.push(v)}),true);
    assert.equal(settleTaskPermission(exec,taskAgent,{resolve:v=>outcomes.push(v)}),true);
    assert.deepEqual(outcomes,[true,false]);
    assert.equal(settleTaskPermission({callId:'ord-1',agent:ordinaryAgent},ordinaryAgent,{resolve:()=>outcomes.push('unexpected')}),false);

    // Task gate denial rejects waiting ACP permission and marks entry rejected for late arrivals.
    let waiterWoke=false,permOutcome=null;
    const pending=new Map([['acp-deny',{entry:{pendingPermission:{resolve:v=>{permOutcome=v;}},waiters:[()=>{waiterWoke=true;}]}}]]);
    const cleanups=[];
    registerTaskAcpRejector({effect:fn=>{const u=fn();if(u)cleanups.push(u);}},pending);
    await rejector({callId:'acp-deny'});
    assert.equal(permOutcome,false);
    assert.equal(waiterWoke,true);
    assert.equal(pending.has('acp-deny'),false);
    for(const u of cleanups)u();
  }finally{
    delete globalThis[sym];
  }
});
