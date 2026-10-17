// Drift: browser client. Plain ES modules, no build step.
// The contract with the server is SPEC.md. Text from the network is only
// ever rendered with textContent, never as HTML.

import { notify, play, soundSettings, setSoundSettings } from './sounds.js';
import { readJSON, readStore, writeJSON, writeStore } from './store.js';
import { createMachine, nextState, onRefusal, refuse } from './machine.js';

// ---------------------------------------------------------------------------
// Words
// ---------------------------------------------------------------------------

const WORDS = {
    tagline: 'Talk to strangers. No sign-up, no app.',
    heroA: 'Walk in.', heroB: 'Say hi.', heroC: 'Leave no trace.',
    heroP: 'Make a card, step into the room and talk to whoever is up right now. Everyone wears their face; a member\'s real photo shows only once you both choose to get closer.',
    point1: 'No email, no phone number, no download',
    point2: 'Real photos only when you both say so',
    point3: 'Your card and chats disappear when you leave',
    hereNow: 'here now',
    makeCard: 'Make your card',
    name: 'Name', namePh: 'What should people call you?',
    age: 'Age', gender: 'I am', tags: 'Here to',
    location: 'Location', locationPh: 'City, optional',
    about: 'About', aboutPh: 'One line about you (optional)',
    photo: 'Add a photo', photoHint: 'Optional. Shown only to people you both choose to get closer to.',
    enter: 'Enter the room',
    fine: 'Adults 18+ only. Kink-friendly, consent first: block or report anyone, any time. Reports are read by real people.',
    email: 'Email', password: 'Password', signIn: 'Sign in', back: 'Back',
    room: 'The Room', lastLines: 'Only the last 20 lines are kept',
    sayRoom: 'Say something to the room...', send: 'Send', sendPhoto: 'Send a photo',
    people: 'Here now', chats: 'Chats',
    knock: 'Knock', open: 'Open chat', leave: 'Leave', you: 'you',
    nobody: 'Nobody matches right now. People drift in all night, check back in a minute.',
    noChats: 'No chats yet. Knock on someone in Here now to start one.',
    pickChat: 'Pick a chat on the left.',
    knockTitle: 'Knock on {name}',
    knockHelp: 'Your first message is a knock. Make it count: you get one until they reply.',
    knockPh: 'Write your knock...',
    knockSent: 'Knock sent',
    knockWait: 'Wait for {name} to answer. You can send more once they reply.',
    knockIn: '{name} knocked',
    knockInHelp: 'Reply to open the door. Photos wait until you both choose to get closer.',
    replyPh: 'Reply to open the door...',
    msgPh: 'Message...',
    doorOpen: 'The door is open. Get closer when you both want to.',
    gone: '{name} left. This chat ends when either of you leaves.',
    ended: '{name} left. This chat has ended.',
    mentioned: '{name} mentioned you in the room',
    newLines: '{n} new messages', newLine: '1 new message',
    knocksWaiting: '{n} knocks waiting', seeAll: 'See all', answerNewest: 'Answer newest', knocked: 'Knocked', later: 'Later',
    decline: 'Decline', declined: '{name} passed on this chat.', declinedHelp: 'No hard feelings. Plenty of people are up tonight.',
    answer: 'Answer', stateBug: 'State bug: {move}',
    gif: 'GIF', gifTitle: 'Send a GIF or photo', recent: 'Recent', gifs: 'GIFs',
    gifSearch: 'Search GIFs', upload: 'Upload', more: 'More', mention: 'Mention',
    noRecents: 'Photos and GIFs you send land here, so you can send them again in one tap.',
    gifOff: 'GIF search is not switched on for this server yet. Upload your own, or pick from Recent.',
    gifNone: 'No GIFs for that. Try another word.',
    clearRecents: 'Clear recents', viewCard: 'View card',
    sounds: 'Sounds', soundsOn: 'Notification sounds', roomSounds: 'Room chatter',
    roomSoundsHelp: 'A soft blip for every room message. Mentions always chime.',
    vibrate: 'Vibrate on phones',
    knockSounds: 'Knocks', knockSoundsHelp: 'The wooden rap when someone knocks or wants to get closer. The tray still shows them.',
    block: 'Block', report: 'Report', remove: 'Remove', close: 'Close',
    reportTitle: 'Report', reportHelp: 'What is wrong? A moderator will see exactly what you saw.',
    note: 'Anything else (optional)', submit: 'Send report', reported: 'Thanks. A moderator will look at it.',
    blocked: 'Blocked. You will not see each other again.',
    editCard: 'Edit your card', save: 'Save', saved: 'Saved',
    changePhoto: 'Change photo', removePhoto: 'Remove photo',
    filtered: 'filtered',
    desk: 'Control panel', reports: 'Reports', words: 'Banned words', audit: 'Audit log',
    backToChat: 'Back to chat',
    wordHelp: 'Changes reach every live chat within a second. Mask hides the word, Block stops the whole message.',
    wordPh: 'Word or phrase', add: 'Add', mask: 'Mask', blockWord: 'Block',
    queueEmpty: 'Nothing waiting. Quiet night.',
    dismiss: 'Dismiss', kick: 'Kick', mute30: 'Mute 30 min', ban24: 'Ban 24h', banIP: 'Ban 24h + device',
    reconnecting: 'Reconnecting...',
    leftRoom: 'You left the room.',
    welcomeBack: 'Welcome back, {name}', continueAs: 'Continue as {name}',
    pastChat: 'Past chat', pastEnded: 'This chat has ended.', knockAgain: 'Knock again',
    elsewhere: 'Drift is open in another tab.', useHere: 'Use here',
    tapToView: 'Tap to view',
    reconsiderTitle: 'Send this as it is?', reconsiderHelp: 'Some of these words can land badly. They will be masked if you send it anyway.',
    edit: 'Edit', sendAnyway: 'Send anyway',
    reportThanks: 'A moderator acted on your report. Thank you.',
    ladder: 'Getting closer', stepFace: 'Face', stepTalk: 'Talk', stepPhotos: 'Photos',
    closer: 'Get closer', closerAccept: 'Get closer too', closerWaiting: 'Waiting for {name}',
    closerAsked: '{name} wants to get closer', closeDone: 'Photos are open',
    closeBanner: 'You both chose to get closer. Photos are open.',
    talkedBefore: 'You talked {when}, {n} messages', talkedBeforeOne: 'You talked {when}, one message',
    sharedTags: 'You are both into {tags}',
    selfNote: 'Note to self', selfNotePh: 'What you talked about, for next time', selfNoteHelp: 'Only you see this, on this device.',
    tomorrow: 'Pick this up tomorrow', tomorrowMine: 'You said you would pick this up tomorrow.',
    tomorrowTheirs: '{name} would like to pick this up tomorrow.', meToo: 'Me too',
    tomorrowSaved: 'Saved. You will hear when {name} is here.', promiseHere: '{name} is here. You said you would pick it up.',
    pickUp: 'pick up',
    letter: 'Write a letter', letterHelp: 'A letter takes a few hours to arrive. One at a time.',
    letterPh: 'Dear {name},', letterSend: 'Send letter', letterSent: 'Your letter is on its way. It arrives around {time}.',
    letterIn: 'A letter from {name} arrived', letters: 'Letters', noLetters: 'No letters yet.',
    opener: 'Need an opener?',
    plain: 'Keep it plain', glowOn: 'Bring back the glow',
    aura: 'Aura', auraHelp: 'A soft glow around your face. Anyone can pick any colour.',
    knocksFrom: 'Who can knock', hideGender: 'Hide my gender from strangers',
    hideGenderHelp: 'People see it once you are talking.',
    seoTitle: 'Free chat with strangers',
    seoP: 'Drift is a free chat room for meeting strangers: random chat without registration, on your phone or laptop. Looking for an Omegle alternative or a Y99 alternative? Walk in, make a card and say hi.',
};

const GENDERS = { female: 'Woman', male: 'Man', 'non-binary': 'Non-binary', femboy: 'Femboy', couple: 'Couple', other: 'Other' };
const AURAS = { '': 'None', ember: 'Ember', rose: 'Rose', lantern: 'Lantern', peach: 'Peach', dusk: 'Dusk', sea: 'Sea' };
const KNOCKS = { anyone: 'Anyone', members: 'Members only', nobody: 'Nobody for now' };
const REASONS = { spam: 'Spam', harassment: 'Harassment', nudity: 'Nudity', violence: 'Violence', hate: 'Hate', underage: 'Under 18', scam: 'Scam', other: 'Other' };

const t = (key, vars = {}) => (WORDS[key] ?? key).replace(/\{(\w+)\}/g, (_, k) => vars[k] ?? '');
const genderLabel = (g) => GENDERS[g] ?? g;
// genderAge reads "Woman, 25", or "25" for someone who hides their gender.
const genderAge = (person) => [genderLabel(person.gender), person.age].filter(Boolean).join(', ');
// Suggestions only: people can type any tag (up to 3, 20 characters each).
// Adults-only community: common kink-scene terms between consenting adults.
// Six ideas keep the card short; any other tag can still be typed.
const TAG_IDEAS = ['just talk', 'dom', 'sub', 'switch', 'roleplay', 'kink curious'];
const MAX_TAGS = 3;

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

// The login and the card outlive the tab, so closing it and coming back is
// one tap. Past chats are kept per account, on this device only.
const TOKEN_KEY = 'drift.token';
const PROFILE_KEY = 'drift.profile';
const chatsKey = (userId) => `drift.chats.${userId}`;
const notesKey = (userId) => `drift.notes.${userId}`;
const promisesKey = (userId) => `drift.tomorrow.${userId}`;
const lettersKey = (userId) => `drift.letters.${userId}`;
const PLAIN_KEY = 'drift.plain';

// debounce runs fn once calls stop for ms, so a burst of messages costs
// one write.
function debounce(fn, ms) {
  let timer = 0;
  return (...args) => { clearTimeout(timer); timer = setTimeout(() => fn(...args), ms); };
}

// h builds DOM nodes. Strings become text nodes, so network text is inert.
function h(tag, props = {}, ...children) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(props)) {
    if (v == null || v === false) continue;
    if (k === 'class') el.className = v;
    else if (k === 'style' && typeof v === 'object') Object.assign(el.style, v);
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (k === 'value') el.value = v;
    else el.setAttribute(k, v === true ? '' : v);
  }
  append(el, children);
  return el;
}
function append(el, children) {
  for (const c of children.flat(Infinity)) {
    if (c == null || c === false) continue;
    el.append(c instanceof Node ? c : document.createTextNode(String(c)));
  }
  return el;
}
const $ = (sel, root = document) => root.querySelector(sel);

// Icons are inline SVG strings we wrote ourselves (no network content).
const ICONS = {
  photo: '<rect x="3" y="5" width="18" height="14" rx="2"/><circle cx="9" cy="10" r="1.6"/><path d="M21 16l-5-5-8 8"/>',
  send: '<path d="M5 12h14M13 6l6 6-6 6"/>',
  flag: '<path d="M5 21V4M5 4h11l-2 4 2 4H5"/>',
  x: '<path d="M6 6l12 12M18 6L6 18"/>',
  check: '<path d="M5 12l4 4 10-10"/>',
  back: '<path d="M15 6l-6 6 6 6"/>',
  shield: '<path d="M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6z"/>',
  trash: '<path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13"/>',
  more: '<circle cx="5" cy="12" r="1.3"/><circle cx="12" cy="12" r="1.3"/><circle cx="19" cy="12" r="1.3"/>',
  bell: '<path d="M6 16V11a6 6 0 0 1 12 0v5l2 2H4z"/><path d="M10 20a2 2 0 0 0 4 0"/>',
  bellOff: '<path d="M6 16V11a6 6 0 0 1 9.5-4.9M18 11v5l2 2H8"/><path d="M10 20a2 2 0 0 0 4 0M4 4l16 16"/>',
  play: '<path d="M8 5l11 7-11 7z"/>',
  door: '<path d="M6 21V4h10v17M4 21h16M13 12h.01"/>',
  at: '<circle cx="12" cy="12" r="4"/><path d="M16 12v1.5a2.5 2.5 0 0 0 5 0V12a9 9 0 1 0-3.5 7.1"/>',
  heart: '<path d="M12 20s-7-4.4-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 10c0 5.6-7 10-7 10z" fill="currentColor"/>',
  spark: '<path d="M12 3v4M12 17v4M3 12h4M17 12h4M6 6l2.5 2.5M15.5 15.5L18 18M6 18l2.5-2.5M15.5 8.5L18 6"/>',
  letter: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3 7l9 6 9-6"/>',
};
function icon(name, size = 20) {
  const span = document.createElement('span');
  span.style.display = 'inline-flex';
  span.innerHTML = `<svg aria-hidden="true" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${ICONS[name]}</svg>`;
  return span;
}

// faceURL is the pixel face drawn from a name; everyone has one. The server
// normalizes the name too, so this only keeps the browser cache tidy.
function faceURL(person) {
  if (person.avatar_url) return person.avatar_url;
  const key = (person.name || '').normalize('NFKC').trim().replace(/\s+/g, ' ').toLowerCase();
  return `/avatar/v2/${encodeURIComponent(key || 'stranger')}`;
}

// pictureStyle shows the best image the viewer may see: a member's photo
// once a door has opened (or their own), otherwise the pixel face.
function pictureStyle(person) {
  const photo = person.photo_url;
  return { backgroundImage: `url("${photo || faceURL(person)}")`, imageRendering: photo ? '' : 'pixelated' };
}

function avatar(person, size = 32) {
  const style = pictureStyle(person);
  return h('span', {
    class: 'avatar',
    'aria-hidden': 'true',
    'data-aura': person.aura || null,
    // The aura is part of the key: a moved node must not keep an old glow.
    'data-picture': `${size} ${person.aura || ''} ${style.backgroundImage}`,
    style: { width: `${size}px`, height: `${size}px`, ...style },
  });
}

// keepPictures moves already-painted pictures from the old tree into its
// rebuilt replacement. Panes are redrawn on every message; a fresh element
// decodes its photo again and flashes blank, a moved one keeps its paint.
// Nodes carrying the same data-picture render identically, so any match
// can stand in. Mutates newRoot in place: it is about to be swapped in.
function keepPictures(oldRoot, newRoot) {
  const painted = Map.groupBy(oldRoot.querySelectorAll('[data-picture]'), (el) => el.dataset.picture);
  for (const fresh of newRoot.querySelectorAll('[data-picture]')) {
    const old = painted.get(fresh.dataset.picture)?.shift();
    if (old) fresh.replaceWith(old);
  }
}

// Flags are SVG images keyed by ISO country code, never emoji.
function flag(cc) {
  if (!/^[A-Z]{2}$/.test(cc || '')) return null;
  return h('img', {
    class: 'flag', alt: cc, title: cc, loading: 'lazy', width: 18, height: 13,
    src: `/flags/${cc.toLowerCase()}.svg`, // bundled from flag-icons (MIT), see web/flags/LICENSE
  });
}

function minutesSince(unix) {
  if (!unix) return 0;
  return Math.max(0, Math.floor((Date.now() / 1000 - unix) / 60));
}
function hereFor(unix) {
  const m = minutesSince(unix);
  return m < 1 ? 'just in' : m < 60 ? `here ${m}m` : `here ${Math.floor(m / 60)}h`;
}
// One formatter for every timestamp: toLocaleTimeString builds a new one
// per call, and every feed line and bubble shows a time.
const CLOCK = new Intl.DateTimeFormat([], { hour: '2-digit', minute: '2-digit' });
function clock(unix) {
  return CLOCK.format(unix * 1000);
}

const DAY_MS = 86400000;
// sameDay compares calendar days in the viewer's time zone, not 24-hour spans.
const sameDay = (aMs, bMs) => new Date(aMs).toDateString() === new Date(bMs).toDateString();
// withinDays: unix (seconds) is less than days old at nowMs.
const withinDays = (unix, days, nowMs) => nowMs - unix * 1000 < days * DAY_MS;

const TOAST_MS = 4200;

// toast shows a note for a few seconds and returns it. Notes with the same
// key replace each other, so a burst of knocks is one toast that keeps count.
function toast(text, bad = false, key = null) {
  let el = key && $(`#toasts [data-key="${CSS.escape(key)}"]`);
  if (el) {
    el.textContent = text;
    clearTimeout(el.timer);
  } else {
    el = h('div', { class: `toast${bad ? ' bad' : ''}`, role: bad ? 'alert' : 'status', 'data-key': key }, text);
    $('#toasts').append(el);
  }
  el.timer = setTimeout(() => el.remove(), TOAST_MS);
  return el;
}

// ---------------------------------------------------------------------------
// Mentions: "@name" in the room. Names may hold single spaces, so a mention
// is matched against the names actually present, longest first.
// ---------------------------------------------------------------------------

const isWordChar = (ch) => !!ch && /[\p{L}\p{N}_]/u.test(ch);

