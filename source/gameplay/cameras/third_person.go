package cameras

import (
	"math"

	"gokeen/engine/mathematics"
)

type ThirdPerson struct {
	Target      maths.Vec3
	Distance    float32
	Height      float32
	Yaw         float32
	Pitch       float32
	Sensitivity float32
	MinPitch    float32
	MaxPitch    float32
}

func NewThirdPerson(target maths.Vec3) *ThirdPerson {
	return &ThirdPerson{
		Target:      target,
		Distance:    10,
		Height:      2,
		Yaw:         0,
		Pitch:       0,
		Sensitivity: 0.002,
		MinPitch:    -1.2,
		MaxPitch:    0.9,
	}
}

func (c *ThirdPerson) Look(dx, dy float32) {
	c.Yaw += dx * c.Sensitivity
	c.Pitch -= dy * c.Sensitivity
	if c.Pitch < c.MinPitch {
		c.Pitch = c.MinPitch
	}
	if c.Pitch > c.MaxPitch {
		c.Pitch = c.MaxPitch
	}
}

func (c *ThirdPerson) Forward() maths.Vec3 {
	cp := float32(math.Cos(float64(c.Pitch)))
	sp := float32(math.Sin(float64(c.Pitch)))
	sy := float32(math.Sin(float64(c.Yaw)))
	cy := float32(math.Cos(float64(c.Yaw)))
	return maths.V3(sy*cp, sp, -cy*cp).Normalize()
}

func (c *ThirdPerson) Position() maths.Vec3 {
	return c.Target.Add(maths.V3(0, c.Height, 0)).Sub(c.Forward().Scale(c.Distance))
}

func (c *ThirdPerson) ViewProjection(aspect float32) (maths.Mat4, maths.Vec3) {
	pos := c.Position()
	view := maths.LookAtRH(pos, c.Target.Add(maths.V3(0, c.Height*0.4, 0)), maths.V3(0, 1, 0))
	proj := maths.PerspectiveRH(70*maths.DegToRad, aspect, 0.1, 1000)
	return proj.Mul(view), pos
}
