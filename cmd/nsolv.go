package main

import (
	"fmt"
	"masters/internal/aero"
	"masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	iofile "masters/internal/ioFile"
	"masters/internal/numMethods/utils"

	"masters/internal/logger"
)

var (
	log = logger.LoggerInit() // Logger for debugging
)

func main() {
	// Конфигурация теперь задаётся программно
	t0 := 0.0
	T := 10.0
	dt := 0.1

	// ===== НОВАЯ ГРАФОВАЯ СИСТЕМА =====
	solveWithGraph(t0, T, dt)
}

// Текущая реализация немного костыльная
// Для корректной работы необходимо фиксирующий узел
// добавить последним в граф
// надо будет подумать над тем как это исправить
func solveWithGraph(t0, T, dt float64) {
	fmt.Println("\n=== Решение задачи о метрономах через графовую систему ===")
	fmt.Printf("Начальные условия: t=0\n")
	fmt.Printf("Диапазон расчёта: t=[%.2f, %.2f], dt=%.4f\n", t0, T, dt)

	graph := config.NewGraph()

	metronome0 := config.NewMovableNode(0,
		1.0,
		1.0,
		0.0,
		0.0,
		0.0,
	)
	graph.AddNode(metronome0)

	metronome1 := config.NewMovableNode(1,
		1.0,
		2.0,
		0.0,
		0.0,
		0.0,
	)
	graph.AddNode(metronome1)

	metronome2 := config.NewMovableNode(2,
		1.0,
		3.0,
		0.0,
		0.0,
		0.0,
	)
	graph.AddNode(metronome2)

	metronome3 := config.NewMovableNode(3,
		1.0,
		4.0,
		0.0,
		0.0,
		0.0,
	)
	graph.AddNode(metronome3)

	fixedPlatform := config.NewFixedNode(4, 0.0)
	graph.AddNode(fixedPlatform)

	graph.AddEdge(0, 1,
		1.0, // k [Н/м] - одинаковая жёсткость для обоих метрономов
		0.0, // d [Н·с/м] - демпфирование
		0.0, // rest - длина покоя
	)

	graph.AddEdge(1, 2,
		1.0, // k [Н/м] - одинаковая жёсткость
		0.0, // d [Н·с/м] - демпфирование
		0.0, // rest
	)

	graph.AddEdge(2, 3,
		1.0, // k₃ [Н/м] - УВЕЛИЧЕННАЯ жёсткость для собственной частоты платформы
		0.0, // d₃ [Н·с/м] - демпфирование
		0.0, // rest
	)

	graph.AddEdge(0, 3,
		1.0, // k₃ [Н/м] - УВЕЛИЧЕННАЯ жёсткость для собственной частоты платформы
		0.0, // d₃ [Н·с/м] - демпфирование
		0.0, // rest
	)

	graph.AddEdge(0, 4,
		1.0, // k₃ [Н/м] - УВЕЛИЧЕННАЯ жёсткость для собственной частоты платформы
		0.0, // d₃ [Н·с/м] - демпфирование
		0.0, // rest
	)

	graph.AddEdge(1, 4,
		1.0, // k₃ [Н/м] - УВЕЛИЧЕННАЯ жёсткость для собственной частоты платформы
		0.0, // d₃ [Н·с/м] - демпфирование
		0.0, // rest
	)

	graph.AddEdge(2, 4,
		1.0, // k₃ [Н/м] - УВЕЛИЧЕННАЯ жёсткость для собственной частоты платформы
		0.0, // d₃ [Н·с/м] - демпфирование
		0.0, // rest
	)

	graph.AddEdge(3, 4,
		1.0, // k₃ [Н/м] - УВЕЛИЧЕННАЯ жёсткость для собственной частоты платформы
		0.0, // d₃ [Н·с/м] - демпфирование
		0.0, // rest
	)

	aero.InitInfluenceKoefMatrix(graph.NodesNumbers())
	aero.SetFlowParameters(
		1.0, // скорость
		1.0, // плотность
		1.0, // длина хорды
		1.0, // обобщённая масаа
	)

	solver := equationsolver.NewGraphSolver(graph, dt)

	iofile.WriteGraphPointsToFiles(solver, graph, t0, T, dt)

	PrintGraph(graph)
	for _, node := range graph.Nodes {
		if err := utils.FindExtrema(node.ID); err != nil {
			log.Errorf("Error finding extrema 4 node: %s, err: %s", node.ID, err)
		}
	}

}

func findMin(l1, l2 int) int {
	if l1 < l2 {
		return l1
	} else {
		return l2
	}
}

func PrintGraph(graph *config.Graph) {
	fmt.Printf("\n=== Структура графа ===\n")

	type EdgeKey struct {
		from, to int
	}

	uniqueEdges := make(map[EdgeKey]bool)
	totalLinks := 0
	for _, node := range graph.Nodes {
		for _, edge := range node.Edges {
			key := EdgeKey{node.ID, edge.TargetID}
			if !uniqueEdges[key] {
				uniqueEdges[key] = true
				totalLinks++
			}
		}
	}

	fmt.Printf("Узлов в графе: %d\n", len(graph.Nodes))
	fmt.Printf("Всего связей: %d\n", totalLinks)
	fmt.Printf("Длина пружины в ненапряжённом состоянии: rest\n")
	fmt.Println()

	// Выводим информацию о каждом узле
	linkCounter := 1
	for _, node := range graph.Nodes {
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
