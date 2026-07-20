package world

import "testing"

// TestSnapshotDeepCopy verifies a WorldSnapshot is unaffected by later
// mutation of the world it was taken from — the property the engine relies
// on to safely publish snapshots to subscribers while it keeps ticking.
func TestSnapshotDeepCopy(t *testing.T) {
	w := NewWorld()
	w.Agents["agent-1"] = &AgentState{ID: "agent-1", Name: "A", Location: "loc_default", Energy: 100, Credits: 10}

	snap := Snapshot(w)
	if len(snap.Agents) != 1 {
		t.Fatalf("expected 1 agent in snapshot, got %d", len(snap.Agents))
	}

	w.Agents["agent-1"].Energy = 5
	w.Agents["agent-1"].Location = "loc_market"
	w.Agents["agent-2"] = &AgentState{ID: "agent-2", Name: "B", Location: "loc_default"}

	if snap.Agents[0].Energy != 100 {
		t.Fatalf("expected snapshot energy to remain 100, got %d", snap.Agents[0].Energy)
	}
	if snap.Agents[0].Location != "loc_default" {
		t.Fatalf("expected snapshot location to remain loc_default, got %s", snap.Agents[0].Location)
	}
	if len(snap.Agents) != 1 {
		t.Fatalf("expected snapshot to still have 1 agent after later mutation, got %d", len(snap.Agents))
	}
}

// TestSnapshotDeterministicOrder verifies agents and locations are sorted
// by ID, independent of Go's randomized map iteration order.
func TestSnapshotDeterministicOrder(t *testing.T) {
	w := NewWorld()
	w.Agents["zeta"] = &AgentState{ID: "zeta", Name: "Z"}
	w.Agents["alpha"] = &AgentState{ID: "alpha", Name: "A"}
	w.Agents["mid"] = &AgentState{ID: "mid", Name: "M"}

	for i := 0; i < 10; i++ {
		snap := Snapshot(w)
		if len(snap.Agents) != 3 || snap.Agents[0].ID != "alpha" || snap.Agents[1].ID != "mid" || snap.Agents[2].ID != "zeta" {
			t.Fatalf("expected agents sorted alpha,mid,zeta; got %+v", snap.Agents)
		}
		if len(snap.Locations) != 3 || snap.Locations[0].ID != "loc_default" || snap.Locations[1].ID != "loc_market" || snap.Locations[2].ID != "loc_park" {
			t.Fatalf("expected locations sorted; got %+v", snap.Locations)
		}
	}
}

// TestSnapshotCapturesTimestep verifies the snapshot reflects the world's
// timestep at the moment it was taken.
func TestSnapshotCapturesTimestep(t *testing.T) {
	w := NewWorld()
	w.Timestep = 7

	snap := Snapshot(w)
	if snap.Timestep != 7 {
		t.Fatalf("expected timestep 7, got %d", snap.Timestep)
	}
	if snap.WorldID != w.ID {
		t.Fatalf("expected world id %s, got %s", w.ID, snap.WorldID)
	}
}
