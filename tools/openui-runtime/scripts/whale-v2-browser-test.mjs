// Real v2 viewer inside its opaque child with the generated Go host shell.
// Headless browser only, no app installation or user profile is started.
import { readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { resolve } from 'node:path';
const require=createRequire(import.meta.url);
let chromium;
for(const path of [process.env.WHALE_UI_PLAYWRIGHT,'../../../runtime/vendor.pre-rc2-backup-20261024/deepseek-harness/node_modules/.pnpm/playwright-core@1.61.1/node_modules/playwright-core']) {
 if(!path)continue;
 try{({chromium}=require(resolve(import.meta.dirname,path)));break}catch{}
}
if(!chromium)throw Error('Set WHALE_UI_PLAYWRIGHT to an installed playwright module.');
const file=resolve(process.argv[2]);
const shell=await readFile(file,'utf8');
const browser=await chromium.launch({headless:true,channel:'msedge',args:['--no-startup-window','--disable-background-networking','--no-first-run']});
try{
 const context=await browser.newContext({viewport:{width:900,height:900}});
 const page=await context.newPage();let errors=[];
 page.on('pageerror',e=>errors.push(e.message));
 page.on('console',m=>{if(m.type()==='error')console.log('Viewer console',m.text().replace(/data:[^'\s]+/g,'data:[embedded]').slice(0,500))});
 await page.route('**/*',route=>route.abort());
 await page.setContent(shell,{waitUntil:'load'});
 const child=page.frameLocator('iframe');
 await child.getByText('Whale UI live fixture',{exact:true}).waitFor();
 await child.getByLabel('Quantity').fill('7');
 await child.getByText('Total 14',{exact:true}).waitFor();
 await child.getByRole('button',{name:'Queue action',exact:true}).click();
 const events=await page.evaluate(()=>window.__whaleOpenUIDrain());
 const action=events.find(e=>e.type==='assistant');
 if(!action||action.message!=='Computed 14'||action.formState?.['$quantity']!==7)throw Error('Assistant/form bridge payload incorrect: '+JSON.stringify(events));
 const state=events.find(e=>e.type==='state');
 if(!state||state.state.format!=='whale-ui-state'||state.state.values['$quantity']!==7)throw Error('State did not persist bound value: '+JSON.stringify(events));
 await page.evaluate(e=>window.__whaleOpenUIAPIResponse({instanceId:e.instanceId,requestId:e.requestId,status:200,body:{ok:true}}),action);
 const source=await page.evaluate(()=>({instanceId:instance,program:render.program,revision:render.revision}));
 await page.evaluate(s=>window.__whaleOpenUIRender({...s,revision:s.revision+1,streaming:true,program:s.program+'\nextra = "pending"'}),source);
 await page.evaluate(s=>window.__whaleOpenUIRender({...s,revision:s.revision+2,streaming:false,program:s.program}),source);
 await child.getByText('Total 14',{exact:true}).waitFor();
 await child.getByRole('button',{name:'Copy result',exact:true}).click();
 let native=await page.evaluate(()=>window.__whaleOpenUIDrain());
 const copy=native.find(e=>e.type==='clipboard');if(!copy||copy.text!=='14')throw Error('Copy action not delivered: '+JSON.stringify(native));
 await page.evaluate(e=>window.__whaleOpenUIAPIResponse({instanceId:e.instanceId,requestId:e.requestId,status:500,body:{ok:false,error:'Fixture clipboard failure'}}),copy);
 await child.getByText('Fixture clipboard failure',{exact:false}).waitFor();
 // Specialized parser/registry, not just the old fence name.
 await page.evaluate(s=>window.__whaleOpenUIRender({instanceId:s.instanceId,revision:s.revision+3,streaming:false,language:'dsh-ui',program:JSON.stringify({title:'Panel fixture',panel:true,items:[{type:'text',content:'Published content'}]})}),source);
 await child.getByText('Published content',{exact:true}).waitFor();
 let panel;for(let i=0;i<80&&!panel;i++){await page.waitForTimeout(50);let q=await page.evaluate(()=>window.__whaleOpenUIDrain());panel=q.find(e=>e.type==='panel')}
 if(!panel||panel.operation.spec.items[0].content!=='Published content')throw Error('Panel publish not emitted');
 for(const appearance of ['light','dark']){
  await page.setViewportSize({width:390,height:500});
  await page.evaluate(appearance=>window.__previewTheme({appearance,variables:{}}),appearance);
  await page.waitForFunction(()=>window.__previewState.height>=80&&window.__previewState.height<240);
  const height=await page.evaluate(()=>window.__previewState.height);
  await page.evaluate(height=>{document.querySelector('iframe').style.height=height+'px'},height);
  await page.setViewportSize({width:390,height});
  const overflow=await child.locator('body').evaluate(el=>el.scrollWidth>el.clientWidth+1);if(overflow)throw Error('Narrow viewer overflow');
  await page.screenshot({path:resolve(file+'.'+appearance+'.png'),fullPage:true});
 }
 await page.setViewportSize({width:900,height:900});
 await page.evaluate(()=>{document.querySelector('iframe').style.height='900px'});
 // Real specialized component uses packaged Mermaid engine through bundle bridge.
 await page.evaluate(s=>window.__whaleOpenUIRender({instanceId:s.instanceId,revision:s.revision+4,streaming:false,language:'whale-ui',program:'root = Col([Title("Engine fixture","md"), Mermaid({code:"flowchart LR\\nA[Start] --> B[Done]"})])'}),source);
 await child.locator('svg').getByText('Start',{exact:true}).waitFor({timeout:30000});
 await child.locator('svg').getByText('Done',{exact:true}).waitFor({timeout:30000});
 await page.evaluate(s=>window.__whaleOpenUIRender({instanceId:s.instanceId,revision:s.revision+5,streaming:false,language:'whale-ui',program:'root = Col([Map([MapMarker("Tokyo",35.68,139.76),MapMarker("Osaka",34.69,135.50)],true,$selected,280), Text("Selected " + $selected),YouTube("dQw4w9WgXcQ","Video fixture")])\n$selected = "Tokyo"'}),source);
 await child.getByRole('button',{name:'2. Osaka',exact:true}).last().waitFor();
 await child.getByRole('button',{name:'2. Osaka',exact:true}).last().click();
 await child.getByText('Selected Osaka',{exact:true}).waitFor();
 await child.getByRole('button',{name:'Load YouTube video',exact:true}).click();
 let requests=[];for(let i=0;i<20;i++){await page.waitForTimeout(25);requests.push(...await page.evaluate(()=>window.__whaleOpenUIDrain()));if(requests.some(e=>e.type==='media')&&requests.some(e=>e.path==='/api/map-tile'))break;}
 const video=requests.find(e=>e.type==='media'&&e.media?.operation==='upsert'&&e.userGesture);
 if(!video||video.media.videoId!=='dQw4w9WgXcQ')throw Error('Native YouTube seat not requested: '+JSON.stringify(requests));
 if(!requests.some(e=>e.path==='/api/map-tile'))throw Error('Map did not use fixed raster tile bridge');
 await page.evaluate(e=>window.__whaleOpenUIAPIResponse({instanceId:e.instanceId,requestId:e.requestId,status:200,body:{ok:true,state:'native-view-created'}}),video);
 await child.getByText('The Host reports that the native YouTube seat was created.',{exact:false}).waitFor({timeout:2000});
 // Reused Image and HtmlView execute through the original prepared resources.
 await page.evaluate(s=>{const html="<!doctype html><html><body><button id='go'>Inside action</button><p id='result'>Before</p><script>document.getElementById('go').addEventListener('click',()=>{document.getElementById('result').textContent='Clicked'})</script></body></html>";window.__whaleOpenUIRender({instanceId:s.instanceId,revision:s.revision+6,streaming:false,language:'whale-ui',program:'root = Col([Image("https://images.example.test/fixture.png","Bridge image"),HtmlView('+JSON.stringify(html)+',120,"Local HTML")])'})},source);
 let imageRequest;for(let i=0;i<40&&!imageRequest;i++){await page.waitForTimeout(25);let q=await page.evaluate(()=>window.__whaleOpenUIDrain());imageRequest=q.find(e=>e.path==='/api/image')}
 if(!imageRequest)throw Error('Image bypassed fixed public image bridge');
 await page.evaluate(e=>{const canvas=document.createElement('canvas');canvas.width=32;canvas.height=32;canvas.getContext('2d').fillRect(0,0,32,32);window.__whaleOpenUIAPIResponse({instanceId:e.instanceId,requestId:e.requestId,status:200,body:{dataUrl:canvas.toDataURL('image/png'),source:'https://images.example.test/fixture.png'}})},imageRequest);
 await child.getByAltText('Bridge image',{exact:true}).waitFor({state:'visible'});
 await child.getByAltText('Bridge image',{exact:true}).evaluate(el=>el.decode());
 await child.getByText('HtmlView is unavailable:',{exact:false}).waitFor();
 await page.evaluate(()=>window.__whaleOpenUIHtmlViewSupport(true));
 const htmlChild=child.frameLocator('iframe[title="Local HTML"]');
 await htmlChild.getByRole('button',{name:'Inside action'}).click();await htmlChild.getByText('Clicked',{exact:true}).waitFor();
 const large='root = Col([Text("Large program visible")])\n#'+ 'x'.repeat(210000);
 await page.evaluate((s)=>window.__whaleOpenUIRender({instanceId:s.instanceId,revision:s.revision+7,streaming:false,language:'whale-ui',program:s.large}),{...source,large});
 await child.getByText('Large program visible',{exact:true}).waitFor();
 const largeEvents=await page.evaluate(()=>window.__whaleOpenUIDrain());const resolved=largeEvents.find(e=>e.type==='resolved');if(!resolved||resolved.program!==''||resolved.revision!==source.revision+7)throw Error('Large program completion exceeded event budget');
 if(errors.length)throw Error('Viewer browser errors: '+errors.join('\n'));
 console.log('PASS real opaque v2 viewer: edits/calculation/state, assistant/clipboard receipts, streaming preservation, dsh-ui panel publish, offline Mermaid, Map selected binding/tile bridge, native YouTube request/reply, 210KiB program completion, 390px light/dark natural height.');
 await context.close();
}finally{await browser.close()}
