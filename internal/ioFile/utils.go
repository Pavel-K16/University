package iofile

import (
	"fmt"
	config "masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	"masters/internal/logger"
	"os"
)

var (
	graphPointsFileTmpl = "../wolfram/paramsAndPoints/graph_points%s.txt"
	log                 = logger.LoggerInit()
)

var KineticEnergy []float64
var PotentialEnergy []float64

func WriteGraphPointsToFiles(solver *equationsolver.GraphSolver, graph *config.Graph, t0, T, dt float64) {
	nodesNum := graph.NodesNumbers()

	graphPointsFiles := make([]*os.File, 0, len(nodesNum))

	for _, node := range graph.Nodes {
		graphPointsFile, _ := os.OpenFile(fmt.Sprintf(graphPointsFileTmpl, fmt.Sprintf("%d", node.ID)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		graphPointsFiles = append(graphPointsFiles, graphPointsFile)
	}

	defer closeGraphPointsFiles(graphPointsFiles)

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
		KineticEnergy = append(KineticEnergy, kineticEnergy)
		potentialEnergy := graph.TotalPotentialEnergy()
		PotentialEnergy = append(PotentialEnergy, potentialEnergy)
	}
}

func closeGraphPointsFiles(graphPointsFiles []*os.File) {
	for _, graphPointsFile := range graphPointsFiles {
		graphPointsFile.Close()
	}
}
