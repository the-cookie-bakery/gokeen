package physics

import "gokeen/engine/mathematics"

func TrianglesFromIndexed(positions []maths.Vec3, indices []uint32) []Triangle {
	out := make([]Triangle, 0, len(indices)/3)
	for i := 0; i+2 < len(indices); i += 3 {
		a := int(indices[i])
		b := int(indices[i+1])
		c := int(indices[i+2])
		if a < 0 || b < 0 || c < 0 || a >= len(positions) || b >= len(positions) || c >= len(positions) {
			continue
		}
		out = append(out, Triangle{A: positions[a], B: positions[b], C: positions[c]})
	}
	return out
}
