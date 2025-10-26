package main

import (
	"fmt"
	"masters/internal/config"
	"masters/internal/defaults"
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
	T := 500.0
	dt := 0.001

	// Записываем коэффициенты связанности в coupled.txt

	// if bodiesConds.IsCoupled {
	// 	equationsolver.SolverCoupled(&bodiesConds, &timeConds)
	// } else {
	// 	equationsolver.Solver(&bodiesConds, &timeConds)
	// }

	// --- Ниже: демонстрационный расчёт универсальным решателем ---
	// Одна лопатка: m=1, k=1, d=1, x0=1, v0=0; запись в том же формате t x(t)

	b := &config.SimpleBody{ID: 0, Mass: 1.0, Position: 1.0, Velocity: 0.0, K: 10.0, D: 1.0}
	bodies := []config.BodyInterface{b}

	forces := config.AssembleForcesFromBodies(bodies)
	reg := config.NewForceRegistry()
	reg.Index(forces)

	// Отладочный вывод отключен

	uSolver := equationsolver.NewSolver(bodies, reg, dt)

	// собираем траекторию X во времени, включая начальное значение
	// n определяется так же, как в основном решателе
	n := int(((T - t0) + 1e-12) / dt)
	X := make([]float64, 0, n+1)
	X = append(X, bodies[0].GetPosition())
	for i := 0; i < n; i++ {
		t := t0 + float64(i)*dt
		_ = t // время передаём в шаг для возможных внешних сил

		// Отладочный вывод отключен

		uSolver.Step(t)
		X = append(X, bodies[0].GetPosition())
	}

	// пишем результат в файл в формате: t value (как в проекте)
	out, _ := os.OpenFile(defaults.Points1FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer out.Close()
	for i, val := range X {
		fmt.Fprintf(out, "%.10f ", t0+float64(i)*dt)
		fmt.Fprintf(out, "%.10f\n", val)
	}

	// --- Расчёт для двух связанных тел ---
	// Тело 1: m=1, k=10, d=1, x0=2, v0=0
	// Тело 2: m=1, k=10, d=1, x0=1, v0=0
	// Связь между телами: k_12=1, d_12=1
	// --- Задача о синхронизации метрономов ---
	// Узел 0: жёстко зафиксированная платформа
	// Узел 1: подвижная платформа, связана с узлом 0 через пружину
	// Узлы 2,3: метрономы на подвижной платформе, жёстко связаны с узлом 1

	fixedPlatform := &config.FixedBody{ID: 0, Position: 0.0} // pos 0

	mobilePlatform := &config.SimpleBody{
		ID: 1, Mass: 1.0, Position: 4.0, Velocity: 0.0, //pos 4
		K: 0.0, D: 0.0, // собственных сил нет
		Couplings: []config.Coupling{
			{J: 0, Kij: 1.0, Dij: 0.0, Rest: 0.0}, // связь с фиксированной платформой
		},
	}

	metronome1 := &config.SimpleBody{
		ID: 2, Mass: 1.0, Position: 2.0, Velocity: 0.0, // pos 2
		K: 1.0, D: 1.0, // собственных сил нет
		Couplings: []config.Coupling{
			{J: 3, Kij: 1.0, Dij: 0.0, Rest: 0.0}, // связь с метрономом 2
		},
	}

	metronome2 := &config.SimpleBody{
		ID: 3, Mass: 1.0, Position: 3.0, Velocity: 0.0, // pos 3
		K: 1.0, D: 0.0, // собственных сил нет
		Couplings: []config.Coupling{
			{
				J:   2,
				Kij: 1.0,
			},
		}, // связи задаются только с одной стороны
	}

	// Записываем параметры подвижной платформы в Params1FilePath
	params1, _ := os.OpenFile(defaults.Params1FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer params1.Close()
	fmt.Fprintf(params1, "%.6f \n", mobilePlatform.K)        // k
	fmt.Fprintf(params1, "%.6f \n", mobilePlatform.Mass)     // m
	fmt.Fprintf(params1, "%.6f \n", mobilePlatform.D)        // d
	fmt.Fprintf(params1, "%.6f \n", t0)                      // t0
	fmt.Fprintf(params1, "%.6f \n", T)                       // t
	fmt.Fprintf(params1, "%.6f \n", mobilePlatform.Position) // x0
	fmt.Fprintf(params1, "%.6f \n", mobilePlatform.Velocity) // v0

	// Записываем параметры первого метронома в Params2FilePath
	params2, _ := os.OpenFile(defaults.Params2FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer params2.Close()
	fmt.Fprintf(params2, "%.6f \n", metronome1.K)        // k
	fmt.Fprintf(params2, "%.6f \n", metronome1.Mass)     // m
	fmt.Fprintf(params2, "%.6f \n", metronome1.D)        // d
	fmt.Fprintf(params2, "%.6f \n", t0)                  // t0
	fmt.Fprintf(params2, "%.6f \n", T)                   // t
	fmt.Fprintf(params2, "%.6f \n", metronome1.Position) // x0
	fmt.Fprintf(params2, "%.6f \n", metronome1.Velocity) // v0

	coupledFile, _ := os.OpenFile("../wolfram/paramsAndPoints/coupled.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer coupledFile.Close()
	// Извлекаем коэффициенты из связей подвижной платформы
	var d12, k12 float64
	if len(mobilePlatform.Couplings) > 0 {
		d12 = mobilePlatform.Couplings[0].Dij
		k12 = mobilePlatform.Couplings[0].Kij
	}
	fmt.Fprintf(coupledFile, "%.10f\n", d12) // d_12
	fmt.Fprintf(coupledFile, "%.10f\n", k12) // k_12

	bodies2 := []config.BodyInterface{fixedPlatform, mobilePlatform, metronome1, metronome2}

	// Автоматически создаём силы на основе конфигурации тел
	forces2 := config.AutoAssembleForcesFromBodies(bodies2)

	reg2 := config.NewForceRegistry()
	reg2.Index(forces2)

	// for _, body := range bodies2 {
	// 	fmt.Printf("Body ID: %d\n", body.GetID())
	// 	fmt.Printf("Body mas: %f\n", body.GetMass())
	// 	fmt.Printf("Body pos: %f\n", body.GetPosition())
	// 	fmt.Printf("BodySpring %f\n", body.GetStiffness())
	// 	fmt.Printf("BodyDamping %f\n", body.GetDamping())
	// 	fmt.Printf("BodyCouplings %+v\n", body.GetCouplings())
	// 	fmt.Println("------------------------------------------")
	// }

	// Отладочный вывод отключен

	uSolver2 := equationsolver.NewSolver(bodies2, reg2, dt)

	// Записываем результаты всех тел:
	// points1.txt - подвижная платформа (индекс 1)
	// points2.txt - первый метроном (индекс 2)
	// points3.txt - второй метроном (индекс 3)
	// points4.txt - фиксированная платформа (индекс 0) - всегда 0
	points1File, _ := os.OpenFile(defaults.Points1FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer points1File.Close()
	points2File, _ := os.OpenFile(defaults.Points2FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer points2File.Close()
	points3File, _ := os.OpenFile("../wolfram/paramsAndPoints/points3.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer points3File.Close()
	points4File, _ := os.OpenFile("../wolfram/paramsAndPoints/points4.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer points4File.Close()

	for t := t0; t <= T; t += dt {
		uSolver2.Step(t)

		// Записываем время и позицию подвижной платформы (индекс 1)
		fmt.Fprintf(points1File, "%.10f %.10f\n", t, bodies2[1].GetPosition())

		// Записываем время и позицию первого метронома (индекс 2)
		fmt.Fprintf(points2File, "%.10f %.10f\n", t, bodies2[2].GetPosition())

		// Записываем время и позицию второго метронома (индекс 3)
		fmt.Fprintf(points3File, "%.10f %.10f\n", t, bodies2[3].GetPosition())

		// Записываем время и позицию фиксированной платформы (индекс 0) - всегда 0
		fmt.Fprintf(points4File, "%.10f %.10f\n", t, bodies2[0].GetPosition())
	}

	// ===== НОВАЯ ГРАФОВАЯ СИСТЕМА =====
	solveWithGraph(t0, T, dt)
}

// solveWithGraph решает задачу о метрономах используя графовую систему
func solveWithGraph(t0, T, dt float64) {
	fmt.Println("\n=== Решение задачи о метрономах через графовую систему ===")

	// Создаём граф
	graph := config.NewGraph()

	// Узел 0: Жёстко закреплённая платформа (неподвижная)
	fixedPlatform := config.NewFixedNode(0, 0.0)
	graph.AddNode(fixedPlatform)

	// Узел 1: Подвижная платформа
	// Масса: 1.0, начальное положение: 4.0, начальная скорость: 0.0
	// Собственных пружины и демпфера НЕТ (K=0, D=0) - все силы через рёбра!
	mobilePlatform := config.NewMovableNode(1, 1.0, 4.0, 0.0, 0.0, 0.0)
	graph.AddNode(mobilePlatform)

	// Узел 2: Метроном 1
	// Масса: 1.0, начальное положение: 1.0, начальная скорость: 0.0
	// Собственных сил НЕТ (K=0, D=0) - вся жёсткость через рёбра!
	metronome1 := config.NewMovableNode(2, 1.0, 2.0, 0.0, 0.0, 0.0)
	graph.AddNode(metronome1)

	// Узел 3: Метроном 2
	// Масса: 1.0, начальное положение: -1.0, начальная скорость: 0.0
	// Собственных сил НЕТ (K=0, D=0) - вся жёсткость через рёбра!
	metronome2 := config.NewMovableNode(3, 1.0, 3.0, 0.0, 0.0, 0.0)
	graph.AddNode(metronome2)

	// --- СВЯЗИ (РЁБРА ГРАФА) ---
	// Все коэффициенты задаются ЗДЕСЬ, в рёбрах!

	// Ребро 0-1: связь между подвижной платформой (1) и неподвижной платформой (0)
	// Пружина: k=0.1, Демпфер: d=0.2
	graph.AddEdge(0, 1, 4.0, 0.0, 0.0)

	// Ребро 1-2: связь между платформой (1) и метрономом 1 (2)
	// Пружина метронома: k=1.0 (метроном прикреплён к платформе), Демпфер: d=0.05
	graph.AddEdge(1, 2, 4.0, 0.0, 0.0)

	// Ребро 1-3: связь между платформой (1) и метрономом 2 (3)
	// Пружина метронома: k=1.0 (метроном прикреплён к платформе), Демпфер: d=0.05
	graph.AddEdge(1, 3, 4.0, 0.0, 0.0)

	// Ребро 2-3: связь между метрономами 1 и 2 для синхронизации
	// Демпфер между метрономами: d=0.01
	graph.AddEdge(2, 3, 4.0, 0.0, 0.0)

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

	// Выполняем расчёт
	iterations := 0
	for t := t0; t <= T; t += dt {
		solver.Step(t)

		// Записываем результаты для каждого узла в соответствующие файлы
		fmt.Fprintf(graphPoints1File, "%.10f %.10f\n", t, graph.GetNode(0).Position) // неподвижная платформа
		fmt.Fprintf(graphPoints2File, "%.10f %.10f\n", t, graph.GetNode(1).Position) // подвижная платформа
		fmt.Fprintf(graphPoints3File, "%.10f %.10f\n", t, graph.GetNode(2).Position) // метроном 1
		fmt.Fprintf(graphPoints4File, "%.10f %.10f\n", t, graph.GetNode(3).Position) // метроном 2

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
