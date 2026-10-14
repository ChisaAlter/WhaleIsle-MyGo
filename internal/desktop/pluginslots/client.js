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
    function apply(ctx) {
      const plugin = new URLSearchParams(location.search).get('desktopPlugin');
      if (!plugin) return;
      let stopped = false, opened = false, revision = 0, views = [], navigationError = '', cancelPending = () => {};
      const listeners = new Set();
      const changed = () => { revision++; for (const listener of listeners) listener(); };
      const entries = slot => ctx.slots.entriesOfSlot(slot);
      const own = entry => entry.options.registrant === plugin;
      const buildViews = () => {
        const rows = [];
        for (const slot of ['main', 'settings.section', 'settings.remote.tab', 'sidebar.page']) {
          for (const entry of entries(slot).filter(own)) {
            const options = entry.options;
            const id = options.key || options.id;
            if (!id || (slot === 'main' && id === 'conversation')) continue;
            const tab = slot === 'sidebar.page' ? entries('sidebar.nav.tab').find(row => row.options.id === id) : null;
            rows.push({slot, id, label: label(tab?.options || options)});
          }
        }
        // A plugin may contribute an item inside an existing Settings section.
        // Follow the section owner's declared child graph rather than guessing
        // a section name or rendering an entry without its owner props.
        const hasContribution = (names, visited = new Set()) => names.some(name => {
          if (visited.has(name)) return false;
          visited.add(name);
          return entries(name).some(entry => own(entry) || hasContribution(Object.keys(entry.options.children || {}), visited));
        });
        for (const section of entries('settings.section')) {
          const options = section.options;
          if (options.id && !rows.some(row => row.slot === 'settings.section' && row.id === options.id)
            && hasContribution(Object.keys(options.children || {}))) {
            rows.push({slot:'settings.section',id:options.id,label:label(options)});
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
      function open(view) {
        cancelPending(); navigationError = '';
        try {
        if (view.slot === 'main') ctx.layout.selectPanel(view.id);
        else if (view.slot === 'settings.section') ctx.settingsNavigation.open(view.id);
        else if (view.slot === 'settings.remote.tab') {
          ctx.settingsNavigation.open('remote');
          cancelPending = activateTab(view);
        } else cancelPending = activateTab(view);
        } catch { navigationError = '该插件视图当前不可用，请刷新或重新启用插件。'; }
        changed();
      }
      function sync() {
        if (stopped) return;
        views = buildViews(); revision++;
        // One initial activation only: do not steal navigation after a user's
        // close/back action, or after the plugin opens its own conversation.
        if (!opened && views.length) { opened = true; open(views[0]); }
        for (const listener of listeners) listener();
      }
      function View() {
        React.useSyncExternalStore(listener => {listeners.add(listener); return () => listeners.delete(listener);}, () => revision);
        return h('div', {role: 'region', 'aria-label': 'Plugin views', style: {pointerEvents:'auto',position:'fixed',right:12,bottom:12,zIndex:10000,maxWidth:'min(640px,90vw)',padding:'8px 12px',border:'1px solid var(--border,#777)',borderRadius:8,background:'var(--bg,#fff)',color:'var(--text,#222)',font:'12px system-ui'}},
          h('span', null, plugin + ' · '),
          navigationError ? h('span', {role:'status'}, navigationError) : null,
          views.length ? views.map(view => h('button', {key: view.slot+':'+view.id,onClick:()=>open(view),style:{margin:'0 4px'}},view.label))
            : h('span', {role:'status'}, '此插件尚无可独立打开的视图。会话内插槽需要在相应会话中使用。'));
      }
      ctx.effect(() => {
        const unlisten = ctx.on('slots/changed', sync);
        const unmount = ctx.slots.inject('shell.overlay', () => ctx.slots.register({name:'shell.overlay',id:'whaleisle-plugin-view-picker'},View));
        sync();
        return () => { stopped = true; cancelPending(); unlisten(); unmount(); listeners.clear(); };
      });
    }
    return {inject: ['slots','layout','settingsNavigation'], apply};
  }
});
