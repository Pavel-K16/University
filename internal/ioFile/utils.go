package iofile

import (
	"fmt"
	equationsolver "masters/internal/equationSolver"
	graph "masters/internal/graph"
	"masters/internal/logger"
	"os"
)

var (
	GraphPointsFileTmpl     = "../wolfram/paramsAndPoints/graph_points%s.txt"
	kineticEnergyFilePath   = "../wolfram/paramsAndPoints/kineticEnergyPoints.txt"
	potentialEnergyFilePath = "../wolfram/paramsAndPoints/potentialEnergyPoints.txt"
)

var KineticEnergy, PotentialEnergy []float64

var (
	log = logger.LoggerInit()
)

func WriteGraphPointsToFiles(solver *equationsolver.GraphSolver, graph *graph.Graph, t0, T, dt float64) {
	graphPointsFiles := make([]*os.File, 0)

	for _, node := range graph.Nodes {
		graphPointsFile, _ := os.OpenFile(fmt.Sprintf(GraphPointsFileTmpl, fmt.Sprintf("%d", node.ID)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		graphPointsFiles = append(graphPointsFiles, graphPointsFile)
	}

	defer closeGraphPointsFiles(graphPointsFiles)
	kineticEnergyFile, _ := os.OpenFile(kineticEnergyFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	potentialEnergyFile, _ := os.OpenFile(potentialEnergyFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)

	defer kineticEnergyFile.Close()
	defer potentialEnergyFile.Close()

	if t0 > 0 {
		fmt.Printf("Эволюция системы от t=0 до t=%.2f (без записи)\n", t0)
		for t := 0.0; t < t0; t += dt {
			solver.Step(t)
		}
		fmt.Printf("Система эволюционировала до t=%.2f\n", t0)
	}

	fmt.Printf("Запускаем расчёт от t=%.2f до t=%.2f\n", t0, T)

	for t := t0; t <= T; t += dt {
		solver.Step(t)
		for idx, file := range graphPointsFiles {
			node := graph.GetNode(idx)
			if node == nil {
				log.Errorf("node is nil for index: %d", idx)
				continue
			}

			fmt.Fprintf(file, "%.10f %.10f\n", t, node.Position)
		}

		kineticEnergy := graph.TotalKineticEnergy()
		fmt.Fprintf(kineticEnergyFile, "%.10f %.10f\n", t, kineticEnergy)
		potentialEnergy := graph.TotalPotentialEnergy()
		fmt.Fprintf(potentialEnergyFile, "%.10f %.10f\n", t, potentialEnergy)
	}
}

func closeGraphPointsFiles(graphPointsFiles []*os.File) {
	for _, graphPointsFile := range graphPointsFiles {
		graphPointsFile.Close()
	}
}
