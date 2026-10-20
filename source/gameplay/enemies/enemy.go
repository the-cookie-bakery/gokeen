package enemies

import (
	"math"

	maths "gokeen/engine/mathematics"
	"gokeen/engine/physics"
)

type Enemy struct {
	Position  maths.Vec3
	Velocity  maths.Vec3
	Home      maths.Vec3
	Direction maths.Vec3
	Yaw       float32
	Radius    float32
	Height    float32
	Speed     float32
	Grounded  bool
	StunTime  float32
}

func New(position maths.Vec3) *Enemy {
	return &Enemy{
		Position:  position,
		Home:      position,
		Direction: maths.V3(1, 0, 0),
		Radius:    0.45,
		Height:    1.8,
		Speed:     2.5,
	}
}

func (e *Enemy) Stun(seconds float32) {
	if seconds > e.StunTime {
		e.StunTime = seconds
	}
}

func (e *Enemy) Stunned() bool {
	return e.StunTime > 0
}

func (e *Enemy) Update(dt float32, world *physics.Mesh) {
	if dt <= 0 {
		return
	}
	if e.StunTime > 0 {
		e.StunTime -= dt
		e.Velocity.X = 0
		e.Velocity.Z = 0
	}
	if e.StunTime <= 0 {
		if e.Direction.LengthSquared() < 0.1 {
			e.Direction = maths.V3(1, 0, 0)
		}
		e.Direction = e.Direction.Normalize()
		e.Velocity.X = e.Direction.X * e.Speed
		e.Velocity.Z = e.Direction.Z * e.Speed
		if e.Position.Sub(e.Home).Length() > 5 {
			e.Direction = e.Home.Sub(e.Position)
			e.Direction.Y = 0
			e.Direction = e.Direction.Normalize()
		}
	}
	e.Yaw = float32(math.Atan2(float64(e.Velocity.X), float64(-e.Velocity.Z)))
	e.Velocity.Y -= 26 * dt
	capsule := physics.NewCapsule(
		e.Position.Add(maths.V3(0, e.Radius, 0)),
		e.Position.Add(maths.V3(0, e.Height-e.Radius, 0)),
		e.Radius,
	)
	if world != nil {
		capsule, _ = world.MoveCapsule(capsule, e.Velocity.Scale(dt))
	} else {
		capsule = capsule.Translated(e.Velocity.Scale(dt))
	}
	e.Position = capsule.A.Sub(maths.V3(0, e.Radius, 0))
	e.Grounded = false
	if world != nil && e.Velocity.Y <= 0.1 {
		if hit, ok := world.Raycast(maths.NewRay(capsule.A.Add(maths.V3(0, 0.05, 0)), maths.V3(0, -1, 0)), e.Radius+0.2); ok && hit.Normal.Y > 0.5 {
			e.Grounded = true
			e.Velocity.Y = 0
		}
	}
	if e.Grounded {
		if e.Position.Sub(e.Home).Length() > 5 {
			e.Direction = e.Home.Sub(e.Position)
			e.Direction.Y = 0
			e.Direction = e.Direction.Normalize()
		}
	}
}

func (e *Enemy) Center() maths.Vec3 {
	return e.Position.Add(maths.V3(0, e.Height*0.5, 0))
}

func (e *Enemy) RayHit(origin, direction maths.Vec3, maxDist float32) (float32, bool) {
	center := e.Center()
	oc := center.Sub(origin)
	t := oc.Dot(direction)
	if t < 0 || t > maxDist {
		return 0, false
	}
	closest := origin.Add(direction.Scale(t))
	if closest.Distance(center) > 1.0 {
		return 0, false
	}
	return t, true
}

func (e *Enemy) ModelMatrix(offset maths.Vec3, scale float32) maths.Mat4 {
	rot := maths.QFromAxisAngle(maths.V3(0, 1, 0), e.Yaw)
	if e.Stunned() {
		rot = maths.QFromAxisAngle(maths.V3(1, 0, 0), 1.2).Mul(rot)
	}
	return maths.M4Translate(e.Position.Add(offset)).
		Mul(maths.M4Rotation(rot)).
		Mul(maths.M4Scale(maths.V3(scale, scale, scale)))
}
