package levels

import (
	"encoding/json"
	"testing"
)

type fakeResolver struct {
	meshes map[string]bool
}

func (f fakeResolver) HasMesh(id string) bool     { return f.meshes[id] }
func (f fakeResolver) HasMaterial(id string) bool { return true }
func (f fakeResolver) HasTexture(id string) bool  { return true }
func (f fakeResolver) HasSkeleton(id string) bool { return true }
func (f fakeResolver) HasClip(id string) bool     { return true }

type yorpProps struct {
	Patrol [][]float32 `json:"patrol"`
}

func testRegistry() *Registry {
	r := NewRegistry()
	_ = r.Register(EntityDefinition{
		Type: "yorp",
		Decode: func(raw json.RawMessage) (any, error) {
			var p yorpProps
			if len(raw) == 0 {
				return p, nil
			}
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
	_ = r.Register(EntityDefinition{Type: "exit"})
	return r
}

const goodLevel = `{
  "format_version": 2,
  "id": "level1",
  "name": "Level 1",
  "environment": {
    "ambient": [0.1, 0.1, 0.1],
    "sun": {"direction": [-0.4, -1.0, 0.2], "color": [1, 1, 1], "intensity": 1.0},
    "fog": {"color": [0.4, 0.2, 0.2], "start": 80, "end": 300}
  },
  "geometry": [
    {"id": "static", "mesh": "level1_visual", "collision": "level1_collision", "transform": {"position": [0,0,0], "rotation": [0,0,0]}}
  ],
  "spawn": {"position": [0, 20, -18], "rotation": [0, 90, 0]},
  "entities": [
    {"id": "yorp_01", "type": "yorp", "transform": {"position": [5,8,20], "rotation": [0,0,0]}, "properties": {"patrol": [[5,8,20],[5,8,30]]}},
    {"id": "exit_01", "type": "exit", "transform": {"position": [10,8,60], "rotation": [0,180,0]}}
  ],
  "lights": [
    {"id": "sun", "type": "directional", "transform": {"position": [0,50,0], "rotation": [-45,0,0]}, "color": [1,1,1], "intensity": 1}
  ],
  "triggers": [
    {"id": "exit_trigger", "type": "exit", "transform": {"position": [10,8,60], "rotation": [0,0,0]}, "size": [2,3,2]}
  ]
}`

func TestLoadValidLevel(t *testing.T) {
	lvl, err := Load([]byte(goodLevel), fakeResolver{map[string]bool{"level1_visual": true, "level1_collision": true}}, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if lvl.ID != "level1" || len(lvl.Entities) != 2 || len(lvl.Geometry) != 1 {
		t.Fatalf("level: %+v", lvl)
	}
	if lvl.Spawn.Position.Vector().Y != 20 {
		t.Fatalf("spawn: %+v", lvl.Spawn)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	bad := `{"format_version":2,"id":"x","name":"x","bogus":true,"environment":{"ambient":[0,0,0]},"spawn":{"position":[0,0,0],"rotation":[0,0,0]}}`
	if _, err := Load([]byte(bad), fakeResolver{}, testRegistry()); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestLoadRejectsDuplicateIDs(t *testing.T) {
	bad := `{"format_version":2,"id":"x","name":"x","environment":{"ambient":[0,0,0]},"geometry":[{"id":"a","mesh":"m","transform":{"position":[0,0,0],"rotation":[0,0,0]}},{"id":"a","mesh":"m","transform":{"position":[0,0,0],"rotation":[0,0,0]}}],"spawn":{"position":[0,0,0],"rotation":[0,0,0]}}`
	if _, err := Load([]byte(bad), fakeResolver{map[string]bool{"m": true}}, testRegistry()); err == nil {
		t.Fatal("expected duplicate ID error")
	}
}

func TestLoadRejectsMissingMesh(t *testing.T) {
	if _, err := Load([]byte(goodLevel), fakeResolver{}, testRegistry()); err == nil {
		t.Fatal("expected missing mesh error")
	}
}

func TestLoadRejectsUnknownEntity(t *testing.T) {
	bad := `{"format_version":2,"id":"x","name":"x","environment":{"ambient":[0,0,0]},"spawn":{"position":[0,0,0],"rotation":[0,0,0]},"entities":[{"id":"a","type":"unknown","transform":{"position":[0,0,0],"rotation":[0,0,0]}}]}`
	if _, err := Load([]byte(bad), fakeResolver{}, testRegistry()); err == nil {
		t.Fatal("expected unknown entity error")
	}
}

func TestLoadRejectsBadVersion(t *testing.T) {
	bad := `{"format_version":99,"id":"x","name":"x","environment":{"ambient":[0,0,0]},"spawn":{"position":[0,0,0],"rotation":[0,0,0]}}`
	if _, err := Load([]byte(bad), fakeResolver{}, testRegistry()); err == nil {
		t.Fatal("expected version error")
	}
}
