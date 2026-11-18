'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const EventEmitter = require('node:events');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const childProcess = require('node:child_process');

const { setDesktopDshHome, clearDesktopDshHome } = require('./git-source/shared/dsh-home');
const authorityMod = require('./git-source/main/workspace-authority');
const { createGitGate, loadOriginalGit } = require('./git-gate.cjs');

test('Git timeout waits for process close before releasing its result', async () => {
  const vm = require('node:vm');
  const child = new EventEmitter();
  child.stdout = new EventEmitter();child.stderr = new EventEmitter();
  let killed=0, expire;
  child.kill=()=>{killed++;};
  const module={exports:{}};
  vm.runInNewContext(fs.readFileSync(path.join(__dirname,'git-source/main/git-exec.js'),'utf8'),{
    module,exports:module.exports,Buffer,process:{platform:'linux',env:{}},
    setTimeout:fn=>{expire=fn;return 1;},clearTimeout(){},
    require:id=>id==='node:child_process'?{spawn:()=>child}:id==='./workspace-authority'?authorityMod:require(id),
  });
  let completed=false;
  const result=module.exports.run('git',['commit'],'/fixture',{timeoutMs:1}).then(value=>{completed=true;return value;});
  expire();await Promise.resolve();
  assert.equal(killed,1);assert.equal(completed,false);
  child.emit('close',null);
  assert.equal((await result).timedOut,true);
});

test('admitted Git writes retain result and ownership until backend completion after revocation', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'mygo-git-owner-'));
  let finish, entered, releases = 0;
  const started = new Promise(resolve => { entered = resolve; });
  const pending = new Promise(resolve => { finish = resolve; });
  const gate = createGitGate({
    backend: {setWorkspaceAuthority(){}, async gitCommit(){entered();return pending;}},
    authority: authorityMod,
    authorize: async () => true,
    acquireWrite: async (cwd, action) => {assert.equal(cwd, fs.realpathSync(root));assert.equal(action,'gitCommit');return () => releases++;},
  });
  try {
    gate.configure({remoteEnabled:true,remoteGitEnabled:true,remoteGitRoots:[root]});
    const result = gate.git.gitCommit(root, 'owned change');
    await started;
    gate.revoke();
    assert.equal(releases,0,'view revocation cannot release a running write');
    let drained=false;
    const drain=gate.drainWrites().then(()=>{drained=true;});
    await Promise.resolve();assert.equal(drained,false,'runtime stop must wait for admitted writes');
    finish({ok:true,commit:'fixture'});
    assert.deepEqual(await result,{ok:true,commit:'fixture'});
    await drain;
    assert.equal(releases,1);
  } finally { fs.rmSync(root,{recursive:true,force:true}); }
});

test('revocation during queued Git ownership prevents the backend write', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'mygo-git-queued-'));
  let admit, waiting, writes=0, releases=0;
  const queued = new Promise(resolve => {waiting=resolve;});
  const gate=createGitGate({backend:{setWorkspaceAuthority(){},async gitPull(){writes++;}},authority:authorityMod,authorize:async()=>true,
    acquireWrite:async()=>{waiting();return new Promise(resolve=>{admit=()=>resolve(()=>releases++);});}});
  try {
    gate.configure({remoteEnabled:true,remoteGitEnabled:true,remoteGitRoots:[root]});
    const result=gate.git.gitPull(root);await queued;gate.revoke();admit();
    await assert.rejects(result,/revoked/);assert.equal(writes,0);assert.equal(releases,1);
  } finally {fs.rmSync(root,{recursive:true,force:true});}
});

