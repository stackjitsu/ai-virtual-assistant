import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import { mountWidget } from '../assets/widget-core.mjs';

export class Element {
  constructor(tag) {
    this.tagName = tag.toUpperCase();
    this.children = [];
    this.attributes = {};
    this.listeners = {};
    this.hidden = false;
    this.disabled = false;
    this.value = '';
    this._text = '';
    if (tag === 'iframe') this.contentWindow = { sent: [], postMessage(data, origin) { this.sent.push({ data, origin }); } };
  }
  set innerHTML(_) { throw new Error('HTML injection sink is forbidden'); }
  get textContent() { return this._text + this.children.map((child) => child.textContent).join(''); }
  set textContent(text) { this._text = String(text); this.children = []; }
  setAttribute(key, value) { this.attributes[key] = String(value); }
  getAttribute(key) { return this.attributes[key] ?? null; }
  hasAttribute(key) { return Object.hasOwn(this.attributes, key); }
  // Focus and selector matching are observable so tests can assert where the
  // widget sends focus and whether a click landed inside the handoff region.
  focus() { this.focused = true; }
  matches(selector) { return (this.selectors ?? []).includes(selector); }
  closest(selector) { return this.matches(selector) ? this : null; }
  append(...nodes) { this.children.push(...nodes); }
  prepend(...nodes) { this.children.unshift(...nodes); }
  replaceChildren(...nodes) { this._text = ''; this.children = nodes; }
  addEventListener(type, listener) { (this.listeners[type] ??= []).push(listener); }
  dispatch(type, data = {}) {
    const event = { preventDefault() {}, ...data };
    for (const listener of this.listeners[type] ?? []) listener(event);
  }
}
export const walk = (element) => [element, ...element.children.flatMap(walk)];
export function documentFixture() {
  const ids = ['status', 'error', 'retry', 'older', 'messages', 'question', 'send', 'transmission', 'message-form',
    'escalate-toggle', 'handoff-menu', 'handoff-email', 'handoff-whatsapp', 'escalation-form', 'escalate-email', 'escalate-note', 'escalate-consent', 'escalate-error', 'escalate-outcome', 'escalate-send'];
  const tags = { form: 'form', 'escalate-toggle': 'button', 'handoff-email': 'button', 'handoff-whatsapp': 'button', 'escalate-send': 'button', 'escalate-email': 'input', 'escalate-consent': 'input', 'escalate-note': 'textarea', 'escalate-error': 'p', 'escalate-outcome': 'p' };
  const elements = Object.fromEntries(ids.map((id) => [id, new Element(tags[id] ?? 'div')]));
  elements.retry.hidden = true;
  elements.older.hidden = true;
  elements.error.hidden = true;
  elements['escalation-form'].hidden = true;
  elements['escalate-error'].hidden = true;
  elements['escalate-outcome'].hidden = true;
  const body = new Element('body');
  body.append(...Object.values(elements));
  return { body, head: new Element('head'), documentElement: { scrollHeight: 650 }, elements,
    getElementById: (id) => elements[id], createElement: (tag) => new Element(tag) };
}
export const creation = { conversation_id: 'a'.repeat(32), capability: 'b'.repeat(64), created_at: '2026-10-20T12:00:00Z' };
export const id = '12345678-1234-4123-8123-123456789abc';
export function turn(requestID = id, text = 'How do I start?', number = 1, clarification = false) {
  return { request_id: requestID, messages: [
    { seq: number * 2 - 1, request_id: requestID, role: 'user', text, created_at: '2026-10-20T12:00:01Z', status: 'submitted', sources: [] },
    { seq: number * 2, request_id: requestID, role: 'assistant', text: 'Use the fictional demo guide.', created_at: '2026-10-20T12:00:02Z',
      status: clarification ? 'needs_clarification' : 'answered', sources: clarification ? [] : [{ id: 'demo-start', title: 'Demo getting started', url: 'https://example.org/help/start' }] },
  ] };
}
export const json = (data, status = 200, headers = {}) => new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json', ...headers } });
export const emptyHistory = () => json({ messages: [], next_before: null });
export const fail = (code, status, message = 'DO NOT RENDER SERVER GARBAGE', headers = {}) => json({ error: { code, message } }, status, headers);
export const settle = async () => { for (let index = 0; index < 12; index++) await new Promise((resolve) => setImmediate(resolve)); };
export function widgetFixture(responses = [json(creation, 201), emptyHistory()], options = {}) {
  const document = documentFixture();
  const events = {};
  const requests = [];
  const sent = [];
  const timers = new Map();
  let sequence = 0;
  const parent = { postMessage(data, origin) { sent.push({ data, origin }); } };
  const platform = {
    document, parent, location: { origin: 'https://assistant.example', hash: '#parent_origin=https%3A%2F%2Fhost.example', ...options.location },
    crypto: { randomUUID: () => id },
    addEventListener(type, listener) { (events[type] ??= []).push(listener); },
    setTimeout(fn, delay) { const key = ++sequence; timers.set(key, { fn, delay }); return key; },
    clearTimeout(key) { timers.delete(key); },
    async fetch(url, init) {
      requests.push({ url, init });
      const response = responses.shift();
      if (response instanceof Error) throw response;
      if (typeof response === 'function') return response(url, init);
      if (!response) throw new Error('Unexpected fetch');
      return response;
    },
  };
  const emit = (type, data) => { for (const listener of events[type] ?? []) listener(data); };
  mountWidget(platform, options.mount);
  const init = (data = { type: 'assistant:init', version: 1, context: {} }, origin = 'https://host.example', source = parent) => emit('message', { data, origin, source });
  // Exercise the actual Send listener; native submission is unavailable in the iframe.
  const submit = (text = 'How do I start?') => { document.elements.question.value = text; document.elements.send.dispatch('click'); };
  return { platform, document, ui: document.elements, requests, sent, timers, responses, init, submit, emit };
}
export async function loaderFixture(attributes = {}, locationOrigin = 'https://host.example') {
  const document = documentFixture();
  document.body.replaceChildren();
  const script = new Element('script');
  script.src = 'https://assistant.example/widget/loader.js';
  for (const [key, value] of Object.entries(attributes)) script.setAttribute(key, value);
  document.currentScript = script;
  const listeners = {};
  const window = { location: { origin: locationOrigin }, addEventListener(type, fn) { listeners[type] = fn; } };
  const source = await readFile(new URL('../assets/loader.js', import.meta.url), 'utf8');
  vm.runInNewContext(source, { window, document, URL, TextEncoder, Number });
  const all = walk(document.body);
  const launcher = all.find((element) => element.tagName === 'BUTTON');
  const open = () => launcher.dispatch('click');
  const frame = () => walk(document.body).find((element) => element.tagName === 'IFRAME');
  const message = (data, origin = 'https://assistant.example', sourceWindow = frame()?.contentWindow) => listeners.message?.({ data, origin, source: sourceWindow });
  return { document, launcher, open, frame, message };
}