function mentionsMe(text) {
  const me = (state.me?.name || '').toLowerCase();
  if (!me || !text) return false;
  const lower = text.toLowerCase();
  for (let i = lower.indexOf('@' + me); i !== -1; i = lower.indexOf('@' + me, i + 1)) {
    if (!isWordChar(lower[i + 1 + me.length])) return true;
  }
  return false;
}

// mentionables is who can be @mentioned (everyone here, and you), longest
// name first and lowercased once. Rebuilding and sorting it for every line
// with an "@" cost a sort of the whole room per line, so it is kept until
// someone arrives, leaves or changes name.
let mentionables = null;
function mentionNames() {
  mentionables ??= [state.me, ...state.online.values()].filter((p) => p?.name)
    .map((card) => ({ card, name: card.name, lower: card.name.toLowerCase() }))
    .sort((a, b) => b.name.length - a.name.length);
  return mentionables;
}
const forgetMentionNames = () => { mentionables = null; };

// richText splits text into plain runs and highlighted mentions of people
// who are here (or you). The result is still text nodes only.
function richText(text) {
  if (!text.includes('@')) return [text];
  const names = mentionNames();
  // Lowercasing can change a string's length (rare scripts); then compare
  // slice by slice so indexes still line up with the original text.
  const lower = text.toLowerCase();
  const sameLength = lower.length === text.length;
  const at = (i, n) => (sameLength ? lower.startsWith(n.lower, i) : text.substr(i, n.name.length).toLowerCase() === n.lower);
  const out = [];
  let from = 0;
  for (let i = text.indexOf('@'); i !== -1; i = text.indexOf('@', i + 1)) {
    if (i < from || isWordChar(text[i - 1])) continue;
    const name = names.find((n) => at(i + 1, n) && !isWordChar(text[i + 1 + n.name.length]))?.name;
    if (!name) continue;
    if (i > from) out.push(text.slice(from, i));
    const self = name.toLowerCase() === state.me?.name.toLowerCase();
    out.push(h('span', { class: `mention${self ? ' me' : ''}` }, text.substr(i, name.length + 1)));
    from = i + 1 + name.length;
  }
  if (from < text.length) out.push(text.slice(from));
  return out;
}

// mentionBox suggests people while typing "@" in input; picking one
// writes "@name ". Returns the popup to place beside the input.
function mentionBox(input) {
  const pop = h('div', { class: 'mention-pop', role: 'listbox', hidden: true });
  let matches = [];
  let pick = 0;
  const query = () => /(?:^|\s)@([^\s@]{0,32})$/.exec(input.value.slice(0, input.selectionStart ?? input.value.length));
  const close = () => { pop.hidden = true; matches = []; };
  const choose = (card) => {
    const caret = input.selectionStart ?? input.value.length;
    const before = input.value.slice(0, caret).replace(/@[^\s@]*$/, `@${card.name} `);
    input.value = before + input.value.slice(caret);
    input.setSelectionRange(before.length, before.length);
    input.focus();
    close();
  };
  const draw = () => {
    pop.replaceChildren(...matches.map((card, i) => h('button', {
      type: 'button', class: 'mention-opt', role: 'option', 'aria-selected': String(i === pick),
      onmousedown: (e) => { e.preventDefault(); choose(card); },
    }, avatar(card, 28), whoName(card), h('small', {}, genderAge(card)))));
    pop.hidden = !matches.length;
  };
  input.addEventListener('input', () => {
    const m = query();
    if (!m) { close(); return; }
    const q = m[1].toLowerCase();
    const people = [...state.online.values()];
    matches = [...people.filter((c) => c.name.toLowerCase().startsWith(q)),
      ...people.filter((c) => !c.name.toLowerCase().startsWith(q) && c.name.toLowerCase().includes(q))].slice(0, 6);
    pick = 0;
    draw();
  });
  input.addEventListener('keydown', (e) => {
    if (pop.hidden) return;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      pick = (pick + (e.key === 'ArrowDown' ? 1 : matches.length - 1)) % matches.length;
      draw();
    } else if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault();
      choose(matches[pick]);
    } else if (e.key === 'Escape') {
      close();
    }
  });
  input.addEventListener('blur', () => setTimeout(close, 100));
  return pop;
}

// ---------------------------------------------------------------------------
// HTTP boundary
// ---------------------------------------------------------------------------

async function api(method, path, body) {
  const headers = {};
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  const isForm = body instanceof FormData;
  if (body && !isForm) headers['Content-Type'] = 'application/json';
  const res = await fetch(path, { method, headers, body: isForm ? body : body && JSON.stringify(body) });
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { /* non-JSON error page */ }
  if (!res.ok) {
    const err = new Error(data?.error || `Request failed (${res.status})`);
    err.status = res.status;
    throw err;
  }
  return data;
}

// IMAGE_TYPES is what the server accepts (SPEC section 2).
const IMAGE_TYPES = 'image/jpeg,image/png,image/gif,image/webp';

async function uploadPhoto(file) {
  const fd = new FormData();
  fd.append('image', file);
  return api('POST', '/media/upload', fd);
}

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

const MAIN_ROOM = 1;
const FEED_KEEP = 120;

const state = {
  token: readStore(TOKEN_KEY),
  me: null,
  online: new Map(), // id -> Card
  feed: [],          // room chat events, oldest first
  convos: new Map(), // peer id -> { peer, msgs, phase, unread, seenDoor }
  archive: new Map(), // peer id -> { peer, msgs, ts }: ended chats kept on this device
  active: null,      // peer id of the open chat
  view: 'room',      // mobile: room | people | chats
  side: 'people',    // desktop right pane: people | chats
  ws: null,
  retry: 0,
  mentions: 0,       // mentions of me not yet seen in the room
  unseenFeed: 0,     // room lines that arrived while scrolled up
  gifSearchOff: false, // the server has no GIF library configured
  unveiled: new Set(), // room image ids someone tapped to see
  lastText: null,    // the last text frame sent, to resend after RECONSIDER
  metBefore: new Map(), // peer id -> { ts, count }: chats from earlier visits
  promises: [],      // { id, name, ts }: people to pick it up with tomorrow
  greeted: new Set(), // promised people already announced this visit
  letters: [],       // { id, from, name, content, sent_at, read }
  notes: {},         // peer id -> my note about them, kept on this device
  plain: readStore(PLAIN_KEY) === '1', // the closeness glow is switched off
};

const BASE_TITLE = document.title;

// SCREEN is which page is showing; each event names the page to go to.
// The desk is reachable straight from boot through a #desk link.
const SCREEN = {
  boot: { enter: 'enter', room: 'room', desk: 'desk' },
  enter: { enter: 'enter', room: 'room', desk: 'desk' },
  room: { room: 'room', enter: 'enter', elsewhere: 'elsewhere', desk: 'desk' },
  desk: { desk: 'desk', room: 'room', enter: 'enter' },
  elsewhere: { room: 'room', enter: 'enter', desk: 'desk' },
};
const screen = createMachine('screen', SCREEN, 'boot');

// CONN is the socket's life. Only retrying may reconnect on its own, so a
// backoff timer that fires after leaving finds the machine closed.
const CONN = {
  idle: { connect: 'connecting', end: 'closed' },
  connecting: { hello: 'live', drop: 'retrying', end: 'closed' },
  live: { drop: 'retrying', end: 'closed' },
  retrying: { connect: 'connecting', end: 'closed' },
  closed: { connect: 'connecting', end: 'closed' },
};
const conn = createMachine('connection', CONN, 'idle');

onRefusal((move) => toast(t('stateBug', { move }), true));

// CHAT is the life of a private chat, drawn in docs/chat-states.md:
// none (not knocked yet), outgoing (I knocked), incoming (they knocked),
// open, declined (they passed on my knock), gone (they left).
const CHAT = {
  none: { knockSent: 'outgoing', knockIn: 'incoming', left: 'gone' },
  outgoing: { opened: 'open', message: 'open', declined: 'declined', left: 'gone' },
  incoming: { opened: 'open', message: 'open', decline: 'none', left: 'gone' },
  open: { opened: 'open', message: 'open', left: 'gone' },
  declined: { knockIn: 'incoming', left: 'gone' },
  gone: { returned: 'none', left: 'gone' },
};

// LADDER is the reveal ladder inside an open chat. It only climbs: face,
// then one side asks (asked by me, or invited by them), then close.
const LADDER = {
  face: { face: 'face', asked: 'asked', invited: 'invited', close: 'close' },
  asked: { asked: 'asked', close: 'close' },
  invited: { invited: 'invited', close: 'close' },
  close: { close: 'close' },
};

// moveChat applies event to a chat. It returns 'moved' when the phase
// changed, 'stayed' for a move within the phase (a new message), and
// 'refused' for a move the table does not allow, which is then ignored.
function moveChat(peerId, c, event) {
  const next = nextState(CHAT, c.phase, event);
  if (next == null) {
    refuse(`chat ${peerId}`, c.phase, event);
    return 'refused';
  }
  const moved = next !== c.phase;
  c.phase = next;
  return moved ? 'moved' : 'stayed';
}

function convo(peerId) {
  let c = state.convos.get(peerId);
  if (!c) {
    c = { peer: state.online.get(peerId) || { id: peerId, name: '?' }, msgs: [], phase: 'none', unread: 0, ladder: 'face' };
    state.convos.set(peerId, c);
  }
  return c;
}

// isStarted: a knock went one way or the other, so the chat is listed.
// A chat opened but never knocked has no messages, even once they leave.
const isStarted = (c) => c.phase !== 'none' && c.msgs.length > 0;

// shownGender colours a name only for someone you have an open chat with,
// so nobody can scan the room and pick out who is a woman.
function shownGender(person) {
  return state.convos.get(person.id)?.phase === 'open' ? person.gender : null;
}

// whoName is a person's name, coloured by gender only where shownGender
// allows it.
function whoName(person, tag = 'span', props = {}) {
  return h(tag, { class: 'who-name', 'data-gender': shownGender(person), ...props }, person.name);
}

const isStaff = () => state.me && (state.me.role === 'moderator' || state.me.role === 'admin');

// ---------------------------------------------------------------------------
// Screen: Enter
// ---------------------------------------------------------------------------

// tagEditor edits a list of free-form tags in place. value() also counts
// text typed but not yet added, so pressing submit never loses a tag.
function tagEditor(tags) {
  const list = h('div', { class: 'tag-list' });
  const input = h('input', { class: 'input tag-input', maxlength: 20, placeholder: 'Type a tag and press Enter', 'aria-label': 'Add a tag' });
  const ideas = h('div', { class: 'tag-ideas' });
  const add = (raw) => {
    const tag = raw.trim().replace(/\s+/g, ' ');
    if (!tag || tags.length >= MAX_TAGS || tags.some((x) => x.toLowerCase() === tag.toLowerCase())) return;
    tags.push(tag);
    draw();
  };
  const draw = () => {
    list.replaceChildren(...tags.map((tag, i) => h('span', { class: 'tag-chip' }, tag,
      h('button', { type: 'button', 'aria-label': `Remove ${tag}`, onclick: () => { tags.splice(i, 1); draw(); } }, icon('x', 12)))));
    const full = tags.length >= MAX_TAGS;
    input.disabled = full;
    input.placeholder = full ? 'Three tags is the limit' : 'Type a tag and press Enter';
    ideas.replaceChildren(...TAG_IDEAS.filter((idea) => !tags.some((x) => x.toLowerCase() === idea)).map((idea) =>
      h('button', { type: 'button', class: 'chip sm', disabled: full, onclick: () => add(idea) }, `+ ${idea}`)));
  };
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' || e.key === ',') { e.preventDefault(); add(input.value); input.value = ''; }
    if (e.key === 'Backspace' && !input.value && tags.length) { tags.pop(); draw(); }
  });
  draw();
  return {
    el: h('div', { class: 'tag-editor' }, list, input, ideas),
    value: () => { add(input.value); input.value = ''; return [...tags]; },
  };
}

function chipGroup(name, options, selected, onPick) {
  const group = h('div', { class: 'chips', role: 'group', 'aria-label': name });
  for (const [value, label] of Object.entries(options)) {
    group.append(h('button', {
      type: 'button', class: 'chip', 'aria-pressed': String(value === selected),
      onclick: () => {
        for (const b of group.children) b.setAttribute('aria-pressed', 'false');
        group.querySelector(`[data-v="${CSS.escape(value)}"]`).setAttribute('aria-pressed', 'true');
        onPick(value);
      },
      'data-v': value,
    }, label));
  }
  return group;
}

// draftFromProfile prefills the enter form from the last visit on this
// device; a ?name= link still wins, since it names a new visitor.
function draftFromProfile(profile, linkName) {
  return {
    mode: profile?.kind === 'member' ? 'signin' : 'guest', // guest | join | signin
    name: linkName || profile?.name || '',
    age: profile?.age ? String(profile.age) : '', gender: profile?.gender || '', tags: [...(profile?.tags || [])],
    location: profile?.location || '', about: profile?.about || '', email: profile?.email || '', photo: null,
  };
}

// enterDraft keeps what was typed while switching between the three ways in.
const enterDraft = draftFromProfile(readJSON(PROFILE_KEY), new URLSearchParams(window.location.search).get('name'));

// rememberProfile keeps the card for the next visit on this device: a
// returning guest finds it filled in, a member finds their email.
function rememberProfile(me) {
  writeJSON(PROFILE_KEY, {
    kind: me.kind, name: me.name, gender: me.gender, age: me.age, tags: me.tags || [],
    location: me.location || '', about: me.about || '', email: me.email || '',
  });
}

// returningAs is the saved card when the form still describes it, so the
// way back in reads "Continue as".
function returningAs(d) {
  const saved = readJSON(PROFILE_KEY);
  if (!saved) return null;
  const same = d.mode === 'signin'
    ? saved.kind === 'member' && d.email.trim() === saved.email
    : d.mode === 'guest' && saved.kind === 'guest' && d.name.trim() === saved.name;
  return same ? saved : null;
}

