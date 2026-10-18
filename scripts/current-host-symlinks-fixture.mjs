// Prepare/copy regression only. No dependency install, build, Host, or service launch.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {prepare} from './current-host.mjs';

const sourceArg=process.argv[2] || process.env.WHALEISLE_CURRENT_ELECTRON_SOURCE;
if(!sourceArg) throw new Error('Pass a fixed official source checkout containing relative symlinks.');
const source=fs.realpathSync(sourceArg);
const revision=spawnSync('git',['-C',source,'rev-parse','HEAD'],{encoding:'utf8'});
assert.equal(revision.status,0,revision.stderr);
const root=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-current-host-symlinks-'));
const work=path.join(root,'prepared'), moved=path.join(root,'relocated');
const links=[];
const inside=(root,p)=>p===root || p.startsWith(root+path.sep);
function collect(directory,destination) {
 for(const entry of fs.readdirSync(directory,{withFileTypes:true})) {
  if(['.git','node_modules'].includes(entry.name)) continue;
  const original=path.join(directory,entry.name), relative=path.join(destination,entry.name);
  if(entry.isSymbolicLink()) {
   const text=fs.readlinkSync(original);
   assert.equal(path.isAbsolute(text),false,`Original link must be relative: ${original}`);
   assert.ok(inside(source,fs.realpathSync(original)),`Original link escapes source: ${original}`);
   links.push({relative,text});
  } else if(entry.isDirectory()) collect(original,relative);
 }
}
try {
 for(const name of ['deepseek-harness','dsh-project']) collect(path.join(source,'vendor',name),name);
 if(fs.existsSync(path.join(source,'src'))) collect(path.join(source,'src'),'desktop-source/src');
 assert.ok(links.length>0,'Fixture requires original relative links to exercise the copy regression.');
 prepare(source,work,revision.stdout.trim());
 function check(directory) {
  for(const {relative,text} of links) {
   const copied=path.join(directory,relative);
   assert.equal(fs.readlinkSync(copied),text,`Copy changed symlink text: ${relative}`);
   assert.ok(inside(directory,fs.realpathSync(copied)),`Link does not resolve within isolated copy: ${relative}`);
  }
 }
 check(work);
 fs.renameSync(work,moved);
 check(moved);
 console.log(`Prepare symlink fixture OK: ${links.length} original relative links preserved and resolve within the relocated directory.`);
} finally {fs.rmSync(root,{recursive:true,force:true});}
