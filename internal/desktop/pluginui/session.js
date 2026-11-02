/* The original conversation owns every component, selector and Session scope.
 * Entry markers are supplied only by the matched renderer response adapter. */
function whaleSessionTurnOf(marker) {
  const raw = marker.closest('[data-turn-tail]')?.getAttribute('data-turn-tail');
  if (typeof raw !== 'string' || !/^\d+$/.test(raw.trim())) return {turn:0, hasTurn:false};
  const turn = Number(raw);
  return Number.isSafeInteger(turn) && turn >= 0 ? {turn, hasTurn:true} : {turn:0, hasTurn:false};
}
function whaleSessionTurnTailOf(marker) {
  const read = name => {
    const raw = marker.getAttribute(name);
    if (typeof raw !== 'string' || !/^\d+$/.test(raw)) return null;
    const value = Number(raw);
    return Number.isSafeInteger(value) && value >= 0 ? value : null;
  };
  const turn = read('data-whale-turn-tail-turn'), seq = read('data-whale-turn-tail-end-seq');
  const ancestor = whaleSessionTurnOf(marker);
  return turn !== null && seq !== null && ancestor.hasTurn && ancestor.turn === turn ? {turn, seq} : null;
}
function whaleBindSessionUI(ctx) {
  const slots = ['conversation.session.header.actions', 'conversation.session.header.utilities', 'conversation.session.header.corner', 'conversation.session.header.lineage', 'conversation.session.header.leading', 'conversation.session.body', 'conversation.view', 'conversation.composer', 'conversation.composer.dock', 'conversation.input.dock', 'conversation.input.overlay', 'conversation.input.left', 'conversation.input.right', 'conversation.input.activity', 'conversation.input.plan', 'conversation.input.permission', 'conversation.input.model', 'conversation.input.managed', 'conversation.input.attachments', 'conversation.chat.node', 'conversation.chat.turnTail', 'conversation.chat.assistant-actions', 'conversation.chat.user-actions', 'conversation.chat.user-editor', 'conversation.chat.commandview', 'conversation.approval.detail', 'conversation.plan-review.actions', 'tool.call.toolview', 'tool.call.images'];
  if (new URLSearchParams(location.search).get('desktopNativeSession') !== '1') return;
  ctx.effect(() => {
    let stopped = false, revision = 0, last = '', requested = {kind:'session', sessionId:'', epoch:''}, navigationPending = false;
    let current = null, selectedHandle = '', surfaceRequest = '', pageMode = false, focusRequested = false, pending = false, serial = 0, bound = null, offSession = () => {};
    let outlets = new Map(), actions = new Map(), sessionActions = new Map();
    const inlineMode = new URLSearchParams(location.search).get('desktopNativeInline') === '1';
    const ids = new WeakMap(), cleanup = [];
    const endpoint = new URL('/__whale_plugin_ui/events', location.href);
    endpoint.protocol = endpoint.protocol === 'https:' ? 'wss:' : 'ws:';
    const socket = new WebSocket(endpoint.href, 'whaleisle.plugin-ui.v1');
    const style = document.createElement('style');
    style.textContent = 'body *{visibility:hidden!important;pointer-events:none!important}[data-whale-session-surface],[data-whale-session-surface] *{visibility:visible!important;pointer-events:auto!important}[data-whale-session-surface]{display:block!important;position:fixed!important;inset:0!important;margin:0!important;padding:12px!important;box-sizing:border-box!important;overflow:auto!important;background:var(--dsw-alias-bg-base);z-index:10}[role="dialog"],[role="dialog"] *,[role="alertdialog"],[role="alertdialog"] *,[role="menu"],[role="menu"] *,[role="listbox"],[role="listbox"] *{visibility:visible!important;pointer-events:auto!important}';
    if (!inlineMode) document.head.appendChild(style);
    if (inlineMode && window.__whaleInlineMirror) window.__whaleInlineMirror.bind(() => {
      const owner = owning();
      if (!owner) return null;
      return {subject:requested, owners:[...outlets].filter(([,record]) => record.slot === 'conversation.chat.turnTail' && validTarget(record) && whaleSessionTurnTailOf(record.marker)).map(([id,record]) => {const tail=whaleSessionTurnTailOf(record.marker);return {handle:id,entry:text(record.entry.options.id || record.entry.options.key),label:entryLabel(record.entry) || text(record.entry.options.id || record.entry.options.key),turn:tail.turn,seq:tail.seq,node:record.marker};})};
    });
    const text = value => String(value || '').trim().slice(0, 8192);
    const entryLabel = entry => { try { return text(typeof entry.options.label === 'function' ? entry.options.label() : entry.options.label); } catch { return ''; } };
    const entries = slot => ctx.slots.entriesOfSlot(slot);
    const handle = node => { let id = ids.get(node); if (!id) { id = String(++serial); ids.set(node, id); } return id; };
    const same = (a, b) => a?.kind === 'session' && b?.kind === 'session' && a.sessionId === b.sessionId && a.epoch === b.epoch;
    const body = () => [...document.querySelectorAll('[data-conversation-content][data-conversation-session]')].find(node => node.getAttribute('data-conversation-session') === requested.sessionId);
    const pageRoot = () => document.querySelector('[data-slot="main.conversation"]');
    const observation = () => typeof window.__whaleSessionEntryOf === 'function';
    function clearSurface() {
      if (current?.contains(document.activeElement)) document.activeElement?.blur();
      current?.removeAttribute('data-whale-session-surface');
      current = null; focusRequested = false; selectedHandle = ''; surfaceRequest = ''; pageMode = false; actions.clear();
    }
    function navigate() {
      if (!navigationPending || !requested.sessionId || window.__whaleSessionNoOnboarding !== true) return;
      const navigation = ctx.get('uiWorkspace');
      if (!navigation?.openSession || !entries('main.conversation').length) return;
      if (!ctx.sessions.list.getSnapshot().byId[requested.sessionId]) return;
      navigationPending = false;
      navigation.openSession(requested.sessionId);
    }
    function owning() {
      if (window.__whaleSessionNoOnboarding !== true) return null;
      const root = body(), binding = requested.sessionId ? ctx.sessions.binding(requested.sessionId) : null;
      if (!root || !binding || binding !== bound || binding.session.getSnapshot().openState !== 'open' || ctx.layout.panelInfo.getSnapshot().activePanelId !== null) return null;
      return {root, binding};
    }
    function bind() {
      const next = requested.sessionId ? ctx.sessions.binding(requested.sessionId) : null;
      if (next === bound) return;
      offSession(); bound = next || null; clearSurface(); sessionActions.clear();
      offSession = next ? next.session.subscribe(schedule) : () => {};
    }
    function rootFor(slot, root) { return slot.startsWith('conversation.session.header.') ? pageRoot() : root; }
    function liveEntry(marker, slot) {
      if (!observation() || !marker?.isConnected) return null;
      const entry = window.__whaleSessionEntryOf(marker, slot);
      return entry && entries(slot).includes(entry) ? entry : null;
    }
    function hasContent(node) {
      if (node.querySelector('[data-slot-error],[data-factory-error]')) return false;
      return !!(text(node.textContent) || node.querySelector('button,input,textarea,img,svg,canvas,video,[contenteditable]'));
    }
    function visibleContent(node) {
      if (node.hidden || node.getAttribute('aria-hidden') === 'true') return false;
      if (node.closest('[data-chain-overlay-fallback]')) return false;
      return hasContent(node);
    }
    function validTarget(record) {
      const owner = owning();
      return !!owner && !!record?.node?.isConnected && !!rootFor(record.slot, owner.root)?.contains(record.marker) && liveEntry(record.marker, record.slot) === record.entry && visibleContent(record.marker);
    }
    function collect(owner) {
      const rows = []; outlets = new Map();
      if (!observation()) return rows;
      for (const slot of slots) {
        const declared = entries(slot); if (!declared.length) continue;
        const scope = rootFor(slot, owner.root);
        const markers = [...(scope?.querySelectorAll('[data-whale-slot-entry]') || [])].filter(node => liveEntry(node, slot) && visibleContent(node));
        for (const marker of markers) {
          const entry = liveEntry(marker, slot), id = handle(marker);
          const callID = marker.closest('[data-chat-call-id]')?.getAttribute('data-chat-call-id') || '';
          const {turn, hasTurn} = whaleSessionTurnOf(marker);
          const tail = slot === 'conversation.chat.turnTail' ? whaleSessionTurnTailOf(marker) : null;
          // A composer chain owns the entire original composer, including its
          // fallback branch. It becomes a seat only when a chain entry is elected.
          const node = slot === 'conversation.composer' ? owner.root.querySelector('[data-composer-seat]') || marker : marker;
          outlets.set(id, {node, marker, slot, entry});
          rows.push({handle:id, slot, label:entryLabel(entry) || text(entry.options.id || entry.options.key) || slot, entryId:text(entry.options.id || entry.options.key), registrant:text(entry.registrant), callId:callID, turn, hasTurn, turnTailSeq:tail?.seq || 0, hasTurnTail:tail !== null, available:true});
          if (rows.length >= 512) return rows;
        }
      }
      return rows;
    }
    const disabled = node => node.disabled === true || node.getAttribute('aria-disabled') === 'true';
    function controlsFor(record) {
      if (!validTarget(record)) return [];
      return [...record.marker.querySelectorAll('button,[role="button"],a[href]')].filter(node => node.isConnected && !node.hidden && node.closest('[data-whale-slot-entry]') === record.marker).slice(0, 128).map(node => {
        const label = text(node.getAttribute('aria-label') || node.textContent || node.getAttribute('title'));
        return {node, handle:handle(node), label, disabled:disabled(node)};
      }).filter(row => row.label);
    }
    function publish() {
      if (stopped) return;
      navigate(); bind();
      const owner = owning();
      const subject = owner ? requested : null;
      const rows = slots.flatMap(slot => entries(slot).map(entry => ({slot, id:text(entry.options.id), key:text(entry.options.key), label:entryLabel(entry), registrant:text(entry.registrant), order:Number.isFinite(entry.options.order) ? entry.options.order : 0, summary:''})));
      let available = [], controls = [], headerControls = [], surface = '', unavailable = '';
      actions = new Map(); sessionActions = new Map();
      if (window.__whaleSessionOnboardingIncompatible) unavailable = 'session-version-incompatible';
      else if (window.__whaleSessionNoOnboarding !== true) unavailable = 'session-version-incompatible';
      else if (!requested.sessionId) unavailable = 'no-session';
      else if (!owner) unavailable = 'session-owner-unavailable';
      if (owner) {
        available = collect(owner);
        if (pageMode) {
          const root = pageRoot();
          if (root?.isConnected && root.contains(owner.root)) { current = root; surface = 'main.conversation'; }
          else clearSurface();
        } else {
          const record = outlets.get(selectedHandle);
          if (!validTarget(record)) clearSurface();
          else { surface = record.slot; if (current !== record.node) { current?.removeAttribute('data-whale-session-surface'); current = record.node; } }
        }
        current?.setAttribute('data-whale-session-surface', '');
        if (current && focusRequested) {
          focusRequested = false;
          const focusTarget = current;
          requestAnimationFrame(() => {
            if (!stopped && current === focusTarget && focusTarget.isConnected && owning()) focusTarget.querySelector('button:not([disabled]),input:not([disabled]),textarea:not([disabled]),[contenteditable="true"],a[href]')?.focus({preventScroll:true});
          });
        }
        for (const [id, record] of outlets) {
          if (!record.slot.startsWith('conversation.session.header.')) continue;
          for (const row of controlsFor(record)) {
            if (headerControls.length >= 128) break;
            sessionActions.set(row.handle, {node:row.node, record, outlet:id});
            headerControls.push({handle:row.handle, outlet:id, slot:record.slot, label:row.label, disabled:row.disabled});
          }
        }
        const selected = outlets.get(selectedHandle);
        for (const row of controlsFor(selected)) {
          actions.set(row.handle, row.node);
          controls.push({handle:row.handle, label:row.label, disabled:row.disabled});
        }
      } else { outlets.clear(); clearSurface(); }
      if (inlineMode) { current?.removeAttribute('data-whale-session-surface'); current=null; surface=''; }
      const snapshot = {locale:ctx.locale.getSnapshot().active, entries:rows, subject, actions:controls, sessionControls:headerControls, observation:observation(), badges:[], section:false, configuration:false, surface, height:current ? Math.min(1000, Math.max(40, current.scrollHeight)) : 0, outlets:available, sessionSurfaceHandle:current && !pageMode ? selectedHandle : '', sessionSurfaceRequest:current && !pageMode ? surfaceRequest : '', unavailable};
      const encoded = JSON.stringify(snapshot);
      if (encoded === last || socket.readyState !== WebSocket.OPEN) return;
      last = encoded; revision++; socket.send(JSON.stringify({type:'state', revision, ...snapshot}));
    }
    function schedule() { if (stopped || pending) return; pending = true; queueMicrotask(() => { pending = false; publish(); }); }
    socket.addEventListener('open', schedule);
    socket.addEventListener('message', event => {
      let message; try { message = JSON.parse(event.data); } catch { return; }
      if (stopped) return;
      if (message.type === 'select' && message.subject?.kind === 'session') { clearSurface(); sessionActions.clear(); requested = message.subject; navigationPending = true; schedule(); return; }
      if (message.type === 'surface' && message.slot === '') { clearSurface(); schedule(); return; }
      if (message.revision !== revision || !same(message.subject, requested)) {
        if (message.type !== 'click' || message.revision !== revision) return;
      }
      if (message.type === 'inlineTail' && inlineMode) {
        const record=outlets.get(message.handle),tail=record?.slot==='conversation.chat.turnTail' ? whaleSessionTurnTailOf(record.marker) : null;
        if (!tail || tail.turn!==message.turnTail?.turn || tail.seq!==message.turnTail?.seq || !validTarget(record)) return;
        const element=record.marker.firstElementChild || record.marker;
        if(message.fragment===true){const u=message.u,v=message.v,range=document.createRange();range.selectNodeContents(record.marker);const r=range.getBoundingClientRect();range.detach();if(!Number.isFinite(u)||!Number.isFinite(v)||u<0||u>r.height||v<0||v>=innerHeight)return;const parent=record.marker.closest('[data-conversation-scroll]')||(()=>{for(let n=record.marker.parentElement;n;n=n.parentElement){if(n.scrollHeight>n.clientHeight+1&&/auto|scroll/.test(getComputedStyle(n).overflowY))return n}return document.scrollingElement})();parent.scrollTop+=r.y+u-v;}else element.scrollIntoView({block:'center',inline:'nearest',behavior:'instant'});schedule();return;
      }
      if (message.type === 'sessionPage' && owning()) { clearSurface(); pageMode = true; focusRequested = true; schedule(); return; }
      if (message.type === 'sessionSurface') {
        const target = outlets.get(message.handle);
        const tail = message.turnTail, currentTail = target?.slot === 'conversation.chat.turnTail' ? whaleSessionTurnTailOf(target.marker) : null;
        if (tail && (!currentTail || currentTail.turn !== tail.turn || currentTail.seq !== tail.seq)) return;
        if (tail && (typeof message.request !== 'string' || !message.request || message.request.length > 64)) return;
        if (validTarget(target)) { selectedHandle = message.handle; surfaceRequest = tail ? message.request : ''; pageMode = false; focusRequested = true; schedule(); }
        return;
      }
      if (message.type === 'sessionClick') {
        const target = sessionActions.get(message.handle);
        if (validTarget(target?.record) && target.node.isConnected && !disabled(target.node)) {
          selectedHandle = target.outlet; surfaceRequest = ''; pageMode = false; focusRequested = true;
          target.node.click(); schedule();
        }
        return;
      }
      if (message.type === 'click' && owning()) {
        const node = actions.get(message.handle), record = outlets.get(selectedHandle);
        if (validTarget(record) && node?.isConnected && record.marker.contains(node) && !disabled(node)) { node.click(); schedule(); }
      }
    });
    const observer = new MutationObserver(schedule);
    observer.observe(document.documentElement, {childList:true, subtree:true, characterData:true, attributes:true, attributeFilter:['disabled', 'aria-disabled', 'aria-expanded', 'aria-hidden', 'data-conversation-session', 'data-whale-slot-entry', 'data-turn-tail', 'data-chat-call-id', 'data-whale-turn-tail-turn', 'data-whale-turn-tail-end-seq']});
    cleanup.push(() => observer.disconnect(), ctx.on('slots/changed', schedule), ctx.locale.subscribe(schedule), ctx.layout.panelInfo.subscribe(schedule), ctx.sessions.list.subscribe(schedule));
    const dispose = () => {
      if (stopped) return;
      stopped = true; clearSurface(); outlets.clear(); sessionActions.clear(); offSession();
      cleanup.reverse().forEach(fn => fn()); style.remove(); window.removeEventListener('pagehide', dispose); socket.close();
    };
    socket.addEventListener('close', dispose); socket.addEventListener('error', dispose); window.addEventListener('pagehide', dispose); schedule();
    return dispose;
  });
}
