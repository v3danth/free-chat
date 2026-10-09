// Drift: browser client. Plain ES modules, no build step.
// The contract with the server is SPEC.md. Text from the network is only
// ever rendered with textContent, never as HTML.

// ---------------------------------------------------------------------------
// Words
// ---------------------------------------------------------------------------

const WORDS = {
    tagline: 'Talk to strangers. No sign-up, no app.',
    heroA: 'Walk in.', heroB: 'Say hi.', heroC: 'Leave no trace.',
    heroP: 'Make a card, step into the room and talk to whoever is up right now. Everyone wears their face; a member\'s real photo shows only after they answer your knock.',
    point1: 'No email, no phone number, no download',
    point2: 'Real photos only after they reply',
    point3: 'Your card and chats disappear when you leave',
    hereNow: 'here now',
    makeCard: 'Make your card',
    name: 'Name', namePh: 'What should people call you?',
    age: 'Age', gender: 'I am', tags: 'Here to',
    location: 'Location', locationPh: 'City, optional',
    about: 'About', aboutPh: 'One line about you (optional)',
    photo: 'Add a photo', photoHint: 'Optional. Shown only to people whose knock you answer, or who answer yours.',
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
    knockInHelp: 'Reply to open the door. Both of you will see each other clearly.',
    replyPh: 'Reply to open the door...',
    msgPh: 'Message...',
    doorOpen: 'The door is open. You can see each other now.',
    gone: '{name} left. This chat ends when either of you leaves.',
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
    ended: 'You left the room.',
    seoTitle: 'Free chat with strangers',
    seoP: 'Drift is a free chat room for meeting strangers: random chat without registration, on your phone or laptop. Looking for an Omegle alternative or a Y99 alternative? Walk in, make a card and say hi.',
};

const GENDERS = { female: 'Woman', male: 'Man', 'non-binary': 'Non-binary', femboy: 'Femboy', couple: 'Couple', other: 'Other' };
const REASONS = { spam: 'Spam', harassment: 'Harassment', nudity: 'Nudity', violence: 'Violence', hate: 'Hate', underage: 'Under 18', scam: 'Scam', other: 'Other' };

const t = (key, vars = {}) => (WORDS[key] ?? key).replace(/\{(\w+)\}/g, (_, k) => vars[k] ?? '');
const genderLabel = (g) => GENDERS[g] ?? g;
// Suggestions only: people can type any tag (up to 3, 20 characters each).
// Adults-only community: common kink-scene terms between consenting adults.
const TAG_IDEAS = ['dom', 'sub', 'switch', 'bdsm', 'roleplay', 'bondage', 'brat', 'praise', 'rope', 'leather',
  'sensory play', 'exhibitionist', 'voyeur', 'kink curious', 'aftercare', 'just talk'];
const MAX_TAGS = 3;

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