function renderEnter(message) {
  if (!screen.send('enter')) return;
  const d = enterDraft;
  const err = h('div', { class: 'error-text', role: 'alert' }, message || '');
  const bind = (key, attrs) => h('input', { class: 'input', value: d[key], ...attrs, oninput: (e) => { d[key] = e.target.value; attrs.oninput?.(e); } });

  // The face (or, when joining with one, the photo) previews live.
  const preview = h('span', { class: 'avatar', style: { width: '56px', height: '56px' } });
  const showPreview = () => Object.assign(preview.style, d.photo
    ? { backgroundImage: `url("${URL.createObjectURL(d.photo)}")`, imageRendering: '' }
    : pictureStyle({ name: d.name }));
  showPreview();

  const returning = returningAs(d);
  const submit = h('button', { class: 'btn primary', type: 'submit', style: { height: '52px', fontSize: '16px' } },
    returning ? t('continueAs', { name: returning.name }) : { guest: t('enter'), join: 'Create account', signin: t('signIn') }[d.mode]);

  const tagBox = tagEditor(d.tags);
  const profileFields = () => [
    h('div', { class: 'photo-pick' },
      d.mode === 'join'
        ? h('label', { for: 'photo-in', style: { cursor: 'pointer' } }, preview)
        : preview,
      h('div', {},
        d.mode === 'join'
          ? [h('label', { for: 'photo-in', style: { cursor: 'pointer', fontWeight: 600 } }, t('photo')), h('br'), h('small', {}, t('photoHint'))]
          : [h('b', {}, 'Your face'), h('br'), h('small', {}, 'Drawn from your name. Change the name, change the face.')]),
      d.mode === 'join' && h('input', {
        type: 'file', accept: IMAGE_TYPES, class: 'sr', id: 'photo-in',
        onchange: (e) => { d.photo = e.target.files[0] || null; showPreview(); },
      })),
    h('div', { class: 'row' },
      h('label', { class: 'field' }, h('span', {}, t('name')), bind('name', { name: 'name', maxlength: 32, required: true, placeholder: t('namePh'), autocomplete: 'off', oninput: () => { if (!d.photo) showPreview(); } })),
      h('label', { class: 'field' }, h('span', {}, t('age')), bind('age', { name: 'age', type: 'number', min: 18, max: 99, required: true, inputmode: 'numeric', placeholder: '18+' }))),
    h('div', { class: 'field' }, h('span', {}, t('gender')), chipGroup(t('gender'), GENDERS, d.gender, (v) => { d.gender = v; })),
    h('div', { class: 'field' }, h('span', {}, t('tags')), tagBox.el),
    h('label', { class: 'field' }, h('span', {}, t('location')), bind('location', { name: 'location', maxlength: 40, placeholder: t('locationPh') })),
    h('label', { class: 'field' }, h('span', {}, t('about')), bind('about', { name: 'about', maxlength: 140, placeholder: t('aboutPh') })),
  ];
  const credentialFields = (newAccount) => [
    h('label', { class: 'field' }, h('span', {}, t('email')), bind('email', { name: 'email', type: 'email', autocomplete: newAccount ? 'email' : 'username', required: true })),
    h('label', { class: 'field' }, h('span', {}, t('password')),
      h('input', { class: 'input', name: 'password', type: 'password', required: true, minlength: newAccount ? 8 : null, autocomplete: newAccount ? 'new-password' : 'current-password' })),
  ];
  const profileBody = () => ({
    name: d.name.trim(), gender: d.gender, age: Number(d.age), tags: tagBox.value(),
    about: d.about.trim(), location: d.location.trim(),
  });

  const form = h('form', {
    class: 'make-card', novalidate: true,
    onsubmit: async (e) => {
      e.preventDefault();
      err.textContent = '';
      const password = form.querySelector('input[name=password]')?.value || '';
      if (d.mode !== 'signin' && !d.gender) { err.textContent = 'Pick who you are.'; return; }
      submit.disabled = true;
      try {
        if (d.mode === 'guest') {
          const res = await api('POST', '/auth/guest', profileBody());
          setSession(res.token, res.user);
        } else {
          if (d.mode === 'join') await api('POST', '/auth/register', { ...profileBody(), email: d.email.trim(), password });
          const res = await api('POST', '/auth/login', { email: d.email.trim(), password });
          setSession(res.token, res.user);
          if (d.mode === 'join' && d.photo) {
            try {
              const up = await uploadPhoto(d.photo);
              state.me = await api('PATCH', '/me', { photo_id: up.id });
            } catch (upErr) {
              toast(upErr.message, true);
            }
          }
        }
        d.photo = null;
        startRoom();
      } catch (ex) {
        err.textContent = ex.message;
        submit.disabled = false;
      }
    },
  },
    h('div', { class: 'enter-tabs', role: 'tablist' }, [['guest', 'Guest'], ['join', 'Create account'], ['signin', 'Sign in']].map(([mode, label]) =>
      h('button', { type: 'button', role: 'tab', class: 'tab', 'aria-selected': String(d.mode === mode), onclick: () => { d.mode = mode; renderEnter(); } }, label))),
    h('h2', {}, returning ? t('welcomeBack', { name: returning.name }) : { guest: t('makeCard'), join: 'Make your account', signin: 'Welcome back' }[d.mode]),
    d.mode === 'guest' && h('p', { class: 'fine' }, 'One visit, no email. Your card disappears when you leave.'),
    d.mode === 'join' && h('p', { class: 'fine' }, 'Keep your name, add your own photo. People see your face until you both choose to get closer.'),
    d.mode !== 'signin' && profileFields(),
    d.mode !== 'guest' && credentialFields(d.mode === 'join'),
    err,
    submit,
    h('p', { class: 'fine' }, d.mode === 'signin' ? 'Staff accounts are created by the admin.' : t('fine')),
  );

  const point = (text) => h('li', {}, icon('check', 18), text);
  $('#app').replaceChildren(
    h('div', { class: 'enter' },
      h('header', { class: 'topbar' },
        brand(),
        h('div', { class: 'right' }, h('a', { class: 'btn sm', href: '/faces' }, 'Find your face'))),
      h('main', { class: 'enter-main' },
        h('section', { class: 'hero' },
          h('h1', {}, t('heroA'), ' ', t('heroB'), h('br'), h('em', {}, t('heroC'))),
          h('p', {}, t('heroP')),
          h('ul', {}, point(t('point1')), point(t('point2')), point(t('point3')))),
        form),
      h('footer', { class: 'seo-foot' },
        h('h2', {}, t('seoTitle')),
        h('p', {}, t('seoP')))),
  );
  form.querySelector('input')?.focus();
}

function setSession(token, me) {
  state.token = token;
  state.me = me;
  writeStore(TOKEN_KEY, token);
  rememberProfile(me);
}

// endSession stops the session in this tab. A kick or an expired login
// keeps the card and past chats so coming back is one tap; pressing Leave
// (forget) wipes them from this device and from every other tab.
function endSession(message, { forget = false } = {}) {
  clearInterval(deskTimer);
  if (forget && state.me) {
    forgetDevice(state.me.id);
    tabChannel?.postMessage('left');
  }
  closeSocket();
  state.token = null;
  state.me = null;
  state.online.clear();
  state.convos.clear();
  state.archive = new Map();
  state.feed = [];
  state.active = null;
  state.mentions = 0;
  state.unseenFeed = 0;
  writeStore(TOKEN_KEY, null);
  location.hash = '';
  renderEnter(message);
}

// forgetDevice removes everything this device kept for the account, for
// people on a shared computer.
function forgetDevice(userId) {
  for (const key of [chatsKey, notesKey, promisesKey, lettersKey]) writeJSON(key(userId), null);
  writeJSON(PROFILE_KEY, null);
  Object.assign(enterDraft, draftFromProfile(null, null));
}

// ---------------------------------------------------------------------------
// Screen: Room
// ---------------------------------------------------------------------------

// Tabs share one login, and the server keeps one socket per person, so two
// live tabs would keep replacing each other. The tab used last takes the
// connection; the others step aside until asked back.
const tabChannel = 'BroadcastChannel' in window ? new BroadcastChannel('drift.tabs') : null;
tabChannel?.addEventListener('message', (e) => {
  if (e.data === 'connected' && screen.is('room') && !conn.is('closed')) stepAside();
  else if (e.data === 'left' && state.token) endSession(t('leftRoom'));
});

function startRoom() {
  state.archive = storedArchive(state.me.id);
  state.metBefore = new Map([...state.archive].map(([id, chat]) => [id, { ts: chat.ts, count: chat.msgs.filter(isMessage).length }]));
  state.promises = livePromises(readJSON(promisesKey(state.me.id)) || [], Date.now());
  state.letters = readJSON(lettersKey(state.me.id)) || [];
  state.notes = readJSON(notesKey(state.me.id)) || {};
  state.greeted = new Set();
  tabChannel?.postMessage('connected');
  renderShell();
  connect();
}

// closeSocket stops this tab's connection for good: ended keeps the close
// handler from reconnecting.
function closeSocket() {
  conn.send('end');
  state.ws?.close();
  state.ws = null;
}

function stepAside() {
  closeSocket();
  renderElsewhere();
}

function renderElsewhere() {
  if (!screen.send('elsewhere')) return;
  $('#app').replaceChildren(h('div', { class: 'enter' },
    h('header', { class: 'topbar' }, brand()),
    h('main', { class: 'enter-main' },
      h('section', { class: 'hero' },
        h('p', {}, t('elsewhere')),
        h('button', { class: 'btn primary', type: 'button', onclick: startRoom }, t('useHere'))))));
}

const brand = () => h('span', { class: 'brand' }, 'Drift', h('b', {}, '.'));
const soundIcon = () => icon(soundSettings().on ? 'bell' : 'bellOff');

function topbar(extra) {
  return h('header', { class: 'topbar' },
    brand(),
    h('div', { class: 'right' },
      h('span', { class: 'live-dot hide-sm', id: 'here-count' }, `${state.online.size + 1} ${t('hereNow')}`),
      extra,
      state.me && h('button', { class: 'icon-btn bare', type: 'button', id: 'sound-btn', onclick: openSounds, title: t('sounds'), 'aria-label': t('sounds') },
        soundIcon()),
      state.me && mePill(),
      h('button', { class: 'btn sm', type: 'button', onclick: () => endSession(t('leftRoom'), { forget: true }) }, t('leave'))));
}

function mePill() {
  return h('button', { class: 'me-pill', type: 'button', onclick: openEditCard, title: t('editCard') },
    avatar(state.me, 28),
    h('span', { class: 'label' }, state.me.name, state.me.tags?.length ? ` · ${state.me.tags[0]}` : ''));
}

function renderShell() {
  clearInterval(deskTimer);
  if (location.hash === '#desk' && isStaff()) return renderDesk();
  if (!screen.send('room')) return;
  const deskLink = isStaff() && h('button', { class: 'btn sm outline', type: 'button', onclick: () => { location.hash = '#desk'; } }, icon('shield', 16), h('span', { class: 'desk-label' }, t('desk')));

  const feed = h('div', {
    class: 'feed', id: 'feed', role: 'log', 'aria-live': 'polite',
    onscroll: () => { if (nearBottom(feed)) seeFeed(); },
  });
  const input = h('input', { class: 'input', id: 'room-input', maxlength: 1000, placeholder: t('sayRoom'), autocomplete: 'off', 'aria-label': t('sayRoom') });
  const sendMedia = (m) => send({ type: 'chat', room_id: MAIN_ROOM, media_id: m.id });
  const composer = h('form', {
    class: 'composer',
    onsubmit: (e) => {
      e.preventDefault();
      const content = input.value.trim();
      if (!content) return;
      send({ type: 'chat', room_id: MAIN_ROOM, content });
      input.value = '';
    },
  },
    mediaButtons('room', sendMedia),
    h('div', { class: 'input-wrap' }, input, mentionBox(input)),
    h('button', { class: 'icon-btn send', type: 'submit', 'aria-label': t('send') }, icon('send')));

  const jump = h('button', {
    class: 'jump', id: 'feed-jump', type: 'button', hidden: true,
    onclick: () => { feed.scrollTop = feed.scrollHeight; seeFeed(); },
  });
  const main = h('section', { class: 'pane main' },
    h('div', { class: 'pane-head' }, h('h2', {}, t('room')), h('small', {}, t('lastLines'))),
    feed, jump, composer);

  const side = h('section', { class: 'pane side', id: 'side' });
  const stage = h('div', { class: 'stage', id: 'stage', 'data-view': state.view }, main, side);

  const mobileTabs = h('nav', { class: 'mobile-tabs', id: 'mobile-tabs' });

  const knocks = h('div', { class: 'knocks', id: 'knocks', 'aria-live': 'polite' });
  $('#app').replaceChildren(h('div', { class: 'shell' }, topbar(deskLink), stage, mobileTabs, knocks));
  redrawFeed();
  renderSide();
  renderMobileTabs();
  renderKnockTray();
}

function unreadTotal() {
  let n = 0;
  for (const c of state.convos.values()) n += c.unread;
  return n;
}

function renderMobileTabs() {
  updateTitle();
  const nav = $('#mobile-tabs');
  if (!nav) return;
  const tab = (view, label, badge) => h('button', {
    class: 'tab', type: 'button', role: 'tab', 'aria-selected': String(state.view === view),
    onclick: () => showView(view),
  }, label, badge ? h('span', { class: 'badge' }, badge) : null);
  nav.replaceChildren(tab('room', t('room'), state.mentions ? `@${state.mentions}` : null), tab('people', t('people'), null), tab('chats', t('chats'), unreadTotal() || null));
}

// showView switches the phone layout to room, people or chats.
function showView(view) {
  state.view = view;
  if (view !== 'room') state.side = view;
  const shown = view === 'chats' ? state.convos.get(state.active) : null;
  if (shown) shown.unread = 0;
  if (view === 'room') state.mentions = 0;
  const stage = $('#stage');
  if (stage) stage.dataset.view = view;
  renderSide();
  renderMobileTabs();
}

// updateTitle puts what is waiting in the tab title, like "(3) Drift".
function updateTitle() {
  const n = unreadTotal() + state.mentions;
  document.title = n ? `(${n}) ${BASE_TITLE}` : BASE_TITLE;
}

// roomVisible: the room feed is on screen (on phones, its tab is showing).
const isPhone = () => matchMedia('(max-width: 960px)').matches;
function roomVisible() {
  return screen.is('room') && document.visibilityState === 'visible' && (state.view === 'room' || !isPhone());
}

function refreshCounts() {
  const here = $('#here-count');
  if (here) here.textContent = `${state.online.size + 1} ${t('hereNow')}`;
  renderMobileTabs();
}

// ---------- Room feed ----------

// A line continues the one before when the same person wrote it within a
// few minutes: it drops the face and name, like a chat app groups bubbles.
const GROUP_SECONDS = 180;
const continues = (ev, prev) => !!prev && prev.sender_id === ev.sender_id && ev.ts - prev.ts < GROUP_SECONDS;

function feedLine(ev, prev, atMe = ev.sender_id !== state.me?.id && mentionsMe(ev.content)) {
  const mine = ev.sender_id === state.me?.id;
  const person = mine ? state.me : state.online.get(ev.sender_id) || { id: ev.sender_id, name: ev.name };
  const cont = continues(ev, prev);
  const acts = h('div', { class: 'acts' });
  if (!mine) acts.append(h('button', { class: 'icon-btn bare', type: 'button', title: t('report'), 'aria-label': t('report'), onclick: () => openReport('message', ev.id) }, icon('flag', 16)));
  if (isStaff()) acts.append(h('button', { class: 'icon-btn bare', type: 'button', title: t('remove'), 'aria-label': t('remove'), onclick: () => modAct(`/admin/messages/${ev.id}/remove`) }, icon('trash', 16)));

  return h('div', { class: `line${mine ? ' mine' : ''}${cont ? ' cont' : ''}${atMe ? ' at-me' : ''}`, 'data-id': ev.id, 'data-aura': person.aura || null },
    cont ? h('span', { class: 'avatar-gap', title: clock(ev.ts) }) : avatar(person, 32),
    h('div', { class: 'body' },
      cont ? null : h('span', { class: 'meta' },
        whoName({ ...person, name: ev.name }, 'button', { type: 'button', onclick: () => !mine && openPerson(ev.sender_id, ev.name) }),
        mine ? h('span', { class: 'you-tag' }, ` (${t('you')})`) : null,
        person.age ? ` · ${genderAge(person)}` : '',
        ` · ${clock(ev.ts)}`,
        ev.filtered ? h('span', { class: 'filtered-tag' }, ` · ${t('filtered')}`) : null),
      ev.content ? h('span', { class: 'text' }, richText(ev.content)) : null,
      ev.image ? shot(ev.image, mine ? null : ev.id) : null),
    acts.children.length ? acts : null);
}

// shot shows an image in chat. Animated GIFs play in place; the lightbox
// opens the full file either way. A room image from someone else arrives
// blurred (veilId is its message id) until tapped, so nobody is shown a
// picture they did not choose to see.
function shot(image, veilId = null) {
  const gif = image.thumb_url.endsWith('.gif');
  const label = gif ? 'Open GIF' : 'Open photo';
  const veiled = veilId != null && !state.unveiled.has(veilId);
  return h('button', {
    type: 'button', class: `shot-btn${gif ? ' gif' : ''}${veiled ? ' veiled' : ''}`, 'aria-label': veiled ? t('tapToView') : label,
    'data-picture': `${image.thumb_url} ${image.url}`,
    onclick: (e) => {
      if (veilId == null || state.unveiled.has(veilId)) { openLightbox(image.url); return; }
      state.unveiled.add(veilId);
      e.currentTarget.classList.remove('veiled');
      e.currentTarget.setAttribute('aria-label', label);
    },
  },
    h('img', { class: 'shot', src: image.thumb_url, alt: '', loading: 'lazy' }),
    gif ? h('span', { class: 'gif-tag' }, t('gif')) : null,
    veiled ? h('span', { class: 'veil-tag' }, t('tapToView')) : null);
}

const nearBottom = (el) => el.scrollHeight - el.scrollTop - el.clientHeight < 80;

// Lines waiting for the next frame. Appending them together costs one
// layout read and one scroll per frame instead of two layouts per line.
let feedQueue = [];
let feedFrame = 0;

// addFeed appends a live room line. Someone scrolled up to read is left
// where they are, with a count of what arrived below instead of a jump.
function addFeed(ev, atMe) {
  const prev = state.feed[state.feed.length - 1];
  state.feed.push(ev);
  if (state.feed.length > FEED_KEEP) state.feed.shift();
  feedQueue.push(feedLine(ev, prev, atMe));
  if (ev.sender_id === state.me?.id) feedQueue.mine = true;
  feedFrame ||= requestAnimationFrame(flushFeed);
}

function flushFeed() {
  feedFrame = 0;
  const lines = feedQueue;
  feedQueue = [];
  const feed = $('#feed');
  if (!feed || !lines.length) return;
  const atBottom = nearBottom(feed); // the one read, before any write
  feed.append(...lines);
  for (let extra = feed.children.length - FEED_KEEP; extra > 0; extra--) feed.firstElementChild.remove();
  if (atBottom || lines.mine) {
    feed.scrollTop = feed.scrollHeight;
    return;
  }
  state.unseenFeed += lines.length;
  const jump = $('#feed-jump');
  if (jump) {
    jump.hidden = false;
    jump.replaceChildren(state.unseenFeed === 1 ? t('newLine') : t('newLines', { n: state.unseenFeed }), ' ', icon('back', 14));
  }
}

// redrawFeed rebuilds the room feed from state and scrolls to the end.
function redrawFeed() {
  feedQueue = []; // state.feed already holds them
  const feed = $('#feed');
  if (!feed) return;
  feed.replaceChildren(...state.feed.map((ev, i) => feedLine(ev, state.feed[i - 1])));
  feed.scrollTop = feed.scrollHeight;
  seeFeed();
}

function seeFeed() {
  state.unseenFeed = 0;
  const jump = $('#feed-jump');
  if (jump) jump.hidden = true;
}

// ---------- Right pane ----------

