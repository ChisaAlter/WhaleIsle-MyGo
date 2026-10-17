'use strict';
const assert=require('node:assert/strict');
const {createGitGate}=require('./git-gate.cjs');
async function main(){
 let created=0,reviewed=0;
 const base={backend:{setWorkspaceAuthority(){},async gitCreateChangeRequest(){created++;return {ok:true};}},authority:{createWorkspaceAuthority(){return {};},isHighRiskWorkspaceRoot(){return false;}},authorize:async()=>true};
 const configure=g=>g.configure({remoteEnabled:true,remoteGitEnabled:true,remoteGitRoot:__dirname});
 let g=createGitGate(base);configure(g);
 await assert.rejects(g.git.gitCreateChangeRequest(__dirname,{message:'native commit'}),/Review and confirm/);assert.equal(created,0);
 g=createGitGate({...base,review:async()=>{reviewed++;return {ok:true,status:'created'};}});configure(g);
 assert.equal((await g.git.gitCreateChangeRequest(__dirname,{message:'native commit'})).status,'created');assert.equal(reviewed,1);assert.equal(created,0);
 await g.git.gitCreateChangeRequest(__dirname,{preserveProvided:true,title:'Reviewed'});assert.equal(created,1);assert.equal(reviewed,1);
 await g.git.gitCreateChangeRequest(__dirname,{generateOnly:true});assert.equal(created,2);assert.equal(reviewed,1);
 g=createGitGate({...base,review:async()=>{g.revoke();return {ok:true};}});configure(g);
 await assert.rejects(g.git.gitCreateChangeRequest(__dirname,{}),/revoked/);
 console.log('phone PR review gate fixture passed');
}
main().catch(e=>{console.error(e);process.exitCode=1});
