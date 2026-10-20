package maths

import "math"

type Ray struct {
	Origin    Vec3
	Direction Vec3
}

func NewRay(origin, direction Vec3) Ray {
	return Ray{Origin: origin, Direction: direction.Normalize()}
}

func (r Ray) At(t float32) Vec3 {
	return r.Origin.Add(r.Direction.Scale(t))
}

// IntersectAABB returns the nearest positive hit distance and whether the ray
// originates inside the box.
func (r Ray) IntersectAABB(b AABB) (distance float32, inside bool, ok bool) {
	tmin := float32(math.Inf(-1))
	tmax := float32(math.Inf(1))
	axes := [3]float32{r.Origin.X, r.Origin.Y, r.Origin.Z}
	dirs := [3]float32{r.Direction.X, r.Direction.Y, r.Direction.Z}
	mins := [3]float32{b.Min.X, b.Min.Y, b.Min.Z}
	maxs := [3]float32{b.Max.X, b.Max.Y, b.Max.Z}
	for i := 0; i < 3; i++ {
		if float32(math.Abs(float64(dirs[i]))) < Epsilon {
			if axes[i] < mins[i] || axes[i] > maxs[i] {
				return 0, false, false
			}
			continue
		}
		inv := 1 / dirs[i]
		t1 := (mins[i] - axes[i]) * inv
		t2 := (maxs[i] - axes[i]) * inv
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tmin {
			tmin = t1
		}
		if t2 < tmax {
			tmax = t2
		}
		if tmin > tmax {
			return 0, false, false
		}
	}
	if tmax < 0 {
		return 0, false, false
	}
	if tmin < 0 {
		return 0, true, true
	}
	return tmin, false, true
}
