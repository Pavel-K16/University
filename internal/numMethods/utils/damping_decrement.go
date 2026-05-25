package utils

import (
	"fmt"
	"math"
	"os"
	"strings"

	config "masters/internal/config"
	inmemory "masters/internal/inMemory"
)

type maxPoint struct {
	t float64
	a float64 // amplitude = |x - xEq|
}

const (
	decrementDetailsLogPath = "../wolfram/paramsAndPoints/decrement_details.log"
	decrementPointsFileTmpl = "../wolfram/paramsAndPoints/decrement_points%d.txt"
	zeroMaxs                = 123456.123456
	oneMax                  = 1232323.1232323
	xEqFlatTol              = 1e-6
	monoXTol                = 1e-12
)

// equilibriumForNodeFromConfig вычисляет равновесное положение xEq для nodeID,
// исходя из "ненапряжённого" состояния пружины к фиксированному узлу.
//
// Логика:
//   - если в конфиге есть ребро from=nodeID -> to=fixed с rest=r, то в графе xEq = xFixed - r
//   - если в конфиге есть ребро from=fixed -> to=nodeID с rest=r, то из-за инвертирования rest
//     (см. Graph.AddEdge) получается xEq = xFixed + r
//
// Для текущей постановки мы ожидаем, что от nodeID есть связь с одним фиксированным узлом.
func equilibriumForNodeFromConfig(cnf *config.Graph, nodeID int) (float64, bool) {
	if cnf == nil {
		return 0, false
	}

	fixedNodes := make([]config.Node, 0)
	for _, n := range cnf.Nodes {
		if n.IsFixed {
			fixedNodes = append(fixedNodes, n)
		}
	}

	if len(fixedNodes) == 0 {
		return 0, false
	}

	// Сначала ищем ребро nodeID -> fixed (предпочтительный случай).
	for _, fixed := range fixedNodes {
		for _, e := range cnf.Edges {
			if e.FromID == nodeID && e.TargetID == fixed.ID {
				return fixed.Position - e.Rest, true
			}
		}
	}

	// Затем ищем ребро fixed -> nodeID (в графе rest инвертируется, поэтому xEq = xFixed + rest).
	for _, fixed := range fixedNodes {
		for _, e := range cnf.Edges {
			if e.FromID == fixed.ID && e.TargetID == nodeID {
				return fixed.Position + e.Rest, true
			}
		}
	}

	return 0, false
}

func writeDecrementPoints(nodeID int, maxs []maxPoint, skipFirstMaxima int) error {
	path := fmt.Sprintf(decrementPointsFileTmpl, nodeID)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	start := skipFirstMaxima
	if start > len(maxs)-2 {
		start = len(maxs) - 2
	}
	if start < 0 {
		return nil
	}

	for i := start; i < len(maxs)-1; i++ {
		a1 := maxs[i].a
		a2 := maxs[i+1].a
		if a1 <= 0 || a2 <= 0 {
			continue
		}
		delta := math.Log(a1 / a2)
		// Время привязываем к правому пику пары (t_{n+1})
		fmt.Fprintf(f, "%.10f %.10f\n", maxs[i+1].t, delta)
	}

	return nil
}

// xMonotoneNonDecreasing проверяет X по порядку точек в слайсе (обычно это рост времени T).
func xMonotoneNonDecreasing(points []inmemory.Point, tol float64) bool {
	for i := 0; i < len(points)-1; i++ {
		if points[i+1].X < points[i].X-tol {
			return false
		}
	}
	return true
}

