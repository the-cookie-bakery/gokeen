package triggers

import maths "gokeen/engine/mathematics"

func Reached(player, trigger maths.Vec3, radius, verticalRange float32) bool {
	dx := player.X - trigger.X
	dz := player.Z - trigger.Z
	dy := player.Y - trigger.Y
	if dy < 0 {
		dy = -dy
	}
	return dx*dx+dz*dz <= radius*radius && dy <= verticalRange
}
