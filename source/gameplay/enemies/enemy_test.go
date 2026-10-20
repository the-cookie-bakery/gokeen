package enemies

import (
	"testing"

	maths "gokeen/engine/mathematics"
	"gokeen/engine/physics"
)

func floorWorld() *physics.Mesh {
	a := maths.V3(-20, 0, -20)
	b := maths.V3(20, 0, -20)
	c := maths.V3(20, 0, 20)
	d := maths.V3(-20, 0, 20)
	return physics.NewMesh([]physics.Triangle{{A: a, B: c, C: b}, {A: a, B: d, C: c}})
}

func TestEnemyFallsAndLands(t *testing.T) {
	e := New(maths.V3(0, 5, 0))
	world := floorWorld()
	for i := 0; i < 240; i++ {
		e.Update(1.0/120.0, world)
	}
	if !e.Grounded {
		t.Fatalf("enemy not grounded: %+v", e)
	}
	if e.Position.Y < -0.1 || e.Position.Y > 0.1 {
		t.Fatalf("enemy y: %v", e.Position.Y)
	}
}

func TestEnemyRayHit(t *testing.T) {
	e := New(maths.V3(0, 0, -5))
	if _, ok := e.RayHit(maths.V3(0, e.Height*0.5, 0), maths.V3(0, 0, -1), 20); !ok {
		t.Fatal("expected hit")
	}
	if _, ok := e.RayHit(maths.V3(0, e.Height*0.5, 0), maths.V3(0, 0, 1), 20); ok {
		t.Fatal("unexpected hit")
	}
}

func TestEnemyStunStopsMovement(t *testing.T) {
	e := New(maths.V3(0, 0, 0))
	e.Stun(2)
	e.Update(1.0/120.0, nil)
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Fatalf("stunned enemy moved: %+v", e.Velocity)
	}
	if !e.Stunned() {
		t.Fatal("enemy should be stunned")
	}
}
