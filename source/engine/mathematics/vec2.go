package maths

import "math"

type Vec2 struct {
	X, Y float32
}

func V2(x, y float32) Vec2 {
	return Vec2{X: x, Y: y}
}

func (v Vec2) Add(o Vec2) Vec2 {
	return Vec2{v.X + o.X, v.Y + o.Y}
}

func (v Vec2) Sub(o Vec2) Vec2 {
	return Vec2{v.X - o.X, v.Y - o.Y}
}

func (v Vec2) Scale(s float32) Vec2 {
	return Vec2{v.X * s, v.Y * s}
}

func (v Vec2) Dot(o Vec2) float32 {
	return v.X*o.X + v.Y*o.Y
}

func (v Vec2) LengthSquared() float32 {
	return v.Dot(v)
}

func (v Vec2) Length() float32 {
	return float32(math.Sqrt(float64(v.LengthSquared())))
}

func (v Vec2) Normalize() Vec2 {
	l := v.Length()
	if l <= Epsilon {
		return Vec2{}
	}
	return v.Scale(1 / l)
}

func (v Vec2) Lerp(o Vec2, t float32) Vec2 {
	return Vec2{v.X + (o.X-v.X)*t, v.Y + (o.Y-v.Y)*t}
}