function renderSide() {
  const side = $('#side');
  if (!side) return;
  const tabs = sideTabs();
  // Presence and messages re-render this pane often: keep where the reader
  // was scrolled to and which control had focus.
  const scrolls = ['.people', '.convo-list'].map((sel) => [sel, side.querySelector(sel)?.scrollTop]);
  const focusKey = side.contains(document.activeElement) ? document.activeElement.dataset.key : null;
  const pane = state.side === 'chats' ? chatsPane() : peoplePane();
  keepPictures(side, pane);
  side.replaceChildren(tabs, pane);
  for (const [sel, top] of scrolls) {
    const el = side.querySelector(sel);
    if (el && top != null) el.scrollTop = top;
  }
  if (focusKey) side.querySelector(`[data-key="${CSS.escape(focusKey)}"]`)?.focus({ preventScroll: true });
  renderKnockTray();
}

function sideTabs() {
  const tab = (key, label, badge) => h('button', {
    class: 'tab', type: 'button', role: 'tab', 'aria-selected': String(state.side === key),
    onclick: () => { state.side = key; renderSide(); },
  }, label, badge ? h('span', { class: 'badge' }, badge) : null);
  return h('div', { class: 'tabs', role: 'tablist' }, tab('people', t('people')), tab('chats', t('chats'), unreadTotal() || null));
}

// The people pane is built once and then kept: a presence event changes one
// card (refreshPerson), not all of them. peopleCards maps id -> card element
// in the pane on screen; peopleDirty marks changes made while it was hidden.
const peopleCards = new Map();
let peopleDirty = false;

function peoplePane() {
  const wrap = h('div', { class: 'people' });
  const people = [...state.online.values()].sort((a, b) => (a.online_since || 0) - (b.online_since || 0));
  peopleCards.clear();
  peopleDirty = false;
  if (!people.length) {
    wrap.append(h('div', { class: 'empty' }, t('nobody')));
    return wrap;
  }
  const cards = people.map((card) => {
    const el = personCard(card);
    peopleCards.set(card.id, el);
    return el;
  });
  wrap.append(h('div', { class: 'strip' }, cards),
    // The DB-IP Lite licence (CC BY 4.0) asks for this link wherever its
    // countries are shown, which is the flags on these cards.
    h('p', { class: 'fine credit' }, h('a', { href: 'https://db-ip.com', rel: 'noopener' }, 'IP Geolocation by DB-IP')));
  return wrap;
}

// peopleShowing: the people pane is on screen (on phones, its tab is open).
function peopleShowing() {
  return screen.is('room') && state.side === 'people' && !document.hidden && (state.view === 'people' || !isPhone());
}

// refreshPerson brings one person's card in line with state: a join adds
// it (people are listed by arrival, so at the end), a leave removes it, any
// other change redraws just that card. While the pane is hidden it only
// marks the pane for one rebuild when it is shown again.
function refreshPerson(id) {
  if (!peopleShowing()) {
    peopleDirty = true;
    return;
  }
  const strip = $('#side .strip');
  const card = state.online.get(id);
  const old = peopleCards.get(id);
  if (!strip || (!card && peopleCards.size === 1 && old)) {
    renderSide(); // first person in, or last one out: the pane changes shape
    return;
  }
  if (!card) {
    old?.remove();
    peopleCards.delete(id);
    return;
  }
  const el = personCard(card);
  peopleCards.set(id, el);
  if (old) old.replaceWith(el);
  else strip.append(el);
}

// The "here 5m" pills tick once a minute, in place.
setInterval(() => {
  if (!peopleShowing()) return;
  for (const [id, el] of peopleCards) {
    const pill = el.querySelector('.pill.here');
    const card = state.online.get(id);
    if (pill && card) pill.textContent = hereFor(card.online_since);
  }
}, 60000);

// tagRow always renders, empty or not, so cards in a row line up.
const tagRow = (tags) => h('span', { class: 'tag-row' }, (tags || []).map((tag) => h('span', { class: 'tag-chip small' }, tag)));

function personCard(card) {
  const c = state.convos.get(card.id);
  const promised = isPromised(card.id);
  return h('div', { class: 'pcard', 'data-tier': card.avatar_tier, 'data-aura': card.aura || null },
    h('button', {
      type: 'button', class: 'photo', 'aria-label': card.name, 'data-key': `card-${card.id}`, onclick: () => openPerson(card.id),
      style: { ...pictureStyle({ ...card, photo_url: c?.peer.photo_url }), border: 'none', width: '100%', cursor: 'pointer' },
    },
      h('span', { class: 'pill here' }, hereFor(card.online_since)),
      card.has_photo && !c?.peer.photo_url ? h('span', { class: 'pill', title: 'Has a photo you will see once you both get closer' }, icon('photo', 14)) : null,
      promised ? h('span', { class: 'pill promise', title: t('tomorrowMine') }, icon('heart', 12), t('pickUp')) : null),
    h('div', { class: 'info' },
      h('span', { class: 'name' }, h('span', {}, whoName(card), `, ${card.age}`), flag(card.country)),
      h('span', { class: 'sub' }, card.about || [genderLabel(card.gender), card.location].filter(Boolean).join(' · ')),
      tagRow(card.tags),
      h('button', { class: 'btn outline sm', type: 'button', 'data-key': `knock-${card.id}`, onclick: () => openChat(card.id) }, c && isStarted(c) ? t('open') : t('knock'))));
}

// ---------- Private chats ----------

// isViewing: peerId's chat is on screen right now (on phones, the Chats tab
// must be the one showing).
function isViewing(peerId) {
  return state.active === peerId && state.side === 'chats' && document.visibilityState === 'visible'
    && (state.view === 'chats' || !isPhone());
}

function showChat(peerId) {
  state.active = peerId;
  const c = state.convos.get(peerId);
  if (c) c.knockSeen = true;
  showView('chats');
}

// syncChat sets a chat to what hello says. It is the one move allowed from
// any phase: after a reconnect the server's doors are the truth. A reload
// keeps a waiting knock and an open chat's messages, kept on the device.
function syncChat(peerId, c, card, d) {
  if (!card) {
    c.phase = 'gone';
    return;
  }
  const ladder = d?.open ? ladderOf(d) : 'face';
  c.peer = {
    ...card, gender: card.gender || (d?.open ? c.peer.gender : undefined),
    photo_url: ladder === 'close' && card.has_photo ? d.peer?.photo_url ?? c.peer.photo_url : undefined,
  };
  c.ladder = ladder;
  if (!d) {
    // A declined knock has no door either; it stays declined until they leave.
    if (c.phase !== 'declined') {
      c.phase = 'none';
      c.msgs = [];
    }
  } else if (d.open) {
    c.phase = 'open';
    if (!c.msgs.length) c.msgs = [...(state.archive.get(peerId)?.msgs || [])];
  } else {
    c.phase = d.knocked_by_me ? 'outgoing' : 'incoming';
    c.msgs = d.knock ? [{ ...d.knock, to: d.knock.from === state.me.id ? peerId : state.me.id, knock: true }] : [];
  }
}

// declineKnock turns down a knock. The server confirms with door_closed,
// which moves the chat; the knocker cannot knock again this visit.
function declineKnock(peerId) {
  send({ type: 'decline', user_id: peerId });
}

// ---------- Knock tray ----------

// renderKnockTray shows knocks waiting for an answer under the top bar, on
// the side of the Chats tab, clear of both composers. However many arrive,
// it is one card: a burst of knocks must not bury the screen. It stays
// until the knocks are answered, declined or put off with Later, and hides
// while the Chats list is showing, where knocks already head the list.
function renderKnockTray() {
  const tray = $('#knocks');
  if (!tray) return;
  const chatsShowing = state.side === 'chats' && (state.view === 'chats' || !isPhone());
  const waiting = [...state.convos].filter(([id, c]) => c.phase === 'incoming' && !c.knockSeen && !isViewing(id))
    .sort((a, b) => lastTs(b[1]) - lastTs(a[1]));
  tray.replaceChildren(...(waiting.length && !chatsShowing ? [knockCard(waiting)] : []));
}

// knockCard sums up the waiting knocks, newest first. Later puts away all
// of them; only a new knock brings the card back.
function knockCard(waiting) {
  const [newestId, newest] = waiting[0];
  const later = () => {
    for (const [, c] of waiting) c.knockSeen = true;
    renderKnockTray();
  };
  const many = waiting.length > 1;
  const quote = newest.msgs[0]?.content || '';
  return h('div', { class: 'knock-card', role: 'status' },
    h('span', { class: 'face-stack' }, waiting.slice(0, 3).map(([, c]) => avatar(c.peer, 40))),
    h('div', { class: 't' },
      h('b', {}, many ? t('knocksWaiting', { n: waiting.length }) : t('knockIn', { name: newest.peer.name })),
      h('small', {}, many ? `${newest.peer.name}: ${quote}` : quote)),
    h('div', { class: 'acts' },
      h('button', { class: 'icon-btn bare', type: 'button', 'aria-label': t('later'), title: t('later'), onclick: later }, icon('x', 16)),
      many
        ? h('button', { class: 'btn sm', type: 'button', onclick: () => showView('chats') }, t('seeAll'))
        : h('button', { class: 'btn sm', type: 'button', onclick: () => declineKnock(newestId) }, t('decline')),
      h('button', { class: 'btn sm primary', type: 'button', onclick: () => openChat(newestId) }, many ? t('answerNewest') : t('answer'))));
}

// openChat goes to a live chat with peerId. Someone online gets a live one
// even if a past chat with them is kept, so knocking again is possible.
function openChat(peerId) {
  if (state.online.has(peerId)) convo(peerId);
  showChat(peerId);
  setTimeout(() => $('#dm-input')?.focus());
}

// pastConvo shapes a kept chat like a live one, so the list and the thread
// draw both; phase 'past' makes it read-only.
function pastConvo(peerId) {
  const kept = state.archive.get(peerId);
  return kept && { peer: kept.peer, msgs: kept.msgs, phase: 'past', unread: 0 };
}

// pastConvos are kept chats with no live chat in their place.
function pastConvos() {
  return [...state.archive.keys()].filter((id) => !state.convos.has(id)).map((id) => [id, pastConvo(id)]);
}

const isEnded = (c) => c.phase === 'gone' || c.phase === 'past' || c.phase === 'declined';

function convoPreview(c) {
  if (c.phase === 'gone') return t('gone', { name: c.peer.name });
  if (c.phase === 'declined') return t('declined', { name: c.peer.name });
  if (c.phase === 'incoming') return `${t('knocked')}: ${c.msgs[0]?.content || ''}`;
  const last = c.msgs[c.msgs.length - 1];
  return !last ? '' : isMessage(last) ? last.content || 'Photo' : markerText(last, c.peer);
}

// Chats mix messages with markers: the door opening, both getting closer,
// a promise to pick it up tomorrow. Markers have no sender.
const isMessage = (m) => m?.from != null;

// Markers added during this visit that no chat has drawn yet. Kept apart
// from the marker itself so render state never reaches localStorage.
const undrawnMarkers = new WeakSet();

// A live chat keeps its latest LIVE_MESSAGES: every draw and closeness scan
// walks them all, and a long night must not slow the chat down. The kept
// archive holds fewer still (PAST_MESSAGES).
const LIVE_MESSAGES = 300;

// keepMessage adds m to c, dropping the oldest past the cap. c.msgs is
// chat state owned by the handlers, so it is changed in place.
function keepMessage(c, m) {
  c.msgs.push(m);
  if (c.msgs.length > LIVE_MESSAGES) c.msgs.splice(0, c.msgs.length - LIVE_MESSAGES);
}

// addMarker puts a moment that is not a message into a live chat.
function addMarker(c, fields) {
  const marker = { ...fields, ts: Date.now() / 1000 };
  undrawnMarkers.add(marker);
  keepMessage(c, marker);
  saveArchiveSoon();
  rerenderChats();
}

// ladderOf is the reveal-ladder state carried by a door or closer frame.
const ladderOf = (f) => (f.close ? 'close' : f.asked_by_me ? 'asked' : f.asked_by_them ? 'invited' : 'face');

function markerText(m, peer) {
  if (m.door) return t('doorOpen');
  if (m.ladder) return t('closeBanner');
  return m.tomorrow === 'me' ? t('tomorrowMine') : t('tomorrowTheirs', { name: peer.name });
}

function convoItem(id, c, glow) {
  return h('button', {
    class: `convo-item${isEnded(c) ? ' gone' : ''}${c.phase === 'incoming' ? ' knock-in' : ''}`, type: 'button',
    'aria-current': String(state.active === id), 'data-peer': id,
    onclick: () => (c.phase === 'past' ? showChat(id) : openChat(id)),
  }, avatar(c.peer, 40),
  h('span', { class: 't' }, h('b', {}, whoName(c.peer), heartFor(glow)), h('small', {}, convoPreview(c))),
  c.unread ? h('span', { class: 'badge' }, c.unread) : null);
}

function chatsPane() {
  // Knocks wait at the top: they ask for a decision, messages do not.
  const items = [...[...state.convos.entries()].filter(([id, c]) => isStarted(c) || id === state.active), ...pastConvos()]
    .sort((a, b) => (b[1].phase === 'incoming') - (a[1].phase === 'incoming') || lastTs(b[1]) - lastTs(a[1]));
  if (!items.length) return h('div', { class: 'people' }, h('div', { class: 'empty' }, t('noChats')));
  // Closeness scans every message, so it is worked out once per chat per draw.
  const glows = new Map(items.map(([id, c]) => [id, glowOf(id, c)]));

  const unreadLetters = state.letters.filter((l) => !l.read).length;
  const list = h('div', { class: 'convo-list' },
    state.letters.length ? h('button', { class: 'convo-item letters-item', type: 'button', onclick: openLetters },
      h('span', { class: 'letters-icon' }, icon('letter', 20)), h('span', { class: 't' }, h('b', {}, t('letters')), h('small', {}, state.letters[0].name)),
      unreadLetters ? h('span', { class: 'badge' }, unreadLetters) : null) : null,
    items.map(([id, c]) => convoItem(id, c, glows.get(id))));
  const shown = state.active == null ? null : state.convos.get(state.active) ?? pastConvo(state.active);
  const glow = shown ? glows.get(state.active) : null;
  return h('div', { class: 'chats', 'data-open': String(Boolean(shown)), 'data-glow': glow },
    list, shown ? convoView(state.active, shown, glow) : h('div', { class: 'convo' }, h('div', { class: 'empty', style: { margin: 'auto' } }, t('pickChat'))));
}

const lastTs = (c) => c.msgs[c.msgs.length - 1]?.ts || 0;

function convoView(peerId, c, glow) {
  const peer = c.peer;
  const meta = [genderLabel(peer.gender), peer.age, peer.location].filter(Boolean).join(' · ');

  const gone = isEnded(c);
  // Report and block live in one menu, as in every messenger: two text
  // buttons in the header crowded the name off a phone screen.
  const head = h('div', { class: `convo-head${gone ? ' gone' : ''}` },
    h('button', { class: 'icon-btn bare', type: 'button', 'aria-label': t('back'), onclick: () => { state.active = null; renderSide(); } }, icon('back')),
    h('button', { class: 'who-btn', type: 'button', onclick: () => openPerson(peerId, peer.name), 'aria-label': `${t('viewCard')}: ${peer.name}` },
      avatar(peer, 44),
      h('span', { class: 'who' }, h('b', {}, whoName(peer), ' ', flag(peer.country), heartFor(glow)), h('small', {}, c.phase === 'past' ? t('pastChat') : gone ? t('ended', { name: peer.name }) : meta))),
    h('button', { class: 'icon-btn bare', type: 'button', 'aria-label': t('more'), title: t('more'), onclick: () => openChatMenu(peerId) }, icon('more')));

  const bubbles = h('div', { class: 'bubbles', id: 'bubbles' });
  const myId = state.me.id;

  if (c.phase === 'incoming' || c.phase === 'outgoing') {
    const knock = c.msgs[0];
    bubbles.append(h('div', { class: 'knock-panel' },
      avatar(peer, 96),
      h('h3', {}, c.phase === 'incoming' ? t('knockIn', { name: peer.name }) : t('knockSent')),
      knock ? h('div', { class: 'knock-quote' }, knock.content) : null,
      memoryCard(peer),
      h('p', {}, c.phase === 'incoming' ? t('knockInHelp') : t('knockWait', { name: peer.name })),
      c.phase === 'incoming' ? h('div', { class: 'actions' },
        h('button', { class: 'btn sm', type: 'button', onclick: () => declineKnock(peerId) }, t('decline')),
        h('button', { class: 'btn sm ghost', type: 'button', onclick: () => blockPerson(peerId) }, t('block'))) : null));
  } else if (c.phase === 'declined') {
    bubbles.append(h('div', { class: 'knock-panel' },
      avatar(peer, 96),
      h('h3', {}, t('declined', { name: peer.name })),
      h('p', {}, t('declinedHelp'))));
  } else if (c.phase === 'none') {
    bubbles.append(h('div', { class: 'knock-panel' },
      avatar(peer, 96),
      h('h3', {}, t('knockTitle', { name: peer.name })),
      peer.about ? h('div', { class: 'knock-quote' }, peer.about) : null,
      memoryCard(peer),
      h('p', {}, t('knockHelp'))));
  } else {
    if (c.phase === 'open') append(bubbles, [memoryCard(peer)]);
    c.msgs.forEach((m, i) => bubbles.append(isMessage(m)
      ? bubbleRow(m, c.msgs[i - 1], c.msgs[i + 1], m.from === myId)
      : markerBanner(m, peerId, peer)));
  }

  const footer = c.phase === 'past' ? pastBar(peerId)
    : c.phase === 'declined' ? endedBar(t('declined', { name: peer.name }))
      : gone ? endedBar(t('ended', { name: peer.name })) : dmComposer(peerId, c);
  const view = h('div', { class: 'convo', 'data-phase': c.phase, 'data-glow': glow, 'data-peer': peerId },
    head, c.phase === 'open' ? ladderBar(peerId, c) : null, bubbles, footer);
  setTimeout(() => { bubbles.scrollTop = bubbles.scrollHeight; });
  return view;
}

