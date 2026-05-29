package utils

import (
	"fmt"
	"os"

	config "masters/internal/config"
	inmemory "masters/internal/inMemory"
	"math"
	"sort"
)

// phaseDiffDownsample — сколько точек максимум писать в phaseDiffStore*.json на один прогон (f,b).
// Полная серия Δφ(t) может быть десятки тысяч шагов интегратора; в JSON и интерактивный
// график нужен разумный объём — равномерно берётся sampleCount точек по времени.
const phaseDiffDownsample = 160

const (
	interbladePhasePointsFileTmpl = "../wolfram/paramsAndPoints/interblade_phase_points%d.txt"
	bladePhasePointsFileTmpl      = "../wolfram/paramsAndPoints/blade_phase_points%d.txt"
)

// StorePhaseDiffForSweep сохраняет разность фаз (в градусах) между узлом и следующим
// подвижным соседом по кольцу для пары аэрокоэффициентов (f, b).
// Фаза восстанавливается согласованно с НУ Самойловича: atan2(-(v+δ_rate·a), ν·a),
// где ν и δ_rate=δ_log·ν берутся из frequencyStore и decrementStore для этого (f,b).
func StorePhaseDiffForSweep(
	cnf *config.Graph,
	pointsStore *inmemory.PointsStore,
	phaseStore *inmemory.PhaseDiffStore,
	decrementStore *inmemory.DecrementStore,
	frequencyStore *inmemory.FreqStore,
	f, b float64,
	skipFirstMaxima int,
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

		deltaF_A, wA := modalPhaseParams(cnf, pointsStore, decrementStore, frequencyStore, nodeID, f, b, skipFirstMaxima, xEqA)
		deltaF_B, wB := modalPhaseParams(cnf, pointsStore, decrementStore, frequencyStore, neighborID, f, b, skipFirstMaxima, xEqB)

		ptsA := pointsStore.GetPoints(nodeID)
		ptsB := pointsStore.GetPoints(neighborID)
		meanDeg, stdDeg, tOut, phaseOut, absPhaseOut := interbladePhaseDiffDeg(
			ptsA, ptsB, xEqA, xEqB,
			deltaF_A, wA, deltaF_B, wB,
			phaseDiffDownsample,
		)
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

// WritePhasePointsForSingleRun пишет Δφ(t) и φ(t) в txt (single run, PARALLEL=false).
// interblade_phase_pointsN.txt: t и φ(neighbor)−φ(node), °.
// blade_phase_pointsN.txt: t и φ(node), °.
func WritePhasePointsForSingleRun(
	cnf *config.Graph,
	pointsStore *inmemory.PointsStore,
	decrementStore *inmemory.DecrementStore,
	frequencyStore *inmemory.FreqStore,
	f, b float64,
	skipFirstMaxima int,
) error {
	if cnf == nil || pointsStore == nil {
		return nil
	}

	movable := movableNodeIDs(cnf)
	if len(movable) < 2 {
		return nil
	}

	for i, nodeID := range movable {
		neighborID := movable[(i+1)%len(movable)]
		xEqA, okA := equilibriumForNodeFromConfig(cnf, nodeID)
		xEqB, okB := equilibriumForNodeFromConfig(cnf, neighborID)
		if !okA || !okB {
			continue
		}

		deltaF_A, wA := modalPhaseParams(cnf, pointsStore, decrementStore, frequencyStore, nodeID, f, b, skipFirstMaxima, xEqA)
		deltaF_B, wB := modalPhaseParams(cnf, pointsStore, decrementStore, frequencyStore, neighborID, f, b, skipFirstMaxima, xEqB)

		ptsA := pointsStore.GetPoints(nodeID)
		ptsB := pointsStore.GetPoints(neighborID)
		_, _, tOut, phaseOut, absOut := interbladePhaseDiffDeg(
			ptsA, ptsB, xEqA, xEqB,
			deltaF_A, wA, deltaF_B, wB,
			0,
		)
		if len(tOut) == 0 {
			continue
		}

		display := make([]float64, len(phaseOut))
		for j, p := range phaseOut {
			display[j] = -p
		}

		if err := writeTwoColumnSeries(fmt.Sprintf(interbladePhasePointsFileTmpl, nodeID), tOut, display); err != nil {
			return err
		}
		if err := writeTwoColumnSeries(fmt.Sprintf(bladePhasePointsFileTmpl, nodeID), tOut, absOut); err != nil {
			return err
		}
	}
	return nil
}

func writeTwoColumnSeries(path string, t, y []float64) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := range t {
		if _, err := fmt.Fprintf(f, "%.10f %.10f\n", t[i], y[i]); err != nil {
			return err
		}
	}
	return f.Close()
}

