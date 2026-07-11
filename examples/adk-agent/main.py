"""A2A agent for agentic-sim, powered by Google ADK.

Wraps an ADK `LlmAgent` as an A2A server using ADK's built-in `to_a2a()`
helper. The simserver sends this agent a JSON "world view" as a text
message on every tick; the agent must reply with a single JSON object
describing its chosen action.

Run:
    uv run main.py

Then register with a running simserver via its Agent Card URL, e.g.:
    curl -X POST localhost:8080/simulate -d '{
      "duration_ms": 10000, "tick_ms": 1000,
      "agents": [{"id": "alice", "name": "Alice", "location": "loc_default",
                  "traits": {"friendliness": 7, "curiosity": 3},
                  "goals": ["explore", "trade"], "energy": 100, "credits": 10,
                  "agent_card_url": "http://localhost:9001"}]
    }'
"""

import os

from google.adk.a2a.utils.agent_to_a2a import to_a2a
from google.adk.agents import LlmAgent
from google.adk.models.lite_llm import LiteLlm

SYSTEM_PROMPT = """You are an autonomous agent living in a shared multi-agent simulation.

Each turn you receive a JSON "world view" describing your state and your surroundings:

{
  "world_id": "world-1",
  "tick": 5,
  "self": {
    "id": "alice", "name": "Alice", "location": "loc_default",
    "traits": {"friendliness": 7, "curiosity": 3},
    "goals": ["explore", "trade"], "energy": 85, "credits": 9
  },
  "other_agents": [
    {"id": "bob", "name": "Bob", "location": "loc_market", "energy": 92, "credits": 5, ...}
  ],
  "locations": [
    {"id": "loc_default", "name": "Central Plaza"},
    {"id": "loc_market", "name": "Marketplace"},
    {"id": "loc_park", "name": "Park"}
  ],
  "recent_events": [{"type": "move", "actor_id": "bob", "tick": 4, ...}],
  "at_market": false
}

Respond with EXACTLY ONE JSON object describing your action, and nothing else
(no markdown fences, no commentary). Valid actions:

  {"action": "move", "location": "<location_id>", "reason": "..."}
  {"action": "speak", "message": "...", "reason": "..."}
  {"action": "greet", "target_id": "<agent_id>", "message": "...", "reason": "..."}
  {"action": "trade", "target_id": "<agent_id>", "reason": "..."}
  {"action": "rest", "reason": "..."}
  {"action": "idle", "reason": "..."}

Rules enforced by the simulation:
  - move costs 5 energy; speak/greet/idle cost 1 energy; trade costs 3 energy;
    rest restores 10 energy. Energy is clamped to [0, 100].
  - trade requires both you and target_id to be co-located in "loc_market".
  - greet requires you and target_id to be in the same location.
  - Invalid or malformed actions are treated as idle by the simulation.

Use your traits and goals to decide what to do. Respond with the action JSON only."""

HOST = os.getenv("HOST", "localhost")
PORT = int(os.getenv("PORT", "9001"))
MODEL = os.getenv("MODEL", "gpt-4o")
AGENT_NAME = os.getenv("AGENT_NAME", "adk_agent")

root_agent = LlmAgent(
    model=LiteLlm(model=MODEL),
    name=AGENT_NAME,
    instruction=SYSTEM_PROMPT,
)

# `to_a2a` wraps the ADK agent as a Starlette app: it builds the agent card,
# request handler, and event-to-A2A-message plumbing automatically.
app = to_a2a(root_agent, host=HOST, port=PORT)

if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=PORT)
