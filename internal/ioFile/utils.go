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
	sumEnergyFilePath       = "../wolfram/paramsAndPoints/sumEnergyPoints.txt"
	dSumEnergyFilePath      = "../wolfram/paramsAndPoints/dSumEnergyPoints.txt"
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
	sumEnergyFile, _ := os.OpenFile(sumEnergyFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	dSumEnergyFile, _ := os.OpenFile(dSumEnergyFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)

	defer sumEnergyFile.Close()
	defer dSumEnergyFile.Close()
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

	var prevSumEnergy float64
	hasPrev := false

	for t := t0; t <= T; t += dt {
		// if t == t0 {
		// 	for idx := range graphPointsFiles {
		// 		log.Debugf("Initial: Force 4 node %d %f", idx, graph.NetForce(idx))
		// 	}
		// }

		solver.Step(t)
		for idx, file := range graphPointsFiles {
			node := graph.GetNode(idx)
			if node == nil {
				log.Errorf("node is nil for index: %d", idx)
				continue
			}

			// if node.ID == 5 {
			// 	log.Debugf("NodeID 5 pos: %f", node.Position)
			// }

			// if t == t0 {
			// 	log.Debugf("Force 4 node %d %f", idx, graph.NetForce(idx))
			// }

			fmt.Fprintf(file, "%.10f %.10f\n", t, node.Position)
		}

		kineticEnergy := graph.TotalKineticEnergy()
		fmt.Fprintf(kineticEnergyFile, "%.10f %.10f\n", t, kineticEnergy)
		potentialEnergy := graph.TotalPotentialEnergy()
		fmt.Fprintf(potentialEnergyFile, "%.10f %.10f\n", t, potentialEnergy)
		sumEnergy := kineticEnergy + potentialEnergy
		fmt.Fprintf(sumEnergyFile, "%.10f %.10f\n", t, sumEnergy)

		// Численная производная полной энергии (равномерный шаг):
		// dE/dt ~ (E_i - E_{i-1}) / dt (назад направленная разность).
		dEdt := 0.0
		if hasPrev && dt > 0 {
			dEdt = (sumEnergy - prevSumEnergy) / dt
		}
		fmt.Fprintf(dSumEnergyFile, "%.10f %.10f\n", t, dEdt)
		prevSumEnergy = sumEnergy
		hasPrev = true
	}
}

func closeGraphPointsFiles(graphPointsFiles []*os.File) {
	for _, graphPointsFile := range graphPointsFiles {
		graphPointsFile.Close()
	}
}