test('gitPublishRepository allowlists remote URLs, checks repo names, and inserts --', async () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'git-publish-sec-'));
  const origSpawn = childProcess.spawn;
  const origExecFile = childProcess.execFile;
  const invocations = [];

  function mockChild(args, cmd) {
    invocations.push([cmd, ...args]);
    const child = new EventEmitter();
    child.stdout = new EventEmitter();
    child.stderr = new EventEmitter();
    child.kill = () => {};
    process.nextTick(() => {
      if (cmd === 'git' && args[0] === 'remote') {
        if (args[1] === '-v' || args.length === 1) {
          child.emit('close', 0, null);
          return;
        }
      }
      if (cmd === 'git' && args[0] === 'rev-parse') {
        child.emit('close', 1, null); // no commits yet -> no push
        return;
      }
      if (cmd === 'gh' && args[0] === 'repo') {
        child.stdout.emit('data', Buffer.from('https://github.com/org/demo\n'));
      }
      child.emit('close', 0, null);
    });
    return child;
  }

  childProcess.spawn = (cmd, args) => mockChild(args, cmd);
  childProcess.execFile = (file, args, opts, cb) => {
    const callback = typeof opts === 'function' ? opts : cb;
    if (callback) callback(null, 'git version 2.40.0\n', '');
    return new EventEmitter();
  };

  delete require.cache[require.resolve('./git-source/main/git-exec')];
  delete require.cache[require.resolve('./git-source/main/git')];
  const gitMod = require('./git-source/main/git');

  try {
    gitMod.setWorkspaceAuthority(authorityMod.createWorkspaceAuthority({ workspace: tmp }));

    for (const badUrl of [
      'ext::sh -c touch /tmp/pwn',
      'EXT::id',
      'fd::1',
      'file:///etc/passwd',
      '-oProxyCommand=id',
      'https://user@host/repo\n-bad',
      'git@host::bad',
    ]) {
      const before = invocations.length;
      const res = await gitMod.gitPublishRepository(tmp, { remoteUrl: badUrl });
      assert.equal(res.ok, false, `must reject unsafe remoteUrl: ${badUrl}`);
      assert.equal(invocations.length, before, `unsafe remoteUrl spawned process: ${badUrl}`);
    }

    for (const badName of ['-evil', '--template=/tmp/x', 'org/evil repo', '../escape', '.']) {
      const before = invocations.length;
      const res = await gitMod.gitPublishRepository(tmp, { name: badName });
      assert.equal(res.ok, false, `must reject unsafe repo name: ${badName}`);
      assert.equal(invocations.length, before, `unsafe name spawned process: ${badName}`);
    }

    for (const goodUrl of [
      'https://github.com/org/repo.git',
      'http://git.example.test/org/repo.git',
      'ssh://git@example.com:2222/org/repo.git',
      'git://example.com/org/repo.git',
      'git@github.com:org/repo.git',
    ]) {
      const before = invocations.length;
      const res = await gitMod.gitPublishRepository(tmp, { remoteUrl: goodUrl });
      assert.equal(res.ok, true, `expected allowed URL to succeed: ${goodUrl}: ${JSON.stringify(res)}`);
      const addCall = invocations.slice(before).find((argv) => argv[0] === 'git' && argv[1] === 'remote' && argv[2] === 'add');
      assert.deepEqual(addCall, ['git', 'remote', 'add', 'origin', '--', goodUrl]);
    }

    const beforeGh = invocations.length;
    const ghRes = await gitMod.gitPublishRepository(tmp, { name: 'org/safe-repo', visibility: 'public' });
    assert.equal(ghRes.ok, true);
    const ghCall = invocations.slice(beforeGh).find((argv) => argv[0] === 'gh');
    assert.deepEqual(ghCall, ['gh', 'repo', 'create', '--public', '--source=.', '--remote=origin', '--yes', '--', 'org/safe-repo']);
  } finally {
    childProcess.spawn = origSpawn;
    childProcess.execFile = origExecFile;
    delete require.cache[require.resolve('./git-source/main/git-exec')];
    delete require.cache[require.resolve('./git-source/main/git')];
    fs.rmSync(tmp, { recursive: true, force: true });
  }
});

