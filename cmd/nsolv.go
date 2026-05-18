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
	frequencyStore := inmemory.NewFrequencyStore()

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

				go func(wg *sync.WaitGroup, cnf *config.Graph, t0, T, dt float64, decrementStore *inmemory.DecrementStore, frequencyStore *inmemory.FreqStore, f, b float64) {
					defer wg.Done()

					pointsStore := inmemory.NewPointsStore()
					skipFirst := 0
					solveParallelSweepJob(cnf, t0, T, dt, pointsStore, decrementStore, frequencyStore, f, b, skipFirst)
				}(&wg, cnf, t0, T, dt, decrementStore, frequencyStore, f, b)
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

		b := -0.2
		f := -0.2

		pointsStore := inmemory.NewPointsStore()
		amplitudeStore := inmemory.NewAmplitudeStore()
		energyStore := inmemory.NewEnergyStore()

		graph := g.NewGraph()

		solveWithGraph(cnf, graph, t0, T, dt, pointsStore, amplitudeStore, decrementStore, energyStore, f, b)

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

		freqs := GetAvgFrequency4Nodes(graph, amplitudeStore)
		for _, freq := range freqs {
			log.Infof("Frequency for node %d: %f", freq.NodeID, freq.Freq)
		}
	}

	log.Infof("Time taken: %v", time.Since(start))
}

type W struct {
	NodeID int
	Freq   float64
}

func GetAvgFrequency4Nodes(g *g.Graph, amplitudeStore *inmemory.AmplitudeStore) []W {

	freqs := make([]W, 0)

	for _, node := range g.Nodes {
		w := 0.0
		n := 0.0

		if node.IsFixed {
			continue
		}
		amplitudes := amplitudeStore.GetAmplitudes(node.ID)
		if len(amplitudes) < 2 {
			continue
		}

		for i := 0; i < len(amplitudes)-1; {
			t1 := amplitudes[i].T
			t2 := amplitudes[i+2].T
			dt := t2 - t1
			if dt > 0 {
				freq := 1 / dt
				w += freq
				n += 1.0
			}
			i += 2
		}
		if n == 0 {
			log.Infof("No frequencies found for node %d", node.ID)

			continue
		}

		freqs = append(freqs, W{NodeID: node.ID, Freq: w / n})
	}

	return freqs
}

func solveWithGraph(cnf *config.Graph, graph *g.Graph, t0, T, dt float64, pointsStore *inmemory.PointsStore, amplitudeStore *inmemory.AmplitudeStore, decrementStore *inmemory.DecrementStore, energyStore *inmemory.EnergyStore, f, b float64) {
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
	ampSkipFirst := 2 // сколько первых аплитуд скипаем
	if err := utils.WriteAmplitudePointsFromGraphFiles(cnf, pointsStore, amplitudeStore, ampSkipFirst); err != nil {
		log.Errorf("Error writing amplitude points: %v", err)
	}

	utils.PrintLogDecrementForAllNodes(cnf, pointsStore, skipFirst, decrementStore, f, b)
}

// solveParallelSweepJob один прогон для пары (f,b): интеграция без записи graph_points*
// и заполнение decrementStore (безопасно при многих горутинах).
func solveParallelSweepJob(cnf *config.Graph, t0, T, dt float64, pointsStore *inmemory.PointsStore, decrementStore *inmemory.DecrementStore, frequencyStore *inmemory.FreqStore, f, b float64, skipFirst int) {
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
