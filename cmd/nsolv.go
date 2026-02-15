package main

import (
	"fmt"
	"masters/internal/aero"
	config "masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	g "masters/internal/graph"
	iofile "masters/internal/ioFile"
	"masters/internal/logger"
	"masters/internal/numMethods/utils"
)

var (
	log = logger.LoggerInit() // Logger for debugging
)

func main() {
	cnf := config.GetConfig()
	aero.AeroEnabled = cnf.Aero.Enabled
	aero.Scale = cnf.Aero.Scale

	T := cnf.Times.T
	t0 := cnf.Times.T0
	dt := cnf.Times.Dt

	solveWithGraph(t0, T, dt)
}

func solveWithGraph(t0, T, dt float64) {
	fmt.Println("\n=== Решение задачи о метрономах через графовую систему ===")
	fmt.Printf("Начальные условия: t=0\n")
	fmt.Printf("Диапазон расчёта: t=[%.2f, %.2f], dt=%.4f\n", t0, T, dt)

	graph := g.NewGraph()

	config.CreateGraph(graph)

	graph.PrintGraph()

	aero.InitInfluenceKoefMatrix(graph.NodesNumbers())
	aero.SetFlowParameters(
		1.0, // скорость
		1.0, // плотность
		1.0, // длина хорды
		1.0, // обобщённая масаа
	)

	solver := equationsolver.NewGraphSolver(graph, dt)

	iofile.WriteGraphPointsToFiles(solver, graph, t0, T, dt)

	//findExtrema(graph)
}

func findExtrema(graph *g.Graph) {
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
