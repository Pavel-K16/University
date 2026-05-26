package inmemory

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const (
	phaseDiffStoreFileTmpl = "../wolfram/paramsAndPoints/phaseDiffStore%d.json"
)

// PhaseDiffRecord — межлопаточная фаза для одной пары (k+, k−) на сетке sweep.
type PhaseDiffRecord struct {
	Koef1      float64   `json:"koef1"`
	Koef2      float64   `json:"koef2"`
	NodeID     int       `json:"nodeId"`
	NeighborID int       `json:"neighborId"`
	MeanDeg    float64   `json:"meanDeg"`
	StdDeg     float64   `json:"stdDeg"`
	T          []float64 `json:"t"`
	PhaseDeg   []float64 `json:"phaseDeg"`
}

type PhaseDiffStore struct {
	mu      sync.Mutex
	records map[int][]PhaseDiffRecord
}

func NewPhaseDiffStore() *PhaseDiffStore {
	return &PhaseDiffStore{records: make(map[int][]PhaseDiffRecord)}
}

func (s *PhaseDiffStore) AddRecord(nodeID int, rec PhaseDiffRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[nodeID] = append(s.records[nodeID], rec)
}

func (s *PhaseDiffStore) WritePhaseDiffStoreToFiles() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for nodeID, recs := range s.records {
		path := fmt.Sprintf(phaseDiffStoreFileTmpl, nodeID)
		data, err := json.Marshal(recs)
		if err != nil {
			return fmt.Errorf("marshal phase diff for node %d: %w", nodeID, err)
		}
		if err := os.WriteFile(path, data, 0o666); err != nil {
			return fmt.Errorf("write phase diff for node %d: %w", nodeID, err)
		}
	}
	return nil
}
