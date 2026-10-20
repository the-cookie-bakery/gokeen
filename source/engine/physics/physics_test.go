package physics

import (
	"testing"

	"gokeen/engine/mathematics"
)

func floorMesh() *Mesh {
	a := maths.V3(-20, 0, -20)
	b := maths.V3(20, 0, -20)
	c := maths.V3(20, 0, 20)
	d := maths.V3(-20, 0, 20)
	return NewMesh([]Triangle{{A: a, B: c, C: b}, {A: a, B: d, C: c}})
}

func TestRaycastHitsFloor(t *testing.T) {
	m := floorMesh()
	hit, ok := m.Raycast(maths.NewRay(maths.V3(0, 5, 0), maths.V3(0, -1, 0)), 100)
	if !ok {
		t.Fatal("expected hit")
	}
	if !close(hit.Distance, 5) || !close(hit.Point.Y, 0) {
		t.Fatalf("hit: %+v", hit)
	}
	if hit.Normal.Y < 0.99 {
		t.Fatalf("normal: %+v", hit.Normal)
	}
}

func TestRaycastMiss(t *testing.T) {
	m := floorMesh()
	if _, ok := m.Raycast(maths.NewRay(maths.V3(0, 5, 0), maths.V3(1, 0, 0)), 100); ok {
		t.Fatal("unexpected hit")
	}
}

func TestOverlapSphere(t *testing.T) {
	m := floorMesh()
	tris := m.OverlapSphere(maths.V3(0, 0.2, 0), 0.5)
	if len(tris) != 2 {
		t.Fatalf("triangles: %d", len(tris))
	}
	if got := m.OverlapSphere(maths.V3(0, 5, 0), 0.5); len(got) != 0 {
		t.Fatalf("unexpected overlap: %d", len(got))
	}
}

func TestMoveCapsuleStopsOnFloor(t *testing.T) {
	m := floorMesh()
	c := NewCapsule(maths.V3(0, 2, 0), maths.V3(0, 4, 0), 0.5)
	c, collided := m.MoveCapsule(c, maths.V3(0, -3, 0))
	if !collided {
		t.Fatal("expected collision")
	}
	if c.A.Y < 0.49 || c.A.Y > 0.51 {
		t.Fatalf("capsule bottom: %v", c.A.Y)
	}
}

func TestMoveCapsuleHorizontal(t *testing.T) {
	m := floorMesh()
	c := NewCapsule(maths.V3(0, 0.5, 0), maths.V3(0, 2.5, 0), 0.5)
	c, collided := m.MoveCapsule(c, maths.V3(2, 0, 0))
	if collided {
		t.Fatal("unexpected horizontal collision")
	}
	if !close(c.A.X, 2) {
		t.Fatalf("capsule x: %v", c.A.X)
	}
}

func TestBVHNodeCount(t *testing.T) {
	m := floorMesh()
	if m.TriangleCount() != 2 || m.NodeCount() != 1 {
		t.Fatalf("tris=%d nodes=%d", m.TriangleCount(), m.NodeCount())
	}
}

func close(a, b float32) bool {
	d := a - b
	return d > -1e-4 && d < 1e-4
}
