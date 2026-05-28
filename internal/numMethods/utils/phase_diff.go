package utils

import (
	config "masters/internal/config"
	inmemory "masters/internal/inMemory"
	"math"
	"sort"
)

const phaseDiffDownsample = 160

// StorePhaseDiffForSweep сохраняет разность фаз (в градусах) между узлом и следующим
// подвижным соседом по кольцу для пары аэрокоэффициентов (f, b).
func StorePhaseDiffForSweep(
	cnf *config.Graph,
	pointsStore *inmemory.PointsStore,
	phaseStore *inmemory.PhaseDiffStore,
	f, b float64,
) {
	if cnf == nil || pointsStore == nil || phaseStore == nil {
		return
	}

	movable := movableNodeIDs(cnf)
	if len(movable) < 2 {
		return
	}

	for i, nodeID := range movable {
		neighborID := movable[(i+1)%len(movable)]
		xEqA, okA := equilibriumForNodeFromConfig(cnf, nodeID)
		xEqB, okB := equilibriumForNodeFromConfig(cnf, neighborID)
		if !okA || !okB {
			continue
		}

		ptsA := pointsStore.GetPoints(nodeID)
		ptsB := pointsStore.GetPoints(neighborID)
		meanDeg, stdDeg, tOut, phaseOut, absPhaseOut := interbladePhaseDiffDeg(ptsA, ptsB, xEqA, xEqB, phaseDiffDownsample)
		if len(tOut) == 0 {
			continue
		}

		phaseStore.AddRecord(nodeID, inmemory.PhaseDiffRecord{
			Koef1:       f,
			Koef2:       b,
			NodeID:      nodeID,
			NeighborID:  neighborID,
			MeanDeg:     meanDeg,
			StdDeg:      stdDeg,
			T:           tOut,
			PhaseDeg:    phaseOut,
			AbsPhaseDeg: absPhaseOut,
		})
	}
}

func movableNodeIDs(cnf *config.Graph) []int {
	ids := make([]int, 0)
	for _, n := range cnf.Nodes {
		if !n.IsFixed {
			ids = append(ids, n.ID)
		}
	}
	sort.Ints(ids)
	return ids
}

func interbladePhaseDiffDeg(
	ptsA, ptsB []inmemory.Point,
	xEqA, xEqB float64,
	sampleCount int,
) (meanDeg, stdDeg float64, tOut, phaseOut, absPhaseOut []float64) {
	n := len(ptsA)
	if n != len(ptsB) || n < 5 {
		return 0, 0, nil, nil, nil
	}

	tSeries := make([]float64, 0, n-2)
	phaseSeries := make([]float64, 0, n-2)
	absPhaseSeries := make([]float64, 0, n-2)

	for i := 1; i < n-1; i++ {
		a := ptsA[i].X - xEqA
		b := ptsB[i].X - xEqB
		dt := ptsA[i+1].T - ptsA[i-1].T
		if dt <= 0 {
			continue
		}
		va := (ptsA[i+1].X - ptsA[i-1].X) / dt
		vb := (ptsB[i+1].X - ptsB[i-1].X) / dt

		ampA := math.Hypot(a, va)
		ampB := math.Hypot(b, vb)
		if ampA < 1e-10 || ampB < 1e-10 {
			continue
		}

		pa := math.Atan2(-va, a)
		pb := math.Atan2(-vb, b)
		diffDeg := (pa - pb) * 180.0 / math.Pi
		absDeg := pa * 180.0 / math.Pi
		tSeries = append(tSeries, ptsA[i].T)
		phaseSeries = append(phaseSeries, diffDeg)
		absPhaseSeries = append(absPhaseSeries, absDeg)
	}

	if len(phaseSeries) == 0 {
		return 0, 0, nil, nil, nil
	}

	unwrapped := unwrapDegrees(phaseSeries)
	meanDeg, stdDeg = meanStd(unwrapped)
	tOut, phaseOut = downsampleSeries(tSeries, unwrapped, sampleCount)
	_, absPhaseOut = downsampleSeries(tSeries, absPhaseSeries, sampleCount)
	return meanDeg, stdDeg, tOut, phaseOut, absPhaseOut
}

func unwrapDegrees(deg []float64) []float64 {
	if len(deg) == 0 {
		return deg
	}
	out := make([]float64, len(deg))
	out[0] = deg[0]
	for i := 1; i < len(deg); i++ {
		d := deg[i] - deg[i-1]
		for d > 180.0 {
			d -= 360.0
		}
		for d < -180.0 {
			d += 360.0
		}
		out[i] = out[i-1] + d
	}
	return out
}

func meanStd(values []float64) (mean, std float64) {
	if len(values) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))
	if len(values) < 2 {
		return mean, 0
	}
	var sq float64
	for _, v := range values {
		d := v - mean
		sq += d * d
	}
	std = math.Sqrt(sq / float64(len(values)))
	return mean, std
}

func downsampleSeries(t, y []float64, n int) ([]float64, []float64) {
	if len(t) == 0 || len(y) == 0 || n <= 0 {
		return nil, nil
	}
	if len(t) <= n {
		outT := make([]float64, len(t))
		outY := make([]float64, len(y))
		copy(outT, t)
		copy(outY, y)
		return outT, outY
	}
	outT := make([]float64, n)
	outY := make([]float64, n)
	last := len(t) - 1
	for i := 0; i < n; i++ {
		idx := int(math.Round(float64(i) * float64(last) / float64(n-1)))
		if idx < 0 {
			idx = 0
		}
		if idx > last {
			idx = last
		}
		outT[i] = t[idx]
		outY[i] = y[idx]
	}
	return outT, outY
}