func EstimateLogDecrementFromGraphPoints(points []inmemory.Point, xEq float64, skipFirstMaxima int) (delta float64, used int, err error) {
	maxs := FindAmplitudeMaximaInMemory(points, xEq)
	if len(maxs) < 2 {
		if len(points) == 0 {
			return 0, 0, fmt.Errorf("not enough amplitude maxima: maxs=%d (need >=2 oscillation peaks; increase T or reduce damping)", len(maxs))
		}

		allAtEq := true
		for _, p := range points {
			if math.Abs(p.X-xEq) > xEqFlatTol {
				allAtEq = false
				break
			}
		}

		if allAtEq {
			return 0, 0, nil
		}

		if len(maxs) == 0 {
			return zeroMaxs, 0, nil
		}

		if len(maxs) == 1 {
			return oneMax, 0, nil
		}

		return 0, 0, fmt.Errorf("not enough amplitude maxima: maxs=%d (need >=2 oscillation peaks; increase T or reduce damping)", len(maxs))
	}

	start := skipFirstMaxima
	// Если интервал времени короткий, а максимумов мало, то строгое пропускание первых
	// может оставить меньше 2 максимумов. В этом случае уменьшим start так,
	// чтобы хотя бы посчитать по 2 максимумам.
	if start > len(maxs)-2 {
		start = len(maxs) - 2
	}

	deltas := make([]float64, 0, len(maxs)-start-1)
	for i := start; i < len(maxs)-1; i++ {
		a1 := maxs[i].a
		a2 := maxs[i+1].a
		if a1 <= 0 || a2 <= 0 {
			continue
		}
		deltas = append(deltas, math.Log(a1/a2))
	}

	if len(deltas) == 0 {
		return 0, 0, fmt.Errorf("failed to compute deltas (all zeros?)")
	}

	var sum float64
	for _, d := range deltas {
		sum += d
	}
	return sum / float64(len(deltas)), len(deltas), nil
}

