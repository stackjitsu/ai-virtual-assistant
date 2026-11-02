(() => {
  'use strict';
  const script = document.currentScript;
  if (!script) return;
  const validOrigin = (value) => {
    try {
      const url = new URL(value);
      return url.origin === value && !url.username && !url.password
        && (url.protocol === 'https:' || (url.protocol === 'http:'
          && ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname)));
    } catch { return false; }
  };
  const context = {};
  for (const [attribute, key] of [['data-page-path', 'page_path'], ['data-locale', 'locale']]) {
    if (script.hasAttribute(attribute)) context[key] = script.getAttribute(attribute);
  }
  let origin;
  try { origin = script.getAttribute('data-assistant-origin') ?? new URL(script.src).origin; }
  catch { origin = ''; }
  const path = context.page_path;
  const locale = context.locale;
  const validContext = (path === undefined || (path.startsWith('/')
    && new TextEncoder().encode(path).length <= 256 && !path.includes('//')
    && !/[?#\\\u0000-\u001f\u007f-\u009f]/u.test(path)))
    && (locale === undefined || (locale.length <= 24 && /^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8}){0,2}$/u.test(locale)));
  const root = document.createElement('section');
  root.className = 'assistant-embed';
  root.setAttribute('aria-label', 'Support assistant');
  if (!validOrigin(origin) || !validOrigin(window.location.origin) || !validContext) {
    const error = document.createElement('p');
    error.setAttribute('role', 'alert');
    error.textContent = 'The assistant could not start: check the embed origin and page context.';
    root.append(error);
    document.body.append(root);
    return;
  }
  const stylesheet = document.createElement('link');
  stylesheet.rel = 'stylesheet';
  stylesheet.href = `${origin}/widget/loader.css`;
  document.head.append(stylesheet);
  const launcher = document.createElement('button');
  launcher.type = 'button';
  launcher.className = 'assistant-launcher';
  launcher.textContent = 'Ask the assistant';
  launcher.setAttribute('aria-label', 'Open support assistant');
  launcher.setAttribute('aria-expanded', 'false');
  const frame = document.createElement('iframe');
  frame.className = 'assistant-frame';
  frame.title = 'AI support assistant chat';
  frame.setAttribute('sandbox', 'allow-scripts allow-same-origin');
  frame.setAttribute('referrerpolicy', 'no-referrer');
  frame.height = '600';
  frame.hidden = true;
  let mounted = false;
  let initialized = false;
  const knownKeys = (data, expected) => data && typeof data === 'object' && !Array.isArray(data)
    && Object.keys(data).length === expected.length && expected.every((key) => Object.hasOwn(data, key));
  const close = () => {
    if (frame.hidden) return;
    frame.hidden = true;
    launcher.setAttribute('aria-expanded', 'false');
    launcher.setAttribute('aria-label', 'Open support assistant');
    launcher.textContent = 'Ask the assistant';
    launcher.focus();
  };
  window.addEventListener('message', (event) => {
    if (!mounted || event.source !== frame.contentWindow || event.origin !== origin) return;
    const data = event.data;
    if (knownKeys(data, ['type', 'version']) && data.type === 'assistant:ready' && data.version === 1 && !initialized) {
      initialized = true;
      frame.contentWindow.postMessage({ type: 'assistant:init', version: 1, context }, origin);
    } else if (initialized && knownKeys(data, ['type', 'version', 'height'])
      && data.type === 'assistant:resize' && data.version === 1 && Number.isFinite(data.height)) {
      frame.height = String(Math.max(300, Math.min(720, Math.round(data.height))));
    } else if (initialized && knownKeys(data, ['type', 'version'])
      && data.type === 'assistant:close' && data.version === 1) {
      close();
    }
  });
  launcher.addEventListener('keydown', (event) => {
    if (event.key !== 'Escape' || frame.hidden) return;
    event.preventDefault();
    close();
  });
  launcher.addEventListener('click', () => {
    if (!mounted) {
      mounted = true;
      frame.src = `${origin}/widget/widget.html#parent_origin=${encodeURIComponent(window.location.origin)}`;
      root.prepend(frame);
    }
    if (!frame.hidden) { close(); return; }
    frame.hidden = false;
    launcher.setAttribute('aria-expanded', String(!frame.hidden));
    launcher.setAttribute('aria-label', frame.hidden ? 'Open support assistant' : 'Close support assistant');
    launcher.textContent = frame.hidden ? 'Ask the assistant' : 'Close assistant';
  });
  root.append(launcher);
  document.body.append(root);
})();
