/* The public type registry is the guide inventory. Rendered outlets are a
 * separate occurrence inventory and are the only source of mounted handles. */
function whaleCreateRightPaneBridge(ctx, source) {
  let registry = null, service = null, offRegistry = () => {}, offMounted = () => {}, offTabs = () => {};
  const pending = new Map();
  const text = value => String(value ?? '').trim().slice(0, 8192);
  const readCopy = (entry, key) => { try { return text(typeof entry[key] === 'function' ? entry[key]() : entry[key]); } catch { return ''; } };
  const same = (a, b) => a?.kind === 'session' && b?.kind === 'session' && a.sessionId === b.sessionId && a.epoch === b.epoch;
  function bind() {
    const nextRegistry = ctx.get('sidebarRightTabs'), nextService = ctx.get('sidebarRight');
    if (nextRegistry !== registry) {
      offRegistry(); registry = nextRegistry || null;
      offRegistry = registry?.subscribe(source.schedule) || (() => {});
    }
    if (nextService !== service) {
      offMounted(); offTabs(); service = nextService || null;
      offMounted = service?.mounted?.subscribe(source.schedule) || (() => {});
      offTabs = service?.openTabs?.subscribe(source.schedule) || (() => {});
    }
  }
  const committed = () => (service?.openTabs?.getSnapshot() || []).filter(tab => tab.sessionId === source.subject().sessionId);
  function ready(kind) {
    const definition = registry?.get(kind);
    return !!definition && !!source.owning() && service?.mounted?.getSnapshot() === source.subject().sessionId
      && typeof service.openTab === 'function'
      && source.entries('sidebar.right.pane.tab').some(entry => text((entry.options || entry).key) === definition.id);
  }
  function kinds() {
    bind();
    return (registry?.guide() || []).map(entry => {
      const providerId = text(entry.providerId), entryId = text(entry.id), icon = entry.icon;
      const iconName = typeof icon === 'string' ? text(icon) : text(icon?.displayName || icon?.name || icon?.render?.displayName || icon?.render?.name);
      return {id:JSON.stringify([providerId, entryId]), providerId, entryId, kind:text(entry.kind), label:readCopy(entry, 'title'), description:readCopy(entry, 'description'),
        icon:icon === undefined ? 'cube' : iconName || JSON.stringify([providerId, entryId, 'icon']), order:Number.isFinite(entry.order) ? Math.trunc(entry.order) : 0, available:ready(entry.kind)};
    });
  }
  function tabs(available) {
    const inventory = committed();
    return available.filter(row => row.slot === 'sidebar.right.pane.tab').flatMap(row => {
      const record = source.outlets().get(row.handle), id = record?.marker.closest('[data-sidebar-right-tab]')?.getAttribute('data-sidebar-right-tab');
      const tab = inventory.find(tab => tab.tabId === id), definition = tab && registry?.get(tab.kind);
      if (!tab || !definition || text((record.entry.options || record.entry).key) !== definition.id) return [];
      let label = ''; try { label = text(definition.title(tab.contentId)); } catch { /* Keep the original outlet label. */ }
      return [{handle:row.handle, id:text(id), kind:text(tab.kind), label:label || row.label}];
    });
  }
  function reply(request, handle, error) { source.send({type:'rightPaneOpened', request:request.id, subject:request.subject, handle, error}); }
  function settle(available) {
    const mounted = tabs(available);
    for (const [id, request] of pending) {
      if (!same(request.subject, source.subject()) || !source.owning() || !ready(request.kind)) {
        pending.delete(id); reply(request, '', 'right pane owner is no longer available'); continue;
      }
      const tab = mounted.find(tab => tab.id === request.tabID && tab.kind === request.kind);
      if (tab) { pending.delete(id); reply(request, tab.handle, ''); }
    }
  }
  function open(message) {
    bind();
    if (typeof message.request !== 'string' || !message.request || message.request.length > 64) return;
    const request = {id:message.request, subject:message.subject, kind:message.kind, tabID:''};
    if (!same(message.subject, source.subject()) || !ready(message.kind) || !(registry.guide() || []).some(entry => entry.kind === message.kind)) {
      reply(request, '', 'right pane kind is no longer available'); return;
    }
    const before = new Set(committed().map(tab => tab.tabId));
    try { service.openTab(message.kind); }
    catch (error) { reply(request, '', text(error?.message) || 'right pane could not be opened'); return; }
    const candidates = committed().filter(tab => tab.kind === message.kind), added = candidates.filter(tab => !before.has(tab.tabId));
    if (added.length === 1) request.tabID = added[0].tabId;
    else {
      // The public controller reports the settled active record when an
      // existing singleton was revealed instead of creating a new occurrence.
      const active = service.active();
      if (active?.kind === message.kind && candidates.some(tab => tab.tabId === active.id)) request.tabID = active.id;
      else if (candidates.length === 1 && registry.get(message.kind)?.multiple !== true) request.tabID = candidates[0].tabId;
    }
    if (!request.tabID) { reply(request, '', 'right pane open did not resolve to a committed tab'); return; }
    pending.set(request.id, request); source.schedule();
  }
  return {kinds, tabs, settle, open,
    reject(message, error) { if (typeof message.request === 'string') reply({id:message.request, subject:message.subject}, '', error); },
    cancel(message) { const request = pending.get(message.request); if (request && same(request.subject, message.subject)) pending.delete(message.request); },
    clear() { for (const request of pending.values()) reply(request, '', 'right pane owner is no longer available'); pending.clear(); },
    dispose() { this.clear(); offRegistry(); offMounted(); offTabs(); registry = null; service = null; },
  };
}
