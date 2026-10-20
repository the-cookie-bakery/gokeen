package entities

import (
	"fmt"
	"reflect"
)

type World struct {
	generations []uint32
	alive       []bool
	free        []uint32
	nextIndex   uint32
	stores      map[reflect.Type]componentStore
}

func NewWorld() *World {
	return &World{stores: make(map[reflect.Type]componentStore)}
}

func (w *World) Create() EntityID {
	if n := len(w.free); n > 0 {
		index := w.free[n-1]
		w.free = w.free[:n-1]
		w.alive[index] = true
		w.generations[index]++
		if w.generations[index] == 0 {
			w.generations[index] = 1
		}
		return makeEntityID(index, w.generations[index])
	}
	index := w.nextIndex
	w.nextIndex++
	w.generations = append(w.generations, 1)
	w.alive = append(w.alive, true)
	return makeEntityID(index, 1)
}

func (w *World) Alive(id EntityID) bool {
	if !id.Valid() {
		return false
	}
	index := id.Index()
	if int(index) >= len(w.alive) || !w.alive[index] {
		return false
	}
	return w.generations[index] == id.Generation()
}

func (w *World) Destroy(id EntityID) {
	if !w.Alive(id) {
		return
	}
	for _, store := range w.stores {
		store.delete(id)
	}
	index := id.Index()
	w.alive[index] = false
	w.free = append(w.free, index)
}

func (w *World) Store[T any]() *Store[T] {
	key := reflect.TypeOf((*T)(nil)).Elem()
	if store, ok := w.stores[key]; ok {
		return store.(*Store[T])
	}
	store := NewStore[T]()
	w.stores[key] = store
	return store
}

func (w *World) Set[T any](id EntityID, value T) error {
	if !w.Alive(id) {
		return fmt.Errorf("set component on dead entity %s", id)
	}
	w.Store[T]().Set(id, value)
	return nil
}

func (w *World) Get[T any](id EntityID) (*T, bool) {
	if !w.Alive(id) {
		return nil, false
	}
	return w.Store[T]().Get(id)
}

func (w *World) Remove[T any](id EntityID) {
	w.Store[T]().Delete(id)
}

func (w *World) Has[T any](id EntityID) bool {
	if !w.Alive(id) {
		return false
	}
	return w.Store[T]().has(id)
}

func (w *World) Count() int {
	n := 0
	for _, alive := range w.alive {
		if alive {
			n++
		}
	}
	return n
}

func (w *World) Entities() []EntityID {
	out := make([]EntityID, 0, w.Count())
	for index, alive := range w.alive {
		if alive {
			out = append(out, makeEntityID(uint32(index), w.generations[index]))
		}
	}
	return out
}

func Each1[T any](w *World, fn func(EntityID, *T)) {
	w.Store[T]().Each(fn)
}

func Each2[A, B any](w *World, fn func(EntityID, *A, *B)) {
	a := w.Store[A]()
	b := w.Store[B]()
	if a.Len() == 0 || b.Len() == 0 {
		return
	}
	if a.Len() <= b.Len() {
		a.Each(func(id EntityID, av *A) {
			if bv, ok := b.Get(id); ok {
				fn(id, av, bv)
			}
		})
		return
	}
	b.Each(func(id EntityID, bv *B) {
		if av, ok := a.Get(id); ok {
			fn(id, av, bv)
		}
	})
}

func Each3[A, B, C any](w *World, fn func(EntityID, *A, *B, *C)) {
	a := w.Store[A]()
	b := w.Store[B]()
	c := w.Store[C]()
	if a.Len() == 0 || b.Len() == 0 || c.Len() == 0 {
		return
	}
	a.Each(func(id EntityID, av *A) {
		bv, ok := b.Get(id)
		if !ok {
			return
		}
		cv, ok := c.Get(id)
		if !ok {
			return
		}
		fn(id, av, bv, cv)
	})
}

func typeName[T any]() string {
	return reflect.TypeOf((*T)(nil)).Elem().String()
}
