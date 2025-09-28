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
	var bodiesConds config.BodiesConds
	var timeConds config.TimeConds

	if err := config.CondsInit(&bodiesConds, &timeConds); err != nil {
		log.Errorf("%s", err)

		os.Exit(1)
	}

	// Записываем коэффициенты связанности в coupled.txt

	// if bodiesConds.IsCoupled {
	// 	equationsolver.SolverCoupled(&bodiesConds, &timeConds)
	// } else {
	// 	equationsolver.Solver(&bodiesConds, &timeConds)
	// }

	// --- Ниже: демонстрационный расчёт универсальным решателем ---
	// Одна лопатка: m=1, k=1, d=1, x0=1, v0=0; запись в том же формате t x(t)
	dt := timeConds.Tau
	t0 := timeConds.T0
	T := timeConds.T

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
	b1 := &config.SimpleBody{
		ID: 0, Mass: 1.0, Position: 1.0, Velocity: 1.0,
		K: 2.0, D: 0.1, Couplings: []config.Coupling{{J: 1, Kij: 1.0, Dij: 0.1, Rest: 0.0}},
	}
	b2 := &config.SimpleBody{
		ID: 1, Mass: 1.0, Position: 2.0, Velocity: 1.0,
		K: 2.0, D: 0.1, Couplings: []config.Coupling{}, // Убираем дублирование!
	}

	params1, _ := os.OpenFile(defaults.Params1FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer params1.Close()
	fmt.Fprintf(params1, "%.6f \n", b1.K)        // k
	fmt.Fprintf(params1, "%.6f \n", b1.Mass)     // m
	fmt.Fprintf(params1, "%.6f \n", b1.D)        // d
	fmt.Fprintf(params1, "%.6f \n", t0)          // t0
	fmt.Fprintf(params1, "%.6f \n", T)           // t
	fmt.Fprintf(params1, "%.6f \n", b1.Position) // x0
	fmt.Fprintf(params1, "%.6f \n", b1.Velocity) // v0

	// пишем параметры второго тела в Params2FilePath (k, m, d, t0, t, x0, v0)
	params2, _ := os.OpenFile(defaults.Params2FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer params2.Close()
	fmt.Fprintf(params2, "%.6f \n", b2.K)        // k
	fmt.Fprintf(params2, "%.6f \n", b2.Mass)     // m
	fmt.Fprintf(params2, "%.6f \n", b2.D)        // d
	fmt.Fprintf(params2, "%.6f \n", t0)          // t0
	fmt.Fprintf(params2, "%.6f \n", T)           // t
	fmt.Fprintf(params2, "%.6f \n", b2.Position) // x0
	fmt.Fprintf(params2, "%.6f \n", b2.Velocity) // v0

	coupledFile, _ := os.OpenFile("../wolfram/paramsAndPoints/coupled.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer coupledFile.Close()
	// Извлекаем коэффициенты из связей первого тела
	var d12, k12 float64
	if len(b1.Couplings) > 0 {
		d12 = b1.Couplings[0].Dij
		k12 = b1.Couplings[0].Kij
	}
	fmt.Fprintf(coupledFile, "%.10f\n", d12) // d_12
	fmt.Fprintf(coupledFile, "%.10f\n", k12) // k_12

	bodies2 := []config.BodyInterface{b1, b2}

	forces2 := config.AssembleForcesFromBodies(bodies2)
	reg2 := config.NewForceRegistry()
	reg2.Index(forces2)

	// Отладочный вывод отключен

	uSolver2 := equationsolver.NewSolver(bodies2, reg2, dt)

	// собираем траектории для обоих тел
	n2 := int(((T - t0) + 1e-12) / dt)
	X1 := make([]float64, 0, n2+1)
	X2 := make([]float64, 0, n2+1)
	X1 = append(X1, bodies2[0].GetPosition())
	X2 = append(X2, bodies2[1].GetPosition())

	for i := 0; i < n2; i++ {
		t := t0 + float64(i)*dt

		// Отладочный вывод отключен

		uSolver2.Step(t)
		X1 = append(X1, bodies2[0].GetPosition())
		X2 = append(X2, bodies2[1].GetPosition())
	}

	// пишем результат первого тела в Points1FilePath
	out1, _ := os.OpenFile(defaults.Points1FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer out1.Close()
	for i, val := range X1 {
		fmt.Fprintf(out1, "%.10f ", t0+float64(i)*dt)
		fmt.Fprintf(out1, "%.10f\n", val)
	}

	// пишем результат второго тела в Points2FilePath
	out2, _ := os.OpenFile(defaults.Points2FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	defer out2.Close()
	for i, val := range X2 {
		fmt.Fprintf(out2, "%.10f ", t0+float64(i)*dt)
		fmt.Fprintf(out2, "%.10f\n", val)
	}
}
