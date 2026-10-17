// Notification sounds, drawn with Web Audio: nothing to download, nothing
// to cache, and each event has its own voice so people can tell what
// happened without looking.
//
//   knock    two wooden raps            someone knocked on you
//   mention  bright three-note chime    someone wrote @you in the room
//   door     warm rising arpeggio       a door opened
//   dm       soft two-note pop          a private message, chat not on screen
//   tick     faint tick                 a private message in the chat you are reading
//   room     barely-there blip          room chatter (off unless chosen)
//
// Bursts: a flood of messages plays one sound, then at most one more when
// the burst window ends, always the most important kind that arrived.

import { readJSON, writeJSON } from './store.js';

const PRIORITY = { room: 1, tick: 2, dm: 3, door: 4, mention: 5, knock: 6 };
const WINDOW_MS = 1200;
const STORE_KEY = 'drift.sounds';

// A kind set to false here is muted on its own; the rest follow `on`.
const DEFAULTS = { on: true, room: false, knock: true, vibrate: true };
let settings = { ...DEFAULTS, ...readJSON(STORE_KEY) };

export const soundSettings = () => ({ ...settings });

export function setSoundSettings(next) {
  settings = { ...settings, ...next };
  writeJSON(STORE_KEY, settings);
}

// Browsers only let audio start after a tap or key press: unlock on each
// gesture (cheap once running) so a context the OS suspended comes back.
let ctx = null;
function unlock() {
  try {
    ctx ??= new (window.AudioContext || window.webkitAudioContext)();
    if (ctx.state === 'suspended') ctx.resume();
  } catch { ctx = null; }
}
for (const ev of ['pointerdown', 'keydown']) addEventListener(ev, unlock, { passive: true, capture: true });

let quietUntil = 0;
let lastKind = null;
let pending = null;
let timer = null;

// notify plays kind now, or folds it into the current burst.
export function notify(kind) {
  if (!settings.on || settings[kind] === false) return;
  const now = performance.now();
  if (now >= quietUntil || PRIORITY[kind] > PRIORITY[lastKind]) {
    fire(kind, now);
    return;
  }
  if (!pending || PRIORITY[kind] > PRIORITY[pending]) pending = kind;
  timer ??= setTimeout(() => {
    timer = null;
    const k = pending;
    pending = null;
    if (k) fire(k, performance.now());
  }, quietUntil - now);
}

function fire(kind, now) {
  lastKind = kind;
  quietUntil = now + WINDOW_MS;
  play(kind);
  if (settings.vibrate && document.visibilityState === 'visible' && navigator.vibrate) {
    const pattern = { knock: [30, 70, 30], mention: [20, 40, 20], dm: [25] }[kind];
    if (pattern) navigator.vibrate(pattern);
  }
}

// play sounds kind right away, ignoring settings and bursts (previews).
export function play(kind) {
  if (!ctx || ctx.state !== 'running') return;
  const t = ctx.currentTime + 0.01;
  VOICES[kind]?.(t);
}

// tone is one enveloped oscillator: a quick attack, an exponential tail.
function tone(freq, at, { dur = 0.12, type = 'sine', gain = 0.15, to = null } = {}) {
  const osc = ctx.createOscillator();
  const amp = ctx.createGain();
  osc.type = type;
  osc.frequency.setValueAtTime(freq, at);
  if (to) osc.frequency.exponentialRampToValueAtTime(to, at + dur);
  amp.gain.setValueAtTime(0.0001, at);
  amp.gain.exponentialRampToValueAtTime(gain, at + 0.008);
  amp.gain.exponentialRampToValueAtTime(0.0001, at + dur);
  osc.connect(amp).connect(ctx.destination);
  osc.start(at);
  osc.stop(at + dur + 0.02);
}

// knuckleNoise is the decaying noise burst of a rap. It is made once per
// context: one buffer can feed any number of sources, and a knock raps twice.
const knuckles = new WeakMap();
function knuckleNoise() {
  let buf = knuckles.get(ctx);
  if (!buf) {
    const len = Math.floor(ctx.sampleRate * 0.05);
    buf = ctx.createBuffer(1, len, ctx.sampleRate);
    const samples = buf.getChannelData(0);
    for (let i = 0; i < len; i++) samples[i] = (Math.random() * 2 - 1) * (1 - i / len) ** 3;
    knuckles.set(ctx, buf);
  }
  return buf;
}

// rap is a knuckle on wood: a short band-passed noise click over a low thud.
function rap(at) {
  const noise = ctx.createBufferSource();
  noise.buffer = knuckleNoise();
  const band = ctx.createBiquadFilter();
  band.type = 'bandpass';
  band.frequency.value = 1100;
  band.Q.value = 3;
  const amp = ctx.createGain();
  amp.gain.value = 0.55;
  noise.connect(band).connect(amp).connect(ctx.destination);
  noise.start(at);
  tone(190, at, { dur: 0.09, type: 'triangle', gain: 0.3, to: 120 });
}

const VOICES = {
  knock: (t) => { rap(t); rap(t + 0.17); },
  mention: (t) => {
    tone(1318.5, t, { dur: 0.16, type: 'triangle', gain: 0.12 });
    tone(1661.2, t + 0.07, { dur: 0.18, type: 'triangle', gain: 0.11 });
    tone(1975.5, t + 0.14, { dur: 0.32, type: 'triangle', gain: 0.1 });
  },
  door: (t) => {
    [523.3, 659.3, 784, 1046.5].forEach((f, i) => tone(f, t + i * 0.085, { dur: 0.4 - i * 0.04, gain: 0.11 }));
  },
  dm: (t) => {
    tone(740, t, { dur: 0.08, gain: 0.16, to: 820 });
    tone(1110, t + 0.075, { dur: 0.14, gain: 0.14 });
  },
  tick: (t) => tone(1400, t, { dur: 0.05, gain: 0.06 }),
  room: (t) => tone(980, t, { dur: 0.06, gain: 0.04 }),
};
