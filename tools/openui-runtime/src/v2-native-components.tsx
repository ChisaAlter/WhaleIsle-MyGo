// Reuse the existing network-free raster map and typed native media seat with
// the owned v2 interpreter's React and field hooks. No second React or SDK store.
import React from 'react';
import { mapComponents } from './components/map';
import { mediaComponents } from './components/youtube';
import { MediaContext,HtmlViewSupportContext,ProgramRevisionContext } from './capabilities';
import { imageComponents } from './components/images';
import { htmlComponents } from './components/html-view';
import { configureResourceTransport,receiveResourceResponse } from './resources';
import { configureMediaTransport,updateMediaContext,receiveMediaResponse,closeMedia } from './media';
const host=(window as any).__whaleMyGo;
let current={enabled:false,streaming:false,revision:0,htmlView:false};
configureResourceTransport(host.post);
configureMediaTransport(host.post);
host.nativeReceive=(data:any)=>{
 if(data.type==='api-response')receiveResourceResponse(data);
 if(data.type==='media-response')receiveMediaResponse(data);
 if(data.type==='render'){current={enabled:data.media===true,streaming:data.streaming,revision:data.revision,htmlView:data.htmlView===true};updateMediaContext(current);}
};
host.register([...mapComponents,...mediaComponents,...imageComponents,...htmlComponents].map(c=>({name:c.name,component:(args:any)=>{
 const props=c.name==='Map'?{...args.props,markers:(args.props.markers??[]).map((m:any)=>m?.typeName==='MapMarker'?m:{typeName:'MapMarker',props:m?.props??m})}:args.props;
 const element=React.createElement(c.component,{...args,props});
 const media=c.name==='YouTube'?React.createElement(MediaContext.Provider,{value:{enabled:current.enabled,revision:current.revision}},element):element;
 return React.createElement(ProgramRevisionContext.Provider,{value:current.revision},React.createElement(HtmlViewSupportContext.Provider,{value:current.htmlView},media));
}})));
window.addEventListener('pagehide',closeMedia,{once:true});
