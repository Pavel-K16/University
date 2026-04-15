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

// EstimateLogDecrementFromGraphPoints читает graph_points{nodeID}.txt,
// считает амплитуды A=|x-xEq|, находит локальные максимумы амплитуды и возвращает оценку логарифмического декремента.
//
// skipFirstMaxima — сколько первых максимумов амплитуды пропустить (обычно 2+).
func findAmplitudeMaximaInMemory(points []inmemory.Point, xEq float64) []maxPoint {
	if len(points) == 0 {
		return nil
	}
	maxs := make([]maxPoint, 0)

	amp := func(i int) float64 {
		return math.Abs(points[i].X - xEq)
	}

	// Левый край: добавляем только если сразу идём вниз (валидный стартовый максимум).
	if len(points) >= 2 {
		a0 := amp(0)
		a1 := amp(1)
		if a0 > a1 {
			maxs = append(maxs, maxPoint{t: points[0].T, a: a0})
		}
	}

	for i := 1; i < len(points)-1; i++ {
		aPrev := amp(i - 1)
		aCurr := amp(i)
		aNext := amp(i + 1)
		dL := aCurr - aPrev
		dR := aNext - aCurr
		if dL > 0 && dR < 0 {
			maxs = append(maxs, maxPoint{t: points[i].T, a: aCurr})
		}
	}

	// Правый край не добавляем: на конце интервала часто незавершённый полупериод,
	// и это даёт ложный "пик" амплитуды.

	return maxs
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

func EstimateLogDecrementFromGraphPoints(points []inmemory.Point, xEq float64, skipFirstMaxima int) (delta float64, used int, err error) {
	maxs := findAmplitudeMaximaInMemory(points, xEq)
	if len(maxs) < 2 {
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
		maxs := findAmplitudeMaximaInMemory(nodePoints, xEq)
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
