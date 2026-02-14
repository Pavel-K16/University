package utils

import (
	"fmt"
	"io"
	graph "masters/internal/graph"
	iofile "masters/internal/ioFile"
	"math"
	"os"
	"strconv"
	"strings"
)

const (
	extremaPoinsFileTmp = "../wolfram/paramsAndPoints/extrema_points%s.txt"
)

func VecAdd(a ...[]float64) []float64 {
	sumVec := make([]float64, 2)

	for _, val := range a {
		sumVec[0] += val[0]
		sumVec[1] += val[1]
	}

	return sumVec
}

func VecMult(k float64, a []float64) []float64 {
	vec := make([]float64, 2)

	vec[0] = a[0] * k
	vec[1] = a[1] * k

	return vec
}

func FindExtrema(nodeID int) error {
	points, err := GetPointsFromFile(nodeID)

	if err != nil {
		log.Errorf("Error: %s", err)

		return nil
	}

	if len(points) < 3 {
		log.Warningf("NodeID: %d, need more than 3 points", nodeID)

		return nil
	}

	extremaList := make([]graph.ExtremaPoint, 0)

	// Проходим по всем точкам, начиная со второй и заканчивая предпоследней
	for i := 1; i < len(points)-1; i++ {
		prev := points[i-1]
		curr := points[i]
		next := points[i+1]

		dtLeft := curr.Time - prev.Time
		dtRight := next.Time - curr.Time

		// Проверяем, что шаги по времени не равны нулю (на всякий случай)
		if dtLeft <= 0 || dtRight <= 0 {
			continue
		}

		derivativeLeft := (curr.Position - prev.Position) / dtLeft
		derivativeRight := (next.Position - curr.Position) / dtRight

		// Локальный максимум: производная слева положительная, справа отрицательная
		// Это означает, что функция растёт до этой точки и убывает после неё
		isMax := derivativeLeft > 0 && derivativeRight < 0

		// Если это локальный максимум, добавляем его
		if isMax {
			extremaList = append(extremaList, graph.ExtremaPoint{
				Time:      curr.Time,
				Amplitude: math.Abs(curr.Position), // абсолютное значение амплитуды
			})
		}
	}

	if err := writeExtremasToFile(nodeID, extremaList); err != nil {
		return err
	}

	log.Infof("Узел %d: найдено %d локальных максимумов\n", nodeID, len(extremaList))

	return nil
}

func writeExtremasToFile(nodeID int, extremaList []graph.ExtremaPoint) error {
	extremaPointsFile, err := os.OpenFile(fmt.Sprintf(extremaPoinsFileTmp, fmt.Sprintf("%d", nodeID)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Errorf("Error: %s", err)

		return err
	}

	defer extremaPointsFile.Close()

	for _, extrema := range extremaList {
		fmt.Fprintf(extremaPointsFile, "%.10f %.10f\n", extrema.Time, extrema.Amplitude)
	}

	return nil
}

func GetPointsFromFile(nodeID int) ([]graph.Point, error) {
	points := make([]graph.Point, 0)
	filePath := fmt.Sprintf(iofile.GraphPointsFileTmpl, fmt.Sprintf("%d", nodeID))

	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("Error openning file: %s", err)

		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("Error reading file: %s", err)

		return nil, err
	}

	strSData := strings.Split(string(data), "\n")
	for _, str := range strSData {
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		splittedStr := strings.Fields(str)
		if len(splittedStr) != 2 {
			continue
		}

		t, err := strconv.ParseFloat(splittedStr[0], 64)
		if err != nil {
			log.Errorf("Error: %s", err)

			return nil, err
		}

		x, err := strconv.ParseFloat(splittedStr[1], 64)
		if err != nil {
			log.Errorf("Error: %s", err)

			return nil, err
		}

		point := graph.Point{
			Time:     t,
			Position: x,
		}

		points = append(points, point)
	}

	return points, nil
}
