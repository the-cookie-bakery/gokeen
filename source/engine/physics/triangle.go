package physics

import (
	"math"

	"gokeen/engine/mathematics"
)

type Triangle struct {
	A, B, C maths.Vec3
}

func (t Triangle) Bounds() maths.AABB {
	return maths.AABBFromPoints(t.A, t.B, t.C)
}

func (t Triangle) Normal() maths.Vec3 {
	return t.B.Sub(t.A).Cross(t.C.Sub(t.A)).Normalize()
}

func (t Triangle) Centroid() maths.Vec3 {
	return t.A.Add(t.B).Add(t.C).Scale(1.0 / 3.0)
}

func closestPointOnTriangle(p, a, b, c maths.Vec3) maths.Vec3 {
	ab := b.Sub(a)
	ac := c.Sub(a)
	ap := p.Sub(a)
	d1 := ab.Dot(ap)
	d2 := ac.Dot(ap)
	if d1 <= 0 && d2 <= 0 {
		return a
	}
	bp := p.Sub(b)
	d3 := ab.Dot(bp)
	d4 := ac.Dot(bp)
	if d3 >= 0 && d4 <= d3 {
		return b
	}
	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		v := d1 / (d1 - d3)
		return a.Add(ab.Scale(v))
	}
	cp := p.Sub(c)
	d5 := ab.Dot(cp)
	d6 := ac.Dot(cp)
	if d6 >= 0 && d5 <= d6 {
		return c
	}
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		w := d2 / (d2 - d6)
		return a.Add(ac.Scale(w))
	}
	va := d3*d6 - d5*d4
	if va <= 0 && (d4-d3) >= 0 && (d5-d6) >= 0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return b.Add(c.Sub(b).Scale(w))
	}
	denom := 1 / (va + vb + vc)
	v := vb * denom
	w := vc * denom
	return a.Add(ab.Scale(v)).Add(ac.Scale(w))
}

func pointTriangleDistance(p, a, b, c maths.Vec3) (maths.Vec3, float32) {
	q := closestPointOnTriangle(p, a, b, c)
	d := p.Sub(q)
	return q, d.Length()
}

func triangleIntersect(ray maths.Ray, tri Triangle, maxDist float32) (t float32, ok bool) {
	const eps = 1e-6
	e1 := tri.B.Sub(tri.A)
	e2 := tri.C.Sub(tri.A)
	pvec := ray.Direction.Cross(e2)
	det := e1.Dot(pvec)
	if math.Abs(float64(det)) < eps {
		return 0, false
	}
	invDet := 1 / det
	tvec := ray.Origin.Sub(tri.A)
	u := tvec.Dot(pvec) * invDet
	if u < 0 || u > 1 {
		return 0, false
	}
	qvec := tvec.Cross(e1)
	v := ray.Direction.Dot(qvec) * invDet
	if v < 0 || u+v > 1 {
		return 0, false
	}
	distance := e2.Dot(qvec) * invDet
	if distance < 0 || distance > maxDist {
		return 0, false
	}
	return distance, true
}
