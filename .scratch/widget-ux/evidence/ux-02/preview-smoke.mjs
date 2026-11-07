import assert from 'node:assert/strict';
import fs from 'node:fs';
import { Browser, sleep } from '/home/me/.agents/skills/design-taste-frontend/scripts/lib/cdp.mjs';
const out = '.scratch/widget-ux/evidence/ux-02/preview';
process.env.TASTE_SKILL_TMP = out;
const browser = await Browser.launch({extraArgs:['--disable-background-networking','--disable-component-update','--disable-sync','--proxy-server=http://127.0.0.1:9','--proxy-bypass-list=127.0.0.1;localhost']});
const results=[];
try {
  for (const width of [1440,768,390]) {
    const page = await browser.newPage({width,height:900,mobile:width===390,reducedMotion:width===390});
    if(width===390) await page.send('Emulation.setEmulatedMedia',{features:[{name:'prefers-color-scheme',value:'dark'},{name:'prefers-reduced-motion',value:'reduce'}]});
    await page.goto('http://127.0.0.1:18102/widget/sample-host.html');
    async function click(expression) {
      const point=await page.eval(`(()=>{${expression};return {x:r.x+r.width/2,y:r.y+r.height/2}})()`);
      for(const type of ['mousePressed','mouseReleased']) await page.send('Input.dispatchMouseEvent',{type,...point,button:'left',clickCount:1});
    }
    await click("const r=document.querySelector('.assistant-launcher').getBoundingClientRect()");
    let ready=false;
    for(let n=0;n<100;n++) {
      ready=await page.eval(`(()=>{const d=document.querySelector('iframe').contentDocument;return !!d?.querySelector('#send')&&!d.querySelector('#send').disabled})()`);
      if(ready)break;
      await sleep(50);
    }
    assert.ok(ready,'preview initialized on '+width);
    await sleep(200);
    await page.screenshot({path:`${out}/preview-${width}.png`,format:'png'});
    await click("const f=document.querySelector('iframe'),o=f.getBoundingClientRect(),b=f.contentDocument.querySelector('#chat-close').getBoundingClientRect(),r={x:o.x+f.clientLeft+b.x,y:o.y+f.clientTop+b.y,width:b.width,height:b.height}");
    await sleep(100);
    assert.ok(await page.eval(`document.querySelector('iframe').hidden&&document.activeElement===document.querySelector('.assistant-launcher')`));
    assert.deepEqual(page.consoleErrors,[]);
    results.push({width,ready:true,trusted_pointer_close:true,launcher_focus:true,reduced_motion:width===390,dark_preference:width===390,exceptions:0});
    await page.close();
  }
} finally {await browser.close();}
fs.writeFileSync(`${out}/browser.json`,JSON.stringify(results,null,2)+'\n');
console.log('Preview smoke: 3/3 viewports initialized and trusted pointer close restored launcher focus');
