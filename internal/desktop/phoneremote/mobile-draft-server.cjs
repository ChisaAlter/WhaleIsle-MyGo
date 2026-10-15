'use strict';
// Adapt only the app entry served by the existing mobile server; installed
// runtime files, pairing, daemon RPC routes, and other assets are untouched.
function installMobileDraftUI(remoteModule, extension) {
 const fs = require('node:fs');
 const path = require('node:path');
 const dir = path.dirname(remoteModule);
 const web = require(path.join(dir, 'mobile-web.js'));
 const serverModule = require(path.join(dir, 'mobile-web-server.js'));
 const createServer = serverModule.createMobileWebServer;
 serverModule.createMobileWebServer = function(options = {}) {
  const root = options.root || web.resolveMobileWebRoot();
  const app = fs.readFileSync(path.join(root, 'app.js'), 'utf8');
  const originalImport = "import { gitCall, hostCall } from './host/backend.js';";
  if (!app.includes(originalImport)) throw Error('Installed mobile app entry is incompatible with draft editing');
  const draftEntry = "sheetItem({ label: 'PR / MR 标题正文草稿', enabled: !state.gitBusy && Boolean(status.refName), onClick: mygoOpenDraftEditor }),\n      ";
  const appWithEntry = app.replace("sheetItem({ label: 'Pull', enabled:", draftEntry + "sheetItem({ label: 'Pull', enabled:");
  const source = appWithEntry.replace(originalImport, "import { gitCall as mygoOriginalGitCall, hostCall } from './host/backend.js';") + '\n' + extension;
  const server = createServer(options);
  const handlers = server.listeners('request'); server.removeAllListeners('request');
  server.on('request', (req, res) => {
   if ((req.url || '').split('?')[0] !== '/app.js' || !['GET','HEAD'].includes(req.method)) {
    for (const handler of handlers) handler.call(server, req, res);
    return;
   }
   res.writeHead(200, {'content-type':'text/javascript; charset=utf-8','cache-control':'no-cache',
    'x-content-type-options':'nosniff','referrer-policy':'no-referrer','x-frame-options':'DENY'});
   res.end(req.method === 'HEAD' ? undefined : source);
  });
  return server;
 };
}
