package graph

import (
	"fmt"
	"masters/internal/aero"
	"masters/internal/logger"

	"masters/internal/config"
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

type Graph struct {
	NodesNum    int
	Period      float64
	FirstNodeID int
	LastNodeID  int
	Nodes       []*config.Node
}

func NewGraph() *Graph {
	return &Graph{Nodes: make([]*config.Node, 0)}
}

func (g *Graph) AddNode(node *config.Node) {
	g.Nodes = append(g.Nodes, node)
}

func (g *Graph) NodesNumbers() []int {
	numbers := make([]int, 0)
	for _, node := range g.Nodes {
		if node.IsFixed {
			continue
		}
		numbers = append(numbers, node.ID)
	}
	return numbers
}

func CreateGraph(graph *Graph) error {
	cnf := config.GetGConfig()

	if cnf == nil {
		log.Warningf("Got empty config")
		if err := config.LoadGraphConfig(); err != nil {
			log.Errorf("Error Load Config: %s", err)
			return err
		}
		cnf = config.GetGConfig()
	}

	// cnf.LastFirstDist
	// graph.Period =
	//nodesIDs := graph.NodesNumbers()

	graph.NodesNum = len(cnf.Nodes)
	num := len(cnf.Nodes) - 1

	period := 0.0

	for i := range cnf.Nodes {
		node := &cnf.Nodes[i]

		if node.IsFixed {
			fixedNode := NewFixedNode(node.ID, node.Position)
			graph.AddNode(fixedNode)

			continue
		}

		if i+1 <= num {
			step := 0.0
			nextNode := cnf.Nodes[i+1]
			if nextNode.IsFixed {
				graph.LastNodeID = node.ID
				step = cnf.LastFirstDist
			} else {
				step = nextNode.Position - node.Position
			}
			period += step
		}
		// добавляем сам объект из конфига, чтобы дальше работать с теми же узлами
		graph.AddNode(node)
	}

	graph.Period = period

	log.Debugf("Period: %f", graph.Period)

	if num > 0 {
		graph.FirstNodeID = graph.GetNode(0).ID
	}

	for _, edge := range cnf.Edges {
		if edge.FromID > num || edge.TargetID > num {
			continue
		}

		graph.AddEdge(edge)
	}

	return nil
}

func (g *Graph) AddEdge(edge config.Edge) {
	if edge.FromID >= len(g.Nodes) || edge.TargetID >= len(g.Nodes) {
		return
	}

	from := g.Nodes[edge.FromID]
	to := g.Nodes[edge.TargetID]

	// Добавляем связь в оба узла.
	// Для "прямого" направления используем edge как есть.
	fromEdge := edge
	from.Edges = append(from.Edges, fromEdge)

	// Для обратного направления меняем местами FromID/TargetID и инвертируем rest.
	toEdge := edge
	toEdge.FromID, toEdge.TargetID = edge.TargetID, edge.FromID
	toEdge.Rest = -edge.Rest
	to.Edges = append(to.Edges, toEdge)
}

func springDeformation(g *Graph, node, target *config.Node, edge config.Edge) float64 {
	targetPos := target.Position
	rest := edge.Rest

	if edge.Periodic {
		switch node.ID {
		case g.FirstNodeID:
			targetPos -= g.Period
			rest *= -1
		case g.LastNodeID:
			targetPos += g.Period
			rest *= -1
		default:
			targetPos = target.Position
		}
	}

	return targetPos - node.Position - rest
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

		dx := springDeformation(g, node, targetNode, edge) // деформация пружины

		dv := node.Velocity - targetNode.Velocity

		springForce := -edge.K * dx

		damperForce := -edge.D * dv //(dx*dx - 1) * dv // var der pol

		//damperForce := -edge.D * (180000*dx*dx - 1) * dv
		force += springForce + damperForce
	}

	//log.Debugf("AeroForce: %f; Force: %f", GetAeroForce(g, nodeID), force)
	aeroForce := 0.0

	if aero.AeroEnabled {
		aeroForce = GetAeroForce(g, nodeID)
		log.Debugf("Aero Force 4 node: %d %f", nodeID, aeroForce)
	}

	//if math.Abs(force) < 10.0 {
	log.Debugf("Usual Force 4 Node %d: %f", nodeID, force)
	//}

	return force + aeroForce
}