// PrintInterbladePhaseSummary печатает среднее и СКО межлопаточной Δφ по всем парам кольца.
// Формула и unwrap — как в StorePhaseDiffForSweep; подпись φ(neighbor)−φ(node).
func PrintInterbladePhaseSummary(
	cnf *config.Graph,
	pointsStore *inmemory.PointsStore,
	decrementStore *inmemory.DecrementStore,
	frequencyStore *inmemory.FreqStore,
	f, b float64,
	skipFirstMaxima int,
) {
	if cnf == nil || pointsStore == nil {
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

		deltaF_A, wA := modalPhaseParams(cnf, pointsStore, decrementStore, frequencyStore, nodeID, f, b, skipFirstMaxima, xEqA)
		deltaF_B, wB := modalPhaseParams(cnf, pointsStore, decrementStore, frequencyStore, neighborID, f, b, skipFirstMaxima, xEqB)

		ptsA := pointsStore.GetPoints(nodeID)
		ptsB := pointsStore.GetPoints(neighborID)
		meanDeg, stdDeg, tOut, _, _ := interbladePhaseDiffDeg(
			ptsA, ptsB, xEqA, xEqB,
			deltaF_A, wA, deltaF_B, wB,
			phaseDiffDownsample,
		)
		if len(tOut) == 0 {
			continue
		}

		// meanDeg внутри: φ(node)−φ(neighbor); в подписи — φ(neighbor)−φ(node).
		fmt.Printf("  φ(%d)−φ(%d): среднее = %.2f°, СКО = %.2f°\n", neighborID, nodeID, -meanDeg, stdDeg)
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

// modalPhaseParams: δ_rate [1/с] = δ_log · ν, ν [1/с] — из store или пересчёт по траектории.
func modalPhaseParams(
	cnf *config.Graph,
	pointsStore *inmemory.PointsStore,
	decrementStore *inmemory.DecrementStore,
	frequencyStore *inmemory.FreqStore,
	nodeID int,
	f, b float64,
	skipFirstMaxima int,
	xEq float64,
) (deltaRate, nu float64) {
	if decrementStore != nil && frequencyStore != nil {
		logDec, okDec := decrementStore.LookupDecrement(nodeID, f, b)
		freq, okFreq := frequencyStore.LookupFreq(nodeID, f, b)
		if okDec && okFreq && freq > 0 {
			return logDec * freq, freq * 2 * math.Pi
		}
	}
	if pointsStore == nil {
		return 0, 1
	}
	pts := pointsStore.GetPoints(nodeID)
	maxs := FindAmplitudeMaximaInMemory(pts, xEq)
	logDec, _, err := EstimateLogDecrementFromGraphPoints(pts, xEq, skipFirstMaxima)
	fk := GetAvgFrequency4Nodes(nodeID, maxs, skipFirstMaxima, f, b)
	if err != nil || fk.Freq <= 0 {
		return 0, 1
	}
	return logDec * fk.Freq, fk.Freq * 2 * math.Pi
}

// phaseModalRad — ψ = atan2(-(v + δ_rate·a), ν·a), согласовано с phaseShift / Самойлович.
func phaseModalRad(a, v, deltaF, w float64) float64 {
	if w == 0 {
		return math.Atan2(-v, a)
	}

	//log.Debugf("deltaF: %f, w: %f, a: %f, v: %f", deltaF, w, a, v)

	return math.Atan2(-(v + deltaF*a), w*a)
}

func interbladePhaseDiffDeg(
	ptsA, ptsB []inmemory.Point,
	xEqA, xEqB float64,
	deltaF_A, wA, deltaF_B, wB float64,
	sampleCount int,
) (meanDeg, stdDeg float64, tOut, phaseOut, absPhaseOut []float64) {
	n := len(ptsA)
	if n != len(ptsB) || n < 5 {
		return 0, 0, nil, nil, nil
	}

	tSeries := make([]float64, 0, n-2)
	paSeries := make([]float64, 0, n-2)
	pbSeries := make([]float64, 0, n-2)

	for i := 1; i < n-1; i++ {
		dispA := ptsA[i].X - xEqA
		dispB := ptsB[i].X - xEqB
		dt := ptsA[i+1].T - ptsA[i-1].T
		if dt <= 0 {
			continue
		}
		va := (ptsA[i+1].X - ptsA[i-1].X) / dt
		vb := (ptsB[i+1].X - ptsB[i-1].X) / dt

		ampA := math.Hypot(dispA, va)
		ampB := math.Hypot(dispB, vb)
		if ampA < 1e-10 || ampB < 1e-10 {
			continue
		}

		pa := phaseModalRad(dispA, va, deltaF_A, wA)
		pb := phaseModalRad(dispB, vb, deltaF_B, wB)
		tSeries = append(tSeries, ptsA[i].T)
		paSeries = append(paSeries, pa*180.0/math.Pi)
		pbSeries = append(pbSeries, pb*180.0/math.Pi)
	}

	if len(paSeries) == 0 {
		return 0, 0, nil, nil, nil
	}

	unwrappedPa := unwrapDegrees(paSeries)
	unwrappedPb := unwrapDegrees(pbSeries)
	unwrappedAbs := unwrappedPa

	diffSeries := make([]float64, len(unwrappedPa))
	for i := range diffSeries {
		diffSeries[i] = unwrappedPa[i] - unwrappedPb[i]
	}
	normalizedDiff := normalizePhaseBranchDegrees(diffSeries)

	meanDeg, stdDeg = meanStd(normalizedDiff)
	if sampleCount <= 0 {
		tOut = append([]float64(nil), tSeries...)
		phaseOut = append([]float64(nil), normalizedDiff...)
		absPhaseOut = append([]float64(nil), unwrappedAbs...)
	} else {
		tOut, phaseOut = downsampleSeries(tSeries, normalizedDiff, sampleCount)
		_, absPhaseOut = downsampleSeries(tSeries, unwrappedAbs, sampleCount)
	}
	return meanDeg, stdDeg, tOut, phaseOut, absPhaseOut
}

// normalizePhaseBranchDegrees — одна ветка (−180°…+180°): вычитаем 360°·round(median/360°).
func normalizePhaseBranchDegrees(deg []float64) []float64 {
	if len(deg) == 0 {
		return deg
	}
	offset := 360.0 * math.Round(medianFloat64(deg)/360.0)
	out := make([]float64, len(deg))
	for i, v := range deg {
		out[i] = v - offset
	}
	return out
}

func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	m := len(cp)
	if m%2 == 1 {
		return cp[m/2]
	}
	return 0.5 * (cp[m/2-1] + cp[m/2])
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
