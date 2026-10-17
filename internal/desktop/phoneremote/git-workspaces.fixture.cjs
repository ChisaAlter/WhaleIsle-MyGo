'use strict';
const assert=require('node:assert/strict'),fs=require('node:fs'),os=require('node:os'),path=require('node:path');
const {createGitGate}=require('./git-gate.cjs');
async function main(){
 const base=fs.mkdtempSync(path.join(os.tmpdir(),'phone-workspaces-'));
 try{
  const a=path.join(base,'a'),b=path.join(base,'b'),outside=path.join(base,'outside');for(const p of [a,b,outside])fs.mkdirSync(p);
  let allowed=new Set([a,b]),seen=[],scope;
  const g=createGitGate({backend:{setWorkspaceAuthority(v){scope=v;},async gitStatus(root){seen.push(root);return root;}},authority:{createWorkspaceAuthority(v){return v;},isHighRiskWorkspaceRoot(){return false;}},authorize:async root=>allowed.has(root)});
  g.configure({remoteEnabled:true,remoteGitEnabled:true,remoteGitRoot:a});
  await assert.rejects(g.git.gitStatus(b),/not authorized/);assert.deepEqual(seen,[]);
  g.configure({remoteEnabled:true,remoteGitEnabled:true,remoteGitRoots:[a,b]});
  assert.deepEqual(scope,{workspace:a,extraWorkspaces:[b]});
  assert.equal(await g.git.gitStatus(b),b);await assert.rejects(g.git.gitStatus(outside),/not authorized/);
  allowed.delete(b);await assert.rejects(g.git.gitStatus(b),/expired/);
  g.revoke();await assert.rejects(g.git.gitStatus(a),/not authorized/);
  console.log('phone selected workspace gate fixture passed');
 }finally{fs.rmSync(base,{recursive:true,force:true});}
}
main().catch(e=>{console.error(e);process.exitCode=1});
