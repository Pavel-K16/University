package config

// AeroInfluenceFunc - функция для получения коэффициента аэродинамического влияния
// sourceID - ID узла-источника (колеблется)
// targetID - ID узла-цели (испытывает силу)
// Возвращает коэффициент влияния
type AeroInfluenceFunc func(sourceID, targetID int) float64

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
	// Функция для получения коэффициента аэродинамического влияния
	// Если nil, аэродинамическое влияние не учитывается
	AeroInfluenceFunc AeroInfluenceFunc
}

func NewGraph() *Graph {
	return &Graph{Nodes: make([]*Node, 0)}
}

func (g *Graph) AddNode(node *Node) {
	g.Nodes = append(g.Nodes, node)
}

func (g *Graph) NodesNumbers() []int {
	numbers := make([]int, len(g.Nodes))
	for i, node := range g.Nodes {
		numbers[i] = node.ID
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

	// 1. Собственные силы узла (не используются в текущей задаче)
	// force += -node.K * node.Position // пружина относительно земли
	// force += -node.D * node.Velocity // демпфер относительно земли

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
		// Противодействует растяжению: если dx > 0 (пружина растянута),
		// то сила отрицательна (притягивает узел к другому узлу)
		springForce := -edge.K * dx

		// Сила от демпфера: F = -d * dv
		// Противодействует движению: если dv > 0 (узел движется быстрее),
		// то сила отрицательна (тормозит узел)
		//damperForce := -edge.D * dv

		damperForce := -edge.D * (dx*dx - 1) * dv // var der pol

		//damperForce := -edge.D * (180000*dx*dx - 1) * dv
		force += springForce + damperForce
	}

	// 3. Аэродинамическое влияние от всех остальных узлов
	if g.AeroInfluenceFunc != nil {
		nodesIDs := g.NodesNumbers()
		aeroDinamicForce := 0.0

		for _, id := range nodesIDs {
			if id == nodeID {
				continue
			}
			// Вызываем функцию для получения коэффициента влияния
			aeroDinamicForce += g.AeroInfluenceFunc(id, nodeID)
		}

		force += aeroDinamicForce
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
