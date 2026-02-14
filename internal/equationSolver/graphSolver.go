package equationsolver

import (
	graph "masters/internal/graph"
	"masters/internal/logger"
	"math"
)

var (
	log = logger.LoggerInit()
)

// GraphSolver решает систему уравнений для графа
type GraphSolver struct {
	graph *graph.Graph
	dt    float64
}

// NewGraphSolver создаёт новый решатель для графа
func NewGraphSolver(graph *graph.Graph, dt float64) *GraphSolver {
	return &GraphSolver{
		graph: graph,
		dt:    dt,
	}
}

// computeAccelerations вычисляет ускорения всех узлов при заданных положениях и скоростях
func computeAccelerations(graph *graph.Graph, positions, velocities map[int]float64) map[int]float64 {
	// Временно устанавливаем состояния
	savedPositions := make(map[int]float64)
	savedVelocities := make(map[int]float64)

	for id := range graph.Nodes {
		if !graph.Nodes[id].IsFixed {
			savedPositions[id] = graph.Nodes[id].Position
			savedVelocities[id] = graph.Nodes[id].Velocity
			graph.Nodes[id].Position = positions[id]
			graph.Nodes[id].Velocity = velocities[id]
		}
	}

	// Вычисляем ускорения
	accelerations := make(map[int]float64)
	for id := range graph.Nodes {
		a := 0.0
		node := graph.Nodes[id]
		if node.IsFixed || node.Mass == 0 {
			accelerations[id] = 0
		} else {
			a = graph.NetForce(id)
			accelerations[id] = a / node.Mass
		}
	}

	// Восстанавливаем состояния
	for id := range graph.Nodes {
		if !graph.Nodes[id].IsFixed {
			graph.Nodes[id].Position = savedPositions[id]
			graph.Nodes[id].Velocity = savedVelocities[id]
		}
	}

	return accelerations
}

// Step выполняет один шаг интегрирования (метод Рунге-Кутты 4-го порядка)
func (s *GraphSolver) Step(t float64) {
	// Текущие состояния из графа (исходные векторы)
	currPos := make(map[int]float64)
	currVel := make(map[int]float64)
	for id := range s.graph.Nodes {
		currPos[id] = s.graph.Nodes[id].Position
		currVel[id] = s.graph.Nodes[id].Velocity
	}

	// k1: производные в начальной точке
	acc1 := computeAccelerations(s.graph, currPos, currVel)

	// k2: промежуточное состояние на середине шага
	pos2 := make(map[int]float64)
	vel2 := make(map[int]float64)
	for id := range s.graph.Nodes {
		if !s.graph.Nodes[id].IsFixed && s.graph.Nodes[id].Mass > 0 {
			pos2[id] = currPos[id] + s.dt*currVel[id]/2
			vel2[id] = currVel[id] + s.dt*acc1[id]/2
		} else {
			pos2[id] = currPos[id]
			vel2[id] = currVel[id]
		}
	}
	acc2 := computeAccelerations(s.graph, pos2, vel2)

	// k3: ещё одно промежуточное состояние
	pos3 := make(map[int]float64)
	vel3 := make(map[int]float64)
	for id := range s.graph.Nodes {
		if !s.graph.Nodes[id].IsFixed && s.graph.Nodes[id].Mass > 0 {
			pos3[id] = currPos[id] + s.dt*vel2[id]/2
			vel3[id] = currVel[id] + s.dt*acc2[id]/2
		} else {
			pos3[id] = currPos[id]
			vel3[id] = currVel[id]
		}
	}
	acc3 := computeAccelerations(s.graph, pos3, vel3)

	// k4: состояние в конце шага
	pos4 := make(map[int]float64)
	vel4 := make(map[int]float64)
	for id := range s.graph.Nodes {
		if !s.graph.Nodes[id].IsFixed && s.graph.Nodes[id].Mass > 0 {
			pos4[id] = currPos[id] + s.dt*vel3[id]
			vel4[id] = currVel[id] + s.dt*acc3[id]
		} else {
			pos4[id] = currPos[id]
			vel4[id] = currVel[id]
		}
	}
	acc4 := computeAccelerations(s.graph, pos4, vel4)

	// Обновляем все узлы по формуле РК4
	for id := range s.graph.Nodes {
		node := s.graph.Nodes[id]
		if node.IsFixed || node.Mass == 0 {
			continue
		}

		// Формула РК4: новое = старое + dt/6 * (k1 + 2k2 + 2k3 + k4)
		newPos := currPos[id] + s.dt/6*(currVel[id]+2*vel2[id]+2*vel3[id]+vel4[id])
		newVel := currVel[id] + s.dt/6*(acc1[id]+2*acc2[id]+2*acc3[id]+acc4[id])

		if math.Abs(newPos) > 10 {
			log.Debugf("Time: %f", t)
			log.Debugf("Id: %d", id)
			log.Debugf("Error pos: %f", newPos)
		}

		// if math.Abs(newVel) > 10 {
		// 	log.Debugf("Time: %f", t)
		// 	log.Debugf("Id: %d", id)
		// 	log.Debugf("Error vel: %f", newVel)
		// }

		s.graph.UpdatePosition(id, newPos)
		s.graph.UpdateVelocity(id, newVel)
	}
}
