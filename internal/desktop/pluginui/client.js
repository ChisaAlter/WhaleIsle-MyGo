/* Runs inside the existing Host browser composition. The Plugins page remains
 * the sole renderSlot owner and supplies forms, revisions and detail subjects. */
window.__ModuleLoader__.load({
  id: 'whaleisle-native-plugin-ui',
  factory() {
    const slots = ['plugins.item', 'plugins.bundle.config', 'plugins.row.config', 'plugins.detail.actions', 'plugins.detail.badge', 'plugins.detail.section', 'plugins.add.actions', 'plugins.bundle.activation'];
    function apply(ctx) {
      whaleBindSessionUI(ctx);
      if (new URLSearchParams(location.search).get('desktopNativePlugins') !== '1') return;
      ctx.effect(() => {
        let stopped = false, revision = 0, last = '', requested = {kind:'list'}, surface = '', current = null, handles = new Map(), serial = 0, pending = false, selectionPending = true, activationPending = '';
        const ids = new WeakMap(), cleanup = [];
        const endpoint = new URL('/__whale_plugin_ui/events', location.href);
        endpoint.protocol = endpoint.protocol === 'https:' ? 'wss:' : 'ws:';
        const socket = new WebSocket(endpoint.href, 'whaleisle.plugin-ui.v1');
        const style = document.createElement('style');
        style.textContent = 'body *{visibility:hidden!important;pointer-events:none!important} [data-whale-plugin-surface],[data-whale-plugin-surface] *{visibility:visible!important;pointer-events:auto!important} [data-whale-plugin-surface]{display:block!important;position:fixed!important;inset:0!important;margin:0!important;padding:12px!important;box-sizing:border-box!important;overflow:auto!important;background:var(--dsw-alias-bg-base);z-index:10} [role="dialog"],[role="dialog"] *,[role="alertdialog"],[role="alertdialog"] *,[role="menu"],[role="menu"] *,[role="listbox"],[role="listbox"] *{visibility:visible!important;pointer-events:auto!important}';
        document.head.appendChild(style);
        const text = value => String(value || '').trim().slice(0, 8192);
        const entries = name => ctx.slots.entriesOfSlot(name);
        const label = entry => { try { return text(typeof entry.options.label === 'function' ? entry.options.label() : entry.options.label); } catch { return ''; } };
        const exact = (selector, attribute, value, root = document) => [...root.querySelectorAll(selector)].find(node => node.getAttribute(attribute) === value);
        const activeSubject = () => {
          let node = document.querySelector('[data-plugin-row-detail]');
          if (node) { const key = node.getAttribute('data-plugin-row-detail'), at = key.indexOf('#'); return at > 0 ? {kind:'row',name:key.slice(0,at),rowId:key.slice(at+1)} : null; }
          node = document.querySelector('[data-plugin-item-detail]');
          if (node) return {kind:'item',itemId:node.getAttribute('data-plugin-item-detail')};
          node = document.querySelector('[data-plugin-detail]');
          return node ? {kind:'bundle',name:node.getAttribute('data-plugin-detail')} : null;
        };
        const same = (a,b) => a?.kind === b?.kind && (a?.name || '') === (b?.name || '') && (a?.rowId || '') === (b?.rowId || '') && (a?.itemId || '') === (b?.itemId || '');
        const detail = () => document.querySelector('[data-plugin-row-detail],[data-plugin-item-detail],[data-plugin-detail]');
        function navigate() {
          if (!entries('main').some(entry => entry.options.key === 'plugins')) return;
          if (selectionPending && ctx.layout.panelInfo.getSnapshot().activePanelId !== 'plugins') ctx.layout.selectPanel('plugins');
          if (selectionPending) {
            if (requested.kind === 'bundle' || requested.kind === 'row') {
              const navigation = ctx.get('pluginNavigation');
              if (!navigation?.openBundle) return;
              navigation.openBundle(requested.name);
            } else {
              const d = detail();
              if (d) d.querySelector('button')?.click();
            }
            selectionPending = false;
          }
          if (activationPending && requested.kind === 'list' && !detail()) {
            const card = exact('[data-plugin-package]','data-plugin-package',activationPending);
            const toggle = card?.querySelector('[role="switch"]');
            if (toggle && !toggle.disabled && toggle.getAttribute('aria-checked') === 'false' && entries('plugins.bundle.activation').some(entry => entry.options.key === activationPending)) {
              activationPending = ''; toggle.click();
            }
          }
          if (requested.kind === 'list' || same(activeSubject(), requested)) return;
          if (requested.kind === 'item') {
            const card = exact('[data-plugin-item]', 'data-plugin-item', requested.itemId);
            const buttons = card ? [...card.querySelectorAll('button[aria-label]')] : [];
            if (buttons.length === 1) buttons[0].click();
            return;
          }
          const bundle = exact('[data-plugin-detail]', 'data-plugin-detail', requested.name);
          if (!bundle) return;
          if (requested.kind === 'row') {
            const row = exact('[data-plugin-row]', 'data-plugin-row', requested.entryId || requested.rowId, bundle);
            const button = row?.querySelector('button[aria-label]');
            if (button) button.click();
          }
        }
        function select(value) {
          if (!value || !['list','bundle','row','item'].includes(value.kind)) return;
          requested = value;
          selectionPending = true;
          surface = '';
          activationPending = '';
          current?.removeAttribute('data-whale-plugin-surface'); current = null;
          schedule();
        }
        function target() {
          if (!surface) return null;
          if (surface === 'plugins.bundle.activation') return document.querySelector('[data-slot="plugins.bundle.activation"]');
          if (surface === 'plugins.add.actions') return document.querySelector('[role="menu"] [data-slot="plugins.add.actions"]');
          const scope = ['plugins.item','plugins.bundle.config','plugins.row.config'].includes(surface) ? detail()?.querySelector('[data-plugin-config]') : detail();
          return scope?.querySelector('[data-slot="' + surface + '"]') || null;
        }
        function publish() {
          if (stopped) return;
          navigate();
          const subject = activeSubject(), root = detail(), matched = same(subject, requested);
          const rows = slots.flatMap(slot => entries(slot).map(entry => {
            const id = text(entry.options.id), key = text(entry.options.key);
            const card = slot === 'plugins.item' ? exact('[data-plugin-item]', 'data-plugin-item', id) : null;
            const summary = card?.querySelector('[data-slot="plugins.item"]');
            return {slot,id,key,label:label(entry),registrant:text(entry.registrant),order:Number.isFinite(entry.options.order) ? entry.options.order : 0,summary:text(summary?.textContent)};
          }));
          handles = new Map();
          const actionRoot = matched ? root?.querySelector('[data-slot="plugins.detail.actions"]') : null;
          const actions = actionRoot ? [...actionRoot.querySelectorAll('button,[role="button"],a[href]')].filter(node => node.isConnected && !node.hidden).slice(0,128).map(node => {
            let handle = ids.get(node); if (!handle) { handle = String(++serial); ids.set(node,handle); }
            handles.set(handle,node);
            return {handle,label:text(node.getAttribute('aria-label') || node.textContent || node.getAttribute('title')),disabled:node.disabled === true || node.getAttribute('aria-disabled') === 'true'};
          }).filter(row => row.label) : [];
          const badge = matched ? root?.querySelector('[data-slot="plugins.detail.badge"]') : null;
          const section = matched ? root?.querySelector('[data-slot="plugins.detail.section"]') : null;
          const configSlot = requested.kind === 'row' ? 'plugins.row.config' : requested.kind === 'item' ? 'plugins.item' : 'plugins.bundle.config';
          const config = matched && !!root?.querySelector('[data-plugin-config] [data-slot="' + configSlot + '"]');
          const activation = document.querySelector('[data-slot="plugins.bundle.activation"]');
          if (activation && (activation.textContent.trim() || activation.querySelector('button,input,img,svg'))) surface='plugins.bundle.activation';
          const next = target();
          if (next !== current) { current?.removeAttribute('data-whale-plugin-surface'); current = next; current?.setAttribute('data-whale-plugin-surface',''); }
          const snapshot = {locale:ctx.locale.getSnapshot().active,entries:rows,subject:matched ? subject : null,actions,badges:badge ? [text(badge.textContent)].filter(Boolean) : [],section:!!section && (!!text(section.textContent) || !!section.querySelector('button,input,img,svg')),configuration:config,surface:surface && current ? surface : '',height:current ? Math.min(1000,Math.max(40,current.scrollHeight)) : 0};
          const encoded = JSON.stringify(snapshot);
          if (encoded === last || socket.readyState !== WebSocket.OPEN) return;
          last = encoded; revision++;
          socket.send(JSON.stringify({type:'state',revision,...snapshot}));
        }
        function schedule() { if (stopped || pending) return; pending = true; queueMicrotask(() => { pending = false; publish(); }); }
        socket.addEventListener('open', schedule);
        socket.addEventListener('message', event => {
          let message; try { message = JSON.parse(event.data); } catch { return; }
          if (stopped) return;
          if (message.type === 'select') select(message.subject);
          else if (message.type === 'enableActivation' && message.revision === revision && requested.kind === 'list' && entries('plugins.bundle.activation').some(entry => entry.options.key === message.name)) { activationPending=message.name; schedule(); }
          else if (message.type === 'surface' && ['',...slots].includes(message.slot)) { surface = message.slot; schedule(); }
          else if (message.type === 'click' && message.revision === revision && same(activeSubject(),requested)) {
            const node = handles.get(message.handle), owner = detail()?.querySelector('[data-slot="plugins.detail.actions"]');
            if (node?.isConnected && owner?.contains(node) && !node.disabled && node.getAttribute('aria-disabled') !== 'true') { surface = 'plugins.detail.actions'; node.click(); schedule(); }
          } else if (message.type === 'add') {
            const button = document.querySelector('[aria-haspopup="menu"][aria-expanded="false"]');
            if (button) button.click(); surface = 'plugins.add.actions'; schedule();
          }
        });
        const observer = new MutationObserver(schedule);
        observer.observe(document.documentElement,{childList:true,subtree:true,characterData:true,attributes:true,attributeFilter:['disabled','aria-disabled','aria-expanded']});
        cleanup.push(() => observer.disconnect());
        cleanup.push(ctx.on('slots/changed',schedule));
        cleanup.push(ctx.locale.subscribe(schedule));
        cleanup.push(ctx.layout.panelInfo.subscribe(schedule));
        const dispose = () => {
          if (stopped) return; stopped = true; handles.clear(); current?.removeAttribute('data-whale-plugin-surface');
          cleanup.reverse().forEach(off => off()); style.remove(); window.removeEventListener('pagehide',dispose); socket.close();
        };
        socket.addEventListener('close',dispose); socket.addEventListener('error',dispose);
        window.addEventListener('pagehide',dispose);
        schedule();
        return dispose;
      });
    }
    return {inject:['slots','locale','layout','sessions'],apply};
  }
});