function readStore(key, store = localStorage) {
  try { return store.getItem(key); } catch { return null; }
}
function writeStore(key, value, store = localStorage) {
  try { value == null ? store.removeItem(key) : store.setItem(key, value); } catch { /* private mode */ }
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
  return h('span', {
    class: 'avatar',
    'aria-hidden': 'true',
    style: { width: `${size}px`, height: `${size}px`, ...pictureStyle(person) },
  });
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
function clock(unix) {
  return new Date(unix * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function toast(text, bad = false) {
  const el = h('div', { class: `toast${bad ? ' bad' : ''}`, role: bad ? 'alert' : 'status' }, text);
  $('#toasts').append(el);
  setTimeout(() => el.remove(), 4200);
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
  token: readStore('token', sessionStorage),
  me: null,
  online: new Map(), // id -> Card
  feed: [],          // room chat events, oldest first
  convos: new Map(), // peer id -> { peer, msgs, phase, unread, seenDoor }
  active: null,      // peer id of the open chat
  view: 'room',      // mobile: room | people | chats
  side: 'people',    // desktop right pane: people | chats
  ws: null,
  retry: 0,
  ended: false,
};

// phase of a private chat: knocked (I knocked), incoming (they knocked),
// open (door opened), gone (they left).
function convo(peerId) {
  let c = state.convos.get(peerId);
  if (!c) {
    c = { peer: state.online.get(peerId) || { id: peerId, name: '?' }, msgs: [], phase: null, unread: 0 };
    state.convos.set(peerId, c);
  }
  return c;
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

// enterDraft keeps what was typed while switching between the three ways in.
const enterDraft = {
  mode: 'guest', // guest | join | signin
  name: new URLSearchParams(window.location.search).get('name') || '',
  age: '', gender: '', tags: [], location: '', about: '', email: '', photo: null,
};

function renderEnter(message) {
  state.screen = 'enter';
  const d = enterDraft;
  const err = h('div', { class: 'error-text', role: 'alert' }, message || '');
  const bind = (key, attrs) => h('input', { class: 'input', value: d[key], ...attrs, oninput: (e) => { d[key] = e.target.value; attrs.oninput?.(e); } });

  // The face (or, when joining with one, the photo) previews live.
  const preview = h('span', { class: 'avatar', style: { width: '56px', height: '56px' } });
  const showPreview = () => Object.assign(preview.style, d.photo
    ? { backgroundImage: `url("${URL.createObjectURL(d.photo)}")`, imageRendering: '' }
    : pictureStyle({ name: d.name }));
  showPreview();

  const submit = h('button', { class: 'btn primary', type: 'submit', style: { height: '52px', fontSize: '16px' } },
    { guest: t('enter'), join: 'Create account', signin: t('signIn') }[d.mode]);

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
        type: 'file', accept: 'image/jpeg,image/png,image/gif,image/webp', class: 'sr', id: 'photo-in',
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
    h('h2', {}, { guest: t('makeCard'), join: 'Make your account', signin: 'Welcome back' }[d.mode]),
    d.mode === 'guest' && h('p', { class: 'fine' }, 'One visit, no email. Your card disappears when you leave.'),
    d.mode === 'join' && h('p', { class: 'fine' }, 'Keep your name, add your own photo. People see your face until they answer your knock.'),
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
        h('span', { class: 'brand' }, 'Drift', h('b', {}, '.')),
        h('div', { class: 'right' }, h('a', { class: 'btn sm', href: '/faces' }, 'Find your face'))),
      h('main', { class: 'enter-main' },
        h('section', { class: 'hero' },
          h('h1', {}, t('heroA'), ' ', t('heroB'), h('br'), h('em', {}, t('heroC'))),
          h('p', {}, t('heroP')),
          h('ul', {}, point(t('point1')), point(t('point2')), point(t('point3')))),
        form),
      h('footer', { class: 'seo-foot' },
        h('h2', {}, t('seoTitle')),
        h('p', {}, t('seoP')),
        h('p', { class: 'fine' }, h('a', { href: 'https://db-ip.com', rel: 'noopener' }, 'IP Geolocation by DB-IP')))),
  );
  form.querySelector('input')?.focus();
}

function setSession(token, me) {
  state.token = token;
  state.me = me;
  writeStore('token', token, sessionStorage);
}

function endSession(message) {
  clearInterval(deskTimer);
  state.ended = true;
  state.ws?.close();
  state.ws = null;
  state.token = null;
  state.me = null;
  state.online.clear();
  state.convos.clear();
  state.feed = [];
  state.active = null;
  writeStore('token', null, sessionStorage);
  location.hash = '';
  renderEnter(message);
}

// ---------------------------------------------------------------------------
// Screen: Room
// ---------------------------------------------------------------------------

function startRoom() {
  state.ended = false;
  renderShell();
  connect();
}

function topbar(extra) {
  return h('header', { class: 'topbar' },
    h('span', { class: 'brand' }, 'Drift', h('b', {}, '.')),
    h('div', { class: 'right' },
      h('span', { class: 'live-dot hide-sm', id: 'here-count' }, `${state.online.size + 1} ${t('hereNow')}`),
      extra,
      state.me && h('button', { class: 'me-pill', type: 'button', onclick: openEditCard, title: t('editCard') },
        avatar(state.me, 28),
        h('span', { class: 'label' }, state.me.name, state.me.tags?.length ? ` · ${state.me.tags[0]}` : '')),
      h('button', { class: 'btn sm', type: 'button', onclick: () => endSession(t('ended')) }, t('leave'))));
}

function renderShell() {
  clearInterval(deskTimer);
  if (location.hash === '#desk' && isStaff()) return renderDesk();
  state.screen = 'room';
  const deskLink = isStaff() && h('button', { class: 'btn sm outline', type: 'button', onclick: () => { location.hash = '#desk'; } }, icon('shield', 16), h('span', { class: 'desk-label' }, t('desk')));

  const feed = h('div', { class: 'feed', id: 'feed', role: 'log', 'aria-live': 'polite' });
  const input = h('input', { class: 'input', id: 'room-input', maxlength: 1000, placeholder: t('sayRoom'), autocomplete: 'off', 'aria-label': t('sayRoom') });
  const photoIn = h('input', {
    type: 'file', accept: 'image/*', class: 'sr', id: 'room-photo',
    onchange: async (e) => {
      const file = e.target.files[0];
      e.target.value = '';
      if (!file) return;
      try {
        const up = await uploadPhoto(file);
        send({ type: 'chat', room_id: MAIN_ROOM, media_id: up.id });
      } catch (ex) { toast(ex.message, true); }
    },
  });
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
    h('label', { class: 'icon-btn', for: 'room-photo', title: t('sendPhoto'), style: { cursor: 'pointer' } }, icon('photo'), h('span', { class: 'sr' }, t('sendPhoto'))),
    photoIn,
    input,
    h('button', { class: 'icon-btn', type: 'submit', 'aria-label': t('send'), style: { background: 'var(--amber)', color: 'var(--amber-ink)', borderColor: 'var(--amber)' } }, icon('send')));

  const main = h('section', { class: 'pane main' },
    h('div', { class: 'pane-head' }, h('h2', {}, t('room')), h('small', {}, t('lastLines'))),
    feed, composer);

  const side = h('section', { class: 'pane side', id: 'side' });
  const stage = h('div', { class: 'stage', id: 'stage', 'data-view': state.view }, main, side);

  const mobileTabs = h('nav', { class: 'mobile-tabs', id: 'mobile-tabs' });

  $('#app').replaceChildren(h('div', { class: 'shell' }, topbar(deskLink), stage, mobileTabs));
  for (const ev of state.feed) feed.append(feedLine(ev));
  feed.scrollTop = feed.scrollHeight;
  renderSide();
  renderMobileTabs();
}

