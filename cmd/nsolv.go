package main

import (
	"fmt"
	"masters/internal/aero"
	config "masters/internal/config"
	equationsolver "masters/internal/equationSolver"
	g "masters/internal/graph"
	inmemory "masters/internal/inMemory"
	iofile "masters/internal/ioFile"
	"masters/internal/logger"
	"masters/internal/numMethods/utils"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	log = logger.LoggerInit() // Logger for debugging
)

const (
	minKoef = -2.0
	maxKoef = 2.0
	step    = 0.1
)

func parallelFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PARALLEL")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func main() {
	start := time.Now()

	nSteps := int(math.Round((maxKoef - minKoef) / step))

	decrementStore := inmemory.NewDecrementStore()
	if parallelFromEnv() {
		wg := sync.WaitGroup{}

		for i := 0; i <= nSteps; i++ {

			f := minKoef + float64(i)*step
			f = math.Round(f*1e10) / 1e10

			for j := 0; j <= nSteps; j++ {

				b := minKoef + float64(j)*step
				b = math.Round(b*1e10) / 1e10

				cnf, err := config.ReadGraphConfig()
				if err != nil {
					fmt.Fprintf(os.Stderr, "config: %v\n", err)
					os.Exit(1)
				}

				aero.AeroEnabled = cnf.Aero.Enabled

				T := cnf.Times.T
				t0 := cnf.Times.T0
				dt := cnf.Times.Dt

				wg.Add(1)

				go func(wg *sync.WaitGroup, cnf *config.Graph, t0, T, dt float64, decrementStore *inmemory.DecrementStore, f, b float64) {
					defer wg.Done()

					pointsStore := inmemory.NewPointsStore()

					solveParallelSweepJob(cnf, t0, T, dt, pointsStore, decrementStore, f, b)
				}(&wg, cnf, t0, T, dt, decrementStore, f, b)
			}
		}

		wg.Wait()

		if err := decrementStore.WriteDecrStoreToFiles(); err != nil {
			log.Errorf("Error writing decrement store to files: %v", err)
		} else {
			log.Info("Decrement store written to files")
		}

	} else {
		cnf, err := config.ReadGraphConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config: %v\n", err)
			os.Exit(1)
		}

		aero.AeroEnabled = cnf.Aero.Enabled

		T := cnf.Times.T
		t0 := cnf.Times.T0
		dt := cnf.Times.Dt

		b := -1.0
		f := 1.0

		pointsStore := inmemory.NewPointsStore()
		energyStore := inmemory.NewEnergyStore()

		graph := g.NewGraph()

		solveWithGraph(cnf, graph, t0, T, dt, pointsStore, decrementStore, energyStore, f, b)

		if err := energyStore.WriteEnenryStoreToFiles(graph.NodesNumbers()); err != nil {
			log.Errorf("Error writing energy store to files: %v", err)
		} else {
			log.Info("Energy store written to files")
		}
	}

	log.Infof("Time taken: %v", time.Since(start))
}

func solveWithGraph(cnf *config.Graph, graph *g.Graph, t0, T, dt float64, pointsStore *inmemory.PointsStore, decrementStore *inmemory.DecrementStore, energyStore *inmemory.EnergyStore, f, b float64) {
	fmt.Println("\n=== Решение задачи о метрономах через графовую систему ===")
	fmt.Printf("Начальные условия: t=0\n")
	fmt.Printf("Диапазон расчёта: t=[%.2f, %.2f], dt=%.4f\n", t0, T, dt)

	if err := g.CreateGraph(graph, cnf); err != nil {
		log.Errorf("CreateGraph: %v", err)
		return
	}

	//graph.PrintGraph()

	setAeroParams(cnf, graph)

	graph.BackAeroKoef = b
	graph.ForwardAeroKoef = f

	solver := equationsolver.NewGraphSolver(graph, dt)

	skipFirst := 2 // сколько первых максимумов амплитуды пропускаем для посчёта декремента

	iofile.WriteGraphPointsToFiles(solver, graph, t0, T, dt, pointsStore, energyStore)
	ampSkipFirst := 0 // сколько первых аплитуд скипаем
	if err := utils.WriteAmplitudePointsFromGraphFiles(cnf, pointsStore, ampSkipFirst); err != nil {
		log.Errorf("Error writing amplitude points: %v", err)
	}

	utils.PrintLogDecrementForAllNodes(cnf, pointsStore, skipFirst, decrementStore, f, b)
}

// solveParallelSweepJob один прогон для пары (f,b): интеграция без записи graph_points*
// и заполнение decrementStore (безопасно при многих горутинах).
func solveParallelSweepJob(cnf *config.Graph, t0, T, dt float64, pointsStore *inmemory.PointsStore, decrementStore *inmemory.DecrementStore, f, b float64) {
	graph := g.NewGraph()

	if err := g.CreateGraph(graph, cnf); err != nil {
		log.Errorf("CreateGraph: %v", err)
		return
	}

	setAeroParams(cnf, graph)

	graph.BackAeroKoef = b
	graph.ForwardAeroKoef = f

	solver := equationsolver.NewGraphSolver(graph, dt)

	simulatePointsInMemory(solver, graph, t0, T, dt, pointsStore)
	utils.StoreLogDecrementForAllNodes(cnf, pointsStore, 2, decrementStore, f, b)
}

func simulatePointsInMemory(solver *equationsolver.GraphSolver, graph *g.Graph, t0, T, dt float64, pointsStore *inmemory.PointsStore) {
	if pointsStore == nil {
		return
	}
	if t0 > 0 {
		for t := 0.0; t < t0; t += dt {
			solver.Step(t)
		}
	}
	for t := t0; t <= T; t += dt {
		solver.Step(t)
		for _, node := range graph.Nodes {
			if node == nil {
				continue
			}
			pointsStore.AddPoint(node.ID, inmemory.Point{T: t, X: node.Position})
		}
	}
}

func setAeroParams(cnf *config.Graph, graph *g.Graph) {
	if aero.AeroEnabled {
		//aero.InitInfluenceKoefMatrix(graph.NodesNumbers())
		aero.Scale = cnf.Aero.Scale

		aero.SetFlowParameters(
			cnf.Aero.V,   // скорость
			cnf.Aero.Pho, // плотность
			cnf.Aero.B,   // длина хорды
			cnf.Aero.M,   // обобщённая масаа
		)
	}
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
