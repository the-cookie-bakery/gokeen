package maths

import "math"

type Vec3 struct {
	X, Y, Z float32
}

func V3(x, y, z float32) Vec3 {
	return Vec3{X: x, Y: y, Z: z}
}

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z}
}

func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z}
}

func (v Vec3) Mul(o Vec3) Vec3 {
	return Vec3{v.X * o.X, v.Y * o.Y, v.Z * o.Z}
}

func (v Vec3) Scale(s float32) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}

func (v Vec3) Neg() Vec3 {
	return Vec3{-v.X, -v.Y, -v.Z}
}

func (v Vec3) Dot(o Vec3) float32 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

func (v Vec3) LengthSquared() float32 {
	return v.Dot(v)
}

func (v Vec3) Length() float32 {
	return float32(math.Sqrt(float64(v.LengthSquared())))
}

func (v Vec3) DistanceSquared(o Vec3) float32 {
	return v.Sub(o).LengthSquared()
}

func (v Vec3) Distance(o Vec3) float32 {
	return float32(math.Sqrt(float64(v.DistanceSquared(o))))
}

func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l <= Epsilon {
		return Vec3{}
	}
	return v.Scale(1 / l)
}

func (v Vec3) Lerp(o Vec3, t float32) Vec3 {
	return Vec3{
		v.X + (o.X-v.X)*t,
		v.Y + (o.Y-v.Y)*t,
		v.Z + (o.Z-v.Z)*t,
	}
}

func (v Vec3) Min(o Vec3) Vec3 {
	return Vec3{min32(v.X, o.X), min32(v.Y, o.Y), min32(v.Z, o.Z)}
}

func (v Vec3) Max(o Vec3) Vec3 {
	return Vec3{max32(v.X, o.X), max32(v.Y, o.Y), max32(v.Z, o.Z)}
}

func (v Vec3) IsFinite() bool {
	return isFinite(v.X) && isFinite(v.Y) && isFinite(v.Z)
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func isFinite(v float32) bool {
	f := float64(v)
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
