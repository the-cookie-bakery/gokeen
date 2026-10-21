package players

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

func TestPlayerFallsAndLands(t *testing.T) {
	p := New(maths.V3(0, 5, 0))
	world := floorWorld()
	for i := 0; i < 240; i++ {
		p.Update(1.0/120.0, Input{}, world)
	}
	if !p.Grounded {
		t.Fatalf("player not grounded: %+v", p)
	}
	if p.Position.Y < -0.05 || p.Position.Y > 0.1 {
		t.Fatalf("player y: %v", p.Position.Y)
	}
}

func TestPlayerJumps(t *testing.T) {
	p := New(maths.V3(0, 0.02, 0))
	world := floorWorld()
	for i := 0; i < 20; i++ {
		p.Update(1.0/120.0, Input{}, world)
	}
	p.Update(1.0/120.0, Input{Jump: true}, world)
	if p.Velocity.Y <= 0 {
		t.Fatalf("jump velocity: %v", p.Velocity.Y)
	}
	if p.Grounded {
		t.Fatal("player should leave the ground when jumping")
	}
}

func TestPlayerMovesForward(t *testing.T) {
	p := New(maths.V3(0, 0.02, 0))
	world := floorWorld()
	x0, z0 := p.Position.X, p.Position.Z
	for i := 0; i < 120; i++ {
		p.Update(1.0/120.0, Input{Forward: 1, Yaw: 0}, world)
	}
	if p.Position.Z >= z0 {
		t.Fatalf("expected movement toward -Z, got z=%v from %v", p.Position.Z, z0)
	}
	if d := p.Position.X - x0; d > 1e-4 || d < -1e-4 {
		t.Fatalf("unexpected side movement: x=%v", p.Position.X)
	}
}

func TestPlayerModelMatrix(t *testing.T) {
	p := New(maths.V3(1, 2, 3))
	m := p.ModelMatrix(maths.V3(0, 0.5, 0), 0.8)
	if !m.IsFinite() {
		t.Fatal("non-finite model matrix")
	}
	got := m.TransformPoint(maths.V3(0, 0, 0))
	if got.X != 1 || got.Y != 2.5 || got.Z != 3 {
		t.Fatalf("model origin: %+v", got)
	}
}

func TestPogoBouncesOnLanding(t *testing.T) {
	p := New(maths.V3(0, 0.02, 0))
	p.PogoEnabled = true
	p.PogoDeployed = true
	world := floorWorld()
	for i := 0; i < 20; i++ {
		p.Update(1.0/120.0, Input{}, world)
	}
	p.Update(1.0/120.0, Input{Jump: true}, world)
	if p.Velocity.Y <= p.JumpSpeed {
		t.Fatalf("pogo velocity should exceed jump velocity: %v", p.Velocity.Y)
	}
	if p.Grounded {
		t.Fatal("pogo should leave the ground")
	}
}

func TestPogoRequiresOwnership(t *testing.T) {
	p := New(maths.V3(0, 0.02, 0))
	p.PogoDeployed = true
	world := floorWorld()
	for i := 0; i < 20; i++ {
		p.Update(1.0/120.0, Input{}, world)
	}
	p.Update(1.0/120.0, Input{Jump: true}, world)
	if p.Velocity.Y > p.JumpSpeed {
		t.Fatalf("pogo bounce without ownership: %v", p.Velocity.Y)
	}
}

func TestTogglePogoRequiresOwnership(t *testing.T) {
	p := New(maths.V3(0, 0, 0))
	p.TogglePogo()
	if p.PogoDeployed {
		t.Fatal("pogo deployed without ownership")
	}
	p.PogoEnabled = true
	p.TogglePogo()
	if !p.PogoDeployed {
		t.Fatal("pogo should deploy when owned")
	}
	p.TogglePogo()
	if p.PogoDeployed {
		t.Fatal("pogo should retract when toggled again")
	}
}
