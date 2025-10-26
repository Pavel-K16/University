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
	n := len(s.graph.Nodes)
	pos := make([]float64, n)
	vel := make([]float64, n)

	// Сохраняем текущие состояния
	for _, node := range s.graph.Nodes {
		pos[node.ID] = node.Position
		vel[node.ID] = node.Velocity
	}

	// Функция для вычисления ускорения узла в текущем состоянии
	accel := func(id int) float64 {
		if s.graph.Nodes[id].IsFixed || s.graph.Nodes[id].Mass == 0 {
			return 0
		}
		return s.graph.NetForce(id) / s.graph.Nodes[id].Mass
	}

	// Вычисляем k1 для всех
	k1vel := make([]float64, n)
	k1acc := make([]float64, n)
	for _, node := range s.graph.Nodes {
		if node.IsFixed || node.Mass == 0 {
			continue
		}
		k1vel[node.ID] = vel[node.ID]
		k1acc[node.ID] = accel(node.ID)
	}

	// Устанавливаем промежуточные позиции для k2
	for _, node := range s.graph.Nodes {
		if node.IsFixed || node.Mass == 0 {
			continue
		}
		node.Position = pos[node.ID] + s.dt*k1vel[node.ID]/2
		node.Velocity = vel[node.ID] + s.dt*k1acc[node.ID]/2
	}

	// Вычисляем k2
	k2vel := make([]float64, n)
	k2acc := make([]float64, n)
	for _, node := range s.graph.Nodes {
		if node.IsFixed || node.Mass == 0 {
			continue
		}
		k2vel[node.ID] = node.Velocity
		k2acc[node.ID] = accel(node.ID)
	}

	// Восстанавливаем и устанавливаем промежуточные позиции для k3
	for _, node := range s.graph.Nodes {
		if node.IsFixed || node.Mass == 0 {
			continue
		}
		node.Position = pos[node.ID] + s.dt*k1vel[node.ID]/2
		node.Velocity = vel[node.ID] + s.dt*k2acc[node.ID]/2
	}

	// Вычисляем k3
	k3vel := make([]float64, n)
	k3acc := make([]float64, n)
	for _, node := range s.graph.Nodes {
		if node.IsFixed || node.Mass == 0 {
			continue
		}
		k3vel[node.ID] = node.Velocity
		k3acc[node.ID] = accel(node.ID)
	}

	// Восстанавливаем и устанавливаем промежуточные позиции для k4
	for _, node := range s.graph.Nodes {
		if node.IsFixed || node.Mass == 0 {
			continue
		}
		node.Position = pos[node.ID] + s.dt*k3vel[node.ID]
		node.Velocity = vel[node.ID] + s.dt*k3acc[node.ID]
	}

	// Вычисляем k4
	k4vel := make([]float64, n)
	k4acc := make([]float64, n)
	for _, node := range s.graph.Nodes {
		if node.IsFixed || node.Mass == 0 {
			continue
		}
		k4vel[node.ID] = node.Velocity
		k4acc[node.ID] = accel(node.ID)
	}

	// Обновляем ВСЕ узлы
	for _, node := range s.graph.Nodes {
		if node.IsFixed || node.Mass == 0 {
			continue
		}
		id := node.ID
		newPos := pos[id] + s.dt/6*(k1vel[id]+2*k2vel[id]+2*k3vel[id]+k4vel[id])
		newVel := vel[id] + s.dt/6*(k1acc[id]+2*k2acc[id]+2*k3acc[id]+k4acc[id])

		s.graph.UpdatePosition(id, newPos)
		s.graph.UpdateVelocity(id, newVel)
	}
}
