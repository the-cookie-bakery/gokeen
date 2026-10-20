package entities

type componentStore interface {
	componentType() string
	delete(id EntityID)
	has(id EntityID) bool
	count() int
	ids() []EntityID
}

type Store[T any] struct {
	values map[EntityID]*T
	order  []EntityID
}

func NewStore[T any]() *Store[T] {
	return &Store[T]{values: make(map[EntityID]*T)}
}

func (s *Store[T]) Set(id EntityID, value T) {
	if s.values == nil {
		s.values = make(map[EntityID]*T)
	}
	if existing, ok := s.values[id]; ok {
		*existing = value
		return
	}
	v := new(T)
	*v = value
	s.values[id] = v
	s.order = append(s.order, id)
}

func (s *Store[T]) Get(id EntityID) (*T, bool) {
	if s == nil {
		return nil, false
	}
	v, ok := s.values[id]
	return v, ok
}

func (s *Store[T]) Delete(id EntityID) {
	s.delete(id)
}

func (s *Store[T]) delete(id EntityID) {
	if s == nil {
		return
	}
	if _, ok := s.values[id]; !ok {
		return
	}
	delete(s.values, id)
	for i, candidate := range s.order {
		if candidate == id {
			s.order[i] = s.order[len(s.order)-1]
			s.order = s.order[:len(s.order)-1]
			break
		}
	}
}

func (s *Store[T]) has(id EntityID) bool {
	if s == nil {
		return false
	}
	_, ok := s.values[id]
	return ok
}

func (s *Store[T]) Len() int {
	if s == nil {
		return 0
	}
	return len(s.values)
}

func (s *Store[T]) count() int {
	return s.Len()
}

func (s *Store[T]) ids() []EntityID {
	if s == nil {
		return nil
	}
	return append([]EntityID(nil), s.order...)
}

func (s *Store[T]) Each(fn func(EntityID, *T)) {
	if s == nil {
		return
	}
	for _, id := range s.order {
		v, ok := s.values[id]
		if ok {
			fn(id, v)
		}
	}
}

func (s *Store[T]) Values() []*T {
	if s == nil {
		return nil
	}
	out := make([]*T, 0, len(s.order))
	for _, id := range s.order {
		if v, ok := s.values[id]; ok {
			out = append(out, v)
		}
	}
	return out
}

func (s *Store[T]) componentType() string {
	return typeName[T]()
}
