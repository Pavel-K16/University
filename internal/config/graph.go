package config

// Edge представляет связь между двумя узлами
type Edge struct {
	TargetID int     // ID узла, с которым связан текущий узел
	K        float64 // коэффициент жёсткости (k_ij)
	D        float64 // коэффициент демпфирования (d_ij)
	Rest     float64 // длина покоя (обычно 0)
}

// Node представляет узел графа (тело) со всеми своими связями
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

// Graph представляет всю систему как граф
type Graph struct {
	Nodes []*Node // все узлы системы
}

// NewGraph создаёт новый пустой граф
func NewGraph() *Graph {
	return &Graph{Nodes: make([]*Node, 0)}
}

// AddNode добавляет узел в граф
func (g *Graph) AddNode(node *Node) {
	if len(g.Nodes) <= node.ID {
		// Расширяем слайс если нужно
		newNodes := make([]*Node, node.ID+1)
		copy(newNodes, g.Nodes)
		g.Nodes = newNodes
	}
	g.Nodes[node.ID] = node
}

// AddEdge добавляет связь между узлами
// Связь добавляется в оба узла (двунаправленная)
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

	var force float64

	// 1. Собственные силы узла (если есть)
	force += -node.K * node.Position // пружина относительно земли
	force += -node.D * node.Velocity // демпфер относительно земли

	// 2. Силы от всех связей этого узла
	// Проходим по всем рёбрам узла, забираем параметры связанных узлов
	for _, edge := range node.Edges {
		targetNode := g.Nodes[edge.TargetID]
		if targetNode == nil {
			continue
		}

		// Вычисляем положение и скорость относительно связанного узла
		dx := node.Position - targetNode.Position - edge.Rest // деформация пружины
		dv := node.Velocity - targetNode.Velocity             // относительная скорость

		// Сила от пружины: F = -k * dx
		springForce := -edge.K * dx

		// Сила от демпфера: F = -d * dv
		damperForce := -edge.D * dv

		force += springForce + damperForce
	}

	return force
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
