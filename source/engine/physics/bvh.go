package physics

import (
	"sort"

	"gokeen/engine/mathematics"
)

const leafSize = 4

type bvhNode struct {
	bounds maths.AABB
	left   int
	right  int
	start  int
	count  int
}

type Mesh struct {
	Triangles []Triangle
	nodes     []bvhNode
	order     []int
	root      int
}

type Hit struct {
	Distance      float32
	Point         maths.Vec3
	Normal        maths.Vec3
	TriangleIndex int
}

func NewMesh(triangles []Triangle) *Mesh {
	m := &Mesh{
		Triangles: append([]Triangle(nil), triangles...),
		order:     make([]int, len(triangles)),
	}
	for i := range m.order {
		m.order[i] = i
	}
	if len(m.Triangles) > 0 {
		m.root = m.build(0, len(m.order))
	} else {
		m.root = -1
	}
	return m
}

func (m *Mesh) TriangleCount() int {
	return len(m.Triangles)
}

func (m *Mesh) NodeCount() int {
	return len(m.nodes)
}

func (m *Mesh) build(start, end int) int {
	nodeIndex := len(m.nodes)
	m.nodes = append(m.nodes, bvhNode{start: start, count: end - start, left: -1, right: -1})
	node := &m.nodes[nodeIndex]
	node.bounds = maths.EmptyAABB()
	for i := start; i < end; i++ {
		node.bounds = node.bounds.ExpandByAABB(m.Triangles[m.order[i]].Bounds())
	}
	if end-start <= leafSize {
		return nodeIndex
	}
	extent := node.bounds.Size()
	axis := 0
	if extent.Y > extent.X && extent.Y >= extent.Z {
		axis = 1
	} else if extent.Z > extent.X && extent.Z >= extent.Y {
		axis = 2
	}
	mid := (start + end) / 2
	sort.Slice(m.order[start:end], func(i, j int) bool {
		a := m.Triangles[m.order[start+i]].Centroid()
		b := m.Triangles[m.order[start+j]].Centroid()
		return component(a, axis) < component(b, axis)
	})
	left := m.build(start, mid)
	right := m.build(mid, end)
	m.nodes[nodeIndex].left = left
	m.nodes[nodeIndex].right = right
	m.nodes[nodeIndex].count = 0
	return nodeIndex
}

func component(v maths.Vec3, axis int) float32 {
	switch axis {
	case 1:
		return v.Y
	case 2:
		return v.Z
	default:
		return v.X
	}
}

func (m *Mesh) Raycast(ray maths.Ray, maxDist float32) (Hit, bool) {
	if m == nil || m.root < 0 {
		return Hit{}, false
	}
	var best Hit
	best.Distance = maxDist
	found := false
	m.raycastNode(m.root, ray, maxDist, &best, &found)
	return best, found
}

func (m *Mesh) raycastNode(nodeIndex int, ray maths.Ray, maxDist float32, best *Hit, found *bool) {
	node := &m.nodes[nodeIndex]
	dist, _, ok := ray.IntersectAABB(node.bounds)
	if !ok || dist > maxDist {
		return
	}
	if node.count > 0 {
		for i := 0; i < node.count; i++ {
			idx := m.order[node.start+i]
			tri := m.Triangles[idx]
			t, hit := triangleIntersect(ray, tri, best.Distance)
			if hit {
				best.Distance = t
				best.Point = ray.At(t)
				best.Normal = tri.Normal()
				best.TriangleIndex = idx
				*found = true
			}
		}
		return
	}
	if node.left >= 0 {
		m.raycastNode(node.left, ray, best.Distance, best, found)
	}
	if node.right >= 0 {
		m.raycastNode(node.right, ray, best.Distance, best, found)
	}
}

func (m *Mesh) OverlapSphere(center maths.Vec3, radius float32) []Triangle {
	if m == nil || m.root < 0 {
		return nil
	}
	var out []Triangle
	query := maths.AABB{Min: center.Sub(maths.V3(radius, radius, radius)), Max: center.Add(maths.V3(radius, radius, radius))}
	m.overlapNode(m.root, query, center, radius, &out)
	return out
}

func (m *Mesh) overlapNode(nodeIndex int, query maths.AABB, center maths.Vec3, radius float32, out *[]Triangle) {
	node := &m.nodes[nodeIndex]
	if !node.bounds.Intersects(query) {
		return
	}
	if node.count > 0 {
		for i := 0; i < node.count; i++ {
			tri := m.Triangles[m.order[node.start+i]]
			if _, d := pointTriangleDistance(center, tri.A, tri.B, tri.C); d <= radius {
				*out = append(*out, tri)
			}
		}
		return
	}
	if node.left >= 0 {
		m.overlapNode(node.left, query, center, radius, out)
	}
	if node.right >= 0 {
		m.overlapNode(node.right, query, center, radius, out)
	}
}
