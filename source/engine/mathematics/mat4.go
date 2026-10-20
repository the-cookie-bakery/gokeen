package maths

import "math"

// Mat4 is a column-major 4x4 matrix compatible with Vulkan.
// Element index = column*4 + row.
type Mat4 [16]float32

func M4Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func (m Mat4) Mul(o Mat4) Mat4 {
	var r Mat4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			var sum float32
			for k := 0; k < 4; k++ {
				sum += m[k*4+row] * o[col*4+k]
			}
			r[col*4+row] = sum
		}
	}
	return r
}

func (m Mat4) TransformPoint(v Vec3) Vec3 {
	x := m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]
	y := m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]
	z := m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]
	w := m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]
	if w != 0 && w != 1 {
		return Vec3{x / w, y / w, z / w}
	}
	return Vec3{x, y, z}
}

func (m Mat4) TransformVector(v Vec3) Vec3 {
	return Vec3{
		m[0]*v.X + m[4]*v.Y + m[8]*v.Z,
		m[1]*v.X + m[5]*v.Y + m[9]*v.Z,
		m[2]*v.X + m[6]*v.Y + m[10]*v.Z,
	}
}

func M4Translate(v Vec3) Mat4 {
	m := M4Identity()
	m[12], m[13], m[14] = v.X, v.Y, v.Z
	return m
}

func M4Scale(v Vec3) Mat4 {
	return Mat4{
		v.X, 0, 0, 0,
		0, v.Y, 0, 0,
		0, 0, v.Z, 0,
		0, 0, 0, 1,
	}
}

func M4Rotation(q Quat) Mat4 {
	return q.Mat4()
}

// PerspectiveRH builds a right-handed perspective matrix producing Vulkan NDC
// (x right, y down, z in [0,1]).
func PerspectiveRH(fovYRadians, aspect, near, far float32) Mat4 {
	if aspect <= 0 {
		aspect = 1
	}
	f := 1 / float32(math.Tan(float64(fovYRadians)/2))
	nf := 1 / (near - far)
	return Mat4{
		f / aspect, 0, 0, 0,
		0, -f, 0, 0,
		0, 0, far * nf, -1,
		0, 0, near * far * nf, 0,
	}
}

func OrthographicRH(left, right, bottom, top, near, far float32) Mat4 {
	rl := 1 / (right - left)
	tb := 1 / (top - bottom)
	fn := 1 / (far - near)
	return Mat4{
		2 * rl, 0, 0, 0,
		0, 2 * tb, 0, 0,
		0, 0, fn, 0,
		-(right + left) * rl, -(top + bottom) * tb, -near * fn, 1,
	}
}

// LookAtRH returns a view matrix using right-handed coordinates where the
// camera looks down -Z.
func LookAtRH(eye, target, up Vec3) Mat4 {
	forward := target.Sub(eye).Normalize()
	if forward.LengthSquared() <= Epsilon {
		forward = V3(0, 0, -1)
	}
	right := forward.Cross(up).Normalize()
	if right.LengthSquared() <= Epsilon {
		right = V3(1, 0, 0)
	}
	trueUp := right.Cross(forward)
	return Mat4{
		right.X, trueUp.X, -forward.X, 0,
		right.Y, trueUp.Y, -forward.Y, 0,
		right.Z, trueUp.Z, -forward.Z, 0,
		-right.Dot(eye), -trueUp.Dot(eye), forward.Dot(eye), 1,
	}
}

