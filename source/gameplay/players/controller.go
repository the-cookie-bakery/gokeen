package players

import (
	"math"

	"gokeen/engine/mathematics"
	"gokeen/engine/physics"
)

type Input struct {
	Forward float32
	Strafe  float32
	Jump    bool
	Pogo    bool
	Yaw     float32
}

type Controller struct {
	Position       maths.Vec3
	Velocity       maths.Vec3
	Radius         float32
	Height         float32
	MoveSpeed      float32
	Acceleration   float32
	AirControl     float32
	JumpSpeed      float32
	Gravity        float32
	Grounded       bool
	Yaw            float32
	PogoEnabled    bool
	PogoSpeed      float32
	Health         int
	MaxHealth      int
	Lives          int
	Ammo           int
	Score          int
	BlueKey        bool
	HasGun         bool
	Spawn          maths.Vec3
	ModelYawOffset float32
}

func New(position maths.Vec3) *Controller {
	return &Controller{
		Position:     position,
		Radius:       0.55,
		Height:       3.2,
		MoveSpeed:    8,
		Acceleration: 55,
		AirControl:   0.35,
		JumpSpeed:    9.5,
		Gravity:      26,
		PogoSpeed:    15,
		Health:       100,
		MaxHealth:    100,
		Lives:        3,
		Spawn:        position,
	}
}

func (c *Controller) Update(dt float32, in Input, world *physics.Mesh) {
	if dt <= 0 {
		return
	}
	c.Yaw = in.Yaw
	forward := maths.V3(float32(math.Sin(float64(in.Yaw))), 0, -float32(math.Cos(float64(in.Yaw)))).Normalize()
	right := forward.Cross(maths.V3(0, 1, 0)).Normalize()
	wish := forward.Scale(in.Forward).Add(right.Scale(in.Strafe))
	if wish.LengthSquared() > 1 {
		wish = wish.Normalize()
	}
	control := float32(1)
	if !c.Grounded {
		control = c.AirControl
	}
	target := wish.Scale(c.MoveSpeed)
	deltaV := target.Sub(maths.V3(c.Velocity.X, 0, c.Velocity.Z))
	maxDelta := c.Acceleration * control * dt
	if deltaV.Length() > maxDelta {
		deltaV = deltaV.Normalize().Scale(maxDelta)
	}
	c.Velocity.X += deltaV.X
	c.Velocity.Z += deltaV.Z
	if wish.LengthSquared() == 0 && c.Grounded {
		c.Velocity.X *= 0.8
		c.Velocity.Z *= 0.8
	}
	c.Velocity.Y -= c.Gravity * dt
	if c.Grounded && in.Jump {
		c.Velocity.Y = c.JumpSpeed
		c.Grounded = false
	}
	capsule := physics.NewCapsule(
		c.Position.Add(maths.V3(0, c.Radius, 0)),
		c.Position.Add(maths.V3(0, c.Height-c.Radius, 0)),
		c.Radius,
	)
	if world != nil {
		capsule, _ = world.MoveCapsule(capsule, c.Velocity.Scale(dt))
	} else {
		capsule = capsule.Translated(c.Velocity.Scale(dt))
	}
	c.Position = capsule.A.Sub(maths.V3(0, c.Radius, 0))
	c.Grounded = c.groundCheck(world, capsule)
	if c.Grounded && c.Velocity.Y <= 0 {
		if c.PogoEnabled && in.Pogo {
			c.Velocity.Y = c.PogoSpeed
			c.Grounded = false
		} else {
			c.Velocity.Y = 0
		}
	}
	if c.Position.Y < -100 {
		c.Respawn()
	}
}

func (c *Controller) groundCheck(world *physics.Mesh, capsule physics.Capsule) bool {
	if world == nil || c.Velocity.Y > 0.1 {
		return false
	}
	origin := capsule.A.Add(maths.V3(0, 0.05, 0))
	hit, ok := world.Raycast(maths.NewRay(origin, maths.V3(0, -1, 0)), c.Radius+0.25)
	return ok && hit.Normal.Y > 0.5
}

func (c *Controller) Respawn() {
	c.Position = c.Spawn
	c.Velocity = maths.Vec3{}
	c.Grounded = false
	if c.Lives > 0 {
		c.Lives--
	}
}

func (c *Controller) ModelMatrix(modelOffset maths.Vec3, scale float32) maths.Mat4 {
	rot := maths.QFromAxisAngle(maths.V3(0, 1, 0), -c.Yaw+c.ModelYawOffset)
	return maths.M4Translate(c.Position.Add(modelOffset)).
		Mul(maths.M4Rotation(rot)).
		Mul(maths.M4Scale(maths.V3(scale, scale, scale)))
}

func (c *Controller) Center() maths.Vec3 {
	return c.Position.Add(maths.V3(0, c.Height*0.5, 0))
}
