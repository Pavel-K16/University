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
	log = logger.LoggerInit()
)

func main() {
	// Конфигурация теперь задаётся программно
	t0 := 900.0
	T := 1000.0
	dt := 0.01

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
		ID: 2, Mass: 100.0, Position: 2.0, Velocity: 0.0, // pos 2
		K: 1.0, D: 1.0, // собственных сил нет
		Couplings: []config.Coupling{
			{J: 3, Kij: 1.0, Dij: 0.0, Rest: 0.0}, // связь с метрономом 2
		},
	}

	metronome2 := &config.SimpleBody{
		ID: 3, Mass: 1.0, Position: 3.0, Velocity: 0.0, // pos 3
		K: 1.0, D: 0.0, // собственных сил нет
		Couplings: []config.Coupling{}, // связи задаются только с одной стороны
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

}
