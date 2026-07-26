// hud.js — renders the agent table and event log from sim-state. Pure DOM
// rendering; takes state in, has no state of its own.

// CREDIT_BAR_MAX is a display-only scale (agents can exceed it; the bar
// just clamps at 100%) since credits are unbounded, unlike energy's 0-100.
const CREDIT_BAR_MAX = 20;

export function renderHud(container, snapshot) {
  if (!snapshot || !snapshot.agents || snapshot.agents.length === 0) {
    container.innerHTML = "";
    const empty = document.createElement("p");
    empty.className = "empty-state";
    empty.textContent = snapshot ? "No agents in this world." : "No run selected.";
    container.appendChild(empty);
    return;
  }

  const table = document.createElement("table");
  const thead = document.createElement("thead");
  thead.innerHTML =
    "<tr><th>Name</th><th>Location</th><th>Mood</th><th>Energy</th><th>Credits</th></tr>";
  table.appendChild(thead);

  const tbody = document.createElement("tbody");
  for (const agent of snapshot.agents) {
    tbody.appendChild(renderAgentRow(agent));
  }
  table.appendChild(tbody);

  container.innerHTML = "";
  const caption = document.createElement("p");
  caption.className = "empty-state";
  caption.textContent = `World ${snapshot.world_id} · tick ${snapshot.timestep}`;
  container.appendChild(caption);
  container.appendChild(table);
}

function renderAgentRow(agent) {
  const tr = document.createElement("tr");

  const name = document.createElement("td");
  name.textContent = agent.name || agent.id;
  tr.appendChild(name);

  const location = document.createElement("td");
  location.textContent = agent.location;
  tr.appendChild(location);

  const mood = document.createElement("td");
  mood.textContent = agent.mood || "—";
  tr.appendChild(mood);

  const energy = document.createElement("td");
  energy.appendChild(bar(agent.energy, 100, energyClass(agent.energy)));
  tr.appendChild(energy);

  const credits = document.createElement("td");
  credits.appendChild(bar(agent.credits, CREDIT_BAR_MAX, "credits", agent.credits));
  tr.appendChild(credits);

  return tr;
}

function energyClass(energy) {
  if (energy < 20) return "energy-critical";
  if (energy < 50) return "energy-low";
  return "";
}

function bar(value, max, extraClass, labelOverride) {
  const track = document.createElement("div");
  track.className = "bar-track";

  const fill = document.createElement("div");
  fill.className = `bar-fill${extraClass ? " " + extraClass : ""}`;
  const pct = Math.max(0, Math.min(100, (value / max) * 100));
  fill.style.width = `${pct}%`;
  track.appendChild(fill);

  const label = document.createElement("span");
  label.textContent = labelOverride !== undefined ? labelOverride : value;
  label.style.marginLeft = "0.4rem";
  const wrapper = document.createElement("div");
  wrapper.style.display = "flex";
  wrapper.style.alignItems = "center";
  wrapper.appendChild(track);
  wrapper.appendChild(label);
  return wrapper;
}

export function renderEventLog(listEl, events) {
  listEl.innerHTML = "";
  for (let i = events.length - 1; i >= 0; i--) {
    listEl.appendChild(renderEventRow(events[i]));
  }
}

function renderEventRow(evt) {
  const li = document.createElement("li");
  const hasError = evt.payload && evt.payload.error;
  if (hasError) li.classList.add("evt-error");

  const tick = document.createElement("span");
  tick.className = "evt-tick";
  tick.textContent = `#${evt.tick}`;
  li.appendChild(tick);

  const type = document.createElement("span");
  type.className = "evt-type";
  type.textContent = evt.type;
  li.appendChild(type);

  const detail = document.createElement("span");
  const target = evt.target_id ? ` -> ${evt.target_id}` : "";
  const errorText = hasError ? ` (${evt.payload.error})` : "";
  detail.textContent = `${evt.actor_id}${target}${errorText}`;
  li.appendChild(detail);

  return li;
}
