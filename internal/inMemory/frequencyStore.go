package inmemory

import (
	"fmt"
	"math"
	"os"
	"sync"
)

const (
	freqStoreFileTmpl = "../wolfram/paramsAndPoints/freqStore%d.txt"
)

type FreqStore struct {
	mu    sync.Mutex
	freqs map[int][]FreqAeroKoef
}

type FreqAeroKoef struct {
	Koef1 float64
	Koef2 float64
	Freq  float64
}

func NewFrequencyStore() *FreqStore {
	return &FreqStore{freqs: make(map[int][]FreqAeroKoef)}
}

func (s *FreqStore) AddFreq(nodeID int, koef1, koef2 float64, freq float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.freqs[nodeID] = append(s.freqs[nodeID], FreqAeroKoef{Koef1: koef1, Koef2: koef2, Freq: freq})
}

func (s *FreqStore) GetFreqs(nodeID int) []FreqAeroKoef {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.freqs[nodeID]
}

// LookupFreq возвращает среднюю частоту (1/с) для узла и пары аэрокоэффициентов.
func (s *FreqStore) LookupFreq(nodeID int, koef1, koef2 float64) (float64, bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.freqs[nodeID] {
		if freqAeroKoefEqual(w.Koef1, koef1) && freqAeroKoefEqual(w.Koef2, koef2) {
			return w.Freq, true
		}
	}
	return 0, false
}

func freqAeroKoefEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
}

func (s *FreqStore) WriteFreqStoreToFiles() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for nodeID, freq := range s.freqs {
		path := fmt.Sprintf(freqStoreFileTmpl, nodeID)

		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
		if err != nil {
			return fmt.Errorf("open frequency file for node %d: %w", nodeID, err)
		}

		for _, w := range freq {
			if _, err := fmt.Fprintf(f, "%f %f %f\n", w.Koef1, w.Koef2, w.Freq); err != nil {
				_ = f.Close()
				return fmt.Errorf("write frequency for node %d: %w", nodeID, err)
			}
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("close frequency file for node %d: %w", nodeID, err)
		}
	}
	return nil
}
