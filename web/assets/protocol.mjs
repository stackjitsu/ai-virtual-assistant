// Boundary validation is shared by the browser controller and Node tests.
const encoder = new TextEncoder();
export const bytes = (text) => encoder.encode(text).length;
const record = (value) => value !== null && typeof value === 'object' && !Array.isArray(value);
export const keys = (value, required, optional = []) => record(value)
  && required.every((key) => Object.hasOwn(value, key))
  && Object.keys(value).every((key) => [...required, ...optional].includes(key));
const controls = /[\u0000-\u001f\u007f-\u009f]/u;
const badText = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/u;
export function validOrigin(value) {
  if (typeof value !== 'string' || controls.test(value)) return false;
  try {
    const url = new URL(value);
    return url.origin === value && !url.username && !url.password
      && (url.protocol === 'https:' || (url.protocol === 'http:'
        && ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname)));
  } catch { return false; }
}
export function validContext(value) {
  if (!keys(value, [], ['page_path', 'locale'])) return false;
  if (Object.hasOwn(value, 'page_path')) {
    const path = value.page_path;
    if (typeof path !== 'string' || !path.startsWith('/') || bytes(path) > 256
      || /[?#\\]/u.test(path) || path.includes('//') || controls.test(path)) return false;
  }
  return !Object.hasOwn(value, 'locale') || (typeof value.locale === 'string'
    && bytes(value.locale) <= 24 && /^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8}){0,2}$/u.test(value.locale));
}
export function parentOrigin(hash) {
  const params = new URLSearchParams(hash.replace(/^#/u, ''));
  const origin = params.get('parent_origin');
  return [...params.keys()].length === 1 && validOrigin(origin) ? origin : null;
}
export const validInit = (data) => keys(data, ['type', 'version', 'context'])
  && data.type === 'assistant:init' && data.version === 1 && validContext(data.context);
export function validText(text, limit = 2000, byteLimit = 8000) {
  return typeof text === 'string' && text.trim().length > 0
    && [...text].length <= limit && bytes(text) <= byteLimit && !badText.test(text)
    && !/[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/u.test(text);
}
const timestamp = (value) => typeof value === 'string'
  && /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,9})?Z$/u.test(value)
  && Number.isFinite(Date.parse(value));
export const validRequestID = (value) => typeof value === 'string' && /^[A-Za-z0-9_-]{8,64}$/u.test(value);
export function validSource(source) {
  if (!keys(source, ['id', 'title'], ['url']) || typeof source.id !== 'string'
    || !/^[a-z0-9][a-z0-9_-]{0,63}$/u.test(source.id)
    || !validText(source.title, 160, 640)) return false;
  if (!Object.hasOwn(source, 'url') || source.url === '') return true;
  if (typeof source.url !== 'string' || bytes(source.url) > 2048 || controls.test(source.url)) return false;
  try {
    const url = new URL(source.url);
    return url.protocol === 'https:' && !url.username && !url.password;
  } catch { return false; }
}
export function validMessage(message) {
  if (!keys(message, ['seq', 'request_id', 'role', 'text', 'created_at', 'status', 'sources'])
    || !Number.isSafeInteger(message.seq) || message.seq < 1 || message.seq > 100
    || !validRequestID(message.request_id) || !timestamp(message.created_at)
    || !Array.isArray(message.sources) || message.sources.length > 3
    || !message.sources.every(validSource)
    || new Set(message.sources.map((source) => source.id)).size !== message.sources.length) return false;
  if (message.role === 'user') return message.seq % 2 === 1 && message.status === 'submitted'
    && message.sources.length === 0 && validText(message.text);
  return message.role === 'assistant' && message.seq % 2 === 0
    && validText(message.text, 4000, 16000)
    && ((message.status === 'answered' && message.sources.length > 0)
      || (message.status === 'needs_clarification' && message.sources.length === 0));
}
export function validHistory(page) {
  return keys(page, ['messages', 'next_before']) && Array.isArray(page.messages)
    && page.messages.length <= 20 && page.messages.every(validMessage)
    && page.messages.every((message, index) => index === 0 || message.seq === page.messages[index - 1].seq + 1)
    && new Set(page.messages.filter((message) => message.role === 'user').map((message) => message.request_id)).size
      === page.messages.filter((message) => message.role === 'user').length
    && new Set(page.messages.filter((message) => message.role === 'assistant').map((message) => message.request_id)).size
      === page.messages.filter((message) => message.role === 'assistant').length
    && page.messages.every((message, index) => index === 0 || message.seq !== page.messages[index - 1].seq + 1
      || message.role !== 'assistant' || message.request_id === page.messages[index - 1].request_id)
    && (page.next_before === null || (page.messages.length > 0
      && page.next_before === page.messages[0].seq && page.next_before > 1));
}
export function validTurn(turn, pending) {
  if (!keys(turn, ['request_id', 'messages']) || turn.request_id !== pending.id
    || !Array.isArray(turn.messages) || turn.messages.length !== 2 || !turn.messages.every(validMessage)) return false;
  const [user, assistant] = turn.messages;
  return user.role === 'user' && assistant.role === 'assistant' && assistant.seq === user.seq + 1
    && user.request_id === pending.id && assistant.request_id === pending.id && user.text === pending.text;
}
export const validCreation = (value) => keys(value, ['conversation_id', 'capability', 'created_at'])
  && typeof value.conversation_id === 'string' && /^[a-f0-9]{32}$/u.test(value.conversation_id)
  && typeof value.capability === 'string' && /^[a-f0-9]{64}$/u.test(value.capability) && timestamp(value.created_at);
// Escalation mirrors follow the additive slice-2 contract exactly; the server
// stays authoritative and these bounds only fail closed for the visitor.
const emailLocalPart = /^[a-z0-9]+(?:[._%+-][a-z0-9]+)*$/u;
const emailDomainPart = /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/u;
export const canonicalEmail = (raw) => (typeof raw === 'string' ? raw.trim().toLowerCase() : '');
export function validEscalationEmail(raw) {
  if (typeof raw !== 'string' || bytes(raw) > 254) return false;
  const email = raw.trim().toLowerCase();
  const at = email.indexOf('@');
  if (at < 1) return false;
  const local = email.slice(0, at);
  const domain = email.slice(at + 1);
  return bytes(local) <= 64 && bytes(domain) <= 253
    && emailLocalPart.test(local) && emailDomainPart.test(domain);
}
export function validEscalationNote(raw) {
  if (typeof raw !== 'string') return false;
  const note = raw.trim();
  return note === '' || validText(note, 1000, 2000);
}
export function validEscalationTicket(value) {
  return keys(value, ['ticket_id', 'recorded_at', 'delivery_outcome', 'message'])
    && /^[a-f0-9]{32}$/u.test(value.ticket_id) && timestamp(value.recorded_at)
    && value.delivery_outcome === 'delivery_not_wired'
    && typeof value.message === 'string' && value.message.length > 0;
}
export const validRatingComment = (raw) => typeof raw === 'string'
  && (raw.trim() === '' || validText(raw.trim(), 1000, 2000));
const ratingTimestamp = (value) => timestamp(value)
  && new Date(value).toISOString().slice(0, 19) === value.slice(0, 19);
export const validRatingResult = (value) => keys(value, ['rating', 'comment', 'rated_at'])
  && Number.isInteger(value.rating) && value.rating >= 1 && value.rating <= 5
  && validRatingComment(value.comment) && value.comment === value.comment.trim()
  && ratingTimestamp(value.rated_at);
export function retrySeconds(value) {
  if (typeof value !== 'string' || !/^\d{1,6}$/u.test(value)) return 1;
  return Math.max(1, Math.min(120, Number(value)));
}
