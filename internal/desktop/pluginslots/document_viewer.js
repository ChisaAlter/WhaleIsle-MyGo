/* Native resource-tab carrier. The original TextPreview remains the owner of
 * its live registry, renderer choice, content reads and session-scoped hooks. */
function whaleBindDocumentViewer(ctx) {
  const query = new URLSearchParams(location.search);
  const session = query.get('desktopDocumentSession');
  const path = query.get('desktopDocumentPath');
  if (!session || !path) return;
  const address = 'dsh-resource://file/session/' + encodeURIComponent(session) + '/' + path.split('/').map(encodeURIComponent).join('/');
      const style = document.createElement('style');
      style.textContent = `
        body { overflow:hidden!important; }
        body > * { visibility:hidden; }
        [data-whale-document-seat] { visibility:visible!important; position:fixed!important; inset:0!important; width:100vw!important; height:100vh!important; transform:none!important; z-index:9999!important; }
        [data-whale-document-seat] * { visibility:visible; }
        [data-whale-document-seat] [data-dockkit-strip] { display:none!important; }
        [role="menu"], [role="menu"] *, [role="tooltip"], [data-whale-document-status] { visibility:visible!important; }
        [data-whale-document-status] { position:fixed; inset:12px; z-index:10000; font:13px system-ui; background:var(--bg,#fff); color:var(--text,#222); }
      `;
      document.head.appendChild(style);
      const status = document.createElement('div');
      status.dataset.whaleDocumentStatus = '';
      status.setAttribute('role', 'status');
      status.textContent = '正在载入资源预览器 / Loading resource viewer…';
      document.body.appendChild(status);

  ctx.effect(() => () => {style.remove();status.remove();});
  ctx.inject(['uiWorkspace', 'sidebarRight', 'documentPreviews'], ready => {
    ready.effect(() => {
      let stopped = false, opened = false, opening = false;
      const recommend = () => {
        window.__whaleDocumentRecommendation = {session,path,extension:ready.documentPreviews.candidates(path).some(definition => definition.priority !== "builtin")};
        window.dispatchEvent(new Event("whale-document-registry"));
      };
      const offRegistry = ready.documentPreviews.subscribe(recommend);
      recommend();
      const sync = () => {
        if (stopped || opening) return;
        const mounted = ready.sidebarRight.mounted.getSnapshot();
        for (const seat of document.querySelectorAll('[data-whale-document-seat]')) seat.removeAttribute('data-whale-document-seat');
        if (mounted !== session) {
          status.hidden = false;
          if (opened) status.textContent = '会话已变化，请返回原生预览 / Session changed; return to native preview.';
          return;
        }
        if (!opened) {
          opening = true;
          try { opened = ready.sidebarRight.openResourceIn(session, address, {kind:'text', expand:true}); }
          catch (error) { status.textContent = String(error?.message || error); }
          finally { opening = false; }
        }
        if (!opened) return;
        const tab = ready.sidebarRight.tabsIn(session).find(tab => tab.contentId === address && tab.kind === 'text');
        const seat = [...document.querySelectorAll('[data-sidebar-right-session]')].find(node => node.dataset.sidebarRightSession === session);
        if (!tab || !seat) return;
        // Keep only this native tab's original resource occurrence on screen.
        if (!seat.querySelector(`[data-dockkit-tab="${CSS.escape(tab.id)}"][aria-selected="true"]`)) {
          opening = true;
          try { ready.sidebarRight.focus(tab.id); } finally { opening = false; }
        }
        seat.setAttribute('data-whale-document-seat', '');
        status.hidden = true;
      };
      const observer = new MutationObserver(sync);
      observer.observe(document.body, {childList:true,subtree:true});
      const offMounted = ready.sidebarRight.mounted.subscribe(sync);
      const offTabs = ready.sidebarRight.openTabs.subscribe(sync);
      // Service disposal hides this exact child. Re-provisioning creates a fresh
      // original owner; no renderer component or hook is copied by this adapter.
      ready.uiWorkspace.openSession(session);
      sync();
      return () => {
        stopped = true; observer.disconnect(); offMounted(); offTabs(); offRegistry();
        window.__whaleDocumentRecommendation = {session,path,extension:false};
        window.dispatchEvent(new Event("whale-document-registry")); status.hidden = false; status.textContent = "资源预览器不可用 / Resource viewer unavailable";
        for (const seat of document.querySelectorAll('[data-whale-document-seat]')) seat.removeAttribute('data-whale-document-seat');
      };
    });
  });
}
