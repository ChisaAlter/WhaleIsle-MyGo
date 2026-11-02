// Explicit original-DOM portals only. This bridge records ownership; it never
// creates, moves, focuses, clones or dispatches input to an original node.
(() => {
 'use strict';
 if (window !== window.top) return;
 const create = env => {
  const doc = document, key = env?.documentKey;
  const records = new Map(), byNode = new WeakMap();
  const limit = 64, textLimit = 65536;
  let closed = false, observer = null;
  const finite = Number.isFinite;
  const textID = value => typeof value === 'string' && value.length > 0 && value.length <= 8192;
  const integer = value => Number.isSafeInteger(value) && value >= 0;
  const element = node => node instanceof Element && node.ownerDocument === doc && node.isConnected && doc.defaultView === window;
  const subjectOK = subject => subject?.kind === 'session' && textID(subject.sessionId) && textID(subject.epoch);
  const sameSubject = (a, b) => {
   try { return subjectOK(a) && subjectOK(b) && a.kind === b.kind && a.sessionId === b.sessionId && a.epoch === b.epoch && env.same(a, b) === true; }
   catch { return false; }
  };
  const retire = token => {
   const record = records.get(token);
   if (!record) return;
   records.delete(token);
   if (byNode.get(record.portalNode) === token) byNode.delete(record.portalNode);
  };
  const retireAll = () => { for (const token of records.keys()) retire(token); };
  function removed(mutations) {
   // A connected node removed and reinserted before the next read is still a
   // different DOM lifetime. Mutation records retire its old token, even in ABA.
   for (const mutation of mutations) for (const node of mutation.removedNodes || []) {
    if (!(node instanceof Node)) continue;
    for (const [token, record] of records) {
     if (node === record.rootNode || node.contains(record.rootNode) || node === record.ownerNode || node.contains(record.ownerNode) || node === record.portalNode || node.contains(record.portalNode)) retire(token);
    }
   }
  }
  function drain() { if (observer) removed(observer.takeRecords()); }
  function sourceNow() {
   if (closed || typeof key !== 'string' || key.length === 0 || key.length > 256 || env.documentKey !== key || window.document !== doc || doc.defaultView !== window || window !== window.top || typeof env.source !== 'function' || typeof env.same !== 'function') return null;
   try {
    const value = env.source();
    if (!value || value.closed || !sameSubject(value.subject, value.subject) || !Array.isArray(value.owners) || value.documentKey !== undefined && value.documentKey !== key) return null;
    return value;
   } catch { return null; }
  }
  function canonical(source, record) {
   if (!sameSubject(record.subject, source.subject) || !element(record.rootNode) || !element(record.ownerNode) || !element(record.portalNode) || !record.rootNode.contains(record.ownerNode)) return false;
   const matching = source.owners.filter(row => row?.handle === record.handle);
   if (matching.length !== 1) return false;
   const row = matching[0];
   if (row.node !== record.rootNode || row.entry !== record.entry || row.turn !== record.turn || row.seq !== record.seq) return false;
   // An explicit owner may be a child, but cannot become ambiguous because
   // another canonical owner appeared around it after registration.
   const containing = source.owners.filter(owner => element(owner?.node) && owner.node.contains(record.ownerNode));
   if (containing.length !== 1 || containing[0] !== row) return false;
   if (!portalAllowed(record.portalNode, record.ownerNode, source)) return false;
   return readText(record.portalNode) !== null;
  }
  function portalAllowed(portal, owner, source) {
   if (!element(portal) || !element(owner) || portal === doc.documentElement || portal === doc.body || portal === doc.getElementById('root') || portal === doc.getElementById('app')) return false;
   if (owner.contains(portal) || portal.contains(owner)) return false;
   // An ancestor of any canonical owner is a shared container, not a portal.
   return !source.owners.some(row => element(row?.node) && row.node.contains(owner) && (row.node.contains(portal) || portal.contains(row.node))) && !source.owners.some(row => element(row?.node) && portal.contains(row.node));
  }
  function readText(node) {
   try { const text = node.textContent; return typeof text === 'string' && text.length <= textLimit ? text : null; }
   catch { return null; }
  }
  function current() {
   drain();
   const source = sourceNow();
   if (!source) { retireAll(); return null; }
   for (const [token, record] of records) if (!canonical(source, record)) retire(token);
   return source;
  }
  function visible(node, rect) {
   if (rect.w <= 0 || rect.h <= 0 || rect.x + rect.w <= 0 || rect.y + rect.h <= 0 || rect.x >= innerWidth || rect.y >= innerHeight || node.getClientRects().length === 0) return false;
   for (let parent = node; parent instanceof Element; parent = parent.parentElement) {
    if (parent.hidden || parent.getAttribute('aria-hidden') === 'true') return false;
    const style = getComputedStyle(parent);
    if (style.display === 'none' || style.visibility === 'hidden' || style.visibility === 'collapse' || Number(style.opacity) === 0) return false;
   }
   return true;
  }
  function descriptor(record) {
   try {
    const bounds = record.portalNode.getBoundingClientRect();
    const rect = {x:bounds.x, y:bounds.y, w:bounds.width, h:bounds.height};
    const text = readText(record.portalNode);
    if (![rect.x, rect.y, rect.w, rect.h].every(finite) || rect.w < 0 || rect.h < 0 || text === null) return null;
    const value = {token:record.token, handle:record.handle, entry:record.entry, turn:record.turn, seq:record.seq, rect, visible:visible(record.portalNode, rect), text};
    if (typeof record.label === 'string' && record.label.length <= 8192) value.label = record.label;
    return value;
   } catch { return null; }
  }
  if (typeof MutationObserver === 'function') {
   observer = new MutationObserver(removed);
   observer.observe(doc, {childList:true, subtree:true});
  }
  const api = {
   registerPortal(value) {
    const source = current(), owner = value?.ownerNode, portal = value?.portalNode;
    if (!source || !observer || !element(owner) || !portalAllowed(portal, owner, source) || readText(portal) === null) return null;
    const owners = source.owners.filter(row => element(row?.node) && row.node.contains(owner));
    if (owners.length !== 1) return null;
    const row = owners[0];
    if (!textID(row.handle) || !textID(row.entry) || !integer(row.turn) || !integer(row.seq) || source.owners.filter(item => item?.handle === row.handle).length !== 1) return null;
    const prior = byNode.get(portal);
    if (prior) { const old = records.get(prior); return old?.ownerNode === owner && canonical(source, old) ? prior : null; }
    if (records.size >= limit) return null;
    for (const old of records.values()) if (old.portalNode.contains(portal) || portal.contains(old.portalNode)) return null;
    const token = crypto.randomUUID();
    if (typeof token !== 'string' || records.has(token)) return null;
    const record = {token, documentKey:key, subject:{kind:source.subject.kind, sessionId:source.subject.sessionId, epoch:source.subject.epoch}, rootNode:row.node, ownerNode:owner, portalNode:portal, handle:row.handle, entry:row.entry, label:row.label, turn:row.turn, seq:row.seq};
    // Reject bad geometry before inserting, so failed registration has no token.
    if (!descriptor(record)) return null;
    records.set(token, record); byNode.set(portal, token);
    return token;
   },
   unregisterPortal(token) { retire(token); return true; },
   portals() {
    if (!current()) return [];
    const values = [];
    for (const [token, record] of records) { const value = descriptor(record); if (value) values.push(value); else retire(token); }
    return values;
   },
   portalTarget(token, hit) {
    if (!current()) return false;
    const record = records.get(token);
    if (!record || !element(hit) || !record.portalNode.contains(hit)) return false;
    const value = descriptor(record);
    return !!value?.visible;
   },
   clear() { retireAll(); return true; },
   close() { closed = true; retireAll(); observer?.disconnect(); observer = null; return true; }
  };
  return Object.freeze(api);
 };
 Object.defineProperty(window, '__whaleCreateInlinePortalBridge', {value:create, writable:false, configurable:false});
})();
