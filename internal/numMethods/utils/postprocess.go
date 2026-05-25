package utils

import (
	"os"
	"strconv"
	"strings"

	config "masters/internal/config"
	inmemory "masters/internal/inMemory"
	"masters/internal/logger"
)

var postprocessLog = logger.LoggerInit()

// DefaultSkipFirstMaxima — сколько первых амплитудных пиков пропускать
// при расчёте декремента и средней частоты (переходный процесс).
const DefaultSkipFirstMaxima = 2

// SkipFirstMaxima возвращает число пропускаемых первых пиков.
// Переопределение: export SKIP_FIRST_MAXIMA=<неотрицательное целое>.
func SkipFirstMaxima() int {
	raw := strings.TrimSpace(os.Getenv("SKIP_FIRST_MAXIMA"))
	if raw == "" {
		return DefaultSkipFirstMaxima
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return DefaultSkipFirstMaxima
	}
	return v
}

// LogFrequencySummary пишет среднюю частоту по узлам (из pointsStore, те же пики что для декремента).
func LogFrequencySummary(cnf *config.Graph, pointsStore *inmemory.PointsStore, skipFirstMaxima int, f, b float64) {
	if cnf == nil || pointsStore == nil {
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
		maxs := FindAmplitudeMaximaInMemory(pointsStore.GetPoints(node.ID), xEq)
		fk := GetAvgFrequency4Nodes(node.ID, maxs, skipFirstMaxima, f, b)
		if fk.Freq == 0 {
			postprocessLog.Infof("No frequencies found for node %d", node.ID)
			continue
		}
		postprocessLog.Infof("Frequency for node %d: %f", node.ID, fk.Freq)
	}
}
