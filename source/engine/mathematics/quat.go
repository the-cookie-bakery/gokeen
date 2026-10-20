package maths

import "math"

type Quat struct {
	W, X, Y, Z float32
}

func QIdentity() Quat {
	return Quat{W: 1}
}

func QFromAxisAngle(axis Vec3, angle float32) Quat {
	axis = axis.Normalize()
	if axis.LengthSquared() <= Epsilon {
		return QIdentity()
	}
	s := float32(math.Sin(float64(angle) / 2))
	return Quat{
		W: float32(math.Cos(float64(angle) / 2)),
		X: axis.X * s,
		Y: axis.Y * s,
		Z: axis.Z * s,
	}
}

// QFromEuler applies rotations around X, then Y, then Z.
func QFromEuler(pitch, yaw, roll float32) Quat {
	qx := QFromAxisAngle(V3(1, 0, 0), pitch)
	qy := QFromAxisAngle(V3(0, 1, 0), yaw)
	qz := QFromAxisAngle(V3(0, 0, 1), roll)
	return qz.Mul(qy).Mul(qx)
}

func QFromEulerDegrees(pitch, yaw, roll float32) Quat {
	return QFromEuler(pitch*DegToRad, yaw*DegToRad, roll*DegToRad)
}

func (q Quat) Mul(o Quat) Quat {
	return Quat{
		W: q.W*o.W - q.X*o.X - q.Y*o.Y - q.Z*o.Z,
		X: q.W*o.X + q.X*o.W + q.Y*o.Z - q.Z*o.Y,
		Y: q.W*o.Y - q.X*o.Z + q.Y*o.W + q.Z*o.X,
		Z: q.W*o.Z + q.X*o.Y - q.Y*o.X + q.Z*o.W,
	}
}

func (q Quat) Conjugate() Quat {
	return Quat{W: q.W, X: -q.X, Y: -q.Y, Z: -q.Z}
}

func (q Quat) LengthSquared() float32 {
	return q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z
}

func (q Quat) Length() float32 {
	return float32(math.Sqrt(float64(q.LengthSquared())))
}

func (q Quat) Normalize() Quat {
	l := q.Length()
	if l <= Epsilon {
		return QIdentity()
	}
	return Quat{q.W / l, q.X / l, q.Y / l, q.Z / l}
}

func (q Quat) Rotate(v Vec3) Vec3 {
	u := Vec3{q.X, q.Y, q.Z}
	t := u.Cross(v).Scale(2)
	return v.Add(t.Scale(q.W)).Add(u.Cross(t))
}

func (q Quat) Slerp(o Quat, t float32) Quat {
	a := q.Normalize()
	b := o.Normalize()
	dot := a.W*b.W + a.X*b.X + a.Y*b.Y + a.Z*b.Z
	if dot < -Epsilon {
		b = Quat{-b.W, -b.X, -b.Y, -b.Z}
		dot = -dot
	}
	if dot > 0.9995 {
		return Quat{
			a.W + (b.W-a.W)*t,
			a.X + (b.X-a.X)*t,
			a.Y + (b.Y-a.Y)*t,
			a.Z + (b.Z-a.Z)*t,
		}.Normalize()
	}
	theta0 := float32(math.Acos(float64(dot)))
	theta := theta0 * t
	sin0 := float32(math.Sin(float64(theta0)))
	sinT := float32(math.Sin(float64(theta)))
	s0 := float32(math.Cos(float64(theta))) - dot*sinT/sin0
	s1 := sinT / sin0
	return Quat{
		a.W*s0 + b.W*s1,
		a.X*s0 + b.X*s1,
		a.Y*s0 + b.Y*s1,
		a.Z*s0 + b.Z*s1,
	}.Normalize()
}

func (q Quat) Mat4() Mat4 {
	w, x, y, z := q.W, q.X, q.Y, q.Z
	return Mat4{
		1 - 2*(y*y+z*z), 2 * (x*y + w*z), 2 * (x*z - w*y), 0,
		2 * (x*y - w*z), 1 - 2*(x*x+z*z), 2 * (y*z + w*x), 0,
		2 * (x*z + w*y), 2 * (y*z - w*x), 1 - 2*(x*x+y*y), 0,
		0, 0, 0, 1,
	}
}

func (q Quat) IsFinite() bool {
	return isFinite(q.W) && isFinite(q.X) && isFinite(q.Y) && isFinite(q.Z)
}
