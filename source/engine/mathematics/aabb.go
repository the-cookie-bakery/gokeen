package maths

import "math"

type AABB struct {
	Min Vec3
	Max Vec3
}

func EmptyAABB() AABB {
	return AABB{
		Min: V3(float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(1))),
		Max: V3(float32(math.Inf(-1)), float32(math.Inf(-1)), float32(math.Inf(-1))),
	}
}

func NewAABB(min, max Vec3) AABB {
	return AABB{Min: min, Max: max}
}

func AABBFromPoints(points ...Vec3) AABB {
	b := EmptyAABB()
	for _, p := range points {
		b = b.ExpandByPoint(p)
	}
	return b
}

func (b AABB) IsValid() bool {
	return b.Min.X <= b.Max.X && b.Min.Y <= b.Max.Y && b.Min.Z <= b.Max.Z
}

func (b AABB) Center() Vec3 {
	return b.Min.Add(b.Max).Scale(0.5)
}

func (b AABB) Size() Vec3 {
	return b.Max.Sub(b.Min)
}

func (b AABB) Extents() Vec3 {
	return b.Size().Scale(0.5)
}

func (b AABB) Volume() float32 {
	s := b.Size()
	if s.X < 0 || s.Y < 0 || s.Z < 0 {
		return 0
	}
	return s.X * s.Y * s.Z
}

func (b AABB) SurfaceArea() float32 {
	s := b.Size()
	if s.X < 0 || s.Y < 0 || s.Z < 0 {
		return 0
	}
	return 2 * (s.X*s.Y + s.Y*s.Z + s.Z*s.X)
}

func (b AABB) ExpandByPoint(p Vec3) AABB {
	return AABB{Min: b.Min.Min(p), Max: b.Max.Max(p)}
}

func (b AABB) ExpandByAABB(o AABB) AABB {
	return AABB{Min: b.Min.Min(o.Min), Max: b.Max.Max(o.Max)}
}

func (b AABB) Expand(amount float32) AABB {
	d := V3(amount, amount, amount)
	return AABB{Min: b.Min.Sub(d), Max: b.Max.Add(d)}
}

func (b AABB) ContainsPoint(p Vec3) bool {
	return p.X >= b.Min.X && p.X <= b.Max.X &&
		p.Y >= b.Min.Y && p.Y <= b.Max.Y &&
		p.Z >= b.Min.Z && p.Z <= b.Max.Z
}

func (b AABB) Intersects(o AABB) bool {
	return b.Min.X <= o.Max.X && b.Max.X >= o.Min.X &&
		b.Min.Y <= o.Max.Y && b.Max.Y >= o.Min.Y &&
		b.Min.Z <= o.Max.Z && b.Max.Z >= o.Min.Z
}

func (b AABB) Translated(v Vec3) AABB {
	return AABB{Min: b.Min.Add(v), Max: b.Max.Add(v)}
}