function unreadTotal() {
  let n = 0;
  for (const c of state.convos.values()) n += c.unread;
  return n;
}

function renderMobileTabs() {
  const nav = $('#mobile-tabs');
  if (!nav) return;
  const tab = (view, label, badge) => h('button', {
    class: 'tab', type: 'button', role: 'tab', 'aria-selected': String(state.view === view),
    onclick: () => {
      state.view = view;
      if (view !== 'room') state.side = view;
      if (view === 'chats' && state.active != null) convo(state.active).unread = 0;
      $('#stage').dataset.view = view;
      renderSide();
      renderMobileTabs();
    },
  }, label, badge ? h('span', { class: 'badge' }, badge) : null);
  nav.replaceChildren(tab('room', t('room')), tab('people', t('people'), null), tab('chats', t('chats'), unreadTotal() || null));
}

function refreshCounts() {
  const here = $('#here-count');
  if (here) here.textContent = `${state.online.size + 1} ${t('hereNow')}`;
  renderMobileTabs();
}

// ---------- Room feed ----------

function feedLine(ev) {
  const mine = ev.sender_id === state.me?.id;
  const person = mine ? state.me : state.online.get(ev.sender_id) || { id: ev.sender_id, name: ev.name };
  const acts = h('div', { class: 'acts' });
  if (!mine) acts.append(h('button', { class: 'icon-btn bare', type: 'button', title: t('report'), 'aria-label': t('report'), onclick: () => openReport('message', ev.id) }, icon('flag', 16)));
  if (isStaff()) acts.append(h('button', { class: 'icon-btn bare', type: 'button', title: t('remove'), 'aria-label': t('remove'), onclick: () => modAct(`/admin/messages/${ev.id}/remove`) }, icon('trash', 16)));

  return h('div', { class: `line${mine ? ' mine' : ''}`, 'data-id': ev.id },
    avatar(person, 32),
    h('div', { class: 'body' },
      h('span', { class: 'meta' },
        h('button', { type: 'button', class: 'who-name', 'data-gender': ev.gender || person.gender, onclick: () => !mine && openPerson(ev.sender_id, ev.name) }, ev.name),
        mine ? h('span', { class: 'you-tag' }, ` (${t('you')})`) : null,
        person.age ? ` · ${genderLabel(person.gender)}, ${person.age}` : '',
        ` · ${clock(ev.ts)}`,
        ev.filtered ? h('span', { class: 'filtered-tag' }, ` · ${t('filtered')}`) : null),
      ev.content ? h('span', { class: 'text' }, ev.content) : null,
      ev.image ? shot(ev.image) : null),
    acts.children.length ? acts : null);
}

