/* Application-owned adapter for live plugin views. Keep the original slot
 * owners, components, injected props and disposal semantics intact. */
window.__ModuleLoader__.load({
  id: 'whaleisle-desktop-plugin-views',
  factory(require) {
    const React = require('react');
    const h = React.createElement;
    const label = options => {
      try { return String(typeof options.label === 'function' ? options.label() : options.label || options.key || options.id || ''); }
      catch { return String(options.key || options.id || ''); }
    };
    // Bind the actual page-local module owner even when this is an ordinary
    // Host page with no desktopPlugin view selection. The isolated gateway
    // authenticates this one narrow channel; native bindings remain disabled.
    function bindClientModules(ctx) {
      const owner = ctx.modules?.entries;
      if (!owner?.state?.getSnapshot || !owner.state.subscribe || !owner.retry || typeof WebSocket === 'undefined') return;
      ctx.effect(() => {
        let stopped = false, retrying = false, unsubscribe = () => {};
        const endpoint = new URL('/__whale_client_modules', location.href);
        endpoint.protocol = endpoint.protocol === 'https:' ? 'wss:' : 'ws:';
        const socket = new WebSocket(endpoint.href, 'whaleisle.client-modules.v1');
        const send = value => {
          if (!stopped && socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify(value));
        };
        const publishDocument = () => {
          const recommendation = window.__whaleDocumentRecommendation;
          if (recommendation) send({type:"document",document:recommendation});
        };
        const publish = () => {
          const {syncing, failures} = owner.state.getSnapshot();
          send({type:'state', syncing, failures:failures.map(({id,message}) => ({id,message}))});
          publishDocument();
        };
        const dispose = () => {
          if (stopped) return;
          stopped = true; unsubscribe();
          window.removeEventListener('pagehide', dispose);
          window.removeEventListener('whale-document-registry', publishDocument);
          socket.close();
        };
        socket.addEventListener('open', () => { if (!stopped) publish(); });
        socket.addEventListener('close', dispose);
        socket.addEventListener('error', dispose);
        socket.addEventListener('message', async event => {
          let command;
          try { command = JSON.parse(event.data); } catch { return; }
          if (stopped || retrying || command?.type !== 'retry' || !Number.isSafeInteger(command.id) || command.id < 1) return;
          retrying = true;
          let error = '';
          try { await owner.retry(); }
          catch (reason) { error = String(reason?.message || reason); }
          finally {
            retrying = false;
            if (!stopped) { publish(); send({type:'retryResult',id:command.id,error}); }
          }
        });
        unsubscribe = owner.state.subscribe(publish);
        window.addEventListener('pagehide', dispose);
        window.addEventListener('whale-document-registry', publishDocument);
        return dispose;
      });
    }
    function apply(ctx) {
      bindClientModules(ctx);
      whaleBindDocumentViewer(ctx);
      whaleBindProductKeyboard(ctx);
      whaleRegisterShortcutImport(ctx, React);
      const plugin = new URLSearchParams(location.search).get('desktopPlugin');
      if (!plugin) return;
      let stopped = false, opened = false, revision = 0, views = [], navigationError = '', activeView = '', activeViewObj = null, focusShell = true, userInteracted = false, cancelPending = () => {}, cancelSettingsGuard = () => {};
      const listeners = new Set();
      const changed = () => { revision++; for (const listener of listeners) listener(); };
      const rootEl = typeof document !== 'undefined' ? document.documentElement : null;
      const setFocusMode = mode => {
        if (!rootEl) return;
        if (mode) {
          if (rootEl.dataset) rootEl.dataset.whalePluginFocus = mode;
          else if (typeof rootEl.setAttribute === 'function') rootEl.setAttribute('data-whale-plugin-focus', mode);
        } else {
          if (rootEl.dataset) delete rootEl.dataset.whalePluginFocus;
          else if (typeof rootEl.removeAttribute === 'function') rootEl.removeAttribute('data-whale-plugin-focus');
        }
      };
      const syncFocus = () => {
        if (stopped || !focusShell || !activeViewObj) { setFocusMode(''); return; }
        if (activeViewObj.slot === 'main') {
          const panelSnap = ctx.layout?.panelInfo?.getSnapshot?.();
          if (panelSnap && panelSnap.activePanelId === null) { setFocusMode(''); return; }
          setFocusMode('main');
          return;
        }
        if (activeViewObj.slot === 'settings.section' || activeViewObj.slot === 'settings.remote.tab') {
          const navSnap = ctx.settingsNavigation?.getSnapshot?.();
          if (navSnap && navSnap.open === false) { setFocusMode(''); return; }
          setFocusMode('settings');
          return;
        }
        setFocusMode('');
      };
      const hasVisibleDialog = () => {
        if (typeof document === 'undefined' || typeof document.querySelectorAll !== 'function') return false;
        return [...document.querySelectorAll('[role="dialog"]')].length > 0;
      };
      // SettingsRoot closes an already-open Settings modal when sessions.phase
      // flips to 'ready' and an onboarding step transitions undefined -> defined,
      // even if that step renders null. Re-open only when closed programmatically
      // before any user interaction and with no visible dialog blocking.
      function guardSettingsAutoOpen(sectionId, view) {
        if (userInteracted || !ctx.settingsNavigation?.subscribe || !ctx.settingsNavigation?.getSnapshot) return () => {};
        let retries = 0, reopenTimer = 0, expireTimer = 0, unsub = () => {};
        const stop = () => {
          unsub(); unsub = () => {};
          clearTimeout(reopenTimer); clearTimeout(expireTimer);
        };
        const check = () => {
          if (stopped || userInteracted) { stop(); return; }
          const snap = ctx.settingsNavigation.getSnapshot();
          if (snap && snap.open === false && !hasVisibleDialog() && retries < 3) {
            retries++;
            clearTimeout(reopenTimer);
            reopenTimer = setTimeout(() => {
              if (stopped || userInteracted || hasVisibleDialog()) return;
              const current = ctx.settingsNavigation.getSnapshot();
              if (current && current.open === false) {
                try {
                  ctx.settingsNavigation.open(sectionId);
                  if (view.slot === 'settings.remote.tab') {
                    cancelPending();
                    cancelPending = activateTab(view);
                  }
                } catch {}
                syncFocus();
              }
            }, 0);
          }
          syncFocus();
        };
        unsub = ctx.settingsNavigation.subscribe(check);
        expireTimer = setTimeout(stop, 5000);
        return stop;
      }
      const entries = slot => ctx.slots.entriesOfSlot(slot);
      // rc.4 StoredEntry keeps ownership and child declarations beside options;
      // options contains only the entry's key/id/order/label/priority.
      const own = entry => entry.registrant === plugin;
      const viewKey = view => [view.slot, view.id, view.bundle || '', view.item || ''].join(':');
      const buildViews = () => {
        const rows = [];
        for (const slot of ['main', 'settings.section', 'settings.remote.tab', 'sidebar.page']) {
          for (const entry of entries(slot).filter(own)) {
            const options = entry.options;
            const id = options.key || options.id;
            if (!id || (slot === 'main' && id === 'conversation')) continue;
            const tab = slot === 'sidebar.page'
              ? entries('sidebar.nav.tab').find(row => row.options.id === id)
              : (slot === 'main' ? entries('sidebar.panellist').find(row => (row.options.id || row.options.key) === id) : null);
            rows.push({slot, id, label: label(tab?.options || options)});
          }
        }
        // A plugin may contribute an item inside an existing Settings section.
        // Follow the section owner's declared child graph rather than guessing
        // a section name or rendering an entry without its owner props.
        const hasContribution = (names, visited = new Set()) => names.some(name => {
          if (visited.has(name)) return false;
          visited.add(name);
          return entries(name).some(entry => own(entry) || hasContribution(Object.keys(entry.children || {}), visited));
        });
        for (const section of entries('settings.section')) {
          const options = section.options;
          if (options.id && !rows.some(row => row.slot === 'settings.section' && row.id === options.id)
            && hasContribution(Object.keys(section.children || {}))) {
            rows.push({slot:'settings.section',id:options.id,label:label(options)});
          }
        }
        // Plugins owns its configuration forms and supplies their Host-backed
        // form/revision props. Follow that main owner's declared child slots;
        // never render plugins.* entries as standalone React components.
        for (const panel of entries('main')) {
          const options = panel.options, children = Object.keys(panel.children || {});
          if (!options.key || options.key === 'conversation' || !hasContribution(children)) continue;
          const bundles = new Set();
          if (children.includes('plugins.bundle.config')) {
            for (const entry of entries('plugins.bundle.config').filter(own)) {
              if (entry.options.key) bundles.add(entry.options.key);
            }
          }
          if (children.includes('plugins.row.config')) {
            for (const entry of entries('plugins.row.config').filter(own)) {
              const key = entry.options.key || '', separator = key.indexOf('#');
              if (separator > 0) bundles.add(key.slice(0, separator));
            }
          }
          for (const bundle of bundles) rows.push({slot:'main',id:options.key,bundle,label:bundle});
          const items = children.includes('plugins.item') ? entries('plugins.item').filter(own) : [];
          for (const item of items) {
            if (item.options.id) rows.push({slot:'main',id:options.key,item:item.options.id,label:label(item.options)});
          }
          if (!bundles.size && !items.length && !rows.some(row => row.slot === 'main' && row.id === options.key)) {
            rows.push({slot:'main',id:options.key,label:label(options)});
          }
        }
        return rows;
      };
      // Sidebar/remote tabs own local React state, with no public navigation
      // service. Activate their existing accessibility control; never clone
      // their component or call a private store/renderer API.
      function activateTab(view) {
        let observer, timer;
        const stop = () => { observer?.disconnect(); clearTimeout(timer); };
        const find = () => {
          const tabs = [...document.querySelectorAll('button[role="tab"]')];
          const matches = tabs.filter(button => view.slot === 'settings.remote.tab'
            ? button.getAttribute('data-dsh-settings-remote-tab') === view.id
            : button.getAttribute('aria-label') === view.label && !button.hasAttribute('data-dsh-settings-remote-tab'));
          if (matches.length !== 1) return false;
          stop(); matches[0].click(); return true;
        };
        if (!find()) {
          observer = new MutationObserver(find);
          observer.observe(document.documentElement, {childList: true, subtree: true});
          timer = setTimeout(() => {
            stop(); if (!stopped) { navigationError = '未能定位此视图的导航控件，请使用页面中的导航。'; changed(); }
          }, 5000);
        }
        return stop;
      }
      function activatePluginItem(id) {
        let observer, timer;
        const stop = () => { observer?.disconnect(); clearTimeout(timer); };
        const find = () => {
          const cards = [...document.querySelectorAll('[data-plugin-item]')].filter(card => card.getAttribute('data-plugin-item') === id);
          if (cards.length !== 1) return false;
          // The original card's accessible title button owns navigation.
          const buttons = [...cards[0].querySelectorAll('button[aria-label]')];
          if (buttons.length !== 1) return false;
          stop(); buttons[0].click(); return true;
        };
        if (!find()) {
          observer = new MutationObserver(find);
          observer.observe(document.documentElement, {childList:true,subtree:true});
          timer = setTimeout(() => {
            stop(); if (!stopped) { navigationError = '未能定位此插件的配置入口，请在插件页面中选择。'; changed(); }
          }, 5000);
        }
        return stop;
      }
      function open(view, fromUser) {
        cancelPending(); cancelSettingsGuard(); navigationError = ''; activeView = viewKey(view); activeViewObj = view;
        if (fromUser) userInteracted = true;
        try {
        if (view.slot === 'main') {
          const navigation = view.bundle ? ctx.get('pluginNavigation') : null;
          if (navigation?.openBundle) navigation.openBundle(view.bundle);
          else ctx.layout.selectPanel(view.id);
          if (view.item) cancelPending = activatePluginItem(view.item);
        }
        else if (view.slot === 'settings.section') {
          ctx.settingsNavigation.open(view.id);
          if (!fromUser) cancelSettingsGuard = guardSettingsAutoOpen(view.id, view);
        }
        else if (view.slot === 'settings.remote.tab') {
          ctx.settingsNavigation.open('remote');
          cancelPending = activateTab(view);
          if (!fromUser) cancelSettingsGuard = guardSettingsAutoOpen('remote', view);
        } else cancelPending = activateTab(view);
        } catch { navigationError = '该插件视图当前不可用，请刷新或重新启用插件。'; }
        syncFocus();
        changed();
      }
      function sync() {
        if (stopped) return;
        views = buildViews(); revision++;
        if (activeView && !views.some(view => viewKey(view) === activeView)) {
          cancelPending(); cancelSettingsGuard(); activeView = ''; activeViewObj = null;
          navigationError = '该插件视图已卸载或停用。';
          syncFocus();
        }
        // One initial activation only: do not steal navigation after a user's
        // close/back action, or after the plugin opens its own conversation.
        if (!opened && views.length) { opened = true; open(views[0], false); }
        for (const listener of listeners) listener();
      }
      function View() {
        React.useSyncExternalStore(listener => {listeners.add(listener); return () => listeners.delete(listener);}, () => revision);
        const canFocus = activeViewObj && (activeViewObj.slot === 'main' || activeViewObj.slot === 'settings.section' || activeViewObj.slot === 'settings.remote.tab');
        return h('div', {role: 'region', 'aria-label': 'Plugin views', style: {pointerEvents:'auto',position:'fixed',right:12,bottom:12,zIndex:10000,maxWidth:'min(640px,90vw)',padding:'8px 12px',border:'1px solid var(--dsw-alias-border-l3,var(--border,#d0d7de))',borderRadius:8,background:'var(--dsw-alias-bg-elevated,var(--bg,#fff))',color:'var(--dsw-alias-label-primary,var(--text,#1f2328))',font:'12px system-ui',boxShadow:'0 2px 10px rgba(0,0,0,0.08)'}},
          h('span', null, plugin + ' · '),
          navigationError ? h('span', {role:'status'}, navigationError) : null,
          views.length ? views.map(view => h('button', {key: viewKey(view),onClick:()=>open(view, true),style:{margin:'0 4px'}},view.label))
            : h('span', {role:'status'}, '此插件尚无可独立打开的视图。会话内插槽需要在相应会话中使用。'),
          canFocus ? h('button', {key:'__whale-focus-toggle',onClick:()=>{userInteracted=true;focusShell=!focusShell;syncFocus();changed();},style:{margin:'0 0 0 8px',opacity:0.85}}, focusShell ? '显示完整界面' : '仅显示插件') : null);
      }
      ctx.effect(() => {
        let styleEl = null;
        if (typeof document !== 'undefined' && document.head && typeof document.createElement === 'function') {
          styleEl = document.createElement('style');
          styleEl.setAttribute('data-whale-plugin-focus-style', '1');
          styleEl.textContent = [
            'html[data-whale-plugin-focus="main"] [data-dsh-frame]{grid-template-columns:0px minmax(0,1fr) 0px 0px!important;border-radius:0!important;padding-top:0!important}',
            'html[data-whale-plugin-focus="main"] [data-dsh-frame-region="sidebar"]{visibility:hidden!important;pointer-events:none!important;width:0!important;min-width:0!important;border:none!important;overflow:hidden!important}',
            'html[data-whale-plugin-focus="main"] [data-dsh-frame-region="center"]{grid-column:1/-1!important;border-radius:0!important;border-left:none!important}',
            'html[data-whale-plugin-focus="main"] [data-main-panel],html[data-whale-plugin-focus="main"] [data-main-panel-full]{grid-row:1/-1!important}',
            'html[data-whale-plugin-focus="main"] [data-side="sidebar"],html[data-whale-plugin-focus="main"] [data-titlebar-trailing],html[data-whale-plugin-focus="main"] [data-shell-leading-band],html[data-whale-plugin-focus="main"] [data-dshd-caption="band"],html[data-whale-plugin-focus="main"] [data-titlebar-row]{display:none!important}',
            'html[data-whale-plugin-focus="settings"] [data-dsh-frame-region="sidebar"],html[data-whale-plugin-focus="settings"] [data-side="sidebar"],html[data-whale-plugin-focus="settings"] [data-titlebar-trailing],html[data-whale-plugin-focus="settings"] [data-shell-leading-band]{visibility:hidden!important;pointer-events:none!important}',
          ].join('');
          document.head.appendChild(styleEl);
        }
        const markUser = () => { userInteracted = true; cancelSettingsGuard(); };
        if (typeof window !== 'undefined' && typeof window.addEventListener === 'function') {
          window.addEventListener('pointerdown', markUser, true);
          window.addEventListener('keydown', markUser, true);
        }
        const unlistenLayout = typeof ctx.layout?.panelInfo?.subscribe === 'function'
          ? ctx.layout.panelInfo.subscribe(syncFocus)
          : () => {};
        const unlistenSettings = typeof ctx.settingsNavigation?.subscribe === 'function'
          ? ctx.settingsNavigation.subscribe(syncFocus)
          : () => {};
        const unlisten = ctx.on('slots/changed', sync);
        const unmount = ctx.slots.inject('shell.overlay', () => ctx.slots.register({name:'shell.overlay',id:'whaleisle-plugin-view-picker'},View));
        sync();
        return () => {
          stopped = true; cancelPending(); cancelSettingsGuard();
          unlisten(); unlistenLayout(); unlistenSettings(); unmount(); listeners.clear();
          setFocusMode('');
          if (styleEl && typeof styleEl.remove === 'function') styleEl.remove();
          if (typeof window !== 'undefined' && typeof window.removeEventListener === 'function') {
            window.removeEventListener('pointerdown', markUser, true);
            window.removeEventListener('keydown', markUser, true);
          }
        };
      });
    }
    return {inject: ['modules','slots','layout','settingsNavigation','shortcuts'], apply};
  }
});