// PrintLogDecrementForAllNodes считает декремент затухания для всех подвижных узлов,
// печатает краткую сводку в консоль и записывает детальные логи в файл.
func PrintLogDecrementForAllNodes(cnf *config.Graph, pointsStore *inmemory.PointsStore, skipFirstMaxima int, decrementStore *inmemory.DecrementStore, f, b float64) {
	if cnf == nil {
		fmt.Printf("LogDecrement: nil config\n")
		return
	}

	type nodeResult struct {
		nodeID int
		xEq    float64
		delta  float64
		used   int
		maxs   []maxPoint
		err    error
	}

	results := make([]nodeResult, 0)
	for _, node := range cnf.Nodes {
		if node.IsFixed {
			continue
		}

		r := nodeResult{nodeID: node.ID}
		xEq, ok := equilibriumForNodeFromConfig(cnf, node.ID)
		if !ok {
			r.err = fmt.Errorf("cannot compute equilibrium")
			results = append(results, r)
			continue
		}
		r.xEq = xEq

		if pointsStore == nil {
			r.err = fmt.Errorf("points store is nil")
			results = append(results, r)
			continue
		}
		nodePoints := pointsStore.GetPoints(node.ID)
		maxs := FindAmplitudeMaximaInMemory(nodePoints, xEq)
		r.maxs = maxs

		delta, used, err := EstimateLogDecrementFromGraphPoints(nodePoints, xEq, skipFirstMaxima)
		if err != nil {
			r.err = err
			results = append(results, r)
			continue
		}
		r.delta = delta
		r.used = used
		if err := writeDecrementPoints(node.ID, r.maxs, skipFirstMaxima); err != nil {
			r.err = fmt.Errorf("write decrement points: %w", err)
		}
		results = append(results, r)

		decrementStore.AddDecrement(node.ID, f, b, delta)
	}

	// Краткая человекочитаемая сводка в процентах (до 2 знаков) перед детальными логами.
	fmt.Println("=== Log Decrement Summary (per node) ===")
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("Node %d: failed (%v)\n", r.nodeID, r.err)
			continue
		}
		decayPercent := (1.0 - math.Exp(-r.delta)) * 100.0
		// delta показываем с нормальной точностью, иначе разные значения выглядят одинаково.
		fmt.Printf("Node %d: delta=%.6f, amplitude decay per peak=%.2f%%\n",
			r.nodeID, r.delta, decayPercent)
	}

	// Детальные логи по каждому узлу (A_n и delta_n) в консоль и в файл.
	var sb strings.Builder
	sb.WriteString("### Logarithmic decrement details (all nodes)\n")
	for _, r := range results {
		if r.err != nil {
			line := fmt.Sprintf("Node %d: failed (%v)\n\n", r.nodeID, r.err)
			sb.WriteString(line)
			continue
		}

		decayPercent := (1.0 - math.Exp(-r.delta)) * 100.0
		head := fmt.Sprintf("Node %d: xEq=%.8f, skipFirst=%d, usedPairs=%d, delta=%.8f, decay=%.2f%%\n",
			r.nodeID, r.xEq, skipFirstMaxima, r.used, r.delta, decayPercent)
		sb.WriteString(head)

		for i := skipFirstMaxima; i < len(r.maxs)-1; i++ {
			line := fmt.Sprintf("  A_%d=%.8f (t=%.4f), A_%d=%.8f (t=%.4f), delta=%.8f\n",
				i, r.maxs[i].a, r.maxs[i].t,
				i+1, r.maxs[i+1].a, r.maxs[i+1].t,
				math.Log(r.maxs[i].a/r.maxs[i+1].a))
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	_ = os.WriteFile(decrementDetailsLogPath, []byte(sb.String()), 0666)
}

// StoreLogDecrementForAllNodes считает средний декремент по подвижным узлам
// и сохраняет в decrementStore без записи decrement_points / decrement_details.log и без вывода в консоль.
func StoreLogDecrementForAllNodes(cnf *config.Graph, pointsStore *inmemory.PointsStore, skipFirstMaxima int, decrementStore *inmemory.DecrementStore, frequencyStore *inmemory.FreqStore, f, b float64) {
	if cnf == nil || pointsStore == nil || decrementStore == nil {
		return
	}
	for _, node := range cnf.Nodes {
		if node.IsFixed {
			continue
		}
		xEq, ok := equilibriumForNodeFromConfig(cnf, node.ID)
		if !ok {
			continue
		}
		nodePoints := pointsStore.GetPoints(node.ID)
		delta, _, err := EstimateLogDecrementFromGraphPoints(nodePoints, xEq, skipFirstMaxima)
		if err != nil {
			log.Errorf("Error estimating log decrement for node %d: %v", node.ID, err)

			continue
		}

		maxs := FindAmplitudeMaximaInMemory(nodePoints, xEq)

		freqKoefs := GetAvgFrequency4Nodes(node.ID, maxs, skipFirstMaxima, f, b)

		frequencyStore.AddFreq(node.ID, freqKoefs.Koef1, freqKoefs.Koef2, freqKoefs.Freq)

		decrementStore.AddDecrement(node.ID, f, b, delta)
	}
}

func GetAvgFrequency4Nodes(nodeID int, maxs []maxPoint, skipFirstMaxima int, f, b float64) inmemory.FreqAeroKoef {
	w := 0.0
	n := 0.0

	start := skipFirstMaxima
	if start > len(maxs)-2 {
		start = len(maxs) - 2
	}
	if start < 0 {
		start = 0
	}

	for i := start; i < len(maxs)-1; {
		if i+2 >= len(maxs) {
			break
		}

		t1 := maxs[i].t
		t2 := maxs[i+2].t
		dt := t2 - t1
		if dt > 0 {
			freq := 1 / dt
			w += freq
			n += 1.0
		}
		i += 2
	}

	if n == 0 {
		//log.Infof("No frequencies found for node %d", nodeID)
		return inmemory.FreqAeroKoef{
			Koef1: f,
			Koef2: b,
			Freq:  0.0,
		}
	}

	return inmemory.FreqAeroKoef{
		Koef1: f,
		Koef2: b,
		Freq:  w / n,
	}
}