function shot(image) {
  return h('button', { type: 'button', class: 'shot-btn', 'aria-label': 'Open photo', onclick: () => openLightbox(image.url) },
    h('img', { class: 'shot', src: image.thumb_url, alt: '', loading: 'lazy' }));
}

function addFeed(ev) {
  state.feed.push(ev);
  if (state.feed.length > FEED_KEEP) state.feed.shift();
  const feed = $('#feed');
  if (!feed) return;
  const atBottom = feed.scrollHeight - feed.scrollTop - feed.clientHeight < 80;
  feed.append(feedLine(ev));
  while (feed.children.length > FEED_KEEP) feed.firstElementChild.remove();
  if (atBottom || ev.sender_id === state.me?.id) feed.scrollTop = feed.scrollHeight;
}

function sysLine(text) {
  const feed = $('#feed');
  if (feed) feed.append(h('div', { class: 'sys' }, text));
}

// ---------- Right pane ----------

function renderSide() {
  const side = $('#side');
  if (!side) return;
  const tab = (key, label, badge) => h('button', {
    class: 'tab', type: 'button', role: 'tab', 'aria-selected': String(state.side === key),
    onclick: () => { state.side = key; renderSide(); },
  }, label, badge ? h('span', { class: 'badge' }, badge) : null);
  const tabs = h('div', { class: 'tabs', role: 'tablist' }, tab('people', t('people')), tab('chats', t('chats'), unreadTotal() || null));
  // Presence and messages re-render this pane often: keep where the reader
  // was scrolled to and which control had focus.
  const scrolls = ['.people', '.convo-list'].map((sel) => [sel, side.querySelector(sel)?.scrollTop]);
  const focusKey = side.contains(document.activeElement) ? document.activeElement.dataset.key : null;
  side.replaceChildren(tabs, state.side === 'chats' ? chatsPane() : peoplePane());
  for (const [sel, top] of scrolls) {
    const el = side.querySelector(sel);
    if (el && top != null) el.scrollTop = top;
  }
  if (focusKey) side.querySelector(`[data-key="${CSS.escape(focusKey)}"]`)?.focus({ preventScroll: true });
}

function peoplePane() {
  const wrap = h('div', { class: 'people' });
  const people = [...state.online.values()].sort((a, b) => (a.online_since || 0) - (b.online_since || 0));
  if (!people.length) {
    wrap.append(h('div', { class: 'empty' }, t('nobody')));
    return wrap;
  }
  wrap.append(h('div', { class: 'strip' }, people.map(personCard)));
  return wrap;
}

// tagRow always renders, empty or not, so cards in a row line up.
const tagRow = (tags) => h('span', { class: 'tag-row' }, (tags || []).map((tag) => h('span', { class: 'tag-chip small' }, tag)));

