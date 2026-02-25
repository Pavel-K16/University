package graph

import (
	"fmt"
	"masters/internal/aero"
	"masters/internal/logger"
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
	Periodic bool
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
	numbers := make([]int, 0)
	for _, node := range g.Nodes {
		if node.IsFixed {
			continue
		}
		numbers = append(numbers, node.ID)
	}
	return numbers
}

func (g *Graph) AddEdge(fromID, toID int, k, d, rest float64, periodic bool) {
	if fromID >= len(g.Nodes) || toID >= len(g.Nodes) {
		return
	}

	from := g.Nodes[fromID]
	to := g.Nodes[toID]

	// Добавляем связь в оба узла
	// rest задаётся для направления from→to, поэтому для обратного направления (to→from) инвертируем знак
	from.Edges = append(from.Edges, Edge{
		TargetID: toID,
		K:        k,
		D:        d,
		Rest:     rest, // rest для направления from→to
		Periodic: periodic,
	})

	to.Edges = append(to.Edges, Edge{
		TargetID: fromID,
		K:        k,
		D:        d,
		Rest:     -rest, // инвертируем знак для направления to→from
		Periodic: periodic,
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

		// rest задаётся для направления "от node к target", поэтому для вычисления dx
		// используем rest как есть (со знаком, заданным в конфиге)
		dx := targetNode.Position - node.Position - edge.Rest // деформация пружины
		//log.Debugf("All dx 4 Node: %d %f", nodeID, dx)

		dv := node.Velocity - targetNode.Velocity // относительная скорость

		springForce := -edge.K * dx

		damperForce := -edge.D * dv //(dx*dx - 1) * dv // var der pol

		//damperForce := -edge.D * (180000*dx*dx - 1) * dv
		force += springForce + damperForce
	}

	//log.Debugf("AeroForce: %f; Force: %f", GetAeroForce(g, nodeID), force)
	aeroForce := 0.0

	if aero.AeroEnabled {
		aeroForce = GetAeroForce(g, nodeID)
		//log.Debugf("Aero Force 4 node: %d %f", nodeID, aeroForce)
	}

	//log.Debugf("Usual Force 4 Node %d: %f", nodeID, force)

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
			dx := target.Position - node.Position - edge.Rest
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

	// Определяем первый и последний узлы в кольце (для учёта замыкания)
	nodesIDs := g.NodesNumbers()
	if len(nodesIDs) == 0 {
		return 0
	}
	firstNodeID := nodesIDs[0]
	lastNodeID := nodesIDs[len(nodesIDs)-1]

	// Вычисляем аэросилу через рёбра графа: для каждого ребра коэффициент зависит от направления
	for _, edge := range node.Edges {
		targetNode := g.Nodes[edge.TargetID]
		if targetNode == nil || targetNode.IsFixed {
			continue
		}

		// Проверяем, является ли это ребро замыканием кольца (первый ↔ последний)
		isClosingEdge := (nodeID == firstNodeID && edge.TargetID == lastNodeID) ||
			(nodeID == lastNodeID && edge.TargetID == firstNodeID)

		// Коэффициент: если nodeID > targetID, то -1, иначе +1
		// Но для замыкания кольца (первый ↔ последний) инвертируем правило
		var aeroKoef float64
		if isClosingEdge {
			// Для замыкания: узел 0 → узел 4: -1, узел 4 → узел 0: +1
			if nodeID == firstNodeID {
				aeroKoef = -1.0 // 0 → 4: -1
			} else {
				aeroKoef = 1.0 // 4 → 0: +1
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
			// Период = число лопаток; координаты 0,1,2,...,n-1.
			// 0 смотрит на 4: 4 «позади» → effective = pos_4 - period.
			// 4 смотрит на 0: 0 «впереди» → effective = pos_0 + period.
			period := float64(len(nodesIDs))
			var pos float64
			switch nodeID {
			case firstNodeID:
				pos = targetNode.Position - period
			case lastNodeID:
				pos = targetNode.Position + period
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
	log.Debugf("aeroDinamicForce 4 Node %d: %f", nodeID, aeroDinamicForce)
	return aeroDinamicForce * aero.Scale
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
