package maths

type Vec4 struct {
	X, Y, Z, W float32
}

func V4(x, y, z, w float32) Vec4 {
	return Vec4{X: x, Y: y, Z: z, W: w}
}

func (v Vec4) Add(o Vec4) Vec4 {
	return Vec4{v.X + o.X, v.Y + o.Y, v.Z + o.Z, v.W + o.W}
}

func (v Vec4) Sub(o Vec4) Vec4 {
	return Vec4{v.X - o.X, v.Y - o.Y, v.Z - o.Z, v.W - o.W}
}

func (v Vec4) Scale(s float32) Vec4 {
	return Vec4{v.X * s, v.Y * s, v.Z * s, v.W * s}
}

func (v Vec4) Dot(o Vec4) float32 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z + v.W*o.W
}

func (v Vec4) XYZ() Vec3 {
	return Vec3{v.X, v.Y, v.Z}
}
