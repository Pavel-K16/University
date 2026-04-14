package inmemory

type DecrementStore struct {
	decrements map[int][]DecrAeroKoefStore
}

type DecrAeroKoefStore struct {
	koef1     float64
	koef2     float64
	decrement float64
}

func NewDecrementStore() *DecrementStore {
	return &DecrementStore{decrements: make(map[int][]DecrAeroKoefStore)}
}

func (s *DecrementStore) AddDecrement(nodeID int, koef1, koef2 float64, decrement float64) {
	s.decrements[nodeID] = append(s.decrements[nodeID], DecrAeroKoefStore{koef1: koef1, koef2: koef2, decrement: decrement})
}

func (s *DecrementStore) GetDecrements(nodeID int) []DecrAeroKoefStore {
	return s.decrements[nodeID]
}
