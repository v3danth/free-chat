// Faces page: a UI over the society mark endpoints (see SPEC.md, "Faces").
// Every image and fact comes from the server; nothing is generated here.

const RANK_LABEL = { initiate: 'Initiate', adept: 'Adept', keeper: 'Keeper', grandmaster: 'Grandmaster' };
const RANK_TEXT = {
  initiate: ['maya', 'A single ring.'],
  adept: ['goblin_mode', 'A double ring.'],
  keeper: ['yuki', 'Double ring, four studs, glowing eyes.'],
  grandmaster: ['night_owl', 'Gold only, rays, the third eye and a moving sheen.'],
};
const STARTERS = ['night_owl', 'maya', 'leo', 'sofia', 'kenji', 'amara', 'noah', 'zara', 'mateo', 'yuki', 'omar', 'lena',
  'velvet_rope', 'midnight_sub', 'brat_tamer', 'quiet_dom', 'silk_and_ash', 'masked_one', 'after_dark', 'lantern_eyes'];
const W1 = ['velvet', 'midnight', 'silent', 'veiled', 'ashen', 'hollow', 'gilded', 'quiet', 'masked', 'crimson', 'lantern', 'sleepless'];
const W2 = ['fox', 'moth', 'raven', 'saint', 'cipher', 'wolf', 'oracle', 'rope', 'silk', 'ember', 'stag', 'hare'];

const $ = (id) => document.getElementById(id);
function el(tag, attrs = {}, ...kids) {
  const n = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k.startsWith('on')) n.addEventListener(k.slice(2), v);
    else n.setAttribute(k, v);
  }
  n.append(...kids.flat().filter((k) => k != null));
  return n;
}

const key = (name) => name.normalize('NFKC').trim().replace(/\s+/g, ' ').toLowerCase() || 'stranger';
const markURL = (name, query = '') => `/avatar/v2/${encodeURIComponent(key(name))}${query}`;

const infoCache = new Map();
function info(name) {
  const k = key(name);
  if (!infoCache.has(k)) infoCache.set(k, fetch(markURL(name, '/info')).then((r) => (r.ok ? r.json() : Promise.reject(new Error(r.status)))));
  return infoCache.get(k);
}

function toast(text) {
  const t = el('div', { class: 'toast', role: 'status' }, text);
  $('toasts').append(t);
  setTimeout(() => t.remove(), 2600);
}

let shown = '';
async function show(name) {
  const display = name.trim() || 'stranger';
  shown = key(name);
  $('big-poly').src = markURL(name, '?style=poly&anim=1');
  $('big-pixel').src = markURL(name, '?style=pixel&scale=9');
  $('big-poly').alt = $('big-pixel').alt = `Society mark for ${display}`;
  $('enter-as').href = `/?name=${encodeURIComponent(name.trim())}`;
  $('save').href = markURL(name, '?style=pixel&scale=16');
  $('save').setAttribute('download', `${key(name).replace(/[^\p{L}\p{N}_-]+/gu, '_')}.png`);
  const url = new URL(location.href);
  url.searchParams.set('name', name.trim());
  history.replaceState(null, '', url);

  let m;
  try { m = await info(name); } catch { return; }
  if (shown !== key(name)) return; // a newer name was typed meanwhile
  for (const id of ['frame-poly', 'frame-pixel']) $(id).dataset.tier = m.rank;
  $('mark-title').textContent = m.title;
  $('mark-number').textContent = `No. ${m.number}`;
  $('big-tier').textContent = RANK_LABEL[m.rank];
  $('big-tier').dataset.tier = m.rank;
  const facts = [
    ['Animal', m.animal], ['Rank', RANK_LABEL[m.rank]], ['Metal', m.metal], ['Mark', m.mark],
    ['Eyes', m.glow ? `${m.eyes}, ${m.glow.toLowerCase()}` : m.eyes], ['Frame', m.frame],
    ['Face width', m.face_width], ['Ears', m.ears], ['Snout', m.snout],
  ];
  $('specs').replaceChildren(...facts.map(([k, v]) => el('div', {}, el('dt', {}, k), el('dd', {}, String(v)))));
}

function pickName(name) {
  $('name').value = name;
  show(name);
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function fillWall(names) {
  $('wall').replaceChildren(...names.map((n) => {
    const tile = el('button', { class: 'tile', type: 'button', onclick: () => pickName(n) },
      el('img', { src: markURL(n), alt: '', loading: 'lazy', width: 100, height: 100 }),
      el('span', {}, n));
    info(n).then((m) => { tile.dataset.tier = m.rank; tile.title = `${m.title}, No. ${m.number}`; }).catch(() => {});
    return tile;
  }));
}

const randomNames = (k) => Array.from({ length: k }, () => {
  const r = (a) => a[Math.floor(Math.random() * a.length)];
  return `${r(W1)}_${r(W2)}${Math.random() < 0.4 ? Math.floor(Math.random() * 99) : ''}`;
});

async function fillRanksAndParts() {
  let parts;
  try { parts = await (await fetch('/avatar/parts')).json(); } catch { return; }
  $('tiers').replaceChildren(...Object.entries(RANK_TEXT).map(([rank, [name, text]]) =>
    el('button', { class: 'tier-card', type: 'button', onclick: () => pickName(name) },
      el('span', { class: 'tier', 'data-tier': rank }, RANK_LABEL[rank]),
      el('span', { class: 'odds' }, `${Math.round(parts.ranks[rank] * 100)}%`),
      el('img', { src: markURL(name), alt: '', width: 96, height: 96 }),
      el('p', {}, `${text} Try "${name}".`))));
  const list = [
    ['Animals', parts.animals.join(', ') + '. One shared mask shape, so every animal belongs to the same order.'],
    ['Metals', parts.metals.join(', ') + '. Gold is reserved for Grandmasters.'],
    ['Eyes', parts.eyes.join(', ') + '. Glowing and slit eyes take one of: ' + parts.glows.join(', ').toLowerCase() + '.'],
    ['Marks on the brow', parts.marks.join(', ') + '. Grandmasters wear the third eye.'],
    ['Frames', parts.frames.join(', ') + '. Rank decides the rings and studs.'],
    ['Title and number', 'A title like "The Silent Fox" and a member number from 0001 to 9999.'],
  ];
  $('parts').replaceChildren(...list.map(([k, v]) => el('div', { class: 'part' }, el('b', {}, k), el('span', {}, v))));
}

let typing;
$('name').addEventListener('input', (e) => {
  clearTimeout(typing);
  typing = setTimeout(() => show(e.target.value), 120);
});
$('shuffle').addEventListener('click', () => fillWall(randomNames(20)));
$('copy').addEventListener('click', async () => {
  try {
    await navigator.clipboard.writeText(location.href);
    toast('Link copied');
  } catch {
    toast(location.href);
  }
});

const start = new URLSearchParams(location.search).get('name');
if (start) $('name').value = start;
show($('name').value);
fillWall(STARTERS);
fillRanksAndParts();
