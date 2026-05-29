package inmemory

import (
	"fmt"
	"math"
	"os"
	"sync"
)

const (
	decrementStoreFileTmpl = "../wolfram/paramsAndPoints/decrementStore%d.txt"
)

type DecrementStore struct {
	mu         sync.Mutex
	decrements map[int][]DecrAeroKoefStore
}

type DecrAeroKoefStore struct {
	koef1     float64
	koef2     float64
	decrement float64
}

func (d DecrAeroKoefStore) Decrement() float64 { return d.decrement }

func NewDecrementStore() *DecrementStore {
	return &DecrementStore{decrements: make(map[int][]DecrAeroKoefStore)}
}

func (s *DecrementStore) AddDecrement(nodeID int, koef1, koef2 float64, decrement float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.decrements[nodeID] = append(s.decrements[nodeID], DecrAeroKoefStore{koef1: koef1, koef2: koef2, decrement: decrement})
}

func (s *DecrementStore) GetDecrements(nodeID int) []DecrAeroKoefStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.decrements[nodeID]
}

// LookupDecrement возвращает лог. декремент для узла и пары аэрокоэффициентов (koef1, koef2).
func (s *DecrementStore) LookupDecrement(nodeID int, koef1, koef2 float64) (float64, bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.decrements[nodeID] {
		if aeroKoefEqual(d.koef1, koef1) && aeroKoefEqual(d.koef2, koef2) {
			return d.decrement, true
		}
	}
	return 0, false
}

func aeroKoefEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
}

func (s *DecrementStore) WriteDecrStoreToFiles() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for nodeID, decrements := range s.decrements {
		path := fmt.Sprintf(decrementStoreFileTmpl, nodeID)

		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
		if err != nil {
			return fmt.Errorf("open decrement file for node %d: %w", nodeID, err)
		}

		for _, d := range decrements {
			decr := d.decrement
			// if decr < 0.1 && decr > -0.1 {
			// 	decr = 0
			// }

			if _, err := fmt.Fprintf(f, "%f %f %f\n", d.koef1, d.koef2, decr); err != nil {
				_ = f.Close()
				return fmt.Errorf("write decrement for node %d: %w", nodeID, err)
			}
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("close decrement file for node %d: %w", nodeID, err)
		}
	}
	return nil
}
