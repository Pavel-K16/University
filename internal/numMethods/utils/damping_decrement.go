package utils

import (
	"fmt"
	"math"

	config "masters/internal/config"
	"masters/internal/graph"
)

type maxPoint struct {
	t float64
	a float64 // amplitude = |x - xEq|
}

// equilibriumForNodeFromConfig вычисляет равновесное положение xEq для nodeID,
// исходя из "ненапряжённого" состояния пружины к фиксированному узлу.
//
// Логика:
// - если в конфиге есть ребро from=nodeID -> to=fixed с rest=r, то в графе xEq = xFixed - r
// - если в конфиге есть ребро from=fixed -> to=nodeID с rest=r, то из-за инвертирования rest
//   (см. Graph.AddEdge) получается xEq = xFixed + r
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
func EstimateLogDecrementFromGraphPoints(nodeID int, xEq float64, skipFirstMaxima int) (delta float64, used int, err error) {
	points, err := GetPointsFromFile(nodeID)
	if err != nil {
		return 0, 0, err
	}
	if len(points) < 3 {
		return 0, 0, fmt.Errorf("not enough points: %d", len(points))
	}

	// Амплитуда A(t)=|x-xEq|
	A := make([]float64, len(points))
	for i := range points {
		A[i] = math.Abs(points[i].Position - xEq)
	}

	// Находим локальные максимумы A[i].
	// Дискретно:
	// - на внутренних точках: если A растёт и затем начинает убывать
	// - на границах: если A[0] >= A[1] и/или A[n-1] >= A[n-2]
	maxs := make([]maxPoint, 0)

	// Левый край
	if len(A) >= 2 && A[0] >= A[1] {
		maxs = append(maxs, maxPoint{t: points[0].Time, a: A[0]})
	}

	for i := 1; i < len(A)-1; i++ {
		dL := A[i] - A[i-1]
		dR := A[i+1] - A[i]
		if dL > 0 && dR < 0 {
			maxs = append(maxs, maxPoint{t: points[i].Time, a: A[i]})
		}
	}

	// Правый край
	if len(A) >= 2 && A[len(A)-1] >= A[len(A)-2] {
		maxs = append(maxs, maxPoint{t: points[len(A)-1].Time, a: A[len(A)-1]})
	}

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

// PrintLogDecrementForConfNode0 посчитает оценку декремента для nodeID=0
// (только для конфигурации conf.json) и распечатает её в консоль.
func PrintLogDecrementForConfNode0(cnf *config.Graph, skipFirstMaxima int) {
	const nodeID = 0

	xEq, ok := equilibriumForNodeFromConfig(cnf, nodeID)
	if !ok {
		fmt.Printf("LogDecrement: cannot compute equilibrium for node %d from config\n", nodeID)
		return
	}

	delta, used, err := EstimateLogDecrementFromGraphPoints(nodeID, xEq, skipFirstMaxima)
	if err != nil {
		fmt.Printf("LogDecrement: error: %v\n", err)
		return
	}

	fmt.Printf("LogDecrement(node %d): xEq=%.8f skipFirst=%d usedPairs=%d delta≈%.8f\n",
		nodeID, xEq, skipFirstMaxima, used, delta)
	_ = graph.Point{} // keep import of graph for future extension
}

