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
	t0 := 0.0
	T := 100.0
	dt := 0.001

	// ===== НОВАЯ ГРАФОВАЯ СИСТЕМА =====
	solveWithGraph(t0, T, dt)
}

// solveWithGraph решает задачу о метрономах используя графовую систему
// Начальные условия всегда относятся к моменту времени t=0
// Расчёт выполняется от t0 до T:
//   - если t0 > 0, сначала эволюционирует система от 0 до t0 (без записи)
//   - затем рассчитывается от t0 до T (с записью результатов)
func solveWithGraph(t0, T, dt float64) {
	fmt.Println("\n=== Решение задачи о метрономах через графовую систему ===")
	fmt.Printf("Начальные условия: t=0\n")
	fmt.Printf("Диапазон расчёта: t=[%.2f, %.2f], dt=%.4f\n", t0, T, dt)

	// Создаём граф
	graph := config.NewGraph()

	// Узел 0: Жёстко закреплённая платформа (неподвижная)
	fixedPlatform := config.NewFixedNode(0, 0.0)
	graph.AddNode(fixedPlatform)

	// --- НАЧАЛЬНЫЕ УСЛОВИЯ (при t=0) ---

	// Узел 1: Подвижная платформа
	// Начальное состояние (t=0): масса=1.0, позиция=0.0, скорость=0.0
	// Собственных пружины и демпфера НЕТ - все силы через рёбра!

	//mobilePlatform := config.NewMovableNode(1, 1.0, 4.0, 0.0, 0.0, 0.0)
	//graph.AddNode(mobilePlatform)

	// Узел 2: Метроном 1
	// Начальное состояние (t=0): масса=1.0, позиция=1.0, скорость=0.0
	// Собственных сил НЕТ (K=0, D=0) - вся жёсткость через рёбра!

	metronome1 := config.NewMovableNode(1, 1.0, 4.0, 0.0, 0.0, 0.0)
	graph.AddNode(metronome1)

	// Узел 3: Метроном 2
	// Начальное состояние (t=0): масса=1.0, позиция=-1.0, скорость=0.0
	// Собственных сил НЕТ (K=0, D=0) - вся жёсткость через рёбра!

	metronome2 := config.NewMovableNode(2, 1.0, 2.0, 0.0, 0.0, 0.0)
	graph.AddNode(metronome2)

	// --- СВЯЗИ (РЁБРА ГРАФА) ---
	// Все коэффициенты задаются ЗДЕСЬ, в рёбрах!

	// Ребро 0-1: связь между подвижной платформой (1) и неподвижной платформой (0)
	// Пружина: k=1.0, Демпфер: d=0.0
	graph.AddEdge(0, 1, 1.0, 0.0, 0.0)

	// Ребро 1-2: связь между платформой (1) и метрономом 1 (2)
	// Пружина: k=1.0, Демпфер: d=0.0
	graph.AddEdge(0, 2, 1.0, 1.0, 0.0)
	graph.AddEdge(1, 2, 1.0, 0.0, 0.0)

	// Ребро 1-3: связь между платформой (1) и метрономом 2 (3)
	// Пружина: k=1.0, Демпфер: d=0.0
	//graph.AddEdge(1, 3, 1.0, 0.0, 0.0)

	// Ребро 2-3: связь между метрономами 1 и 2
	// Пружина: k=1.0, Демпфер: d=0.0
	//graph.AddEdge(2, 3, 1.0, 0.0, 0.0)

	// Выводим информацию о графе
	fmt.Printf("\n=== Структура графа ===\n")
	fmt.Printf("Узлов в графе: %d\n", len(graph.Nodes))
	for _, node := range graph.Nodes {
		fmt.Printf("Узел %d: масса=%.2f, pos=%.2f, vel=%.2f",
			node.ID, node.Mass, node.Position, node.Velocity)
		if node.IsFixed {
			fmt.Print(" [ЗАКРЕПЛЁН]")
		}
		fmt.Printf(" (связей: %d)\n", len(node.Edges))
		// Выводим связи узла
		for _, edge := range node.Edges {
			fmt.Printf("  └─ связь с узлом %d: k=%.2f, d=%.2f\n",
				edge.TargetID, edge.K, edge.D)
		}
	}

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
		fmt.Fprintf(graphPoints3File, "%.10f %.10f\n", t, graph.GetNode(2).Position) //graph.GetNode(2).Position) // метроном 1
		fmt.Fprintf(graphPoints4File, "%.10f %.10f\n", t, 0.0)                       //graph.GetNode(3).Position) // метроном 2

		iterations++
		// if iterations%1000 == 0 {
		// 	fmt.Printf("Время: %.2f, Платформа: %.6f, Метроном 1: %.6f, Метроном 2: %.6f\n",
		// 		t, graph.GetNode(1).Position, graph.GetNode(2).Position, graph.GetNode(3).Position)
		// }
	}

	fmt.Println("\nРасчёт завершён через графовую систему.")
	fmt.Println("Результаты записаны в файлы:")
	fmt.Println("  - graph_points1.txt (неподвижная платформа)")
	fmt.Println("  - graph_points2.txt (подвижная платформа)")
	fmt.Println("  - graph_points3.txt (метроном 1)")
	fmt.Println("  - graph_points4.txt (метроном 2)")
}
