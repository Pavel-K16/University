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
	T := 100.0
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

	graph.PrintGraph()

	//findExtrema(graph)
}

func findExtrema(graph *config.Graph) {
	for _, node := range graph.Nodes {
		if err := utils.FindExtrema(node.ID); err != nil {
			log.Errorf("Error finding extrema 4 node: %d, err: %s", node.ID, err)
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
