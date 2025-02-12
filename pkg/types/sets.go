package types

type Set[T comparable] struct {
	items map[T]struct{}
}

func (s *Set[T]) Has(item T) bool {
	_, ok := s.items[item]
	return ok
}

func (s *Set[T]) Add(item T) {
	s.items[item] = struct{}{}
}

func (s *Set[T]) Remove(item T) {
	delete(s.items, item)
}

func NewSet[T comparable](items ...T) *Set[T] {
	s := &Set[T]{items: make(map[T]struct{})}
	for i := range items {
		s.items[items[i]] = struct{}{}
	}

	return s
}
