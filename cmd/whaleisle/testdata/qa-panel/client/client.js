/* QA fixture: a real third-party client plugin registering a left-sidebar
 * panel and its main view through the ordinary cordis slot loader. */
window.__ModuleLoader__.load({
  id: 'qa-panel',
  factory(require) {
    const React = require('react');
    const h = React.createElement;
    function Icon() { return h('span', { className: 'qa-panel-glyph', 'data-qa': 'qa-panel-icon' }, 'P'); }
    function Page() { return h('div', { className: 'qa-panel-page', role: 'main', 'data-qa': 'qa-panel-view' }, 'QA plugin panel view'); }
    function apply(ctx) {
      ctx.slots.inject('sidebar.panellist', () => ctx.slots.register({ name: 'sidebar.panellist', id: 'qa-panel', order: 60, label: 'QA Panel', locale: 'qa-panel' }, Icon));
      ctx.slots.inject('main', () => ctx.slots.register({ name: 'main', key: 'qa-panel' }, Page));
    }
    const module = { exports: {} };
    module.exports.name = 'qa-panel';
    module.exports.inject = ['slots'];
    module.exports.apply = apply;
    return module.exports;
  }
});
