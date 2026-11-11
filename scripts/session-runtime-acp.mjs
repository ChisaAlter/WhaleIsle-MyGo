import fs from 'node:fs';
import path from 'node:path';
import {execFileSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {copyDesktopPackage} from './session-runtime-desktop-plugins.mjs';

export function applyAcpCompatibility(target) {
  const changes=[];
  // T3 ProviderInstanceId is a <=64-character slug, not a provider route.
  // presets.sanitizeId already bounds agent IDs to 48 slug characters.
  for(const name of ['lib/t3-auth.js','lib/t3-session.js']){
    const filename=path.join(target,name),source=fs.readFileSync(filename,'utf8');
    const before='`dsh:${agent.id}`',after='`dsh-${agent.id}`';
    const count=source.split(before).length-1;
    if(count){fs.writeFileSync(filename,source.split(before).join(after));changes.push({file:name,replacements:count});}
  }
  return {id:'t3-provider-instance-slug',changes};
}

export function drainAcpTurnEvents(target) {
  const name='lib/t3-session.js',filename=path.join(target,name);
  const source=fs.readFileSync(filename,'utf8');
  const nl=source.includes('\r\n')?'\r\n':'\n';
  const before=['.then((result) => {','        this.promptDone = true;'].join(nl);
  const after=['.then(async (result) => {','        await api.drainEvents();','        this.promptDone = true;'].join(nl);
  const count=source.split(before).length-1;
  if(count===1)fs.writeFileSync(filename,source.replace(before,after));
  else if(!source.includes(after))throw Error('ACP turn completion contract changed; inspect event-drain ordering');
  return {id:'t3-drain-turn-events',changes:count?[{file:name,replacements:count}]:[]};
}

export function cancelAcpTurns(target) {
  const helper='lib/mygo-turn-cancellation.js';
  fs.copyFileSync(fileURLToPath(new URL('./acp-turn-cancellation.mjs',import.meta.url)),path.join(target,helper));
  const changes=[];
  for(const [name,before,after] of [
    ['lib/t3-session.js','    this.promptPromise.catch(() => {});','    bindTurnCancellation(this, api);\n    this.promptPromise.catch(() => {});'],
    ['lib/index.js','      await conn.syncModelAndEffort(callOptions.model, callOptions.reasoningEffort, callOptions.signal);','      await conn.pendingCancellation;\n      await conn.syncModelAndEffort(callOptions.model, callOptions.reasoningEffort, callOptions.signal);'],
  ]){
    const filename=path.join(target,name);let source=fs.readFileSync(filename,'utf8');
    const nl=source.includes('\r\n')?'\r\n':'\n',replacement=after.replaceAll('\n',nl);
    if(!source.includes(replacement)){
      if(source.split(before).length!==2)throw Error(`ACP cancellation contract changed: ${name}`);
      source=source.replace(before,replacement);
      if(name==='lib/t3-session.js')source="import { bindTurnCancellation } from './mygo-turn-cancellation.js';"+nl+source;
      fs.writeFileSync(filename,source);changes.push({file:name,replacements:1});
    }
  }
  return {id:'t3-forward-host-cancel',helper,changes};
}

export function constrainAcpTaskPolicy(target) {
  const helper='lib/mygo-task-policy.js';
  fs.copyFileSync(fileURLToPath(new URL('./acp-task-policy.mjs',import.meta.url)),path.join(target,helper));
  const changes=[];
  const edits=[
    ['lib/t3-session.js', [
      [
        "      this.agentConfig.permissionPolicy === 'auto-allow'",
        "      effectiveAutoAllow(this)",
      ],
      [
        "        this._resolvePermission(params, runtimePolicy, requestContext),",
        "        this._resolvePermission(params, taskRuntimePolicy(this), requestContext),",
      ],
      [
        "      handleWriteTextFile: async (request) => {",
        "      handleWriteTextFile: async (request) => {\n        guardTaskClientMutation(this, 'fs/write_text_file');",
      ],
      [
        "          const disposition = t3.AcpClientPolicy.acpMcpToolApprovalElicitationDisposition(\n            runtimePolicy,\n            params,\n          );\n          if (disposition === 'allow') return { action: 'accept', content: {} };",
        "          const disposition = t3.AcpClientPolicy.acpMcpToolApprovalElicitationDisposition(\n            taskRuntimePolicy(this),\n            params,\n          );\n          if (disposition === 'allow' && !isTaskBound(this.dshAgent)) return { action: 'accept', content: {} };",
      ],
      [
        "      handlers.handleCreateTerminal = (request) =>\n        toPromise(this.terminals.create(request));",
        "      handlers.handleCreateTerminal = (request) => {\n        guardTaskClientMutation(this, 'terminal/create');\n        return toPromise(this.terminals.create(request));\n      };",
      ],
      [
        "    if (disposition === 'allow') return autoAllowedOutcome(params);",
        "    if (disposition === 'allow' && !isTaskBound(this.dshAgent)) return autoAllowedOutcome(params);",
      ],
      [
        "      const decisionDeferred = createDeferred();",
        "      const decisionDeferred = createDeferred();\n      if (entry.taskRejected) decisionDeferred.resolve(false);",
      ],
      [
        "    }\n    return autoAllowedOutcome(params);\n  }",
        "    }\n    return isTaskBound(this.dshAgent) ? { outcome: { outcome: 'cancelled' } } : autoAllowedOutcome(params);\n  }",
      ],
    ], "import { effectiveAutoAllow, taskRuntimePolicy, guardTaskClientMutation, isTaskBound } from './mygo-task-policy.js';"],
    ['lib/index.js', [
      [
        "      await conn.ensureReady(callOptions.signal);",
        "      conn.dshAgent = dshAgent;\n      await conn.ensureReady(callOptions.signal);\n      if (conn.wiring?.isAntigravity) await conn._applyAntigravitySessionMode();",
      ],
      [
        "          const approvalService = safeGetService(ctx, 'approval');\n          const targetAgent = exec.agent ?? tracked.dshAgent;",
        "          const approvalService = safeGetService(ctx, 'approval');\n          const targetAgent = exec.agent ?? tracked.dshAgent;\n          if (settleTaskPermission(exec, targetAgent, perm)) continue;",
      ],
      [
        "  const manager = createAcpAgentsManager(ctx, config);",
        "  const manager = createAcpAgentsManager(ctx, config);\n  registerTaskAcpRejector(ctx, manager.pendingToolExecutions);",
      ],
    ], "import { settleTaskPermission, registerTaskAcpRejector } from './mygo-task-policy.js';"],
  ];
  for (const [name, pairs, header] of edits) {
    const filename = path.join(target, name);
    let source = fs.readFileSync(filename, 'utf8');
    const nl = source.includes('\r\n') ? '\r\n' : '\n';
    let count = 0;
    for (const [rawBefore, rawAfter] of pairs) {
      const before = rawBefore.replaceAll('\n', nl), after = rawAfter.replaceAll('\n', nl);
      if (!source.includes(after)) {
        if (source.split(before).length !== 2) throw Error(`ACP task policy contract changed: ${name} (${rawBefore.split('\n')[0].trim()})`);
        source = source.replace(before, after);
        count++;
      }
    }
    if (!source.includes(header)) {
      source = header + nl + source;
    }
    if (count > 0) {
      fs.writeFileSync(filename, source);
      changes.push({file: name, replacements: count});
    }
  }
  return {id: 't3-native-task-policy', helper, changes};
}

export function verifyAcp(target) {
  const read=name=>JSON.parse(fs.readFileSync(path.join(target,name),'utf8'));
  const pkg=read('package.json'),provenance=read('mygo-acp-source.json');
  if(pkg.name!=='dsh-acp-agents'||provenance.package!==pkg.name||provenance.version!==pkg.version)throw Error('ACP package provenance mismatch');
  for(const name of provenance.files){
    const file=path.resolve(target,name),rel=path.relative(path.resolve(target),file);
    if(rel.startsWith('..')||path.isAbsolute(rel)||!fs.statSync(file).isFile())throw Error(`Missing ACP package file: ${name}`);
  }
  for(const name of ['lib/t3-auth.js','lib/t3-session.js']){
    if(fs.readFileSync(path.join(target,name),'utf8').includes('`dsh:${agent.id}`'))throw Error(`T3 ProviderInstanceId still contains colon: ${name}`);
  }
  if(!fs.readFileSync(path.join(target,'lib/t3-session.js'),'utf8').includes('await api.drainEvents();'))throw Error('ACP turn completion does not drain T3 events');
  for(const [name,required] of [
    ['lib/t3-session.js','bindTurnCancellation(this, api);'],
    ['lib/index.js','await conn.pendingCancellation;'],
    ['lib/mygo-turn-cancellation.js','api.cancel()'],
    ['lib/t3-session.js','taskRuntimePolicy(this)'],
    ['lib/t3-session.js','if (entry.taskRejected) decisionDeferred.resolve(false);'],
    ['lib/index.js','settleTaskPermission(exec, targetAgent, perm)'],
    ['lib/index.js','registerTaskAcpRejector(ctx, manager.pendingToolExecutions);'],
    ['lib/mygo-task-policy.js','whaleisle.nativeTaskTools'],
  ]){
    if(!fs.readFileSync(path.join(target,name),'utf8').includes(required))throw Error(`ACP runtime integration missing: ${name} (${required})`);
  }
  return {package:pkg.name,version:pkg.version,sourceHead:provenance.head,workingTree:provenance.workingTree,compatibility:provenance.compatibility};
}

export function stageAcp(sourceRoot, runtimeRoot) {
	const selectedSource=path.resolve(sourceRoot);
  const head=execFileSync('git',['-C',path.dirname(selectedSource),'rev-parse','HEAD'],{encoding:'utf8',windowsHide:true}).trim();
  const source=fs.realpathSync(sourceRoot), vendor=fs.realpathSync(path.join(runtimeRoot,'vendor'));
  if (fs.existsSync(path.join(vendor,'runtime-graph.json'))) throw Error('ACP needs an editable runtime stage');
  const target=path.join(vendor,'dsh-acp');
  if(fs.existsSync(target))throw Error('ACP target already exists; retain it and select a fresh stage');
  const pkg=JSON.parse(fs.readFileSync(path.join(source,'package.json'),'utf8'));
  if(pkg.name!=='dsh-acp-agents')throw Error('Expected the complete desktop ACP package');
  const files=new Map();
  function snapshot(root,relative=''){
    for(const entry of fs.readdirSync(path.join(root,relative),{withFileTypes:true})){
      const name=path.join(relative,entry.name);
      if(entry.isSymbolicLink())throw Error('ACP source snapshot must contain ordinary package files');
      if(entry.isDirectory())snapshot(root,name);else files.set(name,fs.readFileSync(path.join(root,name)));
    }
  }
  // Only the published package paths participate; tools/node_modules are build inputs.
  for(const relative of ['package.json',...pkg.files]){
    if(fs.statSync(path.join(source,relative)).isDirectory())snapshot(source,relative);
    else files.set(relative,fs.readFileSync(path.join(source,relative)));
  }
  copyDesktopPackage(source,target);
  for(const [relative,bytes] of files){
    if(!bytes.equals(fs.readFileSync(path.join(source,relative))) || !bytes.equals(fs.readFileSync(path.join(target,relative))))throw Error('ACP source changed during snapshot; retained destination must not be selected');
  }
  for(const relative of ['lib/index.js','lib/presets.js','lib/t3-runtime.js','lib/t3-session.js','lib/t3-auth.js','lib/mcp-server.js','lib/vendor/t3.cjs','lib/vendor/t3-mcp-bridge.cjs','client/client.js']){
    if(!files.has(path.normalize(relative)))throw Error(`Incomplete ACP package: ${relative}`);
  }
  const compatibility=[applyAcpCompatibility(target),drainAcpTurnEvents(target),cancelAcpTurns(target),constrainAcpTaskPolicy(target)];
  for(const generated of ['lib/mygo-turn-cancellation.js','lib/mygo-task-policy.js']){
    files.set(generated,fs.readFileSync(path.join(target,generated)));
  }
  const result={schemaVersion:1,source,selectedSource,head,workingTree:true,files:[...files.keys()].map(p=>p.split(path.sep).join('/')),package:pkg.name,version:pkg.version,compatibility};
  fs.writeFileSync(path.join(target,'mygo-acp-source.json'),JSON.stringify(result,null,2)+'\n',{flag:'wx'});
  return {target,head:result.head,files:files.size,verified:verifyAcp(target)};
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  const [source,runtime,extra]=process.argv.slice(2);if(!source||!runtime||extra)throw Error('Usage: session-runtime-acp.mjs <dsh-acp source> <editable runtime>');
  console.log(JSON.stringify(stageAcp(source,runtime),null,2));
}
