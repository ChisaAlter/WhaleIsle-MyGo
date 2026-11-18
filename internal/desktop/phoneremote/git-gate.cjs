'use strict';
const gitFS=require('node:fs'),gitPath=require('node:path'),gitModule=require('node:module');
function loadOriginalGit(sources){const cache=new Map();function load(name){if(cache.has(name))return cache.get(name).exports;if(!Object.hasOwn(sources,name))throw Error('Original Git optional module unavailable: '+name);const m={exports:{}};cache.set(name,m);const localRequire=id=>{if(id.startsWith('node:')||['fs','path','os'].includes(id))return require(id);if(!id.startsWith('.'))throw Error('Unexpected original Git dependency');let next=gitPath.posix.normalize(gitPath.posix.join(gitPath.posix.dirname(name),id));if(!next.endsWith('.js'))next+='.js';return load(next);};new Function('require','module','exports','__filename','__dirname',sources[name])(localRequire,m,m.exports,name,gitPath.posix.dirname(name));return m.exports;}return {git:load('main/git.js'),authority:load('main/workspace-authority.js')};}
function createGitGate({backend,authority,authorize,review,acquireWrite}){let generation=0,roots=new Set(),enabled=false;
 const runningWrites=new Set();
 async function drainWrites(){await Promise.allSettled([...runningWrites]);}
 function revoke(){generation++;enabled=false;roots=new Set();backend.setWorkspaceAuthority(authority.createWorkspaceAuthority());}
 function configure(config){
  revoke();if(config.remoteEnabled!==true||config.remoteGitEnabled!==true)return;
  const requested=Array.isArray(config.remoteGitRoots)?config.remoteGitRoots:[config.remoteGitRoot];
  const granted=[];
  for(const raw of requested){
   if(typeof raw!=='string'||!raw)throw Error('Invalid Git workspace root');
   const realpath=gitFS.realpathSync.native||gitFS.realpathSync;
   const canonical=realpath(raw);
   if(!gitFS.statSync(canonical).isDirectory()||gitPath.parse(canonical).root===canonical||authority.hasGitDirSegment?.(canonical)||authority.isHighRiskWorkspaceRoot(canonical))throw Error('Unsafe Git workspace root');
   if(!granted.includes(canonical))granted.push(canonical);
  }
  if(!granted.length)return;
  roots=new Set(granted);backend.setWorkspaceAuthority(authority.createWorkspaceAuthority({workspace:granted[0],extraWorkspaces:granted.slice(1)}));enabled=true;
 }
 const names=['gitStatus','gitFetchForStatus','gitReadPullRequest','gitInit','gitDiff','gitCommit','gitPush','gitPull','gitCreateChangeRequest','gitPublishRepository','gitStatusEntries','gitBranchList','gitSwitchBranch','gitCreateBranch'];const git={};
 for(const name of names)git[name]=async(cwd,...args)=>{
  const epoch=generation;
  if(!enabled||typeof cwd!=='string')throw Error('Mobile Git is not authorized for this workspace');
  const realpath=gitFS.realpathSync.native||gitFS.realpathSync;
  const root=realpath(cwd);
  if(!roots.has(root))throw Error('Mobile Git is not authorized for this workspace');
  if(!await authorize(root,name)||epoch!==generation||!enabled||!roots.has(root))throw Error('Mobile Git desktop workspace permission expired');
  if(name==='gitCreateChangeRequest'&&args[0]?.generateOnly!==true&&(!args[0]?.preserveProvided||!args[0]?.title)){
   if(!review)throw Error('Review and confirm a change-request title/body before creating it');
   const value=await review(root);if(epoch!==generation||!enabled||!roots.has(root))throw Error('Mobile Git permission was revoked');return value;
  }
  const writes = ['gitFetchForStatus','gitInit','gitCommit','gitPush','gitPull','gitPublishRepository','gitSwitchBranch','gitCreateBranch'].includes(name) || (name==='gitCreateChangeRequest'&&args[0]?.generateOnly!==true);
  const release = writes && acquireWrite ? await acquireWrite(root,name) : null;
  try {
   if(epoch!==generation||!enabled||!roots.has(root))throw Error('Mobile Git permission was revoked');
   // Once admitted, an actual write owns its result even if the view or Git
   // grant closes meanwhile. The owner is released only after backend exit.
   const operation=Promise.resolve().then(()=>backend[name](root,...args));
   if(writes)runningWrites.add(operation);
   let value;
   try{value=await operation;}finally{if(writes)runningWrites.delete(operation);}
   if(!writes&&(epoch!==generation||!enabled||!roots.has(root)))throw Error('Mobile Git permission was revoked');
   return value;
  } finally { if(release)release(); }
 };
 return {git,configure,revoke,drainWrites,get enabled(){return enabled;}};
}
if(typeof module!=='undefined')module.exports={createGitGate,loadOriginalGit};
