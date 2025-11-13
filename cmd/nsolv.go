package main

import (
	"fmt"
	"masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	"masters/internal/logger"
	"math"
	"os"
)

var (
	_ = logger.LoggerInit() // Logger for debugging
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

func main() {
	// Конфигурация теперь задаётся программно
	t0 := 0.0
	T := 20.0
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
	metronome1 := config.NewMovableNode(1, 0.210, 0.001, 0.0, 0.0, 0.0)
	graph.AddNode(metronome1)
	//  i    m     x
	mobilePlatform := config.NewMovableNode(2, 8.818, 0.000, 0.0, 0.0, 0.0) // платформа
	graph.AddNode(mobilePlatform)

	metronome2 := config.NewMovableNode(3, 0.210, -0.001, 0.0, 0.0, 0.0)
	graph.AddNode(metronome2)
	// n n_   k     d    L
	graph.AddEdge(1, 2, 37.108, 2.1378, 0.0)
	graph.AddEdge(3, 2, 37.108, 2.1378, 0.0)
	graph.AddEdge(2, 0, 388.71, 3.2656, 0.0)
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

	// Инициализируем структуры для хранения всех точек (для узлов 0, 1, 2, 3)
	nodeIDs := []int{0, 1, 2, 3}
	allPoints := make(map[int][]Point)
	for _, id := range nodeIDs {
		allPoints[id] = make([]Point, 0)
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

		// Сохраняем все точки для последующего анализа экстремумов
		for _, id := range nodeIDs {
			node := graph.GetNode(id)
			if node != nil {
				allPoints[id] = append(allPoints[id], Point{
					Time:     t,
					Position: node.Position,
				})
			}
		}

		iterations++
	}

	fmt.Printf("Расчёт завершён. Всего итераций: %d\n", iterations)
	fmt.Printf("Анализируем экстремумы для каждого узла...\n")

	// Находим все экстремумы для каждого узла
	extrema := findExtrema(allPoints, nodeIDs)

	// Записываем результаты экстремумов в файлы
	writeExtremaToFiles(extrema, nodeIDs)
}

// findExtrema находит все локальные максимумы (амплитудные отклонения) для каждого узла
// используя численную производную: максимум - когда производная слева положительная, справа отрицательная
func findExtrema(allPoints map[int][]Point, nodeIDs []int) map[int][]ExtremaPoint {
	extrema := make(map[int][]ExtremaPoint)

	for _, id := range nodeIDs {
		points := allPoints[id]
		if len(points) < 3 {
			// Нужно минимум 3 точки для определения экстремума
			extrema[id] = make([]ExtremaPoint, 0)
			continue
		}

		extremaList := make([]ExtremaPoint, 0)

		// Проходим по всем точкам, начиная со второй и заканчивая предпоследней
		for i := 1; i < len(points)-1; i++ {
			prev := points[i-1]
			curr := points[i]
			next := points[i+1]

			dtLeft := curr.Time - prev.Time
			dtRight := next.Time - curr.Time

			// Проверяем, что шаги по времени не равны нулю (на всякий случай)
			if dtLeft <= 0 || dtRight <= 0 {
				continue
			}

			derivativeLeft := (curr.Position - prev.Position) / dtLeft
			derivativeRight := (next.Position - curr.Position) / dtRight

			// Локальный максимум: производная слева положительная, справа отрицательная
			// Это означает, что функция растёт до этой точки и убывает после неё
			isMax := derivativeLeft > 0 && derivativeRight < 0

			// Если это локальный максимум, добавляем его
			if isMax {
				extremaList = append(extremaList, ExtremaPoint{
					Time:      curr.Time,
					Amplitude: math.Abs(curr.Position), // абсолютное значение амплитуды
				})
			}
		}

		extrema[id] = extremaList
		fmt.Printf("Узел %d: найдено %d локальных максимумов\n", id, len(extremaList))
	}

	return extrema
}

// writeExtremaToFiles записывает все найденные экстремумы (время, амплитуда) в соответствующие файлы
func writeExtremaToFiles(extrema map[int][]ExtremaPoint, nodeIDs []int) {
	// Имена файлов для амплитуд (соответствуют узлам)
	amplitudeFiles := map[int]string{
		0: "../wolfram/paramsAndPoints/amplitude1.txt", // неподвижная платформа
		1: "../wolfram/paramsAndPoints/amplitude2.txt", // подвижная платформа
		2: "../wolfram/paramsAndPoints/amplitude3.txt", // метроном 1
		3: "../wolfram/paramsAndPoints/amplitude4.txt", // метроном 2
	}

	for _, id := range nodeIDs {
		filePath, exists := amplitudeFiles[id]
		if !exists {
			continue
		}

		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Printf("Ошибка при открытии файла %s: %v\n", filePath, err)
			continue
		}

		extremaList := extrema[id]
		// Записываем все пары (время, амплитуда) для найденных экстремумов
		for _, ext := range extremaList {
			fmt.Fprintf(file, "%.10f %.10f\n", ext.Time, ext.Amplitude)
		}

		fmt.Printf("Узел %d: записано %d экстремумов в файл %s\n", id, len(extremaList), filePath)
		file.Close()
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
