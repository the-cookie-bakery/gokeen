package maths

type Transform struct {
	Position Vec3
	Rotation Quat
	Scale    Vec3
}

func IdentityTransform() Transform {
	return Transform{
		Position: Vec3{},
		Rotation: QIdentity(),
		Scale:    V3(1, 1, 1),
	}
}

func (t Transform) Matrix() Mat4 {
	if t.Rotation.LengthSquared() <= Epsilon {
		t.Rotation = QIdentity()
	}
	return M4Translate(t.Position).Mul(t.Rotation.Mat4()).Mul(M4Scale(t.Scale))
}

func (t Transform) TransformPoint(v Vec3) Vec3 {
	return t.Matrix().TransformPoint(v)
}

func (t Transform) TransformVector(v Vec3) Vec3 {
	return t.Rotation.Rotate(v.Mul(t.Scale))
}

func (t Transform) Inverse() Transform {
	invScale := V3(1, 1, 1)
	if t.Scale.X != 0 {
		invScale.X = 1 / t.Scale.X
	}
	if t.Scale.Y != 0 {
		invScale.Y = 1 / t.Scale.Y
	}
	if t.Scale.Z != 0 {
		invScale.Z = 1 / t.Scale.Z
	}
	invRot := t.Rotation.Conjugate().Normalize()
	invPos := invRot.Rotate(t.Position.Neg().Mul(invScale))
	return Transform{Position: invPos, Rotation: invRot, Scale: invScale}
}

func (t Transform) IsFinite() bool {
	return t.Position.IsFinite() && t.Rotation.IsFinite() && t.Scale.IsFinite()
}

func (t Transform) Lerp(o Transform, alpha float32) Transform {
	return Transform{
		Position: t.Position.Lerp(o.Position, alpha),
		Rotation: t.Rotation.Slerp(o.Rotation, alpha),
		Scale:    t.Scale.Lerp(o.Scale, alpha),
	}
}
