package cameras

import (
	"testing"

	maths "gokeen/engine/mathematics"
)

func TestThirdPersonBehindTarget(t *testing.T) {
	c := NewThirdPerson(maths.V3(0, 0, 0))
	c.Yaw = 0
	c.Pitch = 0
	pos := c.Position()
	if pos.Z <= 0 {
		t.Fatalf("expected camera behind +Z, got %+v", pos)
	}
	vp, _ := c.ViewProjection(16.0 / 9.0)
	if !vp.IsFinite() {
		t.Fatal("non-finite view projection")
	}
}

func TestThirdPersonPitchClamp(t *testing.T) {
	c := NewThirdPerson(maths.V3(0, 0, 0))
	c.Look(0, -10000)
	if c.Pitch > c.MaxPitch {
		t.Fatalf("pitch above max: %v", c.Pitch)
	}
	c.Look(0, 10000)
	if c.Pitch < c.MinPitch {
		t.Fatalf("pitch below min: %v", c.Pitch)
	}
}

func TestThirdPersonMouseRightTurnsRight(t *testing.T) {
	c := NewThirdPerson(maths.Vec3{})
	before := c.Forward()
	c.Look(100, 0)
	after := c.Forward()
	if after.X <= before.X {
		t.Fatalf("expected yaw to turn right: before=%+v after=%+v", before, after)
	}
}
