package entities

import "testing"

type position struct{ X, Y float32 }
type velocity struct{ X, Y float32 }
type health struct{ HP int }

func TestCreateDestroyAndGeneration(t *testing.T) {
	w := NewWorld()
	id := w.Create()
	if !id.Valid() || !w.Alive(id) {
		t.Fatalf("invalid id: %s", id)
	}
	if err := w.Set(id, position{X: 1, Y: 2}); err != nil {
		t.Fatal(err)
	}
	w.Destroy(id)
	if w.Alive(id) {
		t.Fatal("destroyed entity still alive")
	}
	if _, ok := w.Get[position](id); ok {
		t.Fatal("component survived destroy")
	}
	reused := w.Create()
	if reused.Index() != id.Index() {
		t.Fatalf("expected reused index, got %d and %d", reused.Index(), id.Index())
	}
	if reused.Generation() == id.Generation() {
		t.Fatal("generation was not bumped")
	}
	if w.Alive(id) {
		t.Fatal("stale id became alive again")
	}
}

func TestStoresAndQueries(t *testing.T) {
	w := NewWorld()
	a := w.Create()
	b := w.Create()
	c := w.Create()
	_ = w.Set(a, position{X: 1})
	_ = w.Set(a, velocity{X: 10})
	_ = w.Set(b, position{X: 2})
	_ = w.Set(b, velocity{X: 20})
	_ = w.Set(b, health{HP: 5})
	_ = w.Set(c, position{X: 3})

	var seen []EntityID
	Each2(w, func(id EntityID, p *position, v *velocity) {
		seen = append(seen, id)
		p.X += v.X
	})
	if len(seen) != 2 {
		t.Fatalf("Each2 count: %d", len(seen))
	}
	pa, _ := w.Get[position](a)
	pb, _ := w.Get[position](b)
	if pa.X != 11 || pb.X != 22 {
		t.Fatalf("updated positions: %v %v", pa, pb)
	}

	count := 0
	Each3(w, func(id EntityID, p *position, v *velocity, h *health) {
		count++
	})
	if count != 1 {
		t.Fatalf("Each3 count: %d", count)
	}
}

func TestStoreDeterministicOrder(t *testing.T) {
	w := NewWorld()
	ids := []EntityID{w.Create(), w.Create(), w.Create()}
	for _, id := range ids {
		_ = w.Set(id, position{})
	}
	var got []EntityID
	Each1(w, func(id EntityID, _ *position) { got = append(got, id) })
	if len(got) != len(ids) {
		t.Fatalf("count: %d", len(got))
	}
	for i := range ids {
		if got[i] != ids[i] {
			t.Fatalf("order mismatch at %d: %s != %s", i, got[i], ids[i])
		}
	}
	_ = w.Set(ids[1], position{X: 9})
	got = got[:0]
	Each1(w, func(id EntityID, _ *position) { got = append(got, id) })
	if got[1] != ids[1] {
		t.Fatal("updating an existing component changed order")
	}
}
