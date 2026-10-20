package components

import (
	"gokeen/assets"
	"gokeen/engine/animations"
	"gokeen/engine/mathematics"
	"gokeen/entities"
)

type Transform struct {
	Value maths.Transform
}

type Velocity struct {
	Linear maths.Vec3
}

type Model struct {
	Mesh       assets.MeshID
	Visible    bool
	CastShadow bool
}

type Collider struct {
	Radius float32
	Height float32
	Offset maths.Vec3
	Static bool
	Layer  uint32
	Mask   uint32
}

type Health struct {
	Current      int
	Max          int
	Invulnerable float32
}

func (h *Health) Damage(amount int) bool {
	if h.Invulnerable > 0 || amount <= 0 {
		return false
	}
	h.Current -= amount
	if h.Current < 0 {
		h.Current = 0
	}
	return h.Current == 0
}

func (h *Health) Heal(amount int) {
	h.Current += amount
	if h.Current > h.Max {
		h.Current = h.Max
	}
}

type Animation struct {
	Skeleton animations.SkeletonID
	Clips    map[string]animations.ClipID
	Current  string
	Time     float32
	Speed    float32
	Loop     bool
}

func (a *Animation) Play(name string) {
	if a.Clips == nil {
		return
	}
	if _, ok := a.Clips[name]; !ok {
		return
	}
	if a.Current == name {
		return
	}
	a.Current = name
	a.Time = 0
}

type Player struct {
	MoveSpeed    float32
	Acceleration float32
	JumpSpeed    float32
	AirControl   float32
	Grounded     bool
	PogoEnabled  bool
	PogoSpeed    float32
	Health       int
	Lives        int
	Ammo         int
	Score        int
	BlueKey      bool
	Dead         bool
	RespawnAt    maths.Vec3
}

type Weapon struct {
	Ammo         int
	Cooldown     float32
	CooldownTime float32
	Damage       int
	StunTime     float32
	MuzzleOffset maths.Vec3
}

func (w *Weapon) CanFire() bool {
	return w.Cooldown <= 0 && w.Ammo > 0
}

func (w *Weapon) Fire() bool {
	if !w.CanFire() {
		return false
	}
	w.Ammo--
	w.Cooldown = w.CooldownTime
	return true
}

func (w *Weapon) Update(dt float32) {
	if w.Cooldown > 0 {
		w.Cooldown -= dt
		if w.Cooldown < 0 {
			w.Cooldown = 0
		}
	}
}

type EnemyType string

const (
	EnemyYorp   EnemyType = "yorp"
	EnemyGarg   EnemyType = "garg"
	EnemyPatPat EnemyType = "patpat"
)

type EnemyState string

const (
	EnemyIdle    EnemyState = "idle"
	EnemyPatrol  EnemyState = "patrol"
	EnemyChase   EnemyState = "chase"
	EnemyStunned EnemyState = "stunned"
	EnemyDead    EnemyState = "dead"
)

type Enemy struct {
	Type       EnemyType
	State      EnemyState
	Speed      float32
	ChaseRange float32
	Target     entities.EntityID
	StunTime   float32
	Home       maths.Vec3
}

type PickupKind string

const (
	PickupPogo     PickupKind = "pogo"
	PickupPogoSign PickupKind = "pogo_sign"
	PickupTeddy    PickupKind = "teddy"
	PickupLollipop PickupKind = "lollipop"
	PickupPizza    PickupKind = "pizza"
	PickupSoda     PickupKind = "soda"
	PickupGun      PickupKind = "gun"
	PickupKey      PickupKind = "blue_key"
	PickupBook     PickupKind = "book"
	PickupBear     PickupKind = "bear"
)

type Pickup struct {
	Kind      PickupKind
	Amount    int
	BobHeight float32
	SpinSpeed float32
	Respawn   bool
	Collected bool
}

type Trigger struct {
	Type       string
	Target     string
	Properties map[string]string
}

type Camera struct {
	Distance    float32
	Height      float32
	Pitch       float32
	Yaw         float32
	Sensitivity float32
	MinPitch    float32
	MaxPitch    float32
}

type Lifetime struct {
	Remaining float32
}

type Projectile struct {
	Velocity maths.Vec3
	Damage   int
	StunTime float32
	Owner    entities.EntityID
}