function personCard(card) {
  const c = state.convos.get(card.id);
  return h('div', { class: 'pcard', 'data-tier': card.avatar_tier },
    h('button', {
      type: 'button', class: 'photo', 'aria-label': card.name, 'data-key': `card-${card.id}`, onclick: () => openPerson(card.id),
      style: { ...pictureStyle({ ...card, photo_url: c?.peer.photo_url }), border: 'none', width: '100%', cursor: 'pointer' },
    },
      h('span', { class: 'pill' }, hereFor(card.online_since)),
      card.has_photo && !c?.peer.photo_url ? h('span', { class: 'pill', title: 'Has a photo you will see after they reply' }, icon('photo', 14)) : null),
    h('div', { class: 'info' },
      h('span', { class: 'name' }, h('span', {}, h('span', { class: 'who-name', 'data-gender': card.gender }, card.name), `, ${card.age}`), flag(card.country)),
      h('span', { class: 'sub' }, card.about || [genderLabel(card.gender), card.location].filter(Boolean).join(' · ')),
      tagRow(card.tags),
      h('button', { class: 'btn outline sm', type: 'button', 'data-key': `knock-${card.id}`, onclick: () => openChat(card.id) }, c ? t('open') : t('knock'))));
}

// ---------- Private chats ----------

// isViewing: peerId's chat is on screen right now (on phones, the Chats tab
// must be the one showing).
function isViewing(peerId) {
  return state.active === peerId && state.side === 'chats' && document.visibilityState === 'visible'
    && (state.view === 'chats' || !matchMedia('(max-width: 960px)').matches);
}

function openChat(peerId) {
  convo(peerId).unread = 0;
  state.active = peerId;
  state.side = 'chats';
  state.view = 'chats';
  const stage = $('#stage');
  if (stage) stage.dataset.view = 'chats';
  renderSide();
  renderMobileTabs();
  setTimeout(() => $('#dm-input')?.focus());
}

function chatsPane() {
  const items = [...state.convos.entries()].sort((a, b) => lastTs(b[1]) - lastTs(a[1]));
  if (!items.length) return h('div', { class: 'people' }, h('div', { class: 'empty' }, t('noChats')));

  const list = h('div', { class: 'convo-list' }, items.map(([id, c]) => {
    const last = c.msgs[c.msgs.length - 1];
    const preview = c.phase === 'gone' ? t('gone', { name: c.peer.name })
      : !last ? '' : last.door ? t('doorOpen') : last.content || 'Photo';
    return h('button', {
      class: 'convo-item', type: 'button', 'aria-current': String(state.active === id),
      onclick: () => openChat(id),
    }, avatar(c.peer, 40), h('span', { class: 't' }, h('b', { class: 'who-name', 'data-gender': c.peer.gender }, c.peer.name), h('small', {}, preview)), c.unread ? h('span', { class: 'dot' }) : null);
  }));

  const open = state.active != null && state.convos.has(state.active);
  return h('div', { class: 'chats', 'data-open': String(open) }, list, open ? convoView(state.active) : h('div', { class: 'convo' }, h('div', { class: 'empty', style: { margin: 'auto' } }, t('pickChat'))));
}

const lastTs = (c) => c.msgs[c.msgs.length - 1]?.ts || 0;

