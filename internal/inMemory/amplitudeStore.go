package inmemory

import (
	"fmt"
	"os"
	"sync"
)

const (
	amplitudePointsFileTmpl  = "../wolfram/paramsAndPoints/amplitude_points%d.txt"
	dAmplitudePointsFileTmpl = "../wolfram/paramsAndPoints/d_amplitude_points%d.txt"
)

type AmplitudeStore struct {
	mu          sync.RWMutex
	amplitudes  map[int][]AmplitudePoint
	dAmplitudes map[int][]AmplitudePoint
}

type AmplitudePoint struct {
	A float64
	T float64
}

func NewAmplitudeStore() *AmplitudeStore {
	return &AmplitudeStore{
		mu: sync.RWMutex{}, amplitudes: make(map[int][]AmplitudePoint),
		dAmplitudes: make(map[int][]AmplitudePoint),
	}
}

func (s *AmplitudeStore) AddAmplitude(nodeID int, x, t float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.amplitudes[nodeID] = append(s.amplitudes[nodeID], AmplitudePoint{A: x, T: t})

	addDAmpPoint(s, nodeID, x, t)
}

func addDAmpPoint(s *AmplitudeStore, nodeID int, x, t float64) {
	ampPoints := s.amplitudes[nodeID]
	if len(ampPoints) >= 2 {
		prevX := ampPoints[len(ampPoints)-2].A
		prevTime := ampPoints[len(ampPoints)-2].T
		dt := t - prevTime
		dx := x - prevX

		if dt != 0 {
			dA := (dx) / dt
			s.addDAmpPoint(nodeID, t, dA)
		}
	}
}

func (s *AmplitudeStore) addDAmpPoint(nodeID int, t, dx float64) {
	s.dAmplitudes[nodeID] = append(s.dAmplitudes[nodeID], AmplitudePoint{T: t, A: dx})
}

func (s *AmplitudeStore) GetAmplitudes(nodeID int) []AmplitudePoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.amplitudes[nodeID]
}

func (s *AmplitudeStore) GetDAmplitudes(nodeID int) []AmplitudePoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.dAmplitudes[nodeID]
}

func (s *AmplitudeStore) WriteAmplitudeStoreToFiles(nodeIDs []int) error {
	for _, nodeID := range nodeIDs {
		amplitudes := s.GetAmplitudes(nodeID)

		amplitudeFile, err := os.OpenFile(fmt.Sprintf(amplitudePointsFileTmpl, nodeID), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}

		for _, amplitude := range amplitudes {
			fmt.Fprintf(amplitudeFile, "%f %f\n", amplitude.T, amplitude.A)
		}

		if err := amplitudeFile.Close(); err != nil {
			return fmt.Errorf("close amplitude file for node %d: %w", nodeID, err)
		}

		dAmplitudeFile, err := os.OpenFile(fmt.Sprintf(dAmplitudePointsFileTmpl, nodeID), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}

		for _, dAmplitude := range s.dAmplitudes[nodeID] {
			fmt.Fprintf(dAmplitudeFile, "%f %f\n", dAmplitude.T, dAmplitude.A)
		}

		if err := dAmplitudeFile.Close(); err != nil {
			return fmt.Errorf("close d amplitude file for node %d: %w", nodeID, err)
		}
	}

	return nil
}
