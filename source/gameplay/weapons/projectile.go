package weapons

import maths "gokeen/engine/mathematics"

type Projectile struct {
	Position maths.Vec3
	Velocity maths.Vec3
	Life     float32
	Active   bool
}

func Spawn(origin, direction maths.Vec3, speed, life float32) *Projectile {
	return &Projectile{
		Position: origin,
		Velocity: direction.Normalize().Scale(speed),
		Life:     life,
		Active:   true,
	}
}

func (p *Projectile) Advance(dt float32) (previous, current maths.Vec3) {
	previous = p.Position
	p.Position = p.Position.Add(p.Velocity.Scale(dt))
	p.Life -= dt
	if p.Life <= 0 {
		p.Active = false
	}
	return previous, p.Position
}

func (p *Projectile) Deactivate() {
	p.Active = false
	p.Life = 0
}