function convoView(peerId) {
  const c = convo(peerId);
  const peer = c.peer;
  const meta = [genderLabel(peer.gender), peer.age, peer.location].filter(Boolean).join(' · ');

  const head = h('div', { class: 'convo-head' },
    h('button', { class: 'icon-btn bare', type: 'button', 'aria-label': t('back'), onclick: () => { state.active = null; renderSide(); } }, icon('back')),
    avatar(peer, 44),
    h('div', { class: 'who' }, h('b', {}, h('span', { class: 'who-name', 'data-gender': peer.gender }, peer.name), ' ', flag(peer.country)), h('small', {}, meta)),
    h('button', { class: 'btn sm ghost', type: 'button', onclick: () => openReport('user', peerId) }, t('report')),
    h('button', { class: 'btn sm', type: 'button', onclick: () => blockPerson(peerId) }, t('block')));

  const bubbles = h('div', { class: 'bubbles', id: 'bubbles' });
  const myId = state.me.id;

  if (c.phase === 'incoming' || c.phase === 'knocked') {
    const knock = c.msgs[0];
    bubbles.append(h('div', { class: 'knock-panel' },
      avatar(peer, 96),
      h('h3', {}, c.phase === 'incoming' ? t('knockIn', { name: peer.name }) : t('knockSent')),
      knock ? h('div', { class: 'knock-quote' }, knock.content) : null,
      h('p', {}, c.phase === 'incoming' ? t('knockInHelp') : t('knockWait', { name: peer.name }))));
  } else if (!c.phase) {
    bubbles.append(h('div', { class: 'knock-panel' },
      avatar(peer, 96),
      h('h3', {}, t('knockTitle', { name: peer.name })),
      peer.about ? h('div', { class: 'knock-quote' }, peer.about) : null,
      h('p', {}, t('knockHelp'))));
  } else {
    for (const m of c.msgs) {
      if (m.door) {
        bubbles.append(h('div', { class: 'door-banner' }, avatar(peer, 56), h('span', {}, t('doorOpen'))));
        continue;
      }
      const mine = m.from === myId;
      const acts = mine ? null : h('div', { class: 'acts' }, h('button', { class: 'icon-btn bare', type: 'button', 'aria-label': t('report'), onclick: () => openReport('message', m.id) }, icon('flag', 14)));
      bubbles.append(h('div', { class: `bubble${mine ? ' mine' : ''}`, 'data-id': m.id },
        m.content || null, m.image ? shot(m.image) : null, acts));
    }
    if (c.phase === 'gone') bubbles.append(h('div', { class: 'sys' }, t('gone', { name: peer.name })));
  }

  const canSend = c.phase !== 'knocked' && c.phase !== 'gone';
  const input = h('input', {
    class: 'input', id: 'dm-input', maxlength: 1000, autocomplete: 'off', disabled: !canSend,
    placeholder: !c.phase ? t('knockPh') : c.phase === 'incoming' ? t('replyPh') : t('msgPh'), 'aria-label': t('msgPh'),
  });
  const photoAllowed = c.phase === 'open';
  const photoIn = h('input', {
    type: 'file', accept: 'image/*', class: 'sr', id: 'dm-photo',
    onchange: async (e) => {
      const file = e.target.files[0];
      e.target.value = '';
      if (!file) return;
      try {
        const up = await uploadPhoto(file);
        send({ type: 'dm', to: peerId, media_id: up.id });
      } catch (ex) { toast(ex.message, true); }
    },
  });
  const composer = h('form', {
    class: 'composer',
    onsubmit: (e) => {
      e.preventDefault();
      const content = input.value.trim();
      if (!content) return;
      send({ type: 'dm', to: peerId, content });
      input.value = '';
    },
  },
    photoAllowed ? h('label', { class: 'icon-btn', for: 'dm-photo', title: t('sendPhoto'), style: { cursor: 'pointer' } }, icon('photo'), h('span', { class: 'sr' }, t('sendPhoto'))) : null,
    photoAllowed ? photoIn : null,
    input,
    h('button', { class: 'icon-btn', type: 'submit', disabled: !canSend, 'aria-label': t('send'), style: { background: 'var(--amber)', color: 'var(--amber-ink)', borderColor: 'var(--amber)' } }, icon('send')));

  const view = h('div', { class: 'convo' }, head, bubbles, composer);
  setTimeout(() => { bubbles.scrollTop = bubbles.scrollHeight; });
  return view;
}

