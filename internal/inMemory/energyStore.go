package inmemory

import (
	"fmt"
	"os"
	"sync"
)

const (
	energyPointsFileTmpl  = "../wolfram/paramsAndPoints/energy_points%d.txt"
	dEnergyPointsFileTmpl = "../wolfram/paramsAndPoints/d_energy_points%d.txt"
)

type EnergyStore struct {
	mu      sync.RWMutex
	energy  map[int][]EnergyPoint
	dEnergy map[int][]EnergyPoint
}

type EnergyPoint struct {
	Time   float64
	Energy float64
}

func NewEnergyStore() *EnergyStore {
	return &EnergyStore{mu: sync.RWMutex{}, energy: make(map[int][]EnergyPoint), dEnergy: make(map[int][]EnergyPoint)}
}

func (s *EnergyStore) AddEnergyPoint(nodeID int, t, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.energy[nodeID] = append(s.energy[nodeID], EnergyPoint{Time: t, Energy: val})
}

func (s *EnergyStore) AddDEnergyPoint(nodeID int, t, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dEnergy[nodeID] = append(s.dEnergy[nodeID], EnergyPoint{Time: t, Energy: val})
}

func (s *EnergyStore) GetEnergyPoints(nodeID int) []EnergyPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.energy[nodeID]
}

func (s *EnergyStore) GetDEnergyPoints(nodeID int) []EnergyPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.dEnergy[nodeID]
}

func (s *EnergyStore) WriteEnenryStoreToFiles(nodeIDs []int) error {
	for _, nodeID := range nodeIDs {
		enPoints := s.GetEnergyPoints(nodeID)
		dEnPoints := s.GetDEnergyPoints(nodeID)

		enFile, err := os.Create(fmt.Sprintf(energyPointsFileTmpl, nodeID))
		if err != nil {
			return err
		}
		defer enFile.Close()

		for _, enPoint := range enPoints {
			fmt.Fprintf(enFile, "%f %f\n", enPoint.Time, enPoint.Energy)
		}

		dEnFile, err := os.Create(fmt.Sprintf(dEnergyPointsFileTmpl, nodeID))
		if err != nil {
			return err
		}

		defer dEnFile.Close()

		for _, dEnPoint := range dEnPoints {
			fmt.Fprintf(dEnFile, "%f %f\n", dEnPoint.Time, dEnPoint.Energy)
		}

	}

	return nil
}