// bubbleRow is one private message. The report button sits beside the
// bubble, outside it, so it never covers the text; runs of messages from
// one side share a tail like Instagram and WhatsApp, and the last of a run
// carries the time.
function bubbleRow(m, prev, next, mine) {
  const sameSide = (o) => isMessage(o) && (o.from === state.me?.id) === mine;
  const first = !sameSide(prev);
  const last = !sameSide(next) || next.ts - m.ts > GROUP_SECONDS;
  const sender = mine ? state.me : state.convos.get(m.from)?.peer ?? state.online.get(m.from);
  return h('div', { class: `msg${mine ? ' mine' : ''}${first ? ' first' : ''}${last ? ' last' : ''}`, 'data-id': m.id, 'data-aura': sender?.aura || null },
    h('div', { class: `bubble${m.content ? '' : ' media-only'}` }, m.content ? richText(m.content) : null, m.image ? shot(m.image) : null),
    mine ? null : h('button', { class: 'icon-btn bare msg-act', type: 'button', 'aria-label': t('report'), title: t('report'), onclick: () => openReport('message', m.id) }, icon('flag', 14)),
    last ? h('span', { class: 'stamp' }, clock(m.ts)) : null);
}

// markerBanner draws a moment in the chat that is not a message. The chat
// is redrawn on every message, so the reveal animation plays only the first
// time a marker from this visit is drawn.
function markerBanner(m, peerId, peer) {
  const theirPromise = m.tomorrow === 'them' && !isPromised(peerId) && state.convos.get(peerId)?.phase === 'open';
  const fresh = undrawnMarkers.delete(m);
  return h('div', { class: `door-banner${fresh ? ' fresh' : ''}` }, m.door ? avatar(peer, 56) : icon(m.ladder ? 'photo' : 'heart', 22),
    h('span', {}, markerText(m, peer)),
    theirPromise ? h('button', { class: 'btn sm', type: 'button', onclick: () => promiseTomorrow(peerId) }, t('meToo')) : null);
}

// ladderBar is the reveal ladder of an open chat: face, then talk, then
// photos once both people choose to get closer.
function ladderBar(peerId, c) {
  const step = (label, done) => h('span', { class: `step${done ? ' done' : ''}` }, label);
  const action = c.ladder === 'close' ? h('span', { class: 'ladder-note' }, t('closeDone'))
    : c.ladder === 'asked' ? h('span', { class: 'ladder-note' }, t('closerWaiting', { name: c.peer.name }))
      : h('button', { class: 'btn sm primary', type: 'button', onclick: () => send({ type: 'closer', user_id: peerId }) },
        c.ladder === 'invited' ? t('closerAccept') : t('closer'));
  return h('div', { class: 'ladder', role: 'group', 'aria-label': t('ladder') },
    h('span', { class: 'steps' }, step(t('stepFace'), true), step(t('stepTalk'), true), step(t('stepPhotos'), c.ladder === 'close')),
    c.ladder === 'invited' ? h('span', { class: 'ladder-note' }, t('closerAsked', { name: c.peer.name })) : null,
    action);
}

// endedBar replaces the composer once the other person has gone: the bar
// goes dark so nobody types into a chat that can no longer reach anyone.
function endedBar(text) {
  return h('div', { class: 'composer ended', role: 'status' }, icon('door', 18), h('span', {}, text));
}

// pastBar closes a kept chat the same way. If they are here again, a new
// knock starts a new chat: a door never reopens on its own.
function pastBar(peerId) {
  return h('div', { class: 'composer ended', role: 'status' }, icon('door', 18), h('span', {}, t('pastEnded')),
    state.online.has(peerId) ? h('button', { class: 'btn sm', type: 'button', onclick: () => openChat(peerId) }, t('knockAgain')) : null);
}

// ---------- Past chats, kept on this device ----------

// The server deletes messages and their photos after DATA_RETENTION (7 days
// by default), so older chats would only show broken pictures. The caps
// keep the store well under the browser's 5 MB localStorage quota.
const PAST_DAYS = 7;
const PAST_CHATS = 30;
const PAST_MESSAGES = 100;

// keptChat is the part of a chat worth keeping: who it was with and what
// was said.
function keptChat(c) {
  const { id, name, gender, age, country, tags, photo_url: photoURL } = c.peer;
  return { peer: { id, name, gender, age, country, tags, photo_url: photoURL }, msgs: c.msgs.slice(-PAST_MESSAGES), ts: lastTs(c) };
}

// recentChats drops chats past retention and keeps the newest PAST_CHATS.
function recentChats(chats, now) {
  return chats.filter((chat) => withinDays(chat.ts, PAST_DAYS, now)).sort((a, b) => b.ts - a.ts).slice(0, PAST_CHATS);
}

const chatsByPeer = (chats) => new Map(chats.map((chat) => [chat.peer.id, chat]));

function storedArchive(userId) {
  return chatsByPeer(recentChats(readJSON(chatsKey(userId)) || [], Date.now()));
}

// saveArchive copies every chat that has messages into the archive, so it
// is there after the tab closes.
function saveArchive() {
  if (!state.me) return;
  const live = [...state.convos.values()].filter((c) => c.msgs.some(isMessage)).map(keptChat);
  const merged = new Map([...state.archive, ...chatsByPeer(live)]);
  const chats = recentChats([...merged.values()], Date.now());
  state.archive = chatsByPeer(chats);
  writeJSON(chatsKey(state.me.id), chats);
}

// Saving serializes every kept chat, so it waits for a 2 s lull; pagehide
// and the tab going to the background flush it at once.
const saveArchiveSoon = debounce(saveArchive, 2000);
window.addEventListener('pagehide', saveArchive);

// ---------- Closeness: memory, promises, letters, glow ----------

// sharedTags are the "here to" tags two people have in common.
function sharedTags(peer) {
  const mine = new Set((state.me?.tags || []).map((tag) => tag.toLowerCase()));
  return (peer.tags || []).filter((tag) => mine.has(tag.toLowerCase()));
}

// Notes are read from the device once per visit (startRoom) and kept in
// state, since every draw of a chat shows the note.
const readNotes = () => state.notes;

function writeNote(peerId, text) {
  const notes = { ...state.notes };
  if (text) notes[peerId] = text; else delete notes[peerId];
  state.notes = notes;
  writeJSON(notesKey(state.me.id), notes);
}

// dayLabel says when, the way people do: earlier today, yesterday, or the day.
function dayLabel(unix, now = Date.now()) {
  if (sameDay(unix * 1000, now)) return 'earlier today';
  if (sameDay(unix * 1000, now - DAY_MS)) return 'yesterday';
  return `on ${new Date(unix * 1000).toLocaleDateString([], { weekday: 'long', month: 'short', day: 'numeric' })}`;
}

// memoryCard reminds you of someone you have met before: when you last
// talked, what you share, and your own note. Being remembered is what
// lonely people come back for.
function memoryCard(peer) {
  const met = state.metBefore.get(peer.id);
  const shared = sharedTags(peer);
  const note = readNotes()[peer.id];
  if (!met && !shared.length && !note) return null;
  return h('div', { class: 'memory' },
    met ? h('b', {}, t(met.count === 1 ? 'talkedBeforeOne' : 'talkedBefore', { when: dayLabel(met.ts), n: met.count })) : null,
    shared.length ? h('span', {}, t('sharedTags', { tags: shared.join(', ') })) : null,
    note ? h('span', { class: 'memory-note' }, note) : null);
}

// Promises to pick a chat up tomorrow last a few days, then lapse quietly.
const PROMISE_DAYS = 3;
const livePromises = (promises, now) => promises.filter((p) => withinDays(p.ts, PROMISE_DAYS, now));
const isPromised = (peerId) => state.promises.some((p) => p.id === peerId);

// promiseTomorrow keeps the promise on this device and tells the other side.
function promiseTomorrow(peerId) {
  const c = state.convos.get(peerId);
  if (!c) return;
  send({ type: 'tomorrow', user_id: peerId });
  state.promises = [{ id: peerId, name: c.peer.name, ts: Math.floor(Date.now() / 1000) },
    ...livePromises(state.promises, Date.now()).filter((p) => p.id !== peerId)];
  writeJSON(promisesKey(state.me.id), state.promises);
  refreshPerson(peerId);
  addMarker(c, { tomorrow: 'me' });
  toast(t('tomorrowSaved', { name: c.peer.name }));
}

// greetPromised says once per visit that someone you promised is here.
function greetPromised(card) {
  if (!isPromised(card.id) || state.greeted.has(card.id)) return;
  state.greeted.add(card.id);
  toast(t('promiseHere', { name: card.name }), false, `promise-${card.id}`);
}

// Letters are kept on this device, newest first.
const LETTERS_KEEP = 30;

function keepLetter(letter) {
  state.letters = [{ ...letter, read: false }, ...state.letters.filter((l) => l.id !== letter.id)].slice(0, LETTERS_KEEP);
  writeJSON(lettersKey(state.me.id), state.letters);
}

function openLetter(peerId, draft = '') {
  const peer = state.convos.get(peerId)?.peer;
  if (!peer) return;
  const body = h('textarea', { class: 'input', rows: 8, maxlength: 1000, value: draft, placeholder: t('letterPh', { name: peer.name }) });
  formSheet({
    title: t('letter'), help: t('letterHelp'), field: body, submitLabel: t('letterSend'),
    onSubmit: () => {
      const content = body.value.trim();
      if (!content) return false;
      send({ type: 'letter', to: peerId, content });
      return true;
    },
  });
}

function openLetters() {
  const d = sheet(h('div', { class: 'dlg' },
    h('h3', {}, t('letters')),
    state.letters.length ? h('div', { class: 'letters' }, state.letters.map((l) => h('article', { class: `letter${l.read ? '' : ' unread'}` },
      h('b', {}, l.name), h('small', {}, dayLabel(l.sent_at)), h('p', {}, l.content)))) : h('p', { class: 'fine' }, t('noLetters')),
    h('div', { class: 'actions' }, h('button', { class: 'btn primary', type: 'button', onclick: () => d.close() }, t('close')))));
  state.letters = state.letters.map((l) => ({ ...l, read: true }));
  writeJSON(lettersKey(state.me.id), state.letters);
  rerenderChats();
}

// Openers are friendly questions for a quiet start; the night question
// changes once a day.
const OPENERS = [
  'What kept you up tonight?', 'What is a small thing that made your day better?',
  'If you could be anywhere right now, where?', 'What song is stuck in your head?',
  'What is something you are looking forward to?', 'Tea, coffee or something stronger tonight?',
  'What is the best thing you have read or watched lately?', 'Are you a night owl by choice or by accident?',
  'What is one thing people get wrong about you?', 'What would you do with a free day tomorrow?',
];

// opener picks a question, from what the two of you share when you can.
function opener(peer, now) {
  const shared = sharedTags(peer).flatMap((tag) => [`What got you into ${tag}?`, `What is the best part of ${tag} for you?`]);
  const tonight = OPENERS[Math.floor(now / DAY_MS) % OPENERS.length];
  const pool = shared.length ? [...shared, tonight] : [tonight, ...OPENERS];
  return pool[Math.floor(Math.random() * pool.length)];
}

// A chat warms after a quarter of an hour of real back-and-forth, grows
// close when it keeps going fast, and glows when the same two come back
// on another day. It reads only counts and times, never the words.
const WARM_SECONDS = 15 * 60;
const CLOSE_SECONDS = 45 * 60;
const WARM_REPLIES = 20;
const QUICK_GAP_SECONDS = 120;

function closeness(msgs, myId, metBeforeTs, now) {
  if (metBeforeTs && !sameDay(metBeforeTs * 1000, now)) return 'glow';
  const said = msgs.filter(isMessage);
  const mine = said.filter((m) => m.from === myId).length;
  if (said.length < 2 || mine < WARM_REPLIES || said.length - mine < WARM_REPLIES) return null;
  const span = said[said.length - 1].ts - said[0].ts;
  if (span < WARM_SECONDS) return null;
  const recent = said.slice(-WARM_REPLIES);
  const gap = (recent[recent.length - 1].ts - recent[0].ts) / (recent.length - 1);
  return span >= CLOSE_SECONDS && gap < QUICK_GAP_SECONDS ? 'close' : 'warm';
}

// glowOf is the closeness to show for a chat: none for chats that are not
// open, or when the person chose to keep it plain.
function glowOf(peerId, c) {
  if (state.plain || c.phase !== 'open') return null;
  return closeness(c.msgs, state.me.id, state.metBefore.get(peerId)?.ts, Date.now());
}

// heartFor is a small heart beside a close chat; it beats while the chat
// is in full flow.
function heartFor(glow) {
  if (glow !== 'close' && glow !== 'glow') return null;
  return h('span', { class: `heart${glow === 'close' ? ' beat' : ''}`, 'aria-hidden': 'true' }, icon('heart', 14));
}

function setPlain(plain) {
  state.plain = plain;
  writeStore(PLAIN_KEY, plain ? '1' : null);
  rerenderChats();
}

function dmComposer(peerId, c) {
  const phase = c.phase;
  const canSend = phase !== 'outgoing';
  const input = h('input', {
    class: 'input', id: 'dm-input', maxlength: 1000, autocomplete: 'off', disabled: !canSend,
    placeholder: { none: t('knockPh'), incoming: t('replyPh'), outgoing: t('knockSent') }[phase] ?? t('msgPh'), 'aria-label': t('msgPh'),
  });
  return h('form', {
    class: 'composer',
    onsubmit: (e) => {
      e.preventDefault();
      const content = input.value.trim();
      if (!content) return;
      send({ type: 'dm', to: peerId, content });
      input.value = '';
    },
  },
    // A knock must be words; photos and GIFs wait until both get closer.
    phase === 'open' && c.ladder === 'close' ? mediaButtons('dm', (m) => send({ type: 'dm', to: peerId, media_id: m.id })) : null,
    canSend ? h('button', {
      class: 'icon-btn bare', type: 'button', title: t('opener'), 'aria-label': t('opener'),
      onclick: () => { input.value = opener(c.peer, Date.now()); input.focus(); },
    }, icon('spark')) : null,
    input,
    h('button', { class: 'icon-btn send', type: 'submit', disabled: !canSend, 'aria-label': t('send') }, icon('send')));
}

// rerenderChats redraws the chats pane at most once a frame, so a burst of
// messages costs one redraw, not one per message.
let chatsFrame = 0;
function rerenderChats() {
  if (!chatsFrame) chatsFrame = requestAnimationFrame(() => { chatsFrame = 0; redrawChats(); });
}

// showNewMessage puts a message that did not change the chat's phase on
// screen without redrawing the chat: pictures and banners already showing
// stay as they are. Anything it cannot place exactly falls back to a redraw.
function showNewMessage(peerId, c, m) {
  const list = $('#side .convo-list');
  const glow = glowOf(peerId, c);
  const view = $(`#side .convo[data-peer="${peerId}"]`);
  if (chatsFrame || state.side !== 'chats' || !list || (view && (view.dataset.glow ?? null) !== glow)) {
    rerenderChats();
    return;
  }
  const bubbles = view?.querySelector('.bubbles');
  const atBottom = bubbles ? nearBottom(bubbles) : false; // read layout before any write
  renderMobileTabs();
  $('#side .tabs')?.replaceWith(sideTabs());
  const row = list.querySelector(`.convo-item[data-peer="${peerId}"]`);
  row?.remove();
  const newest = list.querySelector('.convo-item:not(.knock-in):not(.letters-item)');
  list.insertBefore(convoItem(peerId, c, glow), newest);
  if (!bubbles) return;
  const mine = m.from === state.me.id;
  const prev = c.msgs[c.msgs.length - 2];
  // The new message may continue the last run: its tail and time move down.
  const prevRow = isMessage(prev) && bubbles.querySelector(`.msg[data-id="${prev.id}"]`);
  if (prevRow && (prev.from === state.me.id) === mine && m.ts - prev.ts <= GROUP_SECONDS) {
    prevRow.classList.remove('last');
    prevRow.querySelector('.stamp')?.remove();
  }
  bubbles.append(bubbleRow(m, prev, undefined, mine));
  if (atBottom || mine) bubbles.scrollTop = bubbles.scrollHeight;
}

