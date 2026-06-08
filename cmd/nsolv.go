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
	minKoef = -1.0
	maxKoef = 1.0
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
	skipFirst := utils.SkipFirstMaxima()

	decrementStore := inmemory.NewDecrementStore()
	frequencyStore := inmemory.NewFrequencyStore()
	phaseDiffStore := inmemory.NewPhaseDiffStore()

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

				go func(wg *sync.WaitGroup, cnf *config.Graph, t0, T, dt float64, decrementStore *inmemory.DecrementStore, frequencyStore *inmemory.FreqStore, phaseDiffStore *inmemory.PhaseDiffStore, f, b float64) {
					defer wg.Done()

					pointsStore := inmemory.NewPointsStore()
					solveParallelSweepJob(cnf, t0, T, dt, pointsStore, decrementStore, frequencyStore, phaseDiffStore, f, b, skipFirst)
				}(&wg, cnf, t0, T, dt, decrementStore, frequencyStore, phaseDiffStore, f, b)
			}
		}

		wg.Wait()

		if err := decrementStore.WriteDecrStoreToFiles(); err != nil {
			log.Errorf("Error writing decrement store to files: %v", err)
		} else {
			log.Info("Decrement store written to files")
		}

		if err := frequencyStore.WriteFreqStoreToFiles(); err != nil {
			log.Errorf("Error writing frequency store to files: %v", err)
		} else {
			log.Info("Frequency store written to files")
		}

		if err := phaseDiffStore.WritePhaseDiffStoreToFiles(); err != nil {
			log.Errorf("Error writing phase diff store to files: %v", err)
		} else {
			log.Info("Phase diff store written to files")
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

		f := 0.2
		b := -0.2
		fl := 0.0
		bl := 0.0

		pointsStore := inmemory.NewPointsStore()
		amplitudeStore := inmemory.NewAmplitudeStore()
		energyStore := inmemory.NewEnergyStore()

		graph := g.NewGraph()

		solveWithGraph(cnf, graph, t0, T, dt, pointsStore, amplitudeStore, decrementStore, energyStore, f, b, fl, bl, skipFirst)

		if err := energyStore.WriteEnergyStoreToFiles(graph.NodesNumbers()); err != nil {
			log.Errorf("Error writing energy store to files: %v", err)
		} else {
			log.Info("Energy store written to files")
		}

		if err := amplitudeStore.WriteAmplitudeStoreToFiles(graph.NodesNumbers()); err != nil {
			log.Errorf("Error writing amplitude store to files: %v", err)
		} else {
			log.Info("Amplitude store written to files")
		}

		utils.LogFrequencySummary(cnf, pointsStore, skipFirst, f, b)
	}

	log.Infof("Time taken: %v", time.Since(start))
}

func solveWithGraph(cnf *config.Graph, graph *g.Graph, t0, T, dt float64, pointsStore *inmemory.PointsStore, amplitudeStore *inmemory.AmplitudeStore, decrementStore *inmemory.DecrementStore, energyStore *inmemory.EnergyStore, f, b, fl, bl float64, skipFirst int) {
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
	graph.ForwardLAeroKoef = fl
	graph.BackLAeroKoef = bl

	solver := equationsolver.NewGraphSolver(graph, dt)

	iofile.WriteGraphPointsToFiles(solver, graph, t0, T, dt, pointsStore, energyStore)
	if err := utils.WriteAmplitudePointsFromGraphFiles(cnf, pointsStore, amplitudeStore, skipFirst); err != nil {
		log.Errorf("Error writing amplitude points: %v", err)
	}

	utils.PrintLogDecrementForAllNodes(cnf, pointsStore, skipFirst, decrementStore, f, b)
	if err := utils.WritePhasePointsForSingleRun(cnf, pointsStore, decrementStore, nil, f, b, skipFirst); err != nil {
		log.Errorf("Error writing phase points: %v", err)
	}
	utils.PrintInterbladePhaseSummary(cnf, pointsStore, decrementStore, nil, f, b, skipFirst)
}

// solveParallelSweepJob один прогон для пары (f,b): интеграция без записи graph_points*
// и заполнение decrementStore (безопасно при многих горутинах).
func solveParallelSweepJob(cnf *config.Graph, t0, T, dt float64, pointsStore *inmemory.PointsStore, decrementStore *inmemory.DecrementStore, frequencyStore *inmemory.FreqStore, phaseDiffStore *inmemory.PhaseDiffStore, f, b float64, skipFirst int) {
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
	utils.StoreLogDecrementForAllNodes(cnf, pointsStore, skipFirst, decrementStore, frequencyStore, f, b)
	utils.StorePhaseDiffForSweep(cnf, pointsStore, phaseDiffStore, decrementStore, frequencyStore, f, b, skipFirst)
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
