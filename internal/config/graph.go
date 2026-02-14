package config

import (
	"masters/internal/aero"
	"masters/internal/logger"
	"slices"
)

var (
	log = logger.LoggerInit()
)

// Point представляет точку на графике (время, позиция)
type Point struct {
	Time     float64
	Position float64
}

// ExtremaPoint представляет точку экстремума (время, амплитуда)
type ExtremaPoint struct {
	Time      float64 // момент времени экстремума
	Amplitude float64 // абсолютное значение амплитуды
}

type Edge struct {
	TargetID int     // ID узла, с которым связан текущий узел
	K        float64 // коэффициент жёсткости (k_ij)
	D        float64 // коэффициент демпфирования (d_ij)
	Rest     float64 // длина покоя (обычноW 0)
}

type Node struct {
	ID       int     // уникальный идентификатор узла
	Mass     float64 // масса узла
	Position float64 // текущее положение
	Velocity float64 // текущая скорость

	// Собственные силы узла относительно "земли" (опционально)
	K float64 // собственная жёсткость (пружина относительно земли)
	D float64 // собственное демпфирование (демпфер относительно земли)

	// Связи этого узла с другими узлами
	Edges []Edge // список всех связей данного узла

	// Флаги состояния
	IsFixed bool // если true, узел жёстко закреплён (не двигается)
}

type Graph struct {
	Nodes []*Node
}

func NewGraph() *Graph {
	return &Graph{Nodes: make([]*Node, 0)}
}

func (g *Graph) AddNode(node *Node) {
	g.Nodes = append(g.Nodes, node)
}

func (g *Graph) NodesNumbers() []int {
	numbers := make([]int, len(g.Nodes)-1)
	i := 0

	for _, node := range g.Nodes {
		if node.IsFixed {
			continue
		}
		numbers[i] = node.ID
		i++
	}

	return numbers
}

func (g *Graph) AddEdge(fromID, toID int, k, d, rest float64) {
	if fromID >= len(g.Nodes) || toID >= len(g.Nodes) {
		return
	}

	from := g.Nodes[fromID]
	to := g.Nodes[toID]

	// Добавляем связь в оба узла
	from.Edges = append(from.Edges, Edge{
		TargetID: toID,
		K:        k,
		D:        d,
		Rest:     rest,
	})

	to.Edges = append(to.Edges, Edge{
		TargetID: fromID,
		K:        k,
		D:        d,
		Rest:     rest,
	})
}

// NetForce вычисляет чистую силу, действующую на узел
func (g *Graph) NetForce(nodeID int) float64 {
	node := g.Nodes[nodeID]

	// Если узел закреплён, сила не влияет на него
	if node == nil || node.IsFixed {
		return 0
	}

	var force float64 = 0.0

	for _, edge := range node.Edges {
		targetNode := g.Nodes[edge.TargetID]
		if targetNode == nil {
			continue
		}

		dx := node.Position - targetNode.Position - edge.Rest // деформация пружины
		dv := node.Velocity - targetNode.Velocity             // относительная скорость

		springForce := -edge.K * dx

		damperForce := -edge.D * dv //(dx*dx - 1) * dv // var der pol

		//damperForce := -edge.D * (180000*dx*dx - 1) * dv
		force += springForce + damperForce
	}

	aeroForce := 0.0 //GetAeroForce(g, nodeID)

	return force + aeroForce
}

func (g *Graph) TotalPotentialEnergy() float64 {
	var sum float64
	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			target := g.GetNode(edge.TargetID)
			if target == nil {
				continue
			}
			dx := node.Position - target.Position - edge.Rest
			sum += 0.25 * edge.K * dx * dx
		}
	}
	return sum
}

func (g *Graph) TotalKineticEnergy() float64 {
	var sum float64
	for _, node := range g.Nodes {
		if node.IsFixed {
			continue
		}
		sum += 0.5 * node.Mass * node.Velocity * node.Velocity
	}
	return sum
}

func GetAeroForce(g *Graph, nodeID int) float64 {
	aeroDinamicForce := 0.0
	nodesIDs := g.NodesNumbers()

	nodesKoef := aero.GetInfluenceKoefs(nodeID)

	for _, nodeKoef := range nodesKoef {
		if !slices.Contains(nodesIDs, nodeKoef.ID) {
			log.Errorf("node %d does't exist in the graph.", nodeKoef.ID)
			log.Errorf("Nodes in graph %+v", nodesIDs)

			return 0
		}

		aeroDinamicForce += nodeKoef.Koef * g.Nodes[nodeKoef.ID].Position
	}

	v := aero.GetFlowVelocity()
	rho := aero.GetFlowDensity()
	b := aero.GetBladeChord()
	//m := aero.GetBladeMass()

	koeff := 0.5 * v * v * b * rho
	aeroDinamicForce *= koeff * 0

	return aeroDinamicForce
}

// GetNode возвращает узел по ID
func (g *Graph) GetNode(id int) *Node {
	if id < 0 || id >= len(g.Nodes) {
		return nil
	}
	return g.Nodes[id]
}

// UpdatePosition обновляет положение узла (если не закреплён)
func (g *Graph) UpdatePosition(nodeID int, newPosition float64) {
	node := g.GetNode(nodeID)
	if node != nil && !node.IsFixed {
		node.Position = newPosition
	}
}

// UpdateVelocity обновляет скорость узла (если не закреплён)
func (g *Graph) UpdateVelocity(nodeID int, newVelocity float64) {
	node := g.GetNode(nodeID)
	if node != nil && !node.IsFixed {
		node.Velocity = newVelocity
	}
}

// Создаём удобные конструкторы

// NewFixedNode создаёт жёстко закреплённый узел
func NewFixedNode(id int, position float64) *Node {
	return &Node{
		ID:       id,
		Mass:     0,
		Position: position,
		Velocity: 0,
		K:        0,
		D:        0,
		Edges:    make([]Edge, 0),
		IsFixed:  true,
	}
}

// NewMovableNode создаёт подвижный узел
func NewMovableNode(id int, mass, position, velocity, k, d float64) *Node {
	return &Node{
		ID:       id,
		Mass:     mass,
		Position: position,
		Velocity: velocity,
		K:        k,
		D:        d,
		Edges:    make([]Edge, 0),
		IsFixed:  false,
	}
}