func (m Mat4) Inverse() (Mat4, bool) {
	var inv Mat4
	inv[0] = m[5]*m[10]*m[15] - m[5]*m[11]*m[14] - m[9]*m[6]*m[15] + m[9]*m[7]*m[14] + m[13]*m[6]*m[11] - m[13]*m[7]*m[10]
	inv[4] = -m[4]*m[10]*m[15] + m[4]*m[11]*m[14] + m[8]*m[6]*m[15] - m[8]*m[7]*m[14] - m[12]*m[6]*m[11] + m[12]*m[7]*m[10]
	inv[8] = m[4]*m[9]*m[15] - m[4]*m[11]*m[13] - m[8]*m[5]*m[15] + m[8]*m[7]*m[13] + m[12]*m[5]*m[11] - m[12]*m[7]*m[9]
	inv[12] = -m[4]*m[9]*m[14] + m[4]*m[10]*m[13] + m[8]*m[5]*m[14] - m[8]*m[6]*m[13] - m[12]*m[5]*m[10] + m[12]*m[6]*m[9]
	inv[1] = -m[1]*m[10]*m[15] + m[1]*m[11]*m[14] + m[9]*m[2]*m[15] - m[9]*m[3]*m[14] - m[13]*m[2]*m[11] + m[13]*m[3]*m[10]
	inv[5] = m[0]*m[10]*m[15] - m[0]*m[11]*m[14] - m[8]*m[2]*m[15] + m[8]*m[3]*m[14] + m[12]*m[2]*m[11] - m[12]*m[3]*m[10]
	inv[9] = -m[0]*m[9]*m[15] + m[0]*m[11]*m[13] + m[8]*m[1]*m[15] - m[8]*m[3]*m[13] - m[12]*m[1]*m[11] + m[12]*m[3]*m[9]
	inv[13] = m[0]*m[9]*m[14] - m[0]*m[10]*m[13] - m[8]*m[1]*m[14] + m[8]*m[2]*m[13] + m[12]*m[1]*m[10] - m[12]*m[2]*m[9]
	inv[2] = m[1]*m[6]*m[15] - m[1]*m[7]*m[14] - m[5]*m[2]*m[15] + m[5]*m[3]*m[14] + m[13]*m[2]*m[7] - m[13]*m[3]*m[6]
	inv[6] = -m[0]*m[6]*m[15] + m[0]*m[7]*m[14] + m[4]*m[2]*m[15] - m[4]*m[3]*m[14] - m[12]*m[2]*m[7] + m[12]*m[3]*m[6]
	inv[10] = m[0]*m[5]*m[15] - m[0]*m[7]*m[13] - m[4]*m[1]*m[15] + m[4]*m[3]*m[13] + m[12]*m[1]*m[7] - m[12]*m[3]*m[5]
	inv[14] = -m[0]*m[5]*m[14] + m[0]*m[6]*m[13] + m[4]*m[1]*m[14] - m[4]*m[2]*m[13] - m[12]*m[1]*m[6] + m[12]*m[2]*m[5]
	inv[3] = -m[1]*m[6]*m[11] + m[1]*m[7]*m[10] + m[5]*m[2]*m[11] - m[5]*m[3]*m[10] - m[9]*m[2]*m[7] + m[9]*m[3]*m[6]
	inv[7] = m[0]*m[6]*m[11] - m[0]*m[7]*m[10] - m[4]*m[2]*m[11] + m[4]*m[3]*m[10] + m[8]*m[2]*m[7] - m[8]*m[3]*m[6]
	inv[11] = -m[0]*m[5]*m[11] + m[0]*m[7]*m[9] + m[4]*m[1]*m[11] - m[4]*m[3]*m[9] - m[8]*m[1]*m[7] + m[8]*m[3]*m[5]
	inv[15] = m[0]*m[5]*m[10] - m[0]*m[6]*m[9] - m[4]*m[1]*m[10] + m[4]*m[2]*m[9] + m[8]*m[1]*m[6] - m[8]*m[2]*m[5]

	det := m[0]*inv[0] + m[1]*inv[4] + m[2]*inv[8] + m[3]*inv[12]
	if float32(math.Abs(float64(det))) <= Epsilon {
		return Mat4{}, false
	}
	invDet := 1 / det
	for i := range inv {
		inv[i] *= invDet
	}
	return inv, true
}

func (m Mat4) IsFinite() bool {
	for _, v := range m {
		if !isFinite(v) {
			return false
		}
	}
	return true
}
