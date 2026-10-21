package pickups

import (
	"testing"

	maths "gokeen/engine/mathematics"
	"gokeen/gameplay/players"
)

func TestPogoAndGunPickups(t *testing.T) {
	p := players.New(maths.V3(0, 0, 0))
	_, ok := Apply(p, Config{Item: Pogo})
	if !ok || !p.PogoEnabled {
		t.Fatal("pogo not applied")
	}
	if p.Score != 100 {
		t.Fatalf("pogo score: %d", p.Score)
	}
	_, ok = Apply(p, Config{Item: Gun, Amount: 30})
	if !ok || !p.HasGun || p.Ammo != 30 {
		t.Fatalf("gun not applied: %+v", p)
	}
	if p.Score != 200 {
		t.Fatalf("gun score: %d", p.Score)
	}
}

func TestHealthAndKey(t *testing.T) {
	p := players.New(maths.V3(0, 0, 0))
	p.Health = 50
	Apply(p, Config{Item: Pizza, Amount: 25})
	if p.Health != 75 {
		t.Fatalf("health: %d", p.Health)
	}
	Apply(p, Config{Item: BlueKey})
	if !p.BlueKey {
		t.Fatal("key not applied")
	}
}

func TestUnknownItemDoesNothing(t *testing.T) {
	p := players.New(maths.V3(0, 0, 0))
	before := p.Score
	if _, ok := Apply(p, Config{Item: "nope"}); ok {
		t.Fatal("unknown item should not apply")
	}
	if p.Score != before {
		t.Fatalf("score changed: %d", p.Score)
	}
}