func (g *Graph) TotalPotentialEnergy() float64 {
	var sum float64
	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			targetNode := g.GetNode(edge.TargetID)
			if targetNode == nil {
				continue
			}
			dx := springDeformation(g, node, targetNode, edge)
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
	node := g.Nodes[nodeID]
	if node == nil {
		return 0
	}

	nodesIDs := g.NodesNumbers()
	if len(nodesIDs) == 0 {
		return 0
	}

	for _, edge := range node.Edges {
		targetNode := g.Nodes[edge.TargetID]
		if targetNode == nil || targetNode.IsFixed {
			continue
		}

		var aeroKoef float64
		if edge.Periodic {
			switch nodeID {
			case g.FirstNodeID:
				aeroKoef = -1.0
			case g.LastNodeID:
				aeroKoef = 1.0
			default:
				if nodeID > edge.TargetID {
					aeroKoef = -1.0
				} else {
					aeroKoef = 1.0
				}
			}
		} else {
			// Обычное правило: если nodeID > targetID, то -1, иначе +1
			if nodeID > edge.TargetID {
				aeroKoef = -1.0
			} else {
				aeroKoef = 1.0
			}
		}

		if edge.Periodic {
			var pos float64
			switch nodeID {
			case g.FirstNodeID:
				pos = targetNode.Position - g.Period
			case g.LastNodeID:
				pos = targetNode.Position + g.Period
			default:
				pos = targetNode.Position
			}
			aeroDinamicForce += aeroKoef * pos
		} else {
			aeroDinamicForce += aeroKoef * targetNode.Position
		}
	}

	v := aero.GetFlowVelocity()
	rho := aero.GetFlowDensity()
	b := aero.GetBladeChord()

	koeff := 0.5 * v * v * b * rho
	aeroDinamicForce *= koeff
	//log.Debugf("aeroDinamicForce 4 Node %d: %f", nodeID, aeroDinamicForce)
	return aeroDinamicForce * aero.Scale
}

// GetNode возвращает узел по ID
func (g *Graph) GetNode(id int) *config.Node {
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
func NewFixedNode(id int, position float64) *config.Node {
	return &config.Node{
		ID:       id,
		Mass:     0,
		Position: position,
		Velocity: 0,
		K:        0,
		D:        0,
		Edges:    make([]config.Edge, 0),
		IsFixed:  true,
	}
}

// NewMovableNode создаёт подвижный узел
func NewMovableNode(id int, mass, position, velocity, k, d float64) *config.Node {
	return &config.Node{
		ID:       id,
		Mass:     mass,
		Position: position,
		Velocity: velocity,
		K:        k,
		D:        d,
		Edges:    make([]config.Edge, 0),
		IsFixed:  false,
	}
}

func (g *Graph) PrintGraph() {
	fmt.Printf("\n=== Структура графа ===\n")

	type EdgeKey struct {
		from, to int
	}

	uniqueEdges := make(map[EdgeKey]bool)
	totalLinks := 0
	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			key := EdgeKey{node.ID, edge.TargetID}
			if !uniqueEdges[key] {
				uniqueEdges[key] = true
				totalLinks++
			}
		}
	}

	fmt.Printf("Узлов в графе: %d\n", len(g.Nodes))
	fmt.Printf("Всего связей: %d\n", totalLinks)
	fmt.Printf("Длина пружины в ненапряжённом состоянии: rest\n")
	fmt.Println()

	// Выводим информацию о каждом узле
	linkCounter := 1
	for _, node := range g.Nodes {
		fmt.Printf("Узел №%d: масса=%.2f, pos=%.2f, vel=%.2f",
			node.ID, node.Mass, node.Position, node.Velocity)
		if node.IsFixed {
			fmt.Print(" [ЗАКРЕПЛЁН]")
		}
		fmt.Printf(" (связей: %d)\n", len(node.Edges))

		// Выводим связи узла
		for _, edge := range node.Edges {
			fmt.Printf("  └─ связь №%d с узлом №%d: k=%.2f, d=%.2f, rest=%.2f\n",
				linkCounter-1, edge.TargetID, edge.K, edge.D, edge.Rest)
			linkCounter++
		}
	}
}
