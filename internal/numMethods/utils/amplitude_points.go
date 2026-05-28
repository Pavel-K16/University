package utils

import (
	"fmt"
	config "masters/internal/config"
	inmemory "masters/internal/inMemory"
	"math"
	"os"
)

const (
	amplitudePointsFileTmpl = "../wolfram/paramsAndPoints/amplitude_points%d.txt"
)

// WriteAmplitudePointsFromGraphFiles читает graph_points*.txt и записывает
// амплитуды-пики A_n=|x(t_n)-xEq| в amplitude_points*.txt для каждого узла из конфига.
// Логика поиска пиков совпадает с той, что используется в декременте затухания.
// skipFirstMaxima - сколько первых пиков пропустить; в файл попадут именно те A_n,
// которые затем используются в вычислении лог-декремента.
func WriteAmplitudePointsFromGraphFiles(cnf *config.Graph, pointsStore *inmemory.PointsStore, amplitudeStore *inmemory.AmplitudeStore, skipFirstMaxima int) error {
	if cnf == nil {
		return fmt.Errorf("nil config")
	}

	for _, node := range cnf.Nodes {
		if node.IsFixed {
			continue
		}

		xEq, ok := equilibriumForNodeFromConfig(cnf, node.ID)
		if !ok {
			// Если опорной пружины к фиксированному узлу нет, считаем равновесие как стартовое положение.
			xEq = node.Position
		}
		if pointsStore == nil {
			return fmt.Errorf("points store is nil")
		}
		nodePoints := pointsStore.GetPoints(node.ID)
		maxs := FindAmplitudeMaximaInMemory(nodePoints, xEq)
		start := skipFirstMaxima
		if start > len(maxs)-1 {
			start = len(maxs)
		}

		outPath := fmt.Sprintf(amplitudePointsFileTmpl, node.ID)
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("open amplitude file for node %d: %w", node.ID, err)
		}

		for i := start; i < len(maxs); i++ {
			amplitudeStore.AddAmplitude(node.ID, maxs[i].a, maxs[i].t)

			m := maxs[i]
			fmt.Fprintf(out, "%.10f %.10f\n", m.t, m.a)
		}
		out.Close()
	}

	return nil
}

func FindAmplitudeMaximaInMemory(points []inmemory.Point, xEq float64) []maxPoint {
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
