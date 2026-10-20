package physics

import (
	"math"

	"gokeen/engine/mathematics"
)

type Capsule struct {
	A      maths.Vec3
	B      maths.Vec3
	Radius float32
}

func NewCapsule(bottom, top maths.Vec3, radius float32) Capsule {
	return Capsule{A: bottom, B: top, Radius: radius}
}

func (c Capsule) Bounds() maths.AABB {
	r := maths.V3(c.Radius, c.Radius, c.Radius)
	return maths.NewAABB(c.A.Min(c.B).Sub(r), c.A.Max(c.B).Add(r))
}

func (c Capsule) Translated(v maths.Vec3) Capsule {
	return Capsule{A: c.A.Add(v), B: c.B.Add(v), Radius: c.Radius}
}

func (c Capsule) Height() float32 {
	return c.A.Distance(c.B)
}

func (c Capsule) Center() maths.Vec3 {
	return c.A.Add(c.B).Scale(0.5)
}

func (c Capsule) Bottom() maths.Vec3 {
	return c.A
}

func (c Capsule) Top() maths.Vec3 {
	return c.B
}

type Contact struct {
	Point         maths.Vec3
	Normal        maths.Vec3
	Depth         float32
	TriangleIndex int
}

func (m *Mesh) CapsuleContacts(c Capsule) []Contact {
	if m == nil || m.root < 0 {
		return nil
	}
	query := c.Bounds()
	var out []Contact
	m.capsuleNode(m.root, query, c, &out)
	return out
}

func (m *Mesh) capsuleNode(nodeIndex int, query maths.AABB, c Capsule, out *[]Contact) {
	node := &m.nodes[nodeIndex]
	if !node.bounds.Intersects(query) {
		return
	}
	if node.count > 0 {
		for i := 0; i < node.count; i++ {
			idx := m.order[node.start+i]
			tri := m.Triangles[idx]
			bestDepth := float32(0)
			var best Contact
			samples := capsuleSamples(c)
			for _, p := range samples {
				q, dist := pointTriangleDistance(p, tri.A, tri.B, tri.C)
				if dist >= c.Radius {
					continue
				}
				depth := c.Radius - dist
				if depth <= bestDepth {
					continue
				}
				normal := p.Sub(q)
				if dist > 1e-6 {
					normal = normal.Scale(1 / dist)
				} else {
					normal = tri.Normal()
					if normal.Dot(p.Sub(tri.A)) < 0 {
						normal = normal.Neg()
					}
				}
				bestDepth = depth
				best = Contact{Point: q, Normal: normal, Depth: depth, TriangleIndex: idx}
			}
			if bestDepth > 0 {
				*out = append(*out, best)
			}
		}
		return
	}
	if node.left >= 0 {
		m.capsuleNode(node.left, query, c, out)
	}
	if node.right >= 0 {
		m.capsuleNode(node.right, query, c, out)
	}
}

func capsuleSamples(c Capsule) [5]maths.Vec3 {
	var out [5]maths.Vec3
	for i := 0; i < 5; i++ {
		t := float32(i) / 4
		out[i] = c.A.Lerp(c.B, t)
	}
	return out
}

// MoveCapsule moves a capsule and resolves penetrations against the mesh. It is
// intentionally conservative for the platformer: small substeps plus iterative
// depenetration. Velocity response is handled by the player controller.
func (m *Mesh) MoveCapsule(c Capsule, delta maths.Vec3) (Capsule, bool) {
	length := delta.Length()
	if length <= maths.Epsilon {
		return c, false
	}
	stepSize := c.Radius * 0.5
	if stepSize < 0.02 {
		stepSize = 0.02
	}
	steps := int(math.Ceil(float64(length / stepSize)))
	if steps < 1 {
		steps = 1
	}
	if steps > 64 {
		steps = 64
	}
	step := delta.Scale(1 / float32(steps))
	collided := false
	for i := 0; i < steps; i++ {
		c = c.Translated(step)
		for iter := 0; iter < 4; iter++ {
			contacts := m.CapsuleContacts(c)
			if len(contacts) == 0 {
				break
			}
			collided = true
			var push maths.Vec3
			for _, contact := range contacts {
				push = push.Add(contact.Normal.Scale(contact.Depth))
			}
			if push.LengthSquared() <= maths.Epsilon {
				break
			}
			c = c.Translated(push)
		}
	}
	return c, collided
}
