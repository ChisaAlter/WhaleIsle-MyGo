'use strict';
// Embedded CommonJS closure. No source extraction or dependency installation.
const readline = require('node:readline');
const Module = require('node:module');
const path = require('node:path');
let service;
let closing = false;
const active = new Set();
const input = readline.createInterface({ input: process.stdin });
function reply(value) { process.stdout.write(JSON.stringify(value) + '\n'); }
input.on('line', async line => {
  let request;
  try {
    request = JSON.parse(line);
    if (!service) {
      if (request.op !== 'initialize') throw new Error('worker-not-initialized');
      const modules = new Map();
      function load(name) {
        if (modules.has(name)) return modules.get(name).exports;
        if (!Object.prototype.hasOwnProperty.call(request.sources, name)) throw new Error('unknown-component-module');
        const mod = new Module(path.join(process.cwd(), name + '.js'));
        modules.set(name, mod);
        mod.filename = path.join(process.cwd(), name + '.js');
        mod.paths = [];
        mod.require = id => id.startsWith('./') ? load(id.slice(2)) : require(id);
        mod._compile(request.sources[name], mod.filename);
        return mod.exports;
      }
      service = load('index').createComponentsService({ userDataDir: request.userData, samplesRoot: request.samplesRoot, execPath: process.execPath });
      reply({ requestId: request.requestId, ok: true });
      return;
    }
    if (closing) throw new Error('worker-closing');
    if (request.op === 'shutdown') {
      closing = true;
      await Promise.allSettled([...active]);
      service.shutdown();
      reply({requestId: request.requestId, ok: true});
      input.close();
      process.stdin.pause();
      return;
    }
    if (!['list','install','start','stop','update','rollback','uninstall'].includes(request.op)) throw new Error('unknown-action');
    const operation = Promise.resolve(service[request.op](request.target, progress => reply({requestId: request.requestId, progress})));
    active.add(operation);
    let result;
    try { result = await operation; } finally { active.delete(operation); }
    reply({ requestId: request.requestId, ...result });
  } catch (error) {
    reply({ requestId: request?.requestId, ok: false, error: error.code || 'component-failed', message: error.message });
  }
});
// EOF means the owning launcher has exited, not merely closed its window.
input.on('close', async () => {
  if (!closing && service) {
    closing = true;
    await Promise.allSettled([...active]);
    service.shutdown();
  }
});
