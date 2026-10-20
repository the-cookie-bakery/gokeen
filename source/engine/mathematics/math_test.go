package maths

import (
	"math"
	"testing"
)

func close32(a, b float32) bool {
	return float32(math.Abs(float64(a-b))) < 1e-4
}

func closeVec3(a, b Vec3) bool {
	return close32(a.X, b.X) && close32(a.Y, b.Y) && close32(a.Z, b.Z)
}

func closeMat4(a, b Mat4) bool {
	for i := range a {
		if !close32(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestVec3Basics(t *testing.T) {
	a := V3(1, 2, 3)
	b := V3(4, 5, 6)
	if got := a.Add(b); got != V3(5, 7, 9) {
		t.Fatalf("Add: %+v", got)
	}
	if got := a.Dot(b); got != 32 {
		t.Fatalf("Dot: %v", got)
	}
	if got := V3(1, 0, 0).Cross(V3(0, 1, 0)); got != V3(0, 0, 1) {
		t.Fatalf("Cross: %+v", got)
	}
	if got := V3(3, 4, 0).Length(); !close32(got, 5) {
		t.Fatalf("Length: %v", got)
	}
	if got := V3(3, 4, 0).Normalize(); !closeVec3(got, V3(0.6, 0.8, 0)) {
		t.Fatalf("Normalize: %+v", got)
	}
}

func TestQuatRotations(t *testing.T) {
	q := QFromAxisAngle(V3(0, 1, 0), HalfPi)
	got := q.Rotate(V3(1, 0, 0))
	if !closeVec3(got, V3(0, 0, -1)) {
		t.Fatalf("yaw rotation: %+v", got)
	}
	inv := q.Conjugate()
	if !closeVec3(inv.Rotate(got), V3(1, 0, 0)) {
		t.Fatalf("inverse rotation failed")
	}
	euler := QFromEulerDegrees(0, 90, 0)
	if !closeVec3(euler.Rotate(V3(1, 0, 0)), V3(0, 0, -1)) {
		t.Fatalf("euler rotation: %+v", euler.Rotate(V3(1, 0, 0)))
	}
}

func TestQuatSlerp(t *testing.T) {
	a := QIdentity()
	b := QFromAxisAngle(V3(0, 1, 0), Pi)
	m := a.Slerp(b, 0.5)
	if !closeVec3(m.Rotate(V3(1, 0, 0)), V3(0, 0, -1)) {
		t.Fatalf("slerp midpoint: %+v", m.Rotate(V3(1, 0, 0)))
	}
}

func TestMat4ComposeAndTransform(t *testing.T) {
	m := M4Translate(V3(10, 0, 0)).Mul(M4Rotation(QFromAxisAngle(V3(0, 1, 0), HalfPi))).Mul(M4Scale(V3(2, 2, 2)))
	got := m.TransformPoint(V3(1, 0, 0))
	if !closeVec3(got, V3(10, 0, -2)) {
		t.Fatalf("point: %+v", got)
	}
	vec := m.TransformVector(V3(1, 0, 0))
	if !closeVec3(vec, V3(0, 0, -2)) {
		t.Fatalf("vector: %+v", vec)
	}
}

func TestMat4Inverse(t *testing.T) {
	m := M4Translate(V3(3, -2, 5)).Mul(M4Rotation(QFromAxisAngle(V3(0, 1, 0), 0.7))).Mul(M4Scale(V3(2, 3, 4)))
	inv, ok := m.Inverse()
	if !ok {
		t.Fatal("inverse not found")
	}
	p := V3(1.5, -0.25, 2)
	round := inv.TransformPoint(m.TransformPoint(p))
	if !closeVec3(round, p) {
		t.Fatalf("round trip: %+v -> %+v", p, round)
	}
	if !closeMat4(m.Mul(inv), M4Identity()) {
		t.Fatalf("identity mismatch:\n%v\n%v", m.Mul(inv), M4Identity())
	}
}

func TestPerspectiveDepthRange(t *testing.T) {
	m := PerspectiveRH(HalfPi, 16.0/9.0, 0.1, 100)
	nearPoint := m.TransformPoint(V3(0, 1, -0.1))
	farPoint := m.TransformPoint(V3(0, 0, -100))
	if !close32(nearPoint.Z, 0) {
		t.Fatalf("near depth: %v", nearPoint.Z)
	}
	if !close32(farPoint.Z, 1) {
		t.Fatalf("far depth: %v", farPoint.Z)
	}
	if !(nearPoint.Y < 0) {
		// Vulkan NDC has Y down; a point above the camera should project down.
		t.Fatalf("expected Y flip for Vulkan, got %v", nearPoint.Y)
	}
}

func TestLookAtRH(t *testing.T) {
	view := LookAtRH(V3(0, 0, 5), V3(0, 0, 0), V3(0, 1, 0))
	got := view.TransformPoint(V3(0, 0, 0))
	if !closeVec3(got, V3(0, 0, -5)) {
		t.Fatalf("view transform: %+v", got)
	}
	right := view.TransformVector(V3(1, 0, 0))
	if !closeVec3(right, V3(1, 0, 0)) {
		t.Fatalf("right vector: %+v", right)
	}
}

func TestAABBAndRay(t *testing.T) {
	b := NewAABB(V3(-1, -1, -1), V3(1, 1, 1))
	if !b.ContainsPoint(V3(0, 0, 0)) {
		t.Fatal("contains failed")
	}
	r := NewRay(V3(0, 0, 5), V3(0, 0, -1))
	dist, inside, ok := r.IntersectAABB(b)
	if !ok || inside || !close32(dist, 4) {
		t.Fatalf("ray hit: dist=%v inside=%v ok=%v", dist, inside, ok)
	}
	r = NewRay(V3(0, 0, 0), V3(0, 0, -1))
	_, inside, ok = r.IntersectAABB(b)
	if !ok || !inside {
		t.Fatal("expected inside hit")
	}
	r = NewRay(V3(0, 2, 0), V3(0, 1, 0))
	if _, _, ok := r.IntersectAABB(b); ok {
		t.Fatal("unexpected hit")
	}
}

func TestTransformMatrix(t *testing.T) {
	tr := Transform{
		Position: V3(2, 3, 4),
		Rotation: QFromAxisAngle(V3(0, 1, 0), HalfPi),
		Scale:    V3(1, 1, 1),
	}
	got := tr.TransformPoint(V3(1, 0, 0))
	if !closeVec3(got, V3(2, 3, 3)) {
		t.Fatalf("transform point: %+v", got)
	}
}
