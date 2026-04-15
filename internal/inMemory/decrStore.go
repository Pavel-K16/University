package inmemory

import (
	"fmt"
	"os"
)

const (
	decrementStoreFileTmpl = "../wolfram/paramsAndPoints/decrementStore%d.txt"
)

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

func (s *DecrementStore) WriteDecrStoreToFiles() error {
	for nodeID, decrements := range s.decrements {
		path := fmt.Sprintf(decrementStoreFileTmpl, nodeID)

		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
		if err != nil {
			return fmt.Errorf("open decrement file for node %d: %w", nodeID, err)
		}

		for _, d := range decrements {
			if _, err := fmt.Fprintf(f, "%f %f %f\n", d.koef1, d.koef2, d.decrement); err != nil {
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