function redrawChats() {
  renderMobileTabs();
  renderKnockTray();
  if (state.side !== 'chats') {
    $('#side .tabs')?.replaceWith(sideTabs()); // only the unread badge changed
    return;
  }
  const focused = document.activeElement?.id === 'dm-input';
  const draft = $('#dm-input')?.value || '';
  renderSide();
  const input = $('#dm-input');
  if (input && !input.disabled) {
    input.value = draft;
    if (focused) input.focus();
  }
}

function blockPerson(peerId) {
  send({ type: 'block', user_id: peerId });
  state.convos.delete(peerId);
  state.archive.delete(peerId);
  saveArchive();
  state.online.delete(peerId);
  if (state.active === peerId) state.active = null;
  forgetMentionNames();
  toast(t('blocked'));
  renderSide();
  refreshCounts();
}

// ---------------------------------------------------------------------------
// Dialogs
// ---------------------------------------------------------------------------

function dialog(...children) {
  const d = h('dialog', { onclose: () => d.remove() }, ...children);
  d.addEventListener('click', (e) => { if (e.target === d) d.close(); });
  document.body.append(d);
  d.showModal();
  return d;
}

function openLightbox(url) {
  const d = dialog(h('img', { src: url, alt: '' }));
  d.classList.add('lightbox');
  d.addEventListener('click', () => d.close());
}

function openPerson(id, fallbackName) {
  const card = state.online.get(id);
  if (!card) { toast(`${fallbackName || 'They'} left.`); return; }
  const c = state.convos.get(id);
  const facts = [genderLabel(c?.peer.gender || card.gender), card.age, ...(card.tags || []), card.location, hereFor(card.online_since)].filter(Boolean);

  const modBox = isStaff() && card.role === 'user' && h('div', { class: 'mod-box' },
    h('b', {}, t('desk')),
    h('div', { class: 'actions', style: { justifyContent: 'flex-start' } },
      h('button', { class: 'btn sm', type: 'button', onclick: () => { modAct(`/admin/users/${id}/kick`); d.close(); } }, t('kick')),
      h('button', { class: 'btn sm', type: 'button', onclick: () => { modAct(`/admin/users/${id}/mute`, { minutes: 30 }); d.close(); } }, t('mute30')),
      h('button', { class: 'btn sm danger', type: 'button', onclick: () => { modAct(`/admin/users/${id}/ban`, { minutes: 1440, ip: false, reason: 'from card' }); d.close(); } }, t('ban24')),
      h('button', { class: 'btn sm danger', type: 'button', onclick: () => { modAct(`/admin/users/${id}/ban`, { minutes: 1440, ip: true, reason: 'from card' }); d.close(); } }, t('banIP'))));

  const d = dialog(h('div', { class: 'dlg' },
    h('div', { class: 'sheet-photo', 'data-aura': card.aura || null, style: pictureStyle({ ...card, photo_url: c?.peer.photo_url }) }),
    h('h3', {}, whoName(card), ' ', flag(card.country)),
    card.about ? h('p', { style: { margin: 0, lineHeight: 1.5, color: 'var(--text-2)' } }, card.about) : null,
    h('div', { class: 'facts' }, facts.map((f) => h('span', { class: 'fact' }, f))),
    h('div', { class: 'actions' },
      h('button', { class: 'btn ghost', type: 'button', onclick: () => { d.close(); openReport('user', id); } }, t('report')),
      h('button', { class: 'btn', type: 'button', onclick: () => { d.close(); blockPerson(id); } }, t('block')),
      h('button', { class: 'btn', type: 'button', onclick: () => { d.close(); mentionInRoom(card.name); } }, icon('at', 16), t('mention')),
      h('button', { class: 'btn primary', type: 'button', onclick: () => { d.close(); openChat(id); } }, c && isStarted(c) ? t('open') : t('knock'))),
    modBox));
}

// mentionInRoom starts a room message addressed to name.
function mentionInRoom(name) {
  if (isPhone() && state.view !== 'room') showView('room');
  const input = $('#room-input');
  if (!input) return;
  const lead = input.value && !input.value.endsWith(' ') ? ' ' : '';
  input.value += `${lead}@${name} `;
  input.focus();
}

// sheet is a dialog that rises from the bottom on phones (an action sheet)
// and sits in the middle on larger screens.
function sheet(...children) {
  const d = dialog(...children);
  d.classList.add('sheet');
  return d;
}

function openChatMenu(peerId) {
  const peer = (state.convos.get(peerId) ?? pastConvo(peerId)).peer;
  const item = (label, onPick, cls = '') => h('button', { class: `sheet-item ${cls}`, type: 'button', onclick: () => { d.close(); onPick(); } }, label);
  const open = state.convos.get(peerId)?.phase === 'open';
  const letters = open && state.me.kind === 'member' && state.online.get(peerId)?.kind === 'member';
  const d = sheet(h('div', { class: 'sheet-list' },
    h('div', { class: 'sheet-head' }, avatar(peer, 36), whoName(peer, 'b')),
    state.online.has(peerId) ? item(t('viewCard'), () => openPerson(peerId, peer.name)) : null,
    open && !isPromised(peerId) ? item(t('tomorrow'), () => promiseTomorrow(peerId)) : null,
    letters ? item(t('letter'), () => openLetter(peerId)) : null,
    item(t('selfNote'), () => openNote(peer)),
    open ? item(state.plain ? t('glowOn') : t('plain'), () => setPlain(!state.plain)) : null,
    item(t('report'), () => openReport('user', peerId), 'danger'),
    item(t('block'), () => blockPerson(peerId), 'danger'),
    item(t('close'), () => {}, 'muted')));
}

function openNote(peer) {
  const input = h('input', { class: 'input', maxlength: 140, value: readNotes()[peer.id] || '', placeholder: t('selfNotePh') });
  formSheet({
    title: t('selfNote'), help: t('selfNoteHelp'), field: input, submitLabel: t('save'),
    onSubmit: () => {
      writeNote(peer.id, input.value.trim());
      toast(t('saved'));
      rerenderChats();
      return true;
    },
  });
}

// formSheet is a one-field form in a sheet: a title, a line of help, the
// field, and Close beside submit. onSubmit returns false to keep it open.
function formSheet({ title, help, field, submitLabel, onSubmit }) {
  const d = sheet(h('form', {
    class: 'dlg',
    onsubmit: (e) => {
      e.preventDefault();
      if (onSubmit()) d.close();
    },
  },
    h('h3', {}, title),
    h('p', { class: 'fine' }, help),
    field,
    h('div', { class: 'actions' },
      h('button', { class: 'btn ghost', type: 'button', onclick: () => d.close() }, t('close')),
      h('button', { class: 'btn primary', type: 'submit' }, submitLabel))));
  setTimeout(() => field.focus());
}

// switchRow is an on/off setting: a label, optional help, and a switch.
function switchRow(label, help, checked, onChange) {
  return h('label', { class: 'switch-row' },
    h('span', {}, h('b', {}, label), help ? h('small', {}, help) : null),
    h('input', { type: 'checkbox', role: 'switch', class: 'switch', checked, onchange: (e) => onChange(e.target.checked) }));
}

// openReconsider asks before a message with flagged words goes out, the
// way a friend would. Edit puts the words back; send anyway goes through.
function openReconsider(error) {
  const frame = state.lastText;
  if (!frame) { toast(error, true); return; }
  const d = sheet(h('div', { class: 'dlg' },
    h('h3', {}, t('reconsiderTitle')),
    h('p', { class: 'fine' }, t('reconsiderHelp')),
    h('div', { class: 'knock-quote' }, frame.content),
    h('div', { class: 'actions' },
      h('button', { class: 'btn ghost', type: 'button', onclick: () => { d.close(); restoreDraft(frame); } }, t('edit')),
      h('button', { class: 'btn primary', type: 'button', onclick: () => { d.close(); send({ ...frame, confirm: true }); } }, t('sendAnyway')))));
}

function restoreDraft(frame) {
  if (frame.type === 'letter') { openLetter(frame.to, frame.content); return; }
  const input = $(frame.type === 'chat' ? '#room-input' : '#dm-input');
  if (!input) return;
  input.value = frame.content;
  input.focus();
}

function openSounds() {
  const s = soundSettings();
  const toggle = (key, label, help) => switchRow(label, help, s[key], (on) => {
    setSoundSettings({ [key]: on });
    $('#sound-btn')?.replaceChildren(soundIcon());
  });
  const sample = (kind, label) => h('button', { class: 'sample', type: 'button', onclick: () => play(kind) }, icon('play', 14), label);
  const d = sheet(h('div', { class: 'dlg' },
    h('h3', {}, t('sounds')),
    toggle('on', t('soundsOn')),
    toggle('knock', t('knockSounds'), t('knockSoundsHelp')),
    toggle('room', t('roomSounds'), t('roomSoundsHelp')),
    'vibrate' in navigator ? toggle('vibrate', t('vibrate')) : null,
    h('p', { class: 'fine' }, 'Every event has its own sound. Tap to hear them.'),
    h('div', { class: 'samples' },
      sample('knock', 'Knock'), sample('mention', 'Mention'), sample('door', 'Door opens'),
      sample('dm', 'Message'), sample('tick', 'In chat'), sample('room', 'Room')),
    h('div', { class: 'actions' }, h('button', { class: 'btn primary', type: 'button', onclick: () => d.close() }, t('close')))));
}

// ---------------------------------------------------------------------------
// Photos, GIFs and recents
// ---------------------------------------------------------------------------

// Recents are images this browser sent, newest first, so a favourite GIF is
// one tap away next time. They stay on this device only.
const RECENTS_KEY = 'drift.recents';
const RECENTS_KEEP = 30;

const readRecents = () => readJSON(RECENTS_KEY) || [];
const writeRecents = (list) => writeJSON(RECENTS_KEY, list.length ? list.slice(0, RECENTS_KEEP) : null);
// remember records an uploaded image (a media View) as sent by me.
function remember(m) {
  const entry = { id: m.id, owner: state.me.id, url: m.url, thumb: m.thumb_url };
  writeRecents([entry, ...readRecents().filter((r) => r.url !== m.url)]);
}
function forget(url) {
  writeRecents(readRecents().filter((r) => r.url !== url));
}

// sendable turns a recent into an image I may share now. An image belongs to
// the profile that uploaded it, so after a new visit it is uploaded again
// from its own URL; one removed since then is dropped from recents.
async function sendable(r) {
  if (r.owner === state.me.id) return r;
  const res = await fetch(r.url);
  if (!res.ok) {
    forget(r.url);
    throw new Error('That one is no longer available.');
  }
  const blob = await res.blob();
  const up = await uploadPhoto(new File([blob], r.url.split('/').pop(), { type: blob.type }));
  forget(r.url);
  return up;
}

// mediaButtons is the photo and GIF buttons of a composer; onSend gets an
// uploaded image ({ id, url, thumb_url }) to post.
function mediaButtons(where, onSend) {
  const deliver = (m) => { remember(m); onSend(m); };
  const fileId = `${where}-photo`;
  return [
    h('label', { class: 'icon-btn', for: fileId, title: t('sendPhoto'), style: { cursor: 'pointer' } }, icon('photo'), h('span', { class: 'sr' }, t('sendPhoto'))),
    filePicker(fileId, deliver),
    h('button', { class: 'icon-btn gif-btn', type: 'button', title: t('gifTitle'), 'aria-label': t('gifTitle'), onclick: () => openGifPicker(deliver) }, t('gif')),
  ];
}

// filePicker is a hidden file input that uploads what is chosen.
function filePicker(id, onUploaded, onError = (msg) => toast(msg, true)) {
  return h('input', {
    type: 'file', accept: IMAGE_TYPES, class: 'sr', id,
    onchange: async (e) => {
      const file = e.target.files[0];
      e.target.value = '';
      if (!file) return;
      try { onUploaded(await uploadPhoto(file)); } catch (ex) { onError(ex.message); }
    },
  });
}

function openGifPicker(deliver) {
  let tab = readRecents().length || state.gifSearchOff ? 'recent' : 'gifs';
  const body = h('div', { class: 'picker-body' });
  const tabs = h('div', { class: 'enter-tabs', role: 'tablist' });
  const d = sheet(h('div', { class: 'picker' },
    h('div', { class: 'picker-top' }, tabs,
      h('button', { class: 'icon-btn bare', type: 'button', 'aria-label': t('close'), onclick: () => d.close() }, icon('x'))),
    body));
  d.classList.add('tall');

  // pick runs one send at a time and closes the picker when it lands.
  let busy = false;
  const pick = async (tile, getImage) => {
    if (busy) return;
    busy = true;
    tile.classList.add('loading');
    try {
      deliver(await getImage());
      d.close();
    } catch (ex) {
      toast(ex.message, true);
      tile.classList.remove('loading');
    } finally {
      busy = false;
    }
  };

  const drawTabs = () => tabs.replaceChildren(...[['recent', t('recent')], ['gifs', t('gifs')]].map(([key, label]) =>
    h('button', { type: 'button', role: 'tab', class: 'tab', 'aria-selected': String(tab === key), onclick: () => { tab = key; draw(); } }, label)));
  // Each draw of the GIF tab gets a fresh signal; switching tabs or closing
  // the picker aborts it.
  let drawn = new AbortController();
  d.addEventListener('close', () => drawn.abort());
  const draw = () => {
    drawn.abort();
    drawn = new AbortController();
    drawTabs();
    body.replaceChildren();
    append(body, [tab === 'recent' ? recentGrid(pick, deliver, d) : gifSearch(pick, drawn.signal)]);
  };
  draw();
}

function recentGrid(pick, deliver, d) {
  const recents = readRecents();
  const grid = h('div', { class: 'gif-grid' },
    h('label', { class: 'gif-tile upload', for: 'picker-photo' }, icon('photo', 22), h('span', {}, t('upload'))),
    filePicker('picker-photo', (m) => { deliver(m); d.close(); }),
    recents.map((r) => {
      const tile = h('button', { class: 'gif-tile', type: 'button', onclick: () => pick(tile, () => sendable(r)) },
        h('img', { src: r.thumb, alt: '', loading: 'lazy', onerror: () => { forget(r.url); tile.remove(); } }));
      return tile;
    }));
  return [
    grid,
    recents.length
      ? h('button', { class: 'btn sm ghost', type: 'button', onclick: (e) => { writeRecents([]); const body = e.target.closest('.picker-body'); body.replaceChildren(); append(body, [recentGrid(pick, deliver, d)]); } }, t('clearRecents'))
      : h('p', { class: 'fine center' }, t('noRecents')),
  ];
}

// gifSearch is the library tab: trending first, then a search as you type.
// It stops watching for more pages when signal aborts.
function gifSearch(pick, signal) {
  const grid = h('div', { class: 'gif-grid' });
  const status = h('p', { class: 'fine center' });
  const credit = h('p', { class: 'credit' });
  const more = h('div', { class: 'sentinel' });
  const input = h('input', { class: 'input', type: 'search', placeholder: t('gifSearch'), 'aria-label': t('gifSearch'), autocomplete: 'off' });
  let query = '';
  let next = '';
  let loading = false;
  let seq = 0;

  const load = async (reset) => {
    if (loading && !reset) return;
    const mine = ++seq;
    loading = true;
    if (reset) { grid.replaceChildren(); next = ''; }
    try {
      const page = await api('GET', `/gifs?q=${encodeURIComponent(query)}&next=${encodeURIComponent(next)}`);
      if (mine !== seq) return;
      state.gifSearchOff = false;
      credit.textContent = page.provider === 'giphy' ? 'Powered by GIPHY' : 'Search via Tenor';
      next = page.next || '';
      grid.append(...page.results.map((g) => {
        const tile = h('button', { class: 'gif-tile', type: 'button', title: g.title || '', style: { aspectRatio: `${g.width || 1} / ${g.height || 1}` },
          onclick: () => pick(tile, () => api('POST', '/gifs/import', { id: g.id })) },
        h('img', { src: g.preview_url, alt: g.title || '', loading: 'lazy', referrerpolicy: 'no-referrer' }));
        return tile;
      }));
      status.textContent = grid.children.length ? '' : t('gifNone');
    } catch (ex) {
      if (mine !== seq) return;
      if (ex.status === 404) state.gifSearchOff = true;
      status.textContent = ex.status === 404 ? t('gifOff') : ex.message;
    } finally {
      if (mine === seq) loading = false;
    }
  };

  input.addEventListener('input', debounce(() => { query = input.value.trim(); load(true); }, 350));
  // Load the next page as the end of the grid scrolls into view.
  const watcher = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && next) load(false);
  });
  watcher.observe(more);
  signal.addEventListener('abort', () => watcher.disconnect());
  if (state.gifSearchOff) status.textContent = t('gifOff');
  else load(true);
  setTimeout(() => input.focus());
  return [state.gifSearchOff ? null : input, status, grid, more, credit];
}

