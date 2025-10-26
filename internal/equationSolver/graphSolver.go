package equationsolver

import (
	"masters/internal/config"
)

// GraphSolver решает систему уравнений для графа
type GraphSolver struct {
	graph *config.Graph
	dt    float64
}

// NewGraphSolver создаёт новый решатель для графа
func NewGraphSolver(graph *config.Graph, dt float64) *GraphSolver {
	return &GraphSolver{
		graph: graph,
		dt:    dt,
	}
}

// Step выполняет один шаг интегрирования (метод Рунге-Кутты 4-го порядка)
func (s *GraphSolver) Step(t float64) {
	// Сохраняем начальные состояния всех узлов
	type NodeState struct {
		ID       int
		Position float64
		Velocity float64
		Mass     float64
	}

	states := make([]NodeState, 0, len(s.graph.Nodes))
	for _, node := range s.graph.Nodes {
		states = append(states, NodeState{
			ID: node.ID, Position: node.Position, Velocity: node.Velocity, Mass: node.Mass,
		})
	}

	// Сохраняем позиции и скорости для расчётов
	pos := make(map[int]float64)
	vel := make(map[int]float64)
	for _, state := range states {
		pos[state.ID] = state.Position
		vel[state.ID] = state.Velocity
	}

	// Функция для временного установления состояния узлов и вычисления ускорения
	computeAcceleration := func(nodeStates []NodeState) map[int]float64 {
		// Устанавливаем временные состояния
		for _, state := range nodeStates {
			node := s.graph.Nodes[state.ID]
			if node != nil && !node.IsFixed {
				node.Position = state.Position
				node.Velocity = state.Velocity
			}
		}

		// Вычисляем ускорения
		acc := make(map[int]float64)
		for _, state := range nodeStates {
			if s.graph.Nodes[state.ID].IsFixed || state.Mass == 0 {
				acc[state.ID] = 0
				continue
			}
			acc[state.ID] = s.graph.NetForce(state.ID) / state.Mass
		}
		return acc
	}

	// k1 - производные в начальной точке
	k1vel := make(map[int]float64)
	k1acc := computeAcceleration(states)
	for _, state := range states {
		if !s.graph.Nodes[state.ID].IsFixed && state.Mass > 0 {
			k1vel[state.ID] = state.Velocity
		}
	}

	// k2 - производные на середине шага, используя k1
	midStates2 := make([]NodeState, 0, len(states))
	for _, state := range states {
		if !s.graph.Nodes[state.ID].IsFixed && state.Mass > 0 {
			midStates2 = append(midStates2, NodeState{
				ID:       state.ID,
				Position: pos[state.ID] + s.dt*k1vel[state.ID]/2,
				Velocity: vel[state.ID] + s.dt*k1acc[state.ID]/2,
				Mass:     state.Mass,
			})
		}
	}
	k2vel := make(map[int]float64)
	k2acc := computeAcceleration(midStates2)
	for _, state := range midStates2 {
		k2vel[state.ID] = state.Velocity // скорость в момент t + dt/2
	}

	// k3 - производные на середине шага, используя k2
	midStates3 := make([]NodeState, 0, len(states))
	for _, state := range states {
		if !s.graph.Nodes[state.ID].IsFixed && state.Mass > 0 {
			midStates3 = append(midStates3, NodeState{
				ID:       state.ID,
				Position: pos[state.ID] + s.dt*k2vel[state.ID]/2,
				Velocity: vel[state.ID] + s.dt*k2acc[state.ID]/2,
				Mass:     state.Mass,
			})
		}
	}
	k3vel := make(map[int]float64)
	k3acc := computeAcceleration(midStates3)
	for _, state := range midStates3 {
		k3vel[state.ID] = state.Velocity
	}

	// k4 - производные в конце шага, используя k3
	endStates := make([]NodeState, 0, len(states))
	for _, state := range states {
		if !s.graph.Nodes[state.ID].IsFixed && state.Mass > 0 {
			endStates = append(endStates, NodeState{
				ID:       state.ID,
				Position: pos[state.ID] + s.dt*k3vel[state.ID],
				Velocity: vel[state.ID] + s.dt*k3acc[state.ID],
				Mass:     state.Mass,
			})
		}
	}
	k4vel := make(map[int]float64)
	k4acc := computeAcceleration(endStates)
	for _, state := range endStates {
		k4vel[state.ID] = state.Velocity
	}

	// Обновляем все узлы по формуле РК4
	for _, state := range states {
		node := s.graph.Nodes[state.ID]
		if node.IsFixed || state.Mass == 0 {
			continue
		}

		newPos := pos[state.ID] + s.dt/6*(k1vel[state.ID]+2*k2vel[state.ID]+2*k3vel[state.ID]+k4vel[state.ID])
		newVel := vel[state.ID] + s.dt/6*(k1acc[state.ID]+2*k2acc[state.ID]+2*k3acc[state.ID]+k4acc[state.ID])

		s.graph.UpdatePosition(state.ID, newPos)
		s.graph.UpdateVelocity(state.ID, newVel)
	}
}
