// Drift: state machines. Every piece of UI state that changes over time is
// a table of state -> event -> next state, so the UI can only be in states
// someone designed. docs/chat-states.md draws each table.

// nextState is the move table allows from state on event, or null.
export const nextState = (table, state, event) => table[state]?.[event] ?? null;

// Refused moves are bugs. On a developer's machine they also reach the
// page, so the simulator's runs surface them without opening the console.
const isDevHost = ['localhost', '127.0.0.1'].includes(location.hostname);
let reportRefusal = null;

// onRefusal registers how the page shows a refused move on a dev host.
export function onRefusal(fn) {
  reportRefusal = fn;
}

export function refuse(machine, state, event) {
  console.error(`state machine "${machine}": no move for "${event}" from "${state}"`);
  if (isDevHost) reportRefusal?.(`${machine}: ${event} from ${state}`);
}

// createMachine holds one current state. send applies a move from the
// table and returns whether it was allowed; onMove sees every allowed move.
export function createMachine(name, table, initial, onMove = null) {
  let state = initial;
  return {
    get state() { return state; },
    is: (...states) => states.includes(state),
    send(event) {
      const next = nextState(table, state, event);
      if (next == null) {
        refuse(name, state, event);
        return false;
      }
      const prev = state;
      state = next;
      onMove?.(next, prev, event);
      return true;
    },
  };
}