function openReport(targetType, targetId) {
  let reason = '';
  const err = h('div', { class: 'error-text', role: 'alert' });
  const note = h('textarea', { class: 'input', rows: 2, maxlength: 500 });
  const d = dialog(h('form', {
    class: 'dlg',
    onsubmit: async (e) => {
      e.preventDefault();
      if (!reason) { err.textContent = 'Pick a reason.'; return; }
      try {
        await api('POST', '/reports', { target_type: targetType, target_id: targetId, reason, note: note.value.trim() });
        d.close();
        toast(t('reported'));
      } catch (ex) { err.textContent = ex.message; }
    },
  },
    h('h3', {}, t('reportTitle')),
    h('p', { class: 'fine' }, t('reportHelp')),
    chipGroup(t('reportTitle'), REASONS, '', (v) => { reason = v; }),
    h('label', { class: 'field' }, h('span', {}, t('note')), note),
    err,
    h('div', { class: 'actions' },
      h('button', { class: 'btn ghost', type: 'button', onclick: () => d.close() }, t('close')),
      h('button', { class: 'btn primary', type: 'submit' }, t('submit')))));
}

function openEditCard() {
  const me = state.me;
  const tagBox = tagEditor([...(me.tags || [])]);
  let photoId; // undefined = unchanged, null = clear
  const err = h('div', { class: 'error-text', role: 'alert' });
  const preview = h('div', { class: 'sheet-photo', style: { height: '180px', ...pictureStyle(me) } });
  const about = h('input', { class: 'input', maxlength: 140, value: me.about || '', placeholder: t('aboutPh') });
  const loc = h('input', { class: 'input', maxlength: 40, value: me.location || '', placeholder: t('locationPh') });
  const photoIn = filePicker('edit-photo', (up) => {
    photoId = up.id;
    Object.assign(preview.style, { backgroundImage: `url("${up.url}")`, imageRendering: '' });
  }, (msg) => { err.textContent = msg; });
  const settings = { aura: me.aura || '', knocks_from: me.knocks_from || 'anyone', hide_gender: !!me.hide_gender };
  const showAura = () => { preview.dataset.aura = settings.aura; };
  // Settings save as they change, like any toggle: a glow that previews but
  // is lost when the sheet closes without Save reads as a broken setting.
  const saveSetting = async (patch) => {
    err.textContent = '';
    try {
      state.me = await api('PATCH', '/me', patch);
      toast(t('saved'), false, 'setting');
      $('.me-pill')?.replaceWith(mePill());
    } catch (ex) { err.textContent = ex.message; }
  };
  const auraLabels = Object.fromEntries(Object.entries(AURAS).map(([key, label]) => [key, [h('i', { class: 'swatch', 'data-aura': key || null }), label]]));
  const d = dialog(h('form', {
    class: 'dlg',
    onsubmit: async (e) => {
      e.preventDefault();
      const body = { tags: tagBox.value(), about: about.value.trim(), location: loc.value.trim() };
      if (photoId !== undefined) body.photo_id = photoId;
      try {
        state.me = await api('PATCH', '/me', body);
        rememberProfile(state.me);
        d.close();
        toast(t('saved'));
        renderShell();
      } catch (ex) { err.textContent = ex.message; }
    },
  },
    h('h3', {}, t('editCard')),
    preview,
    me.kind === 'member'
      ? h('div', { class: 'actions', style: { justifyContent: 'flex-start' } },
        h('label', { class: 'btn sm', for: 'edit-photo' }, t('changePhoto')), photoIn,
        h('button', { class: 'btn sm ghost', type: 'button', onclick: () => { photoId = null; Object.assign(preview.style, pictureStyle({ name: me.name })); } }, t('removePhoto')))
      : h('p', { class: 'fine' }, 'Guests wear the face drawn from their name. Create an account to add your own photo.'),
    h('div', { class: 'field' }, h('span', {}, t('aura')),
      chipGroup(t('aura'), auraLabels, settings.aura, (v) => { settings.aura = v; showAura(); saveSetting({ aura: v }); }),
      h('small', { class: 'fine' }, t('auraHelp'))),
    h('div', { class: 'field' }, h('span', {}, t('knocksFrom')),
      chipGroup(t('knocksFrom'), KNOCKS, settings.knocks_from, (v) => saveSetting({ knocks_from: v }))),
    switchRow(t('hideGender'), t('hideGenderHelp'), settings.hide_gender, (on) => saveSetting({ hide_gender: on })),
    h('div', { class: 'field' }, h('span', {}, t('tags')), tagBox.el),
    h('label', { class: 'field' }, h('span', {}, t('location')), loc),
    h('label', { class: 'field' }, h('span', {}, t('about')), about),
    err,
    h('div', { class: 'actions' },
      h('button', { class: 'btn ghost', type: 'button', onclick: () => d.close() }, t('close')),
      h('button', { class: 'btn primary', type: 'submit' }, t('save')))));
}

async function modAct(path, body) {
  try {
    await api('POST', path, body || {});
    toast('Done.');
    return true;
  } catch (ex) {
    toast(ex.message, true);
    return false;
  }
}

// ---------------------------------------------------------------------------
// WebSocket boundary
// ---------------------------------------------------------------------------

function connect() {
  if (!state.token || !conn.send('connect')) return;
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const ws = new WebSocket(`${proto}://${location.host}/ws?token=${encodeURIComponent(state.token)}`);
  state.ws = ws;
  let opened = false;

  ws.onopen = () => { opened = true; state.retry = 0; };
  // Frames that queued up for us arrive together, one per line.
  ws.onmessage = (e) => {
    for (const line of e.data.split('\n')) {
      let frame;
      try { frame = JSON.parse(line); } catch { continue; }
      (HANDLERS[frame.type] || (() => {}))(frame);
    }
  };
  ws.onclose = async () => {
    if (state.ws !== ws || conn.is('closed')) return;
    state.ws = null;
    conn.send('drop');
    if (!opened) {
      // The upgrade was refused: find out whether the session is still valid.
      try { await api('GET', '/me'); } catch (ex) {
        if (ex.status === 401 || ex.status === 403) { endSession(ex.message); return; }
      }
    }
    if (!conn.is('retrying')) return; // left while the check ran
    toast(t('reconnecting'));
    const delay = Math.min(15000, 500 * 2 ** state.retry++);
    setTimeout(() => { if (conn.is('retrying')) connect(); }, delay);
  };
}

function send(frame) {
  if (state.ws?.readyState !== WebSocket.OPEN) {
    toast(t('reconnecting'), true);
    return;
  }
  if (frame.content) state.lastText = frame;
  state.ws.send(JSON.stringify(frame));
}

const HANDLERS = {
  hello(f) {
    if (!conn.send('hello')) return;
    state.me = f.you;
    rememberProfile(f.you);
    state.online = new Map(f.online.filter((c) => c.id !== f.you.id).map((c) => [c.id, c]));
    forgetMentionNames();
    peopleDirty = true;
    // The server is the source of truth for private chats: doors survive a
    // quick reconnect but close when either side really disconnects.
    const doors = new Map((f.doors || []).map((d) => [d.with, d]));
    for (const id of doors.keys()) if (state.online.has(id)) convo(id);
    for (const [id, c] of state.convos) syncChat(id, c, state.online.get(id), doors.get(id));
    for (const card of state.online.values()) greetPromised(card);
    state.feed = [];
    if (screen.is('room')) renderShell();
    send({ type: 'join', room_id: MAIN_ROOM });
  },

  history(f) {
    if (f.room_id !== MAIN_ROOM) return;
    // History arrives newest first; one rebuild and one scroll for all of it.
    state.feed = f.messages.toReversed().slice(-FEED_KEEP);
    redrawFeed();
  },

  chat(f) {
    if (f.room_id !== MAIN_ROOM) return;
    const atMe = f.sender_id !== state.me?.id && mentionsMe(f.content);
    addFeed(f, atMe);
    if (f.sender_id === state.me?.id) return;
    if (!atMe) {
      notify('room');
      return;
    }
    notify('mention');
    if (!roomVisible()) {
      state.mentions++;
      toast(t('mentioned', { name: f.name }), false, 'mention');
      renderMobileTabs();
    }
  },

  presence(f) {
    forgetMentionNames();
    if (f.user_id === state.me?.id) {
      if (f.user) state.me = { ...state.me, ...f.user, photo_url: f.user.has_photo ? state.me.photo_url : undefined };
      return;
    }
    const c = state.convos.get(f.user_id);
    let moved = false;
    if (f.event === 'leave') {
      state.online.delete(f.user_id);
      if (c) moved = moveChat(f.user_id, c, 'left') === 'moved';
      // A chat never started leaves nothing worth keeping once they go.
      if (c && !c.msgs.length && state.active !== f.user_id) state.convos.delete(f.user_id);
    } else {
      state.online.set(f.user_id, f.user);
      if (f.event === 'join') greetPromised(f.user);
      if (c) {
        // A hidden gender stays known to someone you are talking to.
        const known = c.phase === 'open' ? c.peer.gender : undefined;
        c.peer = { ...f.user, gender: f.user.gender || known, photo_url: f.user.has_photo ? c.peer.photo_url : undefined };
        // The door closed when they left; a new knock starts over.
        if (c.phase === 'gone' && moveChat(f.user_id, c, 'returned') === 'moved') {
          c.msgs = [];
          c.ladder = 'face';
          c.peer.photo_url = undefined;
          moved = true;
        }
      }
    }
    refreshCounts();
    refreshPerson(f.user_id);
    if (moved) rerenderChats();
    renderKnockTray();
  },

  dm(f) {
    const mine = f.from === state.me.id;
    const peerId = mine ? f.to : f.from;
    const c = convo(peerId);
    const move = moveChat(peerId, c, f.knock ? (mine ? 'knockSent' : 'knockIn') : 'message');
    if (move === 'refused') return;
    if (f.knock) {
      c.ladder = 'face';
      c.knockSeen = false;
      c.msgs = []; // a knock opens a new conversation; the old one ended with its door
    }
    keepMessage(c, f);
    if (!mine) {
      const viewing = isViewing(peerId);
      notify(f.knock ? 'knock' : viewing ? 'tick' : 'dm');
      if (!viewing) c.unread++;
    }
    saveArchiveSoon();
    if (move === 'stayed') {
      showNewMessage(peerId, c, f);
      return;
    }
    refreshPerson(peerId); // Knock becomes Open chat on their card
    rerenderChats();
  },

  door(f) {
    const c = convo(f.with.id);
    if (moveChat(f.with.id, c, 'opened') === 'refused') return;
    c.peer = f.with;
    c.ladder = 'face';
    addMarker(c, { door: true });
    notify('door');
    refreshPerson(f.with.id);
  },

  door_closed(f) {
    const c = state.convos.get(f.with);
    if (!c || moveChat(f.with, c, f.by_me ? 'decline' : 'declined') === 'refused') return;
    if (f.by_me) {
      state.convos.delete(f.with);
      if (state.active === f.with) state.active = null;
    }
    refreshPerson(f.with);
    rerenderChats();
  },

  closer(f) {
    const c = state.convos.get(f.with);
    if (!c) return;
    const was = c.ladder;
    const next = nextState(LADDER, was, ladderOf(f));
    if (next == null) {
      refuse(`ladder ${f.with}`, was, ladderOf(f));
      return;
    }
    c.ladder = next;
    if (f.peer) {
      c.peer = { ...c.peer, ...f.peer };
      refreshPerson(f.with); // their photo now shows on the card
    }
    if (next === 'close' && was !== 'close') {
      addMarker(c, { ladder: 'close' });
      notify('door');
      return;
    }
    if (next === 'invited' && was !== 'invited') {
      notify('knock');
      if (!isViewing(f.with)) toast(t('closerAsked', { name: c.peer.name }), false, `closer-${f.with}`);
    }
    saveArchiveSoon();
    rerenderChats();
  },

  tomorrow(f) {
    const c = state.convos.get(f.from);
    if (!c) return;
    addMarker(c, { tomorrow: 'them' });
    notify('dm');
    if (!isViewing(f.from)) toast(t('tomorrowTheirs', { name: c.peer.name }), false, `tomorrow-${f.from}`);
  },

  letter(f) {
    keepLetter(f);
    notify('dm');
    toast(t('letterIn', { name: f.name }), false, `letter-${f.id}`);
    rerenderChats();
  },

  letter_sent(f) {
    toast(t('letterSent', { time: clock(f.deliver_at) }));
  },

  removed(f) {
    const kept = (m) => m.id !== f.message_id;
    state.feed = state.feed.filter(kept);
    for (const c of state.convos.values()) c.msgs = c.msgs.filter(kept);
    // A message a moderator removed must not survive on the device either.
    // Most removals are room lines, which no kept chat holds.
    const holding = [...state.archive].filter(([, chat]) => !chat.msgs.every(kept));
    for (const [id, chat] of holding) state.archive.set(id, { ...chat, msgs: chat.msgs.filter(kept) });
    if (holding.length) saveArchive();
    // Redraw rather than delete the node: the next line may need its face
    // and name back if the removed one headed its group.
    if (screen.is('room')) redrawFeed();
    if (state.side === 'chats') rerenderChats();
  },

  notice(f) {
    if (f.code === 'kicked' || f.code === 'banned') {
      endSession(f.text);
      return;
    }
    toast(f.code === 'report_actioned' ? t('reportThanks') : f.text, f.code !== 'report_actioned');
  },

  error(f) {
    if (f.code === 'RECONSIDER') openReconsider(f.error);
    else toast(f.error, true);
  },
};

// Coming back to the tab clears the mentions you can now see.
document.addEventListener('visibilitychange', () => {
  if (roomVisible() && state.mentions) {
    state.mentions = 0;
    renderMobileTabs();
  }
});

// ---------------------------------------------------------------------------
// Screen: Mod desk
// ---------------------------------------------------------------------------

let deskTab = 'overview';
let deskTimer = null;
const isAdmin = () => state.me?.role === 'admin';

function renderDesk() {
  if (!screen.send('desk')) return;
  clearInterval(deskTimer);
  const back = h('button', { class: 'btn sm', type: 'button', onclick: () => { location.hash = ''; } }, t('backToChat'));
  const main = h('main', {});
  const tabs = [['overview', 'Overview'], ['reports', t('reports')], ['people', 'People online'], ['words', t('words')],
    isAdmin() && ['staff', 'Staff'], ['audit', t('audit')]].filter(Boolean);
  const nav = h('nav', { 'aria-label': t('desk') }, tabs.map(([key, label]) => h('button', {
    type: 'button', 'aria-current': String(deskTab === key),
    onclick: () => { deskTab = key; renderDesk(); },
  }, label)));
  $('#app').replaceChildren(h('div', { class: 'shell' }, topbar(back), h('div', { class: 'desk' }, nav, main)));
  ({ overview: deskOverview, reports: deskReports, people: deskPeople, words: deskWords, staff: deskStaff, audit: deskAudit })[deskTab](main);
}

// ---------- Overview ----------

async function deskOverview(main) {
  const stamp = h('span', { class: 'muted', style: { fontSize: '13px' } });
  const body = h('div', { class: 'overview' });
  main.append(h('div', { class: 'desk-head' }, h('h1', {}, 'Overview'), stamp), body);
  // This page's own timer: it stops itself once the page is gone, whatever
  // happened to the request, and never touches another page's timer.
  const timer = setInterval(() => load(), 15000);
  deskTimer = timer;
  const load = async () => {
    if (!body.isConnected) { clearInterval(timer); return; }
    let o;
    try { o = await api('GET', '/admin/overview'); } catch (ex) {
      if (body.isConnected) body.replaceChildren(h('p', { class: 'error-text' }, ex.message));
      return;
    }
    if (!body.isConnected) { clearInterval(timer); return; }
    stamp.textContent = `Live, updated ${new Date(o.generated_at).toLocaleTimeString()}`;
    body.replaceChildren(...overviewView(o));
  };
  await load();
}

