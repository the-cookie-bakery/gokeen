package pickups

import "gokeen/gameplay/players"

type Item string

const (
	Pogo     Item = "pogo"
	Gun      Item = "gun"
	BlueKey  Item = "blue_key"
	Pizza    Item = "pizza"
	Soda     Item = "soda"
	Lollipop Item = "lollipop"
	Book     Item = "book"
	Bear     Item = "bear"
)

type Config struct {
	Item   Item `json:"item"`
	Amount int  `json:"amount,omitempty"`
	Score  int  `json:"score,omitempty"`
}

func Apply(p *players.Controller, cfg Config) (string, bool) {
	score := cfg.Score
	switch cfg.Item {
	case Pogo:
		p.PogoEnabled = true
		if score == 0 {
			score = 100
		}
		p.Score += score
		return "Pogo stick", true
	case Gun:
		p.HasGun = true
		amount := cfg.Amount
		if amount <= 0 {
			amount = 20
		}
		p.Ammo += amount
		if score == 0 {
			score = 100
		}
		p.Score += score
		return "Ray gun", true
	case BlueKey:
		p.BlueKey = true
		if score == 0 {
			score = 100
		}
		p.Score += score
		return "Blue key", true
	case Pizza:
		amount := cfg.Amount
		if amount <= 0 {
			amount = 25
		}
		p.Health += amount
		if p.Health > p.MaxHealth {
			p.Health = p.MaxHealth
		}
		p.Score += score
		return "Pizza", true
	case Soda:
		amount := cfg.Amount
		if amount <= 0 {
			amount = 10
		}
		p.Ammo += amount
		p.Score += score
		return "Soda", true
	case Lollipop, Book, Bear:
		if score == 0 {
			score = 50
		}
		p.Score += score
		return "Treasure", true
	default:
		if score > 0 {
			p.Score += score
			return "Item", true
		}
		return "Unknown item", false
	}
}
