// Faces page: a UI over the avatar endpoints (see SPEC.md, "Faces").
// Every image and fact comes from the server; nothing is generated here.

const VERSION = 'v1';
const TIER_LABEL = { common: 'Common', rare: 'Rare', epic: 'Epic', legendary: 'Legendary shiny' };
const EAR_LABEL = { pointy: 'pointy', floppy: 'floppy', long: 'long', round: 'round', bumps: 'eye bumps', tufts: 'tufts', pig: 'folded', gills: 'gills' };
const EXAMPLES = {
  common: ['75%', 'based_gremlin', 'Flat background, any animal.'],
  rare: ['17%', 'based_boba', 'Two-tone background.'],
  epic: ['6%', 'cozy_potato', 'Sparkle eyes and a crown or halo.'],
  legendary: ['2%', 'sleepy_bean', 'Shiny colours nobody else gets, holo frame, sparkles.'],
};
const STARTERS = ['night_owl', 'maya', 'leo', 'sofia', 'kenji', 'amara', 'noah', 'zara', 'mateo', 'yuki', 'omar', 'lena',
  'chai_lover', 'lowkey_luna', 'goblin_mode', 'npc_42', 'main_character', 'touch_grass', 'aura_farmer', 'vibe_check',
  'side_quest', 'delulu_dana', 'no_cap_nina', 'its_giving'];
const ADJ = ['sleepy', 'feral', 'cozy', 'lowkey', 'chaotic', 'spicy', 'tiny', 'sus', 'based', 'silly', 'soft', 'glitchy', 'goofy', 'unhinged', 'chill', 'neon'];
const NOUN = ['bean', 'goblin', 'noodle', 'cloud', 'gremlin', 'potato', 'mango', 'pixel', 'muffin', 'comet', 'dumpling', 'waffle', 'boba', 'yapper', 'frog', 'moth'];

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
const faceURL = (name, query = '') => `/avatar/${VERSION}/${encodeURIComponent(key(name))}${query}`;

const infoCache = new Map();
function info(name) {
  const k = key(name);
  if (!infoCache.has(k)) {
    infoCache.set(k, fetch(faceURL(name, '/info')).then((r) => (r.ok ? r.json() : Promise.reject(new Error(r.status)))));
  }
  return infoCache.get(k);
}

function toast(text) {
  const t = el('div', { class: 'toast', role: 'status' }, text);
  $('toasts').append(t);
  setTimeout(() => t.remove(), 2600);
}

// ---------------------------------------------------------------------------
// The big face
// ---------------------------------------------------------------------------

let shown = '';
async function show(name) {
  const display = name.trim() || 'stranger';
  shown = key(name);
  const big = $('big');
  big.src = faceURL(name, '?scale=12');
  big.alt = `Pixel face for ${display}`;
  $('big-name').textContent = display;
  $('enter-as').href = `/?name=${encodeURIComponent(name.trim())}`;
  $('save').href = faceURL(name, '?scale=16');
  $('save').setAttribute('download', `${key(name).replace(/[^\p{L}\p{N}_-]+/gu, '_')}.png`);
  new Image().src = faceURL(name, '?scale=12&blink=1'); // warm the blink frame

  const url = new URL(location.href);
  url.searchParams.set('name', name.trim());
  history.replaceState(null, '', url);

  let f;
  try { f = await info(name); } catch { return; }
  if (shown !== key(name)) return; // a newer name was typed meanwhile
  $('frame').dataset.tier = f.tier;
  $('big-tier').textContent = TIER_LABEL[f.tier];
  $('big-tier').dataset.tier = f.tier;
  const facts = [
    ['Animal', f.animal],
    ['Colour', f.colour],
    ['Face size', `${f.face[0]} x ${f.face[1]}`],
    ['Face shape', f.roundness >= 2.6 ? `boxy ${f.roundness}` : f.roundness > 2.2 ? `soft ${f.roundness}` : `round ${f.roundness}`],
    ['Ears', f.ear_size ? `${EAR_LABEL[f.ears]}, size ${f.ear_size}` : EAR_LABEL[f.ears]],
    ['Eye shape', f.extra === 'shades' ? `${f.eye_shape} (in shades)` : f.eye_shape],
    ['Eye size', f.eye_size],
    ['Eye spacing', f.eye_gap],
    ['Muzzle', f.muzzle ? `${f.muzzle[0]} x ${f.muzzle[1]}` : 'none'],
    ['Mouth', f.mouth],
    ['Markings', f.blush ? `${f.markings}, blush` : f.markings],
    ['Extra', f.extra],
  ];
  $('specs').replaceChildren(...facts.map(([k, v]) => el('div', {}, el('dt', {}, k), el('dd', {}, String(v)))));
}

