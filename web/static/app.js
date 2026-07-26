import { createSimState } from "./sim-state.js";
import { renderHud, renderEventLog } from "./hud.js";

// Every event type the server ever emits over SSE: the seven world.ActionType
// values (internal/world/action.go) plus the "tick" snapshot event. SSE
// delivers named events (`event: <type>`), which do NOT fire EventSource's
// generic onmessage handler — each type needs its own addEventListener.
const EVENT_TYPES = ["tick", "idle", "move", "speak", "interact", "greet", "trade", "rest"];

const DEFAULT_AGENTS_JSON = {
  agents: [
    {
      id: "alice",
      name: "Alice",
      location: "loc_default",
      traits: { friendliness: 7, curiosity: 3 },
      goals: ["explore", "trade"],
      energy: 100,
      credits: 10,
      agent_card_url: "http://adk-agent:9001",
    },
    {
      id: "bob",
      name: "Bob",
      location: "loc_default",
      traits: { friendliness: 4, curiosity: 8 },
      goals: ["gather_info"],
      energy: 100,
      credits: 5,
      agent_card_url: "http://langgraph-agent:9002",
    },
  ],
};

const simState = createSimState();

const runSelect = document.getElementById("run-select");
const refreshButton = document.getElementById("refresh-runs");
const runStatus = document.getElementById("run-status");
const startForm = document.getElementById("start-run-form");
const agentsJsonField = document.getElementById("agents-json");
const durationField = document.getElementById("duration-ms");
const tickField = document.getElementById("tick-ms");
const startError = document.getElementById("start-run-error");
const agentTableContainer = document.getElementById("agent-table-container");
const eventLogEl = document.getElementById("event-log");

agentsJsonField.value = JSON.stringify(DEFAULT_AGENTS_JSON, null, 2);

let eventSource = null;
let statusPollHandle = null;

simState.subscribe(() => {
  renderHud(agentTableContainer, simState.getSnapshot());
  renderEventLog(eventLogEl, simState.getLog());
});
renderHud(agentTableContainer, null);

async function refreshRuns(selectId) {
  const res = await fetch("/simulate");
  if (!res.ok) return;
  const body = await res.json();
  const runs = body.runs || [];

  const previous = selectId || runSelect.value;
  runSelect.innerHTML = "";
  const placeholder = document.createElement("option");
  placeholder.value = "";
  placeholder.textContent = "— select a run —";
  runSelect.appendChild(placeholder);

  for (const run of runs) {
    const opt = document.createElement("option");
    opt.value = run.id;
    opt.textContent = `${run.id} (${run.status}, ${run.ticks} ticks)`;
    runSelect.appendChild(opt);
  }

  if (previous && runs.some((r) => r.id === previous)) {
    runSelect.value = previous;
  }
}

function setRunStatus(status) {
  if (!status) {
    runStatus.textContent = "";
    runStatus.className = "";
    return;
  }
  runStatus.textContent = `${status.status} · tick ${status.ticks}`;
  runStatus.className = status.status === "running" ? "status-running" : status.status === "error" ? "status-error" : "";
}

function stopStatusPoll() {
  if (statusPollHandle !== null) {
    clearInterval(statusPollHandle);
    statusPollHandle = null;
  }
}

function startStatusPoll(runId) {
  stopStatusPoll();
  statusPollHandle = setInterval(async () => {
    const res = await fetch(`/simulate/${encodeURIComponent(runId)}`);
    if (!res.ok) return;
    const status = await res.json();
    setRunStatus(status);
    if (status.status === "completed" || status.status === "error") {
      stopStatusPoll();
      if (eventSource) {
        eventSource.close();
        eventSource = null;
      }
    }
  }, 2000);
}

function disconnect() {
  stopStatusPoll();
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  setRunStatus(null);
}

async function connect(runId) {
  disconnect();
  simState.reset();

  if (!runId) return;

  // Seed with the latest known state so the HUD isn't empty while waiting
  // for the first "tick" event; a 404 here just means the run hasn't
  // emitted its first snapshot yet, so fall back to the stream.
  const worldRes = await fetch(`/simulate/${encodeURIComponent(runId)}/world`);
  if (worldRes.ok) {
    simState.seedSnapshot(await worldRes.json());
  }

  eventSource = new EventSource(`/simulate/${encodeURIComponent(runId)}/stream`);
  for (const type of EVENT_TYPES) {
    eventSource.addEventListener(type, (evt) => {
      simState.apply(JSON.parse(evt.data));
    });
  }

  startStatusPoll(runId);
}

refreshButton.addEventListener("click", () => refreshRuns());
runSelect.addEventListener("change", () => connect(runSelect.value));

startForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  startError.textContent = "";

  let extra;
  try {
    extra = agentsJsonField.value.trim() ? JSON.parse(agentsJsonField.value) : {};
  } catch (err) {
    startError.textContent = `Invalid JSON: ${err.message}`;
    return;
  }

  const body = Object.assign(
    {
      duration_ms: Number(durationField.value),
      tick_ms: Number(tickField.value),
    },
    extra
  );

  const res = await fetch("/simulate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    startError.textContent = `Failed to start run: ${res.status} ${await res.text()}`;
    return;
  }

  const { id } = await res.json();
  await refreshRuns(id);
  runSelect.value = id;
  connect(id);
});

refreshRuns();
