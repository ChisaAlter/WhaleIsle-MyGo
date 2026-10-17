'use strict';
const gitFS=require('node:fs'),gitPath=require('node:path'),gitModule=require('node:module');
function loadOriginalGit(sources){const cache=new Map();function load(name){if(cache.has(name))return cache.get(name).exports;if(!Object.hasOwn(sources,name))throw Error('Original Git optional module unavailable: '+name);const m={exports:{}};cache.set(name,m);const localRequire=id=>{if(id.startsWith('node:')||['fs','path','os'].includes(id))return require(id);if(!id.startsWith('.'))throw Error('Unexpected original Git dependency');let next=gitPath.posix.normalize(gitPath.posix.join(gitPath.posix.dirname(name),id));if(!next.endsWith('.js'))next+='.js';return load(next);};new Function('require','module','exports','__filename','__dirname',sources[name])(localRequire,m,m.exports,name,gitPath.posix.dirname(name));return m.exports;}return {git:load('main/git.js'),authority:load('main/workspace-authority.js')};}
function createGitGate({backend,authority,authorize,review}){let generation=0,roots=new Set(),enabled=false;
 function revoke(){generation++;enabled=false;roots=new Set();backend.setWorkspaceAuthority(authority.createWorkspaceAuthority());}
 function configure(config){
  revoke();if(config.remoteEnabled!==true||config.remoteGitEnabled!==true)return;
  const requested=Array.isArray(config.remoteGitRoots)?config.remoteGitRoots:[config.remoteGitRoot];
  const granted=[];
  for(const raw of requested){
   if(typeof raw!=='string'||!raw)throw Error('Invalid Git workspace root');
   const canonical=gitFS.realpathSync(raw);
   if(!gitFS.statSync(canonical).isDirectory()||gitPath.parse(canonical).root===canonical||authority.isHighRiskWorkspaceRoot(canonical))throw Error('Unsafe Git workspace root');
   if(!granted.includes(canonical))granted.push(canonical);
  }
  if(!granted.length)return;
  roots=new Set(granted);backend.setWorkspaceAuthority(authority.createWorkspaceAuthority({workspace:granted[0],extraWorkspaces:granted.slice(1)}));enabled=true;
 }
 const names=['gitStatus','gitFetchForStatus','gitReadPullRequest','gitInit','gitDiff','gitCommit','gitPush','gitPull','gitCreateChangeRequest','gitPublishRepository','gitStatusEntries','gitBranchList','gitSwitchBranch','gitCreateBranch'];const git={};
 for(const name of names)git[name]=async(cwd,...args)=>{
  const epoch=generation;
  if(!enabled||typeof cwd!=='string')throw Error('Mobile Git is not authorized for this workspace');
  const root=gitFS.realpathSync(cwd);
  if(!roots.has(root))throw Error('Mobile Git is not authorized for this workspace');
  if(!await authorize(root,name)||epoch!==generation||!enabled||!roots.has(root))throw Error('Mobile Git desktop workspace permission expired');
  if(name==='gitCreateChangeRequest'&&args[0]?.generateOnly!==true&&(!args[0]?.preserveProvided||!args[0]?.title)){
   if(!review)throw Error('Review and confirm a change-request title/body before creating it');
   const value=await review(root);if(epoch!==generation||!enabled||!roots.has(root))throw Error('Mobile Git permission was revoked');return value;
  }
  const value=await backend[name](root,...args);if(epoch!==generation||!enabled||!roots.has(root))throw Error('Mobile Git permission was revoked');return value;
 };
 return {git,configure,revoke,get enabled(){return enabled;}};
}
if(typeof module!=='undefined')module.exports={createGitGate,loadOriginalGit};