function rerenderChats() {
  renderMobileTabs();
  if (state.side !== 'chats') {
    renderSide(); // refresh the unread badge on the tab
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
  state.online.delete(peerId);
  if (state.active === peerId) state.active = null;
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
  const facts = [genderLabel(card.gender), card.age, ...(card.tags || []), card.location, hereFor(card.online_since)].filter(Boolean);

  let d;
  const modBox = isStaff() && card.role === 'user' && h('div', { class: 'mod-box' },
    h('b', {}, t('desk')),
    h('div', { class: 'actions', style: { justifyContent: 'flex-start' } },
      h('button', { class: 'btn sm', type: 'button', onclick: () => { modAct(`/admin/users/${id}/kick`); d.close(); } }, t('kick')),
      h('button', { class: 'btn sm', type: 'button', onclick: () => { modAct(`/admin/users/${id}/mute`, { minutes: 30 }); d.close(); } }, t('mute30')),
      h('button', { class: 'btn sm danger', type: 'button', onclick: () => { modAct(`/admin/users/${id}/ban`, { minutes: 1440, ip: false, reason: 'from card' }); d.close(); } }, t('ban24')),
      h('button', { class: 'btn sm danger', type: 'button', onclick: () => { modAct(`/admin/users/${id}/ban`, { minutes: 1440, ip: true, reason: 'from card' }); d.close(); } }, t('banIP'))));

  d = dialog(h('div', { class: 'dlg' },
    h('div', { class: 'sheet-photo', style: pictureStyle({ ...card, photo_url: c?.peer.photo_url }) }),
    h('h3', {}, h('span', { class: 'who-name', 'data-gender': card.gender }, card.name), ' ', flag(card.country)),
    card.about ? h('p', { style: { margin: 0, lineHeight: 1.5, color: 'var(--text-2)' } }, card.about) : null,
    h('div', { class: 'facts' }, facts.map((f) => h('span', { class: 'fact' }, f))),
    h('div', { class: 'actions' },
      h('button', { class: 'btn ghost', type: 'button', onclick: () => { d.close(); openReport('user', id); } }, t('report')),
      h('button', { class: 'btn', type: 'button', onclick: () => { d.close(); blockPerson(id); } }, t('block')),
      h('button', { class: 'btn primary', type: 'button', onclick: () => { d.close(); openChat(id); } }, c ? t('open') : t('knock'))),
    modBox));
}

function openReport(targetType, targetId) {
  let reason = '';
  const err = h('div', { class: 'error-text', role: 'alert' });
  const note = h('textarea', { class: 'input', rows: 2, maxlength: 500 });
  let d;
  d = dialog(h('form', {
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
  const photoIn = h('input', {
    type: 'file', accept: 'image/*', class: 'sr', id: 'edit-photo',
    onchange: async (e) => {
      const file = e.target.files[0];
      if (!file) return;
      try {
        const up = await uploadPhoto(file);
        photoId = up.id;
        Object.assign(preview.style, { backgroundImage: `url("${up.url}")`, imageRendering: '' });
      } catch (ex) { err.textContent = ex.message; }
    },
  });
  let d;
  d = dialog(h('form', {
    class: 'dlg',
    onsubmit: async (e) => {
      e.preventDefault();
      const body = { tags: tagBox.value(), about: about.value.trim(), location: loc.value.trim() };
      if (photoId !== undefined) body.photo_id = photoId;
      try {
        state.me = await api('PATCH', '/me', body);
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
  if (!state.token || state.ended) return;
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const ws = new WebSocket(`${proto}://${location.host}/ws?token=${encodeURIComponent(state.token)}`);
  state.ws = ws;
  let opened = false;

  ws.onopen = () => { opened = true; state.retry = 0; };
  ws.onmessage = (e) => {
    let frame;
    try { frame = JSON.parse(e.data); } catch { return; }
    (HANDLERS[frame.type] || (() => {}))(frame);
  };
  ws.onclose = async () => {
    if (state.ws !== ws || state.ended) return;
    state.ws = null;
    if (!opened) {
      // The upgrade was refused: find out whether the session is still valid.
      try { await api('GET', '/me'); } catch (ex) {
        if (ex.status === 401 || ex.status === 403) { endSession(ex.message); return; }
      }
    }
    toast(t('reconnecting'));
    const delay = Math.min(15000, 500 * 2 ** state.retry++);
    setTimeout(connect, delay);
  };
}

function send(frame) {
  if (state.ws?.readyState === WebSocket.OPEN) state.ws.send(JSON.stringify(frame));
  else toast(t('reconnecting'), true);
}

const HANDLERS = {
  hello(f) {
    state.me = f.you;
    state.online = new Map(f.online.filter((c) => c.id !== f.you.id).map((c) => [c.id, c]));
    // The server is the source of truth for private chats: doors survive a
    // quick reconnect but close when either side really disconnects.
    const doors = new Map((f.doors || []).map((d) => [d.with, d]));
    for (const id of doors.keys()) if (state.online.has(id)) convo(id);
    for (const [id, c] of state.convos) {
      const card = state.online.get(id);
      if (!card) { c.phase = 'gone'; continue; }
      const d = doors.get(id);
      c.peer = { ...card, photo_url: d?.open && card.has_photo ? c.peer.photo_url : undefined };
      if (!d) {
        if (c.phase !== null) { c.phase = null; c.msgs = []; }
      } else {
        c.phase = d.open ? 'open' : d.knocked_by_me ? 'knocked' : 'incoming';
      }
    }
    state.feed = [];
    if (state.screen === 'room') renderShell();
    send({ type: 'join', room_id: MAIN_ROOM });
  },

  history(f) {
    if (f.room_id !== MAIN_ROOM) return;
    state.feed = [];
    const feed = $('#feed');
    if (feed) feed.replaceChildren();
    for (const ev of [...f.messages].reverse()) addFeed(ev);
  },

  chat(f) {
    if (f.room_id === MAIN_ROOM) addFeed(f);
  },

  presence(f) {
    if (f.user_id === state.me?.id) {
      if (f.user) state.me = { ...state.me, ...f.user, photo_url: f.user.has_photo ? state.me.photo_url : undefined };
      return;
    }
    const c = state.convos.get(f.user_id);
    if (f.event === 'leave') {
      state.online.delete(f.user_id);
      if (c) { c.phase = 'gone'; rerenderChats(); }
    } else {
      state.online.set(f.user_id, f.user);
      if (c) {
        c.peer = { ...f.user, photo_url: f.user.has_photo ? c.peer.photo_url : undefined };
        if (c.phase === 'gone') { // the door closed when they left; a new knock starts over
          c.phase = null;
          c.msgs = [];
          c.peer.photo_url = undefined;
        }
      }
    }
    refreshCounts();
    if (state.side === 'people') renderSide();
    else if (c) rerenderChats();
  },

  dm(f) {
    const mine = f.from === state.me.id;
    const peerId = mine ? f.to : f.from;
    const c = convo(peerId);
    if (f.knock) {
      c.phase = mine ? 'knocked' : 'incoming';
      c.msgs = []; // a knock opens a new conversation; the old one ended with its door
    } else if (c.phase !== 'open') {
      c.phase = 'open';
    }
    c.msgs.push(f);
    if (!mine && !isViewing(peerId)) {
      c.unread++;
      if (f.knock) toast(t('knockIn', { name: c.peer.name }));
    }
    rerenderChats();
  },

  door(f) {
    const c = convo(f.with.id);
    c.peer = f.with;
    c.phase = 'open';
    c.msgs.push({ door: true, ts: Date.now() / 1000 });
    rerenderChats();
    if (state.side === 'people') renderSide();
  },

  removed(f) {
    state.feed = state.feed.filter((ev) => ev.id !== f.message_id);
    document.querySelectorAll(`[data-id="${f.message_id}"]`).forEach((el) => el.remove());
    for (const c of state.convos.values()) c.msgs = c.msgs.filter((m) => m.id !== f.message_id);
  },

  notice(f) {
    if (f.code === 'kicked' || f.code === 'banned') {
      endSession(f.text);
      return;
    }
    toast(f.text, true);
  },

  error(f) {
    toast(f.error, true);
  },
};

// ---------------------------------------------------------------------------
// Screen: Mod desk
// ---------------------------------------------------------------------------

let deskTab = 'overview';
let deskTimer = null;
const isAdmin = () => state.me?.role === 'admin';

function renderDesk() {
  state.screen = 'desk';
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
  let d;
  d = dialog(h('div', { class: 'dlg' },
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
  if (state.active != null && isViewing(state.active)) {
    convo(state.active).unread = 0;
    rerenderChats();
  }
});

async function boot() {
  if (!state.token) { renderEnter(); return; }
  try {
    state.me = await api('GET', '/me');
    startRoom();
  } catch {
    writeStore('token', null, sessionStorage);
    state.token = null;
    renderEnter();
  }
}

boot();
