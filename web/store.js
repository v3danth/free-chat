// Browser storage that never throws. Private mode and a full quota turn
// reads into null and writes into no-ops, so callers need no try/catch.

export function readStore(key, store = localStorage) {
  try { return store.getItem(key); } catch { return null; }
}

export function writeStore(key, value, store = localStorage) {
  try { value == null ? store.removeItem(key) : store.setItem(key, value); } catch { /* private mode */ }
}

export function readJSON(key) {
  try { return JSON.parse(readStore(key)); } catch { return null; }
}

export function writeJSON(key, value) {
  writeStore(key, value == null ? null : JSON.stringify(value));
}
