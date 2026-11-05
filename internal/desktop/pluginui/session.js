/* The original conversation owns every component, selector and Session scope.
 * Entry markers are supplied only by the matched renderer response adapter. */
/* The stock composer is the fallback branch of the composer chain, but its input seats are additive:
 * entries that third-party plugins add there are real contributions, not part of the fallback. */
function whaleAdditiveComposerSlot(slot) {
  return slot === 'conversation.composer.dock' || String(slot).startsWith('conversation.input.');
}
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
  const slots = ['conversation.composer.bar','conversation.chat.reasoning.body','conversation.chat.user.content','tool.call.body','sidebar.right.pane.tab','conversation.session.header.actions', 'conversation.session.header.utilities', 'conversation.session.header.corner', 'conversation.session.header.lineage', 'conversation.session.header.leading', 'conversation.session.body', 'conversation.view', 'conversation.composer', 'conversation.composer.dock', 'conversation.input.dock', 'conversation.input.overlay', 'conversation.input.left', 'conversation.input.right', 'conversation.input.activity', 'conversation.input.plan', 'conversation.input.permission', 'conversation.input.model', 'conversation.input.managed', 'conversation.input.attachments', 'conversation.chat.node', 'conversation.chat.turnTail', 'conversation.chat.assistant-actions', 'conversation.chat.user-actions', 'conversation.chat.user-editor', 'conversation.chat.commandview', 'conversation.approval.detail', 'conversation.approval.actions', 'conversation.plan-review.actions', 'tool.call.toolview', 'tool.call.images'];
  if (new URLSearchParams(location.search).get('desktopNativeSession') !== '1') return;
  ctx.effect(() => {
    let stopped = false, revision = 0, last = '', requested = {kind:'session', sessionId:'', epoch:''}, navigationPending = false;
    let current = null, selectedHandle = '', surfaceRequest = '', pageMode = false, focusRequested = false, pending = false, serial = 0, bound = null, offSession = () => {};
    let outlets = new Map(), actions = new Map(), sessionActions = new Map(), nativeActions = new Map();
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
      return {subject:requested, owners:[...outlets].filter(([,record]) => record.slot === 'conversation.chat.turnTail' && validTarget(record) && whaleSessionTurnTailOf(record.marker)).map(([id,record]) => {const tail=whaleSessionTurnTailOf(record.marker);return {handle:id,entry:text((record.entry.options||record.entry).id || (record.entry.options||record.entry).key),label:entryLabel(record.entry) || text((record.entry.options||record.entry).id || (record.entry.options||record.entry).key),turn:tail.turn,seq:tail.seq,node:record.marker};})};
    });
    const text = value => String(value || '').trim().slice(0, 8192);
    const entryLabel = entry => { try { return text(typeof (entry.options||entry).label === 'function' ? (entry.options||entry).label() : (entry.options||entry).label); } catch { return ''; } };
    const entries = slot => ctx.slots.entriesOfSlot(slot);
    const handle = node => { let id = ids.get(node); if (!id) { id = String(++serial); ids.set(node, id); } return id; };
    const same = (a, b) => a?.kind === 'session' && b?.kind === 'session' && a.sessionId === b.sessionId && a.epoch === b.epoch;
    const body = () => [...document.querySelectorAll('[data-conversation-content][data-conversation-session]')].find(node => node.getAttribute('data-conversation-session') === requested.sessionId);
    const pageRoot = () => document.querySelector('[data-slot="main.conversation"]');
    const observation = () => typeof window.__whaleSessionEntryOf === 'function';
    const rightPane = whaleCreateRightPaneBridge(ctx, {subject:() => requested, owning, entries, outlets:() => outlets, schedule,
      send:message => { if (!stopped && socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify(message)); }});
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
    function rootFor(slot, root) { if(slot.startsWith('sidebar.right.'))return [...document.querySelectorAll('[data-sidebar-right-session]')].find(node=>node.getAttribute('data-sidebar-right-session')===requested.sessionId);return slot.startsWith('conversation.session.header.') ? pageRoot() : root; }
    function liveEntry(marker, slot) {
      if (!observation() || !marker?.isConnected) return null;
      const entry = window.__whaleSessionEntryOf(marker, slot);
      return entry && (marker.hasAttribute?.('data-whale-factory-entry') || entries(slot).includes(entry)) ? entry : null;
    }
    function hasContent(node) {
      if (node.querySelector('[data-slot-error],[data-factory-error]')) return false;
      return !!(text(node.textContent) || node.querySelector('button,input,textarea,img,svg,canvas,video,[contenteditable]'));
    }
    function visibleContent(node, slot) {
      if (node.hidden || node.getAttribute('aria-hidden') === 'true' || node.closest('[hidden],[aria-hidden="true"]')) return false;
      if (node.closest('[data-chain-overlay-fallback]') && !whaleAdditiveComposerSlot(slot)) return false;
      return hasContent(node);
    }
    function validTarget(record) {
      const owner = owning();
      if(record?.pendingKey&&ctx.get('uiSession')?.sessionStatus.getSnapshot().get(record.agentID)?.pendingInteraction?.key!==record.pendingKey)return false;
      return !!owner && !!record?.node?.isConnected && !!rootFor(record.slot, owner.root)?.contains(record.marker) && liveEntry(record.marker, record.slot) === record.entry && visibleContent(record.marker, record.slot);
    }
    function collect(owner) {
      const rows = []; outlets = new Map(); const nativeBudget={left:512};
      if (!observation()) return rows;
      for (const slot of slots) {
        const declared = entries(slot);
        const scope = rootFor(slot, owner.root);
        const markers = [...(scope?.querySelectorAll('[data-whale-slot-entry]') || [])].filter(node => liveEntry(node, slot) && visibleContent(node, slot));
        for (const marker of markers) {
          const entry = liveEntry(marker, slot), id = handle(marker); if (whaleReportedRegistrant(entry, entries) === whaleBundledMarker) continue;
          const attribute = name => marker.closest('['+name+']')?.getAttribute(name) || '';
          let callID = attribute('data-whale-request-call') || attribute('data-chat-call-id');
          const agentID = attribute('data-whale-request-agent') || requested.sessionId;
          let sourceRequestID = attribute('data-whale-request-id');
          const requestKey = attribute('data-whale-request-key'), questionPrefix = 'question:'+agentID+':';
          let pendingKey='';
          if(slot==='conversation.approval.detail'||slot==='conversation.approval.actions'){
           const pending=ctx.get('uiSession')?.sessionStatus.getSnapshot().get(agentID)?.pendingInteraction;
           if(pending?.kind!=='approval'||pending.sessionId!==agentID||pending.key!==requestKey)continue;
           pendingKey=pending.key;
          }
          if(slot==='conversation.plan-review.actions'){
           callID=requestKey.startsWith(questionPrefix)?requestKey.slice(questionPrefix.length):'';
           const pending=ctx.get('uiSession')?.sessionStatus.getSnapshot().get(agentID)?.pendingInteraction;
           if(pending?.kind!=='plan-review'||pending.sessionId!==agentID||pending.key!==requestKey)continue;
           pendingKey=pending.key;
           if(!callID&&!pending.callId&&Array.isArray(pending.questions))sourceRequestID='questions:'+JSON.stringify(pending.questions);
          }
          const {turn, hasTurn} = whaleSessionTurnOf(marker);
          const tail = slot === 'conversation.chat.turnTail' ? whaleSessionTurnTailOf(marker) : null;
          const rawSeq=marker.closest('[data-whale-owner-seq]')?.getAttribute('data-whale-owner-seq'),seq=typeof rawSeq==='string'&&/^\d+$/.test(rawSeq)?Number(rawSeq):null;
          // A composer chain owns the entire original composer, including its
          // fallback branch. It becomes a seat only when a chain entry is elected.
          const node = slot === 'conversation.composer' ? owner.root.querySelector('[data-composer-seat]') || marker : marker;
          const record={node, marker, slot, entry,agentID,pendingKey};
          outlets.set(id, record);
          const native=typeof whaleNativeDocument==='function'?whaleNativeDocument(marker,handle,nativeActions,record,nativeBudget):[];
          rows.push({handle:id, slot, label:entryLabel(entry) || text((entry.options||entry).id || (entry.options||entry).key) || slot, entryId:text((entry.options||entry).id || (entry.options||entry).key), registrant:whaleReportedRegistrant(entry,entries), callId:callID, agentId:agentID,sourceRequestId:sourceRequestID,pendingKey, turn, hasTurn, turnTailSeq:tail?.seq || 0, hasTurnTail:tail !== null, available:true,native,seq:seq||0,hasSeq:Number.isSafeInteger(seq)&&seq>=0});
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
      window.__whaleNativeComposer?.connect(message=>{if(socket.readyState===WebSocket.OPEN)socket.send(JSON.stringify(message))},requested);
      const owner = owning();
      const subject = owner ? requested : null;
      const rows = slots.flatMap(slot => entries(slot).map(entry => ({slot, id:text((entry.options||entry).id), key:text((entry.options||entry).key), label:entryLabel(entry), registrant:whaleReportedRegistrant(entry,entries), order:Number.isFinite((entry.options||entry).order) ? (entry.options||entry).order : 0, summary:''})));
      let available = [], controls = [], headerControls = [], surface = '', unavailable = '';
      actions = new Map(); sessionActions = new Map(); nativeActions = new Map();
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
      const cv=window.__whaleNativeConversation,targets=cv?.targets()||[];
      const conversationViews=owner?(cv?.views()||[]).map(row=>({handle:String(row.id),id:String(row.id),label:String(row.label||row.id),target:targets.includes(row.id)?String(row.id):''})):[];
      const rightPaneKinds=rightPane.kinds(), rightPaneTabs=rightPane.tabs(available);
      const snapshot = {rightPaneKinds,rightPaneTabs,conversationViews,locale:ctx.locale.getSnapshot().active, entries:rows, subject, actions:controls, sessionControls:headerControls, observation:observation(), badges:[], section:false, configuration:false, surface, height:current ? Math.min(1000, Math.max(40, current.scrollHeight)) : 0, outlets:available, sessionSurfaceHandle:current && !pageMode ? selectedHandle : '', sessionSurfaceRequest:current && !pageMode ? surfaceRequest : '', unavailable};
      const encoded = JSON.stringify(snapshot);
      if (socket.readyState !== WebSocket.OPEN) return;
      if (encoded !== last) { last = encoded; revision++; socket.send(JSON.stringify({type:'state', revision, ...snapshot})); }
      // State precedes the reply on this socket: native must already own the
      // mounted outlet before an open returns its handle.
      rightPane.settle(available);
    }
    const richDispose=whaleRichSurface.bind(()=>({revision,subject:requested,actions:nativeActions,valid:record=>validTarget(record.owner)}));
    function schedule() { if (stopped || pending) return; pending = true; queueMicrotask(() => { pending = false; publish(); }); }
    socket.addEventListener('open', schedule);
    socket.addEventListener('message', event => {
      let message; try { message = JSON.parse(event.data); } catch { return; }
      if (stopped) return;
      if(message.type==='richRelease'){whaleRichSurface.release(message);return;}
if(message.type==='richLayout'){whaleRichSurface.ack(message,reply=>socket.readyState===WebSocket.OPEN&&socket.send(JSON.stringify(reply)));return;}
      if (message.type==='composerState'||message.type==='composerReply'){window.__whaleNativeComposer?.receive(message);schedule();return;}
      if (message.type === 'select' && message.subject?.kind === 'session') { rightPane.clear(); whaleRichSurface.clear(); clearSurface(); sessionActions.clear(); requested = message.subject; navigationPending = true; schedule(); return; }
      if (message.type === 'surface' && message.slot === '') { clearSurface(); schedule(); return; }
      if (message.type === 'cancelRightPaneOpen') { rightPane.cancel(message); return; }
      if (message.revision !== revision || !same(message.subject, requested)) {
        if (message.type === 'openRightPaneKind') { rightPane.reject(message, 'right pane source changed before the open'); return; }
        if (message.type !== 'click' || message.revision !== revision) return;
      }
      if (message.type === 'openRightPaneKind') { rightPane.open(message); return; }
      if(message.type==='closeRightPaneTab'){const record=outlets.get(message.handle);if(validTarget(record)&&record.slot==='sidebar.right.pane.tab'){const id=record.marker.closest('[data-sidebar-right-tab]')?.getAttribute('data-sidebar-right-tab'),sidebar=ctx.get('sidebarRight');if(id&&sidebar?.mounted?.getSnapshot()===requested.sessionId)sidebar.close(id);schedule()}return;}
      if(message.type==='rightPaneTab'){const record=outlets.get(message.handle);if(validTarget(record)&&record.slot==='sidebar.right.pane.tab'){const id=record.marker.closest('[data-sidebar-right-tab]')?.getAttribute('data-sidebar-right-tab');const tabs=[...document.querySelectorAll('[data-dockkit-tab]')].filter(node=>node.getAttribute('data-dockkit-tab')===id&&node.closest('[data-sidebar-right-session]')?.getAttribute('data-sidebar-right-session')===requested.sessionId);if(tabs.length===1)tabs[0].click();schedule()}return;}
      if(message.type==='conversationView'){const cv=window.__whaleNativeConversation;if(owning()&&cv?.views().some(row=>row.id===message.handle)){cv.selectors.get(requested.sessionId)?.(message.handle);schedule()}return;}
      if(message.type==='nativeInput'){const target=nativeActions.get(message.handle);if(target?.owner&&validTarget(target.owner)&&whaleNativeContains(target.owner,target.node)&&whaleNativeInteraction(message,target)){schedule()}return;}
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
    observer.observe(document.documentElement, {childList:true, subtree:true, characterData:true, attributes:true, attributeFilter:['hidden','aria-hidden','disabled', 'aria-disabled', 'aria-expanded', 'aria-hidden', 'data-conversation-session', 'data-whale-slot-entry', 'data-turn-tail', 'data-chat-call-id', 'data-whale-turn-tail-turn', 'data-whale-turn-tail-end-seq','data-whale-owner-seq','data-whale-request-agent','data-whale-request-id','data-whale-request-call','data-whale-request-key']});
    document.addEventListener?.('input',schedule,true);document.addEventListener?.('change',schedule,true);
    cleanup.push(()=>document.removeEventListener?.('input',schedule,true),()=>document.removeEventListener?.('change',schedule,true),() => observer.disconnect(), ctx.on('slots/changed', schedule), ctx.locale.subscribe(schedule), ctx.layout.panelInfo.subscribe(schedule), ctx.sessions.list.subscribe(schedule));
    const statusOff=ctx.get('uiSession')?.sessionStatus.subscribe(schedule);if(statusOff)cleanup.push(statusOff);
    document.addEventListener('whale-rich-portals',schedule);cleanup.push(()=>{richDispose();document.removeEventListener('whale-rich-portals',schedule)});
    const dispose = () => {
      if (stopped) return;
      rightPane.dispose(); stopped = true; window.__whaleNativeComposer?.disconnect(); clearSurface(); outlets.clear(); sessionActions.clear(); offSession();
      cleanup.reverse().forEach(fn => fn()); style.remove(); window.removeEventListener('pagehide', dispose); socket.close();
    };
    socket.addEventListener('close', dispose); socket.addEventListener('error', dispose); window.addEventListener('pagehide', dispose); schedule();
    return dispose;
  });
}
