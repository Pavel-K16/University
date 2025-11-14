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

	// Неподвижная основа
	fixedPlatform := config.NewFixedNode(0, 0.0)
	graph.AddNode(fixedPlatform)

	// КОНФИГУРАЦИЯ: Метрономы синхронизируются в фазе, платформа в противофазе
	// На основе статей: huygensSynchronization.md и Woafo-Kraenkel
	//
	// Ключевые принципы:
	// 1. Идентичные метрономы (для синхронизации в фазе)
	// 2. Начальные условия в ПРОТИВОФАЗЕ для метрономов (чтобы видеть процесс синхронизации)
	// 3. Тяжёлая платформа с собственной частотой (для противофазного движения)
	//
	// Из статьи Pena-Ramirez: при лёгкой платформе (m₃ = 4.1 кг) метрономы синхронизируются
	// в фазе, даже если начать с противофазных условий (x₁(0) = 1.97 мм, x₂(0) = -2.13 мм)

	// Метроном 1 - идентичные параметры для синхронизации в фазе
	// НАЧАЛЬНО В ПРОТИВОФАЗЕ с метрономом 2 (для визуализации процесса синхронизации)
	metronome1 := config.NewMovableNode(1,
		0.210, // масса m₁ [кг] - идентична метроному 2
		0.003, // начальное положение [м] - НАЧАЛЬНО В ПРОТИВОФАЗЕ (как в статье: 2.7-3.0 мм)
		0.0,   // начальная скорость [м/с]
		0.0,   // собственная жёсткость K
		0.0,   // собственное демпфирование D
	)
	graph.AddNode(metronome1)

	// Подвижная платформа - ТЯЖЁЛАЯ с собственной частотой
	// Тяжёлая платформа имеет инерцию и может колебаться в противофазе
	mobilePlatform := config.NewMovableNode(2,
		15.0,   // масса [кг] - ОЧЕНЬ ТЯЖЁЛАЯ платформа (μ = 0.210/15.0 ≈ 0.014)
		-0.002, // начальное положение [м] - НАЧАЛЬНО В ПРОТИВОФАЗЕ с метрономами
		0.0,    // начальная скорость [м/с]
		0.0,    // собственная жёсткость K
		0.0,    // собственное демпфирование D
	)
	graph.AddNode(mobilePlatform)

	// Метроном 2 - идентичные параметры метроному 1 (для синхронизации в фазе)
	// НАЧАЛЬНО В ПРОТИВОФАЗЕ с метрономом 1 (для визуализации процесса синхронизации)
	metronome2 := config.NewMovableNode(3,
		0.210,  // масса m₂ [кг] - идентична метроному 1
		-0.003, // начальное положение [м] - НАЧАЛЬНО В ПРОТИВОФАЗЕ (как в статье: -2.7--3.0 мм)
		0.0,    // начальная скорость [м/с]
		0.0,    // собственная жёсткость K
		0.0,    // собственное демпфирование D
	)
	graph.AddNode(metronome2)

	// Связи с ОДИНАКОВЫМИ жёсткостями (для синхронизации метрономов в фазе)
	// Связь метроном 1 с платформой
	graph.AddEdge(1, 2,
		37.108, // k [Н/м] - одинаковая жёсткость для обоих метрономов
		2.1378, // d [Н·с/м] - демпфирование
		0.0,    // rest - длина покоя
	)

	// Связь метроном 2 с платформой - та же жёсткость
	graph.AddEdge(3, 2,
		37.108, // k [Н/м] - одинаковая жёсткость
		2.1378, // d [Н·с/м] - демпфирование
		0.0,    // rest
	)

	// Связь платформы с неподвижной основой
	// УВЕЛИЧЕННАЯ жёсткость для создания собственной частоты платформы
	graph.AddEdge(2, 0,
		600.0,  // k₃ [Н/м] - УВЕЛИЧЕННАЯ жёсткость для собственной частоты платформы
		3.2656, // d₃ [Н·с/м] - демпфирование
		0.0,    // rest
	)

	// Ожидаемое поведение (на основе статей):
	// - Метрономы: НАЧИНАЮТ в противофазе (x₁(0) = 0.003 м, x₃(0) = -0.003 м)
	//   → СИНХРОНИЗИРУЮТСЯ в фазе (x₁ ≈ x₃, ẋ₁ ≈ ẋ₃) за ~5-10 секунд
	// - Платформа: колеблется в противофазе из-за большой массы и инерции
	// - Собственная частота платформы: ω₃ = √(600.0/15.0) ≈ 6.32 рад/с
	// - Собственная частота метрономов: ω = √(37.108/0.210) ≈ 13.3 рад/с
	//
	// Из статьи Pena-Ramirez: при лёгкой платформе метрономы синхронизируются в фазе,
	// даже если начать с противофазных условий. Здесь платформа тяжёлая, но метрономы
	// всё равно должны синхронизироваться в фазе благодаря идентичным параметрам.
	fmt.Printf("Конфигурация: Метрономы синхронизируются в фазе, платформа в противофазе\n")
	fmt.Printf("Метрономы: m=0.210 кг, κ=37.108 Н/м, ω ≈ 13.3 рад/с\n")
	fmt.Printf("  Начальные условия: x₁(0) = 0.003 м, x₃(0) = -0.003 м (в противофазе)\n")
	fmt.Printf("  Ожидание: синхронизация в фазе за ~5-10 секунд\n")
	fmt.Printf("Платформа: m=15.0 кг, κ₃=600.0 Н/м, ω₃ ≈ 6.32 рад/с (тяжёлая, своя частота)\n")
	fmt.Printf("  Начальное условие: x₂(0) = -0.002 м (в противофазе с метрономами)\n")

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