function overviewView(o) {
  const n = (x) => (x ?? 0).toLocaleString();
  const of = (x, one, many) => `${n(x)} ${x === 1 ? one : many}`;
  const tile = (label, value, sub, alert) => h('div', { class: `tile${alert ? ' alert' : ''}` },
    h('span', { class: 'tile-label' }, label), h('span', { class: 'tile-value' }, value), sub ? h('span', { class: 'tile-sub' }, sub) : null);
  const tiles = h('div', { class: 'tiles' },
    tile('Online now', n(o.now.online), `${of(o.now.guests, 'guest', 'guests')}, ${of(o.now.members, 'member', 'members')}, ${n(o.now.staff)} staff`),
    tile('Private chats', n(o.now.open_chats), `${of(o.now.knocks_waiting, 'knock', 'knocks')} waiting`),
    tile('New people, 24h', n(o.people.guests_24h + o.people.members_24h), `${of(o.people.guests_24h, 'guest', 'guests')}, ${of(o.people.members_24h, 'member', 'members')}`),
    tile('Messages, 24h', n(o.messages.room_24h + o.messages.private_24h), `${n(o.messages.room_24h)} in the room, ${n(o.messages.private_24h)} private`),
    tile('Open reports', n(o.safety.open_reports), 'waiting for a moderator', o.safety.open_reports > 0),
    tile('Members', n(o.people.members_total), `${n(o.people.members_7d)} joined this week`),
    tile('Sanctions', n(o.safety.banned_users + o.safety.muted_users), `${n(o.safety.banned_users)} banned, ${n(o.safety.muted_users)} muted, ${n(o.safety.ip_bans)} device bans`));

  // Messages per hour: stacked room and private bars.
  const max = Math.max(1, ...o.messages.per_hour.map((b) => b.room + b.private));
  const bars = h('div', { class: 'bars', role: 'img', 'aria-label': 'Messages per hour, last 24 hours' },
    o.messages.per_hour.map((b, i) => {
      const at = new Date(b.hour * 1000);
      const label = at.toLocaleTimeString([], { hour: 'numeric' });
      return h('div', { class: 'bar', title: `${label}: ${b.room} room, ${b.private} private` },
        h('div', { class: 'stack' },
          h('span', { class: 'private', style: { height: `${(b.private / max) * 100}%` } }),
          h('span', { class: 'room', style: { height: `${(b.room / max) * 100}%` } })),
        h('span', { class: 'bar-label' }, i % 3 === 0 || i === 23 ? label : ''));
    }));
  const chart = h('section', { class: 'panel-card wide' },
    h('div', { class: 'card-head' }, h('h2', {}, 'Messages per hour'),
      h('span', { class: 'legend' }, h('i', { class: 'room' }), 'Room', h('i', { class: 'private' }), 'Private')),
    bars, h('p', { class: 'fine' }, `Busiest hour: ${max} messages. Last 24 hours, your local time.`));

  const ranked = (title, entries, render, empty) => h('section', { class: 'panel-card' }, h('h2', {}, title),
    entries.length ? h('ol', { class: 'ranked' }, entries.map(([k, v]) => {
      const top = entries[0][1] || 1;
      return h('li', {}, h('span', { class: 'rk-name' }, render(k)), h('span', { class: 'rk-bar' }, h('span', { style: { width: `${(v / top) * 100}%` } })), h('span', { class: 'rk-n' }, n(v)));
    })) : h('p', { class: 'fine' }, empty));
  const byCount = (obj) => Object.entries(obj || {}).sort((a, b) => b[1] - a[1]);
  const countryName = (cc) => (cc && cc !== 'unknown' ? [flag(cc), ' ', cc] : 'Unknown');

  return [tiles, h('div', { class: 'panels' },
    chart,
    ranked('Online by country', byCount(o.now.countries).slice(0, 8), countryName, 'Nobody online.'),
    ranked('Top tags online', byCount(o.now.tags).slice(0, 8), (k) => k, 'Nobody online has a tag.'),
    ranked('New people by country, 7 days', o.signup_countries_7d.map((c) => [c.country, c.signups]), countryName, 'No sign-ups this week.'),
    ranked('Report reasons, 7 days', byCount(o.safety.report_reasons_7d), (k) => REASONS[k] || k, 'No reports this week.'))];
}

// ---------- People online ----------

function deskPeople(main) {
  const people = [...state.online.values()].sort((a, b) => (a.online_since || 0) - (b.online_since || 0));
  main.append(h('div', { class: 'desk-head' }, h('h1', {}, 'People online'), h('span', { class: 'muted' }, `${people.length} besides you`)));
  if (!people.length) { main.append(h('div', { class: 'empty' }, 'Nobody else is online.')); return; }
  const act = async (path, body) => { if (await modAct(path, body)) renderDesk(); };
  main.append(h('div', { class: 'table' }, people.map((p) => h('div', {},
    avatar(p, 36),
    h('span', { class: 'grow' }, h('b', { class: 'who-name', 'data-gender': p.gender }, p.name), ` ${p.age} · ${genderLabel(p.gender)} · ${p.kind}${p.role !== 'user' ? ` · ${p.role}` : ''} `, flag(p.country),
      h('br'), h('small', { class: 'muted' }, `${[hereFor(p.online_since), ...(p.tags || []), p.location].filter(Boolean).join(' · ')}`)),
    p.role === 'user' && h('span', { class: 'row-actions' },
      h('button', { class: 'btn sm', type: 'button', onclick: () => showMessages(p) }, 'Messages'),
      h('button', { class: 'btn sm', type: 'button', onclick: () => act(`/admin/users/${p.id}/mute`, { minutes: 30 }) }, t('mute30')),
      h('button', { class: 'btn sm', type: 'button', onclick: () => act(`/admin/users/${p.id}/kick`) }, t('kick')),
      h('button', { class: 'btn sm danger', type: 'button', onclick: () => act(`/admin/users/${p.id}/ban`, { minutes: 1440, ip: false, reason: 'from control panel' }) }, t('ban24')))))));
}

async function showMessages(p) {
  let msgs;
  try { msgs = await api('GET', `/admin/users/${p.id}/messages`); } catch (ex) { toast(ex.message, true); return; }
  const d = dialog(h('div', { class: 'dlg' },
    h('h3', {}, `${p.name}: recent messages`),
    msgs?.length
      ? h('div', { class: 'evidence', style: { maxHeight: '50vh', overflowY: 'auto' } }, msgs.map((m) =>
        h('span', {}, m.recipient_id ? '(private) ' : '', h('span', { style: { color: 'var(--text-2)' } }, m.body || '[photo]'))))
      : h('p', { class: 'fine' }, 'No messages in the last 7 days.'),
    h('div', { class: 'actions' }, h('button', { class: 'btn', type: 'button', onclick: () => d.close() }, t('close')))));
}

// ---------- Staff (admin only) ----------

async function deskStaff(main) {
  main.append(h('div', { class: 'desk-head' }, h('h1', {}, 'Staff'), h('span', { class: 'muted' }, 'Only the admin can add or change staff.')));
  let staff;
  try { staff = await api('GET', '/admin/staff'); } catch (ex) { main.append(h('p', { class: 'error-text' }, ex.message)); return; }
  const setRole = async (id, role) => {
    try { await api('PUT', `/admin/users/${id}/role`, { role }); toast('Role updated.'); renderDesk(); } catch (ex) { toast(ex.message, true); }
  };
  main.append(h('div', { class: 'table' }, staff.map((m) => h('div', {},
    avatar(m, 36),
    h('span', { class: 'grow' }, h('b', { class: 'who-name', 'data-gender': m.gender }, m.name), h('br'), h('small', { class: 'muted' }, m.email)),
    h('span', { class: `tag ${m.role === 'admin' ? 'block' : ''}` }, m.role),
    m.id !== state.me.id && h('span', { class: 'row-actions' },
      h('button', { class: 'btn sm', type: 'button', onclick: () => setRole(m.id, m.role === 'admin' ? 'moderator' : 'admin') }, m.role === 'admin' ? 'Make moderator' : 'Make admin'),
      h('button', { class: 'btn sm danger', type: 'button', onclick: () => setRole(m.id, 'user') }, 'Remove from staff'))))));

  // New staff account.
  let role = 'moderator';
  let gender = 'other';
  const err = h('div', { class: 'error-text', role: 'alert' });
  const f = (name, attrs) => h('input', { class: 'input', name, ...attrs });
  const form = h('form', {
    class: 'panel-card staff-form',
    onsubmit: async (e) => {
      e.preventDefault();
      err.textContent = '';
      const v = (k) => form.querySelector(`[name=${k}]`).value.trim();
      try {
        await api('POST', '/admin/staff', { name: v('name'), age: Number(v('age')), gender, email: v('email'), password: form.querySelector('[name=password]').value, role });
        toast('Staff account created. Share the email and password with them privately.');
        renderDesk();
      } catch (ex) { err.textContent = ex.message; }
    },
  },
    h('h2', {}, 'Add a staff member'),
    h('p', { class: 'fine' }, 'Staff sign in with this email and password. Ask them to keep it private; there is no self sign-up for staff.'),
    h('div', { class: 'row' },
      h('label', { class: 'field' }, h('span', {}, t('name')), f('name', { maxlength: 32, required: true, autocomplete: 'off' })),
      h('label', { class: 'field' }, h('span', {}, t('age')), f('age', { type: 'number', min: 18, max: 99, required: true, value: '18' }))),
    h('div', { class: 'field' }, h('span', {}, t('gender')), chipGroup(t('gender'), GENDERS, gender, (v) => { gender = v; })),
    h('label', { class: 'field' }, h('span', {}, t('email')), f('email', { type: 'email', required: true, autocomplete: 'off' })),
    h('label', { class: 'field' }, h('span', {}, 'Temporary password'), f('password', { type: 'password', minlength: 8, required: true, autocomplete: 'new-password' })),
    h('div', { class: 'field' }, h('span', {}, 'Role'), chipGroup('Role', { moderator: 'Moderator', admin: 'Admin' }, role, (v) => { role = v; })),
    err,
    h('div', { class: 'actions', style: { justifyContent: 'flex-start' } }, h('button', { class: 'btn primary', type: 'submit' }, 'Create staff account')));
  main.append(form);
}

async function deskReports(main) {
  main.append(h('h1', {}, t('reports')));
  let reports;
  try { reports = await api('GET', '/admin/reports?status=open'); } catch (ex) { main.append(h('p', { class: 'error-text' }, ex.message)); return; }
  if (!reports?.length) { main.append(h('div', { class: 'empty' }, t('queueEmpty'))); return; }

  // One card per reported thing, oldest first; acting on it closes them all.
  const groups = new Map();
  for (const r of reports) {
    const key = `${r.target_type}:${r.target_id}`;
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(r);
  }
  for (const group of groups.values()) main.append(reportCard(group));
}

function reportCard(group) {
  const r = group[0];
  const count = group.length;
  const notes = group.map((x) => x.note).filter(Boolean);
  const reasons = [...new Set(group.map((x) => x.reason))];
  let ev = r.evidence;
  if (typeof ev === 'string') { try { ev = JSON.parse(ev); } catch { ev = {}; } }
  ev = ev || {};
  const after = (ok) => { if (ok) renderDesk(); };
  const uid = r.target_user_id;

  const evidence = h('div', { class: 'evidence' }, h('span', { style: { fontSize: '12px', letterSpacing: '0.1em', textTransform: 'uppercase' } }, 'Evidence, captured when reported'));
  if (ev.context?.length) {
    for (const m of [...ev.context].reverse()) evidence.append(h('span', {}, `${m.sender_name}: `, h('span', { style: { color: 'var(--text-2)' } }, m.body)));
  }
  if (ev.message) evidence.append(h('span', {}, `${ev.message.sender_name}${ev.message.recipient_id ? ' (private)' : ''}:`), h('span', { class: 'quoted' }, ev.message.body || '[photo]'));
  if (ev.image) evidence.append(h('img', { src: ev.image.url, alt: 'Reported image' }));
  if (ev.user) evidence.append(h('span', { class: 'quoted' }, `${ev.user.name}, ${ev.user.age}`, ev.user.location ? ` · ${ev.user.location}` : '', ev.user.about ? ` · "${ev.user.about}"` : ''));
  for (const note of notes) evidence.append(h('span', {}, 'Reporter note: ', h('span', { style: { color: 'var(--text-2)' } }, note)));

  const actions = h('div', { class: 'actions', style: { justifyContent: 'flex-start' } });
  if (r.target_type === 'message') actions.append(h('button', { class: 'btn sm primary', type: 'button', onclick: async () => after(await modAct(`/admin/messages/${r.target_id}/remove`)) }, t('remove')));
  if (r.target_type === 'media') actions.append(h('button', { class: 'btn sm primary', type: 'button', onclick: async () => after(await modAct(`/admin/media/${r.target_id}/remove`)) }, t('remove')));
  if (uid) {
    actions.append(
      h('button', { class: 'btn sm', type: 'button', onclick: async () => after(await modAct(`/admin/users/${uid}/mute`, { minutes: 30 })) }, t('mute30')),
      h('button', { class: 'btn sm', type: 'button', onclick: async () => after(await modAct(`/admin/users/${uid}/kick`)) }, t('kick')),
      h('button', { class: 'btn sm danger', type: 'button', onclick: async () => after(await modAct(`/admin/users/${uid}/ban`, { minutes: 1440, ip: false, reason: r.reason })) }, t('ban24')),
      h('button', { class: 'btn sm danger', type: 'button', onclick: async () => after(await modAct(`/admin/users/${uid}/ban`, { minutes: 1440, ip: true, reason: r.reason })) }, t('banIP')));
  }
  const dismissAll = async () => {
    let ok = true;
    for (const x of group) ok = (await modAct(`/admin/reports/${x.id}/dismiss`)) && ok;
    after(ok);
  };
  actions.append(h('button', { class: 'btn sm ghost', type: 'button', onclick: dismissAll }, t('dismiss')));

  return h('article', { class: `report${count > 1 ? ' hot' : ''}` },
    h('div', { class: 'top' },
      h('span', { style: { display: 'flex', gap: '10px', alignItems: 'center' } },
        reasons.map((x) => h('span', { class: 'reason' }, REASONS[x] || x)),
        h('span', { class: 'muted', style: { fontSize: '14px' } }, `${r.target_type} #${r.target_id} · ${count} ${count === 1 ? 'report' : 'reports'}`)),
      h('span', { class: 'muted', style: { fontSize: '13px' } }, new Date(r.created_at).toLocaleString())),
    evidence,
    actions);
}

async function deskWords(main) {
  const word = h('input', { class: 'input', placeholder: t('wordPh'), maxlength: 64, 'aria-label': t('wordPh') });
  const action = h('select', { 'aria-label': 'Action' }, h('option', { value: 'mask' }, t('mask')), h('option', { value: 'block' }, t('blockWord')));
  const err = h('div', { class: 'error-text', role: 'alert' });
  main.append(
    h('h1', {}, t('words')),
    h('p', { class: 'fine', style: { maxWidth: '560px' } }, t('wordHelp')),
    h('form', {
      class: 'word-form',
      onsubmit: async (e) => {
        e.preventDefault();
        try {
          await api('POST', '/admin/words', { word: word.value.trim(), action: action.value });
          renderDesk();
        } catch (ex) { err.textContent = ex.message; }
      },
    }, word, action, h('button', { class: 'btn primary', type: 'submit' }, t('add'))),
    err);
  let words;
  try { words = await api('GET', '/admin/words'); } catch (ex) { main.append(h('p', { class: 'error-text' }, ex.message)); return; }
  if (!words?.length) return;
  main.append(h('div', { class: 'table' }, words.map((w) => h('div', {},
    h('span', { class: 'grow' }, w.word),
    h('span', { class: `tag ${w.action}` }, w.action === 'block' ? t('blockWord') : t('mask')),
    h('button', {
      class: 'icon-btn bare', type: 'button', 'aria-label': `${t('remove')} ${w.word}`,
      onclick: async () => { try { await api('DELETE', `/admin/words/${w.id}`); renderDesk(); } catch (ex) { toast(ex.message, true); } },
    }, icon('x', 16))))));
}

async function deskAudit(main) {
  main.append(h('h1', {}, t('audit')));
  let rows;
  try { rows = await api('GET', '/admin/audit'); } catch (ex) { main.append(h('p', { class: 'error-text' }, ex.message)); return; }
  if (!rows?.length) { main.append(h('div', { class: 'empty' }, '-')); return; }
  main.append(h('div', { class: 'table' }, rows.map((a) => h('div', {},
    h('span', { class: 'tag' }, a.action.replace(/_/g, ' ')),
    h('span', { class: 'grow' }, [a.target_name, a.detail].filter(Boolean).join(' · ') || (a.target_id ? `#${a.target_id}` : '')),
    h('span', { class: 'muted', style: { fontSize: '13px', whiteSpace: 'nowrap' } }, `#${a.actor_id} · ${new Date(a.created_at).toLocaleString()}`)))));
}

// ---------------------------------------------------------------------------
// Boot
// ---------------------------------------------------------------------------

window.addEventListener('hashchange', () => { if (state.token && state.me) renderShell(); });
document.addEventListener('visibilitychange', () => {
  if (document.hidden) {
    saveArchive(); // phones may never fire pagehide for a background tab
    return;
  }
  if (peopleDirty && peopleShowing()) renderSide();
  const c = state.convos.get(state.active);
  if (c && isViewing(state.active)) {
    c.unread = 0;
    rerenderChats();
  }
});

// boot resumes the saved login. Only a refusal (expired, revoked, banned)
// drops it, and its reason is shown; a network error keeps it for the next
// try, since the login is still good.
async function boot() {
  if (!state.token) { renderEnter(); return; }
  try {
    state.me = await api('GET', '/me');
    startRoom();
  } catch (ex) {
    if (ex.status === 401 || ex.status === 403) {
      writeStore(TOKEN_KEY, null);
      state.token = null;
    }
    renderEnter(ex.message);
  }
}

boot();
