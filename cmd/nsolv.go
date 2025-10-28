package main

import (
	"fmt"
	"masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	"masters/internal/logger"
	"os"
)

var (
	_ = logger.LoggerInit() // Logger for debugging
)

func main() {
	// Конфигурация теперь задаётся программно
	t0 := 100.0
	T := 120.0
	dt := 0.001

	// ===== НОВАЯ ГРАФОВАЯ СИСТЕМА =====
	solveWithGraph(t0, T, dt)
}

func solveWithGraph(t0, T, dt float64) {
	fmt.Println("\n=== Решение задачи о метрономах через графовую систему ===")
	fmt.Printf("Начальные условия: t=0\n")
	fmt.Printf("Диапазон расчёта: t=[%.2f, %.2f], dt=%.4f\n", t0, T, dt)

	graph := config.NewGraph()

	fixedPlatform := config.NewFixedNode(0, 0.0)
	graph.AddNode(fixedPlatform)
	//  i   m   x    v
	metronome1 := config.NewMovableNode(1, 0.075, 2.0, 0.0, 0.0, 0.0)
	graph.AddNode(metronome1)
	//  i    m     x
	metronome2 := config.NewMovableNode(2, 1.0, 4.0, 0.0, 0.0, 0.0)
	graph.AddNode(metronome2)

	metronome3 := config.NewMovableNode(3, 0.075, 3.0, 0.0, 0.0, 0.0)
	graph.AddNode(metronome3)
	// n n_   k     d    L
	graph.AddEdge(1, 2, 2.24, 0.0, 0.0)
	graph.AddEdge(3, 2, 2.24, 0.0, 0.0)
	graph.AddEdge(2, 0, 0.00001, 0.0, 0.0)
	//graph.AddEdge(1, 2, 1.0, 0.0, 0.0)
	PrintGraph(graph)

	// Создаём решатель
	solver := equationsolver.NewGraphSolver(graph, dt)

	// Открываем файлы для записи результатов
	graphPoints1File, _ := os.OpenFile("../wolfram/paramsAndPoints/graph_points1.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer graphPoints1File.Close()
	graphPoints2File, _ := os.OpenFile("../wolfram/paramsAndPoints/graph_points2.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer graphPoints2File.Close()
	graphPoints3File, _ := os.OpenFile("../wolfram/paramsAndPoints/graph_points3.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer graphPoints3File.Close()
	graphPoints4File, _ := os.OpenFile("../wolfram/paramsAndPoints/graph_points4.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer graphPoints4File.Close()

	// Эволюция системы от t=0 до t0 (без записи результатов)
	if t0 > 0 {
		fmt.Printf("Эволюция системы от t=0 до t=%.2f (без записи)\n", t0)
		for t := 0.0; t < t0; t += dt {
			solver.Step(t)
		}
		fmt.Printf("Система эволюционировала до t=%.2f\n", t0)
	}

	// Выполняем расчёт от t0 до T (с записью результатов)
	fmt.Printf("Запускаем расчёт от t=%.2f до t=%.2f\n", t0, T)
	iterations := 0
	for t := t0; t <= T; t += dt {
		solver.Step(t)

		// Записываем результаты для каждого узла в соответствующие файлы
		fmt.Fprintf(graphPoints1File, "%.10f %.10f\n", t, graph.GetNode(0).Position) // неподвижная платформа
		fmt.Fprintf(graphPoints2File, "%.10f %.10f\n", t, graph.GetNode(1).Position) // подвижная платформа
		fmt.Fprintf(graphPoints3File, "%.10f %.10f\n", t, graph.GetNode(2).Position) //graph.GetNode(2).Position) //graph.GetNode(2).Position) //graph.GetNode(2).Position) // метроном 1
		fmt.Fprintf(graphPoints4File, "%.10f %.10f\n", t, graph.GetNode(3).Position) //graph.GetNode(3).Position) // метроном 2

		iterations++
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
