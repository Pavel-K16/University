package main

import (
	"fmt"
	"masters/internal/aero"
	"masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	iofile "masters/internal/ioFile"

	"masters/internal/logger"
	"math"
	"os"
)

var (
	log = logger.LoggerInit() // Logger for debugging
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

	len1 := len(iofile.KineticEnergy)
	len2 := len(iofile.PotentialEnergy)
	log.Debugf("Len1 : %d Len2 : %d", len1, len2)

	for i := 0; i < len1; i++ {
		log.Debugf("PotEn: %.2f, KinEn: %.2f:", iofile.PotentialEnergy[i], iofile.KineticEnergy[i])
	}
}

func findMin(l1, l2 int) int {
	if l1 < l2 {
		return l1
	} else {
		return l2
	}
}

// FindExtrema находит все локальные максимумы (амплитудные отклонения) для каждого узла
// используя численную производную: максимум - когда производная слева положительная, справа отрицательная
func FindExtrema(allPoints map[int][]Point, nodeIDs []int) map[int][]ExtremaPoint {
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

// WriteExtremaToFiles записывает все найденные экстремумы (время, амплитуда) в соответствующие файлы
func WriteExtremaToFiles(extrema map[int][]ExtremaPoint, nodeIDs []int) {
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
