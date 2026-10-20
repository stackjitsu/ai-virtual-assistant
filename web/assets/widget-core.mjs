import {
  keys, parentOrigin, retrySeconds, validCreation, validHistory, validInit,
  validOrigin, validText, validTurn,
} from './protocol.mjs';

// Public construction takes a browser-like platform, not credentials. Conversation
// capability, pending request and transcript remain inside this iframe closure.
export function mountWidget(platform, { timeoutMs = 25000, handshakeMs = 10000 } = {}) {
  const { document } = platform;
  const element = (id) => document.getElementById(id);
  const ui = Object.fromEntries(['status', 'error', 'retry', 'older', 'messages', 'question', 'send', 'transmission', 'message-form']
    .map((id) => [id, element(id)]));
  const origin = platform.location.origin;
  const parent = parentOrigin(platform.location.hash);
  let initialized = false;
  let stopped = false;
  let conversation = null;
  let context = {};
  let pending = null;
  let nextBefore = null;
  let busy = true;
  let historyReady = false;
  let retryAction = null;
  let cooldownUntil = 0;
  let cooldownTimer = null;
  let handshakeTimer = null;
  let activeController = null;
  const messages = new Map();
  // Twenty maximum-size messages plus escaped JSON and source snapshots fit
  // under 1 MiB; a 256 KiB cap could reject a legitimate bounded history page.
  const responseLimit = 1048576;

  ui.transmission.textContent = `Questions and recent chat context go to ${origin}; bounded excerpts go to OpenAI to compose AI answers.`;
  const resize = () => {
    if (!parent || stopped) return;
    platform.parent.postMessage({ type: 'assistant:resize', version: 1,
      height: Math.max(300, Math.min(720, document.documentElement.scrollHeight)) }, parent);
  };
  function controls() {
    ui.question.disabled = busy || !conversation || !historyReady || pending !== null || stopped;
    ui.send.disabled = ui.question.disabled;
    ui['message-form'].setAttribute('aria-busy', String(busy));
    ui.retry.hidden = !retryAction || stopped;
    ui.retry.disabled = busy || Date.now() < cooldownUntil;
    ui.older.hidden = nextBefore === null;
    ui.older.disabled = busy || pending !== null || stopped;
    resize();
  }
  function status(text) { ui.status.textContent = text; }
  function error(text) {
    ui.error.textContent = text;
    ui.error.hidden = !text;
    resize();
  }
  function terminal(text) {
    stopped = true;
    busy = false;
    retryAction = null;
    error(text);
    status('Chat unavailable.');
    controls();
  }
  const node = (tag, text, className) => {
    const result = document.createElement(tag);
    if (text !== undefined) result.textContent = text;
    if (className) result.className = className;
    return result;
  };
  function render() {
    const nodes = [];
    for (const message of [...messages.values()].sort((left, right) => left.seq - right.seq)) {
      const item = node('li', undefined, `message message-${message.role}`);
      item.append(node('span', message.role === 'user' ? 'You' : (message.status === 'answered' ? 'AI answer' : 'Clarification · limited knowledge'), 'message-label'));
      item.append(node('p', message.text));
      if (message.sources.length) {
        const sources = node('ul', undefined, 'sources');
        sources.setAttribute('aria-label', 'Approved sources');
        for (const source of message.sources) {
          const row = node('li');
          if (source.url) {
            const link = node('a', source.title);
            link.href = source.url;
            link.rel = 'noopener noreferrer';
            row.append(link);
          } else row.textContent = source.title;
          sources.append(row);
        }
        item.append(sources);
        item.append(node('p', 'Source links replace this frame. Copy a link address to keep this guest chat.', 'source-note'));
      }
      nodes.push(item);
    }
    if (pending && ![...messages.values()].some((message) => message.request_id === pending.id)) {
      const item = node('li', undefined, 'message message-user');
      item.append(node('span', 'You · awaiting confirmation', 'message-label'), node('p', pending.text));
      nodes.push(item);
    }
    ui.messages.replaceChildren(...nodes);
    resize();
  }
  function merge(page) {
    // A sequence is immutable. Do not partly render a conflicting server page.
    const signature = (message) => JSON.stringify([message.seq, message.request_id, message.role,
      message.text, message.created_at, message.status, message.sources.map((source) => [source.id, source.title, source.url ?? ''])]);
    for (const message of page) {
      if (messages.has(message.seq) && signature(messages.get(message.seq)) !== signature(message)) throw new Error('invalid');
    }
    for (const message of page) messages.set(message.seq, message);
  }
  function retry(action, label = 'Retry same question') {
    retryAction = action;
    ui.retry.textContent = label;
  }
  function cooldown(seconds) {
    cooldownUntil = Date.now() + seconds * 1000;
    platform.clearTimeout(cooldownTimer);
    cooldownTimer = platform.setTimeout(() => { cooldownUntil = 0; controls(); }, seconds * 1000);
  }
  async function request(path, { method = 'GET', body } = {}) {
    const controller = new AbortController();
    activeController = controller;
    const timer = platform.setTimeout(() => controller.abort(), timeoutMs);
    try {
      const headers = { Accept: 'application/json' };
      if (body !== undefined) headers['Content-Type'] = 'application/json';
      if (conversation) headers.Authorization = `Bearer ${conversation.capability}`;
      const response = await platform.fetch(new URL(path, origin).href, {
        method, headers, body: body === undefined ? undefined : JSON.stringify(body),
        credentials: 'omit', mode: 'same-origin', cache: 'no-store', redirect: 'error', signal: controller.signal,
      });
      if (!/^application\/json(?:\s*;|$)/iu.test(response.headers.get('Content-Type') ?? '')) {
        controller.abort();
        throw new Error('invalid');
      }
      // Bound streamed JSON as well as the advertised size. Do not trust a server's Content-Length.
      if (Number(response.headers.get('Content-Length')) > responseLimit) {
        controller.abort();
        throw new Error('invalid');
      }
      const reader = response.body.getReader();
      const chunks = [];
      let size = 0;
      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          size += value.byteLength;
          if (size > responseLimit) { await reader.cancel(); throw new Error('invalid'); }
          chunks.push(value);
        }
      } finally { reader.releaseLock(); }
      const buffer = new Uint8Array(size);
      let offset = 0;
      for (const chunk of chunks) { buffer.set(chunk, offset); offset += chunk.byteLength; }
      const data = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(buffer));
      return { status: response.status, data, wait: retrySeconds(response.headers.get('Retry-After')) };
    } finally {
      platform.clearTimeout(timer);
      if (activeController === controller) activeController = null;
    }
  }
  const historyPath = (before) => `/api/v1/conversations/${conversation.conversation_id}/messages?limit=20${before === undefined ? '' : `&before=${before}`}`;
  const errorCode = (response) => keys(response.data, ['error'])
    && keys(response.data.error, ['code', 'message']) && typeof response.data.error.code === 'string'
    && typeof response.data.error.message === 'string' ? response.data.error.code : '';
  const errorText = (response) => {
    const code = errorCode(response);
    if (response.status === 429 && code === 'rate_limited') return 'Too many requests. Please wait before retrying.';
    if (response.status === 409 && code === 'conversation_busy') return 'This chat is busy. Wait briefly, then retry the same question.';
    if (response.status === 503 && code === 'provider_unavailable') return 'The AI service is unavailable. Your question has not been answered. Please retry later.';
    if (response.status === 503) return 'The chat service is temporarily unavailable. Please retry later.';
    return 'The request could not be completed safely. Please retry later.';
  };
  function rejectAccess(response) {
    if (response.status === 401 || response.status === 403) {
      terminal('This chat can no longer be accessed. Reload the page to start a new guest chat.');
      return true;
    }
    return false;
  }
  async function create() {
    if (stopped || busy) return;
    busy = true;
    retryAction = null;
    error('');
    status('Connecting to a new guest chat…');
    controls();
    try {
      const response = await request('/api/v1/conversations', { method: 'POST', body: { context } });
      if (stopped) return;
      if (response.status !== 201 || !validCreation(response.data)) {
        if (rejectAccess(response)) return;
        if (response.status === 429) cooldown(response.wait);
        throw new Error('create');
      }
      conversation = response.data;
      busy = false;
      await loadHistory();
    } catch {
      if (stopped) return;
      error('The chat service could not be reached. Retry to start a new guest chat.');
      status('Not connected.');
      retry(create, 'Retry connection');
    } finally { busy = false; controls(); }
  }
  async function loadHistory(before, reconciling = false, previousError = '') {
    if (stopped || busy) return;
    busy = true;
    retryAction = null;
    error('');
    status(reconciling ? 'Checking whether your question was saved…' : 'Loading conversation…');
    controls();
    try {
      const response = await request(historyPath(before));
      if (stopped) return;
      if (rejectAccess(response)) return;
      if (response.status !== 200 || !validHistory(response.data)
        || (before !== undefined && response.data.messages.some((message) => message.seq >= before))) throw new Error('history');
      const page = response.data;
      if (reconciling && pending) {
        const pair = page.messages.filter((message) => message.request_id === pending.id);
        if (pair.length && !validTurn({ request_id: pending.id, messages: pair }, pending)) throw new Error('invalid');
        merge(page.messages);
        if (pair.length === 2) {
          pending = null;
          ui.question.value = '';
          status('Saved reply recovered from history.');
        } else {
          status('No saved reply found. Retry uses the same question ID.');
          error(previousError || 'Delivery was not confirmed. Retry the same question; do not send it as a new question.');
          retry(sendPending);
        }
      } else { merge(page.messages); status('Ready when you are.'); }
      historyReady = true;
      if (before !== undefined || messages.size === page.messages.length) nextBefore = page.next_before;
      render();
    } catch {
      if (stopped) return;
      status('History unavailable.');
      error(reconciling ? 'Delivery is uncertain and history could not be checked. New questions are paused to avoid duplicates.' : 'Conversation history could not be loaded. Please retry.');
      retry(() => loadHistory(before, reconciling, previousError), reconciling ? 'Check delivery again' : 'Retry history');
    } finally { busy = false; controls(); }
  }
  async function sendPending() {
    if (stopped || busy || !pending || Date.now() < cooldownUntil) return;
    busy = true;
    retryAction = null;
    error('');
    status('Sending your question…');
    controls();
    render();
    let reconcile = false;
    let failure = '';
    try {
      const response = await request(`/api/v1/conversations/${conversation.conversation_id}/messages`, {
        method: 'POST', body: { request_id: pending.id, text: pending.text },
      });
      if (stopped) return;
      if (rejectAccess(response)) return;
      if (response.status === 200 && validTurn(response.data, pending)) {
        merge(response.data.messages);
        pending = null;
        ui.question.value = '';
        status(response.data.messages[1].status === 'answered' ? 'AI answer received. Check the approved sources.' : 'Clarification received: approved knowledge is limited.');
        render();
      } else {
        const code = errorCode(response);
        if (response.status === 409 && code === 'conversation_full') {
          terminal('This guest chat has reached its 50-question limit. Reload to start a new chat.');
          return;
        }
        if (response.status === 409 && code === 'idempotency_conflict') {
          terminal('The question could not be safely retried. Reload to start a new guest chat.');
          return;
        }
        failure = errorText(response);
        if ((response.status === 409 && code === 'conversation_busy') || response.status === 429) cooldown(response.wait);
        // A busy turn may still commit. Every uncertain or service-failure POST
        // reconciles with GET, never an automatic second inference request.
        reconcile = response.status >= 500 || response.status === 200
          || response.status === 409 || code === '';
        if (!reconcile) { error(failure); status('Question not completed.'); retry(sendPending); }
      }
    } catch {
      reconcile = true;
      failure = 'Delivery was not confirmed. Retry the same question only after history is checked.';
    } finally { busy = false; controls(); }
    if (reconcile && !stopped) await loadHistory(undefined, true, failure);
  }
  function submitQuestion() {
    if (busy || pending || !conversation || !historyReady || stopped) return;
    const text = ui.question.value.trim();
    if (!validText(text)) {
      error('Enter a question of up to 2,000 characters and 8,000 bytes, without control characters.');
      return;
    }
    try { pending = { id: platform.crypto.randomUUID(), text }; }
    catch { terminal('Secure question IDs are unavailable in this browser. Use an up-to-date secure browser.'); return; }
    void sendPending();
  }
  // Native submission is blocked before the submit event in our deliberately
  // forms-disabled sandbox. A type=button also gets native Enter/Space clicks.
  ui.send.addEventListener('click', submitQuestion);
  ui.question.addEventListener('keydown', (event) => {
    if (event.key !== 'Enter' || (!event.ctrlKey && !event.metaKey)
      || event.isComposing || event.altKey || event.shiftKey) return;
    event.preventDefault();
    if (!event.repeat) submitQuestion();
  });
  // Defensive handling only; real User actions above never rely on submission.
  ui['message-form'].addEventListener('submit', (event) => {
    event.preventDefault();
    submitQuestion();
  });
  ui.retry.addEventListener('click', () => {
    if (!busy && Date.now() >= cooldownUntil && retryAction && !stopped) void retryAction();
  });
  ui.older.addEventListener('click', () => {
    if (!busy && !pending && nextBefore !== null) void loadHistory(nextBefore);
  });
  const onMessage = (event) => {
    if (stopped || initialized || event.source !== platform.parent || event.origin !== parent) return;
    if (!validInit(event.data)) {
      platform.clearTimeout(handshakeTimer);
      terminal('The embedding page supplied invalid context. The assistant cannot connect.');
      return;
    }
    initialized = true;
    platform.clearTimeout(handshakeTimer);
    context = { ...event.data.context };
    busy = false;
    void create();
  };
  platform.addEventListener('message', onMessage);
  platform.addEventListener('pagehide', () => {
    stopped = true;
    activeController?.abort();
    platform.clearTimeout(cooldownTimer);
    platform.clearTimeout(handshakeTimer);
    conversation = null;
    pending = null;
    messages.clear();
  });
  controls();
  if (!parent || !validOrigin(origin) || platform.parent === platform) {
    terminal('Open this assistant through its configured host page. The parent origin is missing or invalid.');
  } else {
    handshakeTimer = platform.setTimeout(() => terminal('The host connection was not verified. Check the embed configuration and reload.'), handshakeMs);
    platform.parent.postMessage({ type: 'assistant:ready', version: 1 }, parent);
  }
}
