'use strict';
(function(root){
  const clamp=(n,a,b)=>Math.min(Math.max(a,b),Math.max(a,n));
  function anchor(rect,size,area,gap=8){
    const x=clamp(rect.X+rect.Width/2-size.width/2,area.X+4,area.X+area.Width-size.width-4);
    let y=rect.Y-size.height-gap;if(y<area.Y+4)y=rect.Y+rect.Height+gap;
    return {x:Math.round(x),y:Math.round(clamp(y,area.Y+4,area.Y+area.Height-size.height-4))};
  }
  function create({body:petBody}={}){
    let layout=null,chat=null;const observed=new WeakSet();
    const overlap=(a,b)=>a.x<b.x+b.width+8&&a.x+a.width+8>b.x&&a.y<b.y+b.height+8&&a.y+a.height+8>b.y;
    function placeSurfaces(){
      if(!layout)return;
      const area=layout.Area?.Width>0?layout.Area:{X:0,Y:0,Width:innerWidth,Height:innerHeight};
      const rect=petBody?petBody.anchorRect(layout):layout,reserved=[];
      if(chat?.Width>0)reserved.push({x:chat.X,y:chat.Y,width:chat.Width,height:chat.Height});
      const constrain=(x,y,size)=>({x:Math.round(clamp(x,area.X+4,area.X+area.Width-size.width-4)),y:Math.round(clamp(y,area.Y+4,area.Y+area.Height-size.height-4)),width:size.width,height:size.height});
      for(const id of ['pet-status-card','pet-look-bubble','pet-notifications','pet-bubble','render-notice']){
        const node=document.getElementById(id);if(!node)continue;
        if(!observed.has(node)){observed.add(node);observer.observe(node);}
        node.style.visibility='';
        if(node.hidden||(!node.textContent.trim()&&!node.querySelector('img')))continue;
        node.style.maxWidth=Math.max(0,area.Width-8)+'px';node.style.maxHeight=Math.max(0,area.Height-8)+'px';
        const size=node.getBoundingClientRect(),above=anchor(rect,size,area);
        // The status dashboard belongs beside the character, with a left-side
        // flip near the work-area edge. Speech and task feedback use head space.
        const sideX=rect.X+rect.Width+8+size.width<=area.X+area.Width-4?rect.X+rect.Width+8:rect.X-size.width-8;
        const candidates=(id==='pet-status-card'?[constrain(sideX,rect.Y,size),constrain(sideX,rect.Y+rect.Height-size.height,size)]:[constrain(above.x,above.y,size)]);
        for(const r of reserved)candidates.push(constrain(r.x,r.y-size.height-8,size),constrain(r.x,r.y+r.height+8,size),constrain(r.x-size.width-8,r.y,size),constrain(r.x+r.width+8,r.y,size));
        candidates.push(constrain(rect.X-size.width-8,rect.Y,size),constrain(rect.X+rect.Width+8,rect.Y,size));
        const point=candidates.find(p=>!reserved.some(r=>overlap(p,r)))||(id==='pet-status-card'?candidates[0]:null);
        if(!point){node.style.visibility='hidden';continue;}
        node.style.left=point.x+'px';node.style.top=point.y+'px';reserved.push(point);
      }
    }
    function update(next,nextChat){
      if(!next||!(next.Width>0)||!(next.Height>0))return;layout=next;
      if(arguments.length>1)chat=nextChat;
      for(const [key,value]of Object.entries({x:next.X,y:next.Y,width:next.Width,height:next.Height}))document.body.style.setProperty('--pet-'+key,value+'px');
      placeSurfaces();
    }
    const observer=new ResizeObserver(placeSurfaces),visibility=new MutationObserver(placeSurfaces);
    visibility.observe(document.body,{subtree:true,childList:true,attributes:true,attributeFilter:['hidden']});
    try{update(JSON.parse(document.body.dataset.layout||'{}'));}catch{}
    root.addEventListener('pagehide',()=>{observer.disconnect();visibility.disconnect();},{once:true});
    return {update,refresh:placeSurfaces};
  }
  root.WhalePetLayout={anchor,create};
})(window);