// Blink: swap to the closed-eyes frame for a moment every few seconds.
function startBlink() {
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) return;
  setInterval(() => {
    const name = $('name').value;
    const big = $('big');
    big.src = faceURL(name, '?scale=12&blink=1');
    setTimeout(() => { if ($('name').value === name) big.src = faceURL(name, '?scale=12'); }, 170);
  }, 3400);
}

function pickName(name) {
  $('name').value = name;
  show(name);
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

// ---------------------------------------------------------------------------
// Wall, rarity, rules
// ---------------------------------------------------------------------------

function fillWall(names) {
  $('wall').replaceChildren(...names.map((n) => {
    const tile = el('button', { class: 'tile', type: 'button', onclick: () => pickName(n) },
      el('img', { src: faceURL(n, '?scale=4'), alt: '', loading: 'lazy', width: 96, height: 96 }),
      el('span', {}, n));
    info(n).then((f) => { tile.dataset.tier = f.tier; tile.title = `${f.animal}, ${TIER_LABEL[f.tier]}`; }).catch(() => {});
    return tile;
  }));
}

const randomNames = (k) => Array.from({ length: k }, () => {
  const r = (a) => a[Math.floor(Math.random() * a.length)];
  return `${r(ADJ)}_${r(NOUN)}${Math.random() < 0.4 ? Math.floor(Math.random() * 99) : ''}`;
});

function fillTiers() {
  $('tiers').replaceChildren(...Object.entries(EXAMPLES).map(([tier, [odds, name, text]]) =>
    el('button', { class: 'tier-card', type: 'button', onclick: () => pickName(name) },
      el('span', { class: 'tier', 'data-tier': tier }, TIER_LABEL[tier]),
      el('span', { class: 'odds' }, odds),
      el('img', { src: faceURL(name, '?scale=4'), alt: '', width: 96, height: 96 }),
      el('p', {}, `${text} Try "${name}".`))));
}

async function fillRules() {
  let rules;
  try { rules = await (await fetch('/avatar/rules')).json(); } catch { return; }
  const range = (r) => (r[0] === r[1] ? `${r[0]}` : `${r[0]}-${r[1]}`);
  const head = el('tr', {}, ...['', 'Animal', 'Chance', 'Face w x h', 'Ears', 'Eye shapes', 'Eye size', 'Eye gap', 'Mouths', 'Colours'].map((h) => el('th', { scope: 'col' }, h)));
  const rows = rules.map((r) => el('tr', {},
    el('td', {}, el('button', { class: 'icon-btn bare', type: 'button', 'aria-label': `Show ${r.example}`, onclick: () => pickName(r.example) }, el('img', { src: faceURL(r.example, '?scale=2'), alt: '' }))),
    el('td', {}, `${r.animal}`, el('br'), el('small', { class: 'muted' }, r.tier)),
    el('td', { class: 'num' }, `${(r.chance * 100).toFixed(1)}%`),
    el('td', { class: 'num' }, `${range(r.face_w)} x ${range(r.face_h)}`),
    el('td', {}, r.ear_size[1] ? `${EAR_LABEL[r.ears]} ${range(r.ear_size)}` : EAR_LABEL[r.ears]),
    el('td', {}, r.eye_shapes.join(', ')),
    el('td', { class: 'num' }, range(r.eye_size)),
    el('td', { class: 'num' }, range(r.eye_gap)),
    el('td', {}, r.mouths.join(', ')),
    el('td', {}, `${r.colours.join(', ')}; shiny: ${r.shiny.replace(/^Shiny /, '')}`)));
  $('rules').replaceChildren(el('thead', {}, head), el('tbody', {}, ...rows));
}

// ---------------------------------------------------------------------------
// Boot
// ---------------------------------------------------------------------------

let typing;
$('name').addEventListener('input', (e) => {
  clearTimeout(typing);
  typing = setTimeout(() => show(e.target.value), 120);
});
$('shuffle').addEventListener('click', () => fillWall(randomNames(24)));
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
fillTiers();
fillRules();
startBlink();
