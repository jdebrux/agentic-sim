// sim-state.js — holds the latest world snapshot and a bounded log of
// action events for the currently-selected run, and notifies subscribers
// whenever either changes. Pure state; no DOM, no networking.

const MAX_LOG_EVENTS = 200;

export function createSimState() {
  let snapshot = null; // world.WorldSnapshot, or null before the first tick event
  let log = []; // most-recent-last action events (everything except "tick")
  const listeners = new Set();

  function notify() {
    for (const fn of listeners) fn();
  }

  return {
    subscribe(fn) {
      listeners.add(fn);
      return () => listeners.delete(fn);
    },

    // reset clears state when switching to a different run, so the UI
    // doesn't show a stale snapshot or mix event logs across runs.
    reset() {
      snapshot = null;
      log = [];
      notify();
    },

    // seedSnapshot sets the initial snapshot from GET /simulate/{id}/world,
    // ahead of the first "tick" event arriving over SSE.
    seedSnapshot(snap) {
      snapshot = snap;
      notify();
    },

    // apply routes one SSE event: "tick" events replace the snapshot,
    // everything else is appended to the bounded event log.
    apply(event) {
      if (event.type === "tick") {
        const payload = event.payload || {};
        if (payload.snapshot) snapshot = payload.snapshot;
      } else {
        log.push(event);
        if (log.length > MAX_LOG_EVENTS) log.shift();
      }
      notify();
    },

    getSnapshot() {
      return snapshot;
    },

    getLog() {
      return log;
    },
  };
}
