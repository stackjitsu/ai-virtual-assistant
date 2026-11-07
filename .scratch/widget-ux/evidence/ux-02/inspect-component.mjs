// Scoped Design Taste inspection of the real embedded widget, not the sample
// host or an uninitialized standalone iframe. Reuse its real-browser regression.
import fs from 'node:fs';
import { probe } from '/home/me/.agents/skills/design-taste-frontend/scripts/lib/probe.mjs';
const directory = process.argv[3];
let source = fs.readFileSync('web/tests/widget-visual.browser.mjs', 'utf8');
source = source.replace('    await screenshot(`${name}-ready`);', `
    const metrics = await evaluate(f, ${JSON.stringify(`(${probe.toString()})()`)});
    fs.writeFileSync(directory + '/' + name + '-taste-probe.json', JSON.stringify(metrics, null, 2) + '\\n');
    const component = await evaluate(f, \`(()=>{
      const b=document.querySelector('#chat-close'),h=document.querySelector('.chat-heading'),r=b.getBoundingClientRect(),s=getComputedStyle(b);
      return {label:b.getAttribute('aria-label'),width:r.width,height:r.height,headingGap:r.left-h.getBoundingClientRect().right,
        color:s.color,background:s.backgroundColor,outline:s.outline,headerBackground:getComputedStyle(document.querySelector('.chat-header')).backgroundColor,
        iframeWidth:innerWidth,iframeHeight:innerHeight,headingLines:document.querySelector('h1').getBoundingClientRect().height/parseFloat(getComputedStyle(document.querySelector('h1')).lineHeight)};
    })()\`);
    fs.writeFileSync(directory + '/' + name + '-component.json', JSON.stringify(component, null, 2) + '\\n');
    await screenshot(\`${'${name}'}-ready\`);`);
await import('data:text/javascript;base64,' + Buffer.from(source).toString('base64'));
