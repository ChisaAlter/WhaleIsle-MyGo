import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import {packageLocalRecovery} from './package-local-recovery.mjs';

async function fixture(t) {
  const parent=await fs.realpath(os.tmpdir()),root=await fs.mkdtemp(path.join(parent,'portable-package-test-'));
  t.after(async()=>{
    const relative=path.relative(parent,root);
    assert.ok(relative.startsWith('portable-package-test-')&&!relative.includes(path.sep)&&!path.isAbsolute(relative));
    await fs.rm(root,{recursive:true,force:true});
  });
  const executable=Buffer.alloc(80);executable.write('MZ');executable.writeUInt32LE(64,60);executable.write('PE\0\0',64);
  const exe=path.join(root,'built.exe');await fs.writeFile(exe,executable);
  const stage=path.join(root,'stage');await fs.mkdir(stage);
  return {root,stage,exe,preview:exe,out:path.join(root,'fresh-package')};
}

test('missing explicit native/static dependencies leave the output absent',async t=>{
  const f=await fixture(t);
  await assert.rejects(packageLocalRecovery(f),/Missing explicit package inputs: terminal, pet, components/);
  await assert.rejects(fs.lstat(f.out),{code:'ENOENT'});
});
test('an existing package is preserved without overwrites or deletion',async t=>{
  const f=await fixture(t);await fs.mkdir(f.out);
  await fs.writeFile(path.join(f.out,'keep.txt'),'existing');
  await assert.rejects(packageLocalRecovery(f),/must be fresh/);
  assert.equal(await fs.readFile(path.join(f.out,'keep.txt'),'utf8'),'existing');
});
test('a package cannot contain or overwrite its runtime input',async t=>{
  const f=await fixture(t);f.out=path.join(f.stage,'package');
  await assert.rejects(packageLocalRecovery(f),/overlaps an input/);
  await assert.rejects(fs.lstat(f.out),{code:'ENOENT'});
});
test('terminal bytes must match the actual binding pin before copying',async t=>{
  const f=await fixture(t);f.terminal=path.join(f.root,'untrusted.dll');await fs.writeFile(f.terminal,'wrong ABI');
  await assert.rejects(packageLocalRecovery(f),/Native terminal SHA256 mismatch/);
  await assert.rejects(fs.lstat(f.out),{code:'ENOENT'});
});
test('an unbuilt entry is rejected before creating a portable marker',async t=>{
  const f=await fixture(t);await fs.writeFile(f.exe,'placeholder');
  await assert.rejects(packageLocalRecovery(f),/not a built Windows executable/);
  await assert.rejects(fs.lstat(f.out),{code:'ENOENT'});
});
