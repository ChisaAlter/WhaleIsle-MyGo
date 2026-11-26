import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
const root=fileURLToPath(new URL('../',import.meta.url));
const finalize=path.join(root,'scripts/finalize-windows-release.ps1');
const psQuote=s=>"'"+s.replaceAll("'","''")+"'";
function run(script){
 const res=spawnSync('pwsh',['-NoProfile','-Command',script],{cwd:root,encoding:'utf8',windowsHide:true});
 if(res.error&&res.error.code==='ENOENT'){
  return spawnSync('powershell.exe',['-NoProfile','-ExecutionPolicy','Bypass','-Command',script],{cwd:root,encoding:'utf8',windowsHide:true});
 }
 return res;
}

test('explicit signature requirement retains unsigned candidate without metadata', {skip:process.platform!=='win32'},()=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'whale-finalize-'));
 try{
  const installer=path.join(dir,'candidate.exe');fs.writeFileSync(installer,'inert candidate');
  const result=run(`& ${psQuote(finalize)} -Installer ${psQuote(installer)} -RequireSignature`);
  assert.notEqual(result.status,0);assert.match(result.stderr,/valid Authenticode signature/);
  assert.equal(fs.readFileSync(installer,'utf8'),'inert candidate');
  assert.equal(fs.existsSync(installer+'.blockmap.json'),false);
  assert.equal(fs.existsSync(path.join(dir,'SHA256SUMS.txt')),false);
 }finally{fs.rmSync(dir,{recursive:true,force:true});}
});

test('default non-development release emits exact unsigned metadata', {skip:process.platform!=='win32'},()=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'whale-finalize-'));
 try{
  const installer=path.join(dir,'candidate.exe');fs.writeFileSync(installer,Buffer.alloc(70000,65));
  const result=run(`Remove-Item Env:WHALEISLE_UPDATE_REQUIRE_SIGNATURE,Env:WHALEISLE_UPDATE_PUBLISHER,Env:WHALEISLE_SIGNING_SCRIPT -ErrorAction SilentlyContinue; & ${psQuote(finalize)} -Installer ${psQuote(installer)}`);
  assert.equal(result.status,0,result.stdout+result.stderr);assert.match(result.stdout,/unsigned release/);
  const bytes=fs.readFileSync(installer),map=JSON.parse(fs.readFileSync(installer+'.blockmap.json'));
  assert.equal(map.fileSize,bytes.length);
  for(const block of map.blocks) assert.equal(block.sha256,crypto.createHash('sha256').update(bytes.subarray(block.offset,block.offset+block.size)).digest('hex'));
  assert.equal(fs.readFileSync(path.join(dir,'SHA256SUMS.txt'),'utf8').trim(),crypto.createHash('sha256').update(bytes).digest('hex')+'  candidate.exe');
 }finally{fs.rmSync(dir,{recursive:true,force:true});}
});

test('final metadata describes bytes after signer and repeated finalization', {skip:process.platform!=='win32'},()=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'whale-finalize-'));
 try{
  const installer=path.join(dir,'candidate.exe'),signer=path.join(dir,'signer.ps1');
  fs.writeFileSync(installer,Buffer.alloc(70000,65));
  fs.writeFileSync(signer,"param([string]$FilePath)\n[IO.File]::AppendAllText($FilePath,'signed-fixture')\n");
  // Development mode exercises byte ordering with an inert signer; it does not certify Authenticode.
  for(let i=0;i<2;i++){
   const result=run(`& ${psQuote(finalize)} -Installer ${psQuote(installer)} -SigningScript ${psQuote(signer)} -Development`);
   assert.equal(result.status,0,result.stdout+result.stderr);
   const bytes=fs.readFileSync(installer),map=JSON.parse(fs.readFileSync(installer+'.blockmap.json'));
   assert.equal(map.fileSize,bytes.length);
   for(const block of map.blocks) assert.equal(block.sha256,crypto.createHash('sha256').update(bytes.subarray(block.offset,block.offset+block.size)).digest('hex'));
   assert.equal(fs.readFileSync(path.join(dir,'SHA256SUMS.txt'),'utf8').trim(),crypto.createHash('sha256').update(bytes).digest('hex')+'  candidate.exe');
  }
  assert.deepEqual(fs.readdirSync(dir).sort(),['SHA256SUMS.txt','candidate.exe','candidate.exe.blockmap.json','signer.ps1']);
 }finally{fs.rmSync(dir,{recursive:true,force:true});}
});

test('RDN publisher matching rejects forged substrings/cross-RDN subjects and package.ps1 forwards ExpectedPublisher', {skip:process.platform!=='win32'},()=>{
 const cases = [
  ['CN=WhaleIsle, O=WhaleIsle Inc, C=US', 'WhaleIsle', true],
  ['CN=WhaleIsle, O=WhaleIsle Inc, C=US', 'CN=WhaleIsle', true],
  ['CN=WhaleIsle, O=WhaleIsle Inc, C=US', 'O=WhaleIsle Inc', true],
  ['CN="WhaleIsle, Inc.", O=Test', 'WhaleIsle, Inc.', true],
  ['OU=WhaleIsle, CN=Other Corp', 'WhaleIsle', false],
  ['CN=NotWhaleIsleCorp, O=Other', 'WhaleIsle', false],
  ['O=WhaleIsle, CN=Other', 'CN=WhaleIsle', false],
  ['L=WhaleIsle, CN=Attacker', 'WhaleIsle', false],
 ];
 const loadAndRun = `
$tokens = $null; $errs = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile(${psQuote(finalize)}, [ref]$tokens, [ref]$errs)
if ($errs.Count -gt 0) { throw "Parse errors: $($errs -join '; ')" }
$fns = $ast.FindAll({ param($n) $n -is [System.Management.Automation.Language.FunctionDefinitionAst] }, $false)
foreach ($fn in $fns) { Invoke-Expression $fn.Extent.Text }
$cases = ConvertFrom-Json ${psQuote(JSON.stringify(cases))}
foreach ($c in $cases) {
  $got = Test-PublisherMatches -Publisher $c[0] -Expected $c[1]
  if ([bool]$got -ne [bool]$c[2]) {
    throw "Mismatch for subject='$($c[0])' expected='$($c[1])': got=$got want=$($c[2])"
  }
}
Write-Output 'RDN_OK'
`;
 const result = run(loadAndRun);
 assert.equal(result.status, 0, result.stdout + result.stderr);
 assert.match(result.stdout, /RDN_OK/);

 const pkgSrc = fs.readFileSync(path.join(root, 'scripts/package.ps1'), 'utf8');
 assert.match(pkgSrc, /\[string\]\$ExpectedPublisher\s*=\s*\$env:WHALEISLE_UPDATE_PUBLISHER/);
 assert.match(pkgSrc, /finalize-windows-release\.ps1'.*-ExpectedPublisher\s+\$ExpectedPublisher/);
});
