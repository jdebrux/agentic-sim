"""A2A agent for agentic-sim, built with LangGraph.

Runs a one-node LangGraph graph that asks a chat model to decide an action
from the world view, then exposes it as an A2A server. Unlike ADK, LangGraph
has no built-in "wrap as A2A server" helper, so this wires the pieces by
hand: AgentExecutor -> DefaultRequestHandler -> A2AStarletteApplication.

Run:
    uv run main.py

Then register with a running simserver via its Agent Card URL, e.g.:
    curl -X POST localhost:8080/simulate -d '{
      "duration_ms": 10000, "tick_ms": 1000,
      "agents": [{"id": "bob", "name": "Bob", "location": "loc_default",
                  "traits": {"friendliness": 4, "curiosity": 8},
                  "goals": ["gather_info"], "energy": 100, "credits": 5,
                  "agent_card_url": "http://localhost:9002"}]
    }'
"""

import os
from typing import TypedDict

import uvicorn
from a2a.server.agent_execution import AgentExecutor, RequestContext
from a2a.server.apps import A2AStarletteApplication
from a2a.server.events import EventQueue
from a2a.server.request_handlers import DefaultRequestHandler
from a2a.server.tasks import InMemoryTaskStore
from a2a.types import AgentCapabilities, AgentCard, AgentSkill
from a2a.utils import new_agent_text_message
from langchain_core.messages import HumanMessage, SystemMessage
from langchain_litellm import ChatLiteLLM
from langgraph.graph import END, START, StateGraph

SYSTEM_PROMPT = """You are an autonomous agent living in a shared multi-agent simulation.

Each turn you receive a JSON "world view" describing your state and your surroundings:

{
  "world_id": "world-1",
  "tick": 5,
  "self": {
    "id": "bob", "name": "Bob", "location": "loc_default",
    "traits": {"friendliness": 4, "curiosity": 8},
    "goals": ["gather_info"], "energy": 85, "credits": 5
  },
  "other_agents": [
    {"id": "alice", "name": "Alice", "location": "loc_market", "energy": 92, "credits": 9, ...}
  ],
  "locations": [
    {"id": "loc_default", "name": "Central Plaza"},
    {"id": "loc_market", "name": "Marketplace"},
    {"id": "loc_park", "name": "Park"}
  ],
  "recent_events": [{"type": "move", "actor_id": "alice", "tick": 4, ...}],
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

MODEL = os.getenv("MODEL", "gpt-4o")
AGENT_NAME = os.getenv("AGENT_NAME", "langgraph-agent")
HOST = os.getenv("HOST", "localhost")
PORT = int(os.getenv("PORT", "9002"))

# API_BASE/API_KEY are optional and provider-agnostic: LiteLLM forwards them
# as-is to whatever backend MODEL resolves to, so the same two env vars work
# for OpenAI, OpenRouter, a self-hosted OpenAI-compatible server (Ollama,
# vLLM, ...), etc. Leave them unset to fall back to LiteLLM's normal
# provider auto-detection (e.g. reading OPENAI_API_KEY for a bare "gpt-4o").
litellm_kwargs: dict[str, str] = {}
if api_base := os.getenv("API_BASE"):
    litellm_kwargs["api_base"] = api_base
if api_key := os.getenv("API_KEY"):
    litellm_kwargs["api_key"] = api_key

llm = ChatLiteLLM(model=MODEL, **litellm_kwargs)


class GraphState(TypedDict):
    world_view: str
    action: str


def _as_text(content: str | list) -> str:
    """Flatten a chat model's content into plain text.

    Reasoning-capable models (via LiteLLM) can return a list of content
    blocks — e.g. [{"type": "thinking", ...}, {"type": "text", "text": "..."}]
    — instead of a plain string. A2A's TextPart requires a str, so this keeps
    only the text blocks and drops thinking/tool-use/etc.
    """
    if isinstance(content, str):
        return content
    parts = []
    for block in content:
        if isinstance(block, str):
            parts.append(block)
        elif isinstance(block, dict) and block.get("type", "text") == "text":
            parts.append(block.get("text", ""))
    return "".join(parts)


async def decide(state: GraphState) -> GraphState:
    response = await llm.ainvoke(
        [
            SystemMessage(content=SYSTEM_PROMPT),
            HumanMessage(content=state["world_view"]),
        ]
    )
    return {"action": _as_text(response.content)}


builder = StateGraph(GraphState)
builder.add_node("decide", decide)
builder.add_edge(START, "decide")
builder.add_edge("decide", END)
graph = builder.compile()


class LangGraphAgentExecutor(AgentExecutor):
    """Bridges A2A requests into a single LangGraph invocation."""

    async def execute(self, context: RequestContext, event_queue: EventQueue) -> None:
        world_view = context.get_user_input()
        result = await graph.ainvoke({"world_view": world_view, "action": ""})
        await event_queue.enqueue_event(new_agent_text_message(result["action"]))

    async def cancel(self, context: RequestContext, event_queue: EventQueue) -> None:
        # This agent never runs long enough to need cancellation.
        pass


agent_card = AgentCard(
    name=AGENT_NAME,
    description="LangGraph-powered agent for agentic-sim",
    url=f"http://{HOST}:{PORT}",
    version="0.1.0",
    capabilities=AgentCapabilities(streaming=False),
    default_input_modes=["text/plain"],
    default_output_modes=["text/plain"],
    skills=[
        AgentSkill(
            id="sim_agent",
            name="Simulation Agent",
            description="Decides an action for the agentic-sim world each tick",
            tags=["simulation", "langgraph"],
        )
    ],
)

request_handler = DefaultRequestHandler(
    agent_executor=LangGraphAgentExecutor(),
    task_store=InMemoryTaskStore(),
)

app = A2AStarletteApplication(agent_card=agent_card, http_handler=request_handler).build()

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=PORT)
