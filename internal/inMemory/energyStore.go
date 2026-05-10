package inmemory

import "sync"

type EnergyStore struct {
	mu      sync.RWMutex
	energy  map[int][]energyPoint
	dEnergy map[int][]energyPoint
}

type energyPoint struct {
	time   float64
	energy float64
}

func NewEnergyStore() *EnergyStore {
	return &EnergyStore{mu: sync.RWMutex{}, energy: make(map[int][]energyPoint), dEnergy: make(map[int][]energyPoint)}
}

func (s *EnergyStore) AddEnergyPoint(nodeID int, t, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.energy[nodeID] = append(s.energy[nodeID], energyPoint{time: t, energy: val})
}

func (s *EnergyStore) AddDEnergyPoint(nodeID int, t, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dEnergy[nodeID] = append(s.dEnergy[nodeID], energyPoint{time: t, energy: val})
}

func (s *EnergyStore) GetEnergyPoints(nodeID int) []energyPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.energy[nodeID]
}

func (s *EnergyStore) GetDEnergyPoints(nodeID int) []energyPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.dEnergy[nodeID]
}