test('workspace-authority rejects sensitive subdirectories including .gnupg/.aws/.kube while allowing safe projects', () => {
  const tmpBase = fs.mkdtempSync(path.join(os.tmpdir(), 'ws-auth-sec-'));
  const fakeHome = path.join(tmpBase, 'home');
  const fakeDshParent = path.join(tmpBase, 'desktop-root');
  const fakeDshHome = path.join(fakeDshParent, 'dsh-home');
  const fakeDshSub = path.join(fakeDshHome, 'storages');
  const siblingProject = path.join(fakeDshParent, 'project-a');
  const tempProject = path.join(tmpBase, 'workspace-in-tmp');
  const origHome = process.env.HOME;
  const origUserProfile = process.env.USERPROFILE;
  const origDshdHome = process.env.DSHD_HOME;

  try {
    fs.mkdirSync(path.join(fakeHome, 'src', 'proj'), { recursive: true });
    fs.mkdirSync(fakeDshSub, { recursive: true });
    fs.mkdirSync(siblingProject, { recursive: true });
    fs.mkdirSync(tempProject, { recursive: true });
    process.env.HOME = fakeHome;
    process.env.USERPROFILE = fakeHome;
    process.env.DSHD_HOME = fakeDshHome;
    setDesktopDshHome(fakeDshHome);

    const realHome = fs.realpathSync(fakeHome);
    for (const dotDir of ['.config', '.ssh', '.gnupg', '.aws', '.kube']) {
      const target = path.join(realHome, dotDir);
      const sub = path.join(target, 'sub');
      fs.mkdirSync(sub, { recursive: true });
      const realTarget = fs.realpathSync(target);
      const realSub = fs.realpathSync(sub);
      const anchors = authorityMod.highRiskAnchorPaths();
      assert.ok(anchors.some((a) => a.toLowerCase() === realTarget.toLowerCase()), `missing anchor for ${dotDir}`);
      assert.equal(authorityMod.isHighRiskWorkspaceRoot(realTarget, anchors), true, `${dotDir} must be rejected`);
      assert.equal(authorityMod.isHighRiskWorkspaceRoot(realSub, anchors), true, `${dotDir}/sub must be rejected`);
    }

    const anchors = authorityMod.highRiskAnchorPaths();
    assert.equal(authorityMod.isHighRiskWorkspaceRoot(realHome, anchors), true, 'home itself must be rejected');
    assert.equal(authorityMod.isHighRiskWorkspaceRoot(fs.realpathSync(path.join(realHome, 'src', 'proj')), anchors), false, '~/src/proj must stay allowed');
    assert.equal(authorityMod.isHighRiskWorkspaceRoot(fs.realpathSync(fakeDshHome), anchors), true);
    assert.equal(authorityMod.isHighRiskWorkspaceRoot(fs.realpathSync(fakeDshSub), anchors), true);
    assert.equal(authorityMod.isHighRiskWorkspaceRoot(fs.realpathSync(siblingProject), anchors), false);
    assert.equal(authorityMod.isHighRiskWorkspaceRoot(fs.realpathSync(tempProject), anchors), false);

    if (process.env.APPDATA && fs.existsSync(process.env.APPDATA)) {
      const appDataSub = path.join(fs.realpathSync(process.env.APPDATA), 'WhaleIsleTestProbe');
      assert.equal(authorityMod.isHighRiskWorkspaceRoot(appDataSub, anchors), true);
    }

    const { git, authority } = loadOriginalGit({
      'main/git.js': 'exports.setWorkspaceAuthority = () => {};',
      'main/workspace-authority.js': fs.readFileSync(path.join(__dirname, 'git-source/main/workspace-authority.js'), 'utf8'),
      'shared/dsh-home.js': fs.readFileSync(path.join(__dirname, 'git-source/shared/dsh-home.js'), 'utf8'),
    });
    const gate = createGitGate({
      backend: git,
      authority,
      authorize: async () => true,
    });
    assert.throws(() => gate.configure({
      remoteEnabled: true,
      remoteGitEnabled: true,
      remoteGitRoots: [fakeDshSub],
    }), /Unsafe Git workspace root/);
    assert.doesNotThrow(() => gate.configure({
      remoteEnabled: true,
      remoteGitEnabled: true,
      remoteGitRoots: [tempProject],
    }));
  } finally {
    if (origHome === undefined) delete process.env.HOME; else process.env.HOME = origHome;
    if (origUserProfile === undefined) delete process.env.USERPROFILE; else process.env.USERPROFILE = origUserProfile;
    if (origDshdHome === undefined) delete process.env.DSHD_HOME; else process.env.DSHD_HOME = origDshdHome;
    clearDesktopDshHome();
    fs.rmSync(tmpBase, { recursive: true, force: true });
  }
});
