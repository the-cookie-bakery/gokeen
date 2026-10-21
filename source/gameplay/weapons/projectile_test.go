package weapons

import (
	"testing"

	maths "gokeen/engine/mathematics"
)

func TestProjectileAdvance(t *testing.T) {
	p := Spawn(maths.Vec3{}, maths.V3(0, 0, -1), 10, 1)
	prev, cur := p.Advance(0.1)
	if prev != (maths.Vec3{}) {
		t.Fatalf("previous: %+v", prev)
	}
	if cur.Z > -0.99 || cur.Z < -1.01 {
		t.Fatalf("current: %+v", cur)
	}
	if !p.Active {
		t.Fatal("projectile expired too early")
	}
}

func TestProjectileExpires(t *testing.T) {
	p := Spawn(maths.Vec3{}, maths.V3(0, 0, -1), 10, 0.1)
	p.Advance(0.2)
	if p.Active {
		t.Fatal("projectile should expire")
	}
}

func TestProjectileDirectionNormalized(t *testing.T) {
	p := Spawn(maths.Vec3{}, maths.V3(0, 0, -10), 5, 1)
	if p.Velocity.Length() < 4.99 || p.Velocity.Length() > 5.01 {
		t.Fatalf("velocity length: %v", p.Velocity.Length())
	}
}
