//go:build functionalacceptance

package pluginui_test

const sessionFixtureClient = `window.__ModuleLoader__.load({id:'whaleisle-session-ui-fixture',factory(require){
  const React=require('react'),h=React.createElement;
  return{inject:['slots','sessions'],apply(ctx){ctx.effect(()=>{
    let active=()=>{};
    window.__setSessionFixture=mode=>{
      active(); active=()=>{};
      if(mode==='none')return;
      const disposers=[];
      const register=(name,options,component)=>disposers.push(ctx.slots.inject(name,()=>ctx.slots.register({name,registrant:'whaleisle-session-ui-fixture',...options},component)));
      if(mode==='header')register('conversation.session.header.actions',{id:'fixture-session-action',label:'Fixture Session action',inject:sessionId=>({rename:async()=>{
        const binding=ctx.sessions.binding(sessionId);if(!binding)throw Error('no binding');
        const result=await binding.session.rename('Fixture renamed '+sessionId);if(!result.ok)throw Error('rename refused');
        document.documentElement.dataset.fixtureRenamed=sessionId;
      }})},p=>{
        const title=p.useProjection('title',v=>v);
        React.useEffect(()=>{window.__fixtureMounts=(window.__fixtureMounts||0)+1;return()=>{window.__fixtureUnmounts=(window.__fixtureUnmounts||0)+1}},[]);
        return h('div',null,h('button',{onClick:p.rename,'data-fixture-session':p.sessionId},'Rename '+p.sessionId),h('output',{'data-fixture-title':''},title||''),h('button',{'data-fixture-draft':'',onClick:()=>{p.inputActions.setDraft('Retained fixture draft');p.inputActions.persistDraft()}},'Set browser draft'));
      });
      if(mode==='declined'){
        register('conversation.composer',{label:'Declined composer',priority:-1000,select:()=>null},()=>h('button',null,'Must never render'));
        register('conversation.chat.node',{key:'fixture-unknown-node',label:'Wrong keyed renderer'},()=>h('button',null,'Must never dispatch'));
      }
      if(mode==='composer')register('conversation.composer',{label:'Selected composer',priority:-1000,select:owner=>owner.sessionId==='fixture-session-a'?{id:owner.sessionId}:null},p=>h('button',{'data-fixture-composer':'',onClick:()=>{document.documentElement.dataset.composerClicked=p.matched.id}},'Composer takeover '+p.matched.id));
      if(mode==='null')register('conversation.session.header.actions',{id:'fixture-null',label:'Empty contribution'},()=>null);
      active=()=>disposers.reverse().forEach(dispose=>dispose());
    };
    window.__withdrawSessionFixture=()=>window.__setSessionFixture('none');
    return()=>{active();delete window.__setSessionFixture;delete window.__withdrawSessionFixture};
  })}};
}});`
