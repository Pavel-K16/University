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

	if len(maxs) < 3 {
		return nil
	}

	start := clampSkipFirstMaxima(skipFirstMaxima, len(maxs))

	for i := start; i < len(maxs)-2; i++ {
		a1 := maxs[i].a
		a2 := maxs[i+2].a
		if a1 <= 0 || a2 <= 0 {
			continue
		}
		delta := math.Log(a1 / a2)
		// Время привязываем к пику через один период (t_{n+2})
		fmt.Fprintf(f, "%.10f %.10f\n", maxs[i+2].t, delta)
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

// Delta — локальные δ_log и f_Hz на сетке пиков (t = t_{i+2}); для ZOH в variant A.
type Delta struct {
	delta float64
	freq  float64
	t     float64
}

func DeltaIndexAt(deltas []Delta, t float64) int {
	n := len(deltas)
	if n == 0 || t < deltas[0].t {
		return -1
	}
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		if deltas[mid].t <= t {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo - 1
}

func DeltaAt(deltas []Delta, t float64) (Delta, bool) {
	if len(deltas) == 0 {
		return Delta{}, false
	}
	if t < deltas[0].t {
		// Обратный ZOH: до первого пика используем первый известный δ и f.
		return deltas[0], true
	}
	k := DeltaIndexAt(deltas, t)
	if k < 0 {
		return Delta{}, false
	}
	return deltas[k], true
}

// phaseHoldUntilT — до этого времени в фазе держим δ[0] (без ранних переключений ZOH).
func phaseHoldUntilT(samples []Delta, skipFirstMaxima int) float64 {
	if len(samples) == 0 {
		return 0
	}
	idx := skipFirstMaxima
	if idx < 1 {
		idx = 1
	}
	if idx >= len(samples) {
		return math.MaxFloat64
	}
	return samples[idx].t
}

func DeltaAtForPhase(deltas []Delta, t, holdUntilT float64) (Delta, bool) {
	if len(deltas) == 0 {
		return Delta{}, false
	}
	if t < holdUntilT {
		return deltas[0], true
	}
	return DeltaAt(deltas, t)
}

// deltaPairAtSynced — для пары лопаток один индекс ZOH (min(kA,kB)), чтобы δ не переключался
// у соседей в разные моменты; до holdUntilT — δ[0] у каждой своей лопатки.
func deltaPairAtSynced(samplesA, samplesB []Delta, t, holdUntilT float64) (Delta, Delta, bool) {
	if len(samplesA) == 0 || len(samplesB) == 0 {
		return Delta{}, Delta{}, false
	}
	if t < holdUntilT {
		return samplesA[0], samplesB[0], true
	}
	kA := DeltaIndexAt(samplesA, t)
	kB := DeltaIndexAt(samplesB, t)
	if kA < 0 || kB < 0 {
		return Delta{}, Delta{}, false
	}
	k := kA
	if kB < k {
		k = kB
	}
	return samplesA[k], samplesB[k], true
}

func clampSkipFirstMaxima(skip, numMaxs int) int {
	start := skip
	if start > numMaxs-3 {
		start = numMaxs - 3
	}
	if start < 0 {
		start = 0
	}
	return start
}

// GetDeltasFromAmplitudeMaxima — δ_log и f_Hz за период между пиками i и i+2, время t_{i+2}.
func GetDeltasFromAmplitudeMaxima(maxs []maxPoint, start int) []Delta {
	if len(maxs) < 3 {
		return nil
	}
	start = clampSkipFirstMaxima(start, len(maxs))
	capN := len(maxs) - start - 2
	if capN < 0 {
		capN = 0
	}
	out := make([]Delta, 0, capN)
	for i := start; i < len(maxs)-2; i++ {
		a1 := maxs[i].a
		a2 := maxs[i+2].a
		dt := maxs[i+2].t - maxs[i].t
		if a1 <= 0 || a2 <= 0 || dt <= 0 {
			continue
		}
		out = append(out, Delta{
			delta: math.Log(a1 / a2),
			freq:  1 / dt,
			t:     maxs[i+2].t,
		})
	}
	return out
}

func EstimateLogDecrementFromGraphPoints(points []inmemory.Point, xEq float64, skipFirstMaxima int) (delta float64, used int, err error) {
	maxs := FindAmplitudeMaximaInMemory(points, xEq)
	if len(maxs) < 3 {
		if len(points) == 0 {
			return 0, 0, fmt.Errorf("not enough amplitude maxima: maxs=%d (need >=3 peaks for period decrement; increase T or reduce damping)", len(maxs))
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

		if len(maxs) <= 2 {
			return oneMax, 0, nil
		}

		return 0, 0, fmt.Errorf("not enough amplitude maxima: maxs=%d (need >=3 peaks for period decrement; increase T or reduce damping)", len(maxs))
	}

	deltas := GetDeltasFromAmplitudeMaxima(maxs, skipFirstMaxima)

	if len(deltas) == 0 {
		return 0, 0, fmt.Errorf("failed to compute deltas (all zeros?)")
	}

	var sum float64
	for _, d := range deltas {
		sum += d.delta
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
		fk := GetAvgFrequency4Nodes(r.nodeID, r.maxs, skipFirstMaxima, f, b)
		deltaFreq := r.delta * fk.Freq
		decayPercentFK := (1.0 - math.Exp(-r.delta*fk.Freq)) * 100.0
		// delta показываем с нормальной точностью, иначе разные значения выглядят одинаково.
		fmt.Printf("Node %d: delta=%.6f; %.6f, amp dec per period=%.6f%%; %.6f%%\n",
			r.nodeID, r.delta, deltaFreq, decayPercent, decayPercentFK)
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
		fk := GetAvgFrequency4Nodes(r.nodeID, r.maxs, skipFirstMaxima, f, b)
		deltaFreq := r.delta * fk.Freq
		head := fmt.Sprintf("Node %d: xEq=%.8f, skipFirst=%d, usedPairs=%d, delta=%.8f, %.8f, decay=%.2f%%\n",
			r.nodeID, r.xEq, skipFirstMaxima, r.used, r.delta, deltaFreq, decayPercent)
		sb.WriteString(head)

		for i := skipFirstMaxima; i < len(r.maxs)-2; i++ {
			line := fmt.Sprintf("  A_%d=%.8f (t=%.4f), A_%d=%.8f (t=%.4f), delta=%.8f\n",
				i, r.maxs[i].a, r.maxs[i].t,
				i+2, r.maxs[i+2].a, r.maxs[i+2].t,
				math.Log(r.maxs[i].a/r.maxs[i+2].a))
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
	samples := GetDeltasFromAmplitudeMaxima(maxs, skipFirstMaxima)
	if len(samples) == 0 {
		return inmemory.FreqAeroKoef{
			Koef1: f,
			Koef2: b,
			Freq:  0.0,
		}
	}

	var sum float64
	for _, s := range samples {
		sum += s.freq
	}

	return inmemory.FreqAeroKoef{
		Koef1: f,
		Koef2: b,
		Freq:  sum / float64(len(samples)),
	}
}
